package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

const KindNote = "note"

type noteFile struct {
	fingerprint string
	path        string
}

// UpdateNotes drops notes whose files are gone before indexing changed ones, so a note moved between folders is not mistaken for a duplicate.
func (x *Index) UpdateNotes(ctx context.Context, directory string) (Summary, error) {
	files, err := noteFiles(directory)
	if err != nil {
		return Summary{}, fmt.Errorf("index notes: %w", err)
	}
	recorded, err := x.recordedNotes(ctx)
	if err != nil {
		return Summary{}, err
	}
	for source := range recorded {
		if _, ok := files[source]; !ok {
			if err = x.dropNote(ctx, source); err != nil {
				return Summary{}, err
			}
		}
	}

	sources := make([]string, 0, len(files))
	for source := range files {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	var summary Summary
	for _, source := range sources {
		file := files[source]
		if recorded[source] == file.fingerprint {
			continue
		}
		content, readErr := os.ReadFile(file.path)
		if readErr != nil {
			return summary, fmt.Errorf("index %s: %w", source, readErr)
		}
		note, parseErr := notes.Parse(source, content)
		if parseErr == nil {
			parseErr = x.indexNote(ctx, note, source, file.fingerprint)
		}
		switch {
		case errors.Is(parseErr, notes.ErrInvalid):
			summary.Problems = append(summary.Problems, parseErr.Error())
			if err = x.dropNote(ctx, source); err != nil {
				return summary, err
			}
		case parseErr != nil:
			return summary, fmt.Errorf("index %s: %w", source, parseErr)
		default:
			summary.Entries++
		}
	}
	return summary, nil
}

func (x *Index) dropNote(ctx context.Context, source string) error {
	tx, err := x.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err = deleteSource(ctx, tx, source); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM notes WHERE path = ?`, source); err != nil {
		return err
	}
	return tx.Commit()
}

func (x *Index) indexNote(ctx context.Context, note notes.Note, source string, current string) error {
	tx, err := x.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var holder string
	err = tx.QueryRowContext(ctx, `SELECT path FROM notes WHERE id = ? AND path != ?`, note.ID, source).Scan(&holder)
	if err == nil {
		return fmt.Errorf("%w %s: id %s already used by %s", notes.ErrInvalid, source, note.ID, holder)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err = deleteSource(ctx, tx, source); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM notes WHERE path = ?`, source); err != nil {
		return err
	}
	writer, err := newEntryWriter(ctx, tx)
	if err != nil {
		return err
	}
	defer writer.close()
	row := entryRow{kind: KindNote, owner: note.Owner, profile: note.Profile, project: note.Project, session: note.Session, text: note.Body, timestamp: note.Date, uuid: note.ID}
	if _, err = writer.write(ctx, row, source, false); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO notes (fingerprint, id, kind, owner, path, project, scope, supersedes) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		current, note.ID, note.Kind, note.Owner, source, note.Project, note.Scope, strings.Join(note.Supersedes, ","))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (x *Index) recordedNotes(ctx context.Context) (map[string]string, error) {
	rows, err := x.db.QueryContext(ctx, `SELECT path, fingerprint FROM notes`)
	if err != nil {
		return nil, fmt.Errorf("read indexed notes: %w", err)
	}
	defer rows.Close()
	recorded := map[string]string{}
	for rows.Next() {
		var path, value string
		if err = rows.Scan(&path, &value); err != nil {
			return nil, err
		}
		recorded[path] = value
	}
	return recorded, rows.Err()
}

// deleteSource tells the external-content FTS table each deleted row's text.
func deleteSource(ctx context.Context, tx *sql.Tx, source string) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, text FROM entries WHERE source = ?`, source)
	if err != nil {
		return err
	}
	type deleted struct {
		id   int64
		text string
	}
	var removed []deleted
	for rows.Next() {
		var row deleted
		if err = rows.Scan(&row.id, &row.text); err != nil {
			_ = rows.Close()
			return err
		}
		removed = append(removed, row)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	for _, row := range removed {
		if _, err = tx.ExecContext(ctx, `INSERT INTO entries_fts (entries_fts, rowid, text) VALUES ('delete', ?, ?)`, row.id, row.text); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM entries WHERE source = ?`, source)
	return err
}

func fingerprint(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano()), nil
}

// WalkDir does not descend a symlinked root, so the root is resolved first. A symlinked note file is skipped: an editor's lock file is a dangling one.
func noteFiles(directory string) (map[string]noteFile, error) {
	files := map[string]noteFile{}
	root, err := filepath.EvalSymlinks(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return files, nil
	}
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() || filepath.Ext(path) != ".md" {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		current, statErr := fingerprint(path)
		if statErr != nil {
			return statErr
		}
		files["notes/"+filepath.ToSlash(relative)] = noteFile{fingerprint: current, path: path}
		return nil
	})
	return files, err
}
