package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/atomicfile"
)

func TestConfigGetReadsABrokenFileAsDefaults(t *testing.T) {
	home := recallFixture(t)
	writeFile(t, filepath.Join(home, "data", "config.json"), `{"checkpoint_gate": fals`)

	output, code := runCommand(t, "config", "get", "checkpoint_gate")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "every option reads as its default")
	require.Contains(t, output, "true\n")
}

func TestConfigGetReadsAutoNoteWritesOffByDefault(t *testing.T) {
	home := recallFixture(t)

	output, code := runCommand(t, "config", "get", "auto_note_writes")
	require.Equal(t, 0, code, output)
	require.Equal(t, "false\n", output)

	writeFile(t, filepath.Join(home, "data", "config.json"), `{"auto_note_writes": true}`)
	output, code = runCommand(t, "config", "get", "auto_note_writes")
	require.Equal(t, 0, code, output)
	require.Equal(t, "true\n", output)
	output, code = runCommand(t, "config", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "auto_note_writes  true  (global)\n")
}

func TestConfigListFlagsKeysNoOptionTakes(t *testing.T) {
	home := recallFixture(t)
	writeFile(t, filepath.Join(home, "data", "config.json"), `{"later_option": 3, "owners": {"corp": {"other": true}}}`)

	output, code := runCommand(t, "config", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Ignored, no such option: later_option, owners.corp.other\n")
}

func TestConfigOwnerOverrideAppliesInThatOwnersProjects(t *testing.T) {
	home := recallFixture(t)
	t.Setenv("CLAUDECODE", "")

	output, code := runCommand(t, "config", "set", "--owner", "corp", "checkpoint_gate", "false")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Set checkpoint_gate to false for owner corp.\n", output)

	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code = runCommand(t, "config", "get", "checkpoint_gate")
	require.Equal(t, 0, code, output)
	require.Equal(t, "false\n", output)
	output, code = runCommand(t, "config", "list")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "checkpoint_gate  false  (owner corp; default true)\n")

	t.Chdir(filepath.Join(home, "w", "personal", "notes"))
	output, code = runCommand(t, "config", "get", "checkpoint_gate")
	require.Equal(t, 0, code, output)
	require.Equal(t, "true\n", output)

	output, code = runCommand(t, "config", "unset", "--owner", "corp", "checkpoint_gate")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Unset checkpoint_gate for owner corp.\n", output)
	output, code = runCommand(t, "config", "get", "--owner", "corp", "checkpoint_gate")
	require.Equal(t, 0, code, output)
	require.Equal(t, "true\n", output)
}

func TestConfigReviewVoiceRoundTripsGloballyAndPerOwner(t *testing.T) {
	home := recallFixture(t)
	t.Setenv("CLAUDECODE", "")
	writeFile(t, filepath.Join(home, "voice.md"), "# Voice\n")
	writeFile(t, filepath.Join(home, "work-voice.md"), "# Voice\n")

	output, code := runCommand(t, "config", "list", "--owner", "corp")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "review_voice  \"\"  (default)\n")

	output, code = runCommand(t, "config", "set", "--owner", "corp", "review_voice", filepath.Join(home, "work-voice.md"))
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "config", "list", "--owner", "corp")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "(owner corp; default \"\")\n")

	output, code = runCommand(t, "config", "set", "review_voice", "~/voice.md")
	require.Equal(t, 0, code, output)
	require.Equal(t, "Set review_voice to "+filepath.Join(home, "voice.md")+" globally.\n", output)

	output, code = runCommand(t, "config", "get", "--owner", "corp", "review_voice")
	require.Equal(t, 0, code, output)
	require.Equal(t, filepath.Join(home, "work-voice.md")+"\n", output)
	output, code = runCommand(t, "config", "get", "--owner", "personal", "review_voice")
	require.Equal(t, 0, code, output)
	require.Equal(t, filepath.Join(home, "voice.md")+"\n", output)

	output, code = runCommand(t, "config", "list", "--owner", "corp")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "review_voice  "+filepath.Join(home, "work-voice.md")+"  (owner corp; global "+filepath.Join(home, "voice.md")+")\n")
	require.Contains(t, output, "voice file ele:review drafts comments from")
}

func TestConfigSetRefusesABrokenFile(t *testing.T) {
	home := recallFixture(t)
	t.Setenv("CLAUDECODE", "")
	path := filepath.Join(home, "data", "config.json")
	writeFile(t, path, `{"checkpoint_gate": fals`)

	output, code := runCommand(t, "config", "set", "checkpoint_gate", "false")
	require.Equal(t, 1, code, output)
	require.Contains(t, output, "fix or remove it before setting an option")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, `{"checkpoint_gate": fals`, string(content))
}

func TestConfigSetRefusesAnUnlistedOwner(t *testing.T) {
	recallFixture(t)
	t.Setenv("CLAUDECODE", "")

	output, code := runCommand(t, "config", "set", "--owner", "acme", "checkpoint_gate", "false")
	require.Equal(t, 1, code, output)
	require.Contains(t, output, `owner-map.json does not list owner "acme"`)
}

func TestConfigSetRefusesInsideASession(t *testing.T) {
	home := recallFixture(t)
	t.Setenv("CLAUDECODE", "1")

	for _, args := range [][]string{{"set", "checkpoint_gate", "false"}, {"unset", "checkpoint_gate"}} {
		output, code := runCommand(t, append([]string{"config"}, args...)...)
		require.Equal(t, 1, code, output)
		require.Contains(t, output, "refuses to run inside one; the user runs it outside the session: ele config "+args[0])
	}
	_, err := os.Stat(filepath.Join(home, "data", "config.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestConfigSetAndUnsetWaitForTheConfigLock(t *testing.T) {
	for _, args := range [][]string{{"set", "checkpoint_gate", "false"}, {"unset", "checkpoint_gate"}} {
		t.Run(args[0], func(t *testing.T) {
			home := recallFixture(t)
			t.Setenv("CLAUDECODE", "")
			output, code := runCommand(t, "config", "set", "checkpoint_gate", "true")
			require.Equal(t, 0, code, output)
			path := filepath.Join(home, "data", "config.json")
			before, err := os.ReadFile(path)
			require.NoError(t, err)
			unlock, err := atomicfile.Lock(path + ".lock")
			require.NoError(t, err)
			done := make(chan int, 1)

			go func() {
				var output bytes.Buffer
				done <- run(context.Background(), append([]string{"config"}, args...), &output, &output)
			}()

			select {
			case <-done:
				t.Fatalf("config %s wrote while another writer held the lock", args[0])
			case <-time.After(300 * time.Millisecond):
			}
			during, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, string(before), string(during))
			unlock()
			require.Equal(t, 0, <-done)
			after, err := os.ReadFile(path)
			require.NoError(t, err)
			require.NotEqual(t, string(before), string(after))
		})
	}
}

func TestConfigUnsetSaysWhenNoValueIsSet(t *testing.T) {
	home := recallFixture(t)
	t.Setenv("CLAUDECODE", "")

	output, code := runCommand(t, "config", "unset", "checkpoint_gate")
	require.Equal(t, 0, code, output)
	require.Equal(t, "checkpoint_gate has no value set globally.\n", output)
	_, err := os.Stat(filepath.Join(home, "data", "config.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
