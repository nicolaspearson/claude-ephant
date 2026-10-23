package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrentNamesTheProfileFromItsDirectory(t *testing.T) {
	require.Equal(t, "claude", Current(""))
	require.Equal(t, "claude", Current("/Users/me/.claude"))
	require.Equal(t, "corp", Current("/Users/me/.claude-corp/"))
}

func TestDiscoverNamesProfilesByDirectory(t *testing.T) {
	home := t.TempDir()
	for _, directory := range []string{".claude/projects", ".claude-corp/projects", ".claude-cache", ".claude-notes/projects-not"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, directory), 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude-backup.json"), []byte("{}"), 0o600))

	roots, err := Discover(home)
	require.NoError(t, err)
	require.Equal(t, []Root{
		{Directory: filepath.Join(home, ".claude"), Name: "claude"},
		{Directory: filepath.Join(home, ".claude-corp"), Name: "corp"},
	}, roots)
}
