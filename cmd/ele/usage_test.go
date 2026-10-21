package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunUsageReportsPerOwner(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	line := func(uuid string, body string) string {
		return `{"cwd":"` + gateway + `","sessionId":"abc3","timestamp":"2026-10-15T11:00:00Z","uuid":"` + uuid + `",` + body + "}\n"
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "abc3.jsonl"),
		line("a", `"type":"user","message":{"content":"run the tests"}`)+
			line("b", `"type":"assistant","message":{"id":"m1","usage":{"input_tokens":10,"cache_creation_input_tokens":2000,"cache_read_input_tokens":30000,"output_tokens":400},"content":[{"type":"tool_use","name":"Bash","input":{"command":"go test"}},{"type":"tool_use","name":"mcp__linear__get_issue","input":{"id":"X-1"}}]}`)+
			line("c", `"type":"attachment","attachment":{"type":"skill_listing","content":"`+strings.Repeat("s", 8000)+`"}`)+
			line("d", `"type":"assistant","message":{"content":[{"type":"text","text":"All pass."}]}`)+
			line("f", `"type":"user","message":{"content":"and lint"}`)+
			line("g", `"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"PostToolUse:Write","content":["ok"]}`))
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "abc3", "subagents", "agent-1.jsonl"),
		line("e", `"type":"attachment","isSidechain":true,"attachment":{"type":"skill_listing","content":"`+strings.Repeat("s", 4000)+`"}`)+
			line("h", `"type":"assistant","isSidechain":true,"message":{"id":"m2","usage":{"input_tokens":4,"cache_creation_input_tokens":1000,"cache_read_input_tokens":0,"output_tokens":100},"content":[{"type":"text","text":"Reviewed."}]}`))

	output, code := runCommand(t, "usage", "--owner", "corp")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "2 sessions, 3 prompts for corp.\n")
	require.Regexp(t, `\nBash +1 +0\n`, output)
	require.Regexp(t, `\nmcp__linear__get_issue +1 +0\n`, output, "MCP tools by their full name")
	require.Regexp(t, `\nskills +7\.8 KiB +3\.9 KiB +1500\n`, output, "12000 bytes, main and subagents, over 2 sessions is about 1500 tokens each")
	require.Regexp(t, `\nPostToolUse:Write +2 B +0 B +<1\n`, output, "a cost too small to round to a token is not shown as none")
	require.Regexp(t, `\ninput, uncached +10 +4 +7\n`, output)
	require.Regexp(t, `\ncache write +2,000 +1,000 +1,500\n`, output)
	require.Regexp(t, `\ncache read +30,000 +0 +15,000\n`, output)
	require.Regexp(t, `\noutput +400 +100 +250\n`, output)
	require.Contains(t, output, "\nInput read from cache: main 93.7%, subagents 0.0%.\n")
	require.Contains(t, output, "\nReplies: 34 characters in 2 text blocks, 11 per prompt.\n")

	output, code = runCommand(t, "usage", "--owner", "personal", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Equal(t, "1 session, 1 prompt for personal.\n\nReplies: 0 characters in 0 text blocks, 0 per prompt.\n", output, "no table without rows")

	output, code = runCommand(t, "usage", "--project", "gateway", "--since", "2026-10-16", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Equal(t, "0 sessions, 0 prompts in gateway since 2026-10-16.\n", output)
}

func TestRunUsageRefusesBadInput(t *testing.T) {
	recallFixture(t)
	for _, test := range []struct {
		args    []string
		message string
	}{
		{args: []string{"usage", "--owner", "nobody"}, message: `ele: unknown owner "nobody"; owner-map.json lists corp, personal`},
		{args: []string{"usage", "--since", "soon"}, message: "ele: "},
		{args: []string{"usage", "extra"}, message: "ele usage [--owner"},
	} {
		output, code := runCommand(t, test.args...)
		require.Equal(t, 2, code, test.args)
		require.Contains(t, output, test.message, test.args)
	}
}
