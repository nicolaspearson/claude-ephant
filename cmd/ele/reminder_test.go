package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

func TestRunReminderCloseReArmsANudge(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	today := time.Now().Format(time.DateOnly)
	week := time.Now().AddDate(0, 0, 7).Format(time.DateOnly)
	output, code := runCommand(t, "remember", "--kind", "reminder", "--every", "7", "--done-file=~/marker", "--session", "", "Run another check.")
	require.Equal(t, 0, code, output)
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	tilde := readRemembered(t, output)
	require.Equal(t, filepath.Join(home, "marker"), tilde.DoneFile, "a ~ the shell left alone is the home directory")
	output, code = runCommand(t, "forget", tilde.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "remember", "--kind", "reminder", "--every", "7", "--done-file", "marker", "--session", "", "Run /hook-stats.")
	require.Equal(t, 0, code, output)
	nudge := readRemembered(t, output)
	require.Equal(t, today, nudge.Due, "--every with no --due is due today")
	require.Equal(t, 7, nudge.Every)
	require.Equal(t, filepath.Join(home, "w", "corp", "gateway", "marker"), nudge.DoneFile, "a relative done file is stored absolute")
	writeFile(t, nudge.DoneFile, "")
	require.NoError(t, os.Chtimes(nudge.DoneFile, time.Now().AddDate(0, 0, -30), time.Now().AddDate(0, 0, -30)))

	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Reminders for corp · gateway, soonest due first:\n"+
		"- due  ↻ due "+today+", reminder "+nudge.ID+" (project, every 7 days): Run /hook-stats.\n"+
		"ele reminder close <id>... or ele reminder reopen <id>... to change one; closing one with an interval re-arms it, ele forget <id> drops it.\n", output)

	output, code = runCommand(t, "reminder", "close", nudge.ID)
	require.Equal(t, 0, code, output)
	require.Equal(t, "Closed reminder "+nudge.ID+"; due again "+week+".\n", output)
	content, err := os.ReadFile(filepath.Join(home, "data", "notes", "corp", nudge.ID+".md"))
	require.NoError(t, err)
	require.Contains(t, string(content), "due: "+week+"\n")
	require.NotContains(t, string(content), "closed:", "a nudge stays open")
	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "- open  ↻ due "+week+", reminder "+nudge.ID+" (project, every 7 days)")

	output, code = runCommand(t, "forget", nudge.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "reminder", "list", "--all")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, nudge.ID, "forget drops a nudge for good")
}

func TestRunReminderListColoursOnlyWithColour(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	t.Setenv("NO_COLOR", "")
	output, code := runCommand(t, "remember", "--kind", "reminder", "--every", "7", "--session", "", "Run /hook-stats.")
	require.Equal(t, 0, code, output)
	nudge := readRemembered(t, output)

	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "- due  ↻ due "+nudge.Due+", reminder "+nudge.ID+" (project, every 7 days)")
	require.NotContains(t, output, "\033", "the model's call passes no flag")

	output, code = runCommand(t, "reminder", "list", "--colour")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "- \033[33mdue\033[39m  \033[35m↻\033[39m due \033[33m"+nudge.Due+"\033[39m, reminder \033[2m"+nudge.ID+"\033[22m (project, \033[35mevery 7 days\033[39m)")

	t.Setenv("NO_COLOR", "1")
	output, code = runCommand(t, "reminder", "list", "--colour")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "\033", "NO_COLOR wins over --colour")

	t.Setenv("NO_COLOR", "")
	writeFile(t, filepath.Join(home, "data", "config.json"), `{"owners": {"corp": {"colour": false}}}`)
	output, code = runCommand(t, "reminder", "list", "--colour")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "\033", "the colour option off wins over --colour")
}

func TestRunReminderListDropsControlCharacters(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	_, err := notes.Write(filepath.Join(home, "data", "notes"), notes.Note{Body: "Renew \033[31mthe\a certificate.", Date: "2026-10-22", Due: "2026-10-22", ID: "20261022-aaaaaa", Kind: notes.KindReminder, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Time: "2026-10-22T09:00:00Z"})
	require.NoError(t, err)

	output, code := runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "reminder 20261022-aaaaaa (project): Renew [31mthe certificate.\n")
}

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
		{args: []string{"--kind", "reminder", "--done-file", "/tmp/marker", "x"}, message: "ele: --done-file needs --every <days>, how long the nudge stays hidden after the file changes\n", name: "done file without every"},
		{args: []string{"--kind", "decision", "--due", "2026-11-01", "x"}, message: "ele: only a reminder takes --done-file, --due or --every; use --kind reminder\n", name: "due on a decision"},
		{args: []string{"--kind", "reminder", "--due", "next week", "x"}, message: "ele: --due \"next week\" is not YYYY-MM-DD\n", name: "due not a date"},
		{args: []string{"--kind", "reminder", "--every", "0", "--due", "2026-11-01", "x"}, message: "ele: --every 0 is not a whole number of days above 0\n", name: "every 0"},
		{args: []string{"--kind", "reminder", "--every", "-1", "x"}, message: "ele: --every -1 is not a whole number of days above 0\n", name: "every below 1"},
		{args: []string{"--kind", "fact", "--every", "0", "x"}, message: "ele: only a reminder takes --done-file, --due or --every; use --kind reminder\n", name: "every on a fact"},
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

func TestRunSessionStartShowsANudgeItsIntervalAfterItsDoneFileChanged(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	noteDirectory := filepath.Join(home, "data", "notes")
	now := time.Now()
	nudge := func(id string, due time.Time, doneFile string, body string) notes.Note {
		note := notes.Note{Body: body, Date: "2026-10-22", DoneFile: doneFile, Due: due.Format(time.DateOnly), Every: 7, ID: id, Kind: notes.KindReminder, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Time: "2026-10-22T09:00:00Z"}
		_, err := notes.Write(noteDirectory, note)
		require.NoError(t, err)
		return note
	}
	marked := func(name string, when time.Time) string {
		path := filepath.Join(home, name)
		writeFile(t, path, "")
		require.NoError(t, os.Chtimes(path, when, when))
		return path
	}
	fresh := nudge("20261022-aaaaaa", now.AddDate(0, 0, -30), marked("fresh", now.AddDate(0, 0, -6)), "Done six days ago.")
	stale := nudge("20261022-bbbbbb", now.AddDate(0, 0, -30), marked("stale", now.AddDate(0, 0, -8)), "Done eight days ago.")
	missing := nudge("20261022-cccccc", now.AddDate(0, 0, 30), filepath.Join(home, "never-written"), "Never done.")
	closedRecently := nudge("20261022-dddddd", now.AddDate(0, 0, 3), marked("rearmed", now.AddDate(0, 0, -30)), "Closed four days ago.")
	plain := nudge("20261022-eeeeee", now, "", "Due today.")
	overdue := nudge("20261022-ffffff", now.AddDate(0, 0, -20), "", "Due twenty days ago.")
	start := func() string {
		var output bytes.Buffer
		code := sessionStart(context.Background(), []string{"--no-sync"}, strings.NewReader(`{"session_id":"s","source":"startup"}`), &output, &output)
		require.Equal(t, 0, code, output.String())
		shown, _ := hookOutput(t, output.String())
		return shown
	}

	shown := start()
	require.Contains(t, shown, "Reminders from ele, due:\n"+
		"- ↻ due "+overdue.Due+", reminder "+overdue.ID+", every 7 days: Due twenty days ago.\n"+
		"- ↻ due "+now.AddDate(0, 0, -1).Format(time.DateOnly)+", reminder "+stale.ID+", every 7 days: Done eight days ago.\n"+
		"- ↻ due "+now.Format(time.DateOnly)+", reminder "+plain.ID+", every 7 days: Due today.\n"+
		"- ↻ due "+now.Format(time.DateOnly)+", reminder "+missing.ID+", every 7 days: Never done.\n"+
		"Tell the user each is due. Once the user says one is done or dropped, run ele reminder close <id>...; until then it shows at every startup, /clear and compaction.\n"+
		"Closing one shown with an interval re-arms it for that many days; ele forget <id> drops it for good.\n\n")
	require.NotContains(t, shown, fresh.ID, "done six days ago, it waits a seventh")
	require.NotContains(t, shown, closedRecently.ID, "closed, it waits for its due date though its done file is older")

	marked("stale", now)
	require.NotContains(t, start(), stale.ID, "a change to the done file marks the nudge done")

	writeFile(t, filepath.Join(home, "data", "config.json"), `{"owners": {"corp": {"reminders": false}}}`)
	require.NotContains(t, start(), "Reminders from ele")
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
