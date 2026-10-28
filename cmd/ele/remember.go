package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/profile"
)

const bodyLineLimit = 3

// The index's owners were resolved under the map it was synced with, and notes keep their folder's owner whether listed or not.
func projectOwner(ctx context.Context, dataDirectory string, ownerMap ownermap.OwnerMap, project string) (string, error) {
	if owner := ownerMap.ProjectOwner(project); owner != "" {
		return owner, nil
	}
	refusal := fmt.Errorf("project %q has no listed owner in owner-map.json or the index; pass --owner <name>", project)
	database := filepath.Join(dataDirectory, "ele.db")
	if _, err := os.Stat(database); errors.Is(err, fs.ErrNotExist) {
		return "", refusal
	}
	searchIndex, err := index.OpenChecked(database)
	if err != nil {
		return "", err
	}
	defer searchIndex.Close()
	current, err := ownermap.Fingerprint(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		return "", err
	}
	indexedUnder, err := searchIndex.Setting(ctx, "owner-map")
	if err != nil {
		return "", err
	}
	if indexedUnder != current {
		return "", fmt.Errorf("the index was not synced under the current owner-map.json, so it cannot name project %q's owner; run ele sync, or pass --owner <name>", project)
	}
	owners, err := searchIndex.ProjectOwners(ctx, project)
	if err != nil {
		return "", err
	}
	owners = slices.DeleteFunc(owners, func(owner string) bool { return !slices.Contains(ownerMap.Owners, owner) })
	switch len(owners) {
	case 0:
		return "", refusal
	case 1:
		return owners[0], nil
	}
	return "", fmt.Errorf("project %q has entries under owners %s; pass --owner <name>", project, strings.Join(owners, ", "))
}

func remember(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("remember", stderr)
	doneFile := flags.String("done-file", "", "file whose last change marks the nudge done; needs --every")
	due := flags.String("due", "", "YYYY-MM-DD or YYYY-MM-DDTHH:MM, local, a reminder first shows from; reminders only, today by default with --every")
	everyValue := flags.String("every", "", "how long a nudge stays hidden after each close: 5m, 2h, 7d, or a bare number of days; whole days, 24h included, re-arm to a date; reminders only")
	everywhere := flags.Bool("everywhere", false, "applies to every session, whatever its owner, owners added later included")
	global := flags.Bool("global", false, "applies to every project of the owner")
	kind := flags.String("kind", "", "decision, fact, handoff, question or reminder")
	owner := flags.String("owner", "", "file under this owner instead of the resolved one")
	projectName := flags.String("project", "", "file under this project instead of the working directory's")
	session := flags.String("session", os.Getenv("CLAUDE_CODE_SESSION_ID"), "id of the session the note came from")
	var supersedes repeatedFlag
	flags.Var(&supersedes, "supersedes", "id of a note this one replaces; repeat for several")
	positional, err := parseInterspersed(flags, args)
	if err != nil {
		return rejectCall(flags, err, stdout, stderr)
	}
	everyGiven := false
	flags.Visit(func(given *flag.Flag) { everyGiven = everyGiven || given.Name == "every" })
	var every time.Duration
	body := strings.TrimSpace(strings.Join(positional, " "))
	if body == "" || *kind == "" {
		return rejectCall(flags, nil, stdout, stderr)
	}
	if *kind == notes.KindPlan {
		fmt.Fprintln(stderr, "ele: a plan note is written by ele plan start <path>, not ele remember")
		return 2
	}
	kinds := slices.DeleteFunc(slices.Clone(notes.Kinds), func(kind string) bool { return kind == notes.KindPlan })
	if !slices.Contains(kinds, *kind) {
		fmt.Fprintf(stderr, "ele: unknown kind %q; use %s\n", *kind, strings.Join(kinds, ", "))
		return 2
	}
	if *kind == notes.KindHandoff && *global {
		fmt.Fprintln(stderr, "ele: a handoff is scoped to its project; drop --global")
		return 2
	}
	if *kind == notes.KindHandoff && *everywhere {
		fmt.Fprintln(stderr, "ele: a handoff is scoped to its project; drop --everywhere")
		return 2
	}
	if *everywhere && (*global || *owner != "" || *projectName != "") {
		fmt.Fprintln(stderr, "ele: --everywhere spans every owner and project; drop --global, --owner and --project")
		return 2
	}
	switch {
	case *kind != notes.KindReminder && (*doneFile != "" || *due != "" || everyGiven):
		fmt.Fprintln(stderr, "ele: only a reminder takes --done-file, --due or --every; use --kind reminder")
		return 2
	case everyGiven:
		if every, err = notes.ParseEvery(*everyValue); err != nil {
			fmt.Fprintf(stderr, "ele: --%v\n", err)
			return 2
		}
	}
	switch {
	case *doneFile != "" && every == 0:
		fmt.Fprintln(stderr, "ele: --done-file needs --every <interval>, how long the nudge stays hidden after the file changes")
		return 2
	case *kind == notes.KindReminder && *due == "" && every == 0:
		fmt.Fprintln(stderr, "ele: a reminder needs --due YYYY-MM-DD or YYYY-MM-DDTHH:MM, when it first shows")
		return 2
	case *due != "" && !notes.ValidDue(*due):
		fmt.Fprintf(stderr, "ele: --due %q is not YYYY-MM-DD or YYYY-MM-DDTHH:MM\n", *due)
		return 2
	}
	if !notes.ValidSession(*session) {
		fmt.Fprintf(stderr, "ele: session %q holds characters other than letters, digits, - and _\n", *session)
		return 2
	}
	if lines := strings.Count(body, "\n") + 1; lines > bodyLineLimit {
		fmt.Fprintf(stderr, "ele: a note is at most %d lines, not %d; keep the point and its reason, drop the rest\n", bodyLineLimit, lines)
		return 2
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
	project, scope := "", notes.ScopeEverywhere
	if *everywhere {
		*owner = notes.OwnerEverywhere
	} else {
		var resolved string
		var code int
		project, resolved, code, err = resolveOwner(ctx, dataDirectory, ownerMap, *projectName, *owner)
		if err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return code
		}
		*owner = resolved
		scope = notes.ScopeProject
		if *global {
			scope = notes.ScopeGlobal
		}
	}

	noteDirectory := filepath.Join(dataDirectory, "notes")
	existing, _, err := notes.ReadOwner(noteDirectory, *owner)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	supersedes = slices.DeleteFunc(supersedes, func(target string) bool { return target == "" })
	for at, target := range supersedes {
		if slices.Contains(supersedes[:at], target) {
			fmt.Fprintf(stderr, "ele: --supersedes %s given twice\n", target)
			return 2
		}
		if !slices.ContainsFunc(existing, func(note notes.Note) bool { return note.ID == target }) {
			fmt.Fprintf(stderr, "ele: %s has no note %s to supersede\n", *owner, target)
			return 2
		}
	}
	if duplicate := sayingTheSame(existing, notes.Note{Body: body, Kind: *kind, Project: project, Scope: scope}); duplicate != "" {
		fmt.Fprintf(stderr, "ele: note %s already says this; nothing written\n", duplicate)
		return 1
	}
	now := time.Now()
	if every > 0 && *due == "" {
		*due = now.Format(time.DateOnly)
	}
	if *doneFile != "" {
		// A quoted or --done-file= path reaches ele with its ~ unexpanded.
		if rest, ok := strings.CutPrefix(*doneFile, "~"); ok && (rest == "" || strings.HasPrefix(rest, "/")) {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				fmt.Fprintf(stderr, "ele: %v\n", homeErr)
				return 1
			}
			*doneFile = home + rest
		}
		if *doneFile, err = filepath.Abs(*doneFile); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
	}
	id, err := notes.NewID(now)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	note := notes.Note{
		Body:       body,
		Date:       now.Format(time.DateOnly),
		DoneFile:   *doneFile,
		Due:        *due,
		Every:      every,
		ID:         id,
		Kind:       *kind,
		Owner:      *owner,
		Profile:    profile.Current(os.Getenv("CLAUDE_CONFIG_DIR")),
		Project:    project,
		Scope:      scope,
		Session:    *session,
		Supersedes: supersedes,
		Time:       now.UTC().Format(time.RFC3339),
	}
	path, err := notes.Write(noteDirectory, note)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Remembered %s: %s\n", id, path)
	return 0
}

func resolveOwner(ctx context.Context, dataDirectory string, ownerMap ownermap.OwnerMap, projectName string, owner string) (string, string, int, error) {
	if projectName != "" && (strings.TrimSpace(projectName) != projectName || !notes.ValidProject(projectName)) {
		return "", "", 2, fmt.Errorf("project %q has surrounding spaces or a control character", projectName)
	}
	if owner != "" && !slices.Contains(ownerMap.Owners, owner) {
		listed := "no owners"
		if len(ownerMap.Owners) > 0 {
			listed = strings.Join(ownerMap.Owners, ", ")
		}
		return "", "", 2, fmt.Errorf("unknown owner %q; owner-map.json lists %s", owner, listed)
	}
	switch {
	case projectName != "" && owner != "":
		return projectName, owner, 0, nil
	case projectName != "":
		resolved, err := projectOwner(ctx, dataDirectory, ownerMap, projectName)
		if err != nil {
			return "", "", 1, err
		}
		return projectName, resolved, 0, nil
	}
	directory, err := os.Getwd()
	if err != nil {
		return "", "", 1, err
	}
	project, resolved := index.NewProjects(ownerMap).Resolve(directory)
	switch {
	case owner != "":
		return project, owner, 0, nil
	case resolved == "":
		return "", "", 1, unownedError{directory: directory, project: project}
	}
	return project, resolved, 0, nil
}

type unownedError struct {
	directory string
	project   string
}

func (e unownedError) Error() string {
	return fmt.Sprintf("no owner in owner-map.json covers %s (project %s); add a path rule or project entry, or pass --owner <name>", e.directory, e.project)
}

func sayingTheSame(existing []notes.Note, candidate notes.Note) string {
	superseded := supersededBy(existing)
	// A closed handoff or reminder no longer shows, so the same words may be saved again; a superseded note shows nowhere.
	hidden := func(note notes.Note) bool {
		return superseded[note.ID] != "" || (note.Kind == notes.KindHandoff && note.Shown != "") || note.Closed != ""
	}
	if hidden(candidate) {
		return ""
	}
	words := strings.Join(strings.Fields(candidate.Body), " ")
	for _, note := range existing {
		sameScope := note.Scope == candidate.Scope && (candidate.Scope == notes.ScopeGlobal || note.Project == candidate.Project)
		if sameScope && note.Kind == candidate.Kind && note.ID != candidate.ID && !hidden(note) && strings.Join(strings.Fields(note.Body), " ") == words {
			return note.ID
		}
	}
	return ""
}
