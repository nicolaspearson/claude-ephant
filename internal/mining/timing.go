package mining

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

var (
	conventionFile = regexp.MustCompile(`CLAUDE\.md|AGENTS\.md|PATTERNS\.md|CONTRIBUTING\.md|\.golangci|\.editorconfig|reviewers/`)
	conventionRead = regexp.MustCompile(`(cat|sed|head).*(CLAUDE|AGENTS|PATTERNS)\.md`)
)

type ReviewerTiming struct {
	ConventionReads int
	GateRuns        int
	Role            string
	Round           int
	Seconds         *float64
	Tools           int
}

type TimedUnit struct {
	Analysis   string
	ReportedAt string
	Unit       dto.MiningUnit
}

type Timing struct {
	ParentBash   float64
	ParentGate   float64
	Reviewers    []ReviewerTiming
	Seconds      float64
	SubagentBash float64
	SubagentGate float64
}

type UnitKey struct {
	Analysis  string
	Session   string
	Timestamp string
}

// ReportedAt is "" for a result that does not parse or carries no report_ts.
func ReportedAt(judgment Judgment) string {
	var parsed result
	if json.Unmarshal(judgment.Result, &parsed) != nil {
		return ""
	}
	return parsed.ReportTimestamp
}

// Time reads each unit's window as its digest does, so a figure and a judgment cover the same lines, and each transcript once. A unit it cannot read is left untimed for the report to count, not fatal.
func Time(archiveDirectory string, units []TimedUnit) map[UnitKey]Timing {
	bySource := map[string][]TimedUnit{}
	for _, timed := range units {
		bySource[timed.Unit.Source] = append(bySource[timed.Unit.Source], timed)
	}
	timings := make(map[UnitKey]Timing, len(units))
	for _, source := range slices.Sorted(maps.Keys(bySource)) {
		if source == "" {
			continue
		}
		file, err := readTranscript(filepath.Join(archiveDirectory, filepath.FromSlash(source)))
		if err != nil {
			continue
		}
		for _, timed := range bySource[source] {
			if timing, err := timeUnit(file, timed); err == nil {
				timings[UnitKey{Analysis: timed.Analysis, Session: timed.Unit.Session, Timestamp: timed.Unit.Timestamp}] = timing
			}
		}
	}
	return timings
}

// bashSeconds pairs each Bash call with its result, both inside lines; a call whose result falls outside is not counted.
func bashSeconds(lines []digestLine) (all float64, gate float64) {
	type call struct {
		at   time.Time
		gate bool
	}
	open := map[string]call{}
	for _, line := range lines {
		for _, block := range line.blocks {
			switch {
			case block.Type == "tool_use" && block.Name == "Bash":
				command, _ := block.Input["command"].(string)
				open[block.ID] = call{at: line.at, gate: runGate.MatchString(command)}
			case block.Type == "tool_result":
				started, ok := open[block.ToolUseID]
				if !ok || started.at.IsZero() || line.at.IsZero() {
					continue
				}
				delete(open, block.ToolUseID)
				elapsed := line.at.Sub(started.at).Seconds()
				all += elapsed
				if started.gate {
					gate += elapsed
				}
			}
		}
	}
	return all, gate
}

func reviewerTiming(lines []digestLine, role string, round int) ReviewerTiming {
	timing := ReviewerTiming{Role: role, Round: round}
	var first, last time.Time
	for _, line := range lines {
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
			if block.Type != "tool_use" {
				continue
			}
			timing.Tools++
			command, _ := block.Input["command"].(string)
			path, _ := block.Input["file_path"].(string)
			switch block.Name {
			case "Bash":
				if runGate.MatchString(command) {
					timing.GateRuns++
				}
				if conventionRead.MatchString(command) {
					timing.ConventionReads++
				}
			case "Read":
				if conventionFile.MatchString(path) {
					timing.ConventionReads++
				}
			}
		}
	}
	if !first.IsZero() {
		elapsed := last.Sub(first).Seconds()
		timing.Seconds = &elapsed
	}
	return timing
}

func timeUnit(file transcriptFile, timed TimedUnit) (Timing, error) {
	opening, err := openingLine(file, timed.Unit)
	if err != nil {
		return Timing{}, err
	}
	var window []digestLine
	if timed.Analysis == "review-cycles" {
		window, _, err = cycleWindow(file, opening, timed.Unit)
	} else {
		window, err = runWindow(file, opening, timed.Unit)
	}
	if err != nil {
		return Timing{}, err
	}
	if reported, err := optionalTime(timed.ReportedAt); err == nil && !reported.IsZero() {
		end := len(window)
		for end > 0 && (window[end-1].at.IsZero() || window[end-1].at.After(reported)) {
			end--
		}
		window = window[:end]
	}
	// A cycle's window opens before its first launch for the classifier's context; timing starts at the launch.
	started := file.lines[opening].at
	for len(window) > 0 && (window[0].at.IsZero() || window[0].at.Before(started)) {
		window = window[1:]
	}
	var timing Timing
	for _, line := range window {
		if !line.at.IsZero() && (line.Type == "assistant" || line.Type == "user") {
			timing.Seconds = max(timing.Seconds, line.at.Sub(started).Seconds())
		}
	}
	timing.ParentBash, timing.ParentGate = bashSeconds(window)
	for _, line := range window {
		if line.Type != "assistant" {
			continue
		}
		for _, block := range line.blocks {
			if block.Type != "tool_use" || (block.Name != "Agent" && block.Name != "Task") {
				continue
			}
			id := file.agents[block.ID]
			if id == "" {
				continue
			}
			subagent, err := readTranscript(subagentPath(file.path, id))
			if err != nil {
				continue
			}
			all, gate := bashSeconds(subagent.lines)
			timing.SubagentBash += all
			timing.SubagentGate += gate
			prompt, _ := block.Input["prompt"].(string)
			reviewers := transcript.Reviewers(prompt)
			if len(reviewers) == 0 {
				continue
			}
			role := reviewers[0]
			if len(reviewers) > 1 {
				role = "combined"
			}
			description, _ := block.Input["description"].(string)
			round := 1
			if rereview.MatchString(description + " " + prompt[:min(600, len(prompt))]) {
				round = 2
			}
			timing.Reviewers = append(timing.Reviewers, reviewerTiming(subagent.lines, role, round))
		}
	}
	return timing, nil
}
