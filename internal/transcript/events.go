package transcript

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/redact"
)

const (
	EventAgent      = "agent"
	EventCommand    = "command"
	EventCompaction = "compaction"
	EventHookBlock  = "hook-block"
	EventInjection  = "injection"
	EventReply      = "reply"
	EventSkill      = "skill"
	EventTokens     = "tokens"
	EventToolError  = "tool-error"
)

// Hook events whose plain stdout reaches the model; for the rest only JSON additionalContext does. As of 2026-10-16
// no transcript records UserPromptSubmit plain stdout, only additional context.
var plainContextHooks = []string{"SessionStart", "UserPromptSubmit"}

var (
	commandName  = regexp.MustCompile(`<command-name>/?([^<\s]+)</command-name>`)
	hookFeedback = regexp.MustCompile(`^([A-Za-z]+) hook feedback:`)
	reviewerFile = regexp.MustCompile(`reviewers/([a-z-]+)\.md`)
)

// Reviewers names the checkpoint reviewer files an agent prompt sends its agent to, sorted, each once.
func Reviewers(prompt string) []string {
	var reviewers []string
	for _, match := range reviewerFile.FindAllStringSubmatch(prompt, -1) {
		reviewers = append(reviewers, match[1])
	}
	slices.Sort(reviewers)
	return slices.Compact(reviewers)
}

func agentAttributes(input map[string]any) map[string]any {
	attributes := map[string]any{}
	if description, _ := input["description"].(string); description != "" {
		attributes["description"] = truncate(description)
	}
	if background, _ := input["run_in_background"].(bool); background {
		attributes["background"] = true
	}
	prompt, _ := input["prompt"].(string)
	if reviewers := Reviewers(prompt); len(reviewers) > 0 {
		attributes["reviewers"] = reviewers
	}
	return attributes
}

func agentType(input map[string]any) string {
	if name, _ := input["subagent_type"].(string); name != "" {
		return name
	}
	return "general-purpose"
}

func attachmentEvents(parsed line) []dto.TranscriptEvent {
	attachment := parsed.Attachment
	var name string
	switch {
	case attachment.Type == "hook_additional_context":
		name = attachment.HookName
	case attachment.Type == "hook_success" && slices.Contains(plainContextHooks, attachment.HookEvent):
		name = attachment.HookName
	case attachment.Type == "skill_listing":
		name = "skills"
	default:
		return nil
	}
	bytes := contentBytes(attachment.Content)
	if bytes == 0 {
		return nil
	}
	return []dto.TranscriptEvent{{
		Attributes:       map[string]any{"bytes": bytes},
		Entrypoint:       parsed.Entrypoint,
		Kind:             EventInjection,
		Name:             name,
		SessionID:        parsed.SessionID,
		Timestamp:        parsed.Timestamp,
		UUID:             parsed.UUID,
		WorkingDirectory: parsed.WorkingDirectory,
	}}
}

func contentBytes(content json.RawMessage) int {
	var text string
	if json.Unmarshal(content, &text) == nil {
		return len(text)
	}
	var parts []string
	if json.Unmarshal(content, &parts) != nil {
		return 0
	}
	total := 0
	for _, part := range parts {
		total += len(part)
	}
	return total
}

func firstLineAfter(text string) string {
	_, rest, _ := strings.Cut(text, "\n")
	for candidate := range strings.SplitSeq(rest, "\n") {
		if strings.TrimSpace(candidate) != "" {
			return truncate(candidate)
		}
	}
	return ""
}

func lineEvents(parsed line, text string, blocks []block) []dto.TranscriptEvent {
	base := dto.TranscriptEvent{Entrypoint: parsed.Entrypoint, SessionID: parsed.SessionID, Timestamp: parsed.Timestamp, UUID: parsed.UUID, WorkingDirectory: parsed.WorkingDirectory}
	event := func(kind string, name string, attributes map[string]any) dto.TranscriptEvent {
		base.Attributes, base.Kind, base.Name = attributes, kind, name
		return base
	}
	if parsed.IsCompactSummary {
		return []dto.TranscriptEvent{event(EventCompaction, "", nil)}
	}
	var events []dto.TranscriptEvent
	if parsed.Type == "user" {
		for _, said := range userTexts(text, blocks) {
			if match := hookFeedback.FindStringSubmatch(said); match != nil {
				events = append(events, event(EventHookBlock, match[1], map[string]any{"reason": firstLineAfter(said)}))
			}
			if match := commandName.FindStringSubmatch(said); match != nil {
				events = append(events, event(EventCommand, match[1], nil))
			}
		}
		for _, block := range blocks {
			if block.Type == "tool_result" && block.IsError {
				events = append(events, event(EventToolError, "", toolErrorAttributes(parsed, block)))
			}
		}
		return events
	}
	if tokens, ok := tokenEvent(parsed, base); ok {
		events = append(events, tokens)
	}
	for _, block := range blocks {
		// The only filter keeping subagent prose out of replies; older transcripts hold it in the parent's file too.
		if block.Type == "text" && !parsed.IsSidechain && strings.TrimSpace(block.Text) != "" {
			events = append(events, event(EventReply, "", map[string]any{"characters": utf8.RuneCountInString(block.Text)}))
		}
		if block.Type != "tool_use" {
			continue
		}
		switch block.Name {
		case "Skill":
			if skill, _ := block.Input["skill"].(string); skill != "" {
				events = append(events, event(EventSkill, skill, nil))
			}
		case "Agent", "Task":
			events = append(events, event(EventAgent, agentType(block.Input), agentAttributes(block.Input)))
		}
	}
	return events
}

func resultText(content json.RawMessage) string {
	var text string
	if json.Unmarshal(content, &text) == nil {
		return text
	}
	var parts []block
	if json.Unmarshal(content, &parts) != nil {
		return ""
	}
	var texts []string
	for _, part := range parts {
		if part.Type == "text" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// A response is written one line per content block, each with its usage, so the message id keys it.
func tokenEvent(parsed line, base dto.TranscriptEvent) (dto.TranscriptEvent, bool) {
	message := parsed.Message
	if message.ID == "" || message.Usage == nil {
		return dto.TranscriptEvent{}, false
	}
	counts := message.Usage
	if counts.CacheCreationInputTokens+counts.CacheReadInputTokens+counts.InputTokens+counts.OutputTokens == 0 {
		return dto.TranscriptEvent{}, false
	}
	base.Attributes = map[string]any{
		"cacheRead":  counts.CacheReadInputTokens,
		"cacheWrite": counts.CacheCreationInputTokens,
		"input":      counts.InputTokens,
		"output":     counts.OutputTokens,
	}
	base.Key = "message:" + message.ID
	base.Kind = EventTokens
	base.Name = message.Model
	return base, true
}

// callLine lets sync name the call: a result line carries no tool name or input. A chain's failing command usually
// reports last, hence the tail.
func toolErrorAttributes(parsed line, result block) map[string]any {
	// Before either cut: a cut token no longer matches a pattern.
	text := redact.Redact(resultText(result.Content))
	attributes := map[string]any{"error": truncate(text)}
	runes := []rune(strings.Join(strings.Fields(text), " "))
	switch rest := runes[min(inputLimit, len(runes)):]; {
	case len(rest) > inputLimit:
		attributes["errorTail"] = "…" + string(rest[len(rest)-inputLimit:])
	case len(rest) > 0:
		attributes["errorTail"] = string(rest)
	}
	if parsed.SourceToolAssistantUUID != "" {
		attributes["callLine"] = parsed.SourceToolAssistantUUID
	}
	return attributes
}

func userTexts(text string, blocks []block) []string {
	if blocks == nil {
		return []string{text}
	}
	var texts []string
	for _, block := range blocks {
		if block.Type == "text" {
			texts = append(texts, block.Text)
		}
	}
	return texts
}
