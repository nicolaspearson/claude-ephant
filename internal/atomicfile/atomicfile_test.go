package atomicfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTryLockReportsAHeldLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", ".lock")
	unlock, err := Lock(path)
	require.NoError(t, err)

	_, held, err := TryLock(path)
	require.NoError(t, err)
	require.False(t, held)

	unlock()
	unlockAgain, held, err := TryLock(path)
	require.NoError(t, err)
	require.True(t, held)
	unlockAgain()
}

func TestWriteReplacesTheFileRatherThanWritingIntoIt(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "last-sync")
	require.NoError(t, os.WriteFile(path, []byte("old\n"), 0o600))
	// A reader holding the old file, as a session start racing a sync does.
	require.NoError(t, os.Link(path, filepath.Join(directory, "held")))

	require.NoError(t, Write(path, []byte("new\n")))

	held, err := os.ReadFile(filepath.Join(directory, "held"))
	require.NoError(t, err)
	require.Equal(t, "old\n", string(held), "written in place, the holder would read the new or half of it")
	written, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "new\n", string(written))
	leftovers, err := filepath.Glob(filepath.Join(directory, "*.tmp"))
	require.NoError(t, err)
	require.Empty(t, leftovers)
}
