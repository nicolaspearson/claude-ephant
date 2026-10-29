package ticket

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const corpusPlan = `---
kind: plan
---

# A plan

## Context

- standing fact

## Actions

- [x] Build the thing
  - as of 2026-10-17: the premise
- [ ] Ship the thing

## Inbox

- An idea
`

type fakeIssue struct {
	body     string
	comments []string
	state    string
	title    string
}

type fakeTracker struct {
	delay    time.Duration
	inFlight int
	issues   map[int]fakeIssue
	most     int
	mutex    sync.Mutex
	remote   string
	searches map[string][]map[string]any
	subIssue map[int][]int
}

func TestBlockedByCountsALineWithNoRefAsUnread(t *testing.T) {
	references, unread := blockedBy("## Blocked by\n\n- #12\n\nBlocked by:\n#13 is related prose", "owner/repo")

	require.Empty(t, references)
	require.Equal(t, 2, unread, "the refs sit on the lines below, where no table reads them")
}

func TestBlockedByReadsOnlyTheRefsLeadingABlockedByLine(t *testing.T) {
	for text, want := range map[string][]string{
		"### Blocked by: #12, #13":                                     {"#12", "#13"},
		"- **Blocked by:** #12, #13 and other/repo#4 — once #99 lands": {"#12", "#13", "other/repo#4"},
		"- [ ] Blocked by #12, #13":                                    {"#12", "#13"},
		"1. Blocked by #12, #13":                                       {"#12", "#13"},
		"Blocked by #1\n\nlater\n\nBlocked by: #2":                     {"#1", "#2"},
		"Blocked by #12 (supersedes #2, closed) and #13":               {"#12", "#13"},
		"Blocked by #12 / #13":                                         {"#12", "#13"},
		"Blocked by #12 or #13":                                        {"#12", "#13"},
		"Blocked by #7 andé #8":                                        {"#7"},
		"Blocked by Owner/Repo#12":                                     {"#12"},
		"Blocked by [#12](https://github.com/o/r/issues/12)":           {"#12"},
		"Blocked by naïve/x#12, é/repo#11 and #٣":                      {"naïve/x#12", "é/repo#11", "#٣"},
		"Blocked by: #12 (auth), #13 (billing)":                        {"#12", "#13"},
		"Blocked by: #12; #13":                                         {"#12", "#13"},
		"Blocked by: **#12**, `#13`":                                   {"#12", "#13"},
		"Blocked by: nothing — see #177":                               nil,
		"This was blocked by nothing like #5":                          nil,
		"| Blocked by | #12, #13 |":                                    {"#12", "#13"},
		"٣. Blocked by #21":                                            {"#21"},
	} {
		references, _ := blockedBy(text, "owner/repo")

		require.Equal(t, want, references, text)
	}
}

func TestBlockerTableTakesStatesFromTheCorpus(t *testing.T) {
	rows := blockerTable([]corpusEntry{{blockers: []string{"#12", "#99"}, name: "10", state: "open"}, {name: "12", state: "closed"}})

	require.Equal(t, []string{"#10    open   #12 closed, #99 ?", "#12    closed -"}, rows)
}

func TestChildrenOfTakesBothSourcesAndNotesEachGap(t *testing.T) {
	tracker := &fakeTracker{
		searches: map[string][]map[string]any{"Part of #87": {{"number": 88, "body": "Part of #87"}, {"number": 90, "body": "Part of #87\n"}, {"number": 91, "body": "Part of #871"}, {"number": 87, "body": "Part of #87"}}},
		subIssue: map[int][]int{87: {64, 88}},
	}

	children, gaps, err := childrenOf(context.Background(), &bytes.Buffer{}, tracker.commands(), "gh", 87)

	require.NoError(t, err)
	require.Equal(t, []int{64, 88, 90}, children, "the union, the map itself left out, a longer number no child")
	require.Equal(t, map[int]string{64: "is a sub-issue but never says 'Part of #87'", 87: "says 'Part of #87' but is not linked as a sub-issue", 90: "says 'Part of #87' but is not linked as a sub-issue"}, gaps)
}

func TestCompareMatchesTheBodyOrWithNoteAComment(t *testing.T) {
	tracker := &fakeTracker{issues: map[int]fakeIssue{164: {body: "Hello\r\nworld\r\n", comments: []string{"first", "the new note\n"}, title: "Title"}}, remote: "git@github.com:owner/repo.git"}
	for _, check := range []struct {
		code     int
		expected string
		note     bool
		output   string
	}{
		{code: 0, expected: "# 164 [open] Title\nHello\r\nworld\n\n", output: "#164: body matches E\n"},
		{code: 0, expected: "the new note", note: true, output: "#164: comment 2 matches E\n"},
		{code: 1, expected: "Hello\nworld", note: true},
		{code: 1, expected: "Hello\rworld\nagain", output: "#164: nothing matches E; closest is the body:\n--- E\n+++ #164 body\n@@ -1,3 +1,2 @@\n Hello\n-world\n-again+world\n"},
		{code: 1, expected: "the new note"},
	} {
		directory := t.TempDir()
		expectPath := filepath.Join(directory, "expected.md")
		require.NoError(t, os.WriteFile(expectPath, []byte(check.expected), 0o600))
		var stdout bytes.Buffer

		code := Compare(context.Background(), &stdout, &bytes.Buffer{}, tracker.commands(), "", 164, expectPath, check.note)

		require.Equal(t, check.code, code, check.expected)
		if check.output != "" {
			require.Equal(t, strings.ReplaceAll(check.output, "E", expectPath), stdout.String(), check.expected)
		}
	}
}

func TestCompareRefusesAFileItCannotRead(t *testing.T) {
	tracker := &fakeTracker{issues: map[int]fakeIssue{164: {body: "x"}}, remote: "git@github.com:owner/repo.git"}
	directory := t.TempDir()
	undecodable := filepath.Join(directory, "binary.md")
	require.NoError(t, os.WriteFile(undecodable, []byte{0xff, 0xfe}, 0o600))

	for _, path := range []string{filepath.Join(directory, "absent.md"), undecodable} {
		var stderr bytes.Buffer

		require.Equal(t, 2, Compare(context.Background(), &bytes.Buffer{}, &stderr, tracker.commands(), "", 164, path, false), path)
		require.True(t, strings.HasPrefix(stderr.String(), "ele: cannot read "+path), stderr.String())
	}
}

func TestCompareShowsTheFirstOfEquallyCloseComments(t *testing.T) {
	tracker := &fakeTracker{issues: map[int]fakeIssue{164: {body: "b", comments: []string{"abcx", "abcy"}}}, remote: "git@github.com:owner/repo.git"}
	expectPath := filepath.Join(t.TempDir(), "expected.md")
	require.NoError(t, os.WriteFile(expectPath, []byte("abcz"), 0o600))
	var stdout bytes.Buffer

	require.Equal(t, 1, Compare(context.Background(), &stdout, &bytes.Buffer{}, tracker.commands(), "", 164, expectPath, true))

	require.True(t, strings.HasPrefix(stdout.String(), "#164: nothing matches "+expectPath+"; closest is the comment 1:\n"), stdout.String())
}

func TestCorpusFetchesInParallelAndListsTicketsInTheOrderAsked(t *testing.T) {
	tracker := &fakeTracker{delay: 20 * time.Millisecond, issues: map[int]fakeIssue{}, remote: "git@github.com:owner/repo.git"}
	for _, number := range []int{1, 3, 5, 7, 9} {
		tracker.issues[number] = fakeIssue{body: "body", title: fmt.Sprintf("Title %d", number)}
	}
	var stdout bytes.Buffer

	code := Corpus(context.Background(), &stdout, &bytes.Buffer{}, tracker.commands(), CorpusOptions{Out: t.TempDir(), Tickets: []int{9, 3, 7, 1, 5, 3}})

	require.Equal(t, 0, code)
	require.Greater(t, tracker.most, 1)
	require.True(t, strings.HasPrefix(stdout.String(), "#9     open   "), stdout.String())
	var listed []string
	for _, line := range strings.Split(stdout.String(), "\n")[:5] {
		listed = append(listed, strings.Fields(line)[0])
	}
	require.Equal(t, []string{"#9", "#3", "#7", "#1", "#5"}, listed)
}

func TestCorpusRefusesADirectoryHoldingAnEarlierRunUnlessCleaned(t *testing.T) {
	tracker := &fakeTracker{issues: map[int]fakeIssue{9: {title: "Nine"}}, remote: "git@github.com:owner/repo.git"}
	for _, stale := range []string{"99.md", "raw/99.md", "manifest.json"} {
		out := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(out, "raw"), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(out, stale), []byte("# 99 [closed] a previous map\n"), 0o600))
		var stderr bytes.Buffer

		require.Equal(t, 1, Corpus(context.Background(), &bytes.Buffer{}, &stderr, tracker.commands(), CorpusOptions{Out: out, Tickets: []int{9}}), stale)
		require.Equal(t, "ele: "+out+" already holds 1 corpus file(s) — pass --clean, or name an empty directory\n", stderr.String())

		require.Equal(t, 0, Corpus(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, tracker.commands(), CorpusOptions{Clean: true, Out: out, Tickets: []int{9}}), stale)
		require.NoFileExists(t, filepath.Join(out, "99.md"))
		require.NoFileExists(t, filepath.Join(out, "raw", "99.md"))
	}
}

func TestCorpusRefusesTwoPlanItemsWithOneKey(t *testing.T) {
	plan := writePlan(t, strings.Replace(corpusPlan, "Ship the thing", "Build the thing", 1))
	var stderr bytes.Buffer

	code := Corpus(context.Background(), &bytes.Buffer{}, &stderr, (&fakeTracker{}).commands(), CorpusOptions{Out: t.TempDir(), Plans: []string{plan}})

	require.Equal(t, 1, code)
	require.Equal(t, "ele: two items share a key, so a ruling could land on either: repo/.documentation/plans/plan.md: Build the thing\n", stderr.String())
}

func TestCorpusRefusesTwoPlansOfOneFileName(t *testing.T) {
	plans := []string{writePlanAt(t, "first", "plan.md", corpusPlan), writePlanAt(t, "second", "plan.md", corpusPlan)}
	var stderr bytes.Buffer

	code := Corpus(context.Background(), &bytes.Buffer{}, &stderr, (&fakeTracker{}).commands(), CorpusOptions{Out: t.TempDir(), Plans: plans})

	require.Equal(t, 1, code)
	require.Equal(t, "ele: two plans share a file name, so one would overwrite the other: plan-01.md; plan-02.md; plan-03.md\n", stderr.String())
}

func TestCorpusWritesNoManifestWhenALaterPlanCannotBeRead(t *testing.T) {
	out := t.TempDir()
	plans := []string{writePlan(t, corpusPlan), filepath.Join(t.TempDir(), "missing.md"), writePlanAt(t, "other", "third.md", corpusPlan)}
	var stderr bytes.Buffer

	code := Corpus(context.Background(), &bytes.Buffer{}, &stderr, (&fakeTracker{}).commands(), CorpusOptions{Out: out, Plans: plans})

	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "missing.md")
	require.NoFileExists(t, filepath.Join(out, manifestName))
}

func TestCorpusKeysANamedPlanItemOnItsNameSoARewordKeepsItsRulings(t *testing.T) {
	named := strings.Replace(corpusPlan, "- [ ] Ship the thing", "- [ ] ship-it: Ship the thing", 1)
	out, ledgerPath := t.TempDir(), filepath.Join(t.TempDir(), "ticket-audits.jsonl")
	entries := corpusManifest(t, out, named)
	require.Equal(t, "repo/.documentation/plans/plan.md: ship-it", entries[1]["key"])
	records := filepath.Join(t.TempDir(), "records.json")
	require.NoError(t, os.WriteFile(records, []byte(`[{"key":"repo/.documentation/plans/plan.md: ship-it","methods":["premises"],"rulings":[{"finding":"f1","ruling":"r1"}]}]`), 0o600))
	require.NoError(t, Record(&bytes.Buffer{}, filepath.Join(out, manifestName), records, ledgerPath, ledgerNow))

	reworded := t.TempDir()
	corpusManifest(t, reworded, strings.Replace(named, "Ship the thing", "Release the thing", 1))
	var output bytes.Buffer
	require.NoError(t, WriteRulings(&output, filepath.Join(reworded, manifestName), ledgerPath))

	require.Equal(t, "repo/.documentation/plans/plan.md: ship-it\n  ruled 2026-10-21: f1 — r1\n  text CHANGED since\n", output.String())
}

func TestCorpusKeysAnInboxItemOnItsTitleAndAnActionOnItsName(t *testing.T) {
	plan := "## Actions\n\n- [ ] ship-it: Ship the thing\n  - done when shipped\n\n## Inbox\n\n- idea-x: try it\n- 2026-10-28: one\n- 2026-10-28: two\n"

	entries := corpusManifest(t, t.TempDir(), plan)

	require.Equal(t, []string{
		"repo/.documentation/plans/plan.md: ship-it",
		"repo/.documentation/plans/plan.md: idea-x: try it",
		"repo/.documentation/plans/plan.md: 2026-10-28: one",
		"repo/.documentation/plans/plan.md: 2026-10-28: two",
	}, fieldOf(entries, "key"))
}

func TestCorpusSplitsAPlanIntoActionsThenInboxItems(t *testing.T) {
	out := t.TempDir()

	entries := corpusManifest(t, out, corpusPlan)

	require.Equal(t, []string{"repo/.documentation/plans/plan.md: Build the thing", "repo/.documentation/plans/plan.md: Ship the thing", "repo/.documentation/plans/plan.md: An idea"}, fieldOf(entries, "key"))
	require.Equal(t, []string{"closed", "open", "inbox"}, fieldOf(entries, "state"))
	require.Equal(t, []string{"plan-01.md", "plan-02.md", "plan-03.md"}, fieldOf(entries, "file"))
	collapsed, err := os.ReadFile(filepath.Join(out, "plan-01.md"))
	require.NoError(t, err)
	require.Equal(t, "# plan-01 [closed] Build the thing\n- [x] Build the thing - as of 2026-10-17: the premise\n", string(collapsed), "sub-bullets are in the item; Context is not")
	raw, err := os.ReadFile(filepath.Join(out, "raw", "plan-01.md"))
	require.NoError(t, err)
	require.Contains(t, string(raw), "\n  - as of", "raw keeps the indent")

	unticked := corpusManifest(t, t.TempDir(), strings.Replace(corpusPlan, "- [x] Build", "- [ ] Build", 1))
	require.Equal(t, entries[0]["hash"], unticked[0]["hash"], "a tick is no change")
	edited := corpusManifest(t, t.TempDir(), strings.Replace(corpusPlan, "the premise", "a new premise", 1))
	require.NotEqual(t, entries[0]["hash"], edited[0]["hash"], "a sub-bullet edit is")
	require.Equal(t, entries[1]["hash"], edited[1]["hash"])
	require.Equal(t, []string{"closed"}, statesOf(planItems("## Inbox\n\n- [x] Done")), "a ticked Inbox item is closed")
}

func TestCorpusTakesSeveralPlansInTheOrderGiven(t *testing.T) {
	out := t.TempDir()
	plans := []string{writePlanAt(t, "repo", "second.md", "## Actions\n\n- [ ] ship-it: Ship\n"), writePlanAt(t, "other", "first.md", corpusPlan)}

	code := Corpus(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, (&fakeTracker{}).commands(), CorpusOptions{Out: out, Plans: plans})
	require.Equal(t, 0, code)
	content, err := os.ReadFile(filepath.Join(out, manifestName))
	require.NoError(t, err)
	var entries []map[string]string
	require.NoError(t, json.Unmarshal(content, &entries))

	require.Equal(t, []string{"second-01.md", "first-01.md", "first-02.md", "first-03.md"}, fieldOf(entries, "file"))
	require.Equal(t, "repo/.documentation/plans/second.md: ship-it", entries[0]["key"])
	require.Equal(t, "other/.documentation/plans/first.md: Build the thing", entries[1]["key"])
}

func TestCorpusWritesBothCopiesAndAManifestAndFlagsAPartialCorpus(t *testing.T) {
	body := "The endpoint takes only an email.\n\nfiller\n\nCorrection: the premise above is wrong."
	tracker := &fakeTracker{
		issues: map[int]fakeIssue{
			155: {body: "the map", title: "The map"},
			164: {body: "Part of nothing\n\n" + body + "\n\nBlocked by: #170, owner/repo#999 — see #155", comments: []string{"first comment"}, title: "Title 164"},
			170: {body: "Part of #MAP", state: "CLOSED", title: "Title 170"},
		},
		remote:   "git@github.com:owner/repo.git",
		searches: map[string][]map[string]any{"Part of #MAP": {{"number": 180, "body": "Part of #MAP"}, {"number": 181, "body": "Part of #MAP", "state": "CLOSED"}}},
		subIssue: map[int][]int{155: {164, 170, 777}},
	}
	out := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := Corpus(context.Background(), &stdout, &stderr, tracker.commands(), CorpusOptions{Map: 155, Out: out})

	require.Equal(t, 2, code, "a partial corpus is not a success")
	require.Equal(t, "#777: could not fetch — gh issue view 777 --json title,state,body failed: not found\n"+
		"note: #164 is a sub-issue but never says 'Part of #155'\n"+
		"note: #777 is a sub-issue but never says 'Part of #155'\n"+
		"warning: #164 says the placeholder 'Part of nothing', in this corpus\n"+
		"warning: #180 says the placeholder 'Part of #MAP', not in this corpus — pass it with --tickets if it belongs\n", stderr.String(), "a closed child's gap is quiet; one that could not be fetched is not known closed")
	collapsed, err := os.ReadFile(filepath.Join(out, "164.md"))
	require.NoError(t, err)
	require.Equal(t, "# 164 [open] Title 164\nPart of nothing The endpoint takes only an email. filler Correction: the premise above is wrong. Blocked by: #170, owner/repo#999 — see #155 first comment\n", string(collapsed))
	raw, err := os.ReadFile(filepath.Join(out, "raw", "164.md"))
	require.NoError(t, err)
	require.Contains(t, string(raw), "email.\n\nfiller\n\n")
	digest := sha1.New()
	for _, name := range []string{"155.md", "164.md", "170.md"} {
		content, readErr := os.ReadFile(filepath.Join(out, name))
		require.NoError(t, readErr)
		digest.Write(content)
	}
	require.Contains(t, stdout.String(), "\ncorpus: 155.md 164.md 170.md\ncorpus hash: "+hex.EncodeToString(digest.Sum(nil))+"\n", "cat <out>/*.md | shasum reproduces it")
	require.Contains(t, stdout.String(), "\n#164   open   #170 closed, #999 ?\n")
	require.True(t, strings.HasSuffix(stdout.String(), "\nFAILED to fetch: #777\nthe corpus is partial — an artefact can read as unproduced because its producer is missing\n"), stdout.String())
	manifest, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("[\n  {\n    \"file\": \"155.md\",\n    \"hash\": \"%s\",\n    \"key\": \"owner/repo#155\",\n    \"state\": \"open\"\n  },\n  {\n    \"file\": \"164.md\",\n    \"hash\": \"%s\",\n    \"key\": \"owner/repo#164\",\n    \"state\": \"open\"\n  },\n  {\n    \"file\": \"170.md\",\n    \"hash\": \"%s\",\n    \"key\": \"owner/repo#170\",\n    \"state\": \"closed\"\n  }\n]\n",
		itemHash("The map\nthe map"), itemHash("Title 164\n"+tracker.issues[164].body+"\n\nfirst comment"), itemHash("Title 170\nPart of #MAP")), string(manifest), "the hash covers title, body and comments")
}

func TestDetectCLIPicksTheRemotesTrackerAndRefusesAGuess(t *testing.T) {
	for _, check := range []struct {
		explicit  string
		installed []string
		remote    string
		want      string
	}{
		{installed: nil, remote: "https://example.com/x", want: "neither glab nor gh is on PATH"},
		{installed: []string{"gh"}, remote: "git@gitlab.com:group/project.git", want: "remote is gitlab but glab is not on PATH"},
		{installed: []string{"gh", "glab"}, remote: "", want: "remote 'unknown' names no tracker and both CLIs are installed — pass --cli glab|gh"},
		{installed: []string{"gh", "glab"}, remote: "git@gitlab.com:group/project.git", want: "glab"},
		{installed: []string{"gh", "glab"}, remote: "https://GitHub.com/owner/repo", want: "gh"},
		{installed: []string{"glab"}, remote: "https://example.com/x", want: "glab"},
		{explicit: "glab", installed: []string{"gh"}, remote: "git@github.com:o/r", want: "glab is not on PATH"},
	} {
		commands := Commands{
			LookPath: func(file string) (string, error) {
				if strings.Contains(strings.Join(check.installed, " "), file) {
					return "/bin/" + file, nil
				}
				return "", errors.New("not found")
			},
			Run: func(context.Context, string, ...string) (string, error) {
				if check.remote == "" {
					return "", errors.New("no remote")
				}
				return check.remote + "\n", nil
			},
		}

		cli, err := detectCLI(context.Background(), commands, check.explicit)

		if err != nil {
			cli = err.Error()
		}
		require.Equal(t, check.want, cli, check)
	}
}

func TestPlaceholderAndPartOfAreLiterals(t *testing.T) {
	require.Equal(t, "Part of #MAP", placeholder("see Part of #MAP."))
	require.Empty(t, placeholder("Part of #MAPPING"))
	require.Equal(t, "Part of nothing", placeholder("Part of nothing"))
	require.True(t, saysPartOf("Part of #87\n\nbody", 87))
	require.False(t, saysPartOf("Part of #871", 87))
	require.False(t, saysPartOf("Part of #87٣", 87), "a decimal digit in any script continues the number")
	require.False(t, saysPartOf("part of the map, see #87", 87))
}

func TestSequenceMatcherRatioAndDiffFollowAutojunkRules(t *testing.T) {
	require.InDelta(t, 0.75, newSequenceMatcher([]rune("abcd"), []rune("bcde")).ratio(), 1e-9)
	require.InDelta(t, 0.006600660066006601, newSequenceMatcher([]rune("aab"), []rune(strings.Repeat("ab", 150))).ratio(), 1e-12, "a character in more than 1% of a long b starts no match, though one can extend into it")
	var diff bytes.Buffer
	writeUnifiedDiff(&diff, splitLines("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n", true), splitLines("1\n2\nthree\n4\n5\n6\n7\n8\n9\nten", true), "a", "b")
	require.Equal(t, "--- a\n+++ b\n@@ -1,10 +1,10 @@\n 1\n 2\n-3\n+three\n 4\n 5\n 6\n 7\n 8\n 9\n-10\n+ten", diff.String())
}

func TestSplitLinesEndsALineAtEachUnicodeBreak(t *testing.T) {
	require.Equal(t, []string{"a", "b", "c", "d", "e", "", "f"}, splitLines("a\r\nb\rc d\x1ce\v\u0085f", false))
	require.Equal(t, []string{"a\r\n", "b"}, splitLines("a\r\nb", true))
	require.Equal(t, []string{"a\x1fb"}, splitLines("a\x1fb", false), "a unit separator ends no line")
}

func TestTrackerProjectReadsEitherRemoteForm(t *testing.T) {
	for remote, want := range map[string]string{
		"git@github.com:owner/repo.git":            "owner/repo",
		"https://github.com/owner/repo":            "owner/repo",
		"https://gitlab.com/group/sub/project.git": "group/sub/project",
	} {
		commands := Commands{Run: func(context.Context, string, ...string) (string, error) { return remote + "\n", nil }}

		project, err := trackerProject(context.Background(), commands)

		require.NoError(t, err, remote)
		require.Equal(t, want, project, remote)
	}
}

func (tracker *fakeTracker) commands() Commands {
	return Commands{
		LookPath: func(file string) (string, error) { return "/bin/" + file, nil },
		Run:      tracker.run,
	}
}

func (tracker *fakeTracker) run(_ context.Context, name string, args ...string) (string, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	switch {
	case call == "git remote get-url origin":
		return tracker.remote + "\n", nil
	case name == "git":
		return filepath.Dir(filepath.Dir(args[1])) + "\n", nil
	case len(args) > 2 && args[0] == "issue" && args[1] == "view":
		tracker.mutex.Lock()
		tracker.inFlight++
		tracker.most = max(tracker.most, tracker.inFlight)
		tracker.mutex.Unlock()
		time.Sleep(tracker.delay)
		tracker.mutex.Lock()
		tracker.inFlight--
		tracker.mutex.Unlock()
		var number int
		fmt.Sscan(args[2], &number)
		issue, ok := tracker.issues[number]
		if !ok {
			return "", fmt.Errorf("%s failed: not found", call)
		}
		state := issue.state
		if state == "" {
			state = "OPEN"
		}
		return encode(map[string]string{"body": issue.body, "state": state, "title": issue.title}), nil
	case args[0] == "api":
		var number int
		fmt.Sscanf(strings.TrimPrefix(args[1], "repos/{owner}/{repo}/issues/"), "%d", &number)
		var items []map[string]any
		if strings.Contains(args[1], "/sub_issues?") {
			for _, child := range tracker.subIssue[number] {
				items = append(items, map[string]any{"number": child})
			}
		} else {
			for _, comment := range tracker.issues[number].comments {
				items = append(items, map[string]any{"body": comment})
			}
		}
		if !strings.HasSuffix(args[1], "&page=1") {
			items = nil
		}
		return encode(items), nil
	case args[0] == "issue" && args[1] == "list":
		hits := tracker.searches[strings.Trim(args[5], `"`)]
		for _, hit := range hits {
			if hit["state"] == nil {
				hit["state"] = "OPEN"
			}
		}
		return encode(hits), nil
	}
	return "", fmt.Errorf("unexpected call: %s", call)
}

func corpusManifest(t *testing.T, out string, plan string) []map[string]string {
	t.Helper()
	code := Corpus(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, (&fakeTracker{}).commands(), CorpusOptions{Out: out, Plans: []string{writePlan(t, plan)}})
	require.Equal(t, 0, code)
	content, err := os.ReadFile(filepath.Join(out, manifestName))
	require.NoError(t, err)
	var entries []map[string]string
	require.NoError(t, json.Unmarshal(content, &entries))
	return entries
}

func encode(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func fieldOf(entries []map[string]string, field string) []string {
	var values []string
	for _, entry := range entries {
		values = append(values, entry[field])
	}
	return values
}

func statesOf(entries []planEntry) []string {
	var states []string
	for _, entry := range entries {
		states = append(states, entry.state)
	}
	return states
}

// The fake tracker's git names the directory two above the plan's as the top.
func writePlan(t *testing.T, text string) string {
	t.Helper()
	return writePlanAt(t, "repo", "plan.md", text)
}

func writePlanAt(t *testing.T, repository string, file string, text string) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "work", repository, ".documentation", "plans")
	require.NoError(t, os.MkdirAll(directory, 0o700))
	path := filepath.Join(directory, file)
	require.NoError(t, os.WriteFile(path, []byte(text), 0o600))
	return path
}
