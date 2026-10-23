package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

func TestUsageCountsEachMessagesTokensOnce(t *testing.T) {
	root := t.TempDir()
	gateway := filepath.Join(root, "w", "gateway")
	require.NoError(t, os.MkdirAll(filepath.Join(gateway, ".git"), 0o700))
	projects := filepath.Join(root, "projects")
	line := func(uuid string, body string) string {
		return `{"cwd":"` + gateway + `","sessionId":"s1","timestamp":"2026-10-15T10:00:00Z","uuid":"` + uuid + `",` + body + "}\n"
	}
	assistant := func(uuid string, message string, input int, cacheWrite int, cacheRead int, output int) string {
		return line(uuid, fmt.Sprintf(`"type":"assistant","message":{"id":%q,"model":"claude-opus-5-5","usage":{"input_tokens":%d,"cache_creation_input_tokens":%d,"cache_read_input_tokens":%d,"output_tokens":%d},"content":[{"type":"text","text":"x"}]}`, message, input, cacheWrite, cacheRead, output))
	}
	content := line("a", `"type":"user","message":{"content":"fix the build"}`) +
		assistant("b", "m1", 3, 100, 1000, 50) +
		assistant("c", "m1", 3, 100, 1000, 50) +
		assistant("d", "m2", 1, 10, 1100, 7)
	write(t, filepath.Join(projects, "-gateway", "s1.jsonl"), content)
	// The same lines archived a second time, as a transcript rewritten in place would be.
	write(t, filepath.Join(projects, "-gateway-copy", "s1.jsonl"), content)
	subagent := assistant("e", "m3", 2, 500, 0, 5) + assistant("f", "m3", 2, 500, 0, 20)
	write(t, filepath.Join(projects, "-gateway", "s1", "subagents", "agent-1.jsonl"), subagent)
	write(t, filepath.Join(projects, "-gateway-copy", "s1", "subagents", "agent-1.jsonl"), subagent)
	// A fork opens with its parent's message, and its folder is walked before the parent's file.
	write(t, filepath.Join(projects, "-gateway", "s1", "subagents", "agent-2.jsonl"), `{"type":"fork-context-ref"}`+"\n"+assistant("g", "m1", 3, 100, 1000, 60))
	index := open(t, filepath.Join(root, "ele.db"))
	update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{Owners: []string{"corp"}, Projects: map[string]string{"gateway": "corp"}})

	usage, err := index.Usage(context.Background(), dto.UsageQuery{Owner: "corp"})
	require.NoError(t, err)
	require.Equal(t, dto.UsageTokens{CacheRead: 2100, CacheWrite: 110, Input: 4, Output: 67}, usage.MainTokens, "m1 split over two lines, the copy and the fork's, count once, at the fork's larger output")
	require.Equal(t, dto.UsageTokens{CacheWrite: 500, Input: 2, Output: 20}, usage.SubagentTokens, "a streamed message's last, largest output count")

	usage, err = index.Usage(context.Background(), dto.UsageQuery{Since: "2026-10-16"})
	require.NoError(t, err)
	require.Equal(t, dto.Usage{}, usage)
}

func TestUsageCountsPromptsToolsInjectionsAndReplies(t *testing.T) {
	root := t.TempDir()
	gateway := filepath.Join(root, "w", "gateway")
	notes := filepath.Join(root, "w", "notes")
	for _, repository := range []string{gateway, notes} {
		require.NoError(t, os.MkdirAll(filepath.Join(repository, ".git"), 0o700))
	}
	projects := filepath.Join(root, "projects")
	line := func(session string, directory string, day string, uuid string, body string) string {
		return `{"cwd":"` + directory + `","sessionId":"` + session + `","timestamp":"2026-10-` + day + `T10:00:00Z","uuid":"` + uuid + `",` + body + "}\n"
	}
	write(t, filepath.Join(projects, "-gateway", "s1.jsonl"),
		line("s1", gateway, "15", "a", `"type":"user","message":{"content":"fix the build"}`)+
			line("s1", gateway, "15", "b", `"type":"assistant","message":{"content":[{"type":"text","text":"On it."},{"type":"tool_use","name":"Bash","input":{"command":"go build"}}]}`)+
			line("s1", gateway, "15", "c", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/x"}},{"type":"tool_use","name":"Bash","input":{"command":"go test"}}]}`)+
			line("s1", gateway, "15", "d", `"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"PostToolUse:Edit","content":["0123456789"]}`)+
			line("s1", gateway, "15", "e", `"type":"attachment","attachment":{"type":"skill_listing","content":"0123456789012345678901234567890123456789"}`)+
			line("s1", gateway, "15", "f", `"type":"user","message":{"content":"and the lint"}`)+
			line("s1", gateway, "15", "g", `"type":"assistant","message":{"content":[{"type":"text","text":"Done now."}]}`))
	write(t, filepath.Join(projects, "-gateway", "s1", "subagents", "agent-1.jsonl"),
		line("s1", gateway, "15", "h", `"type":"user","isSidechain":true,"message":{"content":"review this"}`)+
			line("s1", gateway, "15", "i", `"type":"assistant","isSidechain":true,"message":{"content":[{"type":"text","text":"Reviewed."},{"type":"tool_use","name":"Bash","input":{"command":"git diff"}}]}`)+
			line("s1", gateway, "15", "j", `"type":"attachment","isSidechain":true,"attachment":{"type":"hook_additional_context","hookName":"PostToolUse:Edit","content":["01234"]}`))
	write(t, filepath.Join(projects, "-notes", "s2.jsonl"),
		line("s2", notes, "16", "k", `"type":"user","message":{"content":"read up"}`)+
			line("s2", notes, "16", "l", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"WebSearch","input":{"query":"x"}}]}`))
	index := open(t, filepath.Join(root, "ele.db"))
	ownerMap := ownermap.OwnerMap{Owners: []string{"corp", "personal"}, Projects: map[string]string{"gateway": "corp", "notes": "personal"}}
	update(t, index, projects, filepath.Join(root, "archive"), ownerMap)

	usage, err := index.Usage(context.Background(), dto.UsageQuery{Owner: "corp"})
	require.NoError(t, err)
	require.Equal(t, dto.Usage{
		Injections:      []dto.UsageCount{{Main: 40, Name: "skills"}, {Main: 10, Name: "PostToolUse:Edit", Subagent: 5}},
		Prompts:         2,
		Replies:         2,
		ReplyCharacters: 15,
		Sessions:        1,
		Tools:           []dto.UsageCount{{Main: 2, Name: "Bash", Subagent: 1}, {Main: 1, Name: "Read"}},
	}, usage, "a subagent's brief is no prompt and its prose no reply")

	usage, err = index.Usage(context.Background(), dto.UsageQuery{})
	require.NoError(t, err)
	require.Equal(t, 2, usage.Sessions)
	require.Equal(t, []dto.UsageCount{{Main: 2, Name: "Bash", Subagent: 1}, {Main: 1, Name: "Read"}, {Main: 1, Name: "WebSearch"}}, usage.Tools, "ties broken by name")

	usage, err = index.Usage(context.Background(), dto.UsageQuery{Project: "notes", Since: "2026-10-16"})
	require.NoError(t, err)
	require.Equal(t, dto.Usage{Prompts: 1, Sessions: 1, Tools: []dto.UsageCount{{Main: 1, Name: "WebSearch"}}}, usage)
	usage, err = index.Usage(context.Background(), dto.UsageQuery{Since: "2026-10-17"})
	require.NoError(t, err)
	require.Equal(t, dto.Usage{}, usage)
}
