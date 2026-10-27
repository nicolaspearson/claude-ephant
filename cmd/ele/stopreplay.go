package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

const (
	stopPlaced   = "placed"
	stopRecorded = "recorded"
	// Claude Code writes one per Stop.
	stopSummaryType = "stop_hook_summary"
)

// Hook blocks closer than this, with nothing between, are one stop's hooks.
const blockGroup = 2 * time.Second

type replayStop struct {
	Asked     bool     `json:"asked"`
	Asks      bool     `json:"asks"`
	Owner     string   `json:"owner"`
	Project   string   `json:"project"`
	Session   string   `json:"session"`
	Stop      string   `json:"stop"`
	Timestamp string   `json:"timestamp"`
	Trigger   []string `json:"trigger,omitempty"`
	WhyNot    string   `json:"whyNot,omitempty"`
	key       string
}

type stopSummary struct {
	IsSidechain bool   `json:"isSidechain"`
	Subtype     string `json:"subtype"`
	Timestamp   string `json:"timestamp"`
	Type        string `json:"type"`
	UUID        string `json:"uuid"`
}

// Replays one transcript's lines through the hook's state, judging a stop at each recorded stop_hook_summary not already
// seen or, in a transcript with none, where one must have run: at a group of Stop hook blocks, before a prompt that follows output,
// and at its last line after output.
type replayer struct {
	active    bool // a Stop hook blocked since the last prompt, so the next stop runs with stop_hook_active
	asked     bool // a notes ask blocked since the last recorded stop
	blocked   bool // a Stop hook blocked since the last recorded stop
	blockStop bool // the last placed stop was a hook block's
	edits     []string
	grouped   bool // the current block group's stop was judged here, not skipped as copied
	judged    bool
	lastBlock time.Time
	output    bool
	prompted  bool
	proposed  bool
	recorded  bool
	seen      map[string]bool
	state     stopState
	stops     []replayStop
	trigger   []string
}

func readArchived(path string) ([]byte, error) {
	reader, err := archive.Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return content, nil
}

func replayStamp(at time.Time) string {
	return at.UTC().Format("2006-01-02T15:04:05.000Z")
}

func replayTime(value string, empty time.Time) (time.Time, error) {
	if value == "" {
		return empty, nil
	}
	if at, err := time.Parse(time.RFC3339, value); err == nil {
		return at, nil
	}
	at, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("takes YYYY-MM-DD or an RFC 3339 time like 2026-10-20T04:42:00Z, not %q", value)
	}
	return at, nil
}

func replayTranscript(content []byte, edits []string, seen map[string]bool) []replayStop {
	r := replayer{edits: edits, recorded: bytes.Contains(content, []byte(`"`+stopSummaryType+`"`)), seen: seen, state: stopState{Band: noBand}}
	var lastStamp, lastUUID string
	for line := range bytes.SplitSeq(content, []byte("\n")) {
		if r.recorded && bytes.Contains(line, []byte(`"`+stopSummaryType+`"`)) {
			var summary stopSummary
			if json.Unmarshal(line, &summary) == nil && summary.Type == "system" && summary.Subtype == stopSummaryType && !summary.IsSidechain {
				r.stop(summary.Timestamp, summary.UUID, stopRecorded, r.asked)
				r.active, r.asked, r.blocked = r.blocked, false, false
				continue
			}
		}
		entries, events, err := transcript.Parse(line)
		if err != nil {
			continue
		}
		feedback := false
		for _, event := range events {
			if event.Kind == transcript.EventHookBlock && event.Name == "Stop" {
				r.block(event.Timestamp, event.UUID, strings.Contains(fmt.Sprint(event.Attributes["reason"]), "ele:remember"))
				feedback = true
			}
		}
		for _, entry := range entries {
			r.entry(entry, feedback)
			lastStamp, lastUUID = entry.Timestamp, entry.UUID
		}
		r.step(advance(r.state, nil, events))
	}
	if !r.recorded && r.output {
		r.stop(lastStamp, cmp.Or(lastUUID, "@"+lastStamp)+"#end", stopPlaced, false)
	}
	return r.stops
}

func stopReplay(ctx context.Context, since time.Time, until time.Time, stdout io.Writer, stderr io.Writer) int {
	dataDirectory, err := datadir.Path()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	searchIndex, err := syncedIndex(ctx, false, stderr, "the replay")
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	defer searchIndex.Close()
	transcripts, err := searchIndex.MainTranscripts(ctx, replayStamp(since), replayStamp(until))
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	archiveDirectory := filepath.Join(dataDirectory, "archive", "transcripts")
	seen := map[string]bool{}
	var stops []replayStop
	missing := 0
	for _, listed := range transcripts {
		path := filepath.Join(archiveDirectory, filepath.FromSlash(listed.Source))
		content, readErr := readArchived(path)
		if errors.Is(readErr, fs.ErrNotExist) {
			missing++
			continue
		}
		if readErr != nil {
			fmt.Fprintf(stderr, "ele: %v\n", readErr)
			return 1
		}
		edits, editsErr := subagentEditStamps(path)
		if editsErr != nil {
			fmt.Fprintf(stderr, "ele: %v\n", editsErr)
			return 1
		}
		// Transcripts come oldest first, so a stop copied into a resumed transcript is the original's.
		for _, stop := range replayTranscript(content, edits, seen) {
			seen[stop.key] = true
			at, parseErr := time.Parse(time.RFC3339Nano, stop.Timestamp)
			if parseErr != nil || at.Before(since) || !at.Before(until) {
				continue
			}
			stop.Owner, stop.Project, stop.Session = listed.Owner, listed.Project, listed.Session
			stops = append(stops, stop)
		}
	}
	slices.SortStableFunc(stops, func(a replayStop, b replayStop) int {
		return cmp.Or(strings.Compare(a.Timestamp, b.Timestamp), strings.Compare(a.Session, b.Session))
	})
	encoder := json.NewEncoder(stdout)
	for _, stop := range stops {
		if err = encoder.Encode(stop); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
	}
	if missing > 0 {
		fmt.Fprintf(stderr, "ele: %s in the window not in the archive, so not replayed\n", plural(missing, "transcript", "transcripts"))
	}
	return 0
}

// Keyed by line and position: a later generation of a subagent transcript holds its lines again.
func subagentEditStamps(transcriptPath string) ([]string, error) {
	directory := archive.SubagentDirectory(transcriptPath)
	listing, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var stamps []string
	for _, listed := range listing {
		name := listed.Name()
		if !strings.HasPrefix(name, "agent-") || !strings.HasSuffix(name, ".jsonl.gz") {
			continue
		}
		content, readErr := readArchived(filepath.Join(directory, name))
		if readErr != nil {
			return nil, readErr
		}
		for line := range bytes.SplitSeq(content, []byte("\n")) {
			entries, _, parseErr := transcript.Parse(line)
			if parseErr != nil {
				continue
			}
			for position, entry := range entries {
				key := entry.UUID + "#" + strconv.Itoa(position)
				if subagentWorks(entry) && !seen[key] {
					seen[key] = true
					stamps = append(stamps, entry.Timestamp)
				}
			}
		}
	}
	slices.Sort(stamps)
	return stamps, nil
}

func triggerKind(text string) string {
	switch {
	case strings.HasPrefix(text, "Bash: "):
		return "bash"
	case strings.HasPrefix(text, "AskUserQuestion: "):
		return "question"
	default:
		return "edit"
	}
}

func (r *replayer) block(timestamp string, uuid string, asked bool) {
	if r.recorded {
		r.asked, r.blocked = r.asked || asked, true
		return
	}
	at, _ := time.Parse(time.RFC3339Nano, timestamp)
	if !r.blockStop || r.output || r.prompted || at.Sub(r.lastBlock).Abs() >= blockGroup {
		r.blockStop, r.grouped = true, r.stop(timestamp, uuid, stopPlaced, false)
	}
	r.active, r.lastBlock = true, at
	if r.grouped {
		r.stops[len(r.stops)-1].Asked = r.stops[len(r.stops)-1].Asked || asked
	}
}

// A Stop hook's feedback parses as a prompt; the hook reads it as one, but nobody typed it.
func (r *replayer) entry(entry transcript.Entry, feedback bool) {
	switch {
	case feedback && entry.Kind == transcript.KindPrompt:
	case entry.Kind == transcript.KindPrompt:
		if !r.recorded && r.output {
			r.stop(entry.Timestamp, entry.UUID, stopPlaced, false)
			r.blockStop = false
		}
		r.active, r.prompted = false, true
	default:
		r.output = true
	}
	if entry.Kind == transcript.KindTool {
		probe := r.state
		probe.Pending = false
		if advance(probe, []transcript.Entry{entry}, nil).Pending {
			r.mark(triggerKind(entry.Text))
		}
	}
	r.step(advance(r.state, []transcript.Entry{entry}, nil))
}

func (r *replayer) mark(kind string) {
	if !slices.Contains(r.trigger, kind) {
		r.trigger = append(r.trigger, kind)
	}
}

func (r *replayer) step(next stopState) {
	if next.NotesProposed != r.state.NotesProposed || next.Proposing && !r.state.Proposing || r.state.Pending && !next.Pending {
		r.proposed = true
	}
	if !next.Pending {
		r.trigger = nil
	}
	r.state = next
}

// A stop copied into a resumed transcript ran in the session it was copied from, so the hook did nothing there. The
// hook reads subagent files after the transcript, so an edit counts at the first stop at or after its stamp; at a
// transcript's first judged stop it has no saved state and records the files' sizes unread.
func (r *replayer) stop(timestamp string, uuid string, kind string, asked bool) bool {
	key := cmp.Or(uuid, "@"+timestamp)
	r.output, r.prompted = false, false
	if r.seen[key] {
		return false
	}
	for len(r.edits) > 0 && r.edits[0] <= timestamp {
		if r.judged {
			probe := r.state
			probe.Pending = false
			if subagentEdit(probe, r.edits[0]).Pending {
				r.mark("subagent-edit")
			}
			r.step(subagentEdit(r.state, r.edits[0]))
		}
		r.edits = r.edits[1:]
	}
	stop := replayStop{Asked: asked, Stop: kind, Timestamp: timestamp, key: key}
	var next stopState
	stop.Asks, next = rememberBlock(r.state, r.active)
	if stop.Asks {
		stop.Trigger, r.trigger = r.trigger, nil
	} else {
		stop.WhyNot = r.whyNot()
	}
	r.judged, r.proposed, r.state = true, false, next
	r.stops = append(r.stops, stop)
	return true
}

// A spent stretch holds back only pending work, so it comes last.
func (r *replayer) whyNot() string {
	switch {
	case r.active:
		return "hook-active"
	case r.state.InCheckpoint:
		return "checkpoint"
	case r.state.InReview:
		return "review"
	case !r.state.Pending && r.proposed:
		return "proposal"
	case !r.state.Pending:
		return "nothing-pending"
	default:
		return "stretch"
	}
}
