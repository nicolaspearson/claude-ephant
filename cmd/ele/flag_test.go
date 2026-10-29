package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

var flaggedLine = regexp.MustCompile(`^Flagged session \S+ (?:bad|good), note (\d{8}-[0-9a-f]{6}): (.+)\n$`)

func TestRunFlagSavesAFlagNoteListedAndKeptOutOfTheStartBlock(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	t.Setenv("CLAUDE_CODE_SESSION_ID", "running-1")

	output, code := runCommand(t, "flag", "bad", "Pushed", "past", "a", "failing", "gate.")
	require.Equal(t, 0, code, output)
	bad := readFlagged(t, output)
	written, err := time.Parse(time.RFC3339, bad.Time)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), written, time.Minute)
	require.Equal(t, notes.Note{Body: "Pushed past a failing gate.", Date: time.Now().Format(time.DateOnly), ID: bad.ID, Kind: notes.KindFlag, Owner: "corp", Profile: "claude", Project: "gateway", Scope: notes.ScopeProject, Session: "running-1", Time: bad.Time, Verdict: notes.VerdictBad}, bad)

	output, code = runCommand(t, "flag", "good", "--session", "abc", "Kept the diff small.")
	require.Equal(t, 0, code, output)
	good := readFlagged(t, output)
	require.Equal(t, "abc1", good.Session, "a prefix resolves to the indexed session")
	require.Equal(t, "2026-10-15T10:01:00Z", good.Time, "a flag given afterwards marks the session's last entry, not when it was given")
	require.Equal(t, notes.VerdictGood, good.Verdict)

	output, code = runCommand(t, "flag", "list")
	require.Equal(t, 0, code, output)
	require.Regexp(t, `^Flags for corp · gateway, newest first:\n`, output)
	require.Contains(t, output, "- bad   "+bad.Time+"  session running-  note "+bad.ID+": Pushed past a failing gate.\n")
	require.Contains(t, output, "- good  2026-10-15T10:01:00Z  session abc1  note "+good.ID+": Kept the diff small.\n")
	require.Regexp(t, `ele show <session> --at <time> opens the session at a flag\.\n$`, output)
	output, code = runCommand(t, "flag", "list", "--owner", "corp", "--project", "proxy")
	require.Equal(t, 0, code, output)
	require.Equal(t, "ele has no flags for corp · proxy.\n", output)

	output, code = runCommand(t, "session-start", "--no-sync")
	require.Equal(t, 0, code, output)
	shown, _ := hookOutput(t, output)
	require.NotContains(t, shown, bad.ID)
	require.NotContains(t, shown, good.ID)
	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, bad.ID)
	require.NotContains(t, output, good.ID)

	output, code = runCommand(t, "remember", "--kind", "decision", "--supersedes", good.ID, "Small diffs are the rule, not a flag.")
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "flag", "list")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, good.ID, "a superseded flag leaves the list")
	require.Contains(t, output, bad.ID)
}

func TestRunFlagAtMarksTheIndexedEntryAtOrAfterTheTime(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	t.Setenv("CLAUDE_CODE_SESSION_ID", "running-1")
	flagAt := func(at string) (string, int) {
		return runCommand(t, "flag", "bad", "--session", "abc1", "--at", at, "Pinned without asking.")
	}

	for at, want := range map[string]string{
		"2026-10-15T10:00:00Z":      "2026-10-15T10:00:00Z",
		"2026-10-15T10:00:30Z":      "2026-10-15T10:01:00Z",
		"2026-10-15T14:00:00+04:00": "2026-10-15T10:00:00Z",
	} {
		output, code := flagAt(at)
		require.Equal(t, 0, code, output)
		require.Equal(t, want, readFlagged(t, output).Time, "--at %s", at)
	}
	for _, at := range []string{"2026-10-15T09:59:59Z", "2026-10-15T10:01:01Z"} {
		output, code := flagAt(at)
		require.Equal(t, 1, code, output)
		require.Equal(t, "ele: --at "+at+" is outside session abc1, which runs from 2026-10-15T10:00:00Z to 2026-10-15T10:01:00Z\n", output)
	}
	output, code := flagAt("yesterday")
	require.Equal(t, 2, code, output)
	require.Equal(t, `ele: --at "yesterday" is not an RFC 3339 time, such as 2026-10-29T03:11:39Z`+"\n", output)
	output, code = runCommand(t, "flag", "bad", "--at", "2026-10-15T10:00:00Z", "No session named.")
	require.Equal(t, 2, code, output)
	require.Equal(t, "ele: --at names a moment of an earlier session; pass --session <id> too\n", output)
}

func TestRunForgetRestoresAFlagSayingWhatAnotherDoes(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	t.Setenv("CLAUDE_CODE_SESSION_ID", "running-1")
	flagged := func() notes.Note {
		output, code := runCommand(t, "flag", "bad", "Skipped the gate.")
		require.Equal(t, 0, code, output)
		return readFlagged(t, output)
	}
	first := flagged()
	flagged()

	output, code := runCommand(t, "forget", first.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "forget", "--restore", first.ID)
	require.Equal(t, 0, code, "each flag marks its own moment, so the same words are no duplicate: %s", output)
}

func TestRunFlagRefusesBadInput(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")

	tests := []struct {
		args    []string
		code    int
		message string
		name    string
	}{
		{args: []string{"remember", "--kind", "flag", "Sneaked in."}, code: 2, message: `ele: a flag note is written by ele flag good|bad "<why>", not ele remember` + "\n", name: "flag through remember"},
		{args: []string{"flag", "bad", "Outside a session."}, code: 2, message: "ele: no running session to flag; pass --session <id>\n", name: "no session"},
		{args: []string{"flag", "good", "--session", "abc1"}, code: 2, message: "ele flag good ", name: "no why"},
		{args: []string{"flag", "good", "--session", "abc1", "one\ntwo\nthree\nfour"}, code: 2, message: "ele: a flag is at most 3 lines, not 4", name: "too long"},
		{args: []string{"flag", "bad", "--session", "zzz", "Typo."}, code: 1, message: `ele: no indexed session starts with "zzz"; ele sync indexes a recent one` + "\n", name: "unknown session"},
		{args: []string{"flag", "meh", "Unsure."}, code: 2, message: `ele: unknown flag subcommand "meh"`, name: "unknown verdict"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, code := runCommand(t, test.args...)
			require.Equal(t, test.code, code, output)
			require.Contains(t, output, test.message)
		})
	}
	entries, err := os.ReadDir(filepath.Join(home, "data", "notes", "corp"))
	if err == nil {
		require.Empty(t, entries, "a refused flag writes nothing")
	}
}

func TestRunFlagRefusesASharedSessionPrefix(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "abc2.jsonl"),
		`{"type":"user","cwd":"`+filepath.Join(home, "w", "corp", "gateway")+`","sessionId":"abc2","timestamp":"2026-10-15T11:00:00Z","uuid":"u-abc2","message":{"content":"second"}}`+"\n")
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "flag", "bad", "--session", "abc", "Which one?")
	require.Equal(t, 1, code, output)
	require.Equal(t, `ele: more than one indexed session starts with "abc"; give more of its id`+"\n", output)
}

func readFlagged(t *testing.T, output string) notes.Note {
	t.Helper()
	match := flaggedLine.FindStringSubmatch(output)
	require.NotNil(t, match, output)
	content, err := os.ReadFile(match[2])
	require.NoError(t, err)
	note, err := notes.Parse(match[2], content)
	require.NoError(t, err)
	require.Equal(t, match[1], note.ID)
	return note
}
