package index

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/redact"
)

type entryRow struct {
	directory string
	kind      string
	lineKey   string
	owner     string
	profile   string
	project   string
	session   string
	text      string
	timestamp string
	uuid      string
}

type entryWriter struct {
	directories *sql.Stmt
	entries     *sql.Stmt
	events      *sql.Stmt
	recorded    map[dto.ProjectDirectory]bool
	search      *sql.Stmt
}

type eventPlace struct {
	directory string
	owner     string
	position  int
	profile   string
	project   string
	source    string
	subagent  bool
}

func newEntryWriter(ctx context.Context, tx *sql.Tx) (*entryWriter, error) {
	entries, err := tx.PrepareContext(ctx, `INSERT INTO entries (kind, line_key, owner, profile, project, session, source, subagent, text, timestamp, uuid) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (line_key) DO NOTHING`)
	if err != nil {
		return nil, err
	}
	// Keyed so a transcript re-archived from its start adds no event twice; lines without an id are not deduplicated.
	// Tokens only, others stay insert-or-ignore: keep a message's largest output count, which is its final line; a
	// fork's copy of its parent's message stays the main thread's.
	events, err := tx.PrepareContext(ctx, `INSERT INTO events (attributes, kind, line_key, name, owner, profile, project, session, source, subagent, timestamp, uuid) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (line_key) DO UPDATE SET
			attributes = iif(json_extract(excluded.attributes, '$.output') > json_extract(events.attributes, '$.output'), excluded.attributes, events.attributes),
			source = iif(excluded.subagent < events.subagent, excluded.source, events.source),
			subagent = min(events.subagent, excluded.subagent)
		WHERE excluded.kind = 'tokens' AND (json_extract(excluded.attributes, '$.output') > json_extract(events.attributes, '$.output') OR excluded.subagent < events.subagent)`)
	if err != nil {
		_ = entries.Close()
		return nil, err
	}
	search, err := tx.PrepareContext(ctx, `INSERT INTO entries_fts (rowid, text) VALUES (?, ?)`)
	if err != nil {
		_ = entries.Close()
		_ = events.Close()
		return nil, err
	}
	directories, err := tx.PrepareContext(ctx, `INSERT INTO directories (directory, project) VALUES (?, ?) ON CONFLICT DO NOTHING`)
	if err != nil {
		_ = entries.Close()
		_ = events.Close()
		_ = search.Close()
		return nil, err
	}
	return &entryWriter{directories: directories, entries: entries, events: events, recorded: map[dto.ProjectDirectory]bool{}, search: search}, nil
}

func (w *entryWriter) close() {
	_ = w.directories.Close()
	_ = w.entries.Close()
	_ = w.events.Close()
	_ = w.search.Close()
}

func (w *entryWriter) recordDirectory(ctx context.Context, directory string, project string) error {
	key := dto.ProjectDirectory{Directory: directory, Project: project}
	if directory == "" || w.recorded[key] {
		return nil
	}
	if _, err := w.directories.ExecContext(ctx, directory, project); err != nil {
		return err
	}
	w.recorded[key] = true
	return nil
}

func (w *entryWriter) write(ctx context.Context, row entryRow, source string, subagent bool) (bool, error) {
	if err := w.recordDirectory(ctx, row.directory, row.project); err != nil {
		return false, err
	}
	var lineKey any
	if row.lineKey != "" {
		lineKey = row.lineKey
	}
	text := redact.Redact(row.text)
	result, err := w.entries.ExecContext(ctx, row.kind, lineKey, row.owner, row.profile, row.project, row.session, source, subagent, text, row.timestamp, row.uuid)
	if err != nil {
		return false, err
	}
	written, err := result.RowsAffected()
	if err != nil || written == 0 {
		return false, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return false, err
	}
	_, err = w.search.ExecContext(ctx, id, text)
	return err == nil, err
}

func (w *entryWriter) writeEvent(ctx context.Context, event dto.TranscriptEvent, place eventPlace) (bool, error) {
	if err := w.recordDirectory(ctx, place.directory, place.project); err != nil {
		return false, err
	}
	attributes := map[string]any{}
	for key, value := range event.Attributes {
		if text, ok := value.(string); ok {
			value = redact.Redact(text)
		}
		attributes[key] = value
	}
	encoded, err := json.Marshal(attributes)
	if err != nil {
		return false, err
	}
	var lineKey any
	switch {
	case event.Key != "":
		lineKey = event.Key
	case event.UUID != "":
		lineKey = fmt.Sprintf("%s#%d", event.UUID, place.position)
	}
	result, err := w.events.ExecContext(ctx, string(encoded), event.Kind, lineKey, redact.Redact(event.Name), place.owner, place.profile, place.project, event.SessionID, place.source, place.subagent, event.Timestamp, event.UUID)
	if err != nil {
		return false, err
	}
	changed, err := result.RowsAffected()
	return changed > 0, err
}
