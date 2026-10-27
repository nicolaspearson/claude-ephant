package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"

	"github.com/nicolaspearson/claude-ephant/internal/config"
	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/highlight"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/redact"
)

// Bytes of the notes block, about 1.5k tokens; open questions may exceed it.
const contextBudget = 6000

const (
	// UTF-16 units, assumed from the docs' "characters". As of 2026-10-23 Claude
	// Code shows the model only a preview of hook output past 10,000; the margin
	// holds lines written outside writeSessionStart.
	blockBudget    = 9500
	hookInputLimit = 1 << 20
	planBodyBudget = 2000
	sessionHint    = "A note's discussion: ele show --note <id>.\n"
	sessionPrefix  = 8
	staleAfter     = 24 * time.Hour
)

func handoffSection(dataDirectory string, pending []notes.Note, now time.Time) string {
	var section strings.Builder
	if len(pending) > 1 {
		fmt.Fprintf(&section, "%d handoffs from ele, newest first, each pending until picked or dropped:\n", len(pending))
		for _, handoff := range pending {
			fmt.Fprintf(&section, "- %s:\n  %s\n", noteSource(handoff), strings.ReplaceAll(strings.TrimSpace(redact.Redact(handoff.Body)), "\n", "\n  "))
		}
		return section.String()
	}
	fmt.Fprintf(&section, "Handoff from ele, %s, shown once:\n%s\n", noteSource(pending[0]), strings.TrimSpace(redact.Redact(pending[0].Body)))
	if err := notes.MarkShown(filepath.Join(dataDirectory, "notes"), pending[0], now); err != nil {
		fmt.Fprintf(&section, "ele: %v; the next startup or /clear shows that note as a handoff\n", err)
	}
	return section.String()
}

func noteSource(note notes.Note) string {
	written := note.Date
	if when, err := time.Parse(time.RFC3339, note.Time); err == nil {
		written = when.UTC().Format("2006-01-02 15:04Z")
	}
	if note.Session != "" {
		written += ", session " + note.Session[:min(len(note.Session), sessionPrefix)]
	}
	return "note " + note.ID + " of " + written
}

func hookSource(input io.Reader) string {
	var hook struct {
		Source string `json:"source"`
	}
	readHookInput(input, &hook)
	return hook.Source
}

func liveNotes(all []notes.Note, project string) (questions []notes.Note, others []notes.Note) {
	superseded := map[string]bool{}
	for _, note := range all {
		for _, target := range note.Supersedes {
			superseded[target] = true
		}
	}
	for _, note := range all {
		switch {
		case superseded[note.ID], note.Scope == notes.ScopeProject && note.Project != project, note.Kind == notes.KindHandoff, note.Kind == notes.KindPlan, note.Kind == notes.KindReminder:
		case note.Kind == notes.KindQuestion:
			questions = append(questions, note)
		default:
			others = append(others, note)
		}
	}
	newestFirst := func(a, b notes.Note) int { return cmp.Or(cmp.Compare(b.Date, a.Date), cmp.Compare(b.ID, a.ID)) }
	slices.SortFunc(questions, newestFirst)
	slices.SortFunc(others, newestFirst)
	return questions, others
}

func noteLine(note notes.Note) string {
	from := ""
	if note.Session != "" {
		from = " (session " + note.Session[:min(len(note.Session), sessionPrefix)] + ")"
	}
	return fmt.Sprintf("- %s %s %s%s: %s\n", note.Date, note.Kind, note.ID, from, redact.Redact(strings.Join(strings.Fields(note.Body), " ")))
}

func planSection(ctx context.Context, all []notes.Note, project string, noteDirectory string, source string) (section string, instruction string) {
	active := activePlans(all, project)
	root, err := checkoutRoot()
	if err != nil {
		return fmt.Sprintf("ele could not find the checkout for %s: %v\n", plural(len(active), "active plan", "active plans"), err), ""
	}
	plans := loadPlanSet(root, all, project, noteDirectory)
	state := func(relative string) (string, bool) {
		view, err := planView(ctx, root, relative, planBodyBudget, plans)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return fmt.Sprintf(activePlanMissing, relative), false
		case err != nil:
			// A parse error quotes the file.
			return redact.Redact(err.Error()), false
		}
		return view, true
	}
	ask := "whether to work a plan, and which, before starting anything; ele plan show <path> prints one"
	if len(active) == 1 {
		view, ok := state(active[0].Body)
		if !ok {
			return "ele: " + view + "\n", ""
		}
		section = "Active plan from ele:\n" + view
		ask = "whether to work this plan before starting anything; /ele:plan, which the user runs, picks it up"
	} else {
		var out strings.Builder
		fmt.Fprintf(&out, "%d active plans from ele, newest first:\n", len(active))
		for _, note := range active {
			fmt.Fprintf(&out, "- %s\n", planSummary(root, note.Body, plans))
		}
		section = out.String()
	}
	if source == "compact" {
		return section, "If this session was mid-task when it compacted, carry on, re-reading the plan item from its file if it was one. Otherwise ask the user " + ask + ".\n"
	}
	return section, "Ask the user " + ask + ".\n"
}

// A terminal is never read: run by hand, a hook command would wait on it.
func readHookInput(input io.Reader, hook any) bool {
	if file, ok := input.(*os.File); ok {
		info, err := file.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice != 0 {
			return false
		}
	}
	return json.NewDecoder(io.LimitReader(input, hookInputLimit)).Decode(hook) == nil
}

func reminderSection(due []notes.Note) string {
	var section strings.Builder
	section.WriteString("Reminders from ele, due:\n")
	for _, reminder := range due {
		fmt.Fprintf(&section, "- %sdue %s, reminder %s%s: %s\n", nudgeMarker(reminder), reminder.Due, reminder.ID, interval(reminder), redact.Redact(strings.Join(strings.Fields(reminder.Body), " ")))
	}
	return section.String()
}

func sessionCount(ctx context.Context, dataDirectory string, owner string, project string) (int, error) {
	searchIndex, err := index.OpenChecked(filepath.Join(dataDirectory, "ele.db"))
	if err != nil {
		return 0, err
	}
	defer searchIndex.Close()
	return searchIndex.SessionCount(ctx, owner, project)
}

func sessionStart(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("session-start", stderr)
	noSync := flags.Bool("no-sync", false, "do not start a background sync")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	now := time.Now()
	source := hookSource(stdin)
	var block strings.Builder
	if source == "compact" {
		fmt.Fprintf(&block, "This session compacted at %s; what came before that is a summary, not the conversation.\n", now.UTC().Format(time.RFC3339))
	}
	code := 0
	shown, err := writeSessionStart(ctx, &block, dataDirectory, now, source)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		code = 1
	}
	if !*noSync {
		if err = startSync(dataDirectory); err != nil {
			fmt.Fprintf(&block, "ele could not start a background sync: %v\n", err)
		}
	}
	if err = writeHookOutput(stdout, block.String(), shown); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		code = 1
	}
	return code
}

func startSync(dataDirectory string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dataDirectory, 0o700); err != nil {
		return err
	}
	logFile, err := os.OpenFile(filepath.Join(dataDirectory, "sync.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()
	command := exec.Command(executable, "sync")
	command.Env = append(os.Environ(), backgroundSyncVariable+"=1")
	command.Stdout, command.Stderr = logFile, logFile
	// Own process group: a signal to the hook's group does not reach the sync.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err = command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}

// Plain stdout reaches only the model; a systemMessage reaches the user.
func writeHookOutput(stdout io.Writer, block string, shown string) error {
	if shown == "" {
		_, err := fmt.Fprint(stdout, block)
		return err
	}
	var output struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
			HookEventName     string `json:"hookEventName"`
		} `json:"hookSpecificOutput"`
		SystemMessage string `json:"systemMessage"`
	}
	output.HookSpecificOutput.AdditionalContext = block
	output.HookSpecificOutput.HookEventName = "SessionStart"
	output.SystemMessage = shown
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(output)
}

func writeNotes(stdout io.Writer, owner string, project string, all []notes.Note, invalid int, sessions int, room int) {
	questions, others := liveNotes(all, project)
	header := fmt.Sprintf("Notes from ele for %s:\n", present(owner, project))
	if len(questions)+len(others) == 0 {
		header = fmt.Sprintf("ele has no notes for %s.\n", present(owner, project))
	}
	reserve := trailerRoom + len(sessionHint)
	written, units := len(header), utf16Units(header)
	var shown []notes.Note
	for _, note := range questions {
		next := noteLine(note)
		if units+utf16Units(next) > room-reserve {
			break
		}
		written, units = written+len(next), units+utf16Units(next)
		shown = append(shown, note)
	}
	shownQuestions := len(shown)
	for _, note := range others {
		next := noteLine(note)
		if shownQuestions < len(questions) || written+len(next) > contextBudget-reserve || units+utf16Units(next) > room-reserve {
			break
		}
		written, units = written+len(next), units+utf16Units(next)
		shown = append(shown, note)
	}

	render := func(count int) string {
		var out strings.Builder
		out.WriteString(header)
		withSession := false
		for _, note := range shown[:count] {
			out.WriteString(noteLine(note))
			withSession = withSession || note.Session != ""
		}
		var closing []string
		if hidden := len(others) - max(count-shownQuestions, 0); hidden > 0 {
			closing = append(closing, plural(hidden, "older note", "older notes")+" not shown")
		}
		if hidden := len(questions) - min(count, shownQuestions); hidden > 0 {
			closing = append(closing, plural(hidden, "older question", "older questions")+" not shown, ele context lists them")
		}
		if invalid > 0 {
			closing = append(closing, plural(invalid, "invalid note", "invalid notes")+" skipped; ele sync names each")
		}
		if sessions > 0 {
			closing = append(closing, plural(sessions, "past session", "past sessions")+" indexed for "+project)
		}
		closing = append(closing, "search with /ele:recall")
		out.WriteString(strings.Join(closing, "; "))
		out.WriteString(".\n")
		if withSession {
			out.WriteString(sessionHint)
		}
		return out.String()
	}
	// The reserve is an estimate; a long project name can outgrow it.
	count := len(shown)
	text := render(count)
	for count > 0 && utf16Units(text) > room {
		count--
		text = render(count)
	}
	fmt.Fprint(stdout, text)
}

func writeSessionStart(ctx context.Context, stdout io.Writer, dataDirectory string, now time.Time, source string) (string, error) {
	written := &unitCounter{writer: stdout}
	stdout = written
	syncLog := filepath.Join(dataDirectory, "sync.log")
	stamp, err := os.ReadFile(filepath.Join(dataDirectory, "last-sync"))
	last, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(string(stamp)))
	switch {
	case err != nil:
		fmt.Fprintf(stdout, "ele has no record of a successful sync; %s holds the background syncs' output, newest last.\n", syncLog)
	case parseErr != nil:
		fmt.Fprintf(stdout, "ele's sync stamp is malformed; %s holds the background syncs' output, newest last.\n", syncLog)
	case now.Sub(last) > staleAfter:
		fmt.Fprintf(stdout, "ele last synced %s; %s holds the background syncs' output, newest last.\n", last.Format(time.RFC3339), syncLog)
	}

	ownerMap, err := ownermap.Load(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		return "", err
	}
	// As of 2026-10-15 Claude Code runs a hook in the session's working directory.
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	project, owner := index.NewProjects(ownerMap).Resolve(directory)
	settings, err := config.Load(configPath(dataDirectory))
	coloured, remindersOn := true, true
	if err != nil {
		fmt.Fprintf(stdout, "ele: %v; every option reads as its default.\n", err)
	} else {
		if changed := offDefaults(settings, owner); len(changed) > 0 {
			fmt.Fprintf(stdout, "Config from ele, off its defaults: %s.\n", strings.Join(changed, ", "))
		}
		coloured = colourOn(settings, owner)
		if value, valueErr := settings.Value("reminders", owner); valueErr == nil {
			remindersOn = value.Value == true
		}
	}
	all, invalid, err := notes.ReadFor(filepath.Join(dataDirectory, "notes"), owner)
	if err != nil {
		return "", err
	}
	switch {
	case owner == "" && len(all)+invalid == 0:
		fmt.Fprintf(stdout, "ele shows no notes here: no owner in owner-map.json covers %s. /ele:recall needs --owner <name> or --all.\n", directory)
		return "", nil
	case owner == "":
		fmt.Fprintf(stdout, "ele shows only everywhere notes here: no owner in owner-map.json covers %s. /ele:recall needs --owner <name> or --all.\n", directory)
	}
	var shown []string
	// A resumed or compacted session keeps its context; the handoff is for a fresh one.
	if pending := pendingHandoffs(all, project); len(pending) > 0 && (source == "clear" || source == "startup") {
		handoff := handoffSection(dataDirectory, pending, now)
		shown = append(shown, handoff)
		fmt.Fprint(stdout, handoff)
		if len(pending) > 1 {
			fmt.Fprintln(stdout, "Ask the user which handoff to pick up and whether to drop any. Run ele handoff close <id>... for each picked or dropped; the rest return at the next startup or /clear.")
		}
		fmt.Fprintln(stdout)
	}
	// A plan is not used up when shown; compaction summarises the item being worked.
	if active := activePlans(all, project); len(active) > 0 && (source == "clear" || source == "compact" || source == "startup") {
		section, instruction := planSection(ctx, all, project, filepath.Join(dataDirectory, "notes"), source)
		shown = append(shown, section)
		fmt.Fprint(stdout, section+instruction+"\n")
	}
	if due := dueReminders(all, project, now); remindersOn && len(due) > 0 {
		section := reminderSection(due)
		shown = append(shown, section)
		instruction := "Tell the user each is due. Once the user says one is done or dropped, run ele reminder close <id>...; until then it shows at every startup, /clear and compaction.\n"
		if slices.ContainsFunc(due, func(note notes.Note) bool { return note.Every > 0 }) {
			instruction += "Closing one shown with an interval re-arms it for that many days; ele forget <id> drops it for good.\n"
		}
		fmt.Fprint(stdout, section+instruction+"\n")
	}
	// The count is decoration; a busy or broken index must not cost the session its notes.
	var sessions int
	if owner != "" {
		if sessions, err = sessionCount(ctx, dataDirectory, owner, project); err != nil {
			fmt.Fprintf(stdout, "ele could not count past sessions: %v\n", err)
		}
	}
	writeNotes(stdout, owner, project, all, invalid, sessions, blockBudget-written.units)
	return highlight.Message(strings.TrimSuffix(strings.Join(shown, "\n"), "\n"), coloured), nil
}

type unitCounter struct {
	units  int
	writer io.Writer
}

func (counter *unitCounter) Write(p []byte) (int, error) {
	counter.units += utf16Units(string(p))
	return counter.writer.Write(p)
}

func utf16Units(text string) int {
	units := 0
	for _, character := range text {
		units += utf16.RuneLen(character)
	}
	return units
}
