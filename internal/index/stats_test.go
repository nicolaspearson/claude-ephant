package index

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

func TestStatsCountsBySourceKindOwnerAndProfile(t *testing.T) {
	index := searchFixture(t)
	_, err := index.db.Exec(`INSERT INTO entries (kind, owner, profile, project, session, source, subagent, text, timestamp, uuid) VALUES
		('note', 'corp', 'claude', 'gateway', 'abc9', 'notes/corp/20261015-aaaaaa.md', 0, 'a note', '2026-10-15', '20261015-aaaaaa')`)
	require.NoError(t, err)

	stats, err := index.Stats(context.Background())
	require.NoError(t, err)
	require.Equal(t, []dto.EntryCount{
		{Entries: 1, Kind: "note", Owner: "corp", Source: "notes"},
		{Entries: 1, Kind: "conversation", Owner: "corp", Source: "transcripts"},
		{Entries: 2, Kind: "prompt", Owner: "corp", Source: "transcripts"},
		{Entries: 1, Kind: "prompt", Owner: "personal", Source: "transcripts"},
		{Entries: 1, Kind: "tool", Owner: "corp", Source: "transcripts"},
	}, stats.Counts)
	require.Equal(t, map[string]int{"claude": 3}, stats.Sessions, "a note's session is not counted")
	require.Equal(t, "2026-06-01", stats.First)
	require.Equal(t, "2026-10-15", stats.Last)
}
