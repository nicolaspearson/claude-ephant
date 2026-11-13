package index

import (
	"context"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

// LastLines maps each session to its latest indexed line, a subagent's included: a subagent at work keeps its session
// live.
func (x *Index) LastLines(ctx context.Context, sessions []string) (map[string]string, error) {
	lastLines := make(map[string]string, len(sessions))
	if len(sessions) == 0 {
		return lastLines, nil
	}
	arguments := make([]any, len(sessions))
	for position, session := range sessions {
		arguments[position] = session
	}
	rows, err := x.db.QueryContext(ctx, `SELECT session, max(timestamp) FROM entries
		WHERE source NOT LIKE 'notes/%' AND session IN (?`+strings.Repeat(", ?", len(sessions)-1)+`)
		GROUP BY session`, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var session, timestamp string
		if err = rows.Scan(&session, &timestamp); err != nil {
			return nil, err
		}
		lastLines[session] = timestamp
	}
	return lastLines, rows.Err()
}

// MainTranscripts lists the owned main transcripts with a line in [since, until), ordered by earliest indexed line. The
// owner, project and session are the earliest line's: SQLite takes bare columns from the row min() picks.
func (x *Index) MainTranscripts(ctx context.Context, since string, until string) ([]dto.Transcript, error) {
	rows, err := x.db.QueryContext(ctx, `SELECT min(timestamp), owner, project, session, source FROM entries
		WHERE subagent = 0 AND source NOT LIKE 'notes/%' AND session != '' AND owner != ''
		GROUP BY source
		HAVING sum(timestamp >= ? AND timestamp < ?) > 0
		ORDER BY 1, source`, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var transcripts []dto.Transcript
	for rows.Next() {
		var transcript dto.Transcript
		if err = rows.Scan(&transcript.First, &transcript.Owner, &transcript.Project, &transcript.Session, &transcript.Source); err != nil {
			return nil, err
		}
		transcripts = append(transcripts, transcript)
	}
	return transcripts, rows.Err()
}
