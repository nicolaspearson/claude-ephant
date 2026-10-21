package transcript

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	base := `"cwd":"/w/repo","entrypoint":"cli","sessionId":"s1","timestamp":"2026-10-15T10:00:00Z","uuid":"u1"`
	entry := func(kind string, text string) Entry {
		return Entry{Entrypoint: "cli", Kind: kind, SessionID: "s1", Text: text, Timestamp: "2026-10-15T10:00:00Z", UUID: "u1", WorkingDirectory: "/w/repo"}
	}

	tests := []struct {
		entries []Entry
		line    string
		name    string
	}{
		{
			entries: nil,
			line:    `{"type":"user","origin":{"kind":"task-notification"},` + base + `,"message":{"content":"agent finished"}}`,
			name:    "agent-written user line",
		},
		{
			entries: []Entry{entry(KindConversation, "Done."), entry(KindTool, "Bash: go test ./...")},
			line:    `{"type":"assistant",` + base + `,"message":{"content":[{"type":"thinking","thinking":"hidden"},{"type":"text","text":"Done."},{"type":"tool_use","name":"Bash","input":{"command":"go   test\n./...","description":"run"}}]}}`,
			name:    "assistant text and tool call, thinking skipped",
		},
		{
			entries: nil,
			line:    `{"type":"user","isCompactSummary":true,` + base + `,"message":{"content":"This session is being continued"}}`,
			name:    "compaction summary",
		},
		{
			entries: nil,
			line:    `{"type":"user","origin":{"kind":"coordinator"},` + base + `,"message":{"content":"next task"}}`,
			name:    "coordinator-written user line",
		},
		{
			entries: nil,
			line:    `{"type":"user",` + base + `,"message":{"content":[{"type":"text","text":"  <system-reminder>hook output</system-reminder>"}]}}`,
			name:    "harness tag without origin",
		},
		{
			entries: nil,
			line:    `{"type":"attachment",` + base + `}`,
			name:    "housekeeping line",
		},
		{
			entries: []Entry{entry(KindPrompt, "<task-notification> quoted by me")},
			line:    `{"type":"user","origin":{"kind":"human"},` + base + `,"message":{"content":"<task-notification> quoted by me"}}`,
			name:    "human origin kept whatever it starts with",
		},
		{
			entries: nil,
			line:    `{"type":"user","isMeta":true,` + base + `,"message":{"content":"caveat"}}`,
			name:    "meta user line",
		},
		{
			entries: nil,
			line:    `{"type":"user",` + base + `,"message":{"content":"[Request interrupted by user]"}}`,
			name:    "request interrupted marker",
		},
		{
			entries: []Entry{entry(KindPrompt, "<command-name>/checkpoint</command-name>")},
			line:    `{"type":"user",` + base + `,"message":{"content":"<command-name>/checkpoint</command-name>"}}`,
			name:    "slash command typed by the user",
		},
		{
			entries: []Entry{entry(KindConversation, "Review the diff at /tmp/x")},
			line:    `{"type":"user","isSidechain":true,` + base + `,"message":{"content":"Review the diff at /tmp/x"}}`,
			name:    "subagent brief from its parent",
		},
		{
			entries: []Entry{entry(KindTool, "Edit: /w/repo/a.go")},
			line:    `{"type":"assistant",` + base + `,"message":{"content":[{"type":"tool_use","name":"Edit","input":{"old_string":"x","file_path":"/w/repo/a.go"}}]}}`,
			name:    "tool call keyed by file path",
		},
		{
			entries: []Entry{entry(KindTool, `TaskStop: {"id":7}`)},
			line:    `{"type":"assistant",` + base + `,"message":{"content":[{"type":"tool_use","name":"TaskStop","input":{"id":7}}]}}`,
			name:    "tool call without a primary input",
		},
		{
			entries: []Entry{entry(KindPrompt, "fix it")},
			line:    `{"type":"user",` + base + `,"message":{"content":"fix it"}}`,
			name:    "typed prompt",
		},
		{
			entries: []Entry{entry(KindPrompt, "see this")},
			line:    `{"type":"user",` + base + `,"message":{"content":[{"type":"text","text":"see this"},{"type":"tool_result","content":"output"},{"type":"text","text":"  "}]}}`,
			name:    "user text blocks, tool results skipped",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries, _, err := Parse([]byte(test.line))
			require.NoError(t, err)
			require.Equal(t, test.entries, entries)
		})
	}
}

func TestParseRejectsMalformedLine(t *testing.T) {
	_, _, err := Parse([]byte(`{"type":`))
	require.ErrorContains(t, err, "parse transcript line")
}

func TestParseTruncatesLongToolInput(t *testing.T) {
	entries, _, err := Parse([]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"` + strings.Repeat("é", 400) + `"}}]}}`))
	require.NoError(t, err)
	require.Equal(t, "Bash: "+strings.Repeat("é", inputLimit)+"…", entries[0].Text)
}
