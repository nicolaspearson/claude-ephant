package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	blockedChanged = 20
	logTargetRunes = 120
	// Enough to hold a turn; the whole transcript is megabytes.
	transcriptTail = 400
)

var (
	// Only a turn claiming done is a checkpoint, and only its last text block counts: a done claim followed by more work is not one. As of 2026-10-07, over 7 days: 21 of 384 disputed blocks would still block, and 495 of the other 701.
	doneClaim = regexp.MustCompile(`(?i)git commit|ready (to|for) (commit|merge|review|ship)|(all|work is|that.s|now) done|is complete`)
	// A pass label, not a fixed sentence: demanding the literal "quality review" and "prose audit" blocked turns that had just run the checkpoint.
	prosePass    = regexp.MustCompile(`(?i)(prose|comments?)[[:space:]]*(audit|pass|review|sweep|—|-|:)`)
	qualityPass  = regexp.MustCompile(`(?i)quality[[:space:]]*(review|pass|—|-|:)|bugs[[:space:]]+review`)
	skillInvoked = regexp.MustCompile(`(?i)\x1eTOOL Skill (ele:)?checkpoint|command-name>/?(ele:)?checkpoint|^\x1eTURN */(ele:)?checkpoint`)
)

type GateInput struct {
	Root           string
	Session        string
	StopHookActive bool
	TranscriptPath string
}

// Gate returns the block reason for a turn that changed files and claims done without a checkpoint, or "" to let it end.
func Gate(ctx context.Context, input GateInput, stateFile string) string {
	// One interruption per turn: continuing from this hook always proceeds, so a refusal can never loop.
	if input.StopHookActive {
		return ""
	}
	// Only this turn's edits count. A missing baseline means this turn's start was never seen, and blocking on dirt that predates the session is the false positive this guard exists to avoid.
	baseline, err := ReadState(stateFile)
	if err != nil {
		return ""
	}
	now, err := WriteState(ctx, input.Root)
	if err != nil {
		return ""
	}
	moved := MovedPaths(baseline, now)
	if len(moved) == 0 || Reviewed(ctx, input.Root) || Open(ctx, input.Root, input.Session) || OptedOut(ctx, input.Root, input.Session, moved) {
		return ""
	}
	turn := lastTurn(input.TranscriptPath)
	if !anyLine(doneClaim, finalText(turn)) || anyLine(skillInvoked, turn) {
		return ""
	}
	// Both passes have to appear: naming one and skipping the other is the habit this exists to catch. What the prompt asked for does not count.
	said := spokenText(turn)
	if anyLine(qualityPass, said) && anyLine(prosePass, said) {
		return ""
	}
	if len(moved) > blockedChanged {
		moved = moved[:blockedChanged]
	}
	return "Files changed this turn, the reply says the work is done, and no checkpoint was reported.\n\nChanged: " + strings.Join(moved, " ") +
		"\n\nInvoke the `ele:checkpoint` skill.\n\nIf the work is not done — mid-task, waiting on the user, or an investigation — say it is not a checkpoint, what remains, and stop.\n\nIf the user said this work needs no checkpoint, exempt its paths for this session: `ele checkpoint optout " + shellQuote(input.Root) + " \"<glob>\" ...`"
}

// HookLog appends to the machine-wide hook log; its tab-separated fields and their order are shared with the user's other hooks.
func HookLog(decision string, rule string, target string, now time.Time) {
	stamp := now.UTC().Format("2006-01-02T15:04:05Z")
	file := os.Getenv("HOOK_LOG")
	if file == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		file = filepath.Join(home, ".claude", "logs", "hooks-"+stamp[:7]+".log")
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return
	}
	target = strings.NewReplacer("\t", " ", "\n", " ").Replace(target)
	if runes := []rune(target); len(runes) > logTargetRunes {
		target = string(runes[:logTargetRunes])
	}
	directory := os.Getenv("PWD")
	if directory == "" {
		directory, _ = os.Getwd()
	}
	class := "unclassified"
	if rule == "checkpoint" {
		class = "quality"
	}
	line := strings.Join([]string{stamp, class, decision, rule, target, directory[strings.LastIndex(directory, "/")+1:]}, "\t") + "\n"
	log, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	_, _ = log.WriteString(line)
	_ = log.Close()
}

func anyLine(pattern *regexp.Regexp, text string) bool {
	for line := range strings.SplitSeq(text, "\n") {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

func finalText(turn string) string {
	var block []string
	on := false
	for line := range strings.SplitSeq(turn, "\n") {
		switch {
		case line == "\x1eTEXT":
			on, block = true, nil
		case strings.HasPrefix(line, "\x1e"):
			on = false
		case on:
			block = append(block, line)
		}
	}
	return strings.Join(block, "\n")
}

func lastTurn(path string) string {
	if path == "" {
		return ""
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
	if len(lines) > transcriptTail {
		lines = lines[len(lines)-transcriptTail:]
	}
	var rendered []string
	for _, line := range lines {
		var entry struct {
			IsMeta  bool `json:"isMeta"`
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			Type string `json:"type"`
		}
		if json.Unmarshal([]byte(line), &entry) != nil {
			continue
		}
		switch entry.Type {
		case "user":
			var prompt string
			if !entry.IsMeta && json.Unmarshal(entry.Message.Content, &prompt) == nil {
				rendered = append(rendered, "\x1eTURN "+prompt)
			}
		case "assistant":
			rendered = append(rendered, renderAssistant(entry.Message.Content))
		}
	}
	text := strings.Join(rendered, "\n")
	turnStart := 0
	for at, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "\x1eTURN") {
			turnStart = at
		}
	}
	return strings.TrimRight(strings.Join(strings.Split(text, "\n")[turnStart:], "\n"), "\n")
}

func renderAssistant(content json.RawMessage) string {
	var items []struct {
		Input struct {
			Skill json.RawMessage `json:"skill"`
		} `json:"input"`
		Name string `json:"name"`
		Text string `json:"text"`
		Type string `json:"type"`
	}
	if json.Unmarshal(content, &items) != nil {
		return ""
	}
	var parts []string
	for _, item := range items {
		switch item.Type {
		case "text":
			parts = append(parts, "\x1eTEXT\n"+item.Text)
		case "tool_use":
			parts = append(parts, "\x1eTOOL "+item.Name+" "+skillText(item.Input.Skill))
		}
	}
	return strings.Join(parts, "\n")
}

func shellQuote(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}

// skillText is "" for an absent, null or false skill, a string as is, and anything else as its JSON.
func skillText(raw json.RawMessage) string {
	var value any
	if json.Unmarshal(raw, &value) != nil || value == nil || value == false {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return string(bytes.TrimSpace(raw))
}

func spokenText(turn string) string {
	var said []string
	on := false
	for line := range strings.SplitSeq(turn, "\n") {
		switch {
		case line == "\x1eTEXT":
			on = true
		case strings.HasPrefix(line, "\x1e"):
			on = false
		case on:
			said = append(said, line)
		}
	}
	return strings.Join(said, "\n")
}
