package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

func TestRunSessionStartOutsideAnyOwner(t *testing.T) {
	home := recallFixture(t)
	elsewhere := filepath.Join(home, "elsewhere")
	require.NoError(t, os.MkdirAll(elsewhere, 0o700))
	t.Chdir(elsewhere)

	output, code := runCommand(t, "session-start", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Equal(t, "ele shows no notes here: no owner in owner-map.json covers "+elsewhere+". /ele:recall needs --owner <name> or --all.\n", output)
}

func TestRunSessionStartSaysWhenTheSessionCompacted(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	compacted := regexp.MustCompile(`(?m)^This session compacted at \d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ; what came before that is a summary, not the conversation\.$`)

	tests := []struct {
		input string
		name  string
		want  bool
	}{
		{input: `{"session_id":"s","source":"clear"}`, name: "clear", want: false},
		{input: `{"session_id":"s","source":"compact"}`, name: "compact", want: true},
		{input: "", name: "no hook input", want: false},
		{input: "not json", name: "not JSON", want: false},
		{input: `{"session_id":"s","source":"startup"}`, name: "startup", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			code := sessionStart(context.Background(), []string{"--no-sync"}, strings.NewReader(test.input), &output, &output)
			require.Equal(t, 0, code, output.String())
			require.Equal(t, test.want, compacted.MatchString(output.String()), output.String())
			require.Contains(t, output.String(), "ele has no notes for corp · gateway.\n")
		})
	}
}

func TestRunSessionStartShowsAHandoffOnce(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	start := func(source string) string {
		var output bytes.Buffer
		code := sessionStart(context.Background(), []string{"--no-sync"}, strings.NewReader(`{"session_id":"s","source":"`+source+`"}`), &output, &output)
		require.Equal(t, 0, code, output.String())
		return output.String()
	}
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "Reviews stay in subagents.")
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "remember", "--kind", "handoff", "--session", "abcdef0123", "Next: wire the reviewer agent type.")
	require.Equal(t, 0, code, output)
	first := readRemembered(t, output)

	require.NotContains(t, start("compact"), "Handoff", "a compaction continues the session that wrote it")
	require.NotContains(t, start("resume"), "Handoff")
	shown, message := hookOutput(t, start("startup"))
	written, err := time.Parse(time.RFC3339, first.Time)
	require.NoError(t, err)
	section := "Handoff from ele, note " + first.ID + " of " + written.UTC().Format("2006-01-02 15:04") + "Z, session abcdef01, shown once:\nNext: wire the reviewer agent type."
	require.Equal(t, section, message, "the user sees what the session loaded")
	require.True(t, strings.HasPrefix(shown, section+"\n\nNotes from ele for corp · gateway:\n- "+first.Date+" decision "), shown)
	require.Equal(t, 1, strings.Count(shown, first.ID), "not listed again among the notes")
	require.NotContains(t, start("clear"), "Handoff")

	output, code = runCommand(t, "remember", "--kind", "handoff", "Next: measure the checkpoint.")
	require.Equal(t, 0, code, output)
	second := readRemembered(t, output)
	output, code = runCommand(t, "reindex")
	require.Equal(t, 0, code, output)
	require.Contains(t, start("clear"), "note "+second.ID)
	output, code = runCommand(t, "reindex")
	require.Equal(t, 0, code, output)
	require.NotContains(t, start("startup"), "Handoff", "the mark is in the note's file, which a rebuild of the index keeps")
	content, err := os.ReadFile(filepath.Join(home, "data", "notes", "corp", second.ID+".md"))
	require.NoError(t, err)
	require.Regexp(t, `\nshown: \d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ\n---\n`, string(content))

	// Ids that sort against the times written.
	for _, note := range []notes.Note{
		{Body: "Older next task.\nResume from cmd/ele/mine.go.", Date: "2026-10-17", ID: "20261017-ffffff", Kind: notes.KindHandoff, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Time: "2026-10-17T09:00:00Z"},
		{Body: "Newer next task.", Date: "2026-10-17", ID: "20261017-000000", Kind: notes.KindHandoff, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Session: "0123456789", Time: "2026-10-17T10:00:00Z"},
	} {
		_, err = notes.Write(filepath.Join(home, "data", "notes"), note)
		require.NoError(t, err)
	}
	section = "2 handoffs from ele, newest first, each pending until picked or dropped:\n" +
		"- note 20261017-000000 of 2026-10-17 10:00Z, session 01234567:\n  Newer next task.\n" +
		"- note 20261017-ffffff of 2026-10-17 09:00Z:\n  Older next task.\n  Resume from cmd/ele/mine.go."
	for range 2 {
		shown, message = hookOutput(t, start("startup"))
		require.Equal(t, section, message, "each pending handoff, for the user to choose from, until one is closed")
		require.True(t, strings.HasPrefix(shown, section+"\nAsk the user which handoff to pick up and whether to drop any. Run ele handoff close <id>... for each picked or dropped; the rest return at the next startup or /clear.\n\n"), shown)
	}

	output, code = runCommand(t, "handoff", "close", "20261017-000000")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Closed handoff 20261017-000000.\n", output)
	_, message = hookOutput(t, start("clear"))
	require.Equal(t, "Handoff from ele, note 20261017-ffffff of 2026-10-17 09:00Z, shown once:\nOlder next task.\nResume from cmd/ele/mine.go.", message, "the one not picked returns")
	shown, message = hookOutput(t, start("startup"))
	require.Empty(t, message, "a lone handoff is used up once shown")
	require.NotContains(t, shown, "20261017-000000")
	require.NotContains(t, shown, "20261017-ffffff")
}

func TestRunSessionStartShowsActivePlansEveryTime(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	t.Chdir(gateway)
	start := func(source string) string {
		var output bytes.Buffer
		code := sessionStart(context.Background(), []string{"--no-sync"}, strings.NewReader(`{"session_id":"s","source":"`+source+`"}`), &output, &output)
		require.Equal(t, 0, code, output.String())
		return output.String()
	}
	long := strings.Repeat("  as of 2026-10-17: "+strings.Repeat("x", 80)+"\n", 30)
	writeFile(t, filepath.Join(gateway, "build.md"), "---\nkind: plan\n---\n\n## Actions\n\n- [x] Parser\n- [ ] Start block\n  token=abc123secret\n"+long+"\n## Inbox\n")
	output, code := runCommand(t, "plan", "start", "build.md")
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "remember", "--kind", "handoff", "--session", "", "Next: the start block.")
	require.Equal(t, 0, code, output)
	handoff := readRemembered(t, output)

	require.NotContains(t, start("resume"), "plan", "a resumed transcript already holds the block")
	section := "Active plan from ele:\nPlan build.md: 1 open, 0 blocked, 1 done, 0 in the inbox.\n1 warning; ele plan check build.md lists it.\nNext, line 8:\n- [ ] Start block\n  token=[redacted:assignment]\n"
	shown, message := hookOutput(t, start("startup"))
	require.True(t, strings.HasPrefix(message, "Handoff from ele, note "+handoff.ID), message)
	require.Contains(t, message, "Next: the start block.\n\n"+section)
	require.Equal(t, 19, strings.Count(message, "as of 2026-10-17"), "the body is cut at about 2,000 bytes")
	require.True(t, strings.HasSuffix(message, "\n  [11 lines of the body not shown; ele plan show prints it whole]"), message)
	require.Contains(t, shown, "[11 lines of the body not shown; ele plan show prints it whole]\nAsk the user whether to work this plan before starting anything; /ele:plan, which the user runs, picks it up.\n\nele has no notes", "the instruction is the session's, not the user's")
	for _, source := range []string{"clear", "compact"} {
		_, message = hookOutput(t, start(source))
		require.True(t, strings.HasPrefix(message, section), "a plan is not used up when shown: %s", message)
	}
	shown, _ = hookOutput(t, start("compact"))
	require.Contains(t, shown, "whole]\nIf this session was mid-task when it compacted, carry on, re-reading the plan item from its file if it was one. Otherwise ask the user whether to work this plan", "a compacted session continues its work rather than stopping to ask")

	writeFile(t, filepath.Join(gateway, "broken.md"), "---\nkind: plan\n---\n\n## Actions\n\n- [ ] Only item\n")
	output, code = runCommand(t, "plan", "start", "broken.md")
	require.Equal(t, 0, code, output)
	writeFile(t, filepath.Join(gateway, "broken.md"), "---\ntoken=abc123secret\n---\n")
	output, code = runCommand(t, "plan", "start", "gone.md")
	require.Equal(t, 1, code, output)
	_, err := notes.Write(filepath.Join(home, "data", "notes"), notes.Note{Body: "gone.md", Date: "2026-10-17", ID: "20261017-aaaaaa", Kind: notes.KindPlan, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject})
	require.NoError(t, err)

	shown, message = hookOutput(t, start("clear"))
	require.Regexp(t, `^3 active plans from ele, newest first:\n`, message)
	require.Contains(t, message, "\n- build.md: 1 open, 0 blocked, 1 done, 0 in the inbox. 1 warning; ele plan check lists it. Next: Start block\n")
	require.Contains(t, message, "\n- broken.md: invalid plan: front matter line ")
	require.Contains(t, message, "\n- plan gone.md is not in this checkout; switch to its branch, or ele plan done gone.md closes it")
	require.NotContains(t, message, "as of 2026-10-17", "several plans are one line each")
	require.NotContains(t, message, "abc123secret")
	require.Contains(t, shown, "\nAsk the user whether to work a plan, and which, before starting anything; ele plan show <path> prints one.\n\nele has no notes")
	shown, _ = hookOutput(t, start("compact"))
	require.Contains(t, shown, "\nIf this session was mid-task when it compacted, carry on, re-reading the plan item from its file if it was one. Otherwise ask the user whether to work a plan, and which,", "several plans on a compaction")

	output, code = runCommand(t, "plan", "done", "build.md", "gone.md")
	require.Equal(t, 0, code, output)
	shown, message = hookOutput(t, start("startup"))
	require.True(t, strings.HasPrefix(message, "ele: broken.md: invalid plan: front matter line "), message)
	require.NotContains(t, shown, "abc123secret", "a parse error quotes the file")
	require.NotContains(t, shown, "/ele:plan", "no instruction for a plan that cannot be read")
}

func TestRunHandoffCloseChecksEveryIdFirst(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "handoff", "--session", "", "Next: item 3.")
	require.Equal(t, 0, code, output)
	pending := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "decision", "--session", "", "Reviews stay in subagents.")
	require.Equal(t, 0, code, output)
	decision := readRemembered(t, output)

	for _, identifier := range []string{decision.ID, "20200101-000000"} {
		output, code = runCommand(t, "handoff", "close", pending.ID, identifier)
		require.Equal(t, 1, code, output)
		require.Equal(t, "ele: "+identifier+" is not a pending handoff of corp · gateway; ele context lists them, --project <name> for another project's\n", output)
	}
	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "handoff "+pending.ID, "a refused close marks none")

	output, code = runCommand(t, "handoff", "close", pending.ID, pending.ID)
	require.Equal(t, 0, code, output)
	require.Equal(t, "Closed handoff "+pending.ID+".\n", output, "a repeated id closes once")
	output, code = runCommand(t, "handoff", "close", pending.ID)
	require.Equal(t, 1, code, "a closed handoff is no longer pending: %s", output)
	output, code = runCommand(t, "handoff", "close")
	require.Equal(t, 2, code, output)
}

func TestRunHandoffListShowsEachHandoffsState(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	list := func(args ...string) string {
		output, code := runCommand(t, append([]string{"handoff", "list"}, args...)...)
		require.Equal(t, 0, code, output)
		return output
	}
	require.Equal(t, "ele has no pending handoffs for corp · gateway; --all adds closed and superseded ones.\n", list())
	require.Equal(t, "ele has no handoffs for corp · gateway.\n", list("--all"))

	for _, note := range []notes.Note{
		{Body: "Superseded task.", Date: "2026-10-10", ID: "20261010-dddddd", Kind: notes.KindHandoff, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Shown: "2026-10-10T12:00:00Z"},
		{Body: "Done another way.", Date: "2026-10-11", ID: "20261011-eeeeee", Kind: notes.KindFact, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Supersedes: []string{"20261010-dddddd"}},
		{Body: "Closed task.\nSecond line.", Date: "2026-10-12", ID: "20261012-cccccc", Kind: notes.KindHandoff, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Shown: "2026-10-12T11:02:00Z", Time: "2026-10-12T09:00:00Z"},
		{Body: "Pending task.", Date: "2026-10-14", ID: "20261014-aaaaaa", Kind: notes.KindHandoff, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Session: "0123456789", Time: "2026-10-14T10:00:00Z"},
		{Body: "Another project's.", Date: "2026-10-14", ID: "20261014-bbbbbb", Kind: notes.KindHandoff, Owner: "corp", Project: "proxy", Scope: notes.ScopeProject},
		{Body: "Older pending task.", Date: "2026-10-14", ID: "20261014-ffffff", Kind: notes.KindHandoff, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Time: "2026-10-14T08:00:00Z"},
	} {
		_, err := notes.Write(filepath.Join(home, "data", "notes"), note)
		require.NoError(t, err)
	}
	footer := "ele handoff close <id>... or ele handoff reopen <id>... to change one.\n"
	pending := "- pending  note 20261014-aaaaaa of 2026-10-14 10:00Z, session 01234567: Pending task.\n" +
		"- pending  note 20261014-ffffff of 2026-10-14 08:00Z: Older pending task.\n"
	require.Equal(t, "Handoffs for corp · gateway, newest first:\n"+pending+footer, list())
	require.Equal(t, "Handoffs for corp · gateway, newest first:\n"+pending+
		"- closed 2026-10-12T11:02:00Z  note 20261012-cccccc of 2026-10-12 09:00Z: Closed task. Second line.\n"+
		"- superseded by 20261011-eeeeee  note 20261010-dddddd of 2026-10-10: Superseded task.\n"+footer, list("--all"))
	require.Contains(t, list("--owner", "corp", "--project", "proxy"), "note 20261014-bbbbbb")

	output, code := runCommand(t, "handoff", "list", "extra")
	require.Equal(t, 2, code, output)
}

func TestRunHandoffReopenChecksEveryIdFirst(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "handoff", "--session", "", "Next: item 3.")
	require.Equal(t, 0, code, output)
	closed := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "handoff", "--session", "", "Next: item 4.")
	require.Equal(t, 0, code, output)
	pending := readRemembered(t, output)
	output, code = runCommand(t, "handoff", "close", closed.ID)
	require.Equal(t, 0, code, output)
	_, err := notes.Write(filepath.Join(home, "data", "notes"), notes.Note{Body: "Old next task.", Date: "2026-10-01", ID: "20261001-aaaaaa", Kind: notes.KindHandoff, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Shown: "2026-10-01T10:00:00Z"})
	require.NoError(t, err)
	_, err = notes.Write(filepath.Join(home, "data", "notes"), notes.Note{Body: "Done instead.", Date: "2026-10-02", ID: "20261002-bbbbbb", Kind: notes.KindFact, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Supersedes: []string{"20261001-aaaaaa"}})
	require.NoError(t, err)

	for _, identifier := range []string{pending.ID, "20261001-aaaaaa", "20200101-000000"} {
		output, code = runCommand(t, "handoff", "reopen", closed.ID, identifier)
		require.Equal(t, 1, code, output)
		require.Equal(t, "ele: "+identifier+" is not a closed handoff of corp · gateway; ele handoff list --all lists them, --project <name> for another project's\n", output)
	}
	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, closed.ID, "a refused reopen opens none")

	output, code = runCommand(t, "handoff", "reopen", closed.ID, closed.ID)
	require.Equal(t, 0, code, output)
	require.Equal(t, "Reopened handoff "+closed.ID+".\n", output, "a repeated id reopens once")
	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "handoff "+closed.ID)
	output, code = runCommand(t, "handoff", "reopen")
	require.Equal(t, 2, code, output)
}

func TestRunSessionStartColoursOnlyTheUsersCopyUnlessColourIsOff(t *testing.T) {
	home := recallFixture(t)
	t.Setenv("NO_COLOR", "")
	gateway := filepath.Join(home, "w", "corp", "gateway")
	t.Chdir(gateway)
	writeFile(t, filepath.Join(gateway, "build.md"), "---\nkind: plan\n---\n\n## Actions\n\n- [x] Parser\n- [ ] Start \033[31mblock\n\n## Inbox\n")
	output, code := runCommand(t, "plan", "start", "build.md")
	require.Equal(t, 0, code, output)

	var raw bytes.Buffer
	require.Equal(t, 0, sessionStart(context.Background(), []string{"--no-sync"}, strings.NewReader(`{"session_id":"s","source":"startup"}`), &raw, &raw), raw.String())
	shown, message := hookOutput(t, raw.String())
	require.Equal(t, "\033[1mActive plan from ele:\033[22m\n\033[1mPlan\033[22m build.md: \033[33m1 open\033[39m, \033[2m0 blocked\033[22m, \033[32m1 done\033[39m, \033[2m0 in the inbox\033[22m.\n\033[33m1 warning\033[39m; \033[36mele plan check build.md\033[39m lists it.\n\033[1mNext, line 8:\033[22m\n- [ ] Start [31mblock", message, "a plan cannot send its own escape sequences")
	require.Contains(t, shown, "Plan build.md: 1 open, 0 blocked, 1 done, 0 in the inbox.\n1 warning; ele plan check build.md lists it.\nNext, line 8:\n- [ ] Start \033[31mblock\n", "the session's copy is plain")

	plain := "Active plan from ele:\nPlan build.md: 1 open, 0 blocked, 1 done, 0 in the inbox.\n1 warning; ele plan check build.md lists it.\nNext, line 8:\n- [ ] Start [31mblock"
	writeFile(t, filepath.Join(home, "data", "config.json"), `{"owners": {"corp": {"colour": false}}}`)
	raw.Reset()
	require.Equal(t, 0, sessionStart(context.Background(), []string{"--no-sync"}, strings.NewReader(`{"session_id":"s","source":"startup"}`), &raw, &raw), raw.String())
	_, message = hookOutput(t, raw.String())
	require.Equal(t, plain, message, "colour off for this owner")

	require.NoError(t, os.Remove(filepath.Join(home, "data", "config.json")))
	t.Setenv("NO_COLOR", "1")
	raw.Reset()
	require.Equal(t, 0, sessionStart(context.Background(), []string{"--no-sync"}, strings.NewReader(`{"session_id":"s","source":"startup"}`), &raw, &raw), raw.String())
	_, message = hookOutput(t, raw.String())
	require.Equal(t, plain, message, "NO_COLOR with colour on")
}

func hookOutput(t *testing.T, raw string) (additionalContext string, systemMessage string) {
	t.Helper()
	if !strings.HasPrefix(raw, "{") {
		return raw, ""
	}
	var output struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
			HookEventName     string `json:"hookEventName"`
		} `json:"hookSpecificOutput"`
		SystemMessage string `json:"systemMessage"`
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&output), raw)
	require.False(t, decoder.More(), "one JSON object and nothing after it: %s", raw)
	require.Equal(t, "SessionStart", output.HookSpecificOutput.HookEventName)
	return output.HookSpecificOutput.AdditionalContext, output.SystemMessage
}

func TestRunSessionStartShowsOpenQuestionsThenNewestNotes(t *testing.T) {
	home := recallFixture(t)
	noteDirectory := filepath.Join(home, "data", "notes")
	for _, note := range []notes.Note{
		{Body: "Should ele index\ntool output?", Date: "2026-10-10", ID: "20261010-aaaaaa", Kind: notes.KindQuestion, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject},
		{Body: "Old question.", Date: "2026-10-09", ID: "20261009-bbbbbb", Kind: notes.KindQuestion, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject},
		{Body: "Decided: no.", Date: "2026-10-12", ID: "20261012-cccccc", Kind: notes.KindDecision, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Supersedes: []string{"20261009-bbbbbb"}},
		{Body: "Pin pydantic below 2.14; token=abc123secret", Date: "2026-10-14", ID: "20261014-dddddd", Kind: notes.KindDecision, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject},
		{Body: "Corp deploys on Tuesdays.", Date: "2026-10-13", ID: "20261013-eeeeee", Kind: notes.KindFact, Owner: "corp", Project: "proxy", Scope: notes.ScopeGlobal},
		{Body: "Proxy only.", Date: "2026-10-15", ID: "20261015-ffffff", Kind: notes.KindDecision, Owner: "corp", Project: "proxy", Scope: notes.ScopeProject},
		{Body: "Personal only.", Date: "2026-10-15", ID: "20261015-111111", Kind: notes.KindDecision, Owner: "personal", Scope: notes.ScopeGlobal},
	} {
		_, err := notes.Write(noteDirectory, note)
		require.NoError(t, err)
	}
	writeFile(t, filepath.Join(noteDirectory, "corp", "broken.md"), "no front matter\n")
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))

	output, code := runCommand(t, "session-start", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Notes from ele for corp · gateway:\n"+
		"- 2026-10-10 question 20261010-aaaaaa: Should ele index tool output?\n"+
		"- 2026-10-14 decision 20261014-dddddd: Pin pydantic below 2.14; token=[redacted:assignment]\n"+
		"- 2026-10-13 fact 20261013-eeeeee: Corp deploys on Tuesdays.\n"+
		"- 2026-10-12 decision 20261012-cccccc: Decided: no.\n"+
		"1 invalid note skipped; ele sync names each; 1 past session indexed for gateway; search with /ele:recall.\n", output)
}

func TestWriteNotesHintsOnlyForShownSessions(t *testing.T) {
	question := notes.Note{Body: "Open?", Date: "2026-10-15", ID: "20261015-qqqqqq", Kind: notes.KindQuestion, Scope: notes.ScopeGlobal, Session: "abc1"}
	var output bytes.Buffer
	writeNotes(&output, "corp", "gateway", []notes.Note{question}, 0, 0, blockBudget)
	require.Contains(t, output.String(), "(session abc1)")
	require.True(t, strings.HasSuffix(output.String(), sessionHint), output.String())

	var all []notes.Note
	for i := range 40 {
		all = append(all, notes.Note{Body: strings.Repeat("word ", 30), Date: "2026-10-15", ID: fmt.Sprintf("n%02d", i), Kind: notes.KindFact, Scope: notes.ScopeGlobal})
	}
	output.Reset()
	writeNotes(&output, "corp", "gateway", all, 0, 0, blockBudget)
	shown := strings.Count(output.String(), " fact n")
	require.Less(t, shown, len(all))
	// Newest first by id, so the first note the budget cuts is the one at index len-1-shown.
	all[len(all)-1-shown].Session = "abc1"
	output.Reset()
	writeNotes(&output, "corp", "gateway", all, 0, 0, blockBudget)
	require.NotContains(t, output.String(), sessionHint)

	var crowded []notes.Note
	for i := range 200 {
		crowded = append(crowded, notes.Note{Body: "x", Date: "2026-10-15", ID: fmt.Sprintf("n%03d", i), Kind: notes.KindFact, Scope: notes.ScopeGlobal, Session: "abc1"})
	}
	output.Reset()
	writeNotes(&output, "corp", strings.Repeat("long-project-name-", 3), crowded, 3, 12, blockBudget)
	require.True(t, strings.HasSuffix(output.String(), sessionHint), output.String())
	require.LessOrEqual(t, output.Len(), contextBudget)
}

func TestWriteNotesNamesEachNotesSession(t *testing.T) {
	var output bytes.Buffer
	writeNotes(&output, "corp", "gateway", []notes.Note{
		{Body: "With a session.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Scope: notes.ScopeGlobal, Session: "010a2385-feef-4623-a164-763f5a52ec9b"},
		{Body: "Without.", Date: "2026-10-14", ID: "20261014-bbbbbb", Kind: notes.KindFact, Scope: notes.ScopeGlobal},
	}, 0, 0, blockBudget)
	require.Equal(t, "Notes from ele for corp · gateway:\n"+
		"- 2026-10-15 decision 20261015-aaaaaa (session 010a2385): With a session.\n"+
		"- 2026-10-14 fact 20261014-bbbbbb: Without.\n"+
		"search with /ele:recall.\n"+
		"A note's discussion: ele show --note <id>.\n", output.String())
}

func TestWriteNotesCutsOldestQuestionsPastTheRoomLeft(t *testing.T) {
	var all []notes.Note
	for i := range 40 {
		all = append(all, notes.Note{Body: strings.Repeat("😀 open ", 40), Date: fmt.Sprintf("2026-09-%02d", i%28+1), ID: fmt.Sprintf("q%02d", i), Kind: notes.KindQuestion, Scope: notes.ScopeGlobal})
	}
	all = append(all, notes.Note{Body: "Decided.", Date: "2026-10-15", ID: "d00", Kind: notes.KindDecision, Scope: notes.ScopeGlobal})
	var output bytes.Buffer
	writeNotes(&output, "corp", "gateway", all, 0, 12, 5000)

	text := output.String()
	require.LessOrEqual(t, len(utf16.Encode([]rune(text))), 5000)
	shown := len(regexp.MustCompile(`(?m)^- \S+ question q\d+:`).FindAllString(text, -1))
	require.Positive(t, shown)
	require.Less(t, shown, 40)
	require.Contains(t, text, "question q27:", "the newest question is kept")
	require.NotContains(t, text, "decision d00:")
	require.True(t, strings.HasSuffix(text, fmt.Sprintf("1 older note not shown; %d older questions not shown, ele context lists them; 12 past sessions indexed for gateway; search with /ele:recall.\n", 40-shown)), text)
}

func TestWriteNotesFitsTheLongestClosingLine(t *testing.T) {
	var all []notes.Note
	for i := range 200 {
		all = append(all, notes.Note{Body: "Open?", Date: "2026-10-15", ID: fmt.Sprintf("q%03d", i), Kind: notes.KindQuestion, Scope: notes.ScopeGlobal, Session: "abc1"})
	}
	all = append(all, notes.Note{Body: "Decided.", Date: "2026-10-15", ID: "d00", Kind: notes.KindDecision, Scope: notes.ScopeGlobal})
	var output bytes.Buffer
	writeNotes(&output, "corp", strings.Repeat("long-project-name-", 6), all, 3, 140, 3000)
	require.Contains(t, output.String(), "older questions not shown, ele context lists them; 3 invalid notes skipped")
	require.True(t, strings.HasSuffix(output.String(), sessionHint))
	require.LessOrEqual(t, len(utf16.Encode([]rune(output.String()))), 3000)
}

func TestWriteSessionStartFitsHookOutputCap(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	dataDirectory := filepath.Join(home, "data")
	noteDirectory := filepath.Join(dataDirectory, "notes")
	_, err := notes.Write(noteDirectory, notes.Note{Body: "Run the measurement.", Date: "2026-10-01", Due: "2026-10-01", ID: "20261001-rrrrrr", Kind: notes.KindReminder, Owner: "corp", Scope: notes.ScopeGlobal})
	require.NoError(t, err)
	for i := range 60 {
		_, err = notes.Write(noteDirectory, notes.Note{Body: strings.Repeat("Should ele index tool output? ", 10), Date: "2026-10-10", ID: fmt.Sprintf("20261010-q%05d", i), Kind: notes.KindQuestion, Owner: "corp", Scope: notes.ScopeGlobal})
		require.NoError(t, err)
	}

	var output bytes.Buffer
	_, err = writeSessionStart(context.Background(), &output, dataDirectory, time.Now(), "startup")
	require.NoError(t, err)
	require.LessOrEqual(t, len(utf16.Encode([]rune(output.String()))), blockBudget)
	require.Regexp(t, `\d+ older questions not shown, ele context lists them`, output.String())
}

func TestWriteNotesCutsDecisionsBeforeQuestions(t *testing.T) {
	var all []notes.Note
	for i := range 40 {
		all = append(all, notes.Note{Body: strings.Repeat("decision ", 20), Date: fmt.Sprintf("2026-09-%02d", i%28+1), ID: fmt.Sprintf("d%02d", i), Kind: notes.KindDecision, Scope: notes.ScopeGlobal})
	}
	for i := range 10 {
		all = append(all, notes.Note{Body: strings.Repeat("question ", 20), Date: "2026-01-01", ID: fmt.Sprintf("q%02d", i), Kind: notes.KindQuestion, Scope: notes.ScopeGlobal})
	}
	var output bytes.Buffer
	writeNotes(&output, "corp", "gateway", all, 0, 0, blockBudget)

	text := output.String()
	require.Len(t, regexp.MustCompile(`(?m)^- \S+ question q\d+:`).FindAllString(text, -1), 10)
	shown := len(regexp.MustCompile(`(?m)^- \S+ decision d\d+:`).FindAllString(text, -1))
	require.Positive(t, shown)
	require.Less(t, shown, 40)
	require.Contains(t, text, fmt.Sprintf("%d older notes not shown", 40-shown))
	require.Contains(t, text, "decision d27:", "the newest decision is kept")
}

func TestWriteNotesStaysWithinTheBudget(t *testing.T) {
	var all []notes.Note
	for i := range 200 {
		all = append(all, notes.Note{Body: strings.Repeat("word ", 30), Date: "2026-10-15", ID: fmt.Sprintf("n%03d", i), Kind: notes.KindFact, Scope: notes.ScopeGlobal})
	}
	var output bytes.Buffer
	writeNotes(&output, "corp", "gateway", all, 0, 12, blockBudget)
	require.LessOrEqual(t, output.Len(), contextBudget)
	require.True(t, strings.HasSuffix(output.String(), "older notes not shown; 12 past sessions indexed for gateway; search with /ele:recall.\n"))
}

func TestWriteSessionStartKeepsNotesWhenTheIndexFails(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	dataDirectory := filepath.Join(home, "data")
	_, err := notes.Write(filepath.Join(dataDirectory, "notes"), notes.Note{Body: "Pin pydantic.", Date: "2026-10-14", ID: "20261014-dddddd", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal})
	require.NoError(t, err)
	database := filepath.Join(dataDirectory, "ele.db")
	require.NoError(t, os.Remove(database))
	require.NoError(t, os.Mkdir(database, 0o700))

	var output bytes.Buffer
	_, err = writeSessionStart(context.Background(), &output, dataDirectory, time.Now(), "startup")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(output.String(), "ele could not count past sessions: "), output.String())
	require.True(t, strings.HasSuffix(output.String(), "\n- 2026-10-14 decision 20261014-dddddd: Pin pydantic.\nsearch with /ele:recall.\n"), output.String())
}

func TestWriteSessionStartWarnsWhenSyncIsStale(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	dataDirectory := filepath.Join(home, "data")
	stamp := filepath.Join(dataDirectory, "last-sync")
	tail := "; " + filepath.Join(dataDirectory, "sync.log") + " holds the background syncs' output, newest last.\n"

	var output bytes.Buffer
	_, err := writeSessionStart(context.Background(), &output, dataDirectory, time.Now(), "startup")
	require.NoError(t, err)
	require.NotContains(t, output.String(), "sync")

	output.Reset()
	_, err = writeSessionStart(context.Background(), &output, dataDirectory, time.Now().Add(25*time.Hour), "startup")
	require.NoError(t, err)
	require.Regexp(t, `^ele last synced \d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ`+regexp.QuoteMeta(tail), output.String())

	writeFile(t, stamp, "garbage\n")
	output.Reset()
	_, err = writeSessionStart(context.Background(), &output, dataDirectory, time.Now(), "startup")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(output.String(), "ele's sync stamp is malformed"+tail), output.String())

	require.NoError(t, os.Remove(stamp))
	output.Reset()
	_, err = writeSessionStart(context.Background(), &output, dataDirectory, time.Now(), "startup")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(output.String(), "ele has no record of a successful sync"+tail), output.String())
}
