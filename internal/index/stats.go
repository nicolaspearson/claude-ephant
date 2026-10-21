package index

import (
	"context"
	"fmt"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

const sourceGroup = `CASE WHEN source LIKE 'claude-mem/%' THEN 'claude-mem' WHEN source LIKE 'notes/%' THEN 'notes' ELSE 'transcripts' END`

func (x *Index) Stats(ctx context.Context) (dto.IndexStats, error) {
	stats := dto.IndexStats{Sessions: map[string]int{}}
	rows, err := x.db.QueryContext(ctx, `SELECT `+sourceGroup+`, kind, owner, count(*) FROM entries GROUP BY 1, 2, 3 ORDER BY 1, 2, 3`)
	if err != nil {
		return dto.IndexStats{}, fmt.Errorf("count entries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var count dto.EntryCount
		if err = rows.Scan(&count.Source, &count.Kind, &count.Owner, &count.Entries); err != nil {
			return dto.IndexStats{}, err
		}
		stats.Counts = append(stats.Counts, count)
	}
	if err = rows.Err(); err != nil {
		return dto.IndexStats{}, err
	}

	sessions, err := x.db.QueryContext(ctx, `SELECT profile, count(DISTINCT session) FROM entries WHERE session != '' GROUP BY profile`)
	if err != nil {
		return dto.IndexStats{}, fmt.Errorf("count sessions: %w", err)
	}
	defer sessions.Close()
	for sessions.Next() {
		var profile string
		var count int
		if err = sessions.Scan(&profile, &count); err != nil {
			return dto.IndexStats{}, err
		}
		stats.Sessions[profile] = count
	}
	if err = sessions.Err(); err != nil {
		return dto.IndexStats{}, err
	}

	err = x.db.QueryRowContext(ctx, `SELECT coalesce(min(substr(timestamp, 1, 10)), ''), coalesce(max(substr(timestamp, 1, 10)), '') FROM entries WHERE timestamp != ''`).Scan(&stats.First, &stats.Last)
	if err != nil {
		return dto.IndexStats{}, fmt.Errorf("read date range: %w", err)
	}
	return stats, nil
}
