package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

func TestIntervalNamesItsLargestWholeUnit(t *testing.T) {
	tests := []struct {
		every    time.Duration
		interval string
	}{
		{every: 0, interval: ""},
		{every: time.Minute, interval: ", every 1 minute"},
		{every: time.Hour, interval: ", every 1 hour"},
		{every: 2 * time.Hour, interval: ", every 2 hours"},
		{every: 90 * time.Minute, interval: ", every 90 minutes"},
		{every: notes.Day, interval: ", every 1 day"},
		{every: 7 * notes.Day, interval: ", every 7 days"},
	}
	for _, test := range tests {
		require.Equal(t, test.interval, interval(notes.Note{Every: test.every}), test.every)
	}
}

func TestRunReminderAndForgetReachAnEverywhereNoteFromEachOwner(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	today := time.Now().Format(time.DateOnly)
	output, code := runCommand(t, "remember", "--kind", "reminder", "--everywhere", "--due", today, "--session", "", "Run /hook-stats.")
	require.Equal(t, 0, code, output)
	nudge := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "reminder", "--everywhere", "--due", today, "Run /hook-stats.")
	require.Equal(t, 1, code, "an everywhere note already says it: %s", output)

	t.Chdir(filepath.Join(home, "w", "personal", "notes"))
	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "- due  due "+today+", reminder "+nudge.ID+" (everywhere): Run /hook-stats.\n")
	output, code = runCommand(t, "reminder", "close", nudge.ID)
	require.Equal(t, 0, code, output)

	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, nudge.ID, "closed under one owner, closed under every owner")
	output, code = runCommand(t, "forget", nudge.ID)
	require.Equal(t, 0, code, output)
	require.FileExists(t, filepath.Join(home, "data", "forgotten", notes.OwnerEverywhere, nudge.ID+".md"))
	output, code = runCommand(t, "forget", "--restore", nudge.ID)
	require.Equal(t, 0, code, output)
	require.FileExists(t, filepath.Join(home, "data", "notes", notes.OwnerEverywhere, nudge.ID+".md"))

	elsewhere := filepath.Join(home, "elsewhere")
	require.NoError(t, os.MkdirAll(elsewhere, 0o700))
	t.Chdir(elsewhere)
	output, code = runCommand(t, "reminder", "reopen", nudge.ID)
	require.Equal(t, 0, code, "a session no owner covers manages the everywhere notes it shows: %s", output)
	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Reminders for elsewhere, soonest due first:\n- due  due "+today+", reminder "+nudge.ID+" (everywhere): Run /hook-stats.\n")
	output, code = runCommand(t, "reminder", "close", nudge.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "forget", nudge.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "handoff", "list")
	require.Equal(t, 1, code, "a handoff still needs an owner: %s", output)
}

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
	require.Equal(t, 7*24*time.Hour, nudge.Every)
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

	output, code = runCommand(t, "remember", "--kind", "reminder", "--every", "7", "--done-file", "never-written", "--session", "", "Run the unmarked check.")
	require.Equal(t, 0, code, output)
	unmarked := readRemembered(t, output)
	output, code = runCommand(t, "reminder", "close", unmarked.ID)
	require.Equal(t, 0, code, output)
	require.Equal(t, "Closed reminder "+unmarked.ID+"; its done file cannot be read, so it is due again now.\n", output)

	output, code = runCommand(t, "forget", nudge.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "reminder", "list", "--all")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, nudge.ID, "forget drops a nudge for good")
}

func TestRunReminderCloseReArmsASubDayNudgeToTheMinute(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	dataDirectory := filepath.Join(home, "data")
	output, code := runCommand(t, "remember", "--kind", "reminder", "--every", "5m", "--session", "", "Check the nudge.")
	require.Equal(t, 0, code, output)
	nudge := readRemembered(t, output)
	require.Equal(t, 5*time.Minute, nudge.Every)
	content, err := os.ReadFile(filepath.Join(dataDirectory, "notes", "corp", nudge.ID+".md"))
	require.NoError(t, err)
	require.Contains(t, string(content), "every: 5m\n")
	output, code = runCommand(t, "reminder", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "reminder "+nudge.ID+" (project, every 5 minutes)")

	closing := time.Now()
	output, code = runCommand(t, "reminder", "close", nudge.ID)
	require.Equal(t, 0, code, output)
	closed, err := os.ReadFile(filepath.Join(dataDirectory, "notes", "corp", nudge.ID+".md"))
	require.NoError(t, err)
	rearmed, err := notes.Parse(filepath.Join(dataDirectory, "notes", "corp", nudge.ID+".md"), closed)
	require.NoError(t, err)
	require.Equal(t, "Closed reminder "+nudge.ID+"; due again "+rearmed.Due+".\n", output)
	due, err := time.ParseInLocation(notes.DueMinute, rearmed.Due, time.Local)
	require.NoError(t, err, "a sub-day nudge re-arms to a time")
	require.False(t, due.Before(closing.Add(5*time.Minute)), "re-armed %s, closed %s: never early", rearmed.Due, closing)
	require.True(t, due.Before(closing.Add(7*time.Minute)), "re-armed %s, closed %s", rearmed.Due, closing)

	start := func(now time.Time) string {
		var output bytes.Buffer
		_, err := writeSessionStart(context.Background(), &output, dataDirectory, now, "startup")
		require.NoError(t, err)
		return output.String()
	}
	require.NotContains(t, start(closing.Add(4*time.Minute+59*time.Second)), nudge.ID, "within five minutes of its close")
	require.Contains(t, start(due), "- ↻ due "+rearmed.Due+", reminder "+nudge.ID+", every 5 minutes: Check the nudge.\n")
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
	require.Equal(t, "ele: "+now.ID+" is not a closed reminder of corp · gateway. ele reminder list --all lists them, and --project <name> looks in another project.\n", output)
	output, code = runCommand(t, "reminder", "close", now.ID, "20990101-bogus0")
	require.Equal(t, 1, code, output)
	require.Equal(t, "ele: 20990101-bogus0 is not an open reminder of corp · gateway. ele reminder list lists them, and --project <name> looks in another project.\n", output)
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
