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
	"syscall"
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
	unlock, _, err := lock(directory, syscall.LOCK_EX)
	return unlock, err
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
		err := filepath.WalkDir(source.Directory, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.Type().IsRegular() || filepath.Ext(path) != ".jsonl" {
				return nil
			}
			relative, err := filepath.Rel(source.Directory, path)
			if err != nil {
				return err
			}
			key := source.Profile + "/" + filepath.ToSlash(relative)
			state, copied, err := syncFile(directory, key, path, manifest[key])
			if err != nil {
				return fmt.Errorf("archive %s: %w", path, err)
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
	unlock, held, err := lock(directory, syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return nil, false, nil
	}
	return unlock, held, err
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

func lock(directory string, how int) (func(), bool, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, false, fmt.Errorf("create %s: %w", directory, err)
	}
	file, err := os.OpenFile(filepath.Join(directory, ".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, fmt.Errorf("open lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), how); err != nil {
		_ = file.Close()
		return nil, false, fmt.Errorf("lock %s: %w", directory, err)
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, true, nil
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

	if state.Offset > 0 {
		var hash string
		hash, err = tailHash(source, state.Offset)
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
	if info.Size() <= state.Offset {
		return state, 0, nil
	}

	chunk := make([]byte, info.Size()-state.Offset)
	if _, err = source.ReadAt(chunk, state.Offset); err != nil && !errors.Is(err, io.EOF) {
		return state, 0, err
	}
	end := bytes.LastIndexByte(chunk, '\n')
	if end < 0 {
		return state, 0, nil
	}
	chunk = chunk[:end+1]

	size, err := appendMember(Path(directory, key, state.Generation), chunk, state.Offset == 0)
	if err != nil {
		return state, 0, err
	}
	state.Offset += int64(len(chunk))
	state.ArchiveSize = size
	state.TailHash, err = tailHash(source, state.Offset)
	if err != nil {
		return state, 0, err
	}
	return state, int64(len(chunk)), nil
}

// tailHash fingerprints the bytes just before offset; a mismatch means the source was rewritten, not appended to.
func tailHash(source *os.File, offset int64) (string, error) {
	start := max(offset-tailWindow, 0)
	window := make([]byte, offset-start)
	if _, err := source.ReadAt(window, start); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	sum := sha256.Sum256(window)
	return hex.EncodeToString(sum[:]), nil
}
