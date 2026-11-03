package checkpoint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderAssistantKeepsAMessageWhoseToolInputIsNotAString(t *testing.T) {
	rendered := renderAssistant([]byte(`[{"type":"text","text":"Ready to commit."},{"type":"tool_use","name":"mcp__runner__run","input":{"command":["make","test"]}}]`))

	require.Equal(t, "\x1eTEXT\nReady to commit.\n\x1eTOOL mcp__runner__run ", rendered)
}

func TestRepositoryPathNamesAFileThroughASymlinkedCheckoutAsGitDoes(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal"), 0o700))
	link := filepath.Join(t.TempDir(), "checkout")
	require.NoError(t, os.Symlink(root, link))

	require.Equal(t, "internal/new.go", repositoryPath(root, filepath.Join(link, "internal", "new.go")))
	require.Equal(t, "internal/new.go", repositoryPath(root, filepath.Join(root, "internal", "new.go")))
	require.Equal(t, "/elsewhere/new.go", repositoryPath(root, "/elsewhere/new.go"))
}
