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
	require.ErrorContains(t, config.Set("checkpoint_gat", "", "false"), `no option "checkpoint_gat"; options: checkpoint_gate`)
	require.ErrorContains(t, config.Set("checkpoint_gate", "", "off"), `checkpoint_gate takes true or false, not "off"`)
	require.Empty(t, config.Global)
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
