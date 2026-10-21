package index

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

const (
	KindObservation = "observation"
	KindSummary     = "summary"
)

type claudeMemRow map[string]any

type claudeMemSessions struct {
	byContent map[string]dto.SessionProfile
	byMemory  map[string]dto.SessionProfile
}

type claudeMemTable struct {
	entry func(row claudeMemRow, sessions claudeMemSessions, ownerMap ownermap.OwnerMap) (entryRow, bool)
	name  string
}

var claudeMemTables = []claudeMemTable{
	{name: "observations", entry: func(row claudeMemRow, sessions claudeMemSessions, ownerMap ownermap.OwnerMap) (entryRow, bool) {
		session := sessions.byMemory[row.text("memory_session_id")]
		return memoryEntry(row, KindObservation, session, ownerMap, "title", "subtitle", "narrative", "facts", "text")
	}},
	{name: "session_summaries", entry: func(row claudeMemRow, sessions claudeMemSessions, ownerMap ownermap.OwnerMap) (entryRow, bool) {
		session := sessions.byMemory[row.text("memory_session_id")]
		return memoryEntry(row, KindSummary, session, ownerMap, "request", "investigated", "learned", "completed", "next_steps", "notes")
	}},
	{name: "tool_uses", entry: func(row claudeMemRow, sessions claudeMemSessions, ownerMap ownermap.OwnerMap) (entryRow, bool) {
		// Transcript survives and holds this row.
		session := sessions.byContent[row.text("content_session_id")]
		if session.Profile != "" {
			return entryRow{}, false
		}
		var input map[string]any
		if err := json.Unmarshal([]byte(row.text("tool_input")), &input); err != nil {
			input = map[string]any{"input": row.text("tool_input")}
		}
		project := row.text("project")
		return entryRow{
			directory: row.text("cwd"),
			kind:      transcript.KindTool,
			owner:     ownerMap.Owner(project, row.text("cwd")),
			project:   project,
			session:   row.text("content_session_id"),
			text:      transcript.ToolLine(row.text("tool_name"), input),
			timestamp: row.text("created_at"),
		}, true
	}},
	{name: "user_prompts", entry: func(row claudeMemRow, sessions claudeMemSessions, ownerMap ownermap.OwnerMap) (entryRow, bool) {
		// Transcript survives and holds this row.
		session := sessions.byContent[row.text("content_session_id")]
		if session.Profile != "" || strings.TrimSpace(row.text("prompt_text")) == "" {
			return entryRow{}, false
		}
		return entryRow{
			kind:      transcript.KindPrompt,
			owner:     ownerMap.Owner(session.Project, ""),
			project:   session.Project,
			session:   row.text("content_session_id"),
			text:      row.text("prompt_text"),
			timestamp: row.text("created_at"),
		}, true
	}},
}

// UpdateClaudeMem re-indexes a table whole when it or the session profiles change. Owners come from the live map passed in, never the archived copy.
func (x *Index) UpdateClaudeMem(ctx context.Context, directory string, ownerMap ownermap.OwnerMap) (Summary, error) {
	profilesPath := filepath.Join(directory, dto.SessionProfilesFile)
	sessions, err := loadClaudeMemSessions(profilesPath)
	if errors.Is(err, fs.ErrNotExist) {
		return Summary{}, nil
	}
	if err != nil {
		return Summary{}, err
	}
	shared, err := fingerprint(profilesPath)
	if err != nil {
		return Summary{}, err
	}

	var summary Summary
	for _, table := range claudeMemTables {
		indexed, err := x.updateClaudeMemTable(ctx, directory, table, sessions, ownerMap, shared)
		if err != nil {
			return summary, fmt.Errorf("index claude-mem %s: %w", table.name, err)
		}
		if indexed.Files > 0 {
			summary.Entries += indexed.Entries
			summary.Files++
			summary.Malformed += indexed.Malformed
		}
	}
	return summary, nil
}

func (r claudeMemRow) text(key string) string {
	value, _ := r[key].(string)
	return value
}

func (x *Index) updateClaudeMemTable(ctx context.Context, directory string, table claudeMemTable, sessions claudeMemSessions, ownerMap ownermap.OwnerMap, shared string) (Summary, error) {
	path := filepath.Join(directory, table.name+".jsonl")
	source := "claude-mem/" + table.name + ".jsonl"
	own, err := fingerprint(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Summary{}, nil
	}
	if err != nil {
		return Summary{}, err
	}
	current := own + " " + shared
	var recorded string
	err = x.db.QueryRowContext(ctx, `SELECT fingerprint FROM claude_mem_tables WHERE path = ?`, source).Scan(&recorded)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Summary{}, err
	}
	if recorded == current {
		return Summary{}, nil
	}

	tx, err := x.db.BeginTx(ctx, nil)
	if err != nil {
		return Summary{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if err = deleteSource(ctx, tx, source); err != nil {
		return Summary{}, err
	}
	writer, err := newEntryWriter(ctx, tx)
	if err != nil {
		return Summary{}, err
	}
	defer writer.close()

	file, err := os.Open(path)
	if err != nil {
		return Summary{}, err
	}
	defer file.Close()
	summary := Summary{Files: 1}
	lines := bufio.NewReaderSize(file, 1<<20)
	for {
		line, readErr := lines.ReadBytes('\n')
		if len(line) == 0 && readErr != nil {
			break
		}
		var row claudeMemRow
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.UseNumber()
		if err = decoder.Decode(&row); err != nil {
			summary.Malformed++
			continue
		}
		entry, ok := table.entry(row, sessions, ownerMap)
		if !ok {
			continue
		}
		entry.uuid = fmt.Sprintf("claude-mem:%s:%v", table.name, row["id"])
		if _, err = writer.write(ctx, entry, source, false); err != nil {
			return Summary{}, err
		}
		summary.Entries++
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO claude_mem_tables (fingerprint, path) VALUES (?, ?)
		ON CONFLICT (path) DO UPDATE SET fingerprint = excluded.fingerprint`, current, source)
	if err != nil {
		return Summary{}, err
	}
	return summary, tx.Commit()
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

func fingerprint(paths ...string) (string, error) {
	var parts []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano()))
	}
	return strings.Join(parts, " "), nil
}

func loadClaudeMemSessions(path string) (claudeMemSessions, error) {
	sessions := claudeMemSessions{byContent: map[string]dto.SessionProfile{}, byMemory: map[string]dto.SessionProfile{}}
	file, err := os.Open(path)
	if err != nil {
		return sessions, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var session dto.SessionProfile
		if err := decoder.Decode(&session); err != nil {
			return sessions, fmt.Errorf("read %s: %w", path, err)
		}
		sessions.byContent[session.ContentSessionID] = session
		if session.MemorySessionID != "" {
			sessions.byMemory[session.MemorySessionID] = session
		}
	}
	return sessions, nil
}

// Unknown session: memory_session_id stands in as the session.
func memoryEntry(row claudeMemRow, kind string, session dto.SessionProfile, ownerMap ownermap.OwnerMap, fields ...string) (entryRow, bool) {
	var parts []string
	for _, field := range fields {
		if value := strings.TrimSpace(row.text(field)); value != "" {
			parts = append(parts, value)
		}
	}
	if len(parts) == 0 {
		return entryRow{}, false
	}
	project := row.text("project")
	identifier := session.ContentSessionID
	if identifier == "" {
		identifier = row.text("memory_session_id")
	}
	return entryRow{
		kind:      kind,
		owner:     ownerMap.Owner(project, ""),
		profile:   session.Profile,
		project:   project,
		session:   identifier,
		text:      strings.Join(parts, "\n"),
		timestamp: row.text("created_at"),
	}, true
}
