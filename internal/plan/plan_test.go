package plan

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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

func TestNextIsNoneWhenEveryOpenItemIsBlocked(t *testing.T) {
	plan, err := Parse([]byte("---\nkind: plan\n---\n## Actions\n- [x] Done\n- [ ] Held\n  blocked: waiting\n"))
	require.NoError(t, err)
	_, ok := plan.Next()
	require.False(t, ok)
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
	require.Equal(t, 1, plan.Inbox)
}

func TestParseReadsActionsInFileOrder(t *testing.T) {
	plan, err := Parse([]byte(sample))
	require.NoError(t, err)
	require.Equal(t, "2026-10-17", plan.Created)
	require.Equal(t, ".documentation/specs/design.md", plan.Spec)
	require.Equal(t, []Item{
		{Body: []string{"  asserted: the spec reads from its new path"}, Done: true, Line: 11, Title: "Move the spec"},
		{Body: []string{"  - as of 2026-10-17: parser written", "  next: commands"}, Line: 13, Title: "Parser and commands"},
		{Blocked: []string{"the parser first", "a real session to test on"}, Body: []string{"  blocked: the parser first", "  - blocked: a real session to test on"}, Line: 19, Title: "Start block section"},
		{Body: []string{"  ```", "## Not a heading", "  ```"}, Line: 22, Title: "Skill"},
	}, plan.Actions)
	require.Equal(t, Counts{Blocked: 1, Done: 1, Inbox: 2, Open: 2}, plan.Counts())
	next, ok := plan.Next()
	require.True(t, ok)
	require.Equal(t, "Parser and commands", next.Title)
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

func TestParseRejectsInvalidPlans(t *testing.T) {
	tests := []struct {
		content string
		message string
		name    string
	}{
		{content: "---\nkind: plan\n## Actions\n", message: "is not key: value", name: "front matter never closes"},
		{content: "---\nkind: plan\ncreated: 17 Oct\n---\n## Actions\n", message: `created "17 Oct" is not YYYY-MM-DD`, name: "loose created"},
		{content: "---\nkind: plan\n---\n## Actions\n- [ ] edges: one\n- [x] other: two\n- [ ] edges: three\n", message: `item name "edges" on lines 5 and 7`, name: "name twice"},
		{content: "---\nkind: plan\n---\n## Tasks\n- [ ] x\n", message: "no ## Actions section", name: "no actions section"},
		{content: "# Plan\n", message: "first line is not ---", name: "no front matter"},
		{content: "---\nkind: spec\n---\n## Actions\n", message: `kind "spec" is not plan`, name: "not a plan"},
		{content: "---\nkind: plan\nkind: plan\n---\n## Actions\n", message: `front matter key "kind" appears twice`, name: "repeated key"},
		{content: "---\nkind: plan\nstatus: open\n---\n## Actions\n", message: `unknown front matter key "status"`, name: "unknown key"},
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

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}
