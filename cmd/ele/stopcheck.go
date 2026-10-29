package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/highlight"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/plan"
	"github.com/nicolaspearson/claude-ephant/internal/redact"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

const (
	hintBand = 100000
	noBand   = -1
	// Proposes a skill, not notes: work before it stays pending.
	recipeSkill         = "ele:recipe"
	rememberReason      = "Follow /ele:remember's Propose section now: one Write/Skip question per note, at most four per AskUserQuestion call and as many calls as it takes, then write each approved note yourself with ele remember. Never print the command for the user to run. If nothing is worth keeping, say so in one line."
	rememberWriteReason = "Follow /ele:remember's Propose section now: auto_note_writes is on, so write each note yourself with ele remember without asking, then report each id with the note's words. Never print the command for the user to run. If nothing is worth keeping, say so in one line."
	reviewReader        = "ele:review-reader"
	stopStateMaxAge     = 7 * 24 * time.Hour
	tailWindow          = 1 << 20
)

var (
	checkpointNames = []string{"checkpoint", "ele:checkpoint"}
	proposalNames   = []string{"ele:handoff", "ele:remember", "ele:wrap"}
	// Not Bash: research and review subagents run it to read.
	subagentWorkTools = []string{"Edit", "NotebookEdit", "Write"}
	// Bash unless it is a read: a command can change files, and /clear mid-task loses more than /compact between tasks.
	workTools = []string{"Bash", "Edit", "NotebookEdit", "Write"}
)

type stopState struct {
	Band              int              `json:"band"`
	CheckpointStarted string           `json:"checkpointStarted,omitempty"`
	InCheckpoint      bool             `json:"inCheckpoint"`
	InReview          bool             `json:"inReview"`
	MidTask           bool             `json:"midTask"`
	NotesProposed     string           `json:"notesProposed,omitempty"`
	Offset            int64            `json:"offset"`
	Pending           bool             `json:"pending"`
	Proposing         bool             `json:"proposing"`
	StretchBlocked    bool             `json:"stretchBlocked"`
	Subagents         map[string]int64 `json:"subagents"`
}

type stopOutput struct {
	Decision      string `json:"decision,omitempty"`
	Reason        string `json:"reason,omitempty"`
	SystemMessage string `json:"systemMessage,omitempty"`
}

// A checkpoint's own gate runs and fixes are not new work; the next prompt ends it. A checkpoint starts a stretch but
// keeps unproposed work, so a stop after it still asks for notes. A proposal's own calls, after a remember block or
// proposal until the next prompt, are not work: counted, a proposal that writes nothing would ask again. A review's calls
// are not work either, from its readers' launch across prompts until its log line: its turns end while readers run and at a Print it.
func advance(state stopState, entries []transcript.Entry, events []dto.TranscriptEvent) stopState {
	for _, entry := range entries {
		switch {
		case transcript.Prompted(entry.Kind):
			state.InCheckpoint, state.Proposing = false, false
		case entry.Kind != transcript.KindTool:
		case strings.HasPrefix(entry.Text, "Bash: ") && writesNote(entry.Command):
			state.NotesProposed, state.Pending = max(state.NotesProposed, entry.Timestamp), false
			if !state.InCheckpoint && !state.InReview && !state.Proposing {
				other, work := besideNotes(entry.Command)
				state.MidTask, state.Pending = state.MidTask || other || work, work
			}
		case state.InReview && strings.HasPrefix(entry.Text, "Bash: ") && logsCheckpoint(entry.Command):
			state.InReview = false
		case state.InCheckpoint || state.InReview || state.Proposing:
		case strings.HasPrefix(entry.Text, "AskUserQuestion: "):
			state.Pending = true
		// Mid-task, not work: /clear mid-investigation loses what the reads found.
		case strings.HasPrefix(entry.Text, "Bash: ") && !commandIsWork(entry.Command):
			state.MidTask = true
		case slices.ContainsFunc(workTools, func(name string) bool { return strings.HasPrefix(entry.Text, name+": ") }):
			state.MidTask, state.Pending = true, true
		}
	}
	for _, event := range events {
		switch {
		// Not at the review's command: a run refused at its target never logs, so its stretch would never end.
		case event.Kind == transcript.EventAgent && event.Name == reviewReader:
			state.InReview = true
		case event.Kind != transcript.EventSkill && event.Kind != transcript.EventCommand:
		case slices.Contains(checkpointNames, event.Name):
			state.CheckpointStarted, state.InCheckpoint, state.InReview, state.MidTask, state.StretchBlocked = event.Timestamp, true, false, false, false
		case slices.Contains(proposalNames, event.Name):
			state.NotesProposed, state.Pending, state.Proposing = max(state.NotesProposed, event.Timestamp), false, true
		case event.Name == recipeSkill:
			state.Proposing = true
		}
	}
	return state
}

// An edit stamped before the last proposal was cleared by it. Else, while a checkpoint runs, one stamped after it started
// is a reviewer's. An unstamped one is work: a doubtful edit costs an ask, not lost work.
func subagentEdit(state stopState, timestamp string) stopState {
	stamped := timestamp != ""
	switch {
	case stamped && timestamp < state.NotesProposed:
		state.MidTask = state.MidTask || !state.InCheckpoint
	case !state.InCheckpoint:
		state.MidTask, state.Pending = true, true
	case !stamped || timestamp <= state.CheckpointStarted:
		state.Pending = true
	}
	return state
}

func subagentWorks(entry transcript.Entry) bool {
	return entry.Kind == transcript.KindTool && slices.ContainsFunc(subagentWorkTools, func(tool string) bool { return strings.HasPrefix(entry.Text, tool+": ") })
}

// Only whole lines are consumed: a live transcript's last line can be half written.
func consumeLines(file *os.File, offset int64, consume func(line []byte)) (int64, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return offset, fmt.Errorf("read %s: %w", file.Name(), err)
	}
	lines := bufio.NewReaderSize(file, 1<<20)
	for {
		line, err := lines.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			return offset, nil
		}
		if err != nil {
			return offset, fmt.Errorf("read %s: %w", file.Name(), err)
		}
		offset += int64(len(line))
		consume(line)
	}
}

// Read from the end: the hook runs every turn, on a transcript that can pass 100 MB. A line cut by the window fails
// to parse and is read whole by the next.
func lastContextTokens(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	end := info.Size()
	for window := int64(tailWindow); ; window *= 4 {
		start := max(0, end-window)
		chunk := make([]byte, end-start)
		if _, err = file.ReadAt(chunk, start); err != nil && !errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("read %s: %w", path, err)
		}
		for _, line := range slices.Backward(bytes.Split(chunk, []byte("\n"))) {
			if tokens, ok := transcript.ContextTokens(line); ok {
				return tokens, nil
			}
		}
		if start == 0 {
			return 0, nil
		}
	}
}

func lonePlanFocus(dataDirectory string, directory string, owner string, project string) string {
	all, _, err := notes.ReadOwner(filepath.Join(dataDirectory, "notes"), owner)
	if err != nil {
		return ""
	}
	active := activePlans(all, project)
	if len(active) != 1 {
		return ""
	}
	root, err := filepath.EvalSymlinks(index.GitRoot(directory))
	if err != nil {
		return ""
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(active[0].Body)))
	if err != nil {
		return ""
	}
	parsed, err := plan.Parse(content)
	if err != nil {
		return ""
	}
	next, ok := loadPlanSet(root, all, project, filepath.Join(dataDirectory, "notes")).resolve(parsed).Next()
	if !ok {
		return ""
	}
	return redact.Redact(next.Title)
}

// A file that does not parse reads as a fresh session, so a write cut short by the timeout cannot fail every later stop.
func readStopState(path string) (stopState, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return stopState{Band: noBand}, nil
	}
	if err != nil {
		return stopState{}, err
	}
	var state stopState
	if json.Unmarshal(content, &state) != nil {
		return stopState{Band: noBand}, nil
	}
	return state, nil
}

// Not mid-checkpoint or mid-review: either's turn can end while reviewers run.
func rememberBlock(state stopState, stopHookActive bool) (bool, stopState) {
	if !state.Pending || state.InCheckpoint || state.InReview || state.StretchBlocked || stopHookActive {
		return false, state
	}
	state.Pending, state.Proposing, state.StretchBlocked = false, true, true
	return true, state
}

func removeStaleStopState(dataDirectory string, now time.Time) error {
	directory := filepath.Join(dataDirectory, "stop-check")
	entries, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		info, infoErr := entry.Info()
		if errors.Is(infoErr, fs.ErrNotExist) {
			continue
		}
		if infoErr != nil {
			return infoErr
		}
		if now.Sub(info.ModTime()) > stopStateMaxAge {
			if err = os.Remove(filepath.Join(directory, entry.Name())); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

// Read after the transcript. With no saved state, existing files are skipped unread: a long session's scan cost.
func scanSubagents(directory string, state stopState) (stopState, error) {
	unread := state.Subagents == nil
	if unread {
		state.Subagents = map[string]int64{}
	}
	listing, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	for _, listed := range listing {
		name := listed.Name()
		if !strings.HasPrefix(name, "agent-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		path := filepath.Join(directory, name)
		info, statErr := os.Stat(path)
		if errors.Is(statErr, fs.ErrNotExist) {
			continue
		}
		if statErr != nil {
			return state, statErr
		}
		offset := state.Subagents[name]
		if unread || info.Size() == offset {
			state.Subagents[name] = info.Size()
			continue
		}
		if info.Size() < offset {
			offset = 0
		}
		file, openErr := os.Open(path)
		if errors.Is(openErr, fs.ErrNotExist) {
			continue
		}
		if openErr != nil {
			return state, openErr
		}
		offset, err = consumeLines(file, offset, func(line []byte) {
			entries, _, parseErr := transcript.Parse(line)
			if parseErr != nil {
				return
			}
			for _, entry := range entries {
				if subagentWorks(entry) {
					state = subagentEdit(state, entry.Timestamp)
				}
			}
		})
		file.Close()
		if err != nil {
			return state, err
		}
		state.Subagents[name] = offset
	}
	return state, nil
}

// A transcript shorter than the offset was replaced, so it is read again from the start.
func scanTranscript(path string, state stopState) (stopState, error) {
	file, err := os.Open(path)
	if err != nil {
		return state, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return state, err
	}
	if info.Size() < state.Offset {
		state = stopState{Band: state.Band}
	}
	offset, err := consumeLines(file, state.Offset, func(line []byte) {
		if entries, events, parseErr := transcript.Parse(line); parseErr == nil {
			state = advance(state, entries, events)
		}
	})
	state.Offset = offset
	return state, err
}

// Reset below the threshold: a compacted session that regrows is hinted again.
func sizeHint(tokens int, threshold int, hinted int, midTask bool, focus string) (string, int) {
	if tokens < threshold {
		return "", noBand
	}
	band := (tokens - threshold) / hintBand
	if band <= hinted {
		return "", hinted
	}
	size := fmt.Sprintf("Context %dk.", tokens/1000)
	if !midTask {
		return size + " Write a handoff (/ele:handoff), then /clear", band
	}
	if focus == "" {
		return size + " At a pause: /compact", band
	}
	return size + " At a pause: /compact " + focus, band
}

func stopCheck(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("stop-check", stderr)
	replay := flags.Bool("replay", false, "replay the hook as built over each Stop of the archived main transcripts with a line in the window, one JSON line per stop")
	since := flags.String("since", "", "with --replay, the window's start: YYYY-MM-DD or an RFC 3339 time")
	until := flags.String("until", "", "with --replay, the window's end, exclusive: YYYY-MM-DD or an RFC 3339 time; default now")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 || *replay != (*since != "") || (*until != "" && !*replay) {
		return rejectCall(flags, err, stdout, stderr)
	}
	if *replay {
		from, err := replayTime(*since, time.Time{})
		if err != nil {
			fmt.Fprintf(stderr, "ele: --since: %v\n", err)
			return 2
		}
		to, err := replayTime(*until, time.Now())
		if err != nil {
			fmt.Fprintf(stderr, "ele: --until: %v\n", err)
			return 2
		}
		return stopReplay(ctx, from, to, stdout, stderr)
	}
	var hook struct {
		Directory      string `json:"cwd"`
		SessionID      string `json:"session_id"`
		StopHookActive bool   `json:"stop_hook_active"`
		TranscriptPath string `json:"transcript_path"`
	}
	if !readHookInput(stdin, &hook) || hook.SessionID == "" || !notes.ValidSession(hook.SessionID) || hook.TranscriptPath == "" || hook.Directory == "" {
		fmt.Fprintln(stderr, "ele: stop-check reads a Stop hook's JSON on stdin, with session_id, transcript_path and cwd")
		return 1
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	ownerMap, err := ownermap.Load(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	project, owner := index.NewProjects(ownerMap).Resolve(hook.Directory)
	if owner == "" {
		return 0
	}
	tokens, err := lastContextTokens(hook.TranscriptPath)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	statePath := filepath.Join(dataDirectory, "stop-check", hook.SessionID+".json")
	state, err := readStopState(statePath)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	if state, err = scanTranscript(hook.TranscriptPath, state); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	if state, err = scanSubagents(subagentDirectory(hook.TranscriptPath), state); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	settings := loadConfig(dataDirectory, stderr)
	var output stopOutput
	if hintAt := compactHintAt(settings, owner); hintAt > 0 {
		output.SystemMessage, state.Band = sizeHint(tokens, hintAt, state.Band, state.MidTask, "")
		if output.SystemMessage != "" && state.MidTask {
			output.SystemMessage, _ = sizeHint(tokens, hintAt, noBand, true, lonePlanFocus(dataDirectory, hook.Directory, owner, project))
		}
	} else {
		// Band reset while off, so one hinted before does not hold back the hint once on.
		state.Band = noBand
	}
	// Pending cleared while off, so turning it on asks only for work done after.
	if !notesAskOn(settings, owner) {
		state.Pending = false
	}
	var block bool
	if block, state = rememberBlock(state, hook.StopHookActive); block {
		output.Decision, output.Reason = "block", rememberReason
		if value, valueErr := settings.Value("auto_note_writes", owner); valueErr == nil && value.Value == true {
			output.Reason = rememberWriteReason
		}
	}
	if err = writeStopState(statePath, state); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	if output == (stopOutput{}) {
		return 0
	}
	output.SystemMessage = highlight.Message(output.SystemMessage, colourOn(settings, owner))
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err = encoder.Encode(output); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}

// <session>/subagents, beside the session's transcript.
func subagentDirectory(transcriptPath string) string {
	return filepath.Join(strings.TrimSuffix(transcriptPath, filepath.Ext(transcriptPath)), "subagents")
}

func writeStopState(path string, state stopState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	content, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o600)
}
