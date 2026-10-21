package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

func TestAge(t *testing.T) {
	tests := []struct {
		elapsed time.Duration
		text    string
	}{
		{elapsed: 5 * time.Minute, text: "5m"},
		{elapsed: time.Hour, text: "1h"},
		{elapsed: 3*time.Hour + 12*time.Minute, text: "3h12m"},
		{elapsed: 9*24*time.Hour + 4*time.Hour + 30*time.Minute, text: "9d4h"},
	}
	for _, test := range tests {
		require.Equal(t, test.text, age(test.elapsed))
	}
}

func TestRunStatsCountsWhatTheIndexAndNotesHold(t *testing.T) {
	home := recallFixture(t)
	noteDirectory := filepath.Join(home, "data", "notes")
	for _, note := range []notes.Note{
		{Body: "Pin pydantic.", Date: "2026-10-14", ID: "20261014-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal},
		{Body: "Old question.", Date: "2026-10-13", ID: "20261013-bbbbbb", Kind: notes.KindQuestion, Owner: "corp", Scope: notes.ScopeGlobal},
		{Body: "Answer.", Date: "2026-10-14", ID: "20261014-cccccc", Kind: notes.KindFact, Owner: "corp", Scope: notes.ScopeGlobal, Supersedes: "20261013-bbbbbb"},
		{Body: "Still open?", Date: "2026-10-14", ID: "20261014-dddddd", Kind: notes.KindQuestion, Owner: "corp", Scope: notes.ScopeGlobal},
	} {
		_, err := notes.Write(noteDirectory, note)
		require.NoError(t, err)
	}
	writeFile(t, filepath.Join(noteDirectory, "corp", "broken.md"), "no front matter\n")
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "stats")
	require.Equal(t, 0, code, output)
	for _, line := range []string{
		`Covers 2026-10-13 to 2026-10-15\.`,
		`Entries +7\n`,
		`  by source +notes 4, transcripts 3\n`,
		`  by kind +note 4, prompt 2, conversation 1\n`,
		`  by owner +corp 6, personal 1\n`,
		`Sessions +claude 2\n`,
		`Notes, corp +answered question 1, decision 1, fact 1, invalid 1, open question 1\n`,
		`Notes, personal +none\n`,
		`On disk +archive [0-9.]+ [KMG]?i?B, index [0-9.]+ [KMG]?i?B, notes [0-9.]+ [KMG]?i?B\n`,
	} {
		require.Regexp(t, regexp.MustCompile(line), output)
	}
}

func TestRunStatsOnAFreshInstall(t *testing.T) {
	dataDirectory := filepath.Join(t.TempDir(), "data")
	t.Setenv("ELE_HOME", dataDirectory)

	output, code := runCommand(t, "stats")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "The index is empty.\n")
	require.Regexp(t, `On disk +archive 0 B, index 0 B, notes 0 B\n`, output)
	require.NoDirExists(t, dataDirectory)
}

func TestRunStatusReportsSyncOwnerAndPlugin(t *testing.T) {
	home := recallFixture(t)
	writeFile(t, filepath.Join(home, ".claude", "settings.json"), `{"enabledPlugins": {"ele@claude-ephant": true}}`)
	writeFile(t, filepath.Join(home, ".claude-corp", "projects", "-x", "s.jsonl"), "")
	writeFile(t, filepath.Join(home, ".claude-corp", "settings.json"), `{"enabledPlugins": {"ele@claude-ephant": false, "other@market": {"scope": "project"}}}`)
	writeFile(t, filepath.Join(home, ".claude-work", "projects", "-x", "s.jsonl"), "")
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))

	output, code := runCommand(t, "status")
	require.Equal(t, 0, code, output)
	for _, line := range []string{
		`Last sync +\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ \(just now\)\n`,
		`Sync running +no\n`,
		`Here +corp · gateway\n`,
		`Plugin ele@claude-ephant +claude enabled, corp disabled, work no settings.json\n`,
		`\nNo background sync has written sync.log\.\n$`,
	} {
		require.Regexp(t, regexp.MustCompile(line), output)
	}

	unlock, err := archive.Lock(filepath.Join(home, "data", "archive", "transcripts"))
	require.NoError(t, err)
	defer unlock()
	output, code = runCommand(t, "status")
	require.Equal(t, 0, code, output)
	require.Regexp(t, `Sync running +yes\n`, output)
}

func TestWriteStatusKeepsEveryRowWhenTheOwnerMapIsBroken(t *testing.T) {
	home := recallFixture(t)
	writeFile(t, filepath.Join(home, "data", "owner-map.json"), "{bad")
	writeFile(t, filepath.Join(home, "data", "sync.log"), "\n")

	var output bytes.Buffer
	require.NoError(t, writeStatus(&output, time.Now()))
	require.Regexp(t, `Here +parse .*owner-map\.json: invalid character`, output.String())
	require.Regexp(t, `Sync running +no\n`, output.String())
	require.Contains(t, output.String(), "\nsync.log is empty.\n")
}

func TestWriteStatusLeavesAFreshDataDirectoryUntouched(t *testing.T) {
	home := t.TempDir()
	dataDirectory := filepath.Join(home, "data")
	t.Setenv("HOME", home)
	t.Setenv("ELE_HOME", dataDirectory)

	var output bytes.Buffer
	require.NoError(t, writeStatus(&output, time.Now()))
	require.Regexp(t, `Last sync +none recorded\n`, output.String())
	require.Regexp(t, `Sync running +no\n`, output.String())
	require.Regexp(t, `Plugin ele@claude-ephant +no profiles found\n`, output.String())
	require.NoDirExists(t, dataDirectory)
}

func TestWriteStatusNamesAnUnlistedOwner(t *testing.T) {
	home := recallFixture(t)
	elsewhere := filepath.Join(home, "x")
	writeFile(t, filepath.Join(elsewhere, "file"), "")
	t.Chdir(elsewhere)
	tests := []struct {
		message  string
		name     string
		ownerMap string
	}{
		{message: `Here +no owner covers .*/x; ele owner add-project x <owner>, or ele owner add-path <dir> <owner> for a directory above it, maps it\n`, name: "nothing maps it", ownerMap: `{"owners": ["corp"]}`},
		{message: `Here +a path rule maps .*/x to owner "team", which owner-map\.json does not list in owners; ele owner add team lists it\n`, name: "path rule names an unlisted owner", ownerMap: `{"owners": ["corp"], "paths": {"` + home + `": "team"}}`},
		{message: `Here +project "x" maps to owner "team", which owner-map\.json does not list in owners; ele owner add team lists it\n`, name: "project names an unlisted owner", ownerMap: `{"owners": ["corp"], "projects": {"x": "team"}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writeFile(t, filepath.Join(home, "data", "owner-map.json"), test.ownerMap)
			var output bytes.Buffer
			require.NoError(t, writeStatus(&output, time.Now()))
			require.Regexp(t, test.message, output.String())
		})
	}
}

func TestWriteStatusShowsAgeAndTheSyncLogSinceTheLastSuccess(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	writeFile(t, filepath.Join(home, "data", "last-sync"), "2026-10-15T10:00:00Z\n")
	now := time.Date(2026, 10, 15, 13, 12, 0, 0, time.UTC)
	show := func(log string) string {
		t.Helper()
		writeFile(t, filepath.Join(home, "data", "sync.log"), log)
		var output bytes.Buffer
		require.NoError(t, writeStatus(&output, now))
		return output.String()
	}

	output := show("an error from before headers\n--- background sync 2026-10-15T10:00:00Z\n--- locked 2026-10-15T10:00:00Z\nArchived 1 KiB.\n")
	require.Regexp(t, `Last sync +2026-10-15T10:00:00Z \(3h12m ago\)\n`, output)
	require.Contains(t, output, "\nsync.log holds nothing since the last successful sync.\n", "older lines, and a run locking at the stamp itself, are history")

	output = show("--- background sync 2026-10-15T09:58:00Z\n--- locked 2026-10-15T09:59:00Z\nArchived 1 KiB.\n" +
		"--- background sync 2026-10-15T09:59:30Z\n--- locked 2026-10-15T11:00:00Z\nele: queued failure\n" +
		"--- background sync 2026-10-15T11:30:00Z\nele: parse owner-map.json: bad\n" +
		"--- background sync garbage\nele: undated\n" +
		"--- background sync 2026-10-15T12:30:00Z\n--- locked 2026-10-15T12:30:00Z\n")
	require.Contains(t, output, "\nsync.log since the last successful sync, newest last:\n"+
		"  --- background sync 2026-10-15T09:59:30Z\n  --- locked 2026-10-15T11:00:00Z\n  ele: queued failure\n"+
		"  --- background sync 2026-10-15T11:30:00Z\n  ele: parse owner-map.json: bad\n"+
		"  --- background sync 2026-10-15T12:30:00Z\n  --- locked 2026-10-15T12:30:00Z\n  (no output)\n",
		"a run that queued behind the success, one that failed before the lock, and one still running; not one whose time does not parse")
	require.NotContains(t, output, "Archived 1 KiB")

	output = show("--- background sync 2026-10-15T11:00:00Z\nele: one\n" +
		"--- background sync 2026-10-15T12:00:00Z\n--- locked 2026-10-15T12:00:00Z\npanic: x\nframe 1\nframe 2\nframe 3\nframe 4\nframe 5\nframe 6\nframe 7\n")
	require.Contains(t, output, "newest last:\n  --- background sync 2026-10-15T12:00:00Z\n  frame 1\n", "a long newest run keeps its header and its last lines")
	require.NotContains(t, output, "ele: one", "older runs that do not fit are left out whole")
	require.NotContains(t, output, "(no output)")

	require.NoError(t, os.Remove(filepath.Join(home, "data", "last-sync")))
	output = show("one\ntwo\n")
	require.Contains(t, output, "\nTail of sync.log, newest last:\n  one\n  two\n", "with no stamp to compare against, the plain tail")

	writeFile(t, filepath.Join(home, "data", "last-sync"), "2026-10-15T10:00:00Z\n")
	var future bytes.Buffer
	require.NoError(t, writeStatus(&future, time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)))
	require.Regexp(t, `Last sync +2026-10-15T10:00:00Z \(in the future; check the clock\)\n`, future.String())
}
