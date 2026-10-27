package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadReadsAMissingFileAsDefaults(t *testing.T) {
	config, err := Load(filepath.Join(t.TempDir(), "config.json"))
	require.NoError(t, err)

	value, err := config.Value("checkpoint_gate", "work")
	require.NoError(t, err)
	require.Equal(t, Value{Name: "checkpoint_gate", Source: SourceDefault, Value: true}, value)
}

func TestLoadReadsNullOwnersAsNoOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"owners": null}`), 0o600))
	config, err := Load(path)
	require.NoError(t, err)

	require.NoError(t, config.Set("checkpoint_gate", "work", "false"))
	value, err := config.Value("checkpoint_gate", "work")
	require.NoError(t, err)
	require.Equal(t, false, value.Value)
}

func TestLoadRefusesAKnownOptionOfTheWrongKind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"owners": {"work": {"checkpoint_gate": "no"}}}`), 0o600))

	_, err := Load(path)
	require.ErrorContains(t, err, "owners.work.checkpoint_gate: want a boolean")
}

func TestLoadRefusesAnIntegerOptionHoldingANegativeOrNonWholeNumber(t *testing.T) {
	tests := []struct {
		content string
		name    string
		want    string
	}{
		{content: `{"compact_hint_at": 1.5}`, name: "fraction", want: "compact_hint_at: want a whole number of 0 or more, got 1.5"},
		{content: `{"compact_hint_at": -1}`, name: "negative", want: "compact_hint_at: want a whole number of 0 or more, got -1"},
		{content: `{"compact_hint_at": null}`, name: "null", want: "compact_hint_at: want a whole number of 0 or more, got null"},
		{content: `{"owners": {"work": {"compact_hint_at": "400000"}}}`, name: "string", want: `owners.work.compact_hint_at: want a whole number of 0 or more, got "400000"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			require.NoError(t, os.WriteFile(path, []byte(test.content), 0o600))

			_, err := Load(path)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestLoadRefusesAPathOptionHoldingANonStringOrARelativePath(t *testing.T) {
	tests := []struct {
		content string
		name    string
		want    string
	}{
		{content: `{"review_voice": 3}`, name: "non-string", want: "review_voice: want a path, got 3"},
		{content: `{"review_voice": null}`, name: "null", want: `review_voice: want an absolute or ~/ path, got ""`},
		{content: `{"owners": {"work": {"review_voice": "voice.md"}}}`, name: "relative", want: `owners.work.review_voice: want an absolute or ~/ path, got "voice.md"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			require.NoError(t, os.WriteFile(path, []byte(test.content), 0o600))

			_, err := Load(path)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestSaveKeepsUnknownKeysAndDropsEmptyOwners(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"later_option": 3, "owners": {"work": {"checkpoint_gate": false}}}`), 0o600))
	config, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, []string{"later_option"}, config.Unknown())

	removed, err := config.Unset("checkpoint_gate", "work")
	require.NoError(t, err)
	require.True(t, removed)
	require.NoError(t, config.Set("checkpoint_gate", "", "false"))
	require.NoError(t, Save(path, config))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "{\n  \"checkpoint_gate\": false,\n  \"later_option\": 3\n}\n", string(content))
}

func TestSetRefusesAnUnknownOptionOrAValueOfTheWrongKind(t *testing.T) {
	config := Defaults()
	require.ErrorContains(t, config.Set("checkpoint_gat", "", "false"), `no option "checkpoint_gat"; options: auto_note_writes, checkpoint_gate`)
	require.ErrorContains(t, config.Set("checkpoint_gate", "", "off"), `checkpoint_gate takes true or false, not "off"`)
	require.Empty(t, config.Global)
}

func TestSetStoresAnIntegerAndRefusesANegativeOrNonWholeOne(t *testing.T) {
	config := Defaults()
	require.EqualError(t, config.Set("compact_hint_at", "", "-1"), `compact_hint_at takes a whole number of 0 or more, not "-1"`)
	require.EqualError(t, config.Set("compact_hint_at", "", "400k"), `compact_hint_at takes a whole number of 0 or more, not "400k"`)
	require.Empty(t, config.Global)

	require.NoError(t, config.Set("compact_hint_at", "work", "400000"))
	require.JSONEq(t, `400000`, string(config.Owners["work"]["compact_hint_at"]))
	value, err := config.Value("compact_hint_at", "work")
	require.NoError(t, err)
	require.Equal(t, Value{Name: "compact_hint_at", Owner: "work", Source: SourceOwner, Value: 400000}, value)

	value, err = config.Value("compact_hint_at", "")
	require.NoError(t, err)
	require.Equal(t, Value{Name: "compact_hint_at", Source: SourceDefault, Value: 150000}, value)
}

func TestSetRefusesAPathNamingNoReadableFile(t *testing.T) {
	directory := t.TempDir()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "directory", path: directory, want: "review_voice: " + directory + " is not a regular file"},
		{name: "empty", path: "", want: `review_voice takes an absolute or ~/ path, not ""; ele config unset review_voice returns it to its default`},
		{name: "missing", path: filepath.Join(directory, "voice.md"), want: "review_voice: stat " + filepath.Join(directory, "voice.md") + ": no such file or directory"},
		{name: "relative", path: "voice.md", want: `review_voice takes an absolute or ~/ path, not "voice.md"; ele config unset review_voice returns it to its default`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := Defaults()
			require.EqualError(t, config.Set("review_voice", "", test.path), test.want)
			require.Empty(t, config.Global)
		})
	}
}

func TestSetStoresAPathAsTypedAndValueExpandsIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.WriteFile(filepath.Join(home, "voice.md"), []byte("# Voice\n"), 0o600))
	config := Defaults()

	require.NoError(t, config.Set("review_voice", "work", "~/voice.md"))
	require.JSONEq(t, `"~/voice.md"`, string(config.Owners["work"]["review_voice"]))
	value, err := config.Value("review_voice", "work")
	require.NoError(t, err)
	require.Equal(t, Value{Name: "review_voice", Owner: "work", Source: SourceOwner, Value: filepath.Join(home, "voice.md")}, value)

	value, err = config.Value("review_voice", "")
	require.NoError(t, err)
	require.Equal(t, Value{Name: "review_voice", Source: SourceDefault, Value: ""}, value)
}

func TestValuePrefersTheOwnerThenGlobalThenDefault(t *testing.T) {
	config := Defaults()
	require.NoError(t, config.Set("checkpoint_gate", "", "false"))
	require.NoError(t, config.Set("checkpoint_gate", "work", "true"))

	tests := []struct {
		name  string
		owner string
		want  Value
	}{
		{name: "no owner", owner: "", want: Value{Name: "checkpoint_gate", Source: SourceGlobal, Value: false}},
		{name: "owner override", owner: "work", want: Value{Name: "checkpoint_gate", Owner: "work", Source: SourceOwner, Value: true}},
		{name: "owner without override", owner: "personal", want: Value{Name: "checkpoint_gate", Source: SourceGlobal, Value: false}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := config.Value("checkpoint_gate", test.owner)
			require.NoError(t, err)
			require.Equal(t, test.want, value)
		})
	}
}
