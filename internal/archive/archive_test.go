package archive

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncAppendsCompleteLinesAcrossRuns(t *testing.T) {
	projects, directory := setup(t)
	source := filepath.Join(projects, "-repo", "session.jsonl")
	write(t, source, "a\nb\npart")

	summary := sync(t, projects, directory)
	require.Equal(t, Summary{Bytes: 4, Files: 1}, summary)
	require.Equal(t, "a\nb\n", read(t, Path(directory, "claude/-repo/session.jsonl", 1)))

	appendTo(t, source, "ial\nc\n")
	summary = sync(t, projects, directory)
	require.Equal(t, Summary{Bytes: 10, Files: 1}, summary)
	require.Equal(t, "a\nb\npartial\nc\n", read(t, Path(directory, "claude/-repo/session.jsonl", 1)))

	require.Equal(t, Summary{}, sync(t, projects, directory))
}

func TestSyncArchivesSubagentTranscriptsAtTheirPath(t *testing.T) {
	projects, directory := setup(t)
	write(t, filepath.Join(projects, "-repo", "session", "subagents", "agent-1.jsonl"), "x\n")
	write(t, filepath.Join(projects, "-repo", "notes.txt"), "ignored\n")

	sync(t, projects, directory)
	require.Equal(t, "x\n", read(t, Path(directory, "claude/-repo/session/subagents/agent-1.jsonl", 1)))
	manifest, err := LoadManifest(directory)
	require.NoError(t, err)
	require.Len(t, manifest, 1)
}

func TestSyncRollsBackAppendsTheManifestNeverRecorded(t *testing.T) {
	projects, directory := setup(t)
	source := filepath.Join(projects, "-repo", "session.jsonl")
	write(t, source, "a\n")
	sync(t, projects, directory)

	archived := Path(directory, "claude/-repo/session.jsonl", 1)
	file, err := os.OpenFile(archived, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	writer := gzip.NewWriter(file)
	_, err = writer.Write([]byte("crashed\n"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())

	appendTo(t, source, "b\n")
	sync(t, projects, directory)
	require.Equal(t, "a\nb\n", read(t, archived))
}

func TestSyncStartsNewGenerationWhenSourceIsRewritten(t *testing.T) {
	projects, directory := setup(t)
	source := filepath.Join(projects, "-repo", "session.jsonl")
	write(t, source, "a\nb\n")
	sync(t, projects, directory)

	write(t, source, "x\ny\nz\n")
	summary := sync(t, projects, directory)
	require.Equal(t, Summary{Bytes: 6, Files: 1, Generations: 1}, summary)
	require.Equal(t, "a\nb\n", read(t, Path(directory, "claude/-repo/session.jsonl", 1)))
	require.Equal(t, "x\ny\nz\n", read(t, Path(directory, "claude/-repo/session.jsonl", 2)))
	require.Equal(t, filepath.Join(directory, "claude", "-repo", "session.g2.jsonl.gz"), Path(directory, "claude/-repo/session.jsonl", 2))
}

func TestSyncStartsNewGenerationWhenSourceShrinks(t *testing.T) {
	projects, directory := setup(t)
	source := filepath.Join(projects, "-repo", "session.jsonl")
	write(t, source, "a\nb\n")
	sync(t, projects, directory)

	write(t, source, "c\n")
	require.Equal(t, Summary{Bytes: 2, Files: 1, Generations: 1}, sync(t, projects, directory))
	require.Equal(t, "c\n", read(t, Path(directory, "claude/-repo/session.jsonl", 2)))
}

func TestTryLockReportsAHeldLock(t *testing.T) {
	directory := t.TempDir()
	unlock, err := Lock(directory)
	require.NoError(t, err)

	_, held, err := TryLock(directory)
	require.NoError(t, err)
	require.False(t, held)

	unlock()
	release, held, err := TryLock(directory)
	require.NoError(t, err)
	require.True(t, held)
	release()
}

func appendTo(t *testing.T, path string, content string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = file.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, file.Close())
}

func read(t *testing.T, path string) string {
	t.Helper()
	reader, err := Open(path)
	require.NoError(t, err)
	defer reader.Close()
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(content)
}

func setup(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	return filepath.Join(root, "projects"), filepath.Join(root, "archive")
}

func sync(t *testing.T, projects string, directory string) Summary {
	t.Helper()
	unlock, err := Lock(directory)
	require.NoError(t, err)
	defer unlock()
	summary, err := Sync([]Source{{Directory: projects, Profile: "claude"}}, directory)
	require.NoError(t, err)
	return summary
}

func write(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
