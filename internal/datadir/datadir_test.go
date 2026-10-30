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

func TestMigrateMovesLogsOnce(t *testing.T) {
	home := setHome(t)
	old := seedLogs(t, home)
	data := filepath.Join(home, "Library", "Application Support", "claude-ephant")

	notice := migrate("darwin")

	require.Equal(t, "ele moved "+old[0]+" to "+filepath.Join(data, "logs", CheckpointLog)+".\nele moved "+old[1]+" to "+filepath.Join(data, "logs", TicketLedger)+".", notice)
	for i, file := range []string{CheckpointLog, TicketLedger} {
		require.NoFileExists(t, old[i])
		content, err := os.ReadFile(filepath.Join(data, "logs", file))
		require.NoError(t, err)
		require.Equal(t, file+"\n", string(content))
	}
	require.Equal(t, notice, TakeNotice(data))
	require.Empty(t, migrate("darwin"))
}

func TestMigrateMovesLogsAfterTheDirectory(t *testing.T) {
	home := setHome(t)
	legacy := seedLegacy(t, home)
	old := seedLogs(t, home)
	xdg := filepath.Join(home, ".local", "share", "claude-ephant")

	notice := migrate("linux")

	require.Equal(t, "ele moved its data directory from "+legacy+" to "+xdg+".\nele moved "+old[0]+" to "+filepath.Join(xdg, "logs", CheckpointLog)+".\nele moved "+old[1]+" to "+filepath.Join(xdg, "logs", TicketLedger)+".", notice)
	require.FileExists(t, filepath.Join(xdg, "logs", CheckpointLog))
	require.FileExists(t, filepath.Join(xdg, "logs", TicketLedger))
	require.Equal(t, notice, TakeNotice(xdg))
}

func TestMigrateLeavesBothLogsWhenNewExists(t *testing.T) {
	home := setHome(t)
	old := seedLogs(t, home)
	current := filepath.Join(home, "Library", "Application Support", "claude-ephant", "logs", CheckpointLog)
	require.NoError(t, os.MkdirAll(filepath.Dir(current), 0o700))
	require.NoError(t, os.WriteFile(current, []byte("new\n"), 0o600))

	notice := migrate("darwin")

	require.NotContains(t, notice, old[0])
	require.FileExists(t, old[0])
	content, err := os.ReadFile(current)
	require.NoError(t, err)
	require.Equal(t, "new\n", string(content))
	require.Contains(t, notice, old[1])
}

func TestMigrateLeavesLogsToTheirOverrides(t *testing.T) {
	home := setHome(t)
	old := seedLogs(t, home)
	t.Setenv("CHECKPOINT_LOG", filepath.Join(t.TempDir(), CheckpointLog))

	notice := migrate("darwin")

	require.NotContains(t, notice, old[0])
	require.FileExists(t, old[0])
	require.NoFileExists(t, old[1])

	old = seedLogs(t, home)
	override := t.TempDir()
	t.Setenv("ELE_HOME", override)
	notice = migrate("darwin")
	require.Equal(t, "ELE_HOME is set, so ele did not move "+old[1]+"; it reads and writes only "+filepath.Join(override, "logs", TicketLedger)+", so move it there by hand to keep its history.", notice)
	require.FileExists(t, old[1])
	require.NoDirExists(t, filepath.Join(override, "logs"))
	require.Equal(t, notice, TakeNotice(override))
}

func TestMigrateConcurrentFirstUsesMoveEachLogOnce(t *testing.T) {
	home := setHome(t)
	seedLogs(t, home)
	notices := make([]string, 32)
	var group sync.WaitGroup
	for i := range notices {
		group.Go(func() {
			notices[i] = migrate("darwin")
		})
	}
	group.Wait()

	require.Equal(t, 2, strings.Count(strings.Join(notices, "\n"), "ele moved "))
	content, err := os.ReadFile(filepath.Join(home, "Library", "Application Support", "claude-ephant", "logs", CheckpointLog))
	require.NoError(t, err)
	require.Equal(t, CheckpointLog+"\n", string(content))
}

func TestMigrateFailedLinkKeepsOldLog(t *testing.T) {
	home := setHome(t)
	old := seedLogs(t, home)
	data := filepath.Join(home, "Library", "Application Support", "claude-ephant")
	original := link
	link = func(source string, target string) error {
		return &os.LinkError{Op: "link", Old: source, New: target, Err: syscall.EXDEV}
	}
	t.Cleanup(func() { link = original })

	notice := migrate("darwin")

	require.Contains(t, notice, "ele could not move "+old[0]+" to "+filepath.Join(data, "logs", CheckpointLog))
	require.Contains(t, notice, "cross-device link")
	require.Contains(t, notice, "move it by hand")
	require.FileExists(t, old[0])
	require.Equal(t, notice, migrate("darwin"), "every command retries a failed link")
	require.Equal(t, notice, TakeNotice(data), "a retried failure leaves its notice once")
}

func TestLogPathIsUnderTheDataDirectory(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("ELE_HOME", directory)

	path, err := LogPath(TicketLedger)

	require.NoError(t, err)
	require.Equal(t, filepath.Join(directory, "logs", TicketLedger), path)
}

func seedLegacy(t *testing.T, home string) string {
	t.Helper()
	legacy := filepath.Join(home, "Library", "Application Support", "claude-ephant")
	require.NoError(t, os.MkdirAll(legacy, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "config.json"), []byte("{}\n"), 0o600))
	return legacy
}

func seedLogs(t *testing.T, home string) []string {
	t.Helper()
	var paths []string
	for _, file := range []string{CheckpointLog, TicketLedger} {
		path := filepath.Join(home, ".claude", "logs", file)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(file+"\n"), 0o600))
		paths = append(paths, path)
	}
	return paths
}

func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("CHECKPOINT_LOG", "")
	t.Setenv("ELE_HOME", "")
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	return home
}
