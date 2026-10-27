package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

func TestSearchByOwnerFindsEverywhereNotes(t *testing.T) {
	index := open(t, filepath.Join(t.TempDir(), "ele.db"))
	directory := filepath.Join(t.TempDir(), "notes")
	for _, note := range []notes.Note{
		{Body: "Pin pydantic below 2.14.", Date: "2026-10-27", ID: "20261027-aaaaaa", Kind: notes.KindDecision, Owner: notes.OwnerEverywhere, Scope: notes.ScopeEverywhere},
		{Body: "Pin pydantic in the wrapper.", Date: "2026-10-27", ID: "20261027-bbbbbb", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal},
	} {
		_, err := notes.Write(directory, note)
		require.NoError(t, err)
	}
	_, err := index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)

	results, err := index.Search(context.Background(), Query{Owner: "personal", Text: "pydantic"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "20261027-aaaaaa", results[0].NoteID)
}

func TestUpdateNotesFollowsAMoveBetweenFolders(t *testing.T) {
	index := open(t, filepath.Join(t.TempDir(), "ele.db"))
	directory := filepath.Join(t.TempDir(), "notes")
	note := notes.Note{Body: "Pin pydantic below 2.14.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal}
	path, err := notes.Write(directory, note)
	require.NoError(t, err)
	_, err = index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)

	require.NoError(t, os.Remove(path))
	note.Owner = "personal"
	_, err = notes.Write(directory, note)
	require.NoError(t, err)
	summary, err := index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	require.Equal(t, Summary{Entries: 1}, summary)
	results, err := index.Search(context.Background(), Query{Owner: "personal", Text: "pydantic"})
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestUpdateNotesFollowsTheFiles(t *testing.T) {
	index := searchFixture(t)
	directory := filepath.Join(t.TempDir(), "notes")
	first := notes.Note{Body: "Pin pydantic below 2.14.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject}
	second := notes.Note{Body: "Pin pydantic through the wrapper, not the plugin.", Date: "2026-10-16", ID: "20261016-bbbbbb", Kind: notes.KindDecision, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Supersedes: []string{first.ID}}
	firstPath, err := notes.Write(directory, first)
	require.NoError(t, err)
	_, err = notes.Write(directory, second)
	require.NoError(t, err)

	summary, err := index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	require.Equal(t, Summary{Entries: 2}, summary)
	results, err := index.Search(context.Background(), Query{Owner: "corp", Text: "pydantic"})
	require.NoError(t, err)
	require.Equal(t, []string{"note", "note", "prompt", "conversation", "tool"}, kinds(results))
	require.Equal(t, "decision", results[0].NoteKind)
	superseded := map[string]string{}
	for _, result := range results[:2] {
		superseded[result.Text] = result.SupersededBy
	}
	require.Equal(t, map[string]string{"Pin «pydantic» below 2.14.": second.ID, "Pin «pydantic» through the wrapper, not the plugin.": ""}, superseded)

	summary, err = index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	require.Equal(t, Summary{}, summary)

	edited := first
	edited.Body = "Pin pydantic below 2.14 until chroma-mcp moves on."
	require.NoError(t, os.WriteFile(firstPath, notes.Format(edited), 0o600))
	require.NoError(t, os.Chtimes(firstPath, time.Now().Add(time.Minute), time.Now().Add(time.Minute)))
	summary, err = index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	require.Equal(t, Summary{Entries: 1}, summary)
	results, err = index.Search(context.Background(), Query{Kinds: []string{"note"}, Text: "chroma-mcp"})
	require.NoError(t, err)
	require.Len(t, results, 1)

	require.NoError(t, os.WriteFile(firstPath, []byte("no front matter\n"), 0o600))
	require.NoError(t, os.Chtimes(firstPath, time.Now().Add(2*time.Minute), time.Now().Add(2*time.Minute)))
	summary, err = index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	require.Len(t, summary.Problems, 1)
	require.Contains(t, summary.Problems[0], "notes/corp/20261015-aaaaaa.md: first line is not ---")
	results, err = index.Search(context.Background(), Query{Kinds: []string{"note"}, Text: "chroma-mcp"})
	require.NoError(t, err)
	require.Empty(t, results, "an invalid edit left the previous version indexed")

	require.NoError(t, os.Remove(firstPath))
	duplicate := second
	duplicate.Owner = "personal"
	_, err = notes.Write(directory, duplicate)
	require.NoError(t, err)
	summary, err = index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	require.Len(t, summary.Problems, 1)
	require.Contains(t, summary.Problems[0], "id 20261016-bbbbbb already used by notes/corp/20261016-bbbbbb.md")
	_, err = index.db.Exec(`INSERT INTO entries_fts (entries_fts, rank) VALUES ('integrity-check', 1)`)
	require.NoError(t, err)

	require.NoError(t, os.Remove(filepath.Join(directory, "corp", second.ID+".md")))
	_, err = index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	results, err = index.Search(context.Background(), Query{Kinds: []string{"note"}, Owner: "corp", Text: "wrapper"})
	require.NoError(t, err)
	require.Empty(t, results, "a deleted note's file left it indexed")

	_, err = notes.Write(directory, first)
	require.NoError(t, err)
	_, err = index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	require.NoError(t, index.Reset(context.Background()))
	var remaining int
	require.NoError(t, index.db.QueryRow(`SELECT count(*) FROM notes`).Scan(&remaining))
	require.Zero(t, remaining)
}

func TestUpdateNotesPicksTheLowestSuccessor(t *testing.T) {
	index := open(t, filepath.Join(t.TempDir(), "ele.db"))
	directory := filepath.Join(t.TempDir(), "notes")
	original := notes.Note{Body: "Pin pydantic below 2.14.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal}
	for _, note := range []notes.Note{
		original,
		{Body: "Later successor.", Date: "2026-10-17", ID: "20261017-cccccc", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Supersedes: []string{original.ID}},
	} {
		_, err := notes.Write(directory, note)
		require.NoError(t, err)
	}
	_, err := index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	_, err = notes.Write(directory, notes.Note{Body: "Earlier successor.", Date: "2026-10-16", ID: "20261016-bbbbbb", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Supersedes: []string{original.ID}})
	require.NoError(t, err)
	_, err = index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)

	results, err := index.Search(context.Background(), Query{Kinds: []string{"note"}, Text: "below"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "20261016-bbbbbb", results[0].SupersededBy)
}

func TestUpdateNotesReadsASupersedesListAnOlderEleIndexed(t *testing.T) {
	index := open(t, filepath.Join(t.TempDir(), "ele.db"))
	directory := filepath.Join(t.TempDir(), "notes")
	for _, note := range []notes.Note{
		{Body: "Pin pydantic below 2.14.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal},
		{Body: "Pin httpx below 0.28.", Date: "2026-10-15", ID: "20261015-bbbbbb", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal},
		{Body: "Pin both.", Date: "2026-10-16", ID: "20261016-cccccc", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Supersedes: []string{"20261015-aaaaaa", "20261015-bbbbbb"}},
	} {
		_, err := notes.Write(directory, note)
		require.NoError(t, err)
	}
	_, err := index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	_, err = index.db.Exec(`UPDATE notes SET supersedes = '20261015-aaaaaa, 20261015-bbbbbb' WHERE id = '20261016-cccccc'`)
	require.NoError(t, err)

	results, err := index.Search(context.Background(), Query{Kinds: []string{"note"}, Text: "below"})
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, result := range results {
		require.Equal(t, "20261016-cccccc", result.SupersededBy, "an older ele indexed the front matter value as written, space included")
	}
}

func TestUpdateNotesSkipsSymlinkedNoteFiles(t *testing.T) {
	index := open(t, filepath.Join(t.TempDir(), "ele.db"))
	directory := filepath.Join(t.TempDir(), "notes")
	_, err := notes.Write(directory, notes.Note{Body: "Pin pydantic below 2.14.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal})
	require.NoError(t, err)
	require.NoError(t, os.Symlink("someone@host.123:1700000000", filepath.Join(directory, "corp", ".#20261015-aaaaaa.md")))

	summary, err := index.UpdateNotes(context.Background(), directory)
	require.NoError(t, err)
	require.Equal(t, Summary{Entries: 1}, summary)
}

func TestUpdateNotesWithoutANotesDirectory(t *testing.T) {
	index := open(t, filepath.Join(t.TempDir(), "ele.db"))
	summary, err := index.UpdateNotes(context.Background(), filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, err)
	require.Equal(t, Summary{}, summary)
}
