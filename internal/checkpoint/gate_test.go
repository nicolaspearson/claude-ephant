package checkpoint

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinalTextKeepsOnlyTheLastTextBlock(t *testing.T) {
	turn := "\x1eTURN go on\n\x1eTEXT\nfirst\n\x1eTOOL Bash ls\n\x1eTEXT\nsecond\nline\n\x1eTOOL Edit a.go"

	require.Equal(t, "second\nline", finalText(turn))
	require.Empty(t, finalText("\x1eTURN go on\n\x1eTOOL Bash ls"))
}

func TestGatePatternsMatchTheirWordings(t *testing.T) {
	tests := []struct {
		matches bool
		pattern *regexp.Regexp
		text    string
	}{
		{matches: false, pattern: commitCommand, text: "Committed."},
		{matches: true, pattern: commitCommand, text: "Run git commit -- a.go"},
		{matches: true, pattern: doneClaim, text: "All done."},
		{matches: false, pattern: doneClaim, text: "Done reading the file."},
		{matches: true, pattern: doneClaim, text: "It is now done."},
		{matches: true, pattern: doneClaim, text: "Ready for review."},
		{matches: true, pattern: doneClaim, text: "Ready to commit."},
		{matches: true, pattern: doneClaim, text: "Run git commit -- a.go"},
		{matches: true, pattern: doneClaim, text: "That's done."},
		{matches: true, pattern: doneClaim, text: "The feature is complete."},
		{matches: true, pattern: doneClaim, text: "The work is done."},
		{matches: true, pattern: laterCommit, text: "After review you can then commit."},
		{matches: true, pattern: laterCommit, text: "Once the gate passes, you can commit."},
		{matches: true, pattern: laterCommit, text: "When it passes, you can commit."},
		{matches: false, pattern: laterCommit, text: "You can commit now."},
		{matches: true, pattern: nothingLeft, text: "0 remain."},
		{matches: false, pattern: nothingLeft, text: "Left the change in the tree."},
		{matches: true, pattern: nothingLeft, text: "No one remains."},
		{matches: true, pattern: nothingLeft, text: "No steps remain."},
		{matches: true, pattern: nothingLeft, text: "None to go."},
		{matches: true, pattern: nothingLeft, text: "Nothing is left."},
		{matches: true, pattern: nothingLeft, text: "Zero left."},
		{matches: true, pattern: prosePass, text: "Comment: fine."},
		{matches: true, pattern: prosePass, text: "Comments sweep found none."},
		{matches: true, pattern: prosePass, text: "Prose audit found none."},
		{matches: true, pattern: prosePass, text: "Prose pass found none."},
		{matches: true, pattern: prosePass, text: "Prose review: clean."},
		{matches: true, pattern: prosePass, text: "Prose — clean."},
		{matches: true, pattern: prosePass, text: "Prose-clean."},
		{matches: false, pattern: prosePass, text: "The prose reads well."},
		{matches: true, pattern: qualityPass, text: "Bugs review found none."},
		{matches: false, pattern: qualityPass, text: "Quality is fine."},
		{matches: true, pattern: qualityPass, text: "Quality pass found none."},
		{matches: true, pattern: qualityPass, text: "Quality review: clean."},
		{matches: true, pattern: qualityPass, text: "Quality — clean."},
		{matches: true, pattern: qualityPass, text: "Quality-clean."},
		{matches: true, pattern: qualityPass, text: "Quality: clean."},
		{matches: true, pattern: skillInvoked, text: "\x1eTOOL Skill checkpoint"},
		{matches: true, pattern: skillInvoked, text: "\x1eTOOL Skill ele:checkpoint"},
		{matches: false, pattern: skillInvoked, text: "\x1eTOOL Skill ele:recall"},
		{matches: true, pattern: skillInvoked, text: "\x1eTURN /checkpoint"},
		{matches: true, pattern: skillInvoked, text: "\x1eTURN /ele:checkpoint"},
		{matches: true, pattern: skillInvoked, text: "<command-name>/ele:checkpoint</command-name>"},
		{matches: false, pattern: stillLeft, text: "Changes left in the working tree."},
		{matches: true, pattern: stillLeft, text: "Done; one remains."},
		{matches: true, pattern: stillLeft, text: "One step is still left."},
		{matches: true, pattern: stillLeft, text: "Still to come: group 4."},
		{matches: true, pattern: stillLeft, text: "Still to do: group 4."},
		{matches: true, pattern: stillLeft, text: "Two are left."},
		{matches: true, pattern: stillLeft, text: "Two left to do."},
		{matches: true, pattern: stillLeft, text: "Two more remain."},
		{matches: true, pattern: stillLeft, text: "Two more to go."},
	}
	for _, test := range tests {
		require.Equal(t, test.matches, test.pattern.MatchString(test.text), "%s on %q", test.pattern, test.text)
	}
}

func TestLastTurnStartsAtAPromptWithAnImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s1.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"type":"user","message":{"content":"one"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"first"}]}}
{"type":"user","message":{"content":[{"type":"text","text":"look"},{"type":"image","source":{"type":"base64","data":""}}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"second"}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"ok"}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"answered"},{"type":"image","source":{"type":"base64","data":""}}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"third"}]}}
{"type":"user","message":{"content":[{"type":"text","text":"[Request interrupted by user for tool use]"}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"fourth"}]}}
`), 0o600))

	require.Equal(t, "\x1eTURN look\n\x1eTEXT\nsecond\n\x1eTEXT\nthird\n\x1eTEXT\nfourth", lastTurn(path), "neither a tool result nor an interrupt starts a turn")
}

func TestLastTurnStartsAtTheLastPrompt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s1.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"type":"user","message":{"content":"one"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"first"}]}}
{"type":"user","message":{"content":"two"}}
{"type":"user","isMeta":true,"message":{"content":"Stop hook feedback"}}
{"type":"user","isMeta":"yes","message":{"content":"half decoded"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"second"}]}}
`), 0o600))

	require.Equal(t, "\x1eTURN two\n\x1eTEXT\nsecond", lastTurn(path))
	require.Empty(t, lastTurn(""))
	require.Empty(t, lastTurn(filepath.Join(t.TempDir(), "missing.jsonl")))
}

func TestLastTurnWithItsPromptPastTheTailStartsWithoutOne(t *testing.T) {
	lines := []string{`{"type":"user","message":{"content":"cut off"}}`}
	for range transcriptTail {
		lines = append(lines, `{"type":"assistant","message":{"content":[{"type":"text","text":"working"}]}}`)
	}
	path := filepath.Join(t.TempDir(), "s1.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600))

	turn := lastTurn(path)
	require.NotContains(t, turn, "cut off")
	require.True(t, strings.HasPrefix(turn, "\x1eTEXT\nworking"))
}

func TestRenderAssistantKeepsAMessageWhoseToolInputIsNotAString(t *testing.T) {
	rendered := renderAssistant([]byte(`[{"type":"text","text":"Ready to commit."},{"type":"tool_use","name":"mcp__runner__run","input":{"command":["make","test"]}}]`))

	require.Equal(t, "\x1eTEXT\nReady to commit.\n\x1eTOOL mcp__runner__run ", rendered)
}

func TestRepositoryPathNamesAFileThroughASymlinkedCheckoutAsGitDoes(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal"), 0o700))
	link := filepath.Join(t.TempDir(), "checkout")
	require.NoError(t, os.Symlink(root, link))

	require.Equal(t, "internal/new.go", repositoryPath(root, filepath.Join(link, "internal", "new.go")))
	require.Equal(t, "internal/new.go", repositoryPath(root, filepath.Join(root, "internal", "new.go")))
	require.Equal(t, "/elsewhere/new.go", repositoryPath(root, "/elsewhere/new.go"))
}

func TestSpokenTextJoinsEveryTextBlock(t *testing.T) {
	turn := "\x1eTURN go on\n\x1eTEXT\nfirst\n\x1eTOOL Bash ls\n\x1eTEXT\nsecond\nline\n\x1eTOOL Edit a.go"

	require.Equal(t, "first\nsecond\nline", spokenText(turn))
}
