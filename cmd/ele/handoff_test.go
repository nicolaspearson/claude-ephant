package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

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
		require.Equal(t, "ele: "+identifier+" is not a pending handoff of corp · gateway. ele context lists them, and --project <name> looks in another project.\n", output)
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
		require.Equal(t, "ele: "+identifier+" is not a closed handoff of corp · gateway. ele handoff list --all lists them, and --project <name> looks in another project.\n", output)
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
