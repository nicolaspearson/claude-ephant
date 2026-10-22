package main

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/redact"
)

func closedReminders(all []notes.Note, project string) []notes.Note {
	superseded := supersededBy(all)
	return slices.DeleteFunc(remindersHere(all, project), func(note notes.Note) bool {
		return note.Closed == "" || superseded[note.ID] != ""
	})
}

func dueReminders(all []notes.Note, project string, now time.Time) []notes.Note {
	today := now.Format(time.DateOnly)
	return slices.DeleteFunc(withNextDue(openReminders(all, project), now), func(note notes.Note) bool { return note.Due > today })
}

func interval(note notes.Note) string {
	if note.Every == 0 {
		return ""
	}
	return fmt.Sprintf(", every %d days", note.Every)
}

func listReminders(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("reminder list", stderr)
	listAll := flags.Bool("all", false, "add closed and superseded reminders")
	owner := flags.String("owner", "", "list this owner's reminders instead of the resolved one's")
	projectName := flags.String("project", "", "list this project's reminders instead of the working directory's")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	scope, code, err := projectNotes(ctx, *owner, *projectName)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return code
	}
	superseded := supersededBy(scope.all)
	now := time.Now()
	today := now.Format(time.DateOnly)
	var lines []string
	nudges := false
	for _, note := range withNextDue(remindersHere(scope.all, scope.project), now) {
		state, open := "due", true
		switch {
		case superseded[note.ID] != "":
			state, open = "superseded by "+superseded[note.ID], false
		case note.Closed != "":
			state, open = "closed "+note.Closed, false
		case note.Due > today:
			state = "open"
		}
		if !open && !*listAll {
			continue
		}
		nudges = nudges || note.Every > 0
		lines = append(lines, fmt.Sprintf("- %s  due %s, reminder %s (%s%s): %s\n", state, note.Due, note.ID, note.Scope, interval(note), strings.Join(strings.Fields(redact.Redact(note.Body)), " ")))
	}
	where := present(scope.owner, scope.project)
	switch {
	case len(lines) > 0:
		trailer := "ele reminder close <id>... or ele reminder reopen <id>... to change one.\n"
		if nudges {
			trailer = "ele reminder close <id>... or ele reminder reopen <id>... to change one; closing one with an interval re-arms it, ele forget <id> drops it.\n"
		}
		fmt.Fprintf(stdout, "Reminders for %s, soonest due first:\n%s%s", where, strings.Join(lines, ""), trailer)
	case *listAll:
		fmt.Fprintf(stdout, "ele has no reminders for %s.\n", where)
	default:
		fmt.Fprintf(stdout, "ele has no open reminders for %s; --all adds closed and superseded ones.\n", where)
	}
	return 0
}

func nextDue(note notes.Note, now time.Time) string {
	if note.DoneFile == "" {
		return note.Due
	}
	info, err := os.Stat(note.DoneFile)
	if err != nil {
		// Unreadable done file: due at once.
		return min(note.Due, now.Format(time.DateOnly))
	}
	return max(note.Due, info.ModTime().In(now.Location()).AddDate(0, 0, note.Every).Format(time.DateOnly))
}

func openReminders(all []notes.Note, project string) []notes.Note {
	superseded := supersededBy(all)
	return slices.DeleteFunc(remindersHere(all, project), func(note notes.Note) bool {
		return note.Closed != "" || superseded[note.ID] != ""
	})
}

func rearmed(note notes.Note, now time.Time) string {
	return now.AddDate(0, 0, note.Every).Format(time.DateOnly)
}

func reminderCommand(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	now := time.Now()
	changes := map[string]noteChange{
		"close": {
			apply: func(directory string, note notes.Note) error {
				if note.Every > 0 {
					return notes.SetDue(directory, note, rearmed(note, now))
				}
				return notes.MarkClosed(directory, note, now)
			},
			confirmation: "Closed",
			detail: func(note notes.Note) string {
				if note.Every == 0 {
					return ""
				}
				return "; due again " + rearmed(note, now)
			},
			eligible: openReminders,
			refusal:  "an open reminder of %s; ele reminder list lists them",
		},
		"reopen": {
			apply:        notes.ClearClosed,
			confirmation: "Reopened",
			eligible:     closedReminders,
			refusal:      "a closed reminder of %s; ele reminder list --all lists them",
		},
	}
	if len(args) == 0 {
		return misuse("reminder", args, stdout, stderr)
	}
	if args[0] == "list" {
		return listReminders(ctx, args[1:], stdout, stderr)
	}
	change, ok := changes[args[0]]
	if !ok {
		return misuse("reminder", args, stdout, stderr)
	}
	return applyChange(ctx, "reminder", args, change, stdout, stderr)
}

func remindersHere(all []notes.Note, project string) []notes.Note {
	var matched []notes.Note
	for _, note := range all {
		if note.Kind == notes.KindReminder && (note.Scope == notes.ScopeGlobal || note.Project == project) {
			matched = append(matched, note)
		}
	}
	slices.SortFunc(matched, func(a, b notes.Note) int {
		return cmp.Or(cmp.Compare(a.Due, b.Due), cmp.Compare(a.Time, b.Time), cmp.Compare(a.ID, b.ID))
	})
	return matched
}

// withNextDue replaces each nudge's Due with its next due date and sorts by it.
func withNextDue(reminders []notes.Note, now time.Time) []notes.Note {
	for at := range reminders {
		reminders[at].Due = nextDue(reminders[at], now)
	}
	slices.SortStableFunc(reminders, func(a, b notes.Note) int { return cmp.Compare(a.Due, b.Due) })
	return reminders
}
