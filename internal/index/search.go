package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

var ErrAmbiguousSession = errors.New("session prefix matches more than one session")

var Kinds = []string{KindNote, transcript.KindPrompt, transcript.KindConversation, transcript.KindTool}

// Kinds rank before relevance: notes and the user's own words outrank what was said about them.
const kindRank = `CASE
	WHEN e.kind = 'note' THEN 0
	WHEN e.kind = 'prompt' THEN 1
	WHEN e.kind = 'conversation' THEN 2
	WHEN e.kind = 'tool' THEN 3
	ELSE 4
END`

const resultColumns = `e.kind, e.owner, e.profile, e.project, e.session, e.subagent,
	coalesce((SELECT n.kind FROM notes n WHERE n.path = e.source), ''),
	coalesce((SELECT s.id FROM notes s WHERE instr(',' || replace(s.supersedes, ' ', '') || ',', ',' || e.uuid || ',') > 0 AND e.kind = 'note' ORDER BY s.id LIMIT 1), ''),
	CASE WHEN e.kind = 'note' THEN e.uuid ELSE '' END`

type Query struct {
	Kinds   []string
	Limit   int
	Owner   string
	Project string
	Since   string
	Text    string
}

type Result struct {
	Kind         string
	NoteID       string
	NoteKind     string
	Owner        string
	Profile      string
	Project      string
	Session      string
	Subagent     bool
	SupersededBy string
	Text         string
	Timestamp    string
}

func (x *Index) ProjectOwners(ctx context.Context, project string) ([]string, error) {
	rows, err := x.db.QueryContext(ctx, `SELECT DISTINCT owner FROM entries WHERE project = ? AND owner != '' ORDER BY owner`, project)
	if err != nil {
		return nil, fmt.Errorf("find project owners: %w", err)
	}
	defer rows.Close()
	var owners []string
	for rows.Next() {
		var owner string
		if err = rows.Scan(&owner); err != nil {
			return nil, err
		}
		owners = append(owners, owner)
	}
	return owners, rows.Err()
}

func (x *Index) Search(ctx context.Context, query Query) ([]Result, error) {
	match := matchExpression(query.Text)
	if match == "" {
		return nil, nil
	}
	conditions := []string{"entries_fts MATCH ?"}
	arguments := []any{match}
	if query.Owner != "" {
		conditions = append(conditions, "e.owner IN (?, ?)")
		arguments = append(arguments, query.Owner, notes.OwnerEverywhere)
	}
	if query.Project != "" {
		conditions = append(conditions, "e.project = ?")
		arguments = append(arguments, query.Project)
	}
	if query.Since != "" {
		conditions = append(conditions, "e.timestamp >= ?")
		arguments = append(arguments, query.Since)
	}
	if len(query.Kinds) > 0 {
		conditions = append(conditions, "e.kind IN ("+strings.TrimSuffix(strings.Repeat("?, ", len(query.Kinds)), ", ")+")")
		for _, kind := range query.Kinds {
			arguments = append(arguments, kind)
		}
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	arguments = append(arguments, limit)

	rows, err := x.db.QueryContext(ctx, `
		SELECT `+resultColumns+`, snippet(entries_fts, 0, '«', '»', '…', 24), e.timestamp
		FROM entries_fts JOIN entries e ON e.id = entries_fts.rowid
		WHERE `+strings.Join(conditions, " AND ")+`
		ORDER BY `+kindRank+`, bm25(entries_fts), e.timestamp DESC
		LIMIT ?`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	return scanResults(rows)
}

// Session accepts any prefix that only one session starts with.
func (x *Index) Session(ctx context.Context, prefix string) ([]Result, error) {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(prefix)
	var sessions []string
	rows, err := x.db.QueryContext(ctx, `SELECT DISTINCT session FROM entries WHERE session LIKE ? ESCAPE '\' LIMIT 4`, escaped+"%")
	if err != nil {
		return nil, fmt.Errorf("find session: %w", err)
	}
	for rows.Next() {
		var session string
		if err = rows.Scan(&session); err != nil {
			_ = rows.Close()
			return nil, err
		}
		sessions = append(sessions, session)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	switch {
	case len(sessions) == 0:
		return nil, nil
	case len(sessions) > 3:
		return nil, fmt.Errorf("%w: %s and more", ErrAmbiguousSession, strings.Join(sessions[:3], ", "))
	case len(sessions) > 1:
		return nil, fmt.Errorf("%w: %s", ErrAmbiguousSession, strings.Join(sessions, ", "))
	}

	return x.SessionExactly(ctx, sessions[0])
}

func (x *Index) SessionCount(ctx context.Context, owner string, project string) (int, error) {
	var count int
	err := x.db.QueryRowContext(ctx, `SELECT count(DISTINCT session) FROM entries WHERE owner = ? AND project = ? AND session != ''`, owner, project).Scan(&count)
	return count, err
}

func (x *Index) SessionExactly(ctx context.Context, session string) ([]Result, error) {
	rows, err := x.db.QueryContext(ctx, `
		SELECT `+resultColumns+`, e.text, e.timestamp
		FROM entries e WHERE e.session = ? ORDER BY e.timestamp, e.id`, session)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}
	return scanResults(rows)
}

// FTS5 syntax in user text (stop-check, a:b, AND, NEAR) stays plain words.
func matchExpression(text string) string {
	var terms []string
	for term := range strings.FieldsSeq(text) {
		terms = append(terms, `"`+strings.ReplaceAll(term, `"`, `""`)+`"`)
	}
	return strings.Join(terms, " ")
}

func scanResults(rows *sql.Rows) ([]Result, error) {
	defer rows.Close()
	var results []Result
	for rows.Next() {
		var result Result
		if err := rows.Scan(&result.Kind, &result.Owner, &result.Profile, &result.Project, &result.Session, &result.Subagent, &result.NoteKind, &result.SupersededBy, &result.NoteID, &result.Text, &result.Timestamp); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, rows.Err()
}
