package tail

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinesReturnsWholeShortFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "short.txt")
	require.NoError(t, os.WriteFile(path, []byte("one\ntwo\n"), 0o600))

	lines, err := Lines(path, 5)
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two"}, lines)
}
