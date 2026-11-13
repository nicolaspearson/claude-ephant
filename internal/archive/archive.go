package archive

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/atomicfile"
)

const tailWindow = 4096

var generationSuffix = regexp.MustCompile(`(\.g\d+)?\.jsonl\.gz$`)

type FileState struct {
	ArchiveSize int64  `json:"archiveSize"`
	Generation  int    `json:"generation"`
	Offset      int64  `json:"offset"`
	TailHash    string `json:"tailHash"`
}

// Manifest is keyed by "<profile>/<path within projects/>", slash-separated.
type Manifest map[string]FileState

type Source struct {
	Directory string
	Profile   string
}

type Summary struct {
	Bytes       int64
	Files       int
	Generations int
	Skipped     []string
}

func LoadManifest(directory string) (Manifest, error) {
	manifest := Manifest{}
	content, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return manifest, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return manifest, nil
}

// Lock is advisory and blocks until released.
func Lock(directory string) (func(), error) {
	return atomicfile.Lock(filepath.Join(directory, ".lock"))
}

func Open(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	reader, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return readCloser{Reader: reader, close: func() error {
		_ = reader.Close()
		return file.Close()
	}}, nil
}

// Path keeps the source name for generation 1; later ones are <name>.g<n>.jsonl.gz.
func Path(directory string, key string, generation int) string {
	base := filepath.Join(directory, filepath.FromSlash(key))
	if generation > 1 {
		base = fmt.Sprintf("%s.g%d.jsonl", strings.TrimSuffix(base, ".jsonl"), generation)
	}
	return base + ".gz"
}

// SubagentDirectory is where a transcript's subagent files are, by the transcript's own name, not its lines' session:
// a resumed transcript's lines can carry another session id.
func SubagentDirectory(path string) string {
	return filepath.Join(generationSuffix.ReplaceAllString(path, ""), "subagents")
}

// Sync expects the caller to hold the lock.
func Sync(sources []Source, directory string) (Summary, error) {
	manifest, err := LoadManifest(directory)
	if err != nil {
		return Summary{}, err
	}
	if err := rollback(directory, manifest); err != nil {
		return Summary{}, err
	}

	var summary Summary
	for _, source := range sources {
		// WalkDir does not descend a symlinked root.
		root, err := filepath.EvalSymlinks(source.Directory)
		if err != nil {
			summary.Skipped = append(summary.Skipped, err.Error())
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				summary.Skipped = append(summary.Skipped, err.Error())
				return nil
			}
			if !entry.Type().IsRegular() || filepath.Ext(path) != ".jsonl" {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			key := source.Profile + "/" + filepath.ToSlash(relative)
			state, copied, err := syncFile(directory, key, path, manifest[key])
			if err != nil {
				summary.Skipped = append(summary.Skipped, fmt.Sprintf("archive %s: %v", path, err))
				return nil
			}
			if copied == 0 {
				return nil
			}
			if previous := manifest[key].Generation; previous != 0 && state.Generation != previous {
				summary.Generations++
			}
			manifest[key] = state
			summary.Bytes += copied
			summary.Files++
			if summary.Files%100 == 0 {
				return saveManifest(directory, manifest)
			}
			return nil
		})
		if err != nil {
			return summary, err
		}
	}
	return summary, saveManifest(directory, manifest)
}

// TryLock reports false, holding nothing, when another holder has the lock.
func TryLock(directory string) (func(), bool, error) {
	return atomicfile.TryLock(filepath.Join(directory, ".lock"))
}

type readCloser struct {
	io.Reader
	close func() error
}

func (r readCloser) Close() error {
	return r.close()
}

// appendMember writes chunk as one gzip member. A new file goes through a temporary name, so a crash never leaves a partial one under its real name.
func appendMember(path string, chunk []byte, fresh bool) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return 0, err
	}
	target := path
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if fresh {
		target = path + ".tmp"
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}
	file, err := os.OpenFile(target, flags, 0o600)
	if err != nil {
		return 0, err
	}
	writer, err := gzip.NewWriterLevel(file, gzip.BestSpeed)
	if err != nil {
		_ = file.Close()
		return 0, err
	}
	_, err = writer.Write(chunk)
	if err == nil {
		err = writer.Close()
	}
	if err == nil {
		err = file.Sync()
	}
	var written fs.FileInfo
	if err == nil {
		written, err = file.Stat()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil && fresh {
		err = os.Rename(target, path)
	}
	if err != nil {
		if fresh {
			_ = os.Remove(target)
		}
		return 0, err
	}
	return written.Size(), nil
}

// rollback truncates appends to recorded files that a crash left past the saved manifest.
func rollback(directory string, manifest Manifest) error {
	for key, state := range manifest {
		path := Path(directory, key, state.Generation)
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("manifest names %s but its archive file is missing or unreadable (%w); restore it from a backup. Removing %q from manifest.json instead re-archives the source from scratch and overwrites its first generation", path, err, key)
		}
		if info.Size() > state.ArchiveSize {
			if err := os.Truncate(path, state.ArchiveSize); err != nil {
				return fmt.Errorf("roll back %s: %w", path, err)
			}
		}
	}
	return nil
}

func saveManifest(directory string, manifest Manifest) error {
	content, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	path := filepath.Join(directory, "manifest.json")
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	_, err = file.Write(content)
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(temporary, path)
	}
	if err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func syncFile(directory string, key string, sourcePath string, state FileState) (FileState, int64, error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return state, 0, err
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return state, 0, err
	}
	return syncSource(directory, key, source, info.Size(), state)
}

func syncSource(directory string, key string, source io.ReaderAt, size int64, state FileState) (FileState, int64, error) {
	if state.Offset > 0 {
		hash, err := tailHash(source, state.Offset)
		if err != nil {
			return state, 0, err
		}
		if hash != state.TailHash {
			state = FileState{Generation: state.Generation + 1}
		}
	}
	if state.Generation == 0 {
		state.Generation = 1
	}
	if size <= state.Offset {
		return state, 0, nil
	}

	// One read covers the hash window and the chunk, so the hash recorded is of the bytes archived, even if the source
	// is rewritten mid-sync.
	start := max(state.Offset-tailWindow, 0)
	read := make([]byte, size-start)
	if _, err := source.ReadAt(read, start); err != nil && !errors.Is(err, io.EOF) {
		return state, 0, err
	}
	end := bytes.LastIndexByte(read[state.Offset-start:], '\n')
	if end < 0 {
		return state, 0, nil
	}
	through := state.Offset - start + int64(end) + 1
	chunk := read[state.Offset-start : through]

	archived, err := appendMember(Path(directory, key, state.Generation), chunk, state.Offset == 0)
	if err != nil {
		return state, 0, err
	}
	state.Offset += int64(len(chunk))
	state.ArchiveSize = archived
	state.TailHash = windowHash(read[:through])
	return state, int64(len(chunk)), nil
}

// tailHash fingerprints the bytes just before offset; a mismatch means the source was rewritten, not appended to.
func tailHash(source io.ReaderAt, offset int64) (string, error) {
	start := max(offset-tailWindow, 0)
	window := make([]byte, offset-start)
	if _, err := source.ReadAt(window, start); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return windowHash(window), nil
}

func windowHash(content []byte) string {
	sum := sha256.Sum256(content[max(len(content)-tailWindow, 0):])
	return hex.EncodeToString(sum[:])
}
