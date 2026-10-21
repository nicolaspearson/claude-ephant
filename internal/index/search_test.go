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
	index, _ := searchFixture(t)

	owners, err := index.ProjectOwners(context.Background(), "gateway")
	require.NoError(t, err)
	require.Equal(t, []string{"corp"}, owners)

	owners, err = index.ProjectOwners(context.Background(), "nowhere")
	require.NoError(t, err)
	require.Empty(t, owners)
}

func TestSearchRanksKindsThenRelevanceWithinFilters(t *testing.T) {
	index, _ := searchFixture(t)

	results, err := index.Search(context.Background(), Query{Owner: "corp", Text: "pydantic"})
	require.NoError(t, err)
	require.Equal(t, []string{"prompt", "conversation", "summary", "observation", "tool"}, kinds(results))
	require.True(t, results[4].ClaudeMem)
	require.Equal(t, "pin «pydantic» below 2.14", results[0].Text)

	results, err = index.Search(context.Background(), Query{Kinds: []string{"summary", "observation"}, Owner: "corp", Text: "pydantic"})
	require.NoError(t, err)
	require.Equal(t, []string{"summary", "observation"}, kinds(results))

	results, err = index.Search(context.Background(), Query{Owner: "personal", Text: "pydantic"})
	require.NoError(t, err)
	require.Equal(t, []string{"prompt"}, kinds(results))

	results, err = index.Search(context.Background(), Query{Text: "pydantic"})
	require.NoError(t, err)
	require.Len(t, results, 6)

	results, err = index.Search(context.Background(), Query{Owner: "corp", Since: "2026-10-01", Text: "pydantic"})
	require.NoError(t, err)
	require.Equal(t, []string{"prompt", "conversation"}, kinds(results))

	results, err = index.Search(context.Background(), Query{Limit: 2, Text: "pydantic"})
	require.NoError(t, err)
	require.Len(t, results, 2)
}

func TestSearchTreatsQueryPunctuationAsWords(t *testing.T) {
	index, _ := searchFixture(t)

	results, err := index.Search(context.Background(), Query{Text: `claude-mem "unbalanced`})
	require.NoError(t, err)
	require.Empty(t, results)

	results, err = index.Search(context.Background(), Query{Text: "   "})
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestSessionCountCountsOneOwnersProject(t *testing.T) {
	index, _ := searchFixture(t)
	_, err := index.db.Exec(`INSERT INTO entries (kind, owner, profile, project, session, source, subagent, text, timestamp, uuid) VALUES ('observation', 'corp', '', 'gateway', '', 'claude-mem/observations.jsonl', 0, 'no session', '2026-06-01', 'u')`)
	require.NoError(t, err)

	count, err := index.SessionCount(context.Background(), "corp", "gateway")
	require.NoError(t, err)
	require.Equal(t, 3, count)

	count, err = index.SessionCount(context.Background(), "personal", "gateway")
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSessionReturnsOneSessionByPrefix(t *testing.T) {
	index, _ := searchFixture(t)

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

func TestUpdateClaudeMemSkipsRowsTheTranscriptHoldsAndReindexesAChangedExport(t *testing.T) {
	index, directory := searchFixture(t)

	results, err := index.Search(context.Background(), Query{Text: "duplicate"})
	require.NoError(t, err)
	require.Empty(t, results)

	summary, err := index.UpdateClaudeMem(context.Background(), directory, fixtureOwnerMap)
	require.NoError(t, err)
	require.Equal(t, Summary{}, summary)

	writeLines(t, filepath.Join(directory, "observations.jsonl"), `{"id":1234567,"memory_session_id":"m-old","project":"gateway","title":"rewritten pydantic note","created_at":"2026-06-01T00:00:00Z"}`)
	summary, err = index.UpdateClaudeMem(context.Background(), directory, fixtureOwnerMap)
	require.NoError(t, err)
	require.Equal(t, Summary{Entries: 1, Files: 1}, summary)
	results, err = index.Search(context.Background(), Query{Kinds: []string{"observation"}, Text: "pydantic"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "rewritten «pydantic» note", results[0].Text)
	var identifier string
	require.NoError(t, index.db.QueryRow(`SELECT uuid FROM entries WHERE source = 'claude-mem/observations.jsonl'`).Scan(&identifier))
	require.Equal(t, "claude-mem:observations:1234567", identifier)

	writeLines(t, filepath.Join(directory, "user_prompts.jsonl"), `{"id":3,"content_session_id":"p1","prompt_text":"unrelated words","created_at":"2026-06-02T00:00:00Z"}`)
	_, err = index.UpdateClaudeMem(context.Background(), directory, fixtureOwnerMap)
	require.NoError(t, err)
	_, err = index.db.Exec(`INSERT INTO entries_fts (entries_fts, rank) VALUES ('integrity-check', 1)`)
	require.NoError(t, err, "the FTS index still holds terms of rows deleted from entries")
	results, err = index.Search(context.Background(), Query{Kinds: []string{"prompt"}, Owner: "personal", Text: "pydantic"})
	require.NoError(t, err)
	require.Empty(t, results)

	moved := ownermap.OwnerMap{Owners: []string{"corp", "personal"}, Projects: map[string]string{"gateway": "corp", "notes": "corp"}}
	require.NoError(t, index.Reset(context.Background()))
	_, err = index.db.Exec(`INSERT INTO entries_fts (entries_fts, rank) VALUES ('integrity-check', 1)`)
	require.NoError(t, err, "Reset left FTS rows behind")
	summary, err = index.UpdateClaudeMem(context.Background(), directory, moved)
	require.NoError(t, err)
	require.Equal(t, 4, summary.Files)
	results, err = index.Search(context.Background(), Query{Owner: "corp", Text: "unrelated"})
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func kinds(results []Result) []string {
	var kinds []string
	for _, result := range results {
		kinds = append(kinds, result.Kind)
	}
	return kinds
}

// Transcript session abc1, plus a claude-mem export: no-transcript sessions abc2 (gateway) and p1 (notes), an orphan observation (m-gone), and abc1 rows the transcript already holds.
func searchFixture(t *testing.T) (*Index, string) {
	t.Helper()
	root := t.TempDir()
	repository := filepath.Join(root, "w", "gateway")
	require.NoError(t, os.MkdirAll(filepath.Join(repository, ".git"), 0o700))
	projects := filepath.Join(root, "projects")
	session := func(kind string, content string, timestamp string) string {
		return `{"type":"` + kind + `","cwd":"` + repository + `","sessionId":"abc1","timestamp":"` + timestamp + `","uuid":"` + lineID(kind, content, timestamp) + `","message":{"content":` + content + "}}\n"
	}
	write(t, filepath.Join(projects, "-gateway", "abc1.jsonl"),
		session("user", `"pin pydantic below 2.14"`, "2026-10-15T10:00:00Z")+
			session("assistant", `[{"type":"text","text":"Pinned pydantic in the wrapper and the pydantic tests pass."}]`, "2026-10-15T10:01:00Z"))

	claudeMem := filepath.Join(root, "archive", "claude-mem")
	require.NoError(t, os.MkdirAll(claudeMem, 0o700))
	ownerMap := fixtureOwnerMap
	require.NoError(t, ownermap.Save(filepath.Join(claudeMem, "owner-map.json"), ownerMap))
	writeLines(t, filepath.Join(claudeMem, "session-profiles.jsonl"),
		`{"contentSessionId":"abc1","memorySessionId":"m-new","profile":"claude","project":"gateway"}`,
		`{"contentSessionId":"abc2","memorySessionId":"m-old","profile":"","project":"gateway"}`,
		`{"contentSessionId":"p1","memorySessionId":"","profile":"","project":"notes"}`)
	writeLines(t, filepath.Join(claudeMem, "user_prompts.jsonl"),
		`{"id":1,"content_session_id":"abc1","prompt_text":"duplicate of the transcript","created_at":"2026-10-15T10:00:00Z"}`,
		`{"id":2,"content_session_id":"p1","prompt_text":"read about pydantic","created_at":"2026-06-01T00:00:00Z"}`)
	writeLines(t, filepath.Join(claudeMem, "tool_uses.jsonl"),
		`{"id":1,"content_session_id":"abc2","project":"gateway","cwd":"`+repository+`","tool_name":"Bash","tool_input":"{\"command\":\"uvx --with pydantic\"}","created_at":"2026-06-01T00:00:00Z"}`,
		`{"id":2,"content_session_id":"abc1","project":"gateway","cwd":"`+repository+`","tool_name":"Bash","tool_input":"{\"command\":\"duplicate\"}","created_at":"2026-10-15T00:00:00Z"}`)
	writeLines(t, filepath.Join(claudeMem, "session_summaries.jsonl"),
		`{"id":1,"memory_session_id":"m-old","project":"gateway","request":"fix chroma","learned":"pydantic 2.14 broke mcp","created_at":"2026-06-01T00:00:00Z"}`)
	writeLines(t, filepath.Join(claudeMem, "observations.jsonl"),
		`{"id":1,"memory_session_id":"m-gone","project":"gateway","title":"pydantic pin","narrative":"","created_at":"2026-06-01T00:00:00Z"}`)

	index := open(t, filepath.Join(root, "ele.db"))
	update(t, index, projects, filepath.Join(root, "archive", "transcripts"), ownerMap)
	summary, err := index.UpdateClaudeMem(context.Background(), claudeMem, ownerMap)
	require.NoError(t, err)
	require.Equal(t, Summary{Entries: 4, Files: 4}, summary)
	return index, claudeMem
}

func writeLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	write(t, path, content)
}
