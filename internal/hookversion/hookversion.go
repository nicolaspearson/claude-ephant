package hookversion

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

const Directory = "hook-versions"

type Log struct {
	directory string
	sessions  map[string][]record
}

type record struct {
	at      time.Time
	version string
}

func Open(directory string) *Log {
	return &Log{directory: directory, sessions: map[string][]record{}}
}

// Record appends a line only when the session's last one names another version, so a file holds one line per change of build.
func Record(directory string, session string, version string, now time.Time) error {
	if session == "" || !notes.ValidSession(session) {
		return fmt.Errorf("session %q is not a session id", session)
	}
	path := filepath.Join(directory, session)
	records, err := read(path)
	if err != nil {
		return err
	}
	if len(records) > 0 && records[len(records)-1].version == version {
		return nil
	}
	if err = os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(file, "%s\t%s\n", now.UTC().Format(time.RFC3339Nano), version)
	return errors.Join(err, file.Close())
}

func (l *Log) At(session string, timestamp string) string {
	at, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil || session == "" || !notes.ValidSession(session) {
		return ""
	}
	records, loaded := l.sessions[session]
	if !loaded {
		// An unreadable file leaves the session's events without a version rather than failing the sync.
		records, _ = read(filepath.Join(l.directory, session))
		l.sessions[session] = records
	}
	version := ""
	for _, recorded := range records {
		if recorded.at.After(at) {
			break
		}
		version = recorded.version
	}
	return version
}

func read(path string) ([]record, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var records []record
	lines := bufio.NewScanner(bytes.NewReader(content))
	for lines.Scan() {
		stamp, version, found := strings.Cut(lines.Text(), "\t")
		at, parseErr := time.Parse(time.RFC3339Nano, stamp)
		if !found || parseErr != nil || version == "" {
			continue
		}
		records = append(records, record{at: at, version: version})
	}
	return records, lines.Err()
}
