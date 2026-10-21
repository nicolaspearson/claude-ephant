package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunForgetMovesANoteOutOfContextUntilRestored(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, output)
	note := readRemembered(t, output)

	output, code = runCommand(t, "forget", note.ID)
	require.Equal(t, 0, code, output)
	require.Equal(t, "Forgot note "+note.ID+".\n", output)
	require.FileExists(t, filepath.Join(home, "data", "forgotten", "corp", note.ID+".md"))
	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "Pin pydantic")
	output, code = runCommand(t, "recall", "--kind", "note", "pydantic")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, note.ID, "the sync recall runs drops the forgotten note")

	output, code = runCommand(t, "forget", "--restore", note.ID)
	require.Equal(t, 0, code, output)
	require.Equal(t, "Restored note "+note.ID+".\n", output)
	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Pin pydantic below 2.14.")
}

func TestRunForgetMovesNothingWhenAnIDIsUnknown(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, output)
	note := readRemembered(t, output)

	output, code = runCommand(t, "forget", note.ID, "20261017-ffffff")
	require.Equal(t, 1, code, output)
	require.Equal(t, "ele: corp has no live note 20261017-ffffff; --owner <name> for another owner's\n", output)
	require.FileExists(t, filepath.Join(home, "data", "notes", "corp", note.ID+".md"))

	output, code = runCommand(t, "forget", "--restore", note.ID)
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "corp has no forgotten note "+note.ID)
}

func TestRunForgetMovesNothingWhenANoteIsInBothFolders(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	var ids []string
	for _, words := range []string{"Pin pydantic below 2.14.", "Keep the wrapper."} {
		output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", words)
		require.Equal(t, 0, code, output)
		ids = append(ids, readRemembered(t, output).ID)
	}
	output, code := runCommand(t, "forget", ids[0], ids[1])
	require.Equal(t, 0, code, output)
	writeFile(t, filepath.Join(home, "data", "notes", "corp", ids[1]+".md"), "copied back by hand\n")

	output, code = runCommand(t, "forget", "--restore", ids[0], ids[1])
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "note "+ids[1]+" is in both")
	require.FileExists(t, filepath.Join(home, "data", "forgotten", "corp", ids[0]+".md"), "the note without a clash stays forgotten too")
}

func TestRunForgetOfACorrectionRevivesWhatItSuperseded(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, output)
	original := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "decision", "--session", "", "--supersedes", original.ID, "Pin pydantic below 2.15.")
	require.Equal(t, 0, code, output)
	correction := readRemembered(t, output)

	output, code = runCommand(t, "forget", original.ID)
	require.Equal(t, 1, code, output)
	require.Equal(t, "ele: note "+correction.ID+" supersedes "+original.ID+"; forget "+correction.ID+" first, or leave both\n", output)

	output, code = runCommand(t, "forget", correction.ID)
	require.Equal(t, 0, code, output)
	require.Equal(t, "Forgot note "+correction.ID+".\nNote "+original.ID+", which it superseded, is live again.\n", output)
	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Pin pydantic below 2.14.")
}

func TestRunForgetRestoreRefusesANoteALiveNoteAlreadySays(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, output)
	forgotten := readRemembered(t, output)
	output, code = runCommand(t, "forget", forgotten.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, output)
	live := readRemembered(t, output)

	output, code = runCommand(t, "forget", "--restore", forgotten.ID)
	require.Equal(t, 1, code, output)
	require.Equal(t, "ele: note "+live.ID+" already says what "+forgotten.ID+" says; nothing restored\n", output)
	require.FileExists(t, filepath.Join(home, "data", "forgotten", "corp", forgotten.ID+".md"))

	output, code = runCommand(t, "forget", live.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "forget", "--restore", forgotten.ID, live.ID)
	require.Equal(t, 1, code, output)
	require.Equal(t, "ele: note "+forgotten.ID+" already says what "+live.ID+" says; nothing restored\n", output, "two forgotten notes saying the same are not both restored")
	require.FileExists(t, filepath.Join(home, "data", "forgotten", "corp", forgotten.ID+".md"))
}

func TestRunForgetRestoresAClosedHandoffBesideAPendingOne(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "handoff", "--session", "", "Continue finish-build.")
	require.Equal(t, 0, code, output)
	closed := readRemembered(t, output)
	output, code = runCommand(t, "handoff", "close", closed.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "forget", closed.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "remember", "--kind", "handoff", "--session", "", "Continue finish-build.")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "forget", "--restore", closed.ID)
	require.Equal(t, 0, code, "a closed handoff never shows, so it repeats nothing: %s", output)
	require.FileExists(t, filepath.Join(home, "data", "notes", "corp", closed.ID+".md"))
}

func TestRunForgetRestoresASupersededNoteBesideOneSayingTheSame(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, output)
	original := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "decision", "--session", "", "--supersedes", original.ID, "Pin pydantic below 2.15.")
	require.Equal(t, 0, code, output)
	correction := readRemembered(t, output)
	output, code = runCommand(t, "forget", correction.ID, original.ID)
	require.Equal(t, 1, code, "the correction is checked before it moves, so the original is still superseded: %s", output)
	for _, identifier := range []string{correction.ID, original.ID} {
		output, code = runCommand(t, "forget", identifier)
		require.Equal(t, 0, code, output)
	}
	output, code = runCommand(t, "forget", "--restore", correction.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "forget", "--restore", original.ID)
	require.Equal(t, 0, code, "a superseded note shows nowhere, so it repeats nothing: %s", output)
	require.FileExists(t, filepath.Join(home, "data", "notes", "corp", original.ID+".md"))
}

func TestRunForgetSaysLiveAgainOnlyOnceNothingElseSupersedes(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, output)
	original := readRemembered(t, output)
	var corrections []string
	for _, words := range []string{"Pin pydantic below 2.15.", "Pin pydantic below 2.16."} {
		output, code = runCommand(t, "remember", "--kind", "decision", "--session", "", "--supersedes", original.ID, words)
		require.Equal(t, 0, code, output)
		corrections = append(corrections, readRemembered(t, output).ID)
	}

	output, code = runCommand(t, "forget", corrections[0])
	require.Equal(t, 0, code, output)
	require.Equal(t, "Forgot note "+corrections[0]+".\n", output, "the second correction still supersedes the original")

	output, code = runCommand(t, "forget", corrections[1])
	require.Equal(t, 0, code, output)
	require.Equal(t, "Forgot note "+corrections[1]+".\nNote "+original.ID+", which it superseded, is live again.\n", output)

	output, code = runCommand(t, "forget", "--restore", corrections[0])
	require.Equal(t, 0, code, output)
	require.Equal(t, "Restored note "+corrections[0]+".\nNote "+original.ID+", which it supersedes, is superseded again.\n", output)

	output, code = runCommand(t, "forget", "--restore", corrections[1])
	require.Equal(t, 0, code, output)
	require.Equal(t, "Restored note "+corrections[1]+".\n", output, "the original was superseded already")
}
