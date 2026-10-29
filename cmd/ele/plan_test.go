package main

import (
	"context"
	"os"
	"os/exec"
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

	require.Equal(t, "secret.md: 1 open, 0 blocked, 0 done, 0 in the inbox. Next: Rotate token=[redacted:assignment]", planSummary(context.Background(), root, "secret.md", planSet{}))
	require.Equal(t, "blocked.md: 0 open, 1 blocked, 1 done, 0 in the inbox. 1 warning; ele plan check lists it.", planSummary(context.Background(), root, "blocked.md", planSet{}))
}

func TestRunPlanCheckListsEachActivePlansWarningsWithoutFailing(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	writeFile(t, filepath.Join(gateway, "clean.md"), "---\nkind: plan\n---\n## Actions\n- [ ] alpha: Alpha\n  done when it is\n- [ ] Close\n")
	writeFile(t, filepath.Join(gateway, "stale.md"), "---\nkind: plan\n---\n## Context\n- as of 2026-10-01: old\n## Actions\n- [ ] beta: Beta\n- [ ] Close\n## Inbox\n- undated\n")
	t.Chdir(gateway)
	output, code := runCommand(t, "plan", "check")
	require.Equal(t, 0, code, output)
	require.Equal(t, "ele has no active plan for corp · gateway; ele plan check <path> checks one.\n", output)
	for _, path := range []string{"clean.md", "stale.md"} {
		output, code = runCommand(t, "plan", "start", path)
		require.Equal(t, 0, code, output)
	}

	stale := "Plan stale.md: 3 warnings.\n- line 5: as of 2026-10-01 is 20 days old; re-measure it\n- line 7: beta: no done when line\n- line 10: inbox item has no date; lead it with the date and where it came from\n"
	output, code = runCommand(t, "plan", "check")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Plan clean.md: no warnings.\n")
	require.Contains(t, output, stale)

	writeFile(t, filepath.Join(gateway, "clash.md"), "---\nkind: plan\n---\n## Actions\n- [ ] beta: Again\n  done when it is\n- [ ] Close\n")
	output, code = runCommand(t, "plan", "check", "clash.md")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Plan clash.md: 1 warning.\n- item name \"beta\" is also in stale.md; item names are unique across active and closed plans\n", output, "a plan need not be started to be checked")

	require.NoError(t, os.Remove(filepath.Join(gateway, "clean.md")))
	output, code = runCommand(t, "plan", "check")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "plan clean.md is not in this checkout; switch to its branch, or ele plan done clean.md closes it.\n")
	require.Contains(t, output, stale)

	output, code = runCommand(t, "plan", "check", "clean.md")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: clean.md is not in this checkout\n", output)
	writeFile(t, filepath.Join(gateway, "stale.md"), "---\nkind: spec\n---\n")
	output, code = runCommand(t, "plan", "check")
	require.Equal(t, 1, code)
	require.Contains(t, output, "ele: stale.md: invalid plan: kind \"spec\" is not plan\n")
}

func TestRunPlanCheckWarnsOnlyOnEleDataEleResolves(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	session := "a5491a94-1c2d-4e5f-8a9b-0c1d2e3f4a5b"
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", session+".jsonl"),
		`{"type":"user","cwd":"`+gateway+`","sessionId":"`+session+`","timestamp":"2026-10-15T11:00:00Z","uuid":"u1","message":{"content":"hello"}}`+"\n")
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	unsynced := "b6c7d8e9-0a1b-4c2d-9e3f-405162738495"
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", unsynced+".jsonl"), "")
	_, err := notes.Write(filepath.Join(home, "data", "notes"), notes.Note{Body: "kept", Date: "2026-10-20", ID: "20261020-ead7f7", Kind: notes.KindDecision, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject})
	require.NoError(t, err)
	writeFile(t, filepath.Join(gateway, "cites.md"), "---\nkind: plan\n---\n## Actions\n- [x] a: A\n"+
		"  - the fixture's own id, 0c1d2e3f-4a5b-4c6d-8e7f-a0b1c2d3e4f5\n"+
		"  - seen in session "+session+"\n"+
		"  - ele show 0cc19d4a, a commit\n"+
		"  - ele show a5491a94 --at 11:00\n"+
		"  - from note 20261020-ead7f7\n"+
		"  - from note 20261020-000000\n"+
		"  - in session "+unsynced+", not yet synced\n"+
		"- [ ] b: B\n  done when it is\n"+
		"- [ ] Close\n")
	t.Chdir(gateway)

	warning := "cites ele data, which resolves only in its author's data directory; delete provenance, describe what was observed, and move a dependency into the plan, superseding its note\n"
	output, code = runCommand(t, "plan", "check", "cites.md")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Plan cites.md: 4 warnings.\n- line 7: "+warning+"- line 9: "+warning+"- line 10: "+warning+"- line 12: "+warning, output, "only the indexed session, its prefix, the saved note and the unsynced session's transcript resolve")

	writeFile(t, filepath.Join(home, "data", "ele.db"), "not a database")
	output, code = runCommand(t, "plan", "check", "cites.md")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Plan cites.md: 7 warnings.\n- line 6: "+warning+"- line 7: "+warning+"- line 8: "+warning+"- line 9: "+warning+"- line 10: "+warning+"- line 11: "+warning+"- line 12: "+warning, output, "an unreadable index never silences the check")
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
	summary := `\.md: 1 open, 1 blocked, 1 done, 1 in the inbox\. 1 warning; ele plan check lists it\. Next: Commands\n`
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
	require.Equal(t, "ele: second.md: item name \"commands\" is also in first.md; item names are unique across active and closed plans\n", output)

	writeFile(t, filepath.Join(gateway, "second.md"), "---\nkind: plan\n---\n## Actions\n- [ ] lexer: Lexer\n")
	output, code = runCommand(t, "plan", "start", "second.md")
	require.Equal(t, 0, code, output)
	writeFile(t, filepath.Join(gateway, "second.md"), named)
	output, code = runCommand(t, "plan", "show", "first.md")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Next, line 5:\n- [ ] parser: Parser\n")
	require.True(t, strings.HasSuffix(output, "- [ ] parser: Parser\n"+
		"Warning: item name \"parser\" is also in second.md; item names are unique across active and closed plans.\n"+
		"Warning: item name \"commands\" is also in second.md; item names are unique across active and closed plans.\n"), output)

	output, code = runCommand(t, "plan", "show")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "first.md: 2 open, 0 blocked, 0 done, 0 in the inbox. 3 warnings; ele plan check lists them. Next: parser: Parser Warning: item name \"parser\" is also in second.md, item name \"commands\" is also in second.md; item names are unique across active and closed plans.\n", "several active plans each warn in their summary")
}

func TestRunPlanDoneStoresItemNamesThatResolveEdgesAndStayReserved(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	writeFile(t, filepath.Join(gateway, "build.md"), "---\nkind: plan\n---\n## Actions\n- [x] parser: Parser\n- [ ] commands: Commands\n- [ ] Close it\n")
	writeFile(t, filepath.Join(gateway, "other.md"), "---\nkind: plan\n---\n## Actions\n- [ ] wire: Wire\n  blocked: build/commands; reason\n- [ ] typo: Typo\n  blocked: build/nope\n")
	t.Chdir(gateway)
	for _, path := range []string{"build.md", "other.md"} {
		output, code := runCommand(t, "plan", "start", path)
		require.Equal(t, 0, code, output)
	}

	output, code := runCommand(t, "plan", "show", "other.md")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Plan other.md: 0 open, 2 blocked, 0 done, 0 in the inbox.\n2 warnings; ele plan check other.md lists them.\nBlocked:\n- wire: Wire\n  blocked: build/commands; reason\n- typo: Typo\n  blocked: build/nope\n  flag: blocker build/nope does not exist\nNext: none; every open item is blocked.\n", output)

	output, code = runCommand(t, "plan", "done", "build.md")
	require.Equal(t, 0, code, output)
	require.NoError(t, os.Remove(filepath.Join(gateway, "build.md")))
	output, code = runCommand(t, "plan", "show", "other.md")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "- wire: Wire\n  blocked: build/commands; reason\n  flag: blocker build/commands is in a closed plan and was never done\n- typo", "the closed plan's stored state still holds it")

	writeFile(t, filepath.Join(gateway, "third.md"), "---\nkind: plan\n---\n## Actions\n- [ ] parser: Parser again\n")
	output, code = runCommand(t, "plan", "start", "third.md")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: third.md: item name \"parser\" was used by closed plan build.md; item names are unique across active and closed plans\n", output)
}

func TestRunPlanDoneStoresADeletedPlanFromItsLastCommitAndWarnsOfItsUntickedNames(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	require.NoError(t, os.RemoveAll(filepath.Join(gateway, ".git")))
	git := func(args ...string) {
		output, err := exec.Command("git", append([]string{"-C", gateway, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
		require.NoError(t, err, string(output))
	}
	git("init", "--quiet")
	writeFile(t, filepath.Join(gateway, "build.md"), "---\nkind: plan\n---\n## Actions\n- [x] commands: Commands\n- [ ] parser: Parser\n")
	writeFile(t, filepath.Join(gateway, "other.md"), "---\nkind: plan\n---\n## Actions\n- [ ] wire: Wire\n  blocked: build/commands\n- [ ] read: Read\n  blocked: build/parser\n")
	git("add", "build.md")
	git("commit", "--quiet", "-m", "plan")
	t.Chdir(gateway)
	for _, path := range []string{"build.md", "other.md"} {
		output, code := runCommand(t, "plan", "start", path)
		require.Equal(t, 0, code, output)
	}
	require.NoError(t, os.Remove(filepath.Join(gateway, "build.md")))

	output, code := runCommand(t, "plan", "done", "build.md")
	require.Equal(t, 0, code, output)
	require.True(t, strings.HasPrefix(output, "build.md is gone, so its names were stored from its last commit, where parser is unticked; run ele plan done before deleting a plan.\nDone with plan build.md "), output)

	output, code = runCommand(t, "plan", "show", "other.md")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Plan other.md: 1 open, 1 blocked, 0 done, 0 in the inbox.\n3 warnings; ele plan check other.md lists them.\n"+
		"Blocked:\n- read: Read\n  blocked: build/parser\n  flag: blocker build/parser is in a closed plan and was never done\n"+
		"Flagged:\n- wire: Wire\n  flag: blocker build/commands is done; drop it from blocked:\n"+
		"Next, line 5:\n- [ ] wire: Wire\n  blocked: build/commands\n", output)
}

func TestRunPlanShowFreesAnItemOnceAnotherPlansBlockerIsTicked(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	writeFile(t, filepath.Join(gateway, "build.md"), "---\nkind: plan\n---\n## Actions\n- [ ] commands: Commands\n")
	writeFile(t, filepath.Join(gateway, "other.md"), "---\nkind: plan\n---\n## Actions\n- [ ] wire: Wire\n  blocked: build/commands\n")
	t.Chdir(gateway)
	for _, path := range []string{"build.md", "other.md"} {
		output, code := runCommand(t, "plan", "start", path)
		require.Equal(t, 0, code, output)
	}
	writeFile(t, filepath.Join(gateway, "build.md"), "---\nkind: plan\n---\n## Actions\n- [x] commands: Commands\n")

	output, code := runCommand(t, "plan", "show", "other.md")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Plan other.md: 1 open, 0 blocked, 0 done, 0 in the inbox.\n1 warning; ele plan check other.md lists it.\nFlagged:\n- wire: Wire\n  flag: blocker build/commands is done; drop it from blocked:\nNext, line 5:\n- [ ] wire: Wire\n  blocked: build/commands\n", output)
}

func TestRunPlanStartNamesThePlansItCouldNotCheckAndDoneStoresAClosedPlansRecord(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	records := filepath.Join(home, "data", "plans")
	named := func(name string) string {
		return "---\nkind: plan\n---\n## Actions\n- [ ] " + name + ": " + name + "\n"
	}
	t.Chdir(gateway)
	ids := map[string]string{}
	for _, name := range []string{"broken", "directory", "gone", "missing", "unreadable"} {
		writeFile(t, filepath.Join(gateway, name+".md"), named(name))
		output, code := runCommand(t, "plan", "start", name+".md")
		require.Equal(t, 0, code, output)
		ids[name] = started.FindStringSubmatch(output)[2]
	}
	for _, name := range []string{"gone", "unreadable"} {
		output, code := runCommand(t, "plan", "done", name+".md")
		require.Equal(t, 0, code, output)
	}
	require.NoError(t, os.Remove(filepath.Join(records, ids["gone"]+".json")))
	writeFile(t, filepath.Join(records, ids["unreadable"]+".json"), "not json")
	require.NoError(t, os.Remove(filepath.Join(gateway, "missing.md")))
	writeFile(t, filepath.Join(gateway, "broken.md"), "---\nkind: spec\n---\n")
	require.NoError(t, os.Remove(filepath.Join(gateway, "directory.md")))
	require.NoError(t, os.Mkdir(filepath.Join(gateway, "directory.md"), 0o755))

	writeFile(t, filepath.Join(gateway, "next.md"), named("gone"))
	output, code := runCommand(t, "plan", "start", "next.md")
	require.Equal(t, 0, code, output)
	require.Regexp(t, `^Started plan next\.md as \d{8}-[0-9a-f]{6}; ele plan show prints its next item\.\n`+
		regexp.QuoteMeta("Not checked for a name clash: active plan broken.md, which does not parse; active plan directory.md, which does not read; active plan missing.md, missing from the checkout; "+
			"closed plan gone.md, with no stored record (ele plan done "+ids["gone"]+" writes it); closed plan unreadable.md, whose stored record does not read (ele plan done "+ids["unreadable"]+" rewrites it).\n")+`$`, output,
		"a name the unchecked closed plan used starts unrefused")

	output, code = runCommand(t, "plan", "done", ids["gone"], "unreadable.md", "next.md")
	require.Equal(t, 0, code, output)
	stored, _, found := strings.Cut(output, "Done with plan next.md")
	require.True(t, found, output)
	require.Equal(t, "Stored the item names of closed plan gone.md ("+ids["gone"]+"), so they resolve edges and stay reserved.\n"+
		"Stored the item names of closed plan unreadable.md ("+ids["unreadable"]+"), so they resolve edges and stay reserved.\n", stored)
	writeFile(t, filepath.Join(gateway, "missing.md"), named("missing"))
	writeFile(t, filepath.Join(gateway, "broken.md"), named("broken"))
	require.NoError(t, os.Remove(filepath.Join(gateway, "directory.md")))
	writeFile(t, filepath.Join(gateway, "directory.md"), named("directory"))

	writeFile(t, filepath.Join(gateway, "again.md"), named("unreadable"))
	output, code = runCommand(t, "plan", "start", "again.md")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: again.md: item name \"unreadable\" was used by closed plan unreadable.md; item names are unique across active and closed plans\n", output, "the rewritten record reserves its names")
	writeFile(t, filepath.Join(gateway, "again.md"), named("fresh"))
	output, code = runCommand(t, "plan", "start", "again.md")
	require.Equal(t, 0, code, output)
	require.Regexp(t, `^Started plan again\.md as \S+; ele plan show prints its next item\.\n$`, output, "every plan checked, no line")

	output, code = runCommand(t, "plan", "done", "gone.md")
	require.Equal(t, 1, code, "a closed plan whose record reads is not an argument")
	require.Equal(t, "ele: gone.md is not an active plan of corp · gateway; ele plan list lists them, --project <name> for another project's\n", output)

	require.NoError(t, os.Remove(filepath.Join(records, ids["gone"]+".json")))
	writeFile(t, filepath.Join(gateway, "gone.md"), named("restarted"))
	output, code = runCommand(t, "plan", "start", "gone.md")
	require.Equal(t, 0, code, output)
	require.Regexp(t, `^Started plan gone\.md as \S+; ele plan show prints its next item\.\n$`, output, "a closed plan at its own path cannot clash, so it goes unnamed")
	output, code = runCommand(t, "plan", "done", ids["gone"])
	require.Equal(t, 1, code)
	require.Equal(t, "ele: closed plan gone.md ("+ids["gone"]+") shares its path with an active plan, so its names cannot be read from it\n", output)
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
1 warning; ele plan check .documentation/plans/build.md lists it.
Blocked:
- Start block
  blocked: the parser first
Next, line 12:
- [ ] Commands
  next: done and list
`, output)
	for _, subcommand := range []string{"check", "show"} {
		output, code = runCommand(t, "plan", subcommand, ".documentation/plans/build.md")
		require.Equal(t, 0, code, "the printed path from the checkout root resolves from a subdirectory: "+output)
		require.True(t, strings.HasPrefix(output, "Plan .documentation/plans/build.md: "), output)
	}
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "build.md"), planFile)
	require.NoError(t, os.Symlink(outside, filepath.Join(gateway, "elsewhere")))
	output, code = runCommand(t, "plan", "show", "elsewhere/build.md")
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "is outside the project", "a symlink out of the checkout is refused from a subdirectory too")

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
