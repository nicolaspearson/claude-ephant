package main

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/redact"
)

type noteChange struct {
	apply        func(directory string, note notes.Note) error
	confirmation string
	detail       func(note notes.Note) string
	eligible     func(all []notes.Note, project string) []notes.Note
	lister       string
	load         func(ctx context.Context, owner string, projectName string) (projectScope, int, error)
	wanted       string
}

func handoffCommand(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	now := time.Now()
	changes := map[string]noteChange{
		"close": {
			apply:        func(directory string, note notes.Note) error { return notes.MarkShown(directory, note, now) },
			confirmation: "Closed",
			eligible:     pendingHandoffs,
			lister:       "ele context",
			load:         projectNotes,
			wanted:       "a pending handoff",
		},
		"reopen": {
			apply:        notes.ClearShown,
			confirmation: "Reopened",
			eligible:     closedHandoffs,
			lister:       "ele handoff list --all",
			load:         projectNotes,
			wanted:       "a closed handoff",
		},
	}
	if len(args) == 0 {
		return misuse("handoff", args, stdout, stderr)
	}
	if args[0] == "list" {
		return listHandoffs(ctx, args[1:], stdout, stderr)
	}
	change, ok := changes[args[0]]
	if !ok {
		return misuse("handoff", args, stdout, stderr)
	}
	return applyChange(ctx, "handoff", args, change, stdout, stderr)
}

func applyChange(ctx context.Context, group string, args []string, change noteChange, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet(group+" "+args[0], stderr)
	owner := flags.String("owner", "", args[0]+" this owner's "+group+"s instead of the resolved one's")
	projectName := flags.String("project", "", args[0]+" this project's "+group+"s instead of the working directory's")
	identifiers, err := parseInterspersed(flags, args[1:])
	if err != nil {
		return rejectCall(flags, err, stdout, stderr)
	}
	if len(identifiers) == 0 {
		return rejectCall(flags, nil, stdout, stderr)
	}
	scope, code, err := change.load(ctx, *owner, *projectName)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return code
	}
	candidates := map[string]notes.Note{}
	for _, note := range change.eligible(scope.all, scope.project) {
		candidates[note.ID] = note
	}
	// Checked before any is changed, so a mistyped id changes nothing.
	for _, identifier := range identifiers {
		if _, ok := candidates[identifier]; !ok {
			fmt.Fprintf(stderr, "ele: %s is not %s of %s. %s lists them, and --project <name> looks in another project.\n", identifier, change.wanted, present(scope.owner, scope.project), change.lister)
			return 1
		}
	}
	changed := map[string]bool{}
	for _, identifier := range identifiers {
		if changed[identifier] {
			continue
		}
		changed[identifier] = true
		if err = change.apply(scope.noteDirectory, candidates[identifier]); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		var detail string
		if change.detail != nil {
			detail = change.detail(candidates[identifier])
		}
		fmt.Fprintf(stdout, "%s %s %s%s.\n", change.confirmation, group, identifier, detail)
	}
	return 0
}

func closedHandoffs(all []notes.Note, project string) []notes.Note {
	superseded := supersededBy(all)
	return slices.DeleteFunc(newestOfKind(all, notes.KindHandoff, project), func(note notes.Note) bool {
		return note.Shown == "" || superseded[note.ID] != ""
	})
}

func listHandoffs(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("handoff list", stderr)
	listAll := flags.Bool("all", false, "add closed and superseded handoffs")
	owner := flags.String("owner", "", "list this owner's handoffs instead of the resolved one's")
	projectName := flags.String("project", "", "list this project's handoffs instead of the working directory's")
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
	for _, note := range newestOfKind(scope.all, notes.KindHandoff, scope.project) {
		var state string
		switch {
		case superseded[note.ID] != "":
			state = "superseded by " + superseded[note.ID]
		case note.Shown != "":
			state = "closed " + note.Shown
		default:
			state = "pending"
		}
		if state != "pending" && !*listAll {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s  %s: %s\n", state, noteSource(note), strings.Join(strings.Fields(redact.Redact(note.Body)), " ")))
	}
	where := present(scope.owner, scope.project)
	switch {
	case len(lines) > 0:
		fmt.Fprintf(stdout, "Handoffs for %s, newest first:\n%sele handoff close <id>... or ele handoff reopen <id>... to change one.\n", where, strings.Join(lines, ""))
	case *listAll:
		fmt.Fprintf(stdout, "ele has no handoffs for %s.\n", where)
	default:
		fmt.Fprintf(stdout, "ele has no pending handoffs for %s; --all adds closed and superseded ones.\n", where)
	}
	return 0
}

func pendingHandoffs(all []notes.Note, project string) []notes.Note {
	superseded := supersededBy(all)
	return slices.DeleteFunc(newestOfKind(all, notes.KindHandoff, project), func(note notes.Note) bool {
		return note.Shown != "" || superseded[note.ID] != ""
	})
}
