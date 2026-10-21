package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

func TestResetRebuildsAnEventsTableOfAnOlderShape(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	write(t, filepath.Join(projects, "-w", "s1.jsonl"), `{"type":"assistant","sessionId":"s1","timestamp":"2026-10-15T10:00:00Z","uuid":"u1","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}}`+"\n")
	index := open(t, filepath.Join(root, "ele.db"))
	_, err := index.db.Exec(`DROP TABLE events; CREATE TABLE events (attributes TEXT NOT NULL, id INTEGER PRIMARY KEY, kind TEXT NOT NULL, name TEXT NOT NULL, owner TEXT NOT NULL, profile TEXT NOT NULL, project TEXT NOT NULL, session TEXT NOT NULL, source TEXT NOT NULL, subagent INTEGER NOT NULL, timestamp TEXT NOT NULL, uuid TEXT NOT NULL)`)
	require.NoError(t, err)
	require.NoError(t, index.SetSetting(context.Background(), "index-version", "2"))

	require.NoError(t, index.Reset(context.Background()))
	summary := update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{})
	require.Equal(t, 1, summary.Events)
	version, err := index.Setting(context.Background(), "index-version")
	require.NoError(t, err)
	require.Equal(t, "2", version, "Reset keeps settings")
}

func TestUpdateIndexesEventsThatEntriesLack(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "w", "gateway")
	require.NoError(t, os.MkdirAll(filepath.Join(repository, ".git"), 0o700))
	projects := filepath.Join(root, "projects")
	line := func(entrypoint string, minute string, body string) string {
		return `{"cwd":"` + repository + `","entrypoint":"` + entrypoint + `","sessionId":"s1","timestamp":"2026-10-15T10:` + minute + `:00Z","uuid":"u` + minute + `",` + body + "}\n"
	}
	content := line("cli", "00", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}`) +
		line("cli", "01", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Agent","input":{"description":"review ghp_abcdefghijklmnopqrstuvwxyz0123456789","prompt":"Read reviewers/bugs.md","subagent_type":"general-purpose"}}]}`) +
		line("cli", "02", `"type":"user","isMeta":true,"message":{"content":"Stop hook feedback:\nFiles changed this turn."}`) +
		line("sdk-ts", "03", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"automated"}}]}`) +
		line("cli", "04", `"type":"user","isCompactSummary":true,"message":{"content":"continued"}`) +
		line("cli", "05", `"type":"user","message":{"content":"<command-name>/ghp_abcdefghijklmnopqrstuvwxyz0123456789</command-name>"}`)
	write(t, filepath.Join(projects, "-gateway", "s1.jsonl"), content)
	// The same lines archived a second time, as a transcript rewritten in place would be.
	write(t, filepath.Join(projects, "-gateway-copy", "s1.jsonl"), content)
	index := open(t, filepath.Join(root, "ele.db"))
	ownerMap := ownermap.OwnerMap{Owners: []string{"corp"}, Projects: map[string]string{"gateway": "corp"}}
	summary := update(t, index, projects, filepath.Join(root, "archive"), ownerMap)
	require.Equal(t, 5, summary.Events)

	events, total, err := index.Events(context.Background(), dto.EventQuery{})
	require.NoError(t, err)
	require.Equal(t, 5, total)
	require.Equal(t, []dto.Event{
		{Kind: "skill", Name: "checkpoint", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Timestamp: "2026-10-15T10:00:00Z"},
		{Attributes: map[string]any{"description": "review [redacted:github]", "reviewers": []any{"bugs"}}, Kind: "agent", Name: "general-purpose", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Timestamp: "2026-10-15T10:01:00Z"},
		{Attributes: map[string]any{"reason": "Files changed this turn."}, Kind: "hook-block", Name: "Stop", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Timestamp: "2026-10-15T10:02:00Z"},
		{Kind: "compaction", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Timestamp: "2026-10-15T10:04:00Z"},
		{Kind: "command", Name: "[redacted:github]", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Timestamp: "2026-10-15T10:05:00Z"},
	}, events)

	events, total, err = index.Events(context.Background(), dto.EventQuery{Kind: "skill", Name: "checkpoint", Since: "2026-10-15"})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, events, 1)
	events, total, err = index.Events(context.Background(), dto.EventQuery{Limit: 2})
	require.NoError(t, err)
	require.Equal(t, 5, total)
	require.Equal(t, []string{"compaction", "command"}, []string{events[0].Kind, events[1].Kind}, "the newest two, oldest first")
	events, _, err = index.Events(context.Background(), dto.EventQuery{Since: "2026-10-15T10:05:00Z"})
	require.NoError(t, err)
	require.Len(t, events, 1)
	events, _, err = index.Events(context.Background(), dto.EventQuery{Since: "2026-10-16"})
	require.NoError(t, err)
	require.Empty(t, events)

	require.NoError(t, index.Reset(context.Background()))
	_, total, err = index.Events(context.Background(), dto.EventQuery{})
	require.NoError(t, err)
	require.Zero(t, total)
}

func TestUpdateKeysEventsByLineAndPlace(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	keyless := `{"type":"assistant","sessionId":"s1","timestamp":"2026-10-15T10:00:00Z","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}}` + "\n"
	twice := `{"type":"assistant","sessionId":"s1","timestamp":"2026-10-15T10:01:00Z","uuid":"u1","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}},{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}}` + "\n"
	write(t, filepath.Join(projects, "-w", "s1.jsonl"), keyless+keyless+twice)
	index := open(t, filepath.Join(root, "ele.db"))
	summary := update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{})
	require.Equal(t, 4, summary.Events, "two lines without an id, and two calls on one line")
}
