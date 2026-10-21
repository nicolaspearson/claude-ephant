package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/plan"
	"github.com/nicolaspearson/claude-ephant/internal/profile"
	"github.com/nicolaspearson/claude-ephant/internal/redact"
)

const activePlanMissing = "plan %[1]s is not in this checkout; switch to its branch, or ele plan done %[1]s closes it"

func planCommand(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	subcommands := map[string]func(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int{
		"check": checkPlan,
		"done":  planDone,
		"list":  listPlans,
		"show":  showPlan,
		"start": startPlan,
	}
	if len(args) == 0 || subcommands[args[0]] == nil {
		return misuse("plan", args, stdout, stderr)
	}
	return subcommands[args[0]](ctx, args[1:], stdout, stderr)
}

func activePlans(all []notes.Note, project string) []notes.Note {
	superseded := supersededBy(all)
	return slices.DeleteFunc(newestOfKind(all, notes.KindPlan, project), func(note notes.Note) bool {
		return note.Shown != "" || superseded[note.ID] != ""
	})
}

func checkPlan(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("plan check", stderr)
	arguments, err := parseInterspersed(flags, args)
	if err != nil || len(arguments) > 1 {
		return rejectCall(flags, err, stdout, stderr)
	}
	root, err := checkoutRoot()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	var plans planSet
	var paths []string
	if len(arguments) == 1 {
		relative, pathErr := planPath(root, arguments[0])
		if pathErr != nil {
			fmt.Fprintf(stderr, "ele: %v\n", pathErr)
			return 1
		}
		if scope, _, scopeErr := projectNotes(ctx, "", ""); scopeErr == nil {
			plans = loadPlanSet(root, scope.all, scope.project, scope.noteDirectory)
		}
		paths = []string{relative}
	} else {
		scope, code, scopeErr := projectNotes(ctx, "", "")
		if scopeErr != nil {
			fmt.Fprintf(stderr, "ele: %v\n", scopeErr)
			return code
		}
		active := activePlans(scope.all, scope.project)
		if len(active) == 0 {
			fmt.Fprintf(stdout, "ele has no active plan for %s; ele plan check <path> checks one.\n", present(scope.owner, scope.project))
			return 0
		}
		plans = loadPlanSet(root, scope.all, scope.project, scope.noteDirectory)
		for _, note := range active {
			paths = append(paths, note.Body)
		}
	}
	code := 0
	for _, relative := range paths {
		content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		switch {
		case errors.Is(readErr, fs.ErrNotExist) && len(arguments) == 0:
			fmt.Fprintf(stdout, activePlanMissing+".\n", relative)
			continue
		case errors.Is(readErr, fs.ErrNotExist):
			fmt.Fprintf(stderr, "ele: %s is not in this checkout\n", relative)
			return 1
		case readErr != nil:
			fmt.Fprintf(stderr, "ele: %v\n", readErr)
			return 1
		}
		parsed, parseErr := plan.Parse(content)
		if parseErr != nil {
			fmt.Fprintf(stderr, "%s\n", redact.Redact(fmt.Sprintf("ele: %s: %v", relative, parseErr)))
			code = 1
			continue
		}
		_, warnings := plans.check(relative, parsed)
		if len(warnings) == 0 {
			fmt.Fprintf(stdout, "Plan %s: no warnings.\n", relative)
			continue
		}
		var report strings.Builder
		fmt.Fprintf(&report, "Plan %s: %s.\n", relative, plural(len(warnings), "warning", "warnings"))
		for _, warning := range warnings {
			if warning.Line > 0 {
				fmt.Fprintf(&report, "- line %d: %s\n", warning.Line, warning.Text)
			} else {
				fmt.Fprintf(&report, "- %s\n", warning.Text)
			}
		}
		fmt.Fprint(stdout, redact.Redact(report.String()))
	}
	return code
}

func checkoutRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(index.GitRoot(directory))
}

func listPlans(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("plan list", stderr)
	listAll := flags.Bool("all", false, "add done and superseded plans")
	owner := flags.String("owner", "", "list this owner's plans instead of the resolved one's")
	projectName := flags.String("project", "", "list this project's plans instead of the working directory's")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	scope, code, err := projectNotes(ctx, *owner, *projectName)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return code
	}
	superseded := supersededBy(scope.all)
	var lines []string
	for _, note := range newestOfKind(scope.all, notes.KindPlan, scope.project) {
		var state string
		switch {
		case superseded[note.ID] != "":
			state = "superseded by " + superseded[note.ID]
		case note.Shown != "":
			state = "done " + note.Shown
		default:
			state = "active"
		}
		if state != "active" && !*listAll {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s  %s: %s\n", state, noteSource(note), redact.Redact(note.Body)))
	}
	where := present(scope.owner, scope.project)
	switch {
	case len(lines) > 0:
		fmt.Fprintf(stdout, "Plans for %s, newest first:\n%sele plan show <path> prints one; ele plan done <path|id> closes one.\n", where, strings.Join(lines, ""))
	case *listAll:
		fmt.Fprintf(stdout, "ele has no plans for %s.\n", where)
	default:
		fmt.Fprintf(stdout, "ele has no active plans for %s; --all adds done and superseded ones.\n", where)
	}
	return 0
}

func planDone(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("plan done", stderr)
	owner := flags.String("owner", "", "close this owner's plans instead of the resolved one's")
	projectName := flags.String("project", "", "close this project's plans instead of the working directory's")
	arguments, err := parseInterspersed(flags, args)
	if err != nil {
		return rejectCall(flags, err, stdout, stderr)
	}
	if len(arguments) == 0 {
		return rejectCall(flags, nil, stdout, stderr)
	}
	scope, code, err := projectNotes(ctx, *owner, *projectName)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return code
	}
	root := ""
	if *projectName == "" {
		root, _ = checkoutRoot()
	}
	active := activePlans(scope.all, scope.project)
	var chosen []notes.Note
	// Checked before any is closed, so a mistyped argument closes nothing.
	for _, argument := range arguments {
		relative := ""
		if root != "" {
			relative, _ = planPath(root, argument)
		}
		// The resolved path before the raw body, which from a subdirectory can name another plan.
		at := slices.IndexFunc(active, func(note notes.Note) bool { return note.ID == argument || note.Body == relative })
		if at < 0 {
			at = slices.IndexFunc(active, func(note notes.Note) bool { return note.Body == argument })
		}
		if at < 0 {
			fmt.Fprintf(stderr, "ele: %s is not an active plan of %s; ele plan list lists them, --project <name> for another project's\n", argument, present(scope.owner, scope.project))
			return 1
		}
		if !slices.ContainsFunc(chosen, func(note notes.Note) bool { return note.ID == active[at].ID }) {
			chosen = append(chosen, active[at])
		}
	}
	now := time.Now()
	for _, note := range chosen {
		unticked, storeErr := storePlanItems(ctx, root, scope.noteDirectory, note)
		switch {
		case storeErr != nil:
			fmt.Fprintf(stdout, "Item names of %s not stored, so an edge into it reads as missing and its names stay free: %v\n", note.Body, redact.Redact(storeErr.Error()))
		case unticked != nil:
			verb := "are"
			if len(unticked) == 1 {
				verb = "is"
			}
			fmt.Fprintf(stdout, "%s is gone, so its names were stored from its last commit, where %s %s unticked; run ele plan done before deleting a plan.\n", note.Body, strings.Join(unticked, ", "), verb)
		}
		if err = notes.MarkShown(scope.noteDirectory, note, now); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Done with plan %s (%s).\n", note.Body, note.ID)
	}
	return 0
}

// A plan already deleted falls back to its last commit, which may lack ticks made since, so its unticked names are returned for a warning.
func storePlanItems(ctx context.Context, root string, noteDirectory string, note notes.Note) ([]string, error) {
	if root == "" {
		return nil, errors.New("its checkout is not the working directory's")
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(note.Body)))
	committed := errors.Is(err, fs.ErrNotExist)
	if committed {
		content, err = plan.Committed(ctx, root, note.Body)
	}
	if err != nil {
		return nil, err
	}
	parsed, err := plan.Parse(content)
	if err != nil {
		return nil, err
	}
	if err = plan.WriteStored(planStore(noteDirectory), note.ID, note.Body, parsed); err != nil || !committed {
		return nil, err
	}
	var unticked []string
	for _, action := range parsed.Actions {
		if action.Name != "" && !action.Done {
			unticked = append(unticked, action.Name)
		}
	}
	return unticked, nil
}

func planPath(root string, argument string) (string, error) {
	absolute, err := filepath.Abs(argument)
	if err != nil {
		return "", err
	}
	// The nearest existing parent, since a plan on another branch is missing from the checkout.
	for missing := ""; ; {
		if resolved, resolveErr := filepath.EvalSymlinks(absolute); resolveErr == nil {
			absolute = filepath.Join(resolved, missing)
			break
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			absolute = filepath.Join(absolute, missing)
			break
		}
		missing = filepath.Join(filepath.Base(absolute), missing)
		absolute = parent
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil || !filepath.IsLocal(relative) {
		return "", fmt.Errorf("%s is outside the project at %s", argument, root)
	}
	return filepath.ToSlash(relative), nil
}

// bodyLimit is in bytes; 0 means no cap.
func planView(ctx context.Context, root string, relative string, bodyLimit int, plans planSet) (string, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	parsed, err := plan.Parse(content)
	if err != nil {
		return "", fmt.Errorf("%s: %w", relative, err)
	}
	tick, ticked, err := plan.Previous(ctx, root, relative)
	if err != nil {
		return "", err
	}
	resolved, warnings := plans.check(relative, parsed)
	var view strings.Builder
	writePlanView(&view, relative, resolved, len(warnings), tick, ticked, bodyLimit)
	for _, clash := range plans.clashes(relative, parsed) {
		fmt.Fprintf(&view, "Warning: %s; %s.\n", clash, namesUnique)
	}
	return redact.Redact(view.String()), nil
}

func planSummary(root string, relative string, plans planSet) string {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Sprintf(activePlanMissing, relative)
	}
	if err != nil {
		return redact.Redact(err.Error())
	}
	parsed, err := plan.Parse(content)
	if err != nil {
		return redact.Redact(fmt.Sprintf("%s: %v", relative, err))
	}
	resolved, warnings := plans.check(relative, parsed)
	counts := resolved.Counts()
	summary := fmt.Sprintf("%s: %d open, %d blocked, %d done, %d in the inbox.", relative, counts.Open, counts.Blocked, counts.Done, counts.Inbox)
	if len(warnings) > 0 {
		summary += fmt.Sprintf(" %s; ele plan check lists %s.", plural(len(warnings), "warning", "warnings"), pronoun(len(warnings)))
	}
	if next, ok := resolved.Next(); ok {
		summary += " Next: " + next.Title
	}
	if clashes := plans.clashes(relative, parsed); len(clashes) > 0 {
		summary += fmt.Sprintf(" Warning: %s; %s.", strings.Join(clashes, ", "), namesUnique)
	}
	return redact.Redact(summary)
}

func pronoun(count int) string {
	if count == 1 {
		return "it"
	}
	return "them"
}

func showPlan(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("plan show", stderr)
	arguments, err := parseInterspersed(flags, args)
	if err != nil || len(arguments) > 1 {
		return rejectCall(flags, err, stdout, stderr)
	}
	root, err := checkoutRoot()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	var plans planSet
	var relative string
	missing := "ele: %s is not in this checkout\n"
	if len(arguments) == 1 {
		if relative, err = planPath(root, arguments[0]); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		// Without the project's notes a named plan still shows, unchecked for clashes, and every cross-plan blocker reads as missing.
		if scope, _, scopeErr := projectNotes(ctx, "", ""); scopeErr == nil {
			plans = loadPlanSet(root, scope.all, scope.project, scope.noteDirectory)
		}
	} else {
		scope, code, scopeErr := projectNotes(ctx, "", "")
		if scopeErr != nil {
			fmt.Fprintf(stderr, "ele: %v\n", scopeErr)
			return code
		}
		missing = "ele: " + activePlanMissing + "\n"
		active := activePlans(scope.all, scope.project)
		plans = loadPlanSet(root, scope.all, scope.project, scope.noteDirectory)
		where := present(scope.owner, scope.project)
		switch len(active) {
		case 0:
			fmt.Fprintf(stdout, "ele has no active plan for %s; ele plan start <path> starts one.\n", where)
			return 0
		case 1:
			relative = active[0].Body
		default:
			fmt.Fprintf(stdout, "Active plans for %s, newest first; ele plan show <path> prints one:\n", where)
			for _, note := range active {
				fmt.Fprintf(stdout, "- %s: %s\n", note.ID, planSummary(root, note.Body, plans))
			}
			return 0
		}
	}
	view, err := planView(ctx, root, relative, 0, plans)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(stderr, missing, relative)
		return 1
	}
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	fmt.Fprint(stdout, view)
	return 0
}

func startPlan(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("plan start", stderr)
	arguments, err := parseInterspersed(flags, args)
	if err != nil {
		return rejectCall(flags, err, stdout, stderr)
	}
	if len(arguments) != 1 {
		return rejectCall(flags, nil, stdout, stderr)
	}
	scope, code, err := projectNotes(ctx, "", "")
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return code
	}
	root, err := checkoutRoot()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	relative, err := planPath(root, arguments[0])
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(stderr, "ele: %s does not exist; name a plan file inside the project\n", relative)
		return 1
	}
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	parsed, err := plan.Parse(content)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %s: %v\n", relative, err)
		return 1
	}
	if len(parsed.Actions) == 0 {
		fmt.Fprintf(stderr, "ele: %s has no checkbox under ## Actions\n", relative)
		return 1
	}
	active := activePlans(scope.all, scope.project)
	if at := slices.IndexFunc(active, func(note notes.Note) bool { return note.Body == relative }); at >= 0 {
		fmt.Fprintf(stderr, "ele: %s is already started; ele plan show prints it\n", relative)
		return 1
	}
	if clashes := loadPlanSet(root, scope.all, scope.project, scope.noteDirectory).clashes(relative, parsed); len(clashes) > 0 {
		fmt.Fprintf(stderr, "ele: %s: %s; %s\n", relative, strings.Join(clashes, ", "), namesUnique)
		return 1
	}
	now := time.Now()
	id, err := notes.NewID(now)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	note := notes.Note{
		Body:    relative,
		Date:    now.Format(time.DateOnly),
		ID:      id,
		Kind:    notes.KindPlan,
		Owner:   scope.owner,
		Profile: profile.Current(os.Getenv("CLAUDE_CONFIG_DIR")),
		Project: scope.project,
		Scope:   notes.ScopeProject,
		Session: os.Getenv("CLAUDE_CODE_SESSION_ID"),
		Time:    now.UTC().Format(time.RFC3339),
	}
	if _, err = notes.Write(scope.noteDirectory, note); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Started plan %s as %s; ele plan show prints its next item.\n", relative, id)
	return 0
}

func writePlanView(out *strings.Builder, relative string, parsed plan.Plan, warnings int, tick plan.Tick, ticked bool, bodyLimit int) {
	counts := parsed.Counts()
	fmt.Fprintf(out, "Plan %s: %d open, %d blocked, %d done, %d in the inbox.\n", relative, counts.Open, counts.Blocked, counts.Done, counts.Inbox)
	if warnings > 0 {
		fmt.Fprintf(out, "%s; ele plan check %s lists %s.\n", plural(warnings, "warning", "warnings"), relative, pronoun(warnings))
	}
	switch {
	case ticked && tick.Commit == "":
		fmt.Fprintf(out, "Previous: %s (uncommitted)\n", tick.Title)
	case ticked:
		fmt.Fprintf(out, "Previous: %s (%s, %s)\n", tick.Title, tick.Commit, tick.Date)
	}
	if last := len(parsed.Actions) - 1; last >= 0 && parsed.Actions[last].Done && counts.Inbox > 0 {
		fmt.Fprintf(out, "The closing action is ticked with %s; triage the inbox first.\n", plural(counts.Inbox, "item in the inbox", "items in the inbox"))
	}
	if counts.Blocked > 0 {
		out.WriteString("Blocked:\n")
		for _, action := range parsed.Actions {
			if action.Done || !action.Held {
				continue
			}
			fmt.Fprintf(out, "- %s\n", action.Title)
			for _, reason := range action.Blocked {
				fmt.Fprintf(out, "  blocked: %s\n", reason)
			}
			if !action.After.IsZero() {
				fmt.Fprintf(out, "  after: %s\n", action.After.Local().Format("2006-01-02 15:04 -07:00"))
			}
			for _, flag := range action.Flags {
				fmt.Fprintf(out, "  flag: %s\n", flag)
			}
		}
	}
	if flagged := slices.DeleteFunc(slices.Clone(parsed.Actions), func(action plan.Item) bool { return action.Held || len(action.Flags) == 0 }); len(flagged) > 0 {
		out.WriteString("Flagged:\n")
		for _, action := range flagged {
			fmt.Fprintf(out, "- %s\n", action.Title)
			for _, flag := range action.Flags {
				fmt.Fprintf(out, "  flag: %s\n", flag)
			}
		}
	}
	next, ok := parsed.Next()
	switch {
	case ok:
		fmt.Fprintf(out, "Next, line %d:\n- [ ] %s\n", next.Line, next.Title)
		written := 0
		for at, line := range next.Body {
			if written += len(line) + 1; bodyLimit > 0 && written > bodyLimit {
				fmt.Fprintf(out, "  [%s of the body not shown; ele plan show prints it whole]\n", plural(len(next.Body)-at, "line", "lines"))
				break
			}
			out.WriteString(line)
			out.WriteString("\n")
		}
	case counts.Blocked > 0:
		out.WriteString("Next: none; every open item is blocked.\n")
	default:
		out.WriteString("Next: none; every action is done.\n")
	}
}
