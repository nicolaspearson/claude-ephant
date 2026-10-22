package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/store"
)

var backupParts = []string{"archive", "config.json", "forgotten", "mining", "notes", "owner-map.json"}

func backup(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("backup", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() != 1 {
		return rejectCall(flags, err, stdout, stderr)
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	target, err := backupTarget(flags.Arg(0), dataDirectory)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	sizes, err := copyDataDirectory(ctx, dataDirectory, target)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v; %s is incomplete\n", err, target)
		return 1
	}
	fmt.Fprintf(stdout, "Backed up to %s: %s.\n", target, strings.Join(sizes, ", "))
	return 0
}

func backupTarget(path string, dataDirectory string) (string, error) {
	target, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := resolveExisting(target)
	if err != nil {
		return "", err
	}
	data, err := resolveExisting(dataDirectory)
	if err != nil {
		return "", err
	}
	if relative, relativeErr := filepath.Rel(data, resolved); relativeErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is inside the data directory %s", target, dataDirectory)
	}
	for directory := resolved; ; directory = filepath.Dir(directory) {
		if _, statErr := os.Lstat(filepath.Join(directory, ".git")); statErr == nil {
			return "", fmt.Errorf("%s is inside the git checkout %s; data never lives in a repository", target, directory)
		}
		if parent := filepath.Dir(directory); parent == directory {
			break
		}
	}
	entries, err := os.ReadDir(target)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if len(entries) > 0 {
		return "", fmt.Errorf("%s is not empty; back up into a new or empty directory", target)
	}
	return target, nil
}

// Holds the archive lock, and the mining cache's when it exists, for the whole copy: no sync or judgment writes mid-copy.
func copyDataDirectory(ctx context.Context, dataDirectory string, target string) ([]string, error) {
	unlockArchive, err := archive.Lock(filepath.Join(dataDirectory, "archive", "transcripts"))
	if err != nil {
		return nil, err
	}
	defer unlockArchive()
	if cache := filepath.Join(dataDirectory, "mining", "cache"); directoryExists(cache) {
		unlockMining, err := archive.Lock(cache)
		if err != nil {
			return nil, err
		}
		defer unlockMining()
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		return nil, err
	}
	var sizes []string
	for _, part := range backupParts {
		source := filepath.Join(dataDirectory, part)
		if _, statErr := os.Lstat(source); errors.Is(statErr, fs.ErrNotExist) {
			continue
		}
		size, err := copyTree(source, filepath.Join(target, part))
		if err != nil {
			return nil, err
		}
		sizes = append(sizes, fmt.Sprintf("%s %s", part, humanBytes(size)))
	}
	index := filepath.Join(dataDirectory, "ele.db")
	if _, err := os.Stat(index); err == nil {
		db, err := store.Open(index)
		if err != nil {
			return nil, err
		}
		defer db.Close()
		copied := filepath.Join(target, "ele.db")
		if _, vacuumErr := db.ExecContext(ctx, `VACUUM INTO ?`, copied); vacuumErr != nil {
			return nil, fmt.Errorf("copy the index: %w", vacuumErr)
		}
		info, err := os.Stat(copied)
		if err != nil {
			return nil, err
		}
		sizes = append(sizes, "index "+humanBytes(info.Size()))
	}
	return sizes, nil
}

func copyTree(source string, target string) (int64, error) {
	var total int64
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		switch {
		case entry.IsDir():
			return os.MkdirAll(destination, 0o700)
		// A .tmp is an unlocked write in flight, renamed away mid-copy.
		case entry.Name() == ".lock", strings.HasSuffix(entry.Name(), ".tmp"):
			return nil
		case !entry.Type().IsRegular():
			return fmt.Errorf("%s is not a regular file", path)
		}
		size, err := copyFile(path, destination)
		total += size
		return err
	})
	return total, err
}

func copyFile(source string, target string) (int64, error) {
	reader, err := os.Open(source)
	if err != nil {
		return 0, err
	}
	defer reader.Close()
	writer, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, err
	}
	size, err := io.Copy(writer, reader)
	if err == nil {
		err = writer.Sync()
	}
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	return size, err
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// A target not yet created compares by its real parent.
func resolveExisting(path string) (string, error) {
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			return filepath.Join(append([]string{resolved}, missing...)...), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", err
		}
		missing = append([]string{filepath.Base(path)}, missing...)
		path = parent
	}
}
