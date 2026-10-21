package transcript

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

const (
	KindConversation = "conversation"
	KindPrompt       = "prompt"
	KindTool         = "tool"
)

const inputLimit = 300

// Marks harness text on user lines that carry no origin.
var harnessTags = []string{"<bash-stderr>", "<bash-stdout>", "<fork-boilerplate>", "<local-command-caveat>", "<local-command-stderr>", "<local-command-stdout>", "<system-reminder>", "<task-notification>", "[Request interrupted"}

// Priority order.
var primaryInputs = []string{"command", "file_path", "path", "pattern", "query", "url", "skill", "description", "prompt"}

type Entry struct {
	Entrypoint       string
	Kind             string
	SessionID        string
	Text             string
	Timestamp        string
	UUID             string
	WorkingDirectory string
}

type block struct {
	Content json.RawMessage `json:"content"`
	Input   map[string]any  `json:"input"`
	IsError bool            `json:"is_error"`
	Name    string          `json:"name"`
	Text    string          `json:"text"`
	Type    string          `json:"type"`
}

type usage struct {
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

type line struct {
	Attachment *struct {
		Content   json.RawMessage `json:"content"`
		HookEvent string          `json:"hookEvent"`
		HookName  string          `json:"hookName"`
		Type      string          `json:"type"`
	} `json:"attachment"`
	Entrypoint       string `json:"entrypoint"`
	IsCompactSummary bool   `json:"isCompactSummary"`
	IsMeta           bool   `json:"isMeta"`
	IsSidechain      bool   `json:"isSidechain"`
	Message          *struct {
		Content json.RawMessage `json:"content"`
		ID      string          `json:"id"`
		Model   string          `json:"model"`
		Usage   *usage          `json:"usage"`
	} `json:"message"`
	Origin *struct {
		Kind string `json:"kind"`
	} `json:"origin"`
	SessionID               string `json:"sessionId"`
	SourceToolAssistantUUID string `json:"sourceToolAssistantUUID"`
	Timestamp               string `json:"timestamp"`
	Type                    string `json:"type"`
	UUID                    string `json:"uuid"`
	WorkingDirectory        string `json:"cwd"`
}

// ContextTokens is what a main-thread response read and wrote: the context the next turn starts from.
func ContextTokens(raw []byte) (int, bool) {
	var parsed line
	if json.Unmarshal(raw, &parsed) != nil || parsed.Type != "assistant" || parsed.IsSidechain || parsed.Message == nil || parsed.Message.Usage == nil {
		return 0, false
	}
	counts := parsed.Message.Usage
	total := counts.CacheCreationInputTokens + counts.CacheReadInputTokens + counts.InputTokens + counts.OutputTokens
	return total, total > 0
}

func Parse(raw []byte) ([]Entry, []dto.TranscriptEvent, error) {
	var parsed line
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, nil, fmt.Errorf("parse transcript line: %w", err)
	}
	if parsed.Type == "attachment" && parsed.Attachment != nil {
		return nil, attachmentEvents(parsed), nil
	}
	if parsed.Message == nil || (parsed.Type != "user" && parsed.Type != "assistant") {
		return nil, nil, nil
	}
	var text string
	var blocks []block
	if err := json.Unmarshal(parsed.Message.Content, &text); err != nil {
		if err = json.Unmarshal(parsed.Message.Content, &blocks); err != nil {
			if parsed.IsMeta || parsed.IsCompactSummary {
				return nil, nil, nil
			}
			return nil, nil, fmt.Errorf("parse message content: %w", err)
		}
	}
	events := lineEvents(parsed, text, blocks)
	if parsed.IsMeta || parsed.IsCompactSummary {
		return nil, events, nil
	}
	if parsed.Type == "user" && parsed.Origin != nil && parsed.Origin.Kind != "human" {
		return nil, events, nil
	}
	base := Entry{Entrypoint: parsed.Entrypoint, SessionID: parsed.SessionID, Timestamp: parsed.Timestamp, UUID: parsed.UUID, WorkingDirectory: parsed.WorkingDirectory}
	typed := func(text string) bool {
		return strings.TrimSpace(text) != "" && (parsed.Origin != nil || !harnessWritten(text))
	}
	// A subagent's user lines are its parent agent's brief, not the user's words.
	userKind := KindPrompt
	if parsed.IsSidechain && parsed.Origin == nil {
		userKind = KindConversation
	}

	if blocks == nil {
		if parsed.Type != "user" || !typed(text) {
			return nil, events, nil
		}
		return []Entry{with(base, userKind, text)}, events, nil
	}
	var entries []Entry
	for _, block := range blocks {
		switch {
		case block.Type == "text" && parsed.Type == "user" && typed(block.Text):
			entries = append(entries, with(base, userKind, block.Text))
		case block.Type == "text" && parsed.Type == "assistant" && strings.TrimSpace(block.Text) != "":
			entries = append(entries, with(base, KindConversation, block.Text))
		case block.Type == "tool_use" && parsed.Type == "assistant":
			entries = append(entries, with(base, KindTool, ToolLine(block.Name, block.Input)))
		}
	}
	return entries, events, nil
}

func ToolLine(name string, input map[string]any) string {
	return name + ": " + primaryInput(input)
}

func harnessWritten(text string) bool {
	trimmed := strings.TrimSpace(text)
	for _, tag := range harnessTags {
		if strings.HasPrefix(trimmed, tag) {
			return true
		}
	}
	return false
}

func primaryInput(input map[string]any) string {
	for _, key := range primaryInputs {
		if value, ok := input[key].(string); ok && value != "" {
			return truncate(value)
		}
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	return truncate(string(encoded))
}

func truncate(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if runes := []rune(value); len(runes) > inputLimit {
		return string(runes[:inputLimit]) + "…"
	}
	return value
}

func with(base Entry, kind string, text string) Entry {
	base.Kind = kind
	base.Text = text
	return base
}
