package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var valid = Note{Body: "Pin pydantic below 2.14 in the uvx wrapper.", Date: "2026-10-15", ID: "20261015-a1b2c3", Kind: KindDecision, Owner: "personal", Profile: "claude", Project: "claude-ephant", Scope: ScopeProject, Session: "010a2385"}

func TestClearShownRemovesOnlyThatKey(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "personal")
	require.NoError(t, os.MkdirAll(directory, 0o700))
	path := filepath.Join(directory, "20261016-a1b2c3.md")
	handEdited := "---\nkind:  handoff \nshown : 2026-10-16T08:00:00Z\ndate: 2026-10-16\nowner: personal\nproject: claude-ephant\nscope: project\n---\n\nshown: kept, it is body.\n"
	require.NoError(t, os.WriteFile(path, []byte(handEdited), 0o644))
	note := Note{ID: "20261016-a1b2c3", Owner: "personal"}

	require.NoError(t, ClearShown(root, note))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm(), "a hand-made note keeps its mode")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "---\nkind:  handoff \ndate: 2026-10-16\nowner: personal\nproject: claude-ephant\nscope: project\n---\n\nshown: kept, it is body.\n", string(content))

	cleared := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(path, cleared, cleared))
	require.NoError(t, ClearShown(root, note))
	info, err = os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, cleared, info.ModTime().UTC(), "a note with no shown key is not rewritten")
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temporary file left beside the note")

	require.NoError(t, os.WriteFile(filepath.Join(directory, "broken.md"), []byte("---\nkind: handoff\n"), 0o600))
	require.ErrorIs(t, ClearShown(root, Note{ID: "broken", Owner: "personal"}), ErrInvalid)
}

func TestMarkShownRewritesOnlyThatKey(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "personal")
	require.NoError(t, os.MkdirAll(directory, 0o700))
	path := filepath.Join(directory, "20261016-a1b2c3.md")
	handEdited := "---\nkind:  handoff \ndate: 2026-10-16\nowner: personal\nproject: claude-ephant\nscope: project\n---\n\nNext: item 3.\n  Kept as typed.\n"
	require.NoError(t, os.WriteFile(path, []byte(handEdited), 0o644))
	note := Note{ID: "20261016-a1b2c3", Owner: "personal"}

	require.NoError(t, MarkShown(root, note, time.Date(2026, 10, 16, 9, 0, 0, 0, time.FixedZone("x", 3600))))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm(), "a hand-made note keeps its mode")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "---\nkind:  handoff \ndate: 2026-10-16\nowner: personal\nproject: claude-ephant\nscope: project\nshown: 2026-10-16T08:00:00Z\n---\n\nNext: item 3.\n  Kept as typed.\n", string(content))

	require.NoError(t, MarkShown(root, note, time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)))
	content, err = os.ReadFile(path)
	require.NoError(t, err)
	marked, err := Parse(path, content)
	require.NoError(t, err)
	require.Equal(t, "2026-10-17T00:00:00Z", marked.Shown, "a second mark replaces the first")
	require.Equal(t, 1, strings.Count(string(content), "shown:"))
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temporary file left beside the note")

	require.NoError(t, os.WriteFile(filepath.Join(directory, "broken.md"), []byte("---\nkind: handoff\n"), 0o600))
	require.ErrorIs(t, MarkShown(root, Note{ID: "broken", Owner: "personal"}, time.Now()), ErrInvalid)
}

func TestMarkClosedThenClearClosedRoundTrips(t *testing.T) {
	root := t.TempDir()
	reminder := Note{Body: "Revisit worktree-parallel plans.", Date: "2026-10-22", Due: "2026-11-01", ID: "20261022-a1b2c3", Kind: KindReminder, Owner: "personal", Scope: ScopeGlobal}
	path, err := Write(root, reminder)
	require.NoError(t, err)

	require.NoError(t, MarkClosed(root, reminder, time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "---\ndate: 2026-10-22\ndue: 2026-11-01\nkind: reminder\nowner: personal\nscope: global\nclosed: 2026-11-01T09:00:00Z\n---\nRevisit worktree-parallel plans.\n", string(content))
	closed, err := Parse(path, content)
	require.NoError(t, err)
	require.Equal(t, "2026-11-01T09:00:00Z", closed.Closed)
	require.Empty(t, closed.Shown, "closing is not showing")

	require.NoError(t, ClearClosed(root, reminder))
	content, err = os.ReadFile(path)
	require.NoError(t, err)
	reopened, err := Parse(path, content)
	require.NoError(t, err)
	require.Equal(t, reminder, reopened)
}

func TestMoveRefusesToOverwriteANoteUnderTheNewOwner(t *testing.T) {
	root := t.TempDir()
	note := valid
	note.Owner = "corp"
	_, err := Write(root, note)
	require.NoError(t, err)
	note.Owner = "personal"
	_, err = Write(root, note)
	require.NoError(t, err)

	note.Owner = "corp"
	require.ErrorContains(t, Move(root, note, "personal"), "already exists")
	require.FileExists(t, filepath.Join(root, "corp", note.ID+".md"))
}

func TestMoveRewritesOnlyTheOwnerAndRefilesTheNote(t *testing.T) {
	root := t.TempDir()
	from := filepath.Join(root, "corp", "20261017-a1b2c3.md")
	handEdited := "---\nkind:  decision \ndate: 2026-10-17\nowner : corp\nproject: gateway\nscope: project\n---\n\nowner: corp stays, it is body.\n"
	require.NoError(t, os.MkdirAll(filepath.Dir(from), 0o700))
	require.NoError(t, os.WriteFile(from, []byte(handEdited), 0o644))

	require.NoError(t, Move(root, Note{ID: "20261017-a1b2c3", Owner: "corp"}, "personal"))
	require.NoFileExists(t, from)
	to := filepath.Join(root, "personal", "20261017-a1b2c3.md")
	content, err := os.ReadFile(to)
	require.NoError(t, err)
	require.Equal(t, "---\nkind:  decision \ndate: 2026-10-17\nowner: personal\nproject: gateway\nscope: project\n---\n\nowner: corp stays, it is body.\n", string(content))
	info, err := os.Stat(to)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	_, err = Parse(to, content)
	require.NoError(t, err, "the moved note's owner matches its new folder")
}

func TestNewIDIsDatedAndUnique(t *testing.T) {
	first, err := NewID(time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	second, err := NewID(time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Regexp(t, `^20261015-[0-9a-f]{6}$`, first)
	require.NotEqual(t, first, second)
}

func TestParseReadsAHandEditedNote(t *testing.T) {
	note, err := Parse("notes/personal/20261015-a1b2c3.md", []byte("\xef\xbb\xbf---\nkind:  question \nscope: global\nowner: personal\nsupersedes: 20261014-aaaaaa ,20261014-bbbbbb,\ndate: 2026-10-15\n---\n\nShould ele index tool output?\nIt would triple the index.\n\n"))
	require.NoError(t, err)
	require.Equal(t, Note{Body: "Should ele index tool output?\nIt would triple the index.", Date: "2026-10-15", ID: "20261015-a1b2c3", Kind: KindQuestion, Owner: "personal", Scope: ScopeGlobal, Supersedes: []string{"20261014-aaaaaa", "20261014-bbbbbb"}}, note)
}

func TestParseRejectsInvalidNotes(t *testing.T) {
	front := "---\ndate: 2026-10-15\nkind: decision\nowner: personal\nscope: global\n"
	tests := []struct {
		content string
		message string
		name    string
		path    string
	}{
		{content: front + "kind: fact\n---\nx\n", message: `front matter key "kind" appears twice`, name: "duplicate key", path: "notes/personal/n.md"},
		{content: front + "---\n\n", message: "notes/personal/n.md: empty body", name: "empty body", path: "notes/personal/n.md"},
		{content: front + "closed: 2026-10-22T09:00:00Z\n---\nx\n", message: "decision with a closed, done-file, due or every key, which only a reminder takes", name: "closed on a decision", path: "notes/personal/n.md"},
		{content: front + "due: 2026-10-23\n---\nx\n", message: "decision with a closed, done-file, due or every key, which only a reminder takes", name: "due on a decision", path: "notes/personal/n.md"},
		{content: front + "---\nx\n", message: "no id", name: "empty id", path: "notes/personal/.md"},
		{content: "No front matter\n", message: "first line is not ---", name: "first line not a fence", path: "notes/personal/n.md"},
		{content: front, message: "front matter never closes", name: "front matter never closes", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-27\nkind: decision\nowner: personal\nscope: everywhere\n---\nx\n", message: `everywhere scope but owner "personal"`, name: "everywhere scope under an owner", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-27\nkind: handoff\nowner: _all\nscope: everywhere\n---\nx\n", message: "handoff with everywhere scope", name: "everywhere handoff", path: "notes/_all/n.md"},
		{content: "---\ndate: 2026-10-27\nkind: decision\nowner: _all\nscope: global\n---\nx\n", message: `owner _all with scope "global"`, name: "everywhere owner with another scope", path: "notes/_all/n.md"},
		{content: "---\ndate: 2026-10-15\nkind: handoff\nowner: personal\nscope: global\n---\nx\n", message: "handoff with global scope", name: "global handoff", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-17\nkind: plan\nowner: personal\nscope: global\n---\nplans.md\n", message: "plan with global scope", name: "global plan", path: "notes/personal/n.md"},
		{content: "---\ndate: 15 Oct\nkind: decision\nowner: personal\nscope: global\n---\nx\n", message: "is not YYYY-MM-DD", name: "loose date", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-15\nkind: decision\nowner: corp\nscope: global\n---\nx\n", message: `owner "corp" but filed under "personal"`, name: "owner differs from folder", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-17\nkind: plan\nowner: personal\nproject: claude-ephant\nscope: project\n---\n../elsewhere/plans.md\n", message: `plan body "../elsewhere/plans.md" is not one path inside the project`, name: "plan outside the project", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-17\nkind: plan\nowner: personal\nproject: claude-ephant\nscope: project\n---\n/w/plans.md\n", message: "is not one path inside the project", name: "plan with an absolute path", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-17\nkind: plan\nowner: personal\nproject: claude-ephant\nscope: project\n---\nplans.md\nnext.md\n", message: "is not one path inside the project", name: "plan with two lines", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-15\nkind: decision\nowner: personal\nscope: project\n---\nx\n", message: "project scope with no project", name: "project scope without project", path: "notes/personal/n.md"},
		{content: "---\nclosed: today\ndate: 2026-10-22\ndue: 2026-10-23\nkind: reminder\nowner: personal\nscope: global\n---\nx\n", message: `closed "today" is not an RFC 3339 timestamp`, name: "reminder closed not RFC 3339", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-22\ndue: next week\nkind: reminder\nowner: personal\nscope: global\n---\nx\n", message: `due "next week" is not YYYY-MM-DD or YYYY-MM-DDTHH:MM`, name: "reminder due not a date", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-22\ndue: 2026-10-23T9:30\nkind: reminder\nowner: personal\nscope: global\n---\nx\n", message: `due "2026-10-23T9:30" is not YYYY-MM-DD or YYYY-MM-DDTHH:MM`, name: "reminder due hour not zero-padded", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-22\nkind: reminder\nowner: personal\nscope: global\n---\nx\n", message: "reminder with no due date", name: "reminder without due", path: "notes/personal/n.md"},
		{content: front + "every: 7\n---\nx\n", message: "decision with a closed, done-file, due or every key, which only a reminder takes", name: "every on a decision", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-22\ndone-file: /a\x07b\ndue: 2026-10-23\nevery: 7\nkind: reminder\nowner: personal\nscope: global\n---\nx\n", message: `done-file "/a\ab" holds a control character`, name: "reminder done-file with a control character", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-22\ndone-file: /m\ndue: 2026-10-23\nkind: reminder\nowner: personal\nscope: global\n---\nx\n", message: "done-file with no every", name: "reminder done-file without every", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-22\ndone-file: cache/m\ndue: 2026-10-23\nevery: 7\nkind: reminder\nowner: personal\nscope: global\n---\nx\n", message: `done-file "cache/m" is not an absolute path`, name: "reminder done-file relative", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-22\ndue: 2026-10-23\nevery: 0\nkind: reminder\nowner: personal\nscope: global\n---\nx\n", message: `every "0" is not minutes, hours or days above 0, such as 5m, 2h or 7d`, name: "reminder every 0", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-22\ndue: 2026-10-23\nevery: weekly\nkind: reminder\nowner: personal\nscope: global\n---\nx\n", message: `every "weekly" is not minutes, hours or days above 0, such as 5m, 2h or 7d`, name: "reminder every not a number", path: "notes/personal/n.md"},
		{content: front + "project: a\x07b\n---\nx\n", message: `project "a\ab" holds a control character`, name: "project with a control character", path: "notes/personal/n.md"},
		{content: front + "project: a\u0085b\n---\nx\n", message: `project "a\u0085b" holds a control character`, name: "project with a C1 control character", path: "notes/personal/n.md"},
		{content: front + "session: a b\n---\nx\n", message: `session "a b" holds characters other than letters, digits, - and _`, name: "session with a space", path: "notes/personal/n.md"},
		{content: front + "shown: yesterday\n---\nx\n", message: `shown "yesterday" is not an RFC 3339 timestamp`, name: "shown not RFC 3339", path: "notes/personal/n.md"},
		{content: front + "supersedes: a, a\n---\nx\n", message: `supersedes names "a" twice`, name: "supersedes an id twice", path: "notes/personal/n.md"},
		{content: front + "supersedes: a, n\n---\nx\n", message: "supersedes itself", name: "supersedes itself", path: "notes/personal/n.md"},
		{content: front + "time: 21:52\n---\nx\n", message: `time "21:52" is not an RFC 3339 timestamp`, name: "time not RFC 3339", path: "notes/personal/n.md"},
		{content: front + " tag : x\n---\nx\n", message: `unknown front matter key "tag"`, name: "unknown key", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-15\nkind: todo\nowner: personal\nscope: global\n---\nx\n", message: `kind "todo" is not one of`, name: "unknown kind", path: "notes/personal/n.md"},
		{content: "---\ndate: 2026-10-15\nkind: decision\nowner: personal\nscope: team\n---\nx\n", message: `scope "team" is not one of`, name: "unknown scope", path: "notes/personal/n.md"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse(test.path, []byte(test.content))
			require.ErrorIs(t, err, ErrInvalid)
			require.ErrorContains(t, err, test.message)
		})
	}
}

func TestParseReadsAPlanNote(t *testing.T) {
	note, err := Parse("notes/personal/20261017-a1b2c3.md", []byte("---\ndate: 2026-10-17\nkind: plan\nowner: personal\nproject: claude-ephant\nscope: project\n---\n.documentation/plans/plans.md\n"))
	require.NoError(t, err)
	require.Equal(t, ".documentation/plans/plans.md", note.Body)
}

func TestReadForAddsEverywhereNotesToTheOwners(t *testing.T) {
	directory := t.TempDir()
	_, err := Write(directory, valid)
	require.NoError(t, err)
	everywhere := Note{Body: "The machine runs macOS.", Date: "2026-10-27", ID: "20261027-eeeeee", Kind: KindFact, Owner: OwnerEverywhere, Scope: ScopeEverywhere}
	_, err = Write(directory, everywhere)
	require.NoError(t, err)
	other := valid
	other.ID, other.Owner = "20261015-dddddd", "corp"
	_, err = Write(directory, other)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(directory, OwnerEverywhere, "broken.md"), []byte("no front matter\n"), 0o600))

	read, invalid, err := ReadFor(directory, "personal")
	require.NoError(t, err)
	require.Equal(t, []Note{valid, everywhere}, read)
	require.Equal(t, 1, invalid)

	read, invalid, err = ReadFor(directory, "")
	require.NoError(t, err)
	require.Equal(t, []Note{everywhere}, read, "a session no owner covers reads only everywhere notes")
	require.Equal(t, 1, invalid)
}

func TestReadOwnerSkipsInvalidNotesAndOtherOwners(t *testing.T) {
	directory := t.TempDir()
	_, err := Write(directory, valid)
	require.NoError(t, err)
	other := valid
	other.ID, other.Owner = "20261015-dddddd", "corp"
	_, err = Write(directory, other)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(directory, "personal", "broken.md"), []byte("no front matter\n"), 0o600))
	require.NoError(t, os.Symlink("gone", filepath.Join(directory, "personal", ".#lock.md")))

	read, invalid, err := ReadOwner(directory, "personal")
	require.NoError(t, err)
	require.Equal(t, []Note{valid}, read)
	require.Equal(t, 1, invalid)

	read, invalid, err = ReadOwner(filepath.Join(directory, "missing"), "personal")
	require.NoError(t, err)
	require.Empty(t, read)
	require.Zero(t, invalid)
}

func TestSetDueRewritesOnlyTheDueOfANudge(t *testing.T) {
	root := t.TempDir()
	nudge := Note{Body: "Run /hook-stats.", Date: "2026-10-22", DoneFile: "/home/u/.claude/cache/hook-stats.last", Due: "2026-10-22", Every: 7 * 24 * time.Hour, ID: "20261022-a1b2c3", Kind: KindReminder, Owner: "personal", Scope: ScopeGlobal}
	path, err := Write(root, nudge)
	require.NoError(t, err)

	require.NoError(t, SetDue(root, nudge, "2026-10-29"))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "---\ndate: 2026-10-22\ndone-file: /home/u/.claude/cache/hook-stats.last\ndue: 2026-10-29\nevery: 7\nkind: reminder\nowner: personal\nscope: global\n---\nRun /hook-stats.\n", string(content))
	rearmed, err := Parse(path, content)
	require.NoError(t, err)
	nudge.Due = "2026-10-29"
	require.Equal(t, nudge, rearmed)

	require.ErrorIs(t, SetDue(root, nudge, "next week"), ErrInvalid)
	require.NoError(t, SetDue(root, nudge, "2026-10-29T15:05"), "a sub-day nudge re-arms to a time")
	require.ErrorIs(t, SetDue(root, nudge, ""), ErrInvalid)
}

func TestEveryRoundTripsEachUnit(t *testing.T) {
	tests := []struct {
		every   time.Duration
		value   string
		written string
	}{
		{every: 2 * 24 * time.Hour, value: "2d", written: "2"},
		{every: 2 * time.Hour, value: "2h", written: "2h"},
		{every: 5 * time.Minute, value: "5m", written: "5m"},
		{every: 7 * 24 * time.Hour, value: "7", written: "7"},
		{every: 24 * time.Hour, value: "24h", written: "1"},
		{every: 90 * time.Minute, value: "90m", written: "90m"},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			every, err := ParseEvery(test.value)
			require.NoError(t, err)
			require.Equal(t, test.every, every)
			require.Equal(t, test.written, FormatEvery(every), "whole days are written bare, as builds before sub-day intervals read them")
		})
	}
	for _, value := range []string{"", "0", "0m", "-1", "1h30m", "30s", "7 d", "d", "99999999999999999999"} {
		_, err := ParseEvery(value)
		require.Error(t, err, value)
	}
}

func TestWriteRefusesANoteWithoutAnID(t *testing.T) {
	note := valid
	note.ID = ""
	_, err := Write(t.TempDir(), note)
	require.ErrorIs(t, err, ErrInvalid)
	require.EqualError(t, err, "invalid note: no id")
}

func TestWriteThenParseRoundTrips(t *testing.T) {
	directory := t.TempDir()
	path, err := Write(directory, valid)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(directory, "personal", "20261015-a1b2c3.md"), path)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "---\ndate: 2026-10-15\nkind: decision\nowner: personal\nprofile: claude\nproject: claude-ephant\nscope: project\nsession: 010a2385\n---\nPin pydantic below 2.14 in the uvx wrapper.\n", string(content))
	parsed, err := Parse(path, content)
	require.NoError(t, err)
	require.Equal(t, valid, parsed)

	_, err = Write(directory, valid)
	require.ErrorIs(t, err, os.ErrExist)
}
