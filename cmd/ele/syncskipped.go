package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/atomicfile"
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
	return atomicfile.Write(path, []byte(strings.Join(lines, "\n")+"\n"))
}
