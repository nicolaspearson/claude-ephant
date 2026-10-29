package datadir

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
)

const (
	name       = "claude-ephant"
	noticeFile = "move-notice"
)

var rename = os.Rename

// Migrate moves Linux's old data directory to the XDG one and returns the
// outcome, "" when it did nothing. It also writes the outcome to move-notice:
// a hook's stderr reaches no one.
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

func leaveNotice(directory string, notice string) string {
	_ = os.WriteFile(filepath.Join(directory, noticeFile), []byte(notice+"\n"), 0o600)
	return notice
}

func legacyPath(home string) string {
	return filepath.Join(home, "Library", "Application Support", name)
}

func migrate(goos string) string {
	if goos != "linux" || os.Getenv("ELE_HOME") != "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
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

func moveDue(legacy string, xdg string) bool {
	return exists(legacy) && !exists(xdg)
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
