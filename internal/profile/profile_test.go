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

func TestCurrentNamesARelativeProfileAsDiscoverDoes(t *testing.T) {
	scratch := filepath.Join(t.TempDir(), "scratch")
	require.NoError(t, os.MkdirAll(filepath.Join(scratch, "projects"), 0o700))
	t.Chdir(scratch)

	roots, _, err := Discover(t.TempDir(), ".")
	require.NoError(t, err)
	require.Len(t, roots, 1)
	require.Equal(t, "scratch", roots[0].Name)
	require.Equal(t, "scratch", Current("."))
}

func TestDiscoverNamesProfilesByDirectory(t *testing.T) {
	home := t.TempDir()
	for _, directory := range []string{".claude/projects", ".claude-corp/projects", ".claude-cache", ".claude-notes/projects-not"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, directory), 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude-backup.json"), []byte("{}"), 0o600))

	roots, skipped, err := Discover(home, "")
	require.NoError(t, err)
	require.Empty(t, skipped)
	require.Equal(t, []Root{
		{Directory: filepath.Join(home, ".claude"), Name: "claude"},
		{Directory: filepath.Join(home, ".claude-corp"), Name: "corp"},
	}, roots)
}

func TestDiscoverAddsTheConfigDirectoryOutsideHome(t *testing.T) {
	home := t.TempDir()
	outside := filepath.Join(t.TempDir(), "scratch")
	for _, directory := range []string{filepath.Join(home, ".claude", "projects"), filepath.Join(outside, "projects")} {
		require.NoError(t, os.MkdirAll(directory, 0o700))
	}

	roots, skipped, err := Discover(home, outside+"/")
	require.NoError(t, err)
	require.Empty(t, skipped)
	require.Equal(t, []Root{
		{Directory: filepath.Join(home, ".claude"), Name: "claude"},
		{Directory: outside, Name: "scratch"},
	}, roots)
}

func TestDiscoverListsAConfigDirectoryInHomeOnce(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude-corp", "projects"), 0o700))
	alias := filepath.Join(t.TempDir(), ".claude-corp")
	require.NoError(t, os.Symlink(filepath.Join(home, ".claude-corp"), alias))

	for _, configDirectory := range []string{filepath.Join(home, ".claude-corp"), alias} {
		roots, skipped, err := Discover(home, configDirectory)
		require.NoError(t, err)
		require.Empty(t, skipped)
		require.Equal(t, []Root{{Directory: filepath.Join(home, ".claude-corp"), Name: "corp"}}, roots)
	}
}

func TestDiscoverListsAConfigDirectoryOnceWhenAnotherProfileSharesItsName(t *testing.T) {
	home := t.TempDir()
	for _, directory := range []string{".claude/projects", ".claude-/projects"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, directory), 0o700))
	}

	roots, skipped, err := Discover(home, filepath.Join(home, ".claude"))
	require.NoError(t, err)
	require.Empty(t, skipped)
	require.Len(t, roots, 2)
}

func TestDiscoverSkipsAConfigDirectoryWhoseNameIsTaken(t *testing.T) {
	home := t.TempDir()
	elsewhere := filepath.Join(t.TempDir(), ".claude-corp")
	for _, directory := range []string{filepath.Join(home, ".claude-corp", "projects"), filepath.Join(elsewhere, "projects")} {
		require.NoError(t, os.MkdirAll(directory, 0o700))
	}

	roots, skipped, err := Discover(home, elsewhere)
	require.NoError(t, err)
	require.Equal(t, []Root{{Directory: filepath.Join(home, ".claude-corp"), Name: "corp"}}, roots)
	require.Equal(t, []string{"profile " + elsewhere + ": its name corp is taken by " + filepath.Join(home, ".claude-corp")}, skipped)
}

func TestDiscoverIgnoresAConfigDirectoryWithoutProjects(t *testing.T) {
	home := t.TempDir()
	empty := t.TempDir()

	for _, configDirectory := range []string{empty, filepath.Join(empty, "missing")} {
		roots, skipped, err := Discover(home, configDirectory)
		require.NoError(t, err)
		require.Empty(t, skipped)
		require.Empty(t, roots)
	}
}
