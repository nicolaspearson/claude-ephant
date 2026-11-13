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

	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/store"
)

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		count int64
		text  string
	}{
		{count: 0, text: "0 B"},
		{count: 1023, text: "1023 B"},
		{count: 1536, text: "1.5 KiB"},
		{count: 1048575, text: "1.0 MiB"},
		{count: 2523886221, text: "2.4 GiB"},
	}
	for _, test := range tests {
		require.Equal(t, test.text, humanBytes(test.count))
	}
}

func TestLastSyncAndStopStateAreReplacedRatherThanWrittenInto(t *testing.T) {
	home := recallFixture(t)
	stamp := filepath.Join(home, "data", "last-sync")
	held := filepath.Join(t.TempDir(), "held-stamp")
	require.NoError(t, os.Link(stamp, held))
	before, err := os.ReadFile(stamp)
	require.NoError(t, err)
	skippedList := filepath.Join(home, "data", syncSkippedFile)
	writeFile(t, skippedList, "old\n")
	heldList := filepath.Join(t.TempDir(), "held-list")
	require.NoError(t, os.Link(skippedList, heldList))
	unreadable := filepath.Join(home, ".claude", "projects", "-gateway", "zzz9.jsonl")
	writeFile(t, unreadable, "{}\n")
	require.NoError(t, os.Chmod(unreadable, 0o000))

	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)

	kept, err := os.ReadFile(held)
	require.NoError(t, err)
	require.Equal(t, string(before), string(kept), "last-sync written in place")
	keptList, err := os.ReadFile(heldList)
	require.NoError(t, err)
	require.Equal(t, "old\n", string(keptList), "sync-skipped written in place")

	state := filepath.Join(t.TempDir(), "s1.json")
	require.NoError(t, writeStopState(state, stopState{}))
	heldState := filepath.Join(t.TempDir(), "held-state")
	require.NoError(t, os.Link(state, heldState))
	require.NoError(t, writeStopState(state, stopState{Offset: 42}))
	keptState, err := os.ReadFile(heldState)
	require.NoError(t, err)
	require.NotContains(t, string(keptState), "42", "stop-check state written in place")
}

// A test that forgets its own HOME or ELE_HOME must not reach the real data dir: a background sync there runs this test binary again.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "ele-test-")
	if err != nil {
		panic(err)
	}
	if err = os.Setenv("HOME", home); err == nil {
		err = os.Setenv("ELE_HOME", filepath.Join(home, "data"))
	}
	// A profile the test's own shell runs under would join every discovery.
	if err == nil {
		err = os.Unsetenv("CLAUDE_CONFIG_DIR")
	}
	if err != nil {
		panic(err)
	}
	planClock = func() time.Time { return time.Date(2026, 10, 21, 12, 0, 0, 0, time.Local) }
	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}

func TestRunReindexRebuildsFromTheArchiveAndNotes(t *testing.T) {
	home := recallFixture(t)
	noteDirectory := filepath.Join(home, "data", "notes")
	_, err := notes.Write(noteDirectory, notes.Note{Body: "Pin pydantic below 2.14.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject})
	require.NoError(t, err)
	_, err = notes.Write(noteDirectory, notes.Note{Body: "Pin pydantic through the wrapper.", Date: "2026-10-16", ID: "20261016-bbbbbb", Kind: notes.KindDecision, Owner: "corp", Project: "proxy", Scope: notes.ScopeProject, Supersedes: []string{"20261015-aaaaaa"}})
	require.NoError(t, err)
	writeFile(t, filepath.Join(noteDirectory, "corp", "20261015-broken.md"), "no front matter\n")

	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "; indexed 2 notes.")
	require.Contains(t, output, "Skipped a note: invalid note notes/corp/20261015-broken.md: first line is not ---")

	before, code := runCommand(t, "recall", "--no-sync", "--all", "pydantic")
	require.Equal(t, 0, code, before)
	require.Contains(t, before, "4 matches")
	require.Contains(t, before, "\n2026-10-15 · corp · gateway\n  note decision          (superseded by 20261016-bbbbbb) 20261015-aaaaaa Pin «pydantic» below 2.14.\n")
	require.Contains(t, before, "\n2026-10-16 · corp · proxy\n  note decision          20261016-bbbbbb Pin «pydantic» through the wrapper.\n")
	require.Contains(t, before, "  prompt                 pin «pydantic» below 2.14\n")
	require.Contains(t, before, "Expand a session")

	output, code = runCommand(t, "reindex")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Rebuilt the index from the archive and notes. Archived 0 B from 0 transcripts; indexed 3 entries from 2 files; indexed 2 notes.")

	after, code := runCommand(t, "recall", "--no-sync", "--all", "pydantic")
	require.Equal(t, 0, code, after)
	require.Equal(t, before, after)
}

func TestRunSyncArchivesAndIndexesEveryProfile(t *testing.T) {
	home := t.TempDir()
	dataDirectory := filepath.Join(home, "data")
	t.Setenv("HOME", home)
	t.Setenv("ELE_HOME", dataDirectory)
	for _, profile := range []string{".claude", ".claude-corp"} {
		transcript := filepath.Join(home, profile, "projects", "-repo", profile+".jsonl")
		require.NoError(t, os.MkdirAll(filepath.Dir(transcript), 0o700))
		require.NoError(t, os.WriteFile(transcript, []byte(`{"type":"user","cwd":"/w/repo","sessionId":"s","message":{"content":"remember this"}}`+"\n"), 0o600))
	}

	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, run(context.Background(), []string{"sync"}, &stdout, &stderr), stderr.String())
	require.Equal(t, "Archived 172 B from 2 transcripts; indexed 2 entries from 2 files.\n", stdout.String())
	require.FileExists(t, filepath.Join(dataDirectory, "archive", "transcripts", "corp", "-repo", ".claude-corp.jsonl.gz"))
	require.FileExists(t, filepath.Join(dataDirectory, "last-sync"))

	stdout.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"sync"}, &stdout, &stderr), stderr.String())
	require.Equal(t, "Archived 0 B from 0 transcripts; indexed 0 entries from 0 files.\n", stdout.String())
}

func TestRunSyncArchivesTheConfigDirectoryProfileOutsideHome(t *testing.T) {
	home := t.TempDir()
	dataDirectory := filepath.Join(home, "data")
	scratch := filepath.Join(t.TempDir(), "scratch")
	t.Setenv("HOME", home)
	t.Setenv("ELE_HOME", dataDirectory)
	t.Setenv("CLAUDE_CONFIG_DIR", scratch)
	writeFile(t, filepath.Join(scratch, "projects", "-repo", "s.jsonl"), `{"type":"user","cwd":"/w/repo","sessionId":"s","message":{"content":"remember this"}}`+"\n")

	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Archived 86 B from 1 transcripts; indexed 1 entries from 1 files.\n", output)
	require.FileExists(t, filepath.Join(dataDirectory, "archive", "transcripts", "scratch", "-repo", "s.jsonl.gz"))
}

func TestRunSyncBuildsANewIndexWithoutRebuilding(t *testing.T) {
	dataDirectory := filepath.Join(t.TempDir(), "data")
	t.Setenv("ELE_HOME", dataDirectory)
	writeFile(t, filepath.Join(dataDirectory, "owner-map.json"), `{"owners": ["corp"]}`)

	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "rebuilt")
}

func TestRunSyncDatesABackgroundRunAtItsStartAndAtTheLock(t *testing.T) {
	home := recallFixture(t)
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, syncLogHeader, "a foreground sync writes no header")

	t.Setenv(backgroundSyncVariable, "1")
	output, code = runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	lines := strings.Split(output, "\n")
	started, isHeader := strings.CutPrefix(lines[0], syncLogHeader)
	require.True(t, isHeader, output)
	locked, isLock := strings.CutPrefix(lines[1], syncLogLocked)
	require.True(t, isLock, output)
	header, err := time.Parse(time.RFC3339Nano, started)
	require.NoError(t, err)
	lock, err := time.Parse(time.RFC3339Nano, locked)
	require.NoError(t, err)
	stamp, err := os.ReadFile(filepath.Join(home, "data", "last-sync"))
	require.NoError(t, err)
	last, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(stamp)))
	require.NoError(t, err)
	require.False(t, lock.Before(header))
	require.True(t, last.After(lock), "a run's own stamp is later than its lock line")

	writeFile(t, filepath.Join(home, "data", "owner-map.json"), "{not json")
	output, code = runCommand(t, "sync")
	require.Equal(t, 1, code)
	require.True(t, strings.HasPrefix(output, syncLogHeader), "a failure before the lock still has a time above it: "+output)
}

func TestRunSyncRebuildsAnIndexOfAnotherVersion(t *testing.T) {
	home := recallFixture(t)
	var output string
	var code int
	for _, older := range []string{"", "1"} {
		searchIndex, err := index.Open(filepath.Join(home, "data", "ele.db"))
		require.NoError(t, err)
		require.NoError(t, searchIndex.SetSetting(context.Background(), "index-version", older))
		require.NoError(t, searchIndex.Close())

		output, code = runCommand(t, "sync")
		require.Equal(t, 0, code, output)
		require.Contains(t, output, "The index was built by another version of ele, so it was rebuilt.", older)
		output, code = runCommand(t, "sync")
		require.Equal(t, 0, code, output)
		require.NotContains(t, output, "rebuilt", older)
	}
}

func TestRunSyncRebuildsAnIndexThatCutBeforeRedacting(t *testing.T) {
	home := recallFixture(t)
	db, err := store.Open(filepath.Join(home, "data", "ele.db"))
	require.NoError(t, err)
	// An index built by version 12, which cut before redacting.
	_, err = db.Exec(`UPDATE entries SET text = text || ' ghp_abcdefghijklmn…'; UPDATE settings SET value = '12' WHERE key = 'index-version'`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)

	db, err = store.Open(filepath.Join(home, "data", "ele.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	var fragments int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM entries WHERE instr(text, 'ghp_') > 0`).Scan(&fragments))
	require.Zero(t, fragments)
}

func TestRunSyncRebuildsTablesThatDoNotMatch(t *testing.T) {
	home := recallFixture(t)
	searchIndex, err := index.Open(filepath.Join(home, "data", "ele.db"))
	require.NoError(t, err)
	require.NoError(t, searchIndex.Reset(context.Background()))
	require.NoError(t, searchIndex.Close())
	db, err := store.Open(filepath.Join(home, "data", "ele.db"))
	require.NoError(t, err)
	_, err = db.Exec(`DROP TABLE events; CREATE TABLE events (attributes TEXT NOT NULL, id INTEGER PRIMARY KEY, kind TEXT NOT NULL, name TEXT NOT NULL, owner TEXT NOT NULL, profile TEXT NOT NULL, project TEXT NOT NULL, session TEXT NOT NULL, source TEXT NOT NULL, subagent INTEGER NOT NULL, timestamp TEXT NOT NULL, uuid TEXT NOT NULL)`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	refusal := "the index's tables do not match this version of ele (events lacks column line_key); run ele sync to rebuild it"
	for _, args := range [][]string{{"events", "--no-sync"}, {"remember", "--kind", "fact", "--project", "gateway", "Not written."}, {"stats"}} {
		output, code := runCommand(t, args...)
		require.Equal(t, 1, code, args)
		require.Contains(t, output, "ele: "+refusal, args)
	}
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "session-start", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "ele could not count past sessions: "+refusal)

	output, code = runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "The index's tables did not match this version of ele (events lacks column line_key), so it was rebuilt.")
	output, code = runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "rebuilt")
	output, code = runCommand(t, "events", "--no-sync")
	require.Equal(t, 0, code, output)
}

func TestRunSyncRefusesBadArguments(t *testing.T) {
	for _, args := range [][]string{{"sync", "--background", "extra"}, {"sync", "--foo"}} {
		var stdout, stderr bytes.Buffer
		require.Equal(t, 2, run(context.Background(), args, &stdout, &stderr), args)
		require.Contains(t, stderr.String(), "ele sync [--background]\n")
		require.NotContains(t, stderr.String(), "ele backup")
	}
}

func TestRunSyncReportsAFailedBackgroundStart(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocked, nil, 0o600))
	t.Setenv("ELE_HOME", filepath.Join(blocked, "data"))

	var stdout, stderr bytes.Buffer
	require.Equal(t, 1, run(context.Background(), []string{"sync", "--background"}, &stdout, &stderr))
	require.Contains(t, stderr.String(), "ele: could not start a background sync: ")
}

func TestRunSyncSkipsAConfigDirectoryProfileWhoseNameIsTaken(t *testing.T) {
	home := t.TempDir()
	dataDirectory := filepath.Join(home, "data")
	elsewhere := filepath.Join(t.TempDir(), ".claude-corp")
	t.Setenv("HOME", home)
	t.Setenv("ELE_HOME", dataDirectory)
	t.Setenv("CLAUDE_CONFIG_DIR", elsewhere)
	for _, profile := range []string{filepath.Join(home, ".claude-corp"), elsewhere} {
		writeFile(t, filepath.Join(profile, "projects", "-repo", "s.jsonl"), `{"type":"user","cwd":"/w/repo","sessionId":"s","message":{"content":"remember this"}}`+"\n")
	}

	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Archived 86 B from 1 transcripts")
	require.Contains(t, output, "\nSkipped: profile "+elsewhere+": its name corp is taken by "+filepath.Join(home, ".claude-corp")+"\n")
	require.FileExists(t, filepath.Join(dataDirectory, syncSkippedFile))
}

func TestRunSyncSkipsAFileItCannotReadAndSaysSoUntilASyncSkipsNone(t *testing.T) {
	home := recallFixture(t)
	dataDirectory := filepath.Join(home, "data")
	unreadable := filepath.Join(home, ".claude", "projects", "-gateway", "zzz9.jsonl")
	writeFile(t, unreadable, `{"type":"user","cwd":"/w","sessionId":"zzz9","message":{"content":"hidden"}}`+"\n")
	require.NoError(t, os.Chmod(unreadable, 0o000))
	// Sync names a file under projects/ as it resolves.
	resolved, err := filepath.EvalSymlinks(filepath.Dir(unreadable))
	require.NoError(t, err)
	named := filepath.Join(resolved, "zzz9.jsonl")

	output, code := runCommand(t, "sync")

	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Skipped: archive "+named)
	require.FileExists(t, filepath.Join(dataDirectory, "last-sync"))
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code = runCommand(t, "session-start", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "ele's sync skipped 1 entry; ele status lists them.\n")
	output, code = runCommand(t, "status")
	require.Equal(t, 0, code, output)
	require.Regexp(t, `\nSync skipped +1 entry, listed below\n`, output)
	require.Contains(t, output, "\nSkipped by the last sync:\n  archive "+named)

	require.NoError(t, os.Chmod(unreadable, 0o600))
	output, code = runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.NoFileExists(t, filepath.Join(dataDirectory, syncSkippedFile))
	output, _ = runCommand(t, "session-start", "--no-sync")
	require.NotContains(t, output, "skipped")
}

func TestRunSyncSweepsOldTemporaryNotesFromNotesAndForgotten(t *testing.T) {
	home := recallFixture(t)
	// As a killed ele owner move leaves them, in either tree.
	left := []string{
		filepath.Join(home, "data", "forgotten", "personal", "20261017-aaaaaa.md.123.tmp"),
		filepath.Join(home, "data", "notes", "personal", "20261017-bbbbbb.md.456.tmp"),
	}
	for _, path := range left {
		writeFile(t, path, "half")
		old := time.Now().Add(-2 * 24 * time.Hour)
		require.NoError(t, os.Chtimes(path, old, old))
	}

	output, code := runCommand(t, "sync")

	require.Equal(t, 0, code, output)
	for _, path := range left {
		require.NoFileExists(t, path)
	}
}
