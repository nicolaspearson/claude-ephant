package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

var remembered = regexp.MustCompile(`^Remembered (\d{8}-[0-9a-f]{6}): (.+)\n$`)

func TestRunRememberFilesANoteRecallFinds(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude-corp"))
	t.Setenv("CLAUDE_CODE_SESSION_ID", "s-1")

	output, code := runCommand(t, "remember", "--kind", "question", "Should", "ele", "index", "tool", "output?")
	require.Equal(t, 0, code, output)
	question := readRemembered(t, output)
	written, err := time.Parse(time.RFC3339, question.Time)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), written, time.Minute)
	require.Equal(t, notes.Note{Body: "Should ele index tool output?", Date: time.Now().Format(time.DateOnly), ID: question.ID, Kind: notes.KindQuestion, Owner: "corp", Profile: "corp", Project: "gateway", Scope: notes.ScopeProject, Session: "s-1", Time: question.Time}, question)

	output, code = runCommand(t, "remember", "--kind", "question", "--supersedes", "", "Should", "ele", "index", "subagents?")
	require.Equal(t, 0, code, output)
	related := readRemembered(t, output)
	require.Nil(t, related.Supersedes, "an empty --supersedes names none")

	output, code = runCommand(t, "remember", "--kind", "decision", "--supersedes", question.ID, "--supersedes", question.ID, "Twice.")
	require.Equal(t, 2, code, output)
	require.Equal(t, "ele: --supersedes "+question.ID+" given twice\n", output)

	output, code = runCommand(t, "remember", "--kind", "decision", "--global", "--supersedes", question.ID, "--supersedes", related.ID, "No: tool output would triple the index.")
	require.Equal(t, 0, code, output)
	decision := readRemembered(t, output)
	require.Equal(t, notes.ScopeGlobal, decision.Scope)
	require.Equal(t, []string{question.ID, related.ID}, decision.Supersedes)

	output, code = runCommand(t, "recall", "--kind", "note", "index")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "note decision          "+decision.ID+" No: tool output would triple the «index».")
	require.Contains(t, output, "note question          (superseded by "+decision.ID+") "+question.ID+" Should ele «index» tool output?")
	require.Contains(t, output, "note question          (superseded by "+decision.ID+") "+related.ID+" Should ele «index» subagents?")
}

func TestRunRememberFilesUnderANamedOwnerOutsideTheMap(t *testing.T) {
	home := recallFixture(t)
	elsewhere := filepath.Join(home, "elsewhere")
	require.NoError(t, os.MkdirAll(elsewhere, 0o700))
	t.Chdir(elsewhere)

	output, code := runCommand(t, "remember", "--kind", "fact", "No owner here.")
	require.Equal(t, 1, code)
	require.Contains(t, output, "no owner in owner-map.json covers "+elsewhere)

	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	output, code = runCommand(t, "remember", "--kind", "fact", "--owner", "personal", "Filed by hand.")
	require.Equal(t, 0, code, output)
	note := readRemembered(t, output)
	require.Equal(t, "personal", note.Owner)
	require.Equal(t, "elsewhere", note.Project)
	require.Empty(t, note.Session)

	t.Setenv("CLAUDE_CODE_SESSION_ID", "from-environment")
	output, code = runCommand(t, "remember", "--kind", "fact", "--owner", "personal", "--session", "010a2385-feef", "Filed with its session.")
	require.Equal(t, 0, code, output)
	require.Equal(t, "010a2385-feef", readRemembered(t, output).Session)

	output, code = runCommand(t, "remember", "--kind", "fact", "--owner", "personal", "--session", "", "Filed with no session.")
	require.Equal(t, 0, code, output)
	require.Empty(t, readRemembered(t, output).Session)

	output, code = runCommand(t, "remember", "--kind", "fact", "--owner", "personal", "--session", "a\nkind: decision", "Injected.")
	require.Equal(t, 2, code, output)
	require.Contains(t, output, `session "a\nkind: decision" holds characters other than letters, digits, - and _`)
}

func TestRunRememberFilesUnderANamedProject(t *testing.T) {
	home := recallFixture(t)
	for _, owner := range []string{"corp", "personal"} {
		repository := filepath.Join(home, "w", owner, "shared")
		require.NoError(t, os.MkdirAll(filepath.Join(repository, ".git"), 0o700))
		writeFile(t, filepath.Join(home, ".claude", "projects", "-shared-"+owner, "s-"+owner+".jsonl"),
			`{"type":"user","cwd":"`+repository+`","sessionId":"s-`+owner+`","timestamp":"2026-10-15T09:00:00Z","uuid":"u-`+owner+`","message":{"content":"shared work"}}`+"\n")
	}
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	elsewhere := filepath.Join(home, "elsewhere")
	require.NoError(t, os.MkdirAll(elsewhere, 0o700))
	t.Chdir(elsewhere)

	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "gateway", "Owner from the index.")
	require.Equal(t, 0, code, output)
	note := readRemembered(t, output)
	require.Equal(t, "corp", note.Owner)
	require.Equal(t, "gateway", note.Project)

	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "shared", "Two owners.")
	require.Equal(t, 1, code)
	require.Contains(t, output, `project "shared" has entries under owners corp, personal; pass --owner <name>`)

	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "shared", "--owner", "personal", "Owner named.")
	require.Equal(t, 0, code, output)
	require.Equal(t, "personal", readRemembered(t, output).Owner)

	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "nowhere", "Unknown.")
	require.Equal(t, 1, code)
	require.Contains(t, output, `project "nowhere" has no listed owner in owner-map.json or the index; pass --owner <name>`)

	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "My Repo", "A space is a valid project name.", "--owner", "personal")
	require.Equal(t, 0, code, output)
	require.Equal(t, "My Repo", readRemembered(t, output).Project)

	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "a\nb", "Bad name.")
	require.Equal(t, 2, code)
	require.Contains(t, output, `project "a\nb" has surrounding spaces or a control character`)

	output, code = runCommand(t, "remember", "--kind", "fact", "--project", " ", "--owner", "corp", "Blank name.")
	require.Equal(t, 2, code)
	require.Contains(t, output, `project " " has surrounding spaces or a control character`)

	// The gateway entry names an unlisted owner, so the index decides, but only once it is synced under this map.
	paths := `"paths": {"` + filepath.Join(home, "w", "corp") + `": "corp", "` + filepath.Join(home, "w", "personal") + `": "personal"}`
	writeFile(t, filepath.Join(home, "data", "owner-map.json"), `{"owners": ["corp", "personal"], `+paths+`, "projects": {"gateway": "copr", "nowhere": "personal"}}`)
	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "gateway", "Stale index.")
	require.Equal(t, 1, code)
	require.Contains(t, output, `the index was not synced under the current owner-map.json, so it cannot name project "gateway"'s owner; run ele sync, or pass --owner <name>`)

	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "nowhere", "Owner from the map entry.")
	require.Equal(t, 0, code, output)
	require.Equal(t, "personal", readRemembered(t, output).Owner)

	writeFile(t, filepath.Join(home, "data", "notes", "old", "20261001-aaaaaa.md"), "---\ndate: 2026-10-01\nkind: fact\nowner: old\nproject: legacy\nscope: project\n---\nA note under an owner the map no longer lists.\n")
	output, code = runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "gateway", "Unlisted entry falls to the index.")
	require.Equal(t, 0, code, output)
	require.Equal(t, "corp", readRemembered(t, output).Owner)

	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "legacy", "Only an unlisted owner's note.")
	require.Equal(t, 1, code)
	require.Contains(t, output, `project "legacy" has no listed owner in owner-map.json or the index; pass --owner <name>`)
}

func TestRunRememberHandsOffAgainOnceTheSameHandoffIsClosed(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "handoff", "--session", "", "Continue finish-build.")
	require.Equal(t, 0, code, output)
	first := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "handoff", "--session", "", "Continue finish-build.")
	require.Equal(t, 1, code, "a pending handoff saying the same is a duplicate: %s", output)

	output, code = runCommand(t, "handoff", "close", first.ID)
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "remember", "--kind", "handoff", "--session", "", "Continue finish-build.")
	require.Equal(t, 0, code, output)
}

func TestRunRememberRefusesALiveNoteSayingTheSame(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, output)
	first := readRemembered(t, output)

	output, code = runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin  pydantic\nbelow 2.14.")
	require.Equal(t, 1, code, output)
	require.Equal(t, "ele: note "+first.ID+" already says this; nothing written\n", output)

	for _, args := range [][]string{
		{"--kind", "fact"},
		{"--kind", "decision", "--global"},
		{"--kind", "decision", "--project", "other", "--owner", "corp"},
	} {
		output, code = runCommand(t, append(append([]string{"remember", "--session", ""}, args...), "Pin pydantic below 2.14.")...)
		require.Equal(t, 0, code, "another kind, scope or project is not a duplicate: %v\n%s", args, output)
	}

	output, code = runCommand(t, "remember", "--kind", "decision", "--session", "", "--supersedes", first.ID, "Pin pydantic below 2.15.")
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "remember", "--kind", "decision", "--session", "", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, "a superseded note is not live: %s", output)
}

func TestRunRememberRefusesBadInput(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	tests := []struct {
		args    []string
		code    int
		message string
		name    string
	}{
		{args: []string{"--kind", "decision", "one\ntwo\nthree\nfour"}, code: 2, message: "a note is at most 3 lines, not 4", name: "four lines"},
		{args: []string{"--kind", "decision", "--everywhere", "--global", "x"}, code: 2, message: "--everywhere spans every owner and project; drop --global, --owner and --project", name: "everywhere and global"},
		{args: []string{"--kind", "decision", "--everywhere", "--owner", "corp", "x"}, code: 2, message: "--everywhere spans every owner and project", name: "everywhere and owner"},
		{args: []string{"--kind", "decision", "--everywhere", "--project", "gateway", "x"}, code: 2, message: "--everywhere spans every owner and project", name: "everywhere and project"},
		{args: []string{"--kind", "handoff", "--everywhere", "x"}, code: 2, message: "a handoff is scoped to its project; drop --everywhere", name: "everywhere handoff"},
		{args: []string{"--kind", "handoff", "--global", "x"}, code: 2, message: "a handoff is scoped to its project; drop --global", name: "global handoff"},
		{args: []string{"x"}, code: 2, message: "ele remember --kind", name: "no kind"},
		{args: []string{"--kind", "decision"}, code: 2, message: "ele remember --kind", name: "no words"},
		{args: []string{"--kind", "plan", "plans.md"}, code: 2, message: "a plan note is written by ele plan start <path>, not ele remember", name: "plan"},
		{args: []string{"--kind", "todo", "x"}, code: 2, message: `unknown kind "todo"; use decision, fact, handoff, question`, name: "unknown kind"},
		{args: []string{"--kind", "decision", "--supersedes", "20200101-000000", "x"}, code: 2, message: "corp has no note 20200101-000000 to supersede", name: "unknown supersedes"},
		{args: []string{"--kind", "decision", "--owner", "team", "x"}, code: 2, message: `unknown owner "team"; owner-map.json lists corp, personal`, name: "unlisted owner"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, code := runCommand(t, append([]string{"remember"}, test.args...)...)
			require.Equal(t, test.code, code, output)
			require.Contains(t, output, test.message)
		})
	}
	entries, err := os.ReadDir(filepath.Join(home, "data", "notes", "corp"))
	if !os.IsNotExist(err) {
		require.NoError(t, err)
	}
	require.Empty(t, entries)
}

func TestRunRememberWithoutAnIndexCreatesNone(t *testing.T) {
	dataDirectory := filepath.Join(t.TempDir(), "data")
	t.Setenv("ELE_HOME", dataDirectory)
	writeFile(t, filepath.Join(dataDirectory, "owner-map.json"), `{"owners": ["corp"]}`)

	output, code := runCommand(t, "remember", "--kind", "fact", "--project", "gateway", "No index yet.")
	require.Equal(t, 1, code)
	require.Contains(t, output, `project "gateway" has no listed owner in owner-map.json or the index; pass --owner <name>`)
	require.NoFileExists(t, filepath.Join(dataDirectory, "ele.db"))
}

func readRemembered(t *testing.T, output string) notes.Note {
	t.Helper()
	match := remembered.FindStringSubmatch(output)
	require.NotNil(t, match, output)
	content, err := os.ReadFile(match[2])
	require.NoError(t, err)
	note, err := notes.Parse(match[2], content)
	require.NoError(t, err)
	require.Equal(t, match[1], note.ID)
	require.True(t, strings.HasSuffix(match[2], filepath.Join(note.Owner, note.ID+".md")))
	return note
}
