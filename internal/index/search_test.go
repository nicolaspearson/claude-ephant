package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

var fixtureOwnerMap = ownermap.OwnerMap{Owners: []string{"corp", "personal"}, Projects: map[string]string{"gateway": "corp", "notes": "personal"}}

func TestProjectOwnersListsEachOwnerOnce(t *testing.T) {
	index := searchFixture(t)

	owners, err := index.ProjectOwners(context.Background(), "gateway")
	require.NoError(t, err)
	require.Equal(t, []string{"corp"}, owners)

	owners, err = index.ProjectOwners(context.Background(), "nowhere")
	require.NoError(t, err)
	require.Empty(t, owners)
}

func TestSearchRanksKindsThenRelevanceWithinFilters(t *testing.T) {
	index := searchFixture(t)

	results, err := index.Search(context.Background(), Query{Owner: "corp", Text: "pydantic"})
	require.NoError(t, err)
	require.Equal(t, []string{"prompt", "conversation", "tool"}, kinds(results))
	require.Equal(t, "pin «pydantic» below 2.14", results[0].Text)

	results, err = index.Search(context.Background(), Query{Kinds: []string{"conversation", "tool"}, Owner: "corp", Text: "pydantic"})
	require.NoError(t, err)
	require.Equal(t, []string{"conversation", "tool"}, kinds(results))

	results, err = index.Search(context.Background(), Query{Owner: "personal", Text: "pydantic"})
	require.NoError(t, err)
	require.Equal(t, []string{"prompt"}, kinds(results))

	results, err = index.Search(context.Background(), Query{Text: "pydantic"})
	require.NoError(t, err)
	require.Len(t, results, 4)

	results, err = index.Search(context.Background(), Query{Owner: "corp", Since: "2026-10-01", Text: "pydantic"})
	require.NoError(t, err)
	require.Equal(t, []string{"prompt", "conversation"}, kinds(results))

	results, err = index.Search(context.Background(), Query{Limit: 2, Text: "pydantic"})
	require.NoError(t, err)
	require.Len(t, results, 2)
}

func TestSearchTreatsQueryPunctuationAsWords(t *testing.T) {
	index := searchFixture(t)

	results, err := index.Search(context.Background(), Query{Text: `stop-check "unbalanced`})
	require.NoError(t, err)
	require.Empty(t, results)

	results, err = index.Search(context.Background(), Query{Text: "   "})
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestSessionCountCountsOneOwnersProject(t *testing.T) {
	index := searchFixture(t)
	_, err := index.db.Exec(`INSERT INTO entries (kind, owner, profile, project, session, source, subagent, text, timestamp, uuid) VALUES ('note', 'corp', '', 'gateway', '', 'notes/corp/20260601-aaaaaa.md', 0, 'no session', '2026-06-01', '20260601-aaaaaa')`)
	require.NoError(t, err)

	count, err := index.SessionCount(context.Background(), "corp", "gateway")
	require.NoError(t, err)
	require.Equal(t, 2, count)

	count, err = index.SessionCount(context.Background(), "personal", "gateway")
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSessionReturnsOneSessionByPrefix(t *testing.T) {
	index := searchFixture(t)

	results, err := index.Session(context.Background(), "abc1")
	require.NoError(t, err)
	require.Equal(t, []string{"prompt", "conversation"}, kinds(results))
	require.Equal(t, "pin pydantic below 2.14", results[0].Text)

	_, err = index.Session(context.Background(), "abc")
	require.ErrorIs(t, err, ErrAmbiguousSession)

	results, err = index.Session(context.Background(), "none")
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestResetLeavesNoFTSRows(t *testing.T) {
	index := searchFixture(t)

	require.NoError(t, index.Reset(context.Background()))
	_, err := index.db.Exec(`INSERT INTO entries_fts (entries_fts, rank) VALUES ('integrity-check', 1)`)
	require.NoError(t, err, "Reset left FTS rows behind")
	results, err := index.Search(context.Background(), Query{Text: "pydantic"})
	require.NoError(t, err)
	require.Empty(t, results)
}

func kinds(results []Result) []string {
	var kinds []string
	for _, result := range results {
		kinds = append(kinds, result.Kind)
	}
	return kinds
}

// Sessions abc1 and abc2 in gateway (corp), p1 in notes (personal).
func searchFixture(t *testing.T) *Index {
	t.Helper()
	root := t.TempDir()
	gateway := filepath.Join(root, "w", "gateway")
	notes := filepath.Join(root, "w", "notes")
	for _, repository := range []string{gateway, notes} {
		require.NoError(t, os.MkdirAll(filepath.Join(repository, ".git"), 0o700))
	}
	projects := filepath.Join(root, "projects")
	line := func(session string, directory string, kind string, content string, timestamp string) string {
		return `{"type":"` + kind + `","cwd":"` + directory + `","sessionId":"` + session + `","timestamp":"` + timestamp + `","uuid":"` + lineID(session, kind, content, timestamp) + `","message":{"content":` + content + "}}\n"
	}
	write(t, filepath.Join(projects, "-gateway", "abc1.jsonl"),
		line("abc1", gateway, "user", `"pin pydantic below 2.14"`, "2026-10-15T10:00:00Z")+
			line("abc1", gateway, "assistant", `[{"type":"text","text":"Pinned pydantic in the wrapper and the pydantic tests pass."}]`, "2026-10-15T10:01:00Z"))
	write(t, filepath.Join(projects, "-gateway", "abc2.jsonl"),
		line("abc2", gateway, "user", `"fix chroma"`, "2026-06-01T00:00:00Z")+
			line("abc2", gateway, "assistant", `[{"type":"tool_use","name":"Bash","input":{"command":"uvx --with pydantic"}}]`, "2026-06-01T00:01:00Z"))
	write(t, filepath.Join(projects, "-notes", "p1.jsonl"),
		line("p1", notes, "user", `"read about pydantic"`, "2026-06-01T00:00:00Z"))

	index := open(t, filepath.Join(root, "ele.db"))
	update(t, index, projects, filepath.Join(root, "archive", "transcripts"), fixtureOwnerMap)
	return index
}
