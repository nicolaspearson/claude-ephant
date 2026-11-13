package archive

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	manifest, err := loadManifest(directory)
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

func TestSyncArchivesASymlinkedProjectsDirectory(t *testing.T) {
	projects, directory := setup(t)
	real := filepath.Join(t.TempDir(), "real-projects")
	write(t, filepath.Join(real, "-repo", "session.jsonl"), "a\n")
	require.NoError(t, os.Symlink(real, projects))

	summary := sync(t, projects, directory)

	require.Equal(t, Summary{Bytes: 2, Files: 1}, summary)
	require.Equal(t, "a\n", read(t, Path(directory, "claude/-repo/session.jsonl", 1)))
}

func TestSyncSkipsWhatItCannotReadAndArchivesTheRest(t *testing.T) {
	projects, directory := setup(t)
	write(t, filepath.Join(projects, "-good", "session.jsonl"), "a\n")
	unreadable := filepath.Join(projects, "-bad", "session.jsonl")
	write(t, unreadable, "b\n")
	require.NoError(t, os.Chmod(unreadable, 0o000))
	closed := filepath.Join(projects, "-closed")
	write(t, filepath.Join(closed, "session.jsonl"), "c\n")
	require.NoError(t, os.Chmod(closed, 0o000))
	t.Cleanup(func() { _ = os.Chmod(closed, 0o700) })

	summary := sync(t, projects, directory)

	require.Equal(t, 1, summary.Files)
	require.Equal(t, "a\n", read(t, Path(directory, "claude/-good/session.jsonl", 1)))
	require.Len(t, summary.Skipped, 2)
	require.Contains(t, summary.Skipped[0], filepath.Join("-bad", "session.jsonl"), "a file it cannot open")
	require.Contains(t, summary.Skipped[1], "-closed", "a directory it cannot list")
	manifest, err := loadManifest(directory)
	require.NoError(t, err)
	require.Contains(t, manifest, "claude/-good/session.jsonl", "the manifest is saved past a skipped file")
}

func TestSyncSourceHashesTheBytesItArchived(t *testing.T) {
	directory := t.TempDir()
	archived := "{\"a\":1}\n{\"b\":2}\n"
	// The source rewritten in place, same size, after the first read.
	source := &rewrittenMidSync{after: []byte("{\"x\":9}\n{\"y\":8}\n"), before: []byte(archived), readsBefore: 1}

	state, copied, err := syncSource(directory, "claude/-p/session.jsonl", source, int64(len(archived)), FileState{})

	require.NoError(t, err)
	require.Equal(t, int64(len(archived)), copied)
	require.Equal(t, archived, read(t, Path(directory, "claude/-p/session.jsonl", 1)))
	sum := sha256.Sum256([]byte(archived))
	require.Equal(t, hex.EncodeToString(sum[:]), state.TailHash, "the hash is of the bytes archived, not of the rewrite")
}

func TestSyncSourceHashesTheWindowReachingBackPastTheChunk(t *testing.T) {
	directory := t.TempDir()
	earlier := strings.Repeat("{\"line\":\"earlier\"}\n", 300)
	state, _, err := syncSource(directory, "claude/-p/session.jsonl", strings.NewReader(earlier), int64(len(earlier)), FileState{})
	require.NoError(t, err)
	whole := earlier + "{\"line\":\"appended\"}\n"
	// Verified and read before the rewrite, the two reads a sync of an appended source makes.
	source := &rewrittenMidSync{after: []byte(strings.Repeat("x", len(whole))), before: []byte(whole), readsBefore: 2}

	state, _, err = syncSource(directory, "claude/-p/session.jsonl", source, int64(len(whole)), state)

	require.NoError(t, err)
	require.Equal(t, int64(len(whole)), state.Offset)
	sum := sha256.Sum256([]byte(whole[len(whole)-tailWindow:]))
	require.Equal(t, hex.EncodeToString(sum[:]), state.TailHash, "the window runs back past the appended chunk into bytes archived earlier")
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

type rewrittenMidSync struct {
	after       []byte
	before      []byte
	reads       int
	readsBefore int
}

func (r *rewrittenMidSync) ReadAt(buffer []byte, offset int64) (int, error) {
	content := r.after
	if r.reads < r.readsBefore {
		content = r.before
	}
	r.reads++
	copied := copy(buffer, content[offset:])
	if copied < len(buffer) {
		return copied, io.EOF
	}
	return copied, nil
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
