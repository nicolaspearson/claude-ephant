package datadir

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathDefaultsToApplicationSupport(t *testing.T) {
	home := setHome(t)

	path, err := resolve("darwin")
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

func TestPathOnLinuxUsesXDGDataHome(t *testing.T) {
	setHome(t)
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	path, err := resolve("linux")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dataHome, "claude-ephant"), path)
}

func TestPathOnLinuxDefaultsToLocalShare(t *testing.T) {
	home := setHome(t)

	path, err := resolve("linux")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".local", "share", "claude-ephant"), path)
}

func TestPathOnLinuxIgnoresRelativeXDGDataHome(t *testing.T) {
	home := setHome(t)
	t.Setenv("XDG_DATA_HOME", "relative")

	path, err := resolve("linux")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".local", "share", "claude-ephant"), path)
}

func TestPathOnLinuxKeepsOldDirectoryUntilMoved(t *testing.T) {
	home := setHome(t)
	legacy := seedLegacy(t, home)

	path, err := resolve("linux")
	require.NoError(t, err)
	require.Equal(t, legacy, path)
}

func TestPathOverrideWinsOnLinux(t *testing.T) {
	home := setHome(t)
	seedLegacy(t, home)
	override := t.TempDir()
	t.Setenv("ELE_HOME", override)

	path, err := resolve("linux")
	require.NoError(t, err)
	require.Equal(t, override, path)
	require.Empty(t, migrate("linux"))
	require.DirExists(t, filepath.Join(home, "Library", "Application Support", "claude-ephant"))
}

func TestMigrateMovesOldDirectoryOnce(t *testing.T) {
	home := setHome(t)
	legacy := seedLegacy(t, home)
	xdg := filepath.Join(home, ".local", "share", "claude-ephant")

	notice := migrate("linux")
	require.Equal(t, "ele moved its data directory from "+legacy+" to "+xdg+".", notice)
	require.NoDirExists(t, legacy)
	require.FileExists(t, filepath.Join(xdg, "config.json"))
	path, err := resolve("linux")
	require.NoError(t, err)
	require.Equal(t, xdg, path)

	require.Empty(t, migrate("linux"))
}

func TestMigrateLeavesNoticeForSessionStart(t *testing.T) {
	home := setHome(t)
	legacy := seedLegacy(t, home)
	xdg := filepath.Join(home, ".local", "share", "claude-ephant")

	notice := migrate("linux")
	require.NotEmpty(t, notice)

	path, err := resolve("linux")
	require.NoError(t, err)
	require.Equal(t, notice, TakeNotice(path))
	require.Empty(t, TakeNotice(path))
	require.NoFileExists(t, filepath.Join(xdg, "move-notice"))
	require.NoDirExists(t, legacy)
}

func TestMigrateLeavesBothWhenNewExists(t *testing.T) {
	home := setHome(t)
	legacy := seedLegacy(t, home)
	xdg := filepath.Join(home, ".local", "share", "claude-ephant")
	require.NoError(t, os.MkdirAll(xdg, 0o700))

	require.Empty(t, migrate("linux"))
	require.FileExists(t, filepath.Join(legacy, "config.json"))
	require.NoFileExists(t, filepath.Join(xdg, "config.json"))
	path, err := resolve("linux")
	require.NoError(t, err)
	require.Equal(t, xdg, path)
}

func TestMigrateConcurrentFirstUsesLeaveOneStore(t *testing.T) {
	home := setHome(t)
	legacy := seedLegacy(t, home)
	xdg := filepath.Join(home, ".local", "share", "claude-ephant")

	notices := make([]string, 32)
	var group sync.WaitGroup
	for i := range notices {
		group.Go(func() {
			notices[i] = migrate("linux")
		})
	}
	group.Wait()

	moved := 0
	for _, notice := range notices {
		if notice != "" {
			require.True(t, strings.HasPrefix(notice, "ele moved its data directory"), notice)
			moved++
		}
	}
	require.Equal(t, 1, moved)
	require.NoDirExists(t, legacy)
	require.FileExists(t, filepath.Join(xdg, "config.json"))
}

func TestMigrateSkipsWhileSyncHoldsArchive(t *testing.T) {
	home := setHome(t)
	legacy := seedLegacy(t, home)
	lockDirectory := filepath.Join(legacy, "archive", "transcripts")
	require.NoError(t, os.MkdirAll(lockDirectory, 0o700))
	file, err := os.OpenFile(filepath.Join(lockDirectory, ".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	require.NoError(t, err)
	defer file.Close()
	require.NoError(t, syscall.Flock(int(file.Fd()), syscall.LOCK_EX))

	require.Empty(t, migrate("linux"))
	require.FileExists(t, filepath.Join(legacy, "config.json"))
	path, err := resolve("linux")
	require.NoError(t, err)
	require.Equal(t, legacy, path)
}

func TestMigrateFailedRenameKeepsOldDirectory(t *testing.T) {
	home := setHome(t)
	legacy := seedLegacy(t, home)
	xdg := filepath.Join(home, ".local", "share", "claude-ephant")
	original := rename
	rename = func(string, string) error {
		return &os.LinkError{Op: "rename", Old: legacy, New: xdg, Err: syscall.EXDEV}
	}
	t.Cleanup(func() { rename = original })

	notice := migrate("linux")
	require.Contains(t, notice, "ele could not move its data directory from "+legacy+" to "+xdg)
	require.Contains(t, notice, "cross-device link")
	require.Contains(t, notice, "it stays at "+legacy)
	require.FileExists(t, filepath.Join(legacy, "config.json"))
	path, err := resolve("linux")
	require.NoError(t, err)
	require.Equal(t, legacy, path)
	require.Equal(t, notice, TakeNotice(path))
}

func TestMigrateLeavesMacOSAlone(t *testing.T) {
	home := setHome(t)
	legacy := seedLegacy(t, home)

	require.Empty(t, migrate("darwin"))
	require.FileExists(t, filepath.Join(legacy, "config.json"))
	require.NoDirExists(t, filepath.Join(home, ".local", "share", "claude-ephant"))
}

func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("ELE_HOME", "")
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	return home
}

func seedLegacy(t *testing.T, home string) string {
	t.Helper()
	legacy := filepath.Join(home, "Library", "Application Support", "claude-ephant")
	require.NoError(t, os.MkdirAll(legacy, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "config.json"), []byte("{}\n"), 0o600))
	return legacy
}
