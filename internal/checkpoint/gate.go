package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
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
	commitCommand = regexp.MustCompile(`(?i)git commit`)
	doneClaim     = regexp.MustCompile(`(?i)git commit|ready (to|for) (commit|merge|review|ship)|(all|work is|that.s|now) done|is complete`)
	laterCommit   = regexp.MustCompile(`(?i)\b(once|after|when)\b[^.!?]*\byou can (then )?commit\b`)
	nothingLeft   = regexp.MustCompile(`(?i)\b(nothing|none|no|zero|0)\b[^.;,]*\b(left|remains?|to go)\b`)
	// A pass label, not a fixed sentence: demanding the literal "quality review" and "prose audit" blocked turns that had just run the checkpoint.
	prosePass    = regexp.MustCompile(`(?i)(prose|comments?)[[:space:]]*(audit|pass|review|sweep|—|-|:)`)
	qualityPass  = regexp.MustCompile(`(?i)quality[[:space:]]*(review|pass|—|-|:)|bugs[[:space:]]+review`)
	skillInvoked = regexp.MustCompile(`(?i)\x1eTOOL Skill (ele:)?checkpoint|command-name>/?(ele:)?checkpoint|^\x1eTURN */(ele:)?checkpoint`)
	// A count of what is left, "now done; two are left", reports a step; a bare "left" does not, as in "changes left in the working tree".
	stillLeft = regexp.MustCompile(`(?i)\b(are|is) (still )?left\b|\bleft to (do|go)\b|\b(one|two|three|four|five|[0-9]+) (more )?(remains?|to go)\b|\bstill to (do|go|come)\b`)
)

type GateInput struct {
	// IsWork says whether a Bash command may write; nil counts every one as work.
	IsWork         func(command string) bool
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
	if len(moved) == 0 && now.Capped == 0 {
		return ""
	}
	turn := lastTurn(input.TranscriptPath)
	// Past the cap a path has no hash to move: the turn's edits stand in, and a placeholder for a Bash write, which names none.
	if now.Capped > 0 {
		edited, bashWork := turnEdits(turn, input.Root, input.IsWork)
		moved = append(moved, slices.Collect(maps.Keys(edited))...)
		if len(edited) == 0 && bashWork {
			moved = append(moved, fmt.Sprintf("(Bash writes past the %d-path cap)", pathCap))
		}
		slices.Sort(moved)
		moved = slices.Compact(moved)
	}
	moved = ownPaths(moved, turn, input.Root, input.IsWork)
	if len(moved) == 0 || Covered(ctx, input.Root, now, moved) || Open(ctx, input.Root, input.Session) || OptedOut(ctx, input.Root, input.Session, moved) {
		return ""
	}
	if !claimsDone(finalText(turn)) || anyLine(skillInvoked, withoutBash(turn)) {
		return ""
	}
	// Spoken text only: a prompt naming the quality and prose passes is not them run.
	said := spokenText(turn)
	if anyLine(qualityPass, said) && anyLine(prosePass, said) {
		return ""
	}
	if len(moved) > blockedChanged {
		moved = moved[:blockedChanged]
	}
	return "Files changed this turn, the reply says the work is done, and no checkpoint was reported.\n\nChanged: " + strings.Join(moved, " ") +
		"\n\nThe `ele:checkpoint` skill reviews work before it is reported done.\n\nWork not done — mid-task, waiting on the user, or an investigation — is not a checkpoint: the reply says so and what remains, and the turn ends.\n\nWork the user said needs no checkpoint has its paths exempted for this session with `ele checkpoint optout " + shellQuote(input.Root) + " \"<glob>\" ...`"
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

func claimsDone(text string) bool {
	commit := false
	for line := range strings.SplitSeq(text, "\n") {
		switch {
		case !doneClaim.MatchString(line):
		case commitCommand.MatchString(line):
			commit = true
		case !stillLeft.MatchString(line) || nothingLeft.MatchString(line):
			return true
		}
	}
	return commit && !anyLine(laterCommit, text)
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

func inputString(raw json.RawMessage) string {
	var text string
	_ = json.Unmarshal(raw, &text)
	return text
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
			if prompt, typed := promptText(entry.Message.Content); typed && !entry.IsMeta {
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

// ownPaths is the moved paths this turn's repository edits named, since sessions share the tree; with none, every
// moved path if a Bash command may have written, since Bash names no path, else none.
func ownPaths(moved []string, turn string, root string, isWork func(string) bool) []string {
	// The tail cut the turn's start, and with it edits this turn made.
	if !strings.HasPrefix(turn, "\x1eTURN") {
		return moved
	}
	paths, bashWork := turnEdits(turn, root, isWork)
	if len(paths) == 0 {
		if bashWork {
			return moved
		}
		return nil
	}
	var own []string
	for _, path := range moved {
		if paths[path] {
			own = append(own, path)
		}
	}
	return own
}

func turnEdits(turn string, root string, isWork func(string) bool) (map[string]bool, bool) {
	edited := map[string]bool{}
	bashWork := false
	for line := range strings.SplitSeq(turn, "\n") {
		call, ok := strings.CutPrefix(line, "\x1eTOOL ")
		if !ok {
			continue
		}
		name, argument, _ := strings.Cut(call, " ")
		value, err := strconv.Unquote(argument)
		switch name {
		case "Bash":
			bashWork = bashWork || err != nil || isWork == nil || isWork(value)
		case "Edit", "MultiEdit", "NotebookEdit", "Write":
			if path := repositoryPath(root, value); err == nil && value != "" && !filepath.IsAbs(path) && !strings.HasPrefix(path, "..") {
				edited[path] = true
			}
		}
	}
	return edited, bashWork
}

// A typed prompt's content is a string, or, with an image pasted, an array of text and image parts; a tool result's
// is an array of tool_result parts, and Claude Code's interrupt marker a lone text part, neither of which starts a turn.
func promptText(content json.RawMessage) (string, bool) {
	var prompt string
	if json.Unmarshal(content, &prompt) == nil {
		return prompt, true
	}
	var parts []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	}
	if json.Unmarshal(content, &parts) != nil {
		return "", false
	}
	var texts []string
	typed := false
	for _, part := range parts {
		switch part.Type {
		case "tool_result":
			// An answer with an image attached carries one beside its tool result: still the turn's own reply.
			return "", false
		case "image":
			typed = true
		case "text":
			if strings.HasPrefix(part.Text, "[Request interrupted by user") {
				continue
			}
			typed = true
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n"), typed
}

func renderAssistant(content json.RawMessage) string {
	// Raw, not string: another tool's input of another type under one of these keys would fail the whole message.
	var items []struct {
		Input struct {
			Command      json.RawMessage `json:"command"`
			FilePath     json.RawMessage `json:"file_path"`
			NotebookPath json.RawMessage `json:"notebook_path"`
			Skill        json.RawMessage `json:"skill"`
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
			// Quoted: a command or path spans lines, and the turn is read line by line.
			switch item.Name {
			case "Bash":
				parts = append(parts, "\x1eTOOL Bash "+strconv.Quote(inputString(item.Input.Command)))
			case "Edit", "MultiEdit", "NotebookEdit", "Write":
				path := item.Input.FilePath
				if item.Name == "NotebookEdit" {
					path = item.Input.NotebookPath
				}
				parts = append(parts, "\x1eTOOL "+item.Name+" "+strconv.Quote(inputString(path)))
			default:
				parts = append(parts, "\x1eTOOL "+item.Name+" "+skillText(item.Input.Skill))
			}
		}
	}
	return strings.Join(parts, "\n")
}

// repositoryPath is path relative to root, as git names it; a symlinked /tmp or checkout spells the same file another way.
func repositoryPath(root string, path string) string {
	if !filepath.IsAbs(path) {
		return filepath.ToSlash(filepath.Clean(path))
	}
	if relative, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(relative)
	}
	directory, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return path
	}
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return path
	}
	if relative, err := filepath.Rel(physicalRoot, filepath.Join(directory, filepath.Base(path))); err == nil && !strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(relative)
	}
	return path
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

// withoutBash drops Bash lines: a command that greps transcripts for a skill call is not one.
func withoutBash(turn string) string {
	var kept []string
	for line := range strings.SplitSeq(turn, "\n") {
		if !strings.HasPrefix(line, "\x1eTOOL Bash ") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}
