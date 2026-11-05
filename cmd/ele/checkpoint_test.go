package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/checkpoint"
)

const (
	doneClaim   = "Ready to commit:\n```sh\ngit commit -m \"x\"\n```"
	editMain    = "EDIT:main.go"
	hookSession = "checkpoint-test-session"
	logRecord   = `{"change":{"files":1,"lines":2},"findings":[],"mode":"small","mutations":{"failed":0,"inconclusive":0,"run":0},"notes":{"approved":0,"proposed":0},"rereview":"skipped","scope":"uncommitted"}`
)

func TestRunCheckpointCheckNamesThePathCap(t *testing.T) {
	root := hookRepository(t)
	for index := range 201 {
		writeTestFile(t, root, fmt.Sprintf("a/%03d.txt", index), "x\n")
	}

	output, code := runCheckpoint(t, "check", root)

	require.Equal(t, 1, code)
	require.Contains(t, output, "has changed paths past the fingerprint's cap: 1")
}

func TestRunCheckpointDensityMeasuresTheWholeRepositoryFromASubdirectory(t *testing.T) {
	root, _ := checkpointRepository(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("x := 1\n"), 0o600))
	added, err := exec.Command("git", "-C", root, "add", "main.go").CombinedOutput()
	require.NoError(t, err, string(added))

	output, code := runCheckpointDensity(t, filepath.Join(root, "sub"))

	require.Equal(t, 0, code, output)
	require.True(t, strings.HasPrefix(output, "file "), output)
	require.Contains(t, output, "\nmain.go ")
}

func TestRunCheckpointDensityRefusesOutsideARepositoryAndAnUnknownRef(t *testing.T) {
	root, _ := checkpointRepository(t)
	t.Chdir(t.TempDir())

	output, code := runCheckpointDensity(t)
	require.Equal(t, 2, code)
	require.Equal(t, "ele: the working directory is not inside a git repository\n", output)

	output, code = runCheckpointDensity(t, "--ref", "nowhere", root)
	require.Equal(t, 1, code)
	require.True(t, strings.HasPrefix(output, "ele: git "), output)
	require.Contains(t, output, "unknown revision")
}

func TestRunCheckpointGateBlocksADoneClaimOverThisTurnsEditsWithNoCheckpoint(t *testing.T) {
	for name, test := range map[string]struct {
		block      bool
		transcript []string
	}{
		"bug-fix prose that only looks like labels":           {block: true, transcript: []string{"PROMPT:go", editMain, "Fixed the off-by-one bug — loop stops at len. Trimmed a stale comment — it named the old function. Ready to commit."}},
		"bugs and prose reviewer labels":                      {transcript: []string{"PROMPT:go", editMain, "**Bugs reviewer:** two fixed. **Prose reviewer:** one comment cut. Ready to commit."}},
		"changes left in the working tree":                    {block: true, transcript: []string{"PROMPT:go", editMain, "Ready for review; changes left in the working tree."}},
		"checkpoint skill invoked in an earlier turn":         {block: true, transcript: []string{"PROMPT:earlier", "SKILL:checkpoint", "**Quality — ran.** **Prose — ran.**", "PROMPT:now", editMain, "Changed one more thing. Ready to commit."}},
		"checkpoint skill invoked this turn":                  {transcript: []string{"PROMPT:go", editMain, "SKILL:checkpoint", "Done. Ready to commit."}},
		"claimed done early, ended mid-task":                  {transcript: []string{"PROMPT:go", editMain, "Ready to commit.", "SKILL:none", "Found one more case; fixing it next turn."}},
		"claimed done, nothing reported":                      {block: true, transcript: []string{"PROMPT:go", editMain, doneClaim}},
		"commit message naming what is left":                  {block: true, transcript: []string{"PROMPT:go", editMain, "Here:\n```sh\ngit commit -m \"parser: two steps are left\"\n```"}},
		"commit offered after a step":                         {transcript: []string{"PROMPT:go", editMain, "After your run passes you can commit:\n```sh\ngit commit -m \"x\"\n```"}},
		"commit offered for later":                            {transcript: []string{"PROMPT:go", editMain, "Once the suite passes, you can commit:\n```sh\ngit commit -m \"x\"\n```"}},
		"commit offered now, mid-item":                        {block: true, transcript: []string{"PROMPT:go", editMain, "Still waiting on your run. To commit the progress lines now:\n```sh\ngit commit -m \"x\"\n```"}},
		"commit offered when the review lands":                {transcript: []string{"PROMPT:go", editMain, "When the review lands, you can then commit:\n```sh\ngit commit -m \"x\"\n```"}},
		"diff hunk before the done claim":                     {block: true, transcript: []string{"PROMPT:go", editMain, "Here:\n```diff\n@@ -1 +1 @@\n-old\n+new\n```\nReady to commit."}},
		"edit, ends on a question":                            {transcript: []string{"PROMPT:go", editMain, "Added the seed. Want me to add the second variant too?"}},
		"ele:checkpoint skill invoked this turn":              {transcript: []string{"PROMPT:go", editMain, "SKILL:ele:checkpoint", "Done. Ready to commit."}},
		"em-dash pass labels":                                 {transcript: []string{"PROMPT:go", editMain, "**Quality — ran now.** **Prose — ran now, two trims.** Ready to commit."}},
		"nothing left to do":                                  {block: true, transcript: []string{"PROMPT:go", editMain, "All done, nothing left to do."}},
		"progress count":                                      {transcript: []string{"PROMPT:go", editMain, "The parser step is now done; two are left."}},
		"progress count of what is still to do":               {transcript: []string{"PROMPT:go", editMain, "The parser step is now done, with the writer still to do."}},
		"progress count of what remains":                      {transcript: []string{"PROMPT:go", editMain, "The parser step is now done; three more remain."}},
		"progress phrase with nothing left":                   {block: true, transcript: []string{"PROMPT:go", editMain, "The parser step is now done."}},
		"prose named but no quality pass":                     {block: true, transcript: []string{"PROMPT:go", editMain, "**Prose — ran now, two trims.** Ready to commit."}},
		"quality named but no prose pass":                     {block: true, transcript: []string{"PROMPT:go", editMain, "# Quality review of the diff. Ready to commit."}},
		"quality review and comment audit":                    {transcript: []string{"PROMPT:go", editMain, "**Quality review — partial credit.** **Comment audit — stale.** Ready to commit."}},
		"the prompt asked for the passes, the reply ran none": {block: true, transcript: []string{"PROMPT:please\ndo a quality review and a prose audit", editMain, "Edited the file. Ready to commit."}},
		"user typed /checkpoint":                              {transcript: []string{"PROMPT:<command-message>checkpoint</command-message>\n<command-name>/checkpoint</command-name>", editMain, doneClaim}},
		"user typed /ele:checkpoint":                          {transcript: []string{"PROMPT:/ele:checkpoint", editMain, doneClaim}},
		"work not claimed done":                               {transcript: nil},
	} {
		t.Run(name, func(t *testing.T) {
			root := hookRepository(t)
			snapshotHook(t)
			dirty(t, root)

			output := gateHook(t, transcriptFile(t, test.transcript...), false)

			require.Equal(t, test.block, strings.Contains(output, `"hookSpecificOutput"`), output)
		})
	}
}

func TestRunCheckpointGateFollowsTheRunTheRecordAndTheOptout(t *testing.T) {
	root := hookRepository(t)
	t.Setenv("CLAUDE_CODE_SESSION_ID", hookSession)
	done := transcriptFile(t, "PROMPT:go", editMain, "EDIT:deck/one.md", doneClaim)
	turn := func(change func()) string {
		snapshotHook(t)
		change()
		return gateHook(t, done, false)
	}
	freeze := func() {
		_, code := runCheckpoint(t, "freeze", root, t.TempDir(), "--", "main.go")
		require.Equal(t, 0, code)
	}

	require.Empty(t, turn(func() { dirty(t, root); freeze() }), "frozen, not recorded")
	_, code := runCheckpoint(t, "record", root)
	require.Equal(t, 0, code)
	require.Empty(t, turn(func() {}), "reviewed state, skill call out of view")
	require.Contains(t, turn(func() { dirty(t, root) }), `"hookSpecificOutput"`, "record closed the run")

	run, err := checkpoint.RunFile(context.Background(), root, hookSession)
	require.NoError(t, err)
	require.Contains(t, turn(func() {
		dirty(t, root)
		freeze()
		old := time.Now().Add(-3 * time.Hour)
		require.NoError(t, os.Chtimes(run+".open", old, old))
	}), `"hookSpecificOutput"`, "open marker abandoned hours ago")
	require.Contains(t, turn(func() {
		dirty(t, root)
		freeze()
		_, closed := runCheckpoint(t, "close", root)
		require.Equal(t, 0, closed)
	}), `"hookSpecificOutput"`, "close ended the run")
	require.Contains(t, turn(func() {
		dirty(t, root)
		t.Setenv("CLAUDE_CODE_SESSION_ID", "another-session")
		freeze()
		t.Setenv("CLAUDE_CODE_SESSION_ID", hookSession)
	}), `"hookSpecificOutput"`, "another session's run is open")

	require.NoError(t, os.MkdirAll(filepath.Join(root, "deck"), 0o700))
	output, code := runCheckpoint(t, "optout", root, "deck/*")
	require.Equal(t, 0, code, output)
	require.Equal(t, "opted out for this session: deck/*\n", output)
	runGit(t, root, "checkout", "-q", "--", ".")
	require.Empty(t, turn(func() { writeTestFile(t, root, "deck/one.md", "slide\n") }), "every moved path exempted")
	require.Contains(t, turn(func() { writeTestFile(t, root, "deck/one.md", "slide 2\n"); dirty(t, root) }), `"hookSpecificOutput"`, "a moved path outside the exemption")
}

func TestRunCheckpointGateNamesWhatMovedAndLogsTheBlock(t *testing.T) {
	root := hookRepository(t)
	snapshotHook(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "coverage.out"), []byte("coverage\n"), 0o600))
	dirty(t, root)

	output := gateHook(t, transcriptFile(t, "PROMPT:go", "EDIT:coverage.out", editMain, doneClaim), false)

	var block stopOutput
	require.NoError(t, json.Unmarshal([]byte(output), &block), output)
	require.Equal(t, "Stop", block.HookSpecificOutput.HookEventName)
	require.Contains(t, block.HookSpecificOutput.AdditionalContext, "no checkpoint was reported.\n\nChanged: coverage.out main.go\n\n")
	require.Contains(t, block.HookSpecificOutput.AdditionalContext, "The `ele:checkpoint` skill reviews work before it is reported done.")
	require.Contains(t, block.HookSpecificOutput.AdditionalContext, "`ele checkpoint optout '"+root+"' \"<glob>\" ...`")
	logged, err := os.ReadFile(os.Getenv("HOOK_LOG"))
	require.NoError(t, err)
	fields := strings.Split(strings.TrimSuffix(string(logged), "\n"), "\t")
	require.Equal(t, []string{"quality", "block", "checkpoint", filepath.Base(root), filepath.Base(os.Getenv("PWD"))}, fields[1:])
}

func TestRunCheckpointGateOffClearsTheBaselineAndABrokenConfigKeepsItOn(t *testing.T) {
	root := hookRepository(t)
	stateFile, err := checkpoint.StateFile(hookSession, root)
	require.NoError(t, err)
	done := transcriptFile(t, "PROMPT:go", editMain, doneClaim)

	snapshotHook(t)
	dirty(t, root)
	setGate(t, "false")
	require.Empty(t, gateHook(t, done, false))
	require.NoFileExists(t, stateFile)

	setGate(t, "")
	snapshotHook(t)
	setGate(t, "false")
	snapshotHook(t)
	require.NoFileExists(t, stateFile, "the snapshot clears a baseline while the gate is off")
	setGate(t, "")
	dirty(t, root)
	require.Empty(t, gateHook(t, done, false), "no baseline from while the gate was off")

	require.NoError(t, os.WriteFile(filepath.Join(os.Getenv("ELE_HOME"), "config.json"), []byte(`{"checkpoint_gate": fals`), 0o600))
	snapshotHook(t)
	dirty(t, root)
	require.Contains(t, gateHook(t, done, false), `"hookSpecificOutput"`)
}

func TestRunCheckpointGatePassesTurnsThatAreNotCheckpoints(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T, root string) bool{
		"clean tree": func(t *testing.T, root string) bool {
			snapshotHook(t)
			return false
		},
		"dirty before the turn, unchanged during it": func(t *testing.T, root string) bool {
			dirty(t, root)
			snapshotHook(t)
			return false
		},
		"no baseline: dirt predates the session": func(t *testing.T, root string) bool {
			dirty(t, root)
			return false
		},
		"stop hook already fired": func(t *testing.T, root string) bool {
			snapshotHook(t)
			dirty(t, root)
			return true
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := hookRepository(t)
			stopHookActive := setup(t, root)

			require.Empty(t, gateHook(t, transcriptFile(t, "PROMPT:go", editMain, doneClaim), stopHookActive))
		})
	}
	t.Run("outside a repository", func(t *testing.T) {
		hookRepository(t)
		t.Chdir(t.TempDir())
		require.Empty(t, gateHook(t, transcriptFile(t, "PROMPT:go", editMain, doneClaim), false))
	})
}

func TestRunCheckpointGateSeesAnEditPastThePathCap(t *testing.T) {
	root := hookRepository(t)
	for index := range 200 {
		writeTestFile(t, root, fmt.Sprintf("a/%03d.txt", index), "x\n")
	}
	snapshotHook(t)
	dirty(t, root)

	output := gateHook(t, transcriptFile(t, "PROMPT:go", editMain, doneClaim), false)

	require.Contains(t, output, `"hookSpecificOutput"`, "main.go sorts past the cap, so no hash of it moved")
	require.Contains(t, output, "Changed: main.go")

	snapshotHook(t)
	dirty(t, root)
	output = gateHook(t, transcriptFile(t, "PROMPT:go", "BASH:echo x > main.go", doneClaim), false)
	require.Contains(t, output, "Changed: (Bash writes past the 200-path cap)", "a Bash write names no path to weigh")
}

func TestRunCheckpointGateWeighsOnlyWhatThisTurnWrote(t *testing.T) {
	for name, test := range map[string]struct {
		block   string
		entries []string
	}{
		"a Bash grep for a skill call is not one":                  {block: "Changed: main.go\n", entries: []string{"PROMPT:go", editMain, "BASH:grep -rn 'command-name>/ele:checkpoint' .", doneClaim}},
		"another session's file, the turn edited a file not moved": {entries: []string{"PROMPT:go", "EDIT:other.go", doneClaim}},
		"another session's file, the turn edited nothing":          {entries: []string{"PROMPT:go", doneClaim}},
		"another session's file, the turn only read through Bash":  {entries: []string{"PROMPT:go", "BASH:git status --short; git diff", doneClaim}},
		"the turn wrote through Bash beside an edit elsewhere":     {block: "Changed: main.go theirs.go\n", entries: []string{"PROMPT:go", "EDIT:/elsewhere/brief.md", "BASH:sed -i '' 's/a/b/' main.go", doneClaim}},
		"the turn wrote through Bash, naming no path":              {block: "Changed: main.go theirs.go\n", entries: []string{"PROMPT:go", "BASH:sed -i '' 's/a/b/' main.go", doneClaim}},
		"the turn's NotebookEdit beside another session's file":    {block: "Changed: main.go\n", entries: []string{"PROMPT:go", "NOTEBOOK:main.go", doneClaim}},
		"the turn's Write beside another session's file":           {block: "Changed: main.go\n", entries: []string{"PROMPT:go", "WRITE:main.go", doneClaim}},
		"the turn's edit beside another session's file":            {block: "Changed: main.go\n", entries: []string{"PROMPT:go", editMain, doneClaim}},
		"the turn's start past the transcript's tail":              {block: "Changed: main.go theirs.go\n", entries: append(append([]string{"PROMPT:go", editMain}, slices.Repeat([]string{"Reading."}, 400)...), doneClaim)},
	} {
		t.Run(name, func(t *testing.T) {
			root := hookRepository(t)
			snapshotHook(t)
			dirty(t, root)
			writeTestFile(t, root, "theirs.go", "package main\n")

			output := gateHook(t, transcriptFile(t, test.entries...), false)

			if test.block == "" {
				require.Empty(t, output)
				return
			}
			var block stopOutput
			require.NoError(t, json.Unmarshal([]byte(output), &block), output)
			require.Contains(t, block.HookSpecificOutput.AdditionalContext, test.block)
		})
	}
	t.Run("ele's data directory inside the repository", func(t *testing.T) {
		root := hookRepository(t)
		t.Setenv("ELE_HOME", filepath.Join(root, ".ele"))
		snapshotHook(t)
		writeTestFile(t, root, ".ele/notes/note.md", "kept\n")

		require.Empty(t, gateHook(t, transcriptFile(t, "PROMPT:go", "BASH:ele remember --kind decision x", doneClaim), false))

		snapshotHook(t)
		dirty(t, root)
		writeTestFile(t, root, ".ele/notes/other.md", "kept\n")
		writeTestFile(t, root, ".ele-other/x.md", "theirs\n")
		var block stopOutput
		output := gateHook(t, transcriptFile(t, "PROMPT:go", "BASH:sed -i '' 's/a/b/' main.go", doneClaim), false)
		require.NoError(t, json.Unmarshal([]byte(output), &block), output)
		require.Contains(t, block.HookSpecificOutput.AdditionalContext, "Changed: .ele-other/x.md main.go\n")

		for index := range 250 {
			writeTestFile(t, root, ".ele/notes/"+strconv.Itoa(index)+".md", "kept\n")
		}
		snapshotHook(t)
		dirty(t, root)
		output = gateHook(t, transcriptFile(t, "PROMPT:go", editMain, doneClaim), false)
		require.NoError(t, json.Unmarshal([]byte(output), &block), output)
		require.Contains(t, block.HookSpecificOutput.AdditionalContext, "Changed: main.go\n")
	})
	t.Run("the turn's own files recorded while another session's are dirty", func(t *testing.T) {
		root := hookRepository(t)
		t.Setenv("CLAUDE_CODE_SESSION_ID", hookSession)
		snapshotHook(t)
		dirty(t, root)
		_, code := runCheckpoint(t, "freeze", root, t.TempDir(), "--", "main.go")
		require.Equal(t, 0, code)
		_, code = runCheckpoint(t, "record", root)
		require.Equal(t, 0, code)
		writeTestFile(t, root, "theirs.go", "package main\n")

		require.Empty(t, gateHook(t, transcriptFile(t, "PROMPT:go", editMain, doneClaim), false))
	})
	t.Run("the turn reverted a recorded file", func(t *testing.T) {
		root := hookRepository(t)
		t.Setenv("CLAUDE_CODE_SESSION_ID", hookSession)
		dirty(t, root)
		_, code := runCheckpoint(t, "freeze", root, t.TempDir(), "--", "main.go")
		require.Equal(t, 0, code)
		_, code = runCheckpoint(t, "record", root)
		require.Equal(t, 0, code)
		snapshotHook(t)
		runGit(t, root, "checkout", "-q", "--", "main.go")

		require.Empty(t, gateHook(t, transcriptFile(t, "PROMPT:go", editMain, doneClaim), false))
	})
}

func TestRunCheckpointLogAppendsFromASubdirectory(t *testing.T) {
	root, logPath := checkpointRepository(t)

	output, code := runCheckpointLog(t, logRecord, filepath.Join(root, "sub"))

	require.Equal(t, 0, code, output)
	require.Equal(t, "logged: "+root+"\n", output)
	written, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Contains(t, string(written), `"repository":"`+root+`"`)
}

func TestRunCheckpointLogRefusesARecordAndKeepsTheFreezeTimeForTheRetry(t *testing.T) {
	root, logPath := checkpointRepository(t)
	t.Setenv("CLAUDE_CODE_SESSION_ID", "this-session")
	run, err := checkpoint.RunFile(context.Background(), root, "this-session")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(run+".started", []byte("1\n"), 0o600))

	output, code := runCheckpointLog(t, strings.Replace(logRecord, `"small"`, `"medium"`, 1), root)

	require.Equal(t, 2, code)
	require.Equal(t, "ele: record refused:\n  mode \"medium\" is not one of full, small\n", output)
	require.NoFileExists(t, logPath)
	require.FileExists(t, run+".started")

	output, code = runCheckpointLog(t, "{", root)
	require.Equal(t, 2, code)
	require.True(t, strings.HasPrefix(output, "ele: the record is not JSON: "), output)
}

func TestRunCheckpointLogRefusesOutsideARepository(t *testing.T) {
	t.Chdir(t.TempDir())

	for _, args := range [][]string{{}, {""}} {
		output, code := runCheckpointLog(t, logRecord, args...)

		require.Equal(t, 2, code, args)
		require.Equal(t, "ele: the working directory is not inside a git repository\n", output, "an empty root reads as none, as the script's ${1:-.} did")
	}
}

func TestRunCheckpointRecordSubcommandsRefuseBadCalls(t *testing.T) {
	root := hookRepository(t)
	work := t.TempDir()
	for name, args := range map[string][]string{
		"freeze with no paths":      {"freeze", root, work, "--"},
		"freeze into no directory":  {"freeze", root, filepath.Join(work, "missing"), "--", "main.go"},
		"freeze with a bare --base": {"freeze", root, work, "--base"},
		"optout with no glob":       {"optout", root},
		"two roots":                 {"check", root, root},
	} {
		_, code := runCheckpoint(t, args...)
		require.Equal(t, 2, code, name)
	}
	output, code := runCheckpoint(t, "optout", root, "deck/*")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: CLAUDE_CODE_SESSION_ID is unset\n", output)
	t.Setenv("CLAUDE_CODE_SESSION_ID", hookSession)
	output, code = runCheckpoint(t, "optout", root, "*")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: '*' is too broad; name the paths\n", output)
	output, code = runCheckpoint(t, "check", root)
	require.Equal(t, 1, code)
	require.Equal(t, "not covered: "+root+" has changed since the last checkpoint, or none ran\n", output)
}

func TestRunCheckpointSnapshotKeepsTheBaselineOfATurnStillRunning(t *testing.T) {
	root := hookRepository(t)
	snapshotHook(t)
	dirty(t, root)
	snapshotHook(t)

	require.Contains(t, gateHook(t, transcriptFile(t, "PROMPT:go", editMain, doneClaim), false), `"hookSpecificOutput"`)
}

func TestRunCheckpointStatsRefusesBadArguments(t *testing.T) {
	t.Setenv("CHECKPOINT_LOG", filepath.Join(t.TempDir(), "checkpoints.jsonl"))

	for _, args := range [][]string{{"7"}, {"--since", "7d", "7"}} {
		output, code := runCheckpointStats(t, args...)
		require.Equal(t, 2, code, "a bare number of days is refused")
		require.True(t, strings.HasPrefix(output, "ele checkpoint stats [--since <YYYY-MM-DD|Nd>]\n"), output)
	}

	output, code := runCheckpointStats(t, "--since", "last week")
	require.Equal(t, 2, code)
	require.Equal(t, "ele: --since takes YYYY-MM-DD or a number of days like 7d, not \"last week\"\n", output)

	output, code = runCheckpointStats(t, "--baseline")
	require.Equal(t, 2, code, "the mined baseline moved to ele mine --report")
	require.Contains(t, output, "flag provided but not defined: -baseline")
}

func TestRunCheckpointStatsWindowsBySince(t *testing.T) {
	t.Setenv("CHECKPOINT_LOG", filepath.Join(t.TempDir(), "checkpoints.jsonl"))
	today := time.Now()

	output, code := runCheckpointStats(t, "--since", "7d")
	require.Equal(t, 0, code, output)
	require.True(t, strings.HasPrefix(output, "Since "+today.AddDate(0, 0, -7).Format(time.DateOnly)+":"), output)
	output, code = runCheckpointStats(t, "--since", "2026-01-02")
	require.Equal(t, 0, code, output)
	require.True(t, strings.HasPrefix(output, "Since 2026-01-02:"), output)
	output, code = runCheckpointStats(t)
	require.Equal(t, 0, code, output)
	require.True(t, strings.HasPrefix(output, "Since "+today.AddDate(0, 0, -30).Format(time.DateOnly)+":"), output)
}

func checkpointRepository(t *testing.T) (string, string) {
	t.Helper()
	directory := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"commit", "-q", "--allow-empty", "-m", "first"}} {
		output, err := exec.Command("git", append([]string{"-C", directory, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
		require.NoError(t, err, string(output))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(directory, "sub"), 0o700))
	root, err := checkpoint.Root(context.Background(), directory)
	require.NoError(t, err)
	logPath := filepath.Join(t.TempDir(), "checkpoints.jsonl")
	t.Setenv("CHECKPOINT_LOG", logPath)
	t.Setenv("CHECKPOINT_REVIEWED_DIR", filepath.Join(t.TempDir(), "reviewed"))
	return root, logPath
}

func dirty(t *testing.T, root string) {
	t.Helper()
	writeTestFile(t, root, "main.go", "package main // "+time.Now().Format(time.RFC3339Nano)+"\n")
}

func gateHook(t *testing.T, transcriptPath string, stopHookActive bool) string {
	t.Helper()
	input, err := json.Marshal(map[string]any{"session_id": hookSession, "stop_hook_active": stopHookActive, "transcript_path": transcriptPath})
	require.NoError(t, err)
	var stdout, stderr bytes.Buffer
	code := checkpointCommand(context.Background(), []string{"gate"}, bytes.NewReader(input), &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())
	require.Empty(t, stderr.String())
	return stdout.String()
}

// hookRepository is a repository holding a committed main.go, entered, with every path the hooks write kept under the test.
func hookRepository(t *testing.T) string {
	t.Helper()
	root, _ := checkpointRepository(t)
	writeTestFile(t, root, "main.go", "package main\n")
	runGit(t, root, "add", "main.go")
	runGit(t, root, "commit", "-qm", "main")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	t.Setenv("ELE_HOME", t.TempDir())
	t.Setenv("HOOK_LOG", filepath.Join(t.TempDir(), "hooks.log"))
	t.Setenv("TMPDIR", t.TempDir())
	t.Chdir(root)
	t.Setenv("PWD", root)
	return root
}

func runCheckpoint(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var output bytes.Buffer
	code := checkpointCommand(context.Background(), args, strings.NewReader(""), &output, &output)
	return output.String(), code
}

func runCheckpointDensity(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var output bytes.Buffer
	code := checkpointCommand(context.Background(), append([]string{"density"}, args...), strings.NewReader(""), &output, &output)
	return output.String(), code
}

func runCheckpointLog(t *testing.T, record string, args ...string) (string, int) {
	t.Helper()
	var output bytes.Buffer
	code := checkpointCommand(context.Background(), append([]string{"log"}, args...), strings.NewReader(record), &output, &output)
	return output.String(), code
}

func runCheckpointStats(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var output bytes.Buffer
	code := checkpointCommand(context.Background(), append([]string{"stats"}, args...), strings.NewReader(""), &output, &output)
	return output.String(), code
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
	require.NoError(t, err, string(output))
}

func setGate(t *testing.T, value string) {
	t.Helper()
	path := filepath.Join(os.Getenv("ELE_HOME"), "config.json")
	if value == "" {
		require.NoError(t, os.RemoveAll(path))
		return
	}
	require.NoError(t, os.WriteFile(path, []byte(`{"checkpoint_gate": `+value+`}`), 0o600))
}

func snapshotHook(t *testing.T) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := checkpointCommand(context.Background(), []string{"snapshot"}, strings.NewReader(`{"session_id":"`+hookSession+`"}`), &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())
	require.Empty(t, stdout.String()+stderr.String())
}

// transcriptFile writes a turn: a typed prompt (PROMPT:), a Bash call (BASH:), an Edit, NotebookEdit or Write of a path
// under the working directory (EDIT:, NOTEBOOK:, WRITE:), a skill call (SKILL:) or assistant text.
func transcriptFile(t *testing.T, entries ...string) string {
	t.Helper()
	tool := func(name string, input map[string]any) map[string]any {
		return map[string]any{"message": map[string]any{"content": []any{map[string]any{"input": input, "name": name, "type": "tool_use"}}, "role": "assistant"}, "type": "assistant"}
	}
	absolute := func(path string) string {
		path, err := filepath.Abs(path)
		require.NoError(t, err)
		return path
	}
	var lines bytes.Buffer
	for _, entry := range entries {
		var line map[string]any
		switch {
		case strings.HasPrefix(entry, "BASH:"):
			line = tool("Bash", map[string]any{"command": strings.TrimPrefix(entry, "BASH:")})
		case strings.HasPrefix(entry, "EDIT:"):
			line = tool("Edit", map[string]any{"file_path": absolute(strings.TrimPrefix(entry, "EDIT:"))})
		case strings.HasPrefix(entry, "NOTEBOOK:"):
			line = tool("NotebookEdit", map[string]any{"notebook_path": absolute(strings.TrimPrefix(entry, "NOTEBOOK:"))})
		case strings.HasPrefix(entry, "PROMPT:"):
			line = map[string]any{"isMeta": false, "message": map[string]any{"content": strings.TrimPrefix(entry, "PROMPT:"), "role": "user"}, "type": "user"}
		case strings.HasPrefix(entry, "SKILL:"):
			line = tool("Skill", map[string]any{"skill": strings.TrimPrefix(entry, "SKILL:")})
		case strings.HasPrefix(entry, "WRITE:"):
			line = tool("Write", map[string]any{"file_path": absolute(strings.TrimPrefix(entry, "WRITE:"))})
		default:
			line = map[string]any{"message": map[string]any{"content": []any{map[string]any{"text": entry, "type": "text"}}, "role": "assistant"}, "type": "assistant"}
		}
		encoded, err := json.Marshal(line)
		require.NoError(t, err)
		lines.Write(append(encoded, '\n'))
	}
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	require.NoError(t, os.WriteFile(path, lines.Bytes(), 0o600))
	return path
}

func writeTestFile(t *testing.T, root string, file string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, file)), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, file), []byte(content), 0o600))
}
