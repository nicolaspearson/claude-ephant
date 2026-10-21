package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

func forget(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("forget", stderr)
	owner := flags.String("owner", "", "this owner's notes instead of the resolved one's")
	restore := flags.Bool("restore", false, "bring forgotten notes back")
	identifiers, err := parseInterspersed(flags, args)
	if err != nil {
		return rejectCall(flags, err, stdout, stderr)
	}
	if len(identifiers) == 0 {
		return rejectCall(flags, nil, stdout, stderr)
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
	_, resolved, code, err := resolveOwner(ctx, dataDirectory, ownerMap, "", *owner)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return code
	}
	noteDirectory := filepath.Join(dataDirectory, "notes")
	forgottenDirectory := filepath.Join(dataDirectory, "forgotten")
	from, to, verb := noteDirectory, forgottenDirectory, "Forgot"
	if *restore {
		from, to, verb = forgottenDirectory, noteDirectory, "Restored"
	}
	candidates, _, err := notes.ReadOwner(from, resolved)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	live, _, err := notes.ReadOwner(noteDirectory, resolved)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	byID := map[string]notes.Note{}
	for _, note := range candidates {
		byID[note.ID] = note
	}
	superseded := supersededBy(live)
	// Grows as each restore is checked, so two forgotten notes saying the same are not both restored.
	restoring := slices.Clone(live)
	// Checked before any is moved, so a mistyped id or a clash moves nothing.
	for _, identifier := range identifiers {
		if _, ok := byID[identifier]; !ok {
			where := "live"
			if *restore {
				where = "forgotten"
			}
			fmt.Fprintf(stderr, "ele: %s has no %s note %s; --owner <name> for another owner's\n", resolved, where, identifier)
			return 1
		}
		if by := superseded[identifier]; !*restore && by != "" {
			fmt.Fprintf(stderr, "ele: note %s supersedes %s; forget %s first, or leave both\n", by, identifier, by)
			return 1
		}
		if *restore {
			if duplicate := sayingTheSame(restoring, byID[identifier]); duplicate != "" {
				fmt.Fprintf(stderr, "ele: note %s already says what %s says; nothing restored\n", duplicate, identifier)
				return 1
			}
			restoring = append(restoring, byID[identifier])
		}
		if _, err = os.Lstat(filepath.Join(to, resolved, identifier+".md")); err == nil {
			fmt.Fprintf(stderr, "ele: note %s is in both %s and %s; nothing moved\n", identifier, noteDirectory, forgottenDirectory)
			return 1
		}
	}
	moved := map[string]bool{}
	var relocated []notes.Note
	for _, identifier := range identifiers {
		if moved[identifier] {
			continue
		}
		moved[identifier] = true
		if err = notes.Relocate(from, to, byID[identifier]); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		relocated = append(relocated, byID[identifier])
		fmt.Fprintf(stdout, "%s note %s.\n", verb, identifier)
	}
	after := slices.DeleteFunc(slices.Clone(live), func(note notes.Note) bool { return moved[note.ID] })
	if *restore {
		after = append(slices.Clone(live), relocated...)
	}
	before, remaining := supersessorCounts(live), supersessorCounts(after)
	reported := map[string]bool{}
	for _, identifier := range identifiers {
		target := byID[identifier].Supersedes
		if reported[target] || !slices.ContainsFunc(after, func(note notes.Note) bool { return note.ID == target }) {
			continue
		}
		reported[target] = true
		switch {
		case !*restore && remaining[target] == 0:
			fmt.Fprintf(stdout, "Note %s, which it superseded, is live again.\n", target)
		case *restore && before[target] == 0:
			fmt.Fprintf(stdout, "Note %s, which it supersedes, is superseded again.\n", target)
		}
	}
	return 0
}

func supersessorCounts(all []notes.Note) map[string]int {
	counts := map[string]int{}
	for _, note := range all {
		if note.Supersedes != "" {
			counts[note.Supersedes]++
		}
	}
	return counts
}
