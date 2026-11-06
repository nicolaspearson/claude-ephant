package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrentNamesAProfileWhoseNameIsTakenByItsPath(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude-hyve", "projects"), 0o700))
	skippedFolder := filepath.Join(t.TempDir(), ".claude-hyve")
	require.NoError(t, os.MkdirAll(filepath.Join(skippedFolder, "projects"), 0o700))
	emptyFolder := filepath.Join(t.TempDir(), ".claude-hyve")
	require.NoError(t, os.MkdirAll(emptyFolder, 0o700))

	_, skipped, err := Discover(home, skippedFolder)
	require.NoError(t, err)
	require.Len(t, skipped, 1)
	require.Equal(t, skippedFolder, Current(home, skippedFolder))
	_, skipped, err = Discover(home, emptyFolder)
	require.NoError(t, err)
	require.Empty(t, skipped)
	require.Equal(t, emptyFolder, Current(home, emptyFolder))
}

func TestCurrentNamesARelativeProfileAsDiscoverDoes(t *testing.T) {
	scratch := filepath.Join(t.TempDir(), "scratch")
	require.NoError(t, os.MkdirAll(filepath.Join(scratch, "projects"), 0o700))
	t.Chdir(scratch)
	home := t.TempDir()

	roots, _, err := Discover(home, ".")
	require.NoError(t, err)
	require.Len(t, roots, 1)
	require.Equal(t, "scratch", roots[0].Name)
	require.Equal(t, "scratch", Current(home, "."))
}

func TestCurrentNamesASymlinkedProfileAsDiscoverDoes(t *testing.T) {
	home := t.TempDir()
	store := filepath.Join(t.TempDir(), "claude-store")
	require.NoError(t, os.MkdirAll(filepath.Join(store, "projects"), 0o700))
	require.NoError(t, os.Symlink(store, filepath.Join(home, ".claude-hyve")))
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(filepath.Join(home, ".claude-hyve"), link))

	for _, configDirectory := range []string{filepath.Join(home, ".claude-hyve"), link, store} {
		roots, skipped, err := Discover(home, configDirectory)
		require.NoError(t, err)
		require.Empty(t, skipped, configDirectory)
		require.Equal(t, []Root{{Directory: filepath.Join(home, ".claude-hyve"), Name: "hyve"}}, roots, configDirectory)
		require.Equal(t, "hyve", Current(home, configDirectory), configDirectory)
	}
}

func TestCurrentNamesTheProfileFromItsDirectory(t *testing.T) {
	home := t.TempDir()
	require.Equal(t, "claude", Current(home, ""))
	require.Equal(t, "claude", Current(home, "/Users/me/.claude"))
	require.Equal(t, "corp", Current(home, "/Users/me/.claude-corp/"))
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
	require.Equal(t, []Root{{Directory: filepath.Join(home, ".claude"), Name: "claude"}}, roots)
	require.Equal(t, []string{"profile " + filepath.Join(home, ".claude-") + ": its name claude is taken by " + filepath.Join(home, ".claude")}, skipped)
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

func TestDiscoverSkipsAHomeProfileWhoseNameIsTaken(t *testing.T) {
	home := t.TempDir()
	for _, directory := range []string{".claude/projects", ".claude-/projects", ".claude-claude/projects"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, directory), 0o700))
	}

	roots, skipped, err := Discover(home, "")
	require.NoError(t, err)
	require.Equal(t, []Root{{Directory: filepath.Join(home, ".claude"), Name: "claude"}}, roots)
	require.Equal(t, []string{
		"profile " + filepath.Join(home, ".claude-") + ": its name claude is taken by " + filepath.Join(home, ".claude"),
		"profile " + filepath.Join(home, ".claude-claude") + ": its name claude is taken by " + filepath.Join(home, ".claude"),
	}, skipped)
}

func TestDiscoverListsALinkedProfileOnceUnderItsRealPath(t *testing.T) {
	home := t.TempDir()
	for _, directory := range []string{".claude/projects", ".claude-main/projects"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, directory), 0o700))
	}
	require.NoError(t, os.Symlink(filepath.Join(home, ".claude"), filepath.Join(home, ".claude-work")))
	require.NoError(t, os.Symlink(filepath.Join(home, ".claude-main"), filepath.Join(home, ".claude-alias")))

	roots, skipped, err := Discover(home, "")
	require.NoError(t, err)
	require.Empty(t, skipped, "a link is the same profile, not a skip")
	require.Equal(t, []Root{
		{Directory: filepath.Join(home, ".claude"), Name: "claude"},
		{Directory: filepath.Join(home, ".claude-main"), Name: "main"},
	}, roots)
	require.Equal(t, "claude", Current(home, filepath.Join(home, ".claude-work")))
	require.Equal(t, "main", Current(home, filepath.Join(home, ".claude-alias")))
}

func TestDiscoverReadsAHomeHoldingGlobCharacters(t *testing.T) {
	home := filepath.Join(t.TempDir(), "a[b*")
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude-corp", "projects"), 0o700))

	roots, _, err := Discover(home, "")
	require.NoError(t, err)
	require.Equal(t, []Root{{Directory: filepath.Join(home, ".claude-corp"), Name: "corp"}}, roots)

	roots, _, err = Discover(filepath.Join(t.TempDir(), "missing"), "")
	require.NoError(t, err)
	require.Empty(t, roots)
}
