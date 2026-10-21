package transcript

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

func TestParseEvents(t *testing.T) {
	base := `"cwd":"/w/repo","entrypoint":"cli","sessionId":"s1","timestamp":"2026-10-15T10:00:00Z","uuid":"u1"`
	event := func(kind string, name string, attributes map[string]any) dto.TranscriptEvent {
		return dto.TranscriptEvent{Attributes: attributes, Entrypoint: "cli", Kind: kind, Name: name, SessionID: "s1", Timestamp: "2026-10-15T10:00:00Z", UUID: "u1", WorkingDirectory: "/w/repo"}
	}

	tests := []struct {
		entries int
		events  []dto.TranscriptEvent
		line    string
		name    string
	}{
		{
			entries: 2,
			events: []dto.TranscriptEvent{event(EventReply, "", map[string]any{"characters": 10}), event(EventAgent, "general-purpose", map[string]any{
				"background":  true,
				"description": "Phase 1 review: bugs",
				"reviewers":   []string{"bugs", "prose"},
			})},
			line: `{"type":"assistant",` + base + `,"message":{"content":[{"type":"text","text":"Launching."},{"type":"tool_use","name":"Agent","input":{"description":"Phase 1 review: bugs","prompt":"Read /x/reviewers/prose.md, then /x/reviewers/bugs.md and /x/reviewers/bugs.md again.","run_in_background":true,"subagent_type":"general-purpose"}}]}}`,
			name: "agent launch naming reviewer files",
		},
		{
			entries: 0,
			events:  []dto.TranscriptEvent{event(EventCompaction, "", nil)},
			line:    `{"type":"user","isCompactSummary":true,` + base + `,"message":{"content":"This session is being continued"}}`,
			name:    "compaction summary",
		},
		{
			entries: 0,
			events:  []dto.TranscriptEvent{event(EventInjection, "PostToolUse:Edit", map[string]any{"bytes": 7})},
			line:    `{"type":"attachment",` + base + `,"attachment":{"type":"hook_additional_context","hookEvent":"PostToolUse","hookName":"PostToolUse:Edit","content":["abc","defg"]}}`,
			name:    "hook additional context",
		},
		{
			entries: 0,
			events:  nil,
			line:    `{"type":"attachment",` + base + `,"attachment":{"type":"hook_additional_context","hookEvent":"PostToolUse","hookName":"PostToolUse:Edit","content":[""]}}`,
			name:    "hook additional context with nothing in it",
		},
		{
			entries: 0,
			events:  []dto.TranscriptEvent{event(EventHookBlock, "PreToolUse", map[string]any{"reason": "Blocked: cd chained with an edit."})},
			line:    `{"type":"user","isMeta":true,` + base + `,"message":{"content":[{"type":"text","text":"PreToolUse hook feedback:\nBlocked: cd chained with an edit."}]}}`,
			name:    "hook block in a text block",
		},
		{
			entries: 1,
			events:  []dto.TranscriptEvent{event(EventAgent, "general-purpose", map[string]any{})},
			line:    `{"type":"assistant",` + base + `,"message":{"content":[{"type":"tool_use","name":"Task","input":{"prompt":"look"}}]}}`,
			name:    "older Task tool with no type or description",
		},
		{
			entries: 1,
			events:  nil,
			line:    `{"type":"assistant",` + base + `,"message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}`,
			name:    "ordinary tool call",
		},
		{
			entries: 0,
			events:  nil,
			line:    `{"type":"attachment",` + base + `,"attachment":{"type":"hook_success","hookEvent":"PostToolUse","hookName":"PostToolUse:Edit","content":"formatted","stdout":"formatted"}}`,
			name:    "plain stdout of a hook the model does not see",
		},
		{
			entries: 1,
			events:  []dto.TranscriptEvent{event(EventReply, "", map[string]any{"characters": 5})},
			line:    `{"type":"assistant",` + base + `,"message":{"content":[{"type":"text","text":"héllo"},{"type":"text","text":"  "}]}}`,
			name:    "reply counted in characters, blank text skipped",
		},
		{
			entries: 0,
			events:  []dto.TranscriptEvent{event(EventInjection, "SessionStart:startup", map[string]any{"bytes": 5})},
			line:    `{"type":"attachment",` + base + `,"attachment":{"type":"hook_success","hookEvent":"SessionStart","hookName":"SessionStart:startup","content":"notes","stdout":"notes"}}`,
			name:    "session start plain stdout",
		},
		{
			entries: 0,
			events:  []dto.TranscriptEvent{event(EventInjection, "skills", map[string]any{"bytes": 9})},
			line:    `{"type":"attachment",` + base + `,"attachment":{"type":"skill_listing","content":"- a: does"}}`,
			name:    "skill listing",
		},
		{
			entries: 1,
			events:  []dto.TranscriptEvent{event(EventSkill, "checkpoint", nil)},
			line:    `{"type":"assistant",` + base + `,"message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}}`,
			name:    "skill tool call",
		},
		{
			entries: 0,
			events:  []dto.TranscriptEvent{event(EventHookBlock, "Stop", map[string]any{"reason": "Files changed this turn, and no checkpoint was reported."})},
			line:    `{"type":"user","isMeta":true,` + base + `,"message":{"content":"Stop hook feedback:\n\nFiles changed this turn, and no checkpoint was reported.\n\nChanged: a.go"}}`,
			name:    "stop hook block",
		},
		{
			entries: 1,
			events:  nil,
			line:    `{"type":"assistant","isSidechain":true,` + base + `,"message":{"content":[{"type":"text","text":"subagent prose"}]}}`,
			name:    "subagent prose in its parent's file is no reply",
		},
		{
			entries: 1,
			events:  nil,
			line:    `{"type":"assistant",` + base + `,"message":{"id":"msg_2","model":"<synthetic>","usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0},"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`,
			name:    "synthetic message with no tokens",
		},
		{
			entries: 0,
			events:  nil,
			line:    `{"type":"attachment",` + base + `,"attachment":{"type":"hook_system_message","hookEvent":"SessionStart","hookName":"SessionStart:startup","content":"shown to the user"}}`,
			name:    "system message shown only to the user",
		},
		{
			entries: 1,
			events: []dto.TranscriptEvent{func() dto.TranscriptEvent {
				tokens := event(EventTokens, "claude-opus-5-5", map[string]any{"cacheRead": 27924, "cacheWrite": 22290, "input": 2, "output": 504})
				tokens.Key = "message:msg_1"
				return tokens
			}()},
			line: `{"type":"assistant",` + base + `,"message":{"id":"msg_1","model":"claude-opus-5-5","usage":{"input_tokens":2,"cache_creation_input_tokens":22290,"cache_read_input_tokens":27924,"output_tokens":504,"cache_creation":{"ephemeral_1h_input_tokens":22290}},"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`,
			name: "token usage keyed by message",
		},
		{
			entries: 0,
			events:  []dto.TranscriptEvent{event(EventToolError, "", map[string]any{"callLine": "a1", "error": "Exit code 1 freeze: a.go has no change"})},
			line:    `{"type":"user",` + base + `,"sourceToolAssistantUUID":"a1","message":{"content":[{"type":"tool_result","tool_use_id":"t1","is_error":true,"content":[{"type":"text","text":"Exit code 1\nfreeze: a.go has no change"}]}]}}`,
			name:    "tool error in a text array",
		},
		{
			entries: 0,
			events:  []dto.TranscriptEvent{event(EventToolError, "", map[string]any{"callLine": "a1", "error": "<tool_use_error>File has not been read yet.</tool_use_error>"})},
			line:    `{"type":"user",` + base + `,"sourceToolAssistantUUID":"a1","message":{"content":[{"type":"tool_result","tool_use_id":"t1","is_error":true,"content":"<tool_use_error>File has not been read yet.</tool_use_error>"}]}}`,
			name:    "tool error in plain content",
		},
		{
			entries: 0,
			events:  nil,
			line:    `{"type":"user",` + base + `,"sourceToolAssistantUUID":"a1","message":{"content":[{"type":"tool_result","tool_use_id":"t1","is_error":false,"content":"ok"}]}}`,
			name:    "tool result without an error",
		},
		{
			entries: 1,
			events:  []dto.TranscriptEvent{event(EventCommand, "checkpoint", nil)},
			line:    `{"type":"user",` + base + `,"message":{"content":"<command-message>checkpoint</command-message>\n<command-name>/checkpoint</command-name>"}}`,
			name:    "typed slash command",
		},
		{
			entries: 1,
			events:  nil,
			line:    `{"type":"user",` + base + `,"message":{"content":"I keep seeing Stop hook feedback: in my sessions"}}`,
			name:    "user prose that only mentions hook feedback",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries, events, err := Parse([]byte(test.line))
			require.NoError(t, err)
			require.Len(t, entries, test.entries)
			require.Equal(t, test.events, events)
		})
	}
}

func TestParseEventsTruncatesALongReason(t *testing.T) {
	_, events, err := Parse([]byte(`{"type":"user","isMeta":true,"message":{"content":"Stop hook feedback:\n` + strings.Repeat("x", 400) + `"}}`))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, strings.Repeat("x", inputLimit)+"…", events[0].Attributes["reason"])
}

func TestParseEventsTruncatesALongToolError(t *testing.T) {
	_, events, err := Parse([]byte(`{"type":"user","message":{"content":[{"type":"tool_result","is_error":true,"content":"` + strings.Repeat("x", 400) + `"}]}}`))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, strings.Repeat("x", inputLimit)+"…", events[0].Attributes["error"])
}
