package index

import (
	"context"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

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
