package index

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/store"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

const schema = `
CREATE TABLE IF NOT EXISTS archive_files (
	archive_size  INTEGER NOT NULL,
	indexed_bytes INTEGER NOT NULL,
	path          TEXT PRIMARY KEY
);
CREATE TABLE IF NOT EXISTS claude_mem_tables (
	fingerprint TEXT NOT NULL,
	path        TEXT PRIMARY KEY
);
CREATE TABLE IF NOT EXISTS directories (
	directory TEXT NOT NULL,
	project   TEXT NOT NULL,
	PRIMARY KEY (directory, project)
);
CREATE TABLE IF NOT EXISTS entries (
	id        INTEGER PRIMARY KEY,
	kind      TEXT NOT NULL,
	line_key  TEXT UNIQUE,
	owner     TEXT NOT NULL,
	profile   TEXT NOT NULL,
	project   TEXT NOT NULL,
	session   TEXT NOT NULL,
	source    TEXT NOT NULL,
	subagent  INTEGER NOT NULL,
	text      TEXT NOT NULL,
	timestamp TEXT NOT NULL,
	uuid      TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS events (
	attributes TEXT NOT NULL,
	id         INTEGER PRIMARY KEY,
	kind       TEXT NOT NULL,
	line_key   TEXT UNIQUE,
	name       TEXT NOT NULL,
	owner      TEXT NOT NULL,
	profile    TEXT NOT NULL,
	project    TEXT NOT NULL,
	session    TEXT NOT NULL,
	source     TEXT NOT NULL,
	subagent   INTEGER NOT NULL,
	timestamp  TEXT NOT NULL,
	uuid       TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS notes (
	fingerprint TEXT NOT NULL,
	id          TEXT PRIMARY KEY,
	kind        TEXT NOT NULL,
	owner       TEXT NOT NULL,
	path        TEXT NOT NULL UNIQUE,
	project     TEXT NOT NULL,
	scope       TEXT NOT NULL,
	supersedes  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS entries_owner_project ON entries (owner, project, session);
DROP INDEX IF EXISTS entries_session;
CREATE INDEX IF NOT EXISTS entries_session_kind ON entries (session, kind, timestamp);
CREATE INDEX IF NOT EXISTS entries_source ON entries (source);
CREATE INDEX IF NOT EXISTS events_kind_name ON events (kind, name, timestamp);
DROP INDEX IF EXISTS events_session;
CREATE INDEX IF NOT EXISTS events_session_kind ON events (session, kind, timestamp);
CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(text, content='entries', content_rowid='id');`

// Version is bumped when the schema or what sync derives changes; an index holding another version is rebuilt.
const Version = "9"

type Index struct {
	db       *sql.DB
	problems []string
}

type Projects struct {
	ownerMap ownermap.OwnerMap
	roots    map[string]string
}

type Summary struct {
	Automated int
	Entries   int
	Events    int
	Files     int
	Malformed int
	Problems  []string
}

func NewProjects(ownerMap ownermap.OwnerMap) *Projects {
	return &Projects{ownerMap: ownerMap, roots: map[string]string{}}
}

// Open applies the schema only to a new or matching index: a CREATE INDEX on a missing column fails before Reset can repair it.
func Open(path string) (*Index, error) {
	db, err := store.Open(path)
	if err != nil {
		return nil, err
	}
	problems, err := schemaProblems(context.Background(), db)
	if err == nil && len(problems) == 0 {
		if _, err = db.Exec(schema); err != nil {
			err = fmt.Errorf("create index schema: %w", err)
		}
	}
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Index{db: db, problems: problems}, nil
}

// OpenChecked refuses an index whose tables differ; sync uses Open so it can rebuild one.
func OpenChecked(path string) (*Index, error) {
	searchIndex, err := Open(path)
	if err != nil {
		return nil, err
	}
	if problems := searchIndex.SchemaProblems(); len(problems) > 0 {
		_ = searchIndex.Close()
		return nil, fmt.Errorf("the index's tables do not match this version of ele (%s); run ele sync to rebuild it", strings.Join(problems, "; "))
	}
	return searchIndex, nil
}

func (x *Index) Close() error {
	return x.db.Close()
}

func (x *Index) ProjectDirectories(ctx context.Context) ([]dto.ProjectDirectory, error) {
	rows, err := x.db.QueryContext(ctx, `SELECT directory, project FROM directories ORDER BY project, directory`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var directories []dto.ProjectDirectory
	for rows.Next() {
		var directory dto.ProjectDirectory
		if err = rows.Scan(&directory.Directory, &directory.Project); err != nil {
			return nil, err
		}
		directories = append(directories, directory)
	}
	return directories, rows.Err()
}

// Update expects the caller to hold the archive lock.
func (x *Index) Update(ctx context.Context, directory string, projects *Projects) (Summary, error) {
	var summary Summary
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.Type().IsRegular() || filepath.Ext(path) != ".gz" {
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		file, err := x.updateFile(ctx, path, filepath.ToSlash(relative), projects)
		if err != nil {
			return fmt.Errorf("index %s: %w", relative, err)
		}
		if file.Files > 0 {
			summary.Automated += file.Automated
			summary.Entries += file.Entries
			summary.Events += file.Events
			summary.Files++
			summary.Malformed += file.Malformed
		}
		return nil
	})
	return summary, err
}

func (x *Index) updateFile(ctx context.Context, path string, relative string, projects *Projects) (Summary, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Summary{}, err
	}
	var archiveSize, indexed int64
	err = x.db.QueryRowContext(ctx, `SELECT archive_size, indexed_bytes FROM archive_files WHERE path = ?`, relative).Scan(&archiveSize, &indexed)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Summary{}, err
	}
	if archiveSize == info.Size() {
		return Summary{}, nil
	}

	reader, err := archive.Open(path)
	if err != nil {
		return Summary{}, err
	}
	defer reader.Close()
	if _, err = io.CopyN(io.Discard, reader, indexed); err != nil {
		return Summary{}, fmt.Errorf("skip %d indexed bytes: %w", indexed, err)
	}

	tx, err := x.db.BeginTx(ctx, nil)
	if err != nil {
		return Summary{}, err
	}
	defer func() { _ = tx.Rollback() }()
	writer, err := newEntryWriter(ctx, tx)
	if err != nil {
		return Summary{}, err
	}
	defer writer.close()

	profile, _, _ := strings.Cut(relative, "/")
	subagent := strings.Contains(relative, "/subagents/")
	summary := Summary{Files: 1}
	lines := bufio.NewReaderSize(reader, 1<<20)
	for {
		line, readErr := lines.ReadBytes('\n')
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return Summary{}, readErr
		}
		indexed += int64(len(line))
		entries, events, parseErr := transcript.Parse(line)
		if parseErr != nil {
			summary.Malformed++
			continue
		}
		for position, event := range events {
			if strings.HasPrefix(event.Entrypoint, "sdk-") {
				continue
			}
			project, owner := projects.Resolve(event.WorkingDirectory)
			written, writeErr := writer.writeEvent(ctx, event, eventPlace{directory: event.WorkingDirectory, owner: owner, position: position, profile: profile, project: project, source: relative, subagent: subagent})
			if writeErr != nil {
				return Summary{}, writeErr
			}
			if written {
				summary.Events++
			}
		}
		for position, entry := range entries {
			if strings.HasPrefix(entry.Entrypoint, "sdk-") {
				summary.Automated++
				continue
			}
			project, owner := projects.Resolve(entry.WorkingDirectory)
			row := entryRow{directory: entry.WorkingDirectory, kind: entry.Kind, owner: owner, profile: profile, project: project, session: entry.SessionID, text: entry.Text, timestamp: entry.Timestamp, uuid: entry.UUID}
			if entry.UUID != "" {
				row.lineKey = fmt.Sprintf("%s#%d", entry.UUID, position)
			}
			written, writeErr := writer.write(ctx, row, relative, subagent)
			if writeErr != nil {
				return Summary{}, writeErr
			}
			if written {
				summary.Entries++
			}
		}
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO archive_files (archive_size, indexed_bytes, path) VALUES (?, ?, ?)
		ON CONFLICT (path) DO UPDATE SET archive_size = excluded.archive_size, indexed_bytes = excluded.indexed_bytes`, info.Size(), indexed, relative)
	if err != nil {
		return Summary{}, err
	}
	return summary, tx.Commit()
}

// Resolve walks to the git root so a subdirectory lands on its repo; outside a repo the directory names itself.
func (p *Projects) Resolve(directory string) (project string, owner string) {
	if directory == "" {
		return "", ""
	}
	root, ok := p.roots[directory]
	if !ok {
		root = GitRoot(directory)
		p.roots[directory] = root
	}
	project = filepath.Base(root)
	return project, p.ownerMap.Owner(project, directory)
}

func GitRoot(directory string) string {
	for current := directory; ; {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return directory
		}
		current = parent
	}
}
