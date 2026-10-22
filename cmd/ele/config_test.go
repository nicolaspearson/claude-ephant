package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
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

func TestSessionStartNamesOptionsOffTheirDefaults(t *testing.T) {
	home := recallFixture(t)
	writeFile(t, filepath.Join(home, "data", "config.json"), `{"checkpoint_gate": false}`)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))

	output, code := runCommand(t, "session-start", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Config from ele, off its defaults: checkpoint_gate false (global).\n")

	writeFile(t, filepath.Join(home, "data", "config.json"), `{"checkpoint_gate": true}`)
	output, code = runCommand(t, "session-start", "--no-sync")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "Config from ele")
}

func TestStatusConfigRowNamesOptionsOffTheirDefaults(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	path := filepath.Join(home, "data", "config.json")

	tests := []struct {
		content string
		name    string
		row     string
	}{
		{content: `{"checkpoint_gate": fals`, name: "broken", row: `(?m)^Config +parse .*config\.json: .*; every option reads as its default$`},
		{content: `{}`, name: "defaults", row: `(?m)^Config +defaults$`},
		{content: `{"owners": {"corp": {"checkpoint_gate": false}}}`, name: "owner override", row: `(?m)^Config +checkpoint_gate false \(owner corp; default true\)$`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writeFile(t, path, test.content)
			output, code := runCommand(t, "status")
			require.Equal(t, 0, code, output)
			require.Regexp(t, test.row, output)
		})
	}
}
