package plan

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const sample = `---
kind: plan
created: 2026-10-17
spec: .documentation/specs/design.md
---

# Plans

## Actions

- [x] Move the spec
  asserted: the spec reads from its new path
- [ ] Parser and commands
  - as of 2026-10-17: parser written
  next: commands

### Later

- [ ] Start block section
  blocked: the parser first
  - blocked: a real session to test on
- [ ] Skill
` + "  ```\n## Not a heading\n  ```\n" + `
## Inbox

- 2026-10-17, from a checkpoint: trim the help text
- 2026-10-17: rename the flag

## Out of scope

- Colour themes: NO_COLOR covers it (ruled 2026-10-18, from the inbox)
- [ ] Not an action
`

func TestCheckSaysWhenAPlanIsReadyToClose(t *testing.T) {
	tests := []struct {
		content string
		name    string
		want    []Warning
	}{
		{content: "- [x] a: A\n- [x] Close\n", name: "closed"},
		{content: "- [ ] Close\n", name: "closing alone"},
		{content: "- [x] a: A\n- [ ] Close\n## Inbox\n- 2026-10-21, new\n", name: "inbox first", want: []Warning{{Line: 6, Text: "every action but the closing one is done; ready to close once the inbox is triaged"}}},
		{content: "- [ ] a: A\n  done when it is\n- [ ] Close\n", name: "open work"},
		{content: "- [x] a: A\n- [ ] Close\n", name: "ready", want: []Warning{{Line: 6, Text: "every action but the closing one is done; ready to close"}}},
		{content: "- [x] a: A\n- [x] Close\n## Inbox\n- 2026-10-21, new\n", name: "ticked over an inbox", want: []Warning{{Line: 6, Text: "the closing action is ticked while the inbox holds items; triage them first"}}},
		{content: "- [ ] a: A\n  done when it is\n- [x] Close\n## Inbox\n- 2026-10-21, new\n", name: "ticked over an inbox and open work", want: []Warning{{Line: 7, Text: "the closing action is ticked while other actions are open"}, {Line: 7, Text: "the closing action is ticked while the inbox holds items; triage them first"}}},
		{content: "- [ ] a: A\n  done when it is\n- [x] Close\n", name: "ticked over open work", want: []Warning{{Line: 7, Text: "the closing action is ticked while other actions are open"}}},
	}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.Local)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n" + test.content))
			require.NoError(t, err)
			require.Equal(t, test.want, parsed.Resolve(now, nil).Check(now))
		})
	}
}

func TestCheckWarnsOnAnOpenUnnamedAction(t *testing.T) {
	tests := []struct {
		content string
		name    string
		want    []Warning
	}{
		{content: "- [ ] a: A\n  done when it is\n- [ ] Close\n", name: "closing"},
		{content: "- [ ] Unnamed\n  done when it is\n- [ ] Close\n", name: "open", want: []Warning{{Line: 5, Text: "action has no name; lead its title with a kebab-case name and a colon"}}},
		{content: "- [x] Unnamed\n- [ ] b: B\n  done when it is\n- [ ] Close\n", name: "ticked"},
	}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.Local)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n" + test.content))
			require.NoError(t, err)
			require.Equal(t, test.want, parsed.Resolve(now, nil).Check(now))
		})
	}
}

func TestCheckWarnsOnEachLineFormItHoldsOrDrops(t *testing.T) {
	tests := []struct {
		content string
		name    string
		want    []Warning
	}{
		{content: "- [ ] a: A\n  - done when x, and `- [ ] b: B` in code\n- [ ] Close\n", name: "checkbox in a code span"},
		{content: "- [ ] a: A\n  - done when x\n  - blocked: b\n- [ ] b: B\n  - done when x\n  - blocked: a\n- [ ] Close\n", name: "cycle", want: []Warning{{Line: 5, Text: "a: blocked in a cycle, a → b → a, which no tick can free"}, {Line: 8, Text: "b: blocked in a cycle, b → a → b, which no tick can free"}}},
		{content: "- [ ] a: A\n  - done when x\n  - blocked:\n- [ ] Close\n", name: "empty blocked", want: []Warning{{Line: 7, Text: "a: blocked: names no item; it holds an open item until fixed"}}},
		{content: "- [x] a: A\n- [ ] Close\n## Inbox\n* 2026-10-21 star\n+ 2026-10-21 plus\n1. 2026-10-21 numbered\n", name: "inbox lines", want: []Warning{
			{Line: 6, Text: "every action but the closing one is done; ready to close"},
			{Line: 8, Text: "inbox item is not a - line, so ele does not count it"},
			{Line: 9, Text: "inbox item is not a - line, so ele does not count it"},
			{Line: 10, Text: "inbox item is not a - line, so ele does not count it"},
		}},
		{content: "- [ ] a: A\n  - done when x- [ ] b: B\n  - done when y\n- [ ] Close\n", name: "merged header", want: []Warning{{Line: 6, Text: "a checkbox follows other text, so ele reads no item there; put it on a line of its own"}}},
		{content: "- [ ] a: A\n  - done when x\n  - blocked: c d\n- [ ] c: C\n  - done when x\n- [ ] d: D\n  - done when x\n- [ ] Close\n", name: "names without commas", want: []Warning{{Line: 5, Text: `a: blocked: "c d" reads as prose; separate its names with commas`}}},
		{content: "1. [ ] a: A\n  - done when x\n- [ ] Close\n", name: "numbered item", want: []Warning{{Line: 5, Text: "item is not a - [ ] or - [x] line, so ele does not read it"}}},
		{content: "+ [x] a: A\n- [ ] Close\n", name: "plus item", want: []Warning{{Line: 5, Text: "item is not a - [ ] or - [x] line, so ele does not read it"}}},
		{content: "- [ ] a: A\n  - done when x\n  - blocked: a\n- [ ] Close\n", name: "self-block", want: []Warning{{Line: 5, Text: "a: blocked in a cycle, a → a, which no tick can free"}}},
		{content: "* [ ] a: A\n  - done when x\n- [ ] Close\n", name: "star item", want: []Warning{{Line: 5, Text: "item is not a - [ ] or - [x] line, so ele does not read it"}}},
		{content: "- [x] a: A\n  - blocked: b\n- [ ] b: B\n  - done when x\n- [ ] Close\n", name: "ticked over an open blocker", want: []Warning{{Line: 5, Text: "a: ticked while blocker b is open"}}},
		{content: "- [ ] a: A\n  - done when x\n  - blocked: c,\n- [ ] c: C\n  - done when x\n- [ ] Close\n", name: "trailing comma", want: []Warning{{Line: 7, Text: `a: blocked: "c," is not a list of item names; it holds an open item until fixed`}}},
		{content: "- [ ] a: A\n  - done when x\n  blocked: c,\n  after: 2099-01-01\n- [ ] c: C\n  - done when x\n- [ ] Close\n", name: "trailing comma before a bare after", want: []Warning{{Line: 7, Text: `a: blocked: "c," is not a list of item names; it holds an open item until fixed`}}},
		{content: "- [ ] a: A\n  blocked: c,\n  done when x\n- [ ] c: C\n  - done when x\n- [ ] Close\n", name: "trailing comma before a bare done when", want: []Warning{{Line: 6, Text: `a: blocked: "c," is not a list of item names; it holds an open item until fixed`}}},
		{content: "- [ ] a: A\n  - blocked: c,\n  ```\n  - done when x\n  ```\n- [ ] c: C\n  - done when x\n- [ ] Close\n", name: "trailing comma before a fence", want: []Warning{{Line: 5, Text: "a: no done when line"}, {Line: 6, Text: `a: blocked: "c," is not a list of item names; it holds an open item until fixed`}}},
		{content: "- [ ] a: A\n  - blocked: c,\n  - done when x\n- [ ] c: C\n  - done when x\n- [ ] Close\n", name: "trailing comma before a list item", want: []Warning{{Line: 6, Text: `a: blocked: "c," is not a list of item names; it holds an open item until fixed`}}},
	}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.Local)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n" + test.content))
			require.NoError(t, err)
			require.Equal(t, test.want, parsed.Resolve(now, nil).Check(now))
		})
	}
}

func TestCheckWarnsOnEleData(t *testing.T) {
	tests := []struct {
		line string
		name string
		warn bool
	}{
		{line: "  - asserted in 0cc19d4 and 2c1db93a: kept", name: "commit sha"},
		{line: "  - from the 2026-10-22 audit", name: "date"},
		{line: "  - ele show added in 0cc19d4", name: "ele show in prose"},
		{line: "  - ele show --note 20261020-ead7f7 holds it", name: "ele show of a note", warn: true},
		{line: "  - ele show --note 20261020 holds it", name: "ele show of a note prefix", warn: true},
		{line: "  - ele show 24469c4a --at <time> opened it", name: "ele show of a session", warn: true},
		{line: "  - ele show --at \"2026-10-22 10:00\" --context 5 24469c4a opened it", name: "ele show of a session after flags", warn: true},
		{line: "  - first captured from question note 20261020-ead7f7", name: "note id", warn: true},
		{line: "  - run ele show --note <id>", name: "note placeholder"},
		{line: "  - run ele show <session> --at <time>", name: "session placeholder"},
		{line: "  - observed in session a5491a94-1c2d-4e5f-8a9b-0c1d2e3f4a5b", name: "session uuid", warn: true},
	}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.Local)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [x] a: A\n" + test.line + "\n- [ ] b: B\n  done when it is\n- [ ] Close\n"))
			require.NoError(t, err)
			var want []Warning
			if test.warn {
				want = []Warning{{Line: 6, Text: "cites ele data, which resolves only in its author's data directory; delete provenance, describe what was observed, and move a dependency into the plan, superseding its note"}}
			}
			require.Equal(t, want, parsed.Resolve(now, nil).Check(now))
		})
	}
}

func TestCheckWarnsOnStaleClaimsAndInboxItemsAndMissingDoneWhens(t *testing.T) {
	content := "---\nkind: plan\n---\n## Context (as of 2026-09-01)\n" +
		"- As of 2026-10-01: old fact\n" +
		"- as of 2026-10-10: fresh enough\n" +
		"## Actions\n" +
		"- [x] done-item: Done\n  as of 2026-09-01: history\n" +
		"- [ ] no-when: No done-when\n  as of 2026-10-01 and as of 2026-09-30: two dates\n  ```\n  as of 2026-01-01: fenced\n  ```\n" +
		"- [ ] with-when: Has one\n  - done when it is\n  after: 2026-10-20\n  blocked: nope\n" +
		"- [ ] Closing action\n" +
		"## Inbox\n- 2026-10-20, fresh\n- 2026-10-10, stale\n- undated\n" +
		"## Out of scope\n- as of 2026-01-01 ruled out\n### Dropped (as of 2026-01-01)\n"
	parsed, err := Parse([]byte(content))
	require.NoError(t, err)
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.Local)
	passed := time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local).Format("2006-01-02 15:04 -07:00")
	require.Equal(t, []Warning{
		{Line: 4, Text: "as of 2026-09-01 is 50 days old; re-measure it"},
		{Line: 5, Text: "as of 2026-10-01 is 20 days old; re-measure it"},
		{Line: 10, Text: "no-when: no done when line"},
		{Line: 11, Text: "as of 2026-09-30 is 21 days old; re-measure it"},
		{Line: 15, Text: "with-when: after: " + passed + " has passed; drop it"},
		{Line: 15, Text: "with-when: blocker nope does not exist"},
		{Line: 22, Text: "inbox item from 2026-10-10 is 11 days old; triage it"},
		{Line: 23, Text: "inbox item has no date; lead it with the date and where it came from"},
	}, parsed.Resolve(now, nil).Check(now))
}

func TestCheckWarnsOnWhatItCannotReadAndHoldsALooseAfter(t *testing.T) {
	plan, err := Parse([]byte("---\nkind: plan\ncreated: 17 Oct\nstatus: open\n---\n## Actions\n- [ ] a: A\n  - done when x\n  after: soon\n- [ ] b: B\n  - done when y\n- [ ] Close\n"))
	require.NoError(t, err)
	require.Empty(t, plan.Created)
	resolved := plan.Resolve(time.Now(), nil)
	require.True(t, resolved.Actions[0].Held)
	require.False(t, resolved.Actions[1].Held)
	require.Equal(t, []Warning{
		{Line: 3, Text: `created "17 Oct" is not YYYY-MM-DD; ignored`},
		{Line: 4, Text: `unknown front matter key "status"; ignored`},
		{Line: 9, Text: `a: after "soon" is neither YYYY-MM-DD nor YYYY-MM-DD HH:MMZ; it holds an open item until fixed`},
	}, resolved.Check(time.Now()))

	empty, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [ ] a: A\n  - done when x\n  - after:\n- [ ] Close\n"))
	require.NoError(t, err)
	require.Equal(t, `""`, empty.Actions[0].AfterInvalid)
	require.True(t, empty.Resolve(time.Now(), nil).Actions[0].Held, "an empty after: holds its item")
}

func TestCommittedReadsAPlanDeletedSinceItsLastCommit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "plans.md")
	runGit(t, root, "init", "--quiet")
	require.NoError(t, os.WriteFile(path, []byte("first\n"), 0o600))
	runGit(t, root, "add", "plans.md")
	runGit(t, root, "commit", "--quiet", "-m", "plan")
	require.NoError(t, os.WriteFile(path, []byte("second\n"), 0o600))
	runGit(t, root, "commit", "--quiet", "-am", "edit")
	require.NoError(t, os.Remove(path))

	content, err := Committed(context.Background(), root, "plans.md")
	require.NoError(t, err)
	require.Equal(t, "second\n", string(content), "deletion uncommitted")

	runGit(t, root, "commit", "--quiet", "-am", "delete")
	content, err = Committed(context.Background(), root, "plans.md")
	require.NoError(t, err)
	require.Equal(t, "second\n", string(content), "deletion committed")

	_, err = Committed(context.Background(), root, "never.md")
	require.ErrorContains(t, err, "never.md was never committed")
}

func TestNextIsNoneWhenEveryOpenItemIsBlocked(t *testing.T) {
	plan, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [x] Done\n- [ ] Held\n  blocked: waiting\n"))
	require.NoError(t, err)
	_, ok := plan.Next()
	require.False(t, ok)
}

func TestNextLeavesTheClosingActionUntilEveryOtherIsDone(t *testing.T) {
	tests := []struct {
		content string
		name    string
		want    string
	}{
		{content: "- [ ] a: A\n  after: 2099-01-01\n- [ ] Close\n", name: "after"},
		{content: "- [ ] Close\n", name: "closing alone", want: "Close"},
		{content: "- [ ] a: A\n  blocked: b\n- [ ] b: B\n  blocked: a\n- [ ] Close\n", name: "cycle"},
		{content: "- [x] a: A\n- [ ] Close\n", name: "every other done", want: "Close"},
		{content: "- [ ] a: A\n  blocked: the user's call\n- [ ] Close\n", name: "prose"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n" + test.content))
			require.NoError(t, err)
			next, _ := parsed.Resolve(time.Now(), nil).Next()
			require.Equal(t, test.want, next.Title)
		})
	}
}

func TestParseClosesAFenceOnlyOnItsOwnMarker(t *testing.T) {
	content := "---\nkind: plan\n---\n## Actions\n" +
		"- [ ] Tilde\n  ~~~\n  ```go\n  blocked: an example, not a reason\n  ~~~\n" +
		"- [ ] Four\n  ````\n  ```\n  ````\n" +
		"- [ ] Inline\n  ```run x``` then text\n" +
		"- [ ] Column\n  ```\n```\n" +
		"- [X] Windows\r\n  asserted: kept\r\n" +
		"## Inbox\n- one\n"
	plan, err := Parse([]byte(content))
	require.NoError(t, err)
	require.Equal(t, []Item{
		{Body: []string{"  ~~~", "  ```go", "  blocked: an example, not a reason", "  ~~~"}, Line: 5, Title: "Tilde"},
		{Body: []string{"  ````", "  ```", "  ````"}, Line: 10, Title: "Four"},
		{Body: []string{"  ```run x``` then text"}, Line: 14, Title: "Inline"},
		{Body: []string{"  ```", "```"}, Line: 16, Title: "Column"},
		{Body: []string{"  asserted: kept"}, Done: true, Line: 19, Title: "Windows"},
	}, plan.Actions)
	require.Len(t, plan.Inbox, 1)
}

func TestParseCountsAFencesIndentFromItsListItem(t *testing.T) {
	parsed, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n" +
		"- [ ] a: A\n      ```\n" +
		"- [ ] b: B\n\t\t```\n" +
		"- [ ] c: C\n  - an example:\n    ```\n  blocked: not a reason\n    ```\n" +
		"- [ ] d: D\n  - an example:\n        ```\n  blocked: a reason\n" +
		"- [ ] e: E\n  - an example:\n      ```\n  blocked: not a reason\n      ```\n" +
		"- [ ] f: F\n  - a nested item\n  back at the item\n      ```\n  blocked: a reason\n" +
		"- [ ] Close\n"))
	require.NoError(t, err)
	var names []string
	for _, action := range parsed.Actions {
		names = append(names, action.Name)
	}
	require.Equal(t, []string{"a", "b", "c", "d", "e", "f", ""}, names, "four columns past the item's content is no fence")
	require.Empty(t, parsed.Actions[2].Blocked, "a fence under a nested item is a fence")
	require.Equal(t, []string{"a reason"}, parsed.Actions[3].Blocked, "four columns past a nested item's content is no fence")
	require.Empty(t, parsed.Actions[4].Blocked, "two columns past a nested item's content is a fence")
	require.Equal(t, []string{"a reason"}, parsed.Actions[5].Blocked, "a line back at the item's indent ends the nested item")
}

func TestParseReadsABOMAndATabIndentedBody(t *testing.T) {
	parsed, err := Parse([]byte("\xef\xbb\xbf---\nkind: plan\n---\n## Actions\n- [ ] a: A\n\tblocked: b\n- [ ] b: B\n"))
	require.NoError(t, err)
	require.Len(t, parsed.Actions, 2)
	require.Equal(t, []Edge{{Name: "b"}}, parsed.Actions[0].Edges)
}

func TestParseReadsActionsInFileOrder(t *testing.T) {
	plan, err := Parse([]byte(sample))
	require.NoError(t, err)
	require.Equal(t, "2026-10-17", plan.Created)
	require.Equal(t, ".documentation/specs/design.md", plan.Spec)
	require.Equal(t, []Item{
		{Body: []string{"  asserted: the spec reads from its new path"}, Done: true, Line: 11, Title: "Move the spec"},
		{Body: []string{"  - as of 2026-10-17: parser written", "  next: commands"}, Line: 13, Title: "Parser and commands"},
		{Blocked: []string{"the parser first", "a real session to test on"}, Body: []string{"  blocked: the parser first", "  - blocked: a real session to test on"}, Held: true, Line: 19, Prose: []string{"the parser first", "a real session to test on"}, Title: "Start block section"},
		{Body: []string{"  ```", "## Not a heading", "  ```"}, Line: 22, Title: "Skill"},
	}, plan.Actions)
	require.Equal(t, Counts{Blocked: 1, Done: 1, Inbox: 2, Open: 2}, plan.Counts())
	next, ok := plan.Next()
	require.True(t, ok)
	require.Equal(t, "Parser and commands", next.Title)
}

func TestParseReadsBlockedAndAfterInAnyCaseAndAWrappedList(t *testing.T) {
	parsed, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n" +
		"- [ ] a: A\n  - Blocked: b,\n    c; once both land\n  - AFTER: 2099-01-01\n  - next: more\n" +
		"- [ ] b: B\n  blocked: c; waits on the review, the docs,\n  after: 2099-01-02\n- [ ] c: C\n- [ ] Close\n"))
	require.NoError(t, err)
	require.Equal(t, []Edge{{Name: "b"}, {Name: "c"}}, parsed.Actions[0].Edges)
	require.Equal(t, []string{"b, c; once both land"}, parsed.Actions[0].Blocked)
	require.Equal(t, time.Date(2099, 1, 1, 0, 0, 0, 0, time.Local), parsed.Actions[0].After)
	require.True(t, parsed.Resolve(time.Now(), nil).Actions[0].Held)
	require.Equal(t, time.Date(2099, 1, 2, 0, 0, 0, 0, time.Local), parsed.Actions[1].After, "a comma in the reason joins nothing")
}

func TestParseReadsEdgesOnlyFromALeadingListOfNames(t *testing.T) {
	plan, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [ ] a: A\n" +
		"  blocked: judge-blocks, mining-reports/report-figures; once the week ends\n" +
		"  - blocked: the parser first\n" +
		"  blocked: waits on the judging; and more\n"))
	require.NoError(t, err)
	require.Equal(t, []Edge{{Name: "judge-blocks"}, {Name: "report-figures", Plan: "mining-reports"}}, plan.Actions[0].Edges)
	require.Equal(t, []string{"the parser first", "waits on the judging; and more"}, plan.Actions[0].Prose)
}

func TestParseReadsTheKebabNameAnItemLeadsWith(t *testing.T) {
	plan, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [ ] judge-blocks: sort the blocks\n- [x] edges: `blocked:` names\n- [ ] Fold it: into the spec\n- [ ] no-space:here\n- [ ] Trailing-: dash\n"))
	require.NoError(t, err)
	var names []string
	for _, action := range plan.Actions {
		names = append(names, action.Name)
	}
	require.Equal(t, []string{"judge-blocks", "edges", "", "", ""}, names)
	require.Equal(t, "judge-blocks: sort the blocks", plan.Actions[0].Title, "the title keeps its name")
}

func TestParseRecordsWhatEachCitationNames(t *testing.T) {
	parsed, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n" +
		"- [ ] a: A, from note 20261020-ead7f7 in session a5491a94-1c2d-4e5f-8a9b-0c1d2e3f4a5b\n" +
		"  - ele show --at 10:00 24469c4a and ele show --note 20261020-ead7f7\n" +
		"  - asserted in 0cc19d4\n"))
	require.NoError(t, err)
	require.Equal(t, []Citation{
		{Line: 5, References: []Reference{{ID: "20261020-ead7f7", Kind: ReferenceNote}, {ID: "a5491a94-1c2d-4e5f-8a9b-0c1d2e3f4a5b", Kind: ReferenceSession}}},
		{Line: 6, References: []Reference{{ID: "20261020-ead7f7", Kind: ReferenceNote}, {ID: "24469c4a", Kind: ReferenceSession}, {ID: "20261020-ead7f7", Kind: ReferenceNote}}},
	}, parsed.Citations)
}

func TestParseRejectsInvalidPlans(t *testing.T) {
	tests := []struct {
		content string
		message string
		name    string
	}{
		{content: "---\nkind: plan\n---\n## Actions\n- [ ] a: A\n  after: 2026-10-22\n  after: 2026-10-23\n", message: "a second after: on line 7", name: "after twice"},
		{content: "", message: "first line is not ---", name: "empty file"},
		{content: "---\nkind: plan\n## Actions\n", message: "is not key: value", name: "front matter line not key: value"},
		{content: "---\nkind: plan\n", message: "front matter never closes", name: "front matter never closes"},
		{content: "---\nkind: plan\n---\n## Actions\n- [ ] edges: one\n- [x] other: two\n- [ ] edges: three\n", message: `item name "edges" on lines 5 and 7`, name: "name twice"},
		{content: "---\nkind: plan\n---\n## Tasks\n- [ ] x\n", message: "no ## Actions section", name: "no actions section"},
		{content: "# Plan\n", message: "first line is not ---", name: "no front matter"},
		{content: "---\nkind: spec\n---\n## Actions\n", message: `kind "spec" is not plan`, name: "not a plan"},
		{content: "---\nkind: plan\nkind: plan\n---\n## Actions\n", message: `front matter key "kind" appears twice`, name: "repeated key"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.content))
			require.ErrorIs(t, err, ErrInvalid)
			require.ErrorContains(t, err, test.message)
		})
	}
}

func TestParseSkipsAFencedHeading(t *testing.T) {
	plan, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n\n```\n## Inbox\n- [ ] in a fence\n```\n- [ ] Real\n"))
	require.NoError(t, err)
	require.Len(t, plan.Actions, 1)
	require.Equal(t, "Real", plan.Actions[0].Title)
}

func TestPreviousFollowsARenameARevertAndAReword(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "--quiet")
	write := func(path string, content string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(root, path), []byte("---\nkind: plan\n---\n## Actions\n"+content), 0o600))
	}

	write("plans.md", "- [ ] a: A\n- [ ] b: B\n- [ ] Close\n")
	runGit(t, root, "add", "plans.md")
	runGit(t, root, "commit", "--quiet", "-m", "plan")
	write("plans.md", "- [x] a: A, reworded\n- [ ] b: B\n- [ ] Close\n")
	runGit(t, root, "commit", "--quiet", "-am", "tick a")
	runGit(t, root, "mv", "plans.md", "moved.md")
	runGit(t, root, "commit", "--quiet", "-m", "move")
	tick, ok, err := Previous(context.Background(), root, "moved.md")
	require.NoError(t, err)
	require.True(t, ok, "the tick before the rename")
	require.Equal(t, "a: A, reworded", tick.Title, "a reword in the ticking commit is still its flip")

	write("moved.md", "- [x] a: A, reworded\n- [x] b: B\n- [ ] Close\n")
	runGit(t, root, "commit", "--quiet", "-am", "tick b")
	runGit(t, root, "revert", "--no-edit", "HEAD")
	tick, ok, err = Previous(context.Background(), root, "moved.md")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "a: A, reworded", tick.Title, "a reverted tick is not previous")
}

func TestPreviousIsNoneBeforeTheFirstCommitAndFailsOnABrokenRepository(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "--quiet")
	_, ok, err := Previous(context.Background(), root, "plans.md")
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, os.WriteFile(filepath.Join(root, "plans.md"), []byte("---\nkind: plan\n---\n## Actions\n- [ ] Close\n"), 0o600))
	runGit(t, root, "add", "plans.md")
	runGit(t, root, "commit", "--quiet", "-m", "plan")
	require.NoError(t, os.RemoveAll(filepath.Join(root, ".git", "objects")))
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git", "objects"), 0o700))
	_, _, err = Previous(context.Background(), root, "plans.md")
	require.Error(t, err, "a repository missing its objects is not outside git")

	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, nil, 0o600))
	_, _, err = Previous(context.Background(), file, "plans.md")
	require.ErrorContains(t, err, "git rev-parse", "a rev-parse git refuses is not outside git")

	t.Setenv("PATH", "")
	_, _, err = Previous(context.Background(), root, "plans.md")
	require.Error(t, err, "a missing git is not outside git")
}

func TestPreviousIsNoneOutsideGit(t *testing.T) {
	_, ok, err := Previous(context.Background(), t.TempDir(), "plans.md")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestPreviousPrefersAnUncommittedTick(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "plans.md")
	runGit(t, root, "init", "--quiet")
	write := func(content string) {
		t.Helper()
		require.NoError(t, os.WriteFile(path, []byte("---\nkind: plan\n---\n## Actions\n"+content), 0o600))
	}

	write("- [ ] First\n- [ ] Second\n- [ ] Third\n")
	runGit(t, root, "add", "plans.md")
	runGit(t, root, "commit", "--quiet", "-m", "plan")
	_, ok, err := Previous(context.Background(), root, "plans.md")
	require.NoError(t, err)
	require.False(t, ok, "nothing ticked yet")

	write("- [ ] First\n- [ ] Second\n- [ ] Third\n- [x] Written ticked\n")
	_, ok, err = Previous(context.Background(), root, "plans.md")
	require.NoError(t, err)
	require.False(t, ok, "a box written ticked was never flipped")

	write("- [x] First\n  asserted: it works\n- [ ] Second\n- [ ] Third\n")
	runGit(t, root, "commit", "--quiet", "-am", "tick first")
	write("- [x] First\n  asserted: it works\n- [ ] Second\n- [ ] Third\n  as of 2026-10-17: started\n")
	runGit(t, root, "commit", "--quiet", "-am", "progress on third")
	tick, ok, err := Previous(context.Background(), root, "plans.md")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "First", tick.Title)
	require.Regexp(t, `^[0-9a-f]{7,}$`, tick.Commit, "a commit that ticks nothing is skipped")
	require.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, tick.Date)

	write("- [x] First\n  asserted: it works\n- [x] Second\n- [ ] Third\n")
	tick, ok, err = Previous(context.Background(), root, "plans.md")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, Tick{Title: "Second"}, tick)

	write("- [x] First\n  asserted: it works\n- [x] Second\n- [x] Third\n")
	tick, _, err = Previous(context.Background(), root, "plans.md")
	require.NoError(t, err)
	require.Equal(t, Tick{Title: "Third"}, tick, "of two ticks in one diff, the later in the file")
}

func TestPreviousTakesTheTickStillTickedOfACommitThatTickedTwo(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "plans.md")
	runGit(t, root, "init", "--quiet")
	write := func(content string) {
		t.Helper()
		require.NoError(t, os.WriteFile(path, []byte("---\nkind: plan\n---\n## Actions\n"+content), 0o600))
	}

	write("- [ ] a: A\n- [ ] b: B\n- [ ] Close\n")
	runGit(t, root, "add", "plans.md")
	runGit(t, root, "commit", "--quiet", "-m", "plan")
	write("- [x] a: A\n- [x] b: B\n- [ ] Close\n")
	runGit(t, root, "commit", "--quiet", "-am", "tick a and b")
	write("- [x] a: A\n- [ ] b: B\n- [ ] Close\n")
	runGit(t, root, "commit", "--quiet", "-am", "untick b")
	tick, ok, err := Previous(context.Background(), root, "plans.md")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "a: A", tick.Title)
}

func TestResolveFreesABlockedItemOnceItsBlockerIsTicked(t *testing.T) {
	content := "---\nkind: plan\n---\n## Actions\n- [ ] first: First\n- [ ] second: Second\n  blocked: first, other/item; reason\n"
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	other := BlockerOpen
	lookup := func(edge Edge) Blocker {
		if edge != (Edge{Name: "item", Plan: "other"}) {
			return BlockerMissing
		}
		return other
	}

	held, err := Parse([]byte(content))
	require.NoError(t, err)
	resolved := held.Resolve(now, lookup)
	require.True(t, resolved.Actions[1].Held)
	require.Equal(t, Counts{Blocked: 1, Open: 1}, resolved.Counts())

	ticked, err := Parse([]byte(strings.Replace(content, "- [ ] first", "- [x] first", 1)))
	require.NoError(t, err)
	require.True(t, ticked.Resolve(now, lookup).Actions[1].Held, "the other plan's item still holds it")
	other = BlockerDone
	resolved = ticked.Resolve(now, lookup)
	require.False(t, resolved.Actions[1].Held)
	require.Equal(t, []string{"blocker first is done; drop it from blocked:", "blocker other/item is done; drop it from blocked:"}, resolved.Actions[1].Flags)
	next, ok := resolved.Next()
	require.True(t, ok)
	require.Equal(t, "second", next.Name)
}

func TestResolveHoldsAnItemOnABlockerInAPlanThisProjectLacksAndSaysWhereItsWaitGoes(t *testing.T) {
	plan, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [ ] a: A\n  blocked: elsewhere/item\n"))
	require.NoError(t, err)
	resolved := plan.Resolve(time.Now(), func(Edge) Blocker { return BlockerNoPlan })
	require.True(t, resolved.Actions[0].Held)
	require.Equal(t, []string{"blocker elsewhere/item names no started or closed plan of this project; start that plan, or write a wait on another repository's plan as prose, with no comma before its semicolon"}, resolved.Actions[0].Flags)
}

func TestResolveHoldsAnItemOnAClosedPlansUntickedItemAndFlagsIt(t *testing.T) {
	plan, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [ ] a: A\n  blocked: closed/item\n"))
	require.NoError(t, err)
	resolved := plan.Resolve(time.Now(), func(Edge) Blocker { return BlockerAbandoned })
	require.True(t, resolved.Actions[0].Held)
	require.Equal(t, []string{"blocker closed/item is in a closed plan and was never done"}, resolved.Actions[0].Flags)
}

func TestResolveHoldsAnItemOnAMissingBlockerAndFlagsIt(t *testing.T) {
	plan, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [ ] a: A\n  blocked: typo, gone/item\n"))
	require.NoError(t, err)
	resolved := plan.Resolve(time.Now(), func(Edge) Blocker { return BlockerMissing })
	require.True(t, resolved.Actions[0].Held)
	require.Equal(t, []string{"blocker typo does not exist", "blocker gone/item does not exist"}, resolved.Actions[0].Flags)
}

func TestResolveHoldsAnItemUntilItsAfterDate(t *testing.T) {
	plan, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [ ] a: A\n  after: 2026-10-22 19:28Z\n- [ ] b: B\n  - after: 2026-10-23\n"))
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 10, 22, 19, 28, 0, 0, time.UTC), plan.Actions[0].After)
	require.Equal(t, time.Date(2026, 10, 23, 0, 0, 0, 0, time.Local), plan.Actions[1].After)

	before := plan.Resolve(time.Date(2026, 10, 22, 19, 27, 0, 0, time.UTC), nil)
	require.True(t, before.Actions[0].Held)
	at := plan.Resolve(time.Date(2026, 10, 22, 19, 28, 0, 0, time.UTC), nil)
	require.False(t, at.Actions[0].Held)
	require.True(t, at.Actions[1].Held)
	require.False(t, plan.Resolve(time.Date(2026, 10, 23, 0, 0, 0, 0, time.Local), nil).Actions[1].Held)
}

func TestWriteStoredLeavesAReaderOfTheEarlierFileItWhole(t *testing.T) {
	directory := t.TempDir()
	earlier, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [ ] a: A\n"))
	require.NoError(t, err)
	require.NoError(t, WriteStored(directory, "id", "earlier.md", earlier))
	reader, err := os.Open(filepath.Join(directory, "id.json"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close() })

	later, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [x] a: A\n- [ ] b: B\n"))
	require.NoError(t, err)
	require.NoError(t, WriteStored(directory, "id", "later.md", later))

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	var held Stored
	require.NoError(t, json.Unmarshal(content, &held))
	require.Equal(t, Stored{Items: []StoredItem{{Name: "a"}}, Path: "earlier.md"}, held)
	stored, ok, err := ReadStored(directory, "id")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, Stored{Items: []StoredItem{{Done: true, Name: "a"}, {Name: "b"}}, Path: "later.md"}, stored)
}

func TestWriteStoredRemovesItsTemporaryFileWhenTheRenameFails(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(directory, "id.json", "occupied"), 0o755))
	plan, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [ ] a: A\n"))
	require.NoError(t, err)

	require.Error(t, WriteStored(directory, "id", "plan.md", plan))

	left, err := filepath.Glob(filepath.Join(directory, "*.tmp"))
	require.NoError(t, err)
	require.Empty(t, left)
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}
