package mining

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/redact"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

const (
	cycleLeadIn  = 15 * time.Minute
	cycleReach   = 3 * time.Hour
	runReach     = 4 * time.Hour
	shownAskings = 2
	shownHumans  = 3
	shownLongs   = 2
	shownTools   = 30
)

var (
	agentID           = regexp.MustCompile(`agentId: ([A-Za-z0-9]+)`)
	checkpointCommand = regexp.MustCompile(`<command-name>/?(ele:)?checkpoint</command-name>`)
	checkpointSkill   = regexp.MustCompile(`^(ele:)?checkpoint$`)
	coveredLine       = regexp.MustCompile(`(?m)^(not )?covered: `)
	cycleStyle        = style{bash: reviewGate, editTools: []string{"Edit", "MultiEdit", "Write"}, humanLimit: 1000, longAbove: 1200, textLimit: 1200}
	generation        = regexp.MustCompile(`(\.g\d+)?\.jsonl\.gz$`)
	handback          = regexp.MustCompile(`<tool-use-id>([^<]+)</tool-use-id>|<task-id>([^<]+)</task-id>|<agent-message from="([^"]+)"`)
	harnessPrefixes   = []string{"<task-notification", "<agent-message", "Stop hook feedback", "<local-command", "Caveat:", "[Request interrupted", "Another Claude session", "<system-reminder"}
	recordCommand     = regexp.MustCompile(`(?m)(^|[;&|{(]\s*)\S*(checkpoint-reviewed\.sh"?\s+record\b|checkpoint-log\.sh\b|ele checkpoint log\b)`)
	reportLabel       = regexp.MustCompile(`\*\*(Bugs|Conventions|Prose|Spec)\b[^*\n]{0,30}reviewer`)
	reviewGate        = regexp.MustCompile(`go test|go build|golangci|\btask [a-z]|npm (run )?(test|lint|build)|pnpm|vitest|tsc|pytest|make |test\.sh|bun |playwright|checkpoint-|ele checkpoint |mutat|sed -i|perl -pi|git stash|git diff`)
	runGate           = regexp.MustCompile(`\b(go test|go build|go vet|golangci|task (test|lint|check|ci|gate|build|verify)|npm (run )?(test|lint|build|check)|pnpm|yarn (test|lint)|vitest|jest|pytest|make (test|lint|check|ci)|cargo test|tsc|eslint|test\.sh|bun test)\b`)
	runStyle          = style{editTools: []string{"Edit", "MultiEdit", "NotebookEdit", "Write"}, humanLimit: 1200, longAbove: 1500, otherTools: true, textLimit: 1500}
)

type agentCall struct {
	at          string
	description string
	failure     string
	prompt      string
	subagent    *subagentRun
}

type collected struct {
	agents     []agentCall
	bash       int
	covered    *bool
	edits      int
	events     []string
	gate       int
	hookBlocks int
	last       string
	longs      [][2]string
}

type contentBlock struct {
	Content   json.RawMessage `json:"content"`
	ID        string          `json:"id"`
	IsError   bool            `json:"is_error"`
	Input     map[string]any  `json:"input"`
	Name      string          `json:"name"`
	Text      string          `json:"text"`
	ToolUseID string          `json:"tool_use_id"`
	Type      string          `json:"type"`
}

type cycleHeader struct {
	Owner     string `json:"owner,omitempty"`
	Project   string `json:"project,omitempty"`
	Rereviews int    `json:"rereviews"`
	Reviewers int    `json:"reviewers"`
	Session   string `json:"session"`
	Timestamp string `json:"timestamp"`
}

type digestLine struct {
	Attachment *struct {
		Prompt json.RawMessage `json:"prompt"`
	} `json:"attachment"`
	Content json.RawMessage `json:"content"`
	IsMeta  bool            `json:"isMeta"`
	Message *struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
	Origin *struct {
		Kind string `json:"kind"`
	} `json:"origin"`
	Timestamp     string          `json:"timestamp"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
	Type          string          `json:"type"`

	at        time.Time
	blocks    []contentBlock
	handbacks []string
	text      string
}

type runHeader struct {
	AgentDescriptions []string   `json:"agentDescriptions"`
	AgentSeconds      []*float64 `json:"agentSeconds"`
	Agents            int        `json:"agents"`
	BashCommands      int        `json:"bashCommands"`
	Covered           *bool      `json:"covered"`
	DurationSeconds   float64    `json:"durationSeconds"`
	Edits             int        `json:"edits"`
	End               string     `json:"end"`
	Era               string     `json:"era"`
	GateCommands      int        `json:"gateCommands"`
	HookBlocks        int        `json:"hookBlocks"`
	NextHuman         string     `json:"nextHuman,omitempty"`
	Owner             string     `json:"owner,omitempty"`
	Project           string     `json:"project,omitempty"`
	Session           string     `json:"session"`
	Timestamp         string     `json:"timestamp"`
	TriggeredByHook   bool       `json:"triggeredByHook"`
}

type style struct {
	bash       *regexp.Regexp
	editTools  []string
	humanLimit int
	longAbove  int
	otherTools bool
	textLimit  int
}

type subagentRun struct {
	final   string
	seconds *float64
	tools   [][2]string
}

type transcriptFile struct {
	agents   map[string]string
	failures map[string]string
	lines    []digestLine
	path     string
}

// Digest renders a unit from its archived transcript and its subagents' for a classifier to judge. Transcript text is redacted before it is cut, so a cut cannot leave half a secret the patterns miss.
func Digest(archiveDirectory string, analysis string, unit dto.MiningUnit) (string, error) {
	if unit.Source == "" {
		return "", fmt.Errorf("unit %s %s: no archived transcript", unit.Session, unit.Timestamp)
	}
	file, err := readTranscript(filepath.Join(archiveDirectory, filepath.FromSlash(unit.Source)))
	if err != nil {
		return "", err
	}
	opening, err := openingLine(file, unit)
	if err != nil {
		return "", err
	}
	if analysis == "review-cycles" {
		return cycleDigest(file, opening, unit)
	}
	return runDigest(file, opening, unit)
}

func clip(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit])
}

func collect(file transcriptFile, window []digestLine, how style) collected {
	result := collected{last: window[0].Timestamp}
	for _, line := range window {
		if line.Timestamp != "" && (line.Type == "assistant" || line.Type == "user") {
			result.last = line.Timestamp
		}
		if line.isHuman() {
			result.events = append(result.events, fmt.Sprintf("%s HUMAN %s", line.Timestamp, quote(line.textOf(), how.humanLimit)))
		}
		if line.Type == "user" {
			said := line.textOf()
			if strings.Contains(said, "Stop hook feedback") && strings.Contains(said, "no checkpoint was reported") {
				result.hookBlocks++
				result.events = append(result.events, line.Timestamp+" HOOK-BLOCK")
			}
			if match := coveredLine.FindStringSubmatch(said); match != nil && result.covered == nil {
				covered := match[1] == ""
				result.covered = &covered
			}
		}
		if line.Type != "assistant" {
			continue
		}
		for _, block := range line.blocks {
			switch {
			case block.Type == "text" && strings.TrimSpace(block.Text) != "":
				text := redact.Redact(block.Text)
				event := fmt.Sprintf("%s TEXT %s", line.Timestamp, clip(text, how.textLimit))
				if utf8.RuneCountInString(text) > how.textLimit {
					event += " [TRUNCATED, see FINAL TEXTS]"
				}
				result.events = append(result.events, event)
				if utf8.RuneCountInString(text) > how.longAbove {
					result.longs = append(result.longs, [2]string{line.Timestamp, text})
				}
			case block.Type != "tool_use":
			case block.Name == "Agent" || block.Name == "Task":
				description, _ := block.Input["description"].(string)
				description = redact.Redact(description)
				prompt, _ := block.Input["prompt"].(string)
				call := agentCall{at: line.Timestamp, description: description, prompt: quote(prompt, 1500)}
				if id := file.agents[block.ID]; id != "" {
					call.subagent = readSubagent(subagentPath(file.path, id))
				}
				call.failure = file.failures[block.ID]
				event := fmt.Sprintf("%s AGENT %s", line.Timestamp, description)
				if how.otherTools {
					event += " dur=" + seconds(call.subagent)
				}
				result.agents = append(result.agents, call)
				result.events = append(result.events, event)
			case block.Name == "Bash":
				command, _ := block.Input["command"].(string)
				command = redact.Redact(command)
				if runGate.MatchString(command) {
					result.gate++
				}
				result.bash++
				if how.bash == nil || how.bash.MatchString(command) {
					result.events = append(result.events, fmt.Sprintf("%s BASH %s", line.Timestamp, clip(command, 200)))
				}
			case slices.Contains(how.editTools, block.Name):
				result.edits++
				path, _ := block.Input["file_path"].(string)
				result.events = append(result.events, fmt.Sprintf("%s %s %s", line.Timestamp, block.Name, redact.Redact(path)))
			case how.otherTools:
				encoded, _ := json.Marshal(block.Input)
				result.events = append(result.events, fmt.Sprintf("%s %s %s", line.Timestamp, block.Name, quote(string(encoded), 200)))
			}
		}
	}
	return result
}

func cycleDigest(file transcriptFile, opening int, unit dto.MiningUnit) (string, error) {
	window, next, err := cycleWindow(file, opening, unit)
	if err != nil {
		return "", err
	}
	var after []digestLine
	if next >= 0 {
		after = nextHumans(file.lines[next:])
	}
	found := collect(file, window, cycleStyle)

	var digest strings.Builder
	header, err := json.Marshal(cycleHeader{Owner: unit.Owner, Project: unit.Project, Rereviews: unit.Rereviews, Reviewers: unit.Reviewers, Session: unit.Session, Timestamp: unit.Timestamp})
	if err != nil {
		return "", err
	}
	digest.Write(header)
	digest.WriteString("\n\n=== EVENTS\n" + strings.Join(found.events, "\n") + "\n")
	for _, call := range found.agents {
		final := ""
		if call.subagent != nil {
			final = clip(call.subagent.final, 7000)
		}
		fmt.Fprintf(&digest, "\n=== AGENT %s at %s\n--- prompt\n%s\n", call.description, call.at, call.prompt)
		writeFailure(&digest, call)
		fmt.Fprintf(&digest, "--- final\n%s\n", final)
	}
	writeLongs(&digest, found.longs, 20000)
	for _, human := range after {
		fmt.Fprintf(&digest, "\n=== NEXT HUMAN %s\n%s\n", human.Timestamp, quote(human.textOf(), 1200))
	}
	return digest.String(), nil
}

// A cycle's window opens at the last human message before its first launch or the first line 15 minutes back, whichever is nearer, and closes at its session's next cycle, three hours on, or its report. next is the line after a report that closed it, or -1.
func cycleWindow(file transcriptFile, opening int, unit dto.MiningUnit) (window []digestLine, next int, err error) {
	lines := file.lines
	started := lines[opening].at
	until, err := optionalTime(unit.Until)
	if err != nil {
		return nil, -1, err
	}
	first := opening
	for first > 0 {
		line := lines[first]
		if !line.at.IsZero() && (line.isHuman() || started.Sub(line.at) >= cycleLeadIn) {
			break
		}
		first--
	}
	end := opening
	for ; end < len(lines); end++ {
		at := lines[end].at
		if !at.IsZero() && ((!until.IsZero() && !at.Before(until)) || at.Sub(started) > cycleReach) {
			break
		}
	}
	window = lines[first:end]
	back := reviewersBack(file, window)
	if report := reportCut(window[back:]); report >= 0 {
		return window[:back+report+1], first + back + report + 1, nil
	}
	return window, -1, nil
}

func nextHumans(lines []digestLine) []digestLine {
	var humans []digestLine
	for _, line := range lines {
		if len(humans) == shownHumans {
			break
		}
		if line.isHuman() {
			humans = append(humans, line)
		}
	}
	return humans
}

func openingLine(file transcriptFile, unit dto.MiningUnit) (int, error) {
	for position, line := range file.lines {
		if line.Timestamp == unit.Timestamp {
			return position, nil
		}
	}
	return -1, fmt.Errorf("unit %s %s: no line at that time in %s", unit.Session, unit.Timestamp, unit.Source)
}

func optionalTime(timestamp string) (time.Time, error) {
	if timestamp == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, timestamp)
}

func quote(text string, limit int) string {
	return clip(redact.Redact(text), limit)
}

func readSubagent(path string) *subagentRun {
	file, err := readTranscript(path)
	if err != nil || len(file.lines) == 0 {
		return nil
	}
	run := &subagentRun{}
	var first, last time.Time
	for _, line := range file.lines {
		if !line.at.IsZero() {
			if first.IsZero() {
				first = line.at
			}
			last = line.at
		}
		if line.Type != "assistant" {
			continue
		}
		for _, block := range line.blocks {
			switch {
			case block.Type == "text" && strings.TrimSpace(block.Text) != "":
				run.final = block.Text
			case block.Type == "tool_use":
				argument := ""
				for _, key := range []string{"command", "file_path", "pattern", "message"} {
					if value, ok := block.Input[key].(string); ok && value != "" {
						argument = value
						break
					}
				}
				run.tools = append(run.tools, [2]string{block.Name, quote(argument, 200)})
				if block.Name == "SubagentHandback" {
					run.final, _ = block.Input["message"].(string)
				}
			}
		}
	}
	run.final = redact.Redact(run.final)
	if !first.IsZero() {
		elapsed := last.Sub(first).Seconds()
		run.seconds = &elapsed
	}
	return run
}

// readTranscript skips lines that do not parse, as an interrupted write leaves one.
func readTranscript(path string) (transcriptFile, error) {
	reader, err := archive.Open(path)
	if err != nil {
		return transcriptFile{}, err
	}
	defer reader.Close()
	file := transcriptFile{agents: map[string]string{}, failures: map[string]string{}, path: path}
	buffered := bufio.NewReaderSize(reader, 1<<20)
	for {
		raw, readErr := buffered.ReadBytes('\n')
		if len(raw) > 0 {
			var line digestLine
			if json.Unmarshal(raw, &line) == nil {
				line.parse(file)
				line.readHandbacks()
				file.lines = append(file.lines, line)
			}
		}
		if errors.Is(readErr, io.EOF) {
			return file, nil
		}
		if readErr != nil {
			return transcriptFile{}, fmt.Errorf("read %s: %w", path, readErr)
		}
	}
}

// reportCut finds a checkpoint's report, or -1: the end of the turn that first runs the record or log command, else the last text carrying reviewer labels.
func reportCut(window []digestLine) int {
	labelled := -1
	for position, line := range window {
		if line.records() {
			return turnEnd(window, position)
		}
		if reportLabel.MatchString(line.assistantText()) {
			labelled = position
		}
	}
	return labelled
}

func resultText(content json.RawMessage) string {
	var text string
	if json.Unmarshal(content, &text) == nil {
		return text
	}
	var blocks []contentBlock
	if json.Unmarshal(content, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, block := range blocks {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// reviewersBack is where every reviewer launched in the window has handed back, or its end while one is out: text before it is progress, not a report.
func reviewersBack(file transcriptFile, window []digestLine) int {
	out := map[string]string{}
	back := 0
	for position, line := range window {
		for _, block := range line.blocks {
			if prompt, _ := block.Input["prompt"].(string); line.Type == "assistant" && block.Type == "tool_use" && (block.Name == "Agent" || block.Name == "Task") && len(transcript.Reviewers(prompt)) > 0 {
				out[block.ID] = file.agents[block.ID]
			}
		}
		for _, id := range line.handbacks {
			for launch, agent := range out {
				if id == launch || id == agent {
					delete(out, launch)
				}
			}
		}
		if len(out) > 0 {
			back = len(window)
		} else if back == len(window) {
			back = position + 1
		}
	}
	return back
}

func runDigest(file transcriptFile, opening int, unit dto.MiningUnit) (string, error) {
	lines := file.lines
	started := lines[opening].at
	window, err := runWindow(file, opening, unit)
	if err != nil {
		return "", err
	}
	var askings []digestLine
	for position := opening - 1; position >= 0 && len(askings) < shownAskings; position-- {
		if line := lines[position]; line.isHuman() && !checkpointCommand.MatchString(line.textOf()) {
			askings = append([]digestLine{line}, askings...)
		}
	}
	triggeredByHook := false
	for _, line := range lines[max(0, opening-6) : opening+1] {
		if line.Type == "user" && strings.Contains(line.textOf(), "Stop hook feedback") {
			triggeredByHook = true
		}
	}
	after := nextHumans(lines[opening+len(window):])
	found := collect(file, window, runStyle)

	ended, err := time.Parse(time.RFC3339Nano, found.last)
	if err != nil {
		return "", fmt.Errorf("unit %s: %w", unit.Session, err)
	}
	header := runHeader{
		Agents:          len(found.agents),
		BashCommands:    found.bash,
		Covered:         found.covered,
		DurationSeconds: ended.Sub(started).Seconds(),
		Edits:           found.edits,
		End:             found.last,
		Era:             "inline",
		GateCommands:    found.gate,
		HookBlocks:      found.hookBlocks,
		Owner:           unit.Owner,
		Project:         unit.Project,
		Session:         unit.Session,
		Timestamp:       unit.Timestamp,
		TriggeredByHook: triggeredByHook,
	}
	if !started.Before(subagentEraAt) {
		header.Era = "subagent"
	}
	if len(after) > 0 {
		header.NextHuman = after[0].Timestamp
	}
	for _, call := range found.agents {
		header.AgentDescriptions = append(header.AgentDescriptions, call.description)
		var elapsed *float64
		if call.subagent != nil {
			elapsed = call.subagent.seconds
		}
		header.AgentSeconds = append(header.AgentSeconds, elapsed)
	}
	encoded, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	var digest strings.Builder
	digest.Write(encoded)
	digest.WriteString("\n\n")
	for _, asking := range askings {
		fmt.Fprintf(&digest, "=== PRIOR USER ASK %s\n%s\n", asking.Timestamp, quote(asking.textOf(), 2000))
	}
	digest.WriteString("\n=== RUN EVENTS\n" + strings.Join(found.events, "\n") + "\n")
	for _, call := range found.agents {
		fmt.Fprintf(&digest, "\n=== AGENT %s at %s\n--- prompt\n%s\n", call.description, call.at, call.prompt)
		writeFailure(&digest, call)
		if call.subagent == nil {
			continue
		}
		fmt.Fprintf(&digest, "--- subagent duration %ss, tools:\n", strings.TrimSuffix(seconds(call.subagent), "s"))
		for _, tool := range call.subagent.tools[:min(shownTools, len(call.subagent.tools))] {
			fmt.Fprintf(&digest, "  %s %s\n", tool[0], clip(tool[1], 100))
		}
		digest.WriteString("--- final\n" + clip(call.subagent.final, 9000) + "\n")
	}
	writeLongs(&digest, found.longs, 30000)
	for _, human := range after {
		fmt.Fprintf(&digest, "\n=== NEXT HUMAN %s\n%s\n", human.Timestamp, quote(human.textOf(), 1500))
	}
	return digest.String(), nil
}

// A run's window closes at the next checkpoint invocation, its Until, four hours on, or its report.
func runWindow(file transcriptFile, opening int, unit dto.MiningUnit) ([]digestLine, error) {
	lines := file.lines
	started := lines[opening].at
	until, err := optionalTime(unit.Until)
	if err != nil {
		return nil, err
	}
	end := opening + 1
	for ; end < len(lines); end++ {
		line := lines[end]
		if !line.at.IsZero() && ((!until.IsZero() && !line.at.Before(until)) || line.at.Sub(started) > runReach) {
			break
		}
		if line.invokesCheckpoint() {
			break
		}
	}
	window := lines[opening:end]
	back := reviewersBack(file, window)
	if report := reportCut(window[back:]); report >= 0 {
		window = window[:back+report+1]
	}
	return window, nil
}

func seconds(run *subagentRun) string {
	if run == nil || run.seconds == nil {
		return "?"
	}
	return fmt.Sprintf("%.0fs", *run.seconds)
}

// subagentPath reads the transcript's own name, not its lines' session: a resumed transcript's lines can carry another session id.
func subagentPath(transcriptPath string, agent string) string {
	directory, name := filepath.Split(transcriptPath)
	return filepath.Join(directory, generation.ReplaceAllString(name, ""), "subagents", "agent-"+agent+".jsonl.gz")
}

// turnEnd is the last assistant text from position on before the user next speaks.
func turnEnd(window []digestLine, position int) int {
	end := position
	for next := position + 1; next < len(window) && !window[next].isHuman(); next++ {
		if window[next].assistantText() != "" {
			end = next
		}
	}
	return end
}

func writeFailure(digest *strings.Builder, call agentCall) {
	if call.failure != "" {
		digest.WriteString("--- launch failed\n" + quote(call.failure, 500) + "\n")
	}
}

func writeLongs(digest *strings.Builder, longs [][2]string, limit int) {
	for _, long := range longs[max(0, len(longs)-shownLongs):] {
		fmt.Fprintf(digest, "\n=== FINAL TEXTS (full) %s\n%s\n", long[0], clip(long[1], limit))
	}
}

func (l *digestLine) assistantText() string {
	if l.Type != "assistant" {
		return ""
	}
	var texts []string
	for _, block := range l.blocks {
		if block.Type == "text" {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, " ")
}

func (l *digestLine) invokesCheckpoint() bool {
	if l.Type == "assistant" {
		for _, block := range l.blocks {
			if skill, _ := block.Input["skill"].(string); block.Type == "tool_use" && block.Name == "Skill" && checkpointSkill.MatchString(skill) {
				return true
			}
		}
		return false
	}
	return l.isHuman() && checkpointCommand.MatchString(l.textOf())
}

// isHuman is a line the user typed: no tool result, no harness wrapper, and a human origin where one is recorded.
func (l *digestLine) isHuman() bool {
	if l.Type != "user" || l.IsMeta {
		return false
	}
	for _, block := range l.blocks {
		if block.Type == "tool_result" {
			return false
		}
	}
	said := strings.TrimLeft(l.textOf(), " \t\r\n")
	if said == "" {
		return false
	}
	for _, prefix := range harnessPrefixes {
		if strings.HasPrefix(said, prefix) {
			return false
		}
	}
	return l.Origin == nil || l.Origin.Kind == "human"
}

// parse records each tool result's agent id, which names its subagent transcript, or its error; any result but a background launch's ack hands its tool use back.
func (l *digestLine) parse(file transcriptFile) {
	if l.Timestamp != "" {
		l.at, _ = time.Parse(time.RFC3339Nano, l.Timestamp)
	}
	if l.Message == nil {
		return
	}
	if json.Unmarshal(l.Message.Content, &l.text) == nil {
		return
	}
	if json.Unmarshal(l.Message.Content, &l.blocks) != nil {
		l.blocks = nil
		return
	}
	var result struct {
		AgentID string `json:"agentId"`
		Status  string `json:"status"`
	}
	_ = json.Unmarshal(l.ToolUseResult, &result)
	for _, block := range l.blocks {
		if block.Type != "tool_result" || block.ToolUseID == "" {
			continue
		}
		text := resultText(block.Content)
		if result.Status != "async_launched" {
			l.handbacks = append(l.handbacks, block.ToolUseID)
		}
		// A finished agent's result quotes its report, which can name another agent id.
		switch match := agentID.FindStringSubmatch(text); {
		case block.IsError:
			file.failures[block.ToolUseID] = text
		case result.AgentID != "":
			file.agents[block.ToolUseID] = result.AgentID
		case match != nil:
			file.agents[block.ToolUseID] = match[1]
		}
	}
}

// readHandbacks reads a background agent's finish from wherever the harness delivered it: a queued notice, its attachment or the message carrying it.
func (l *digestLine) readHandbacks() {
	if l.Type == "assistant" {
		return
	}
	var queued, prompt string
	_ = json.Unmarshal(l.Content, &queued)
	if l.Attachment != nil {
		_ = json.Unmarshal(l.Attachment.Prompt, &prompt)
	}
	for _, said := range []string{l.textOf(), queued, prompt} {
		for _, match := range handback.FindAllStringSubmatch(said, -1) {
			l.handbacks = append(l.handbacks, match[1]+match[2]+match[3])
		}
	}
}

func (l *digestLine) records() bool {
	if l.Type != "assistant" {
		return false
	}
	for _, block := range l.blocks {
		if command, _ := block.Input["command"].(string); block.Type == "tool_use" && block.Name == "Bash" && recordCommand.MatchString(command) {
			return true
		}
	}
	return false
}

func (l *digestLine) textOf() string {
	if l.blocks == nil {
		return l.text
	}
	var parts []string
	for _, block := range l.blocks {
		switch block.Type {
		case "text":
			parts = append(parts, block.Text)
		case "tool_result":
			parts = append(parts, resultText(block.Content))
		}
	}
	return strings.Join(parts, "\n")
}
