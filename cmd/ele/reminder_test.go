package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

func TestRunReminderListsClosesAndReopens(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	today := time.Now().Format(time.DateOnly)
	output, code := runCommand(t, "remember", "--kind", "reminder", "--due", "2099-01-01", "--session", "", "Renew the gateway certificate.")
	require.Equal(t, 0, code, output)
	later := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "reminder", "--due", today, "--global", "--session", "", "Review the plan candidates.")
	require.Equal(t, 0, code, output)
	now := readRemembered(t, output)
	require.Equal(t, today, now.Due)

	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Reminders for corp · gateway, soonest due first:\n"+
		"- due  due "+today+", reminder "+now.ID+" (global): Review the plan candidates.\n"+
		"- open  due 2099-01-01, reminder "+later.ID+" (project): Renew the gateway certificate.\n"+
		"ele reminder close <id>... or ele reminder reopen <id>... to change one.\n", output)

	output, code = runCommand(t, "reminder", "reopen", now.ID)
	require.Equal(t, 1, code, output)
	require.Contains(t, output, now.ID+" is not a closed reminder of corp · gateway")
	output, code = runCommand(t, "reminder", "close", now.ID, "20990101-bogus0")
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "20990101-bogus0 is not an open reminder of corp · gateway")
	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, now.ID, "a mistyped id closes nothing")

	output, code = runCommand(t, "reminder", "close", now.ID)
	require.Equal(t, 0, code, output)
	require.Equal(t, "Closed reminder "+now.ID+".\n", output)
	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, now.ID)
	output, code = runCommand(t, "reminder", "list", "--all")
	require.Equal(t, 0, code, output)
	require.Regexp(t, `- closed \d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ  due `+today+`, reminder `+now.ID, output)
	output, code = runCommand(t, "remember", "--kind", "reminder", "--due", "2099-01-01", "Renew the gateway certificate.")
	require.Equal(t, 1, code, "an open reminder already says it: %s", output)
	output, code = runCommand(t, "remember", "--kind", "reminder", "--due", today, "--global", "--session", "", "Review the plan candidates.")
	require.Equal(t, 0, code, "a closed reminder no longer shows, so its words may be saved again: %s", output)
	again := readRemembered(t, output)
	output, code = runCommand(t, "forget", again.ID)
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "reminder", "reopen", now.ID)
	require.Equal(t, 0, code, output)
	require.Equal(t, "Reopened reminder "+now.ID+".\n", output)
	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "- due  due "+today+", reminder "+now.ID)
}

func TestRunRememberRefusesAReminderWithoutADate(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	tests := []struct {
		args    []string
		message string
		name    string
	}{
		{args: []string{"--kind", "decision", "--due", "2026-11-01", "x"}, message: "ele: only a reminder takes --due; use --kind reminder\n", name: "due on a decision"},
		{args: []string{"--kind", "reminder", "--due", "next week", "x"}, message: "ele: --due \"next week\" is not YYYY-MM-DD\n", name: "due not a date"},
		{args: []string{"--kind", "reminder", "x"}, message: "ele: a reminder needs --due YYYY-MM-DD, the day it first shows\n", name: "reminder without due"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, code := runCommand(t, append([]string{"remember"}, test.args...)...)
			require.Equal(t, 2, code, output)
			require.Equal(t, test.message, output)
		})
	}
}

func TestRunSessionStartShowsRemindersFromTheirDueDateUntilClosed(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	noteDirectory := filepath.Join(home, "data", "notes")
	now := time.Now()
	reminder := func(id string, due time.Time, scope string, project string, body string) notes.Note {
		note := notes.Note{Body: body, Date: "2026-10-22", Due: due.Format(time.DateOnly), ID: id, Kind: notes.KindReminder, Owner: "corp", Project: project, Scope: scope, Time: "2026-10-22T09:00:00Z"}
		_, err := notes.Write(noteDirectory, note)
		require.NoError(t, err)
		return note
	}
	tomorrow := reminder("20261022-aaaaaa", now.AddDate(0, 0, 1), notes.ScopeProject, "gateway", "Not yet.")
	dueToday := reminder("20261022-bbbbbb", now, notes.ScopeProject, "gateway", "Due today.")
	overdue := reminder("20261022-cccccc", now.AddDate(0, 0, -3), notes.ScopeGlobal, "", "Overdue, every project.")
	closed := reminder("20261022-dddddd", now.AddDate(0, 0, -1), notes.ScopeProject, "gateway", "Already done.")
	require.NoError(t, notes.MarkClosed(noteDirectory, closed, now))
	elsewhere := reminder("20261022-eeeeee", now, notes.ScopeProject, "billing", "Another project's.")
	replaced := reminder("20261022-ffffff", now, notes.ScopeProject, "gateway", "Reworded below.")
	_, err := notes.Write(noteDirectory, notes.Note{Body: "Reworded.", Date: "2026-10-22", Due: now.AddDate(0, 0, 7).Format(time.DateOnly), ID: "20261022-ababab", Kind: notes.KindReminder, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Supersedes: []string{replaced.ID}})
	require.NoError(t, err)
	// Fills the notes budget, which reminders sit outside.
	for index := range 80 {
		_, err := notes.Write(noteDirectory, notes.Note{Body: strings.Repeat("filler ", 12), Date: "2026-10-01", ID: "20261001-" + strings.Repeat("0", 4) + string(rune('a'+index/26)) + string(rune('a'+index%26)), Kind: notes.KindFact, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject})
		require.NoError(t, err)
	}
	start := func(source string) (string, string) {
		var output bytes.Buffer
		code := sessionStart(context.Background(), []string{"--no-sync"}, strings.NewReader(`{"session_id":"s","source":"`+source+`"}`), &output, &output)
		require.Equal(t, 0, code, output.String())
		return hookOutput(t, output.String())
	}
	section := "Reminders from ele, due:\n" +
		"- due " + overdue.Due + ", reminder " + overdue.ID + ": Overdue, every project.\n" +
		"- due " + dueToday.Due + ", reminder " + dueToday.ID + ": Due today.\n"

	for _, source := range []string{"clear", "compact", "startup"} {
		shown, message := start(source)
		require.Contains(t, shown, section+"Tell the user each is due. Once the user says one is done or dropped, run ele reminder close <id>...; until then it shows at every startup, /clear and compaction.\n\nNotes from ele", source)
		require.Contains(t, message, strings.TrimSuffix(section, "\n"), "the user sees the reminders the session loaded")
		require.Contains(t, shown, "older notes not shown", "the notes budget is spent")
		for _, hidden := range []notes.Note{tomorrow, closed, elsewhere, replaced} {
			require.NotContains(t, shown, hidden.ID, hidden.Body)
		}
	}

	writeFile(t, filepath.Join(home, "data", "config.json"), `{"owners": {"corp": {"reminders": false}}}`)
	shown, message := start("startup")
	require.NotContains(t, shown, "Reminders from ele")
	require.NotContains(t, message, "Reminders from ele")
	require.Contains(t, shown, "Config from ele, off its defaults: reminders false")
}
