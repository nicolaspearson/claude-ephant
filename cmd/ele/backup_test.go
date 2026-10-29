package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/store"
)

func TestBackupCopiesNotesBeforePlans(t *testing.T) {
	notesAt, plansAt := slices.Index(backupParts, "notes"), slices.Index(backupParts, "plans")
	require.NotEqual(t, -1, notesAt)
	require.NotEqual(t, -1, plansAt)
	require.Less(t, notesAt, plansAt, "ele plan done stores a plan's names before marking its note closed, so a note copied as closed has whatever names were stored in the plans copied after it")
}

func TestRunBackupCopiesTheDataDirectoryAndTheIndex(t *testing.T) {
	home := recallFixture(t)
	data := filepath.Join(home, "data")
	writeFile(t, filepath.Join(data, "forgotten", "personal", "20261017-bbbbbb.md"), "a forgotten note\n")
	writeFile(t, filepath.Join(data, "notes", "personal", "20261017-aaaaaa.md"), "a note\n")
	writeFile(t, filepath.Join(data, "notes", "personal", "20261017-aaaaaa.md.123.tmp"), "half")
	writeFile(t, filepath.Join(data, "mining", "cache", "checkpoint-runs.jsonl"), "{}\n")
	target := filepath.Join(home, "backups", "2026-10-17")

	output, code := runCommand(t, "backup", target)
	require.Equal(t, 0, code, output)

	for _, path := range []string{
		filepath.Join("forgotten", "personal", "20261017-bbbbbb.md"),
		filepath.Join("mining", "cache", "checkpoint-runs.jsonl"),
		filepath.Join("notes", "personal", "20261017-aaaaaa.md"),
		"owner-map.json",
	} {
		want, err := os.ReadFile(filepath.Join(data, path))
		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join(target, path))
		require.NoError(t, err, path)
		require.Equal(t, string(want), string(got), path)
	}
	archived, err := filepath.Glob(filepath.Join(target, "archive", "transcripts", "*", "*", "*"))
	require.NoError(t, err)
	require.NotEmpty(t, archived)
	for _, path := range []string{
		filepath.Join("archive", "transcripts", ".lock"),
		"last-sync",
		filepath.Join("notes", "personal", "20261017-aaaaaa.md.123.tmp"),
		"sync.log",
	} {
		require.NoFileExists(t, filepath.Join(target, path))
	}

	db, err := store.OpenReadOnly(filepath.Join(target, "ele.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	var entries int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM entries`).Scan(&entries))
	require.Equal(t, 3, entries)
}

func TestRunBackupKeepsClosedPlansSoARestoreResolvesTheirEdges(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	writeFile(t, filepath.Join(gateway, "build.md"), "---\nkind: plan\n---\n## Actions\n- [ ] commands: Commands\n")
	writeFile(t, filepath.Join(gateway, "other.md"), "---\nkind: plan\n---\n## Actions\n- [ ] wire: Wire\n  blocked: build/commands\n")
	t.Chdir(gateway)
	for _, path := range []string{"build.md", "other.md"} {
		output, code := runCommand(t, "plan", "start", path)
		require.Equal(t, 0, code, output)
	}
	output, code := runCommand(t, "plan", "done", "build.md")
	require.Equal(t, 0, code, output)
	require.NoError(t, os.Remove(filepath.Join(gateway, "build.md")))
	target := filepath.Join(home, "backup")
	output, code = runCommand(t, "backup", target)
	require.Equal(t, 0, code, output)

	t.Setenv("ELE_HOME", target)
	output, code = runCommand(t, "plan", "show", "other.md")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "  blocked: build/commands\n  flag: blocker build/commands is in a closed plan and was never done\n")
}

func TestRunBackupLeavesOutImportOnceItIsGone(t *testing.T) {
	home := recallFixture(t)
	target := filepath.Join(home, "backup")

	output, code := runCommand(t, "backup", target)
	require.Equal(t, 0, code, output)
	require.NoDirExists(t, filepath.Join(target, "import"))
	require.FileExists(t, filepath.Join(target, "ele.db"))
}

func TestRunBackupRefusesATargetItCannotOwn(t *testing.T) {
	home := recallFixture(t)
	checkout := filepath.Join(home, "w", "personal", "notes")
	occupied := filepath.Join(home, "occupied")
	writeFile(t, filepath.Join(occupied, "kept"), "kept")
	tests := []struct {
		name   string
		target string
	}{
		{name: "inside a git checkout", target: filepath.Join(checkout, "backup")},
		{name: "inside the data directory", target: filepath.Join(home, "data", "backup")},
		{name: "not empty", target: occupied},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, code := runCommand(t, "backup", test.target)
			require.Equal(t, 1, code, output)
			require.NoFileExists(t, filepath.Join(test.target, "ele.db"))
		})
	}
	kept, err := os.ReadFile(filepath.Join(occupied, "kept"))
	require.NoError(t, err)
	require.Equal(t, "kept", string(kept))
}

func TestRunBackupWaitsForEachLock(t *testing.T) {
	tests := []struct {
		directory string
		name      string
	}{
		{directory: filepath.Join("archive", "transcripts"), name: "archive"},
		{directory: filepath.Join("mining", "cache"), name: "mining cache"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			home := recallFixture(t)
			target := filepath.Join(home, "backup")
			unlock, err := archive.Lock(filepath.Join(home, "data", test.directory))
			require.NoError(t, err)

			done := make(chan int, 1)
			go func() {
				var output bytes.Buffer
				done <- run(context.Background(), []string{"backup", target}, &output, &output)
			}()
			select {
			case <-done:
				t.Fatal("backup finished while another writer held the lock")
			case <-time.After(300 * time.Millisecond):
			}
			require.NoFileExists(t, filepath.Join(target, "ele.db"))
			unlock()
			require.Equal(t, 0, <-done)
			require.FileExists(t, filepath.Join(target, "ele.db"))
		})
	}
}
