package datadir

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
)

const (
	CheckpointLog = "checkpoints.jsonl"
	TicketLedger  = "ticket-audits.jsonl"

	name       = "claude-ephant"
	noticeFile = "move-notice"
)

var (
	link   = os.Link
	rename = os.Rename
)

func LogPath(file string) (string, error) {
	directory, err := Path()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "logs", file), nil
}

// Migrate moves Linux's old data directory to the XDG one, then the logs from
// ~/.claude/logs into the data directory, and returns the outcome, "" when it
// did nothing. It also writes the outcome to move-notice: a hook's stderr
// reaches no one.
func Migrate() string {
	return migrate(runtime.GOOS)
}

// Path is ELE_HOME, else the platform's data directory; on Linux the old one
// while it exists and the XDG one does not.
func Path() (string, error) {
	return resolve(runtime.GOOS)
}

// TakeNotice returns the directory's move-notice, "" when none, and removes it.
func TakeNotice(directory string) string {
	path := filepath.Join(directory, noticeFile)
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	_ = os.Remove(path)
	return strings.TrimSpace(string(content))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Appends, since one first use can move the directory and both logs, but
// once per line: a failed move retries at every command.
func leaveNotice(directory string, notice string) string {
	path := filepath.Join(directory, noticeFile)
	if content, err := os.ReadFile(path); err == nil && slices.Contains(strings.Split(string(content), "\n"), notice) {
		return notice
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err == nil {
		_, _ = file.WriteString(notice + "\n")
		_ = file.Close()
	}
	return notice
}

func legacyPath(home string) string {
	return filepath.Join(home, "Library", "Application Support", name)
}

// ELE_HOME leaves a notice instead of moving, CHECKPOINT_LOG skips the
// checkpoint log's: a store they point at must not take the user's own logs.
func migrate(goos string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	override := os.Getenv("ELE_HOME") != ""
	var notices []string
	if goos == "linux" && !override {
		if notice := moveDirectory(home); notice != "" {
			notices = append(notices, notice)
		}
	}
	directory, err := resolve(goos)
	if err != nil {
		return strings.Join(notices, "\n")
	}
	for _, file := range []string{CheckpointLog, TicketLedger} {
		if file == CheckpointLog && os.Getenv("CHECKPOINT_LOG") != "" {
			continue
		}
		old := filepath.Join(home, ".claude", "logs", file)
		var notice string
		if override {
			notice = pointAtLog(old, directory, file)
		} else {
			notice = moveLog(old, directory, file)
		}
		if notice != "" {
			notices = append(notices, notice)
		}
	}
	return strings.Join(notices, "\n")
}

// move holds the parent's lock against another first use, and skips the move,
// reporting false, while a sync holds the old directory's archive.
func move(legacy string, xdg string) (bool, error) {
	parent := filepath.Dir(xdg)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return false, err
	}
	directory, err := os.Open(parent)
	if err != nil {
		return false, err
	}
	defer directory.Close()
	if err = syscall.Flock(int(directory.Fd()), syscall.LOCK_EX); err != nil {
		return false, fmt.Errorf("lock %s: %w", parent, err)
	}
	if !moveDue(legacy, xdg) {
		return false, nil
	}
	archiveDirectory := filepath.Join(legacy, "archive", "transcripts")
	if exists(archiveDirectory) {
		unlock, held, lockErr := archive.TryLock(archiveDirectory)
		if lockErr != nil {
			return false, lockErr
		}
		if !held {
			return false, nil
		}
		defer unlock()
	}
	if err = rename(legacy, xdg); err != nil {
		return false, err
	}
	return true, nil
}

func moveDirectory(home string) string {
	legacy, xdg := legacyPath(home), xdgPath(home)
	if !moveDue(legacy, xdg) {
		return ""
	}
	moved, err := move(legacy, xdg)
	if err != nil {
		return leaveNotice(legacy, fmt.Sprintf("ele could not move its data directory from %s to %s: %v; it stays at %s. Move it by hand, or set ELE_HOME to keep it there.", legacy, xdg, err, legacy))
	}
	if !moved {
		return ""
	}
	return leaveNotice(xdg, fmt.Sprintf("ele moved its data directory from %s to %s.", legacy, xdg))
}

func moveDue(legacy string, xdg string) bool {
	return exists(legacy) && !exists(xdg)
}

// Links, not renames: a rename would replace a log another first use moved or a write created.
func moveLog(old string, directory string, file string) string {
	current := filepath.Join(directory, "logs", file)
	if !exists(old) || exists(current) {
		return ""
	}
	err := os.MkdirAll(filepath.Dir(current), 0o700)
	if err == nil {
		err = link(old, current)
	}
	if errors.Is(err, fs.ErrExist) || errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	if err != nil {
		return leaveNotice(directory, fmt.Sprintf("ele could not move %s to %s: %v; ele reads and writes only the new path, so move it by hand before ele next writes there.", old, current, err))
	}
	if err := os.Remove(old); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return leaveNotice(directory, fmt.Sprintf("ele linked %s to %s but could not remove the old name: %v; ele writes to %s from now on.", old, current, err, current))
	}
	return leaveNotice(directory, fmt.Sprintf("ele moved %s to %s.", old, current))
}

func pointAtLog(old string, directory string, file string) string {
	current := filepath.Join(directory, "logs", file)
	if !exists(old) || exists(current) {
		return ""
	}
	return leaveNotice(directory, fmt.Sprintf("ELE_HOME is set, so ele did not move %s; it reads and writes only %s, so move it there by hand to keep its history.", old, current))
}

func resolve(goos string) (string, error) {
	if override := os.Getenv("ELE_HOME"); override != "" {
		absolute, err := filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("resolve ELE_HOME %s: %w", override, err)
		}
		return absolute, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}
	legacy := legacyPath(home)
	if goos != "linux" {
		return legacy, nil
	}
	xdg := xdgPath(home)
	if moveDue(legacy, xdg) {
		return legacy, nil
	}
	return xdg, nil
}

// The XDG spec says to ignore a relative XDG_DATA_HOME.
func xdgPath(home string) string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(dataHome) {
		return filepath.Join(dataHome, name)
	}
	return filepath.Join(home, ".local", "share", name)
}
