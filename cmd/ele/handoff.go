package main

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/redact"
)

type noteChange struct {
	apply        func(directory string, note notes.Note) error
	confirmation string
	detail       func(note notes.Note) string
	eligible     func(all []notes.Note, project string) []notes.Note
	refusal      string
}

type projectScope struct {
	all           []notes.Note
	dataDirectory string
	noteDirectory string
	owner         string
	project       string
}

func handoffCommand(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	now := time.Now()
	changes := map[string]noteChange{
		"close": {
			apply:        func(directory string, note notes.Note) error { return notes.MarkShown(directory, note, now) },
			confirmation: "Closed",
			eligible:     pendingHandoffs,
			refusal:      "a pending handoff of %s; ele context lists them",
		},
		"reopen": {
			apply:        notes.ClearShown,
			confirmation: "Reopened",
			eligible:     closedHandoffs,
			refusal:      "a closed handoff of %s; ele handoff list --all lists them",
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
	scope, code, err := projectNotes(ctx, *owner, *projectName)
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
			fmt.Fprintf(stderr, "ele: %s is not "+change.refusal+", --project <name> for another project's\n", identifier, present(scope.owner, scope.project))
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

func projectNotes(ctx context.Context, owner string, projectName string) (projectScope, int, error) {
	dataDirectory, err := datadir.Path()
	if err != nil {
		return projectScope{}, 1, err
	}
	ownerMap, err := ownermap.Load(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		return projectScope{}, 1, err
	}
	project, resolved, code, err := resolveOwner(ctx, dataDirectory, ownerMap, projectName, owner)
	if err != nil {
		return projectScope{}, code, err
	}
	noteDirectory := filepath.Join(dataDirectory, "notes")
	all, _, err := notes.ReadOwner(noteDirectory, resolved)
	if err != nil {
		return projectScope{}, 1, err
	}
	return projectScope{all: all, dataDirectory: dataDirectory, noteDirectory: noteDirectory, owner: resolved, project: project}, 0, nil
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

// An id's suffix is random, so the time written orders a day's notes.
func newestOfKind(all []notes.Note, kind string, project string) []notes.Note {
	var matched []notes.Note
	for _, note := range all {
		if note.Kind == kind && note.Project == project {
			matched = append(matched, note)
		}
	}
	slices.SortFunc(matched, func(a, b notes.Note) int {
		return cmp.Or(cmp.Compare(b.Date, a.Date), cmp.Compare(b.Time, a.Time), cmp.Compare(b.ID, a.ID))
	})
	return matched
}

func supersededBy(all []notes.Note) map[string]string {
	superseded := map[string]string{}
	for _, note := range all {
		for _, target := range note.Supersedes {
			superseded[target] = note.ID
		}
	}
	return superseded
}
