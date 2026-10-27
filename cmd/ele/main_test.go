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
	"github.com/nicolaspearson/claude-ephant/internal/store"
)

// A test that forgets its own HOME or ELE_HOME must not reach the real data dir: a background sync there runs this test binary again.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "ele-test-")
	if err != nil {
		panic(err)
	}
	if err = os.Setenv("HOME", home); err == nil {
		err = os.Setenv("ELE_HOME", filepath.Join(home, "data"))
	}
	if err != nil {
		panic(err)
	}
	planClock = func() time.Time { return time.Date(2026, 10, 21, 12, 0, 0, 0, time.Local) }
	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}

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
