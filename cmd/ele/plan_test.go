package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

var started = regexp.MustCompile(`^Started plan (\S+) as (\d{8}-[0-9a-f]{6});`)

const planFile = `---
kind: plan
created: 2026-10-17
---

## Actions

- [x] Parser
  asserted: parses the sample
- [ ] Start block
  blocked: the parser first
- [ ] Commands
  next: done and list

## Inbox

- 2026-10-17: rename a flag
`

func TestPlanSummaryRedactsTheNextTitleAndOmitsANextThatIsNone(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "secret.md"), "---\nkind: plan\n---\n\n## Actions\n\n- [ ] Rotate token=abc123secret\n")
	writeFile(t, filepath.Join(root, "blocked.md"), "---\nkind: plan\n---\n\n## Actions\n\n- [x] Parser\n- [ ] Commands\n  blocked: the parser first\n")

	require.Equal(t, "secret.md: 1 open, 0 blocked, 0 done, 0 in the inbox. Next: Rotate token=[redacted:assignment]", planSummary(root, "secret.md", nil))
	require.Equal(t, "blocked.md: 0 open, 1 blocked, 1 done, 0 in the inbox.", planSummary(root, "blocked.md", nil))
}

func TestRunPlanDonePrefersThePathFromTheWorkingDirectory(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	writeFile(t, filepath.Join(gateway, "plans.md"), planFile)
	writeFile(t, filepath.Join(gateway, "docs", "plans.md"), planFile)
	t.Chdir(gateway)
	output, code := runCommand(t, "plan", "start", "docs/plans.md")
	require.Equal(t, 0, code, output)
	_, err := notes.Write(filepath.Join(home, "data", "notes"), notes.Note{Body: "plans.md", Date: "2099-01-01", ID: "20990101-aaaaaa", Kind: notes.KindPlan, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject})
	require.NoError(t, err)

	t.Chdir(filepath.Join(gateway, "docs"))
	output, code = runCommand(t, "plan", "done", "plans.md")
	require.Equal(t, 0, code, output)
	require.True(t, strings.HasPrefix(output, "Done with plan docs/plans.md "), "the newer root plans.md matches the raw body, but the resolved path wins: "+output)
}

func TestRunPlanShowNamesEachActivePlanOrAMissingFile(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	writeFile(t, filepath.Join(gateway, "first.md"), planFile)
	writeFile(t, filepath.Join(gateway, "second.md"), planFile)
	t.Chdir(gateway)
	for _, path := range []string{"first.md", "second.md"} {
		output, code := runCommand(t, "plan", "start", path)
		require.Equal(t, 0, code, output)
	}

	output, code := runCommand(t, "plan", "show")
	require.Equal(t, 0, code, output)
	summary := `\.md: 1 open, 1 blocked, 1 done, 1 in the inbox\. Next: Commands\n`
	require.Regexp(t, `^Active plans for corp · gateway, newest first; ele plan show <path> prints one:\n- \S+: (first|second)`+summary+`- \S+: (first|second)`+summary+`$`, output, "the next item skips the blocked one")

	require.NoError(t, os.Remove(filepath.Join(gateway, "first.md")))
	output, code = runCommand(t, "plan", "show", "first.md")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: first.md is not in this checkout\n", output)

	output, code = runCommand(t, "plan", "done", "first.md", "nowhere.md")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: nowhere.md is not an active plan of corp · gateway; ele plan list lists them, --project <name> for another project's\n", output)
	output, code = runCommand(t, "plan", "list")
	require.Equal(t, 0, code, output)
	require.Equal(t, 2, strings.Count(output, "- active"), "a mistyped argument closes nothing")

	output, code = runCommand(t, "plan", "done", "second.md")
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "plan", "show")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: plan first.md is not in this checkout; switch to its branch, or ele plan done first.md closes it\n", output)
}

func TestRunPlanStartRefusesANameAnotherActivePlanHoldsAndShowWarnsOfOne(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	named := "---\nkind: plan\n---\n## Actions\n- [ ] parser: Parser\n- [ ] commands: Commands\n"
	writeFile(t, filepath.Join(gateway, "first.md"), named)
	writeFile(t, filepath.Join(gateway, "second.md"), strings.Replace(named, "parser: Parser", "lexer: Lexer", 1))
	t.Chdir(gateway)
	output, code := runCommand(t, "plan", "start", "first.md")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "plan", "start", "second.md")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: second.md: item name \"commands\" is also in first.md; item names are unique across active plans\n", output)

	writeFile(t, filepath.Join(gateway, "second.md"), "---\nkind: plan\n---\n## Actions\n- [ ] lexer: Lexer\n")
	output, code = runCommand(t, "plan", "start", "second.md")
	require.Equal(t, 0, code, output)
	writeFile(t, filepath.Join(gateway, "second.md"), named)
	output, code = runCommand(t, "plan", "show", "first.md")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Next, line 5:\n- [ ] parser: Parser\n")
	require.True(t, strings.HasSuffix(output, "- [ ] parser: Parser\n"+
		"Warning: item name \"parser\" is also in second.md; item names are unique across active plans.\n"+
		"Warning: item name \"commands\" is also in second.md; item names are unique across active plans.\n"), output)

	output, code = runCommand(t, "plan", "show")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "first.md: 2 open, 0 blocked, 0 done, 0 in the inbox. Next: parser: Parser Warning: item name \"parser\" is also in second.md, item name \"commands\" is also in second.md; item names are unique across active plans.\n", "several active plans each warn in their summary")
}

func TestRunPlanStartRefusesWhatIsNotAPlan(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	writeFile(t, filepath.Join(gateway, "spec.md"), "---\nkind: spec\n---\n## Actions\n- [ ] x\n")
	writeFile(t, filepath.Join(gateway, "empty.md"), "---\nkind: plan\n---\n## Actions\n\n## Inbox\n")
	writeFile(t, filepath.Join(home, "w", "corp", "outside.md"), planFile)
	t.Chdir(gateway)

	tests := []struct {
		message string
		name    string
		path    string
	}{
		{message: "ele: missing.md does not exist; name a plan file inside the project\n", name: "missing file", path: "missing.md"},
		{message: "ele: empty.md has no checkbox under ## Actions\n", name: "no checkbox", path: "empty.md"},
		{message: "ele: spec.md: invalid plan: kind \"spec\" is not plan\n", name: "not a plan", path: "spec.md"},
		{message: "ele: ../outside.md is outside the project at ", name: "outside the project", path: "../outside.md"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, code := runCommand(t, "plan", "start", test.path)
			require.Equal(t, 1, code)
			require.True(t, strings.HasPrefix(output, test.message), output)
		})
	}
	output, code := runCommand(t, "plan", "list", "--all")
	require.Equal(t, 0, code, output)
	require.Equal(t, "ele has no plans for corp · gateway.\n", output)
}

func TestRunPlanStartsShowsListsAndCloses(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	writeFile(t, filepath.Join(gateway, ".documentation", "plans", "build.md"), planFile)
	t.Chdir(filepath.Join(gateway, ".documentation"))

	output, code := runCommand(t, "plan", "start", "plans/build.md")
	require.Equal(t, 0, code, output)
	match := started.FindStringSubmatch(output)
	require.NotNil(t, match, output)
	require.Equal(t, ".documentation/plans/build.md", match[1], "the path is from the checkout root, not the working directory")
	id := match[2]

	output, code = runCommand(t, "plan", "start", "plans/build.md")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: .documentation/plans/build.md is already started; ele plan show prints it\n", output)

	output, code = runCommand(t, "plan", "show")
	require.Equal(t, 0, code, output)
	require.Equal(t, `Plan .documentation/plans/build.md: 1 open, 1 blocked, 1 done, 1 in the inbox.
Blocked:
- Start block
  blocked: the parser first
Next, line 12:
- [ ] Commands
  next: done and list
`, output)

	output, code = runCommand(t, "plan", "list")
	require.Equal(t, 0, code, output)
	require.Regexp(t, `^Plans for corp · gateway, newest first:\n- active  note `+id+` of .*: \.documentation/plans/build\.md\n`, output)

	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, id, "a plan note is left out of the notes list")

	output, code = runCommand(t, "plan", "done", "plans/build.md")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Done with plan .documentation/plans/build.md ("+id+").\n", output)

	output, code = runCommand(t, "plan", "show")
	require.Equal(t, 0, code, output)
	require.Equal(t, "ele has no active plan for corp · gateway; ele plan start <path> starts one.\n", output)
	output, code = runCommand(t, "plan", "list", "--all")
	require.Equal(t, 0, code, output)
	require.Regexp(t, `\n- done \S+  note `+id+` of `, output)
}
