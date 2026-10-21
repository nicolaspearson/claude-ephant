package datadir

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathDefaultsToApplicationSupport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ELE_HOME", "")
	t.Setenv("HOME", home)

	path, err := Path()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "Library", "Application Support", "claude-ephant"), path)
}

func TestPathResolvesRelativeOverride(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	t.Setenv("ELE_HOME", "data")

	path, err := Path()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(directory, "data"), path)
}
