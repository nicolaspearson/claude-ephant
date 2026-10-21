package index

import (
	"context"
	"database/sql"
	"errors"
)

// Reset drops and recreates the derived tables, so a rebuild picks up a schema change; settings are kept.
func (x *Index) Reset(ctx context.Context) error {
	tx, err := x.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, statement := range []string{
		`DROP TABLE IF EXISTS archive_files`,
		`DROP TABLE IF EXISTS claude_mem_tables`,
		`DROP TABLE IF EXISTS directories`,
		`DROP TABLE IF EXISTS entries`,
		`DROP TABLE IF EXISTS entries_fts`,
		`DROP TABLE IF EXISTS events`,
		`DROP TABLE IF EXISTS notes`,
		schema,
	} {
		if _, err = tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	x.problems = nil
	return nil
}

func (x *Index) SetSetting(ctx context.Context, key string, value string) error {
	_, err := x.db.ExecContext(ctx, `INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// Setting returns "" for a key never set.
func (x *Index) Setting(ctx context.Context, key string) (string, error) {
	var value string
	err := x.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}
