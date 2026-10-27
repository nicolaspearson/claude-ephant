package index

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

func TestMainTranscriptsListsThoseWithALineInTheWindowByFirstLine(t *testing.T) {
	index := open(t, filepath.Join(t.TempDir(), "ele.db"))
	_, err := index.db.Exec(`INSERT INTO entries (kind, owner, profile, project, session, source, subagent, text, timestamp, uuid) VALUES
		('prompt', 'corp', 'claude', 'gateway', 'copy', 'claude/-gateway/copy.jsonl.gz', 0, 'resumed', '2026-10-20T09:00:00.000Z', 'c1'),
		('prompt', 'corp', 'claude', 'gateway', 'early', 'claude/-gateway/early.jsonl.gz', 0, 'before', '2026-10-19T23:00:00.000Z', 'e1'),
		('prompt', 'corp', 'claude', 'gateway', 'early', 'claude/-gateway/early.jsonl.gz', 0, 'inside', '2026-10-20T01:00:00.000Z', 'e2'),
		('tool', 'corp', 'claude', 'gateway', 'early', 'claude/-gateway/early/subagents/agent-a.jsonl.gz', 1, 'Edit: x', '2026-10-20T02:00:00.000Z', 's1'),
		('note', 'corp', 'claude', 'gateway', 'early', 'notes/corp/20261020-aaaaaa.md', 0, 'a note', '2026-10-20T03:00:00.000Z', 'n1'),
		('prompt', 'personal', 'claude', 'notes', 'late', 'claude/-notes/late.jsonl.gz', 0, 'at until', '2026-10-21T00:00:00.000Z', 'l1'),
		('prompt', 'corp', 'claude', 'gateway', 'moved', 'claude/-gateway/moved.jsonl.gz', 0, 'starts here', '2026-10-20T04:00:00.000Z', 'm1'),
		('prompt', 'personal', 'claude', 'notes', 'moved', 'claude/-gateway/moved.jsonl.gz', 0, 'moves on', '2026-10-20T05:00:00.000Z', 'm2'),
		('prompt', 'personal', 'claude', 'notes', 'old', 'claude/-notes/old.jsonl.gz', 0, 'before since', '2026-10-19T00:00:00.000Z', 'o1'),
		('prompt', '', 'claude', 'scratch', 'stray', 'claude/-scratch/stray.jsonl.gz', 0, 'unowned', '2026-10-20T06:00:00.000Z', 'x1')`)
	require.NoError(t, err)

	transcripts, err := index.MainTranscripts(context.Background(), "2026-10-20T00:00:00.000Z", "2026-10-21T00:00:00.000Z")
	require.NoError(t, err)
	require.Equal(t, []dto.Transcript{
		{First: "2026-10-19T23:00:00.000Z", Owner: "corp", Project: "gateway", Session: "early", Source: "claude/-gateway/early.jsonl.gz"},
		{First: "2026-10-20T04:00:00.000Z", Owner: "corp", Project: "gateway", Session: "moved", Source: "claude/-gateway/moved.jsonl.gz"},
		{First: "2026-10-20T09:00:00.000Z", Owner: "corp", Project: "gateway", Session: "copy", Source: "claude/-gateway/copy.jsonl.gz"},
	}, transcripts, "an owned line inside the window, not a subagent's or a note's; the window excludes until; one row a transcript, named by its first line")
}
