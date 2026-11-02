package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const syncSkippedFile = "sync-skipped"

func readSyncSkipped(dataDirectory string) ([]string, error) {
	content, err := os.ReadFile(filepath.Join(dataDirectory, syncSkippedFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(content), "\n"), "\n"), nil
}

func writeSyncSkipped(dataDirectory string, skipped []string) error {
	path := filepath.Join(dataDirectory, syncSkippedFile)
	if len(skipped) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	lines := make([]string, len(skipped))
	for position, file := range skipped {
		lines[position] = strings.ReplaceAll(file, "\n", " ")
	}
	return writeAtomically(path, []byte(strings.Join(lines, "\n")+"\n"))
}

// A reader racing the write sees the old file or the new one, never half of either.
func writeAtomically(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	_, err = temporary.Write(content)
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(temporary.Name(), path)
	}
	if err != nil {
		_ = os.Remove(temporary.Name())
	}
	return err
}
