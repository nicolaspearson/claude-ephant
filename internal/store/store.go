package store

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	if err := private(path); err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return open(path, "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
}

// OpenReadOnly expects no writer and an empty -wal: immutable=1 skips locking and ignores the WAL.
func OpenReadOnly(path string) (*sql.DB, error) {
	return open(path, "immutable=1&mode=ro")
}

func open(path string, query string) (*sql.DB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	dsn := (&url.URL{
		Scheme:   "file",
		Path:     absolute,
		RawQuery: query,
	}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return db, nil
}

// SQLite creates -wal and -shm with the database's mode, so a 0600 database
// keeps all three private whatever mode its directory has. A refused chmod, as
// on a file another user owns, is passed over: failing it fails every hook.
func private(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0o600)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	for _, suffix := range []string{"", "-shm", "-wal"} {
		_ = os.Chmod(path+suffix, 0o600)
	}
	return nil
}
