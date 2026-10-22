package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

func TestDiscussionRegions(t *testing.T) {
	local := time.Local
	t.Cleanup(func() { time.Local = local })
	time.Local = time.FixedZone("UTC-10", -10*3600)
	entry := func(kind string, minute int, text string) index.Result {
		return index.Result{Kind: kind, Text: text, Timestamp: fmt.Sprintf("2026-10-15T11:%02d:00Z", minute)}
	}
	// "checkpoint" and "ticket" fill the session; only the decision turns, the note and its echo name "premises".
	entries := []index.Result{
		entry("prompt", 0, "Run the checkpoint on the ticket."),
		entry("conversation", 1, "The checkpoint passed; the ticket is closed."),
		entry("conversation", 2, "Another checkpoint, another ticket."),
		entry("prompt", 3, "Drop the shortcut: re-check premises every time."),
		entry("conversation", 4, "Decided: re-check premises every time, written this session or not."),
		entry("conversation", 5, "Checkpoint log for the ticket."),
		entry("conversation", 6, "More checkpoint work on another ticket."),
		{Kind: "note", Text: "Re-check premises every time.", Timestamp: "2026-10-15"},
		entry("conversation", 8, "Proposed note: re-check a ticket's premises every time, even ones written this session."),
		entry("conversation", 9, "Long after."),
		{Kind: "prompt", Text: "The evening, still the 15th locally.", Timestamp: "2026-10-16T05:00:00Z"},
	}
	note := notes.Note{Body: "Re-check a ticket's premises every time, including ones written this session.", Time: "2026-10-15T11:30:00Z"}
	tests := []struct {
		anchors    []int
		matched    bool
		name       string
		note       notes.Note
		separation int
	}{
		{anchors: []int{4, 8, 3}, matched: true, name: "common words do not count", note: notes.Note{Body: "They should note this: premises, every time, written.", Time: "2026-10-15T11:30:00Z"}, separation: 0},
		{anchors: []int{9}, matched: false, name: "no shared word falls back to the last entry before the note", note: notes.Note{Body: "Unrelated wording.", Time: "2026-10-15T11:30:00Z"}, separation: 0},
		{anchors: nil, matched: false, name: "nothing before the note", note: notes.Note{Body: "Anything.", Time: "2026-10-15T10:00:00Z"}, separation: 0},
		{anchors: []int{8, 4, 3}, matched: true, name: "rare words outrank the session's common ones, the echo included", note: note, separation: 0},
		{anchors: []int{8, 4}, matched: true, name: "regions keep apart", note: note, separation: 2},
		{anchors: []int{10}, matched: false, name: "untimed uses the end of its local date", note: notes.Note{Body: "Unrelated wording.", Date: "2026-10-15"}, separation: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			anchors, matched := discussionRegions(entries, test.note, test.separation)
			require.Equal(t, test.anchors, anchors)
			require.Equal(t, test.matched, matched)
		})
	}

	// Turns repeating the note's frequent words outscore the one holding its rare word when every word counts the same.
	report := []index.Result{
		entry("conversation", 0, "The start block reads the transcript."),
		entry("conversation", 1, "Start the transcript block again."),
		entry("conversation", 2, "Done: the start block and the transcript are kept."),
		entry("prompt", 3, "Should resume skip it?"),
		entry("conversation", 4, "The block starts from the transcript."),
	}
	anchors, matched := discussionRegions(report, notes.Note{Body: "Resume skips the start block because the transcript holds it.", Time: "2026-10-15T11:30:00Z"}, 0)
	require.True(t, matched)
	require.Equal(t, 3, anchors[0], "the rare word wins")

	anchors, matched = discussionRegions([]index.Result{entry("prompt", 0, "Skip resume.")}, notes.Note{Body: "Skip resume.", Time: "2026-10-15T11:30:00Z"}, 0)
	require.Equal(t, []int{0}, anchors)
	require.True(t, matched, "a word every entry holds still matches")

	onlyNote := []index.Result{{Kind: "note", Text: "alpha bravo", Timestamp: "2026-10-15T10:59:00Z"}, entry("prompt", 0, "charlie")}
	anchors, matched = discussionRegions(onlyNote, notes.Note{Body: "Alpha bravo.", Time: "2026-10-15T11:30:00Z"}, 0)
	require.Equal(t, []int{1}, anchors, "a note row never anchors")
	require.False(t, matched)
}

func TestDistinctiveWords(t *testing.T) {
	require.Equal(t, []string{"größe", "resume", "session-start", "again"}, distinctiveWords("Für die Größe: resume the session-start, and resume it again."))
}

func TestParseSince(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		since string
		value string
	}{
		{name: "date", since: "2026-10-01", value: "2026-10-01"},
		{name: "days back", since: "2026-10-08", value: "7d"},
		{name: "empty", since: "", value: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			since, err := parseSince(test.value, now)
			require.NoError(t, err)
			require.Equal(t, test.since, since)
		})
	}
	_, err := parseSince("last week", now)
	require.ErrorContains(t, err, "YYYY-MM-DD")
}

func TestPlural(t *testing.T) {
	require.Equal(t, "1 more entry", plural(1, "more entry", "more entries"))
	require.Equal(t, "2 more entries", plural(2, "more entry", "more entries"))
	require.Equal(t, "0 matches", plural(0, "match", "matches"))
}

func TestRunRecallFollowsAnOwnerMapEdit(t *testing.T) {
	home := recallFixture(t)
	writeFile(t, filepath.Join(home, "data", "owner-map.json"),
		`{"owners": ["corp", "personal"], "paths": {"`+filepath.Join(home, "w")+`": "personal"}}`)

	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "owner-map.json changed, so the index was rebuilt.")

	output, code = runCommand(t, "recall", "--no-sync", "--owner", "personal", "pydantic")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, `2 matches for "pydantic" in owner personal`)

	output, code = runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "rebuilt")
}

func TestRunRecallRefusesBadScopeAndKinds(t *testing.T) {
	recallFixture(t)

	tests := []struct {
		args    []string
		message string
		name    string
	}{
		{args: []string{"--all", "--owner", "corp", "x"}, message: "pass --all or --owner, not both", name: "all with owner"},
		{args: []string{"--kind", "tools", "x"}, message: `unknown kind "tools"`, name: "unknown kind"},
		{args: []string{"--owner", "corpe", "x"}, message: `unknown owner "corpe"; owner-map.json lists corp, personal`, name: "unknown owner"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, code := runCommand(t, append([]string{"recall", "--no-sync"}, test.args...)...)
			require.Equal(t, 2, code, output)
			require.Contains(t, output, test.message)
		})
	}

	output, code := runCommand(t, "recall", "--no-sync", "--all", "--kind", "prompt, conversation", "pydantic")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "2 matches")
}

func TestRunRecallScopesToTheWorkingDirectorysOwner(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	elsewhere := filepath.Join(home, "elsewhere")
	require.NoError(t, os.MkdirAll(elsewhere, 0o700))

	t.Chdir(gateway)
	output, code := runCommand(t, "recall", "--no-sync", "pydantic")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, `1 match for "pydantic" in owner corp, best first:`)
	require.Contains(t, output, "abc1 · 2026-10-15 · corp · gateway")
	require.Contains(t, output, "pin «pydantic» below 2.14")
	require.NotContains(t, output, "personal")

	output, code = runCommand(t, "recall", "pydantic", "--no-sync", "--all")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, `2 matches for "pydantic" in all owners`)

	output, code = runCommand(t, "recall", "--no-sync", "--all", "--limit", "1", "--", "pydantic", "--all")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, `No matches for "pydantic --all" in all owners.`)

	output, code = runCommand(t, "recall", "--no-sync", "--all", "--limit", "1", "pydantic")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, `Top 1 match (--limit) for "pydantic"`)

	t.Chdir(elsewhere)
	output, code = runCommand(t, "recall", "--no-sync", "pydantic")
	require.Equal(t, 1, code)
	require.Contains(t, output, "no owner in owner-map.json covers "+elsewhere)

	output, code = runCommand(t, "recall", "--no-sync", "--owner", "personal", "nothing-matches-this")
	require.Equal(t, 0, code, output)
	require.Equal(t, "No matches for \"nothing-matches-this\" in owner personal.\n", output)
}

func TestRunRecallSearchesWithoutWaitingForABusySync(t *testing.T) {
	home := recallFixture(t)
	unlock, err := archive.Lock(filepath.Join(home, "data", "archive", "transcripts"))
	require.NoError(t, err)
	defer unlock()

	done := make(chan string, 1)
	go func() {
		output, _ := runCommand(t, "recall", "--all", "pydantic")
		done <- output
	}()
	select {
	case output := <-done:
		require.Contains(t, output, "2 matches")
		require.NotContains(t, output, "sync failed")
	case <-time.After(10 * time.Second):
		t.Fatal("recall waited for the held archive lock")
	}
}

func TestRunRecallTakesTheOwnerFromANamedProject(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "personal", "notes"))

	output, code := runCommand(t, "recall", "--no-sync", "--project", "gateway", "pydantic")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, `1 match for "pydantic" in owner corp, best first:`)
	require.Contains(t, output, "abc1 · 2026-10-15 · corp · gateway")

	output, code = runCommand(t, "recall", "--no-sync", "--project", "gateway", "--owner", "personal", "pydantic")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, `No matches for "pydantic" in owner personal.`)

	output, code = runCommand(t, "recall", "--no-sync", "--all", "--project", "nowhere", "pydantic")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, `No matches for "pydantic" in all owners.`)

	output, code = runCommand(t, "recall", "--no-sync", "--project", "nowhere", "pydantic")
	require.Equal(t, 1, code)
	require.Contains(t, output, `project "nowhere" has no listed owner in owner-map.json or the index; pass --owner <name>`)
}

func TestRunReindexRebuildsFromTheArchiveAndNotes(t *testing.T) {
	home := recallFixture(t)
	noteDirectory := filepath.Join(home, "data", "notes")
	_, err := notes.Write(noteDirectory, notes.Note{Body: "Pin pydantic below 2.14.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject})
	require.NoError(t, err)
	_, err = notes.Write(noteDirectory, notes.Note{Body: "Pin pydantic through the wrapper.", Date: "2026-10-16", ID: "20261016-bbbbbb", Kind: notes.KindDecision, Owner: "corp", Project: "proxy", Scope: notes.ScopeProject, Supersedes: []string{"20261015-aaaaaa"}})
	require.NoError(t, err)
	writeFile(t, filepath.Join(noteDirectory, "corp", "20261015-broken.md"), "no front matter\n")

	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "; indexed 2 notes.")
	require.Contains(t, output, "Skipped a note: invalid note notes/corp/20261015-broken.md: first line is not ---")

	before, code := runCommand(t, "recall", "--no-sync", "--all", "pydantic")
	require.Equal(t, 0, code, before)
	require.Contains(t, before, "4 matches")
	require.Contains(t, before, "\n2026-10-15 · corp · gateway\n  note decision          (superseded by 20261016-bbbbbb) 20261015-aaaaaa Pin «pydantic» below 2.14.\n")
	require.Contains(t, before, "\n2026-10-16 · corp · proxy\n  note decision          20261016-bbbbbb Pin «pydantic» through the wrapper.\n")
	require.Contains(t, before, "  prompt                 pin «pydantic» below 2.14\n")
	require.Contains(t, before, "Expand a session")

	output, code = runCommand(t, "reindex")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Rebuilt the index from the archive and notes. Archived 0 B from 0 transcripts; indexed 3 entries from 2 files; indexed 2 notes.")

	after, code := runCommand(t, "recall", "--no-sync", "--all", "pydantic")
	require.Equal(t, 0, code, after)
	require.Equal(t, before, after)
}

func TestRunShowCentresOnAMatch(t *testing.T) {
	recallFixture(t)

	output, code := runCommand(t, "show", "abc", "--around", "WRAPPER", "--context", "0")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "Session abc1 · corp · gateway · claude · entries 2–2 of 2")
	require.Contains(t, output, "conversation Pinned it in the wrapper")
	require.NotContains(t, output, "pin pydantic below 2.14")

	output, code = runCommand(t, "show", "abc", "--at", "2026-10-15T10:00:30Z", "--context", "0")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "entries 2–2 of 2")
	require.Contains(t, output, "conversation Pinned it in the wrapper")

	output, code = runCommand(t, "show", "abc", "--at", "2026-10-15T10:01:00Z", "--context", "0")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "entries 2–2 of 2")

	output, code = runCommand(t, "show", "abc", "--at", "2026-10-15T10:00", "--context", "0")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "entries 1–1 of 2")

	output, code = runCommand(t, "show", "abc", "--at", "2026-10-15T10:02:00Z")
	require.Equal(t, 1, code)
	require.Contains(t, output, "no entry in session abc1 is at or after 2026-10-15T10:02:00Z")

	output, code = runCommand(t, "show", "abc", "--at", "2026-10-15T10:00:00Z", "--around", "wrapper")
	require.Equal(t, 2, code, output)

	output, code = runCommand(t, "show", "zzz")
	require.Equal(t, 1, code)
	require.Contains(t, output, `no indexed session starts with "zzz"`)

	output, code = runCommand(t, "show", "abc", "--context", "-1")
	require.Equal(t, 2, code)
	require.Contains(t, output, "--context must be 0 or more")
}

func TestRunShowCentresOnTheDiscussionNotItsNote(t *testing.T) {
	home := recallFixture(t)
	noteDirectory := filepath.Join(home, "data", "notes")
	// Dated on the fixture's day, so the note sorts before the discussion it shares a word with.
	for _, note := range []notes.Note{
		{Body: "Keep the wrapper pin.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Session: "abc1"},
		{Body: "Only a note here.", Date: "2026-10-15", ID: "20261015-bbbbbb", Kind: notes.KindDecision, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject, Session: "lonely1"},
	} {
		_, err := notes.Write(noteDirectory, note)
		require.NoError(t, err)
	}
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "show", "abc1", "--around", "wrapper", "--context", "0")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "conversation Pinned it in the wrapper")
	require.NotContains(t, output, "Keep the wrapper pin.")

	output, code = runCommand(t, "show", "lonely", "--around", "only", "--context", "0")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "note         Only a note here.")
}

func TestRunShowNoteKeepsTheEntryBeforeItWhenTheWindowOverflows(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	var transcript strings.Builder
	for minute := range 20 {
		text := fmt.Sprintf("entry %02d %s", minute, strings.Repeat("long ", 150))
		if minute == 19 {
			text = "the last word before the note"
		}
		fmt.Fprintf(&transcript, `{"type":"user","cwd":%q,"sessionId":"long1","timestamp":"2026-10-15T11:%02d:00Z","uuid":"u%02d","message":{"content":%q}}`+"\n", gateway, minute, minute, text)
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "long1.jsonl"), transcript.String())
	_, err := notes.Write(filepath.Join(home, "data", "notes"), notes.Note{Body: "Decided.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Session: "long1", Time: "2026-10-15T12:00:00Z"})
	require.NoError(t, err)
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "show", "--note", "20261015-aaaaaa", "--context", "10")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "prompt       the last word before the note\n… ")
	require.Regexp(t, `^Session long1 · corp · 21 entries · note 20261015-aaaaaa's discussion\n— entries (\d+)–21\n`, output)
	require.Contains(t, output, "entry 18 long")
	require.Regexp(t, `\n… \d+ more entries not shown \(output limit\); ele show long1 --around <text> reaches them\n$`, output)
	require.LessOrEqual(t, len(output), outputBudget)
}

func TestRunShowNoteShowsEachPartOfItsDiscussion(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	texts := []string{
		"Run the checkpoint.", "The checkpoint passed.", "Another checkpoint.",
		"Should resume skip the start block? " + strings.Repeat("reasoning ", 140) + "END-OF-ANSWER",
		"Checkpoint again.", "More checkpoint work.", "Checkpoint log.", "Checkpoint stats.", "Checkpoint review.",
		"Proposed: resume skips the start block.", "Checkpoint done.",
	}
	var transcript strings.Builder
	for minute, text := range texts {
		fmt.Fprintf(&transcript, `{"type":"user","cwd":%q,"sessionId":"parts1","timestamp":"2026-10-15T11:%02d:00Z","uuid":"p%02d","message":{"content":%q}}`+"\n", gateway, minute, minute, text)
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "parts1.jsonl"), transcript.String())
	_, err := notes.Write(filepath.Join(home, "data", "notes"), notes.Note{Body: "Resume skips the start block.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Session: "parts1", Time: "2026-10-15T12:00:00Z"})
	require.NoError(t, err)
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "show", "--note", "20261015-aaaaaa")
	require.Equal(t, 0, code, output)
	require.True(t, strings.HasPrefix(output, "Session parts1 · corp · 12 entries · note 20261015-aaaaaa's discussion\n— entries 4–6\n"), output)
	require.Contains(t, output, "END-OF-ANSWER", "an anchor is shown past textLimit")
	require.Contains(t, output, "\n— entries 10–12\n")
	require.Less(t, strings.Index(output, "— entries 4–6"), strings.Index(output, "— entries 10–12"), "parts in session order")
	require.Contains(t, output, "Another checkpoint.", "one neighbour before")
	require.NotContains(t, output, "The checkpoint passed.", "not two")
	require.LessOrEqual(t, len(output), outputBudget)

	output, code = runCommand(t, "show", "--note", "20261015-aaaaaa", "--context", "0")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "Another checkpoint.", "--context 0 shows no neighbour")
	require.Contains(t, output, "END-OF-ANSWER", "but still the part itself")
	output, code = runCommand(t, "show", "--note", "20261015-aaaaaa", "--context", "2")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "The checkpoint passed.", "--context 2 shows two")
}

func TestRunShowNoteKeepsPartsApartAndWithinBudget(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	wide := strings.Repeat("├── ", 600)
	texts := []string{"Filler one.", "Should resume skip? " + wide, "Filler two.", "Resume skip, again. " + wide, "Filler three.", "Filler four.", "Filler five.", "Resume skip, a third time. " + wide, "Filler six."}
	var transcript strings.Builder
	for minute, text := range texts {
		fmt.Fprintf(&transcript, `{"type":"user","cwd":%q,"sessionId":"apart1","timestamp":"2026-10-15T11:%02d:00Z","uuid":"a%02d","message":{"content":%q}}`+"\n", gateway, minute, minute, text)
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "apart1.jsonl"), transcript.String())
	_, err := notes.Write(filepath.Join(home, "data", "notes"), notes.Note{Body: "Resume skips.", Date: "2026-10-15", ID: "20261015-bbbbbb", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Session: "apart1", Time: "2026-10-15T12:00:00Z"})
	require.NoError(t, err)
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "show", "--note", "20261015-bbbbbb", "--context", "1")
	require.Equal(t, 0, code, output)
	require.LessOrEqual(t, len(output), outputBudget, "wide characters stay within the byte budget")
	require.Equal(t, 2, strings.Count(output, "— entries "), "the strong entry two after the first gets no part of its own")
	require.NotContains(t, output, "Resume skip, again.")
	require.Contains(t, output, "Resume skip, a third time.")
}

func TestRunShowOpensANotesDiscussion(t *testing.T) {
	local := time.Local
	t.Cleanup(func() { time.Local = local })
	time.Local = time.UTC
	home := recallFixture(t)
	noteDirectory := filepath.Join(home, "data", "notes")
	for _, note := range []notes.Note{
		{Body: "After the prompt.", Date: "2026-10-15", ID: "20261015-aaaaaa", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Session: "abc1", Time: "2026-10-15T10:00:30Z"},
		{Body: "After the reply.", Date: "2026-10-15", ID: "20261015-bbbbbb", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Session: "abc1", Time: "2026-10-15T10:05:00Z"},
		{Body: "Untimed.", Date: "2026-10-15", ID: "20261015-cccccc", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Session: "abc1"},
		{Body: "No session.", Date: "2026-10-15", ID: "20261015-dddddd", Kind: notes.KindDecision, Owner: "personal", Scope: notes.ScopeGlobal},
		{Body: "Before anything.", Date: "2026-10-15", ID: "20261015-eeeeee", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Session: "abc1", Time: "2026-10-15T09:00:00Z"},
		{Body: "Prefix only.", Date: "2026-10-15", ID: "20261015-ffffff", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Session: "abc", Time: "2026-10-15T10:05:00Z"},
		{Body: "Pydantic stays below the release that broke it.", Date: "2026-10-15", ID: "20261015-gggggg", Kind: notes.KindDecision, Owner: "corp", Scope: notes.ScopeGlobal, Session: "abc1", Time: "2026-10-15T10:05:00Z"},
	} {
		_, err := notes.Write(noteDirectory, note)
		require.NoError(t, err)
	}
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)

	tests := []struct {
		args []string
		code int
		name string
		want string
	}{
		{args: []string{"--note", "20261015-dddddd"}, code: 1, name: "no session", want: "note 20261015-dddddd records no session"},
		{args: []string{"--note", "../x"}, code: 1, name: "not an id", want: `"../x" is not a note id`},
		{args: []string{"--note", "20261015-aaaaaa", "--around", "pin"}, code: 2, name: "note and around together", want: "ele show --note <id>"},
		{args: []string{"--note", "20261015-aaaaaa", "abc1"}, code: 2, name: "note and session together", want: "ele show --note <id>"},
		{args: []string{"--note", "20261015-eeeeee"}, code: 1, name: "nothing precedes the note", want: "no entry in session abc1 precedes note 20261015-eeeeee"},
		{args: []string{"--note", "20261015-ffffff"}, code: 1, name: "session matched only as a prefix", want: "no entry in session abc precedes note 20261015-ffffff"},
		{args: []string{"--note", "20261015-zzzzzz"}, code: 1, name: "unknown note", want: "no note 20261015-zzzzzz in"},
		{args: []string{"--note", "20261015-cccccc"}, code: 0, name: "untimed falls back to the end of its date", want: "conversation Pinned it in the wrapper.\n"},
		{args: []string{"--note", "20261015-aaaaaa"}, code: 0, name: "written after the prompt", want: "prompt       pin pydantic below 2.14\n"},
		{args: []string{"--note", "20261015-bbbbbb"}, code: 0, name: "written after the reply", want: "conversation Pinned it in the wrapper.\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shown, exit := runCommand(t, append(append([]string{"show"}, test.args...), "--context", "0")...)
			require.Equal(t, test.code, exit, shown)
			require.Contains(t, shown, test.want)
		})
	}

	output, code = runCommand(t, "show", "--note", "20261015-aaaaaa", "--context", "1")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "prompt       pin pydantic below 2.14\n")
	require.NotContains(t, output, "Pinned it in the wrapper")

	output, code = runCommand(t, "show", "--note", "20261015-gggggg", "--context", "1")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "prompt       pin pydantic below 2.14\n")
	require.Contains(t, output, "conversation Pinned it in the wrapper.\n")
}

func TestWriteResultsOffersShowOnlyForSessions(t *testing.T) {
	var output bytes.Buffer
	writeResults(&output, "pydantic", index.Query{}, []index.Result{{Kind: "note", NoteKind: "decision", Owner: "corp", Text: "Pin pydantic.", Timestamp: "2026-10-15"}})
	require.NotContains(t, output.String(), "Expand a session")
}

func TestWriteResultsStaysWithinTheBudget(t *testing.T) {
	for words := 2; words <= 60; words++ {
		var results []index.Result
		for session := range 40 {
			length := words
			if session%2 == 1 {
				length = 1
			}
			for range 3 {
				results = append(results, index.Result{Kind: "conversation", Owner: "corp", Project: "gateway", Session: fmt.Sprintf("session-%02d", session), Text: strings.Repeat("word ", length), Timestamp: "2026-10-15T10:00:00Z"})
			}
		}
		var output bytes.Buffer
		writeResults(&output, strings.Repeat("query ", 50), index.Query{Limit: len(results), Owner: "corp"}, results)

		require.LessOrEqual(t, output.Len(), outputBudget, "%d words per match", words)
		next, cut := 0, false
		for block := range strings.SplitSeq(output.String(), "\n\n") {
			if !strings.HasPrefix(block, "session-") {
				continue
			}
			require.False(t, cut, "%d words per match: a session printed after one was cut short", words)
			require.Contains(t, block, "\n  conversation", "%d words per match: a session heading printed without a match under it", words)
			require.True(t, strings.HasPrefix(block, fmt.Sprintf("session-%02d ", next)), "%d words per match: sessions skipped out of rank order at %q", words, block[:10])
			cut = strings.Count(block, "\n  conversation") < 3
			next++
		}
	}
}

// Transcript lines carry unique ids, and the index keeps one entry per id and place.
func lineID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:8])
}

// recallFixture syncs one corp session (abc1) and one personal session (def2) into a fresh data dir.
func recallFixture(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ELE_HOME", filepath.Join(home, "data"))
	t.Setenv("NO_COLOR", "1")
	gateway := filepath.Join(home, "w", "corp", "gateway")
	notes := filepath.Join(home, "w", "personal", "notes")
	for _, repository := range []string{gateway, notes} {
		require.NoError(t, os.MkdirAll(filepath.Join(repository, ".git"), 0o700))
	}
	line := func(session string, directory string, kind string, content string, minute string) string {
		return `{"type":"` + kind + `","cwd":"` + directory + `","sessionId":"` + session + `","timestamp":"2026-10-15T10:` + minute + `:00Z","uuid":"` + lineID(session, kind, content, minute) + `","message":{"content":` + content + "}}\n"
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "abc1.jsonl"),
		line("abc1", gateway, "user", `"pin pydantic below 2.14"`, "00")+
			line("abc1", gateway, "assistant", `[{"type":"text","text":"Pinned it in the wrapper."}]`, "01"))
	writeFile(t, filepath.Join(home, ".claude", "projects", "-notes", "def2.jsonl"),
		line("def2", notes, "user", `"read up on pydantic"`, "05"))
	writeFile(t, filepath.Join(home, "data", "owner-map.json"),
		`{"owners": ["corp", "personal"], "paths": {"`+filepath.Join(home, "w", "corp")+`": "corp", "`+filepath.Join(home, "w", "personal")+`": "personal"}}`)

	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	return home
}

func runCommand(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var output bytes.Buffer
	code := run(context.Background(), args, &output, &output)
	return output.String(), code
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
