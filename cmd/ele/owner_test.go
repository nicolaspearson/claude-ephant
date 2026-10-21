package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

func TestOwnerAddListsTheOwnerSorted(t *testing.T) {
	home := recallFixture(t)

	output, code := runCommand(t, "owner", "add", "acme", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Added owner acme.\nNo indexed project changes owner.\nThe next sync rebuilds the index under the new map.\n", output)

	loaded, err := ownermap.Load(filepath.Join(home, "data", "owner-map.json"))
	require.NoError(t, err)
	require.Equal(t, []string{"acme", "corp", "personal"}, loaded.Owners)
}

func TestOwnerAddMovesTheNotesOfAProjectItsListingReowns(t *testing.T) {
	home := recallFixture(t)
	writeFile(t, filepath.Join(home, "data", "owner-map.json"),
		`{"owners": ["corp", "personal"], "paths": {"`+filepath.Join(home, "w", "corp")+`": "corp"}, "projects": {"gateway": "team"}}`)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "pin pydantic below 2.14")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "owner", "add", "team", "--no-sync")
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "this change strands 1 note of gateway under corp, which would move to team")

	output, code = runCommand(t, "owner", "add", "team", "--move", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Projects changing owner:\n  gateway: corp → team\nMoved 1 note of gateway from corp to team.\n")
}

func TestOwnerAddPathReownsTheProjectsBelowIt(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")

	output, code := runCommand(t, "recall", "--owner", "personal", "--no-sync", "pydantic")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "abc1")

	output, code = runCommand(t, "owner", "add-path", gateway, "personal", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Mapped path "+gateway+" to personal.\nProjects changing owner:\n  gateway: corp → personal\nThe next sync rebuilds the index under the new map.\n", output)

	output, code = runCommand(t, "recall", "--owner", "personal", "pydantic")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "abc1", "the sync after the write files gateway's session under personal")
}

func TestOwnerAddProjectReplacesAnEntryAndSaysWhatItWas(t *testing.T) {
	home := recallFixture(t)

	output, code := runCommand(t, "owner", "add-project", "gateway", "personal", "--no-sync")
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "owner", "add-project", "gateway", "corp", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Mapped project gateway to corp (was personal).\nProjects changing owner:\n  gateway: personal → corp\n")

	output, code = runCommand(t, "owner", "add-project", "gateway", "corp", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Equal(t, "owner-map.json already says that; nothing written.\n", output)

	loaded, err := ownermap.Load(filepath.Join(home, "data", "owner-map.json"))
	require.NoError(t, err)
	require.Equal(t, map[string]string{"gateway": "corp"}, loaded.Projects)
}

func TestOwnerMovesAProjectsForgottenNotesWithIt(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "pin pydantic below 2.14")
	require.Equal(t, 0, code, output)
	note := readRemembered(t, output)
	output, code = runCommand(t, "forget", note.ID)
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "owner", "add-project", "gateway", "personal", "--no-sync")
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "this change strands 1 forgotten note of gateway under corp, which would move to personal")

	output, code = runCommand(t, "owner", "add-project", "gateway", "personal", "--move", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Moved 1 forgotten note of gateway from corp to personal.\n")
	output, code = runCommand(t, "forget", "--restore", note.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "pin pydantic below 2.14", "restored under personal, which now owns gateway")
}

func TestOwnerMovesAProjectsNotesOnlyWithMove(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	t.Chdir(gateway)
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "pin pydantic below 2.14")
	require.Equal(t, 0, code, output)
	corpNotes, err := filepath.Glob(filepath.Join(home, "data", "notes", "corp", "*.md"))
	require.NoError(t, err)
	require.Len(t, corpNotes, 1)
	mapPath := filepath.Join(home, "data", "owner-map.json")
	original, err := os.ReadFile(mapPath)
	require.NoError(t, err)

	output, code = runCommand(t, "owner", "add-project", "gateway", "personal", "--no-sync")
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "this change strands 1 note of gateway under corp, which would move to personal; --move refiles them, or nothing is written")
	unchanged, err := os.ReadFile(mapPath)
	require.NoError(t, err)
	require.Equal(t, string(original), string(unchanged))
	require.FileExists(t, corpNotes[0])

	output, code = runCommand(t, "owner", "add-project", "gateway", "personal", "--move", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Moved 1 note of gateway from corp to personal.\n")
	require.NoFileExists(t, corpNotes[0])

	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "pin pydantic below 2.14", "the moved note is read under personal, which now owns gateway")
}

func TestOwnerMovesNothingWhenANoteIsFiledUnderBothOwners(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	for _, words := range []string{"pin pydantic below 2.14", "keep the wrapper"} {
		output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", words)
		require.Equal(t, 0, code, output)
	}
	corpNotes, err := filepath.Glob(filepath.Join(home, "data", "notes", "corp", "*.md"))
	require.NoError(t, err)
	require.Len(t, corpNotes, 2)
	writeFile(t, filepath.Join(home, "data", "notes", "personal", filepath.Base(corpNotes[1])), "copied by hand\n")

	output, code := runCommand(t, "owner", "add-project", "gateway", "personal", "--move", "--no-sync")
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "is filed under both corp and personal; nothing moved or written")
	require.FileExists(t, corpNotes[0], "the note without a clash stays too")
	require.FileExists(t, corpNotes[1])
}

func TestOwnerRefilesNotesNoOwnerWouldCoverOnlyWithMoveTo(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "pin pydantic below 2.14")
	require.Equal(t, 0, code, output)
	corp := filepath.Join(home, "w", "corp")
	mapPath := filepath.Join(home, "data", "owner-map.json")
	original, err := os.ReadFile(mapPath)
	require.NoError(t, err)

	refusals := []struct {
		args    []string
		message string
	}{
		{args: nil, message: "no single owner would cover the project of 1 note of gateway under corp after this change; --move-to <owner> refiles them"},
		{args: []string{"--move"}, message: "--move-to <owner> refiles them"},
		{args: []string{"--move-to", "team"}, message: `unknown owner "team"`},
	}
	for _, refusal := range refusals {
		output, code = runCommand(t, append([]string{"owner", "remove-path", corp, "--no-sync"}, refusal.args...)...)
		require.Equal(t, 1, code, output)
		require.Contains(t, output, refusal.message)
	}
	unchanged, err := os.ReadFile(mapPath)
	require.NoError(t, err)
	require.Equal(t, string(original), string(unchanged))

	output, code = runCommand(t, "owner", "remove-path", corp, "--move-to", "personal", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Projects changing owner:\n  gateway: corp → (no owner)\nMoved 1 note of gateway from corp to personal.\n")
	output, code = runCommand(t, "context", "--owner", "personal", "--project", "gateway")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "pin pydantic below 2.14")
}

func TestOwnerRefusesMoveToWhenNoNoteIsStranded(t *testing.T) {
	home := recallFixture(t)

	output, code := runCommand(t, "owner", "add-project", "gateway", "personal", "--move-to", "personal", "--no-sync")
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "--move-to is for notes no single owner would cover after this change, and none would be")
	loaded, err := ownermap.Load(filepath.Join(home, "data", "owner-map.json"))
	require.NoError(t, err)
	require.Empty(t, loaded.Projects)
}

func TestOwnerRefusesWhileASyncHoldsTheLock(t *testing.T) {
	home := recallFixture(t)
	unlock, held, err := archive.TryLock(filepath.Join(home, "data", "archive", "transcripts"))
	require.NoError(t, err)
	require.True(t, held)
	defer unlock()

	output, code := runCommand(t, "owner", "add-path", "/w", "corp", "--no-sync")
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "a sync holds the archive lock")
}

func TestOwnerRefusesWhileTheIndexIsIncomplete(t *testing.T) {
	tests := []struct {
		message   string
		name      string
		statement string
	}{
		{message: "run ele sync to rebuild it", name: "index predates the directories table", statement: `DROP TABLE directories`},
		{message: "no sync has completed under this version of ele", name: "no sync completed", statement: `DELETE FROM settings WHERE key = 'owner-map'`},
		{message: "no sync has completed under this version of ele", name: "older index version", statement: `UPDATE settings SET value = 'before-` + index.Version + `' WHERE key = 'index-version'`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			home := recallFixture(t)
			db, err := sql.Open("sqlite", filepath.Join(home, "data", "ele.db"))
			require.NoError(t, err)
			_, err = db.Exec(test.statement)
			require.NoError(t, err)
			require.NoError(t, db.Close())

			output, code := runCommand(t, "owner", "add-path", "/w", "corp", "--no-sync")
			require.Equal(t, 1, code, output)
			require.Contains(t, output, "the index cannot say which projects change owner")
			require.Contains(t, output, test.message)
		})
	}
}

func TestOwnerRefusesWithoutWriting(t *testing.T) {
	home := recallFixture(t)
	mapPath := filepath.Join(home, "data", "owner-map.json")
	original, err := os.ReadFile(mapPath)
	require.NoError(t, err)

	tests := []struct {
		args    []string
		code    int
		message string
		name    string
	}{
		{args: []string{"owner", "remove-path", "/nowhere"}, code: 1, message: `no path rule for "/nowhere"`, name: "absent path rule"},
		{args: []string{"owner", "remove-project", "gateway"}, code: 1, message: `no project entry for "gateway"`, name: "absent project entry"},
		{args: []string{"owner", "add", "corp"}, code: 1, message: `already lists owner "corp"`, name: "listed owner"},
		{args: []string{"owner", "add-path", "/w"}, code: 2, message: "ele owner add-path [--move]", name: "missing owner"},
		{args: []string{"owner", "add", "../x"}, code: 1, message: `owner "../x" must be letters`, name: "owner not a folder name"},
		{args: []string{"owner", "add-project", "a/b", "corp"}, code: 1, message: `project "a/b" must be a directory name`, name: "project with a separator"},
		{args: []string{"owner", "add-path", "w/corp", "corp"}, code: 1, message: `path "w/corp" is relative`, name: "relative path"},
		{args: []string{"owner", "rename", "a", "b"}, code: 2, message: "ele owner remove-project", name: "unknown subcommand"},
		{args: []string{"owner", "add-path", "/w", "team"}, code: 1, message: `unknown owner "team"; owner-map.json lists corp, personal, and ele owner add team adds it`, name: "unlisted owner"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, code := runCommand(t, append(test.args, "--no-sync")...)
			require.Equal(t, test.code, code, output)
			require.Contains(t, output, test.message)
			unchanged, err := os.ReadFile(mapPath)
			require.NoError(t, err)
			require.Equal(t, string(original), string(unchanged))
		})
	}
}
