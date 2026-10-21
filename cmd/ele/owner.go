package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

// Owner becomes a folder name under notes/.
var ownerShape = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

type ownerChange struct {
	apply     func(ownerMap *ownermap.OwnerMap, arguments []string) (string, error)
	arguments int
	project   bool
}

type noteMove struct {
	forgotten bool
	from      string
	notes     []notes.Note
	project   string
	to        string
}

type ownerOptions struct {
	move   bool
	moveTo string
	noSync bool
}

type ownerWrite struct {
	after       ownermap.OwnerMap
	before      ownermap.OwnerMap
	described   string // empty when the map already said what was asked
	directories []dto.ProjectDirectory
	moves       []noteMove
}

// add re-owns too: listing an owner turns on any rule or entry already naming it.
var ownerChanges = map[string]ownerChange{
	"add":            {apply: addOwner, arguments: 1},
	"add-path":       {apply: addPath, arguments: 2},
	"add-project":    {apply: addProject, arguments: 2, project: true},
	"remove-path":    {apply: removePath, arguments: 1},
	"remove-project": {apply: removeProject, arguments: 1, project: true},
}

func ownerCommand(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		return misuse("owner", args, stdout, stderr)
	}
	change, ok := ownerChanges[args[0]]
	if !ok {
		return misuse("owner", args, stdout, stderr)
	}
	flags := newFlagSet("owner "+args[0], stderr)
	var options ownerOptions
	flags.BoolVar(&options.move, "move", false, "refile the notes of each project that changes owner under its new owner")
	flags.StringVar(&options.moveTo, "move-to", "", "refile notes no single owner would cover under this listed owner")
	flags.BoolVar(&options.noSync, "no-sync", false, "write the map without starting a background sync")
	arguments, err := parseInterspersed(flags, args[1:])
	if err != nil || len(arguments) != change.arguments {
		return rejectCall(flags, err, stdout, stderr)
	}
	code, err := changeOwners(ctx, change, arguments, options, stdout)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
	}
	return code
}

func addOwner(ownerMap *ownermap.OwnerMap, arguments []string) (string, error) {
	owner := arguments[0]
	if !ownerShape.MatchString(owner) {
		return "", fmt.Errorf("owner %q must be letters, digits, - and _, starting with a letter or digit", owner)
	}
	if slices.Contains(ownerMap.Owners, owner) {
		return "", fmt.Errorf("owner-map.json already lists owner %q", owner)
	}
	ownerMap.Owners = append(ownerMap.Owners, owner)
	return fmt.Sprintf("Added owner %s.", owner), nil
}

func addPath(ownerMap *ownermap.OwnerMap, arguments []string) (string, error) {
	directory, owner := arguments[0], arguments[1]
	if !filepath.IsAbs(directory) {
		return "", fmt.Errorf("path %q is relative; give the directory's absolute path", directory)
	}
	if err := listedOwner(*ownerMap, owner); err != nil {
		return "", err
	}
	directory = filepath.Clean(directory)
	previous := ownerMap.Paths[directory]
	ownerMap.Paths[directory] = owner
	return fmt.Sprintf("Mapped path %s to %s%s.", directory, owner, was(previous)), nil
}

func addProject(ownerMap *ownermap.OwnerMap, arguments []string) (string, error) {
	project, owner := arguments[0], arguments[1]
	if err := validProjectName(project); err != nil {
		return "", err
	}
	if err := listedOwner(*ownerMap, owner); err != nil {
		return "", err
	}
	previous := ownerMap.Projects[project]
	ownerMap.Projects[project] = owner
	return fmt.Sprintf("Mapped project %s to %s%s.", project, owner, was(previous)), nil
}

func changeOwners(ctx context.Context, change ownerChange, arguments []string, options ownerOptions, stdout io.Writer) (int, error) {
	dataDirectory, err := datadir.Path()
	if err != nil {
		return 1, err
	}
	written, err := writeOwnerChange(ctx, dataDirectory, change, arguments, options)
	if err != nil {
		return 1, err
	}
	if written.described == "" {
		fmt.Fprintln(stdout, "owner-map.json already says that; nothing written.")
		return 0, nil
	}
	fmt.Fprintln(stdout, written.described)
	if owners := ownermap.Moves(written.before, written.after, written.directories); len(owners) > 0 {
		fmt.Fprintln(stdout, "Projects changing owner:")
		for _, owner := range owners {
			fmt.Fprintf(stdout, "  %s: %s → %s\n", owner.Project, ownerOrNone(owner.From), ownerOrNone(owner.To))
		}
	} else {
		fmt.Fprintln(stdout, "No indexed project changes owner.")
	}
	for _, planned := range written.moves {
		fmt.Fprintf(stdout, "Moved %s of %s from %s to %s.\n", noteCount(planned), planned.project, planned.from, planned.to)
	}
	if options.noSync {
		fmt.Fprintln(stdout, "The next sync rebuilds the index under the new map.")
		return 0, nil
	}
	if err = startSync(dataDirectory); err != nil {
		return 1, fmt.Errorf("start background sync: %w; run ele sync to rebuild the index under the new map", err)
	}
	fmt.Fprintln(stdout, "Started a background sync, which rebuilds the index under the new map; ele status shows it.")
	return 0, nil
}

// A missing index has no directories yet. An unfinished or older one is refused: its table may be partial.
func indexedDirectories(ctx context.Context, dataDirectory string) ([]dto.ProjectDirectory, error) {
	database := filepath.Join(dataDirectory, "ele.db")
	if _, err := os.Stat(database); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	searchIndex, err := index.OpenChecked(database)
	if err != nil {
		return nil, err
	}
	defer searchIndex.Close()
	version, err := searchIndex.Setting(ctx, "index-version")
	if err != nil {
		return nil, err
	}
	indexedUnder, err := searchIndex.Setting(ctx, "owner-map")
	if err != nil {
		return nil, err
	}
	if version != index.Version || indexedUnder == "" {
		return nil, errors.New("no sync has completed under this version of ele; run ele sync, then retry")
	}
	return searchIndex.ProjectDirectories(ctx)
}

func listedOwner(ownerMap ownermap.OwnerMap, owner string) error {
	if slices.Contains(ownerMap.Owners, owner) {
		return nil
	}
	listed := strings.Join(ownerMap.Owners, ", ")
	if listed == "" {
		listed = "none"
	}
	return fmt.Errorf("unknown owner %q; owner-map.json lists %s, and ele owner add %s adds it", owner, listed, owner)
}

func noteCount(planned noteMove) string {
	if planned.forgotten {
		return plural(len(planned.notes), "forgotten note", "forgotten notes")
	}
	return plural(len(planned.notes), "note", "notes")
}

func noteTree(dataDirectory string, forgotten bool) string {
	if forgotten {
		return filepath.Join(dataDirectory, "forgotten")
	}
	return filepath.Join(dataDirectory, "notes")
}

func ownerMapsEqual(before ownermap.OwnerMap, after ownermap.OwnerMap) bool {
	return slices.Equal(before.Owners, after.Owners) && maps.Equal(before.Paths, after.Paths) && maps.Equal(before.Projects, after.Projects)
}

func ownerOrNone(owner string) string {
	if owner == "" {
		return "(no owner)"
	}
	return owner
}

func ownersOf(ownerMap ownermap.OwnerMap, directories []dto.ProjectDirectory) []string {
	var owners []string
	for _, directory := range directories {
		if owner := ownerMap.Owner(directory.Project, directory.Directory); !slices.Contains(owners, owner) {
			owners = append(owners, owner)
		}
	}
	return owners
}

// Uncovered moves carry no destination: no single owner covers their project after the change.
// Forgotten notes move too, so a later restore lands under the covering owner.
func plannedNoteMoves(dataDirectory string, before ownermap.OwnerMap, after ownermap.OwnerMap, directories []dto.ProjectDirectory) ([]noteMove, []noteMove, error) {
	byProject := map[string][]dto.ProjectDirectory{}
	for _, directory := range directories {
		byProject[directory.Project] = append(byProject[directory.Project], directory)
	}
	var moves, uncovered []noteMove
	for _, owner := range before.Owners {
		for _, forgotten := range []bool{false, true} {
			all, _, err := notes.ReadOwner(noteTree(dataDirectory, forgotten), owner)
			if err != nil {
				return nil, nil, err
			}
			filed := map[string][]notes.Note{}
			for _, note := range all {
				if note.Scope == notes.ScopeProject {
					filed[note.Project] = append(filed[note.Project], note)
				}
			}
			for _, project := range slices.Sorted(maps.Keys(filed)) {
				projectDirectories := byProject[project]
				if len(projectDirectories) == 0 {
					projectDirectories = []dto.ProjectDirectory{{Project: project}}
				}
				if !slices.Contains(ownersOf(before, projectDirectories), owner) {
					continue
				}
				planned := noteMove{forgotten: forgotten, from: owner, notes: filed[project], project: project}
				owners := ownersOf(after, projectDirectories)
				switch {
				case slices.Contains(owners, owner):
				case len(owners) == 1 && owners[0] != "":
					planned.to = owners[0]
					moves = append(moves, planned)
				default:
					uncovered = append(uncovered, planned)
				}
			}
		}
	}
	return moves, uncovered, nil
}

func removePath(ownerMap *ownermap.OwnerMap, arguments []string) (string, error) {
	directory := filepath.Clean(arguments[0])
	previous, ok := ownerMap.Paths[directory]
	if !ok {
		return "", fmt.Errorf("owner-map.json has no path rule for %q", directory)
	}
	delete(ownerMap.Paths, directory)
	return fmt.Sprintf("Removed path %s%s.", directory, was(previous)), nil
}

func removeProject(ownerMap *ownermap.OwnerMap, arguments []string) (string, error) {
	project := arguments[0]
	previous, ok := ownerMap.Projects[project]
	if !ok {
		return "", fmt.Errorf("owner-map.json has no project entry for %q", project)
	}
	delete(ownerMap.Projects, project)
	return fmt.Sprintf("Removed project %s%s.", project, was(previous)), nil
}

func validProjectName(project string) error {
	if project == "" || strings.TrimSpace(project) != project || strings.ContainsRune(project, filepath.Separator) || !notes.ValidProject(project) {
		return fmt.Errorf("project %q must be a directory name, with no separator, control character or surrounding space", project)
	}
	return nil
}

// Holds the archive lock from reading the map to saving it: a rebuild empties the directories table, a sync must not
// index notes mid-move, and a second owner command must not plan from the map this one replaces.
func writeOwnerChange(ctx context.Context, dataDirectory string, change ownerChange, arguments []string, options ownerOptions) (ownerWrite, error) {
	unlock, held, err := archive.TryLock(filepath.Join(dataDirectory, "archive", "transcripts"))
	if err != nil {
		return ownerWrite{}, err
	}
	if !held {
		return ownerWrite{}, errors.New("a sync holds the archive lock, so the index cannot yet say which projects change owner; ele status shows it, then retry")
	}
	defer unlock()
	mapPath := filepath.Join(dataDirectory, "owner-map.json")
	before, err := ownermap.Load(mapPath)
	if err != nil {
		return ownerWrite{}, err
	}
	after := before.Clone()
	described, err := change.apply(&after, arguments)
	if err != nil {
		return ownerWrite{}, err
	}
	if ownerMapsEqual(before, after) {
		return ownerWrite{}, nil
	}
	if options.moveTo != "" {
		if err = listedOwner(after, options.moveTo); err != nil {
			return ownerWrite{}, err
		}
	}
	directories, err := indexedDirectories(ctx, dataDirectory)
	if err != nil {
		return ownerWrite{}, fmt.Errorf("the index cannot say which projects change owner: %w", err)
	}
	if change.project && !slices.ContainsFunc(directories, func(directory dto.ProjectDirectory) bool { return directory.Project == arguments[0] }) {
		directories = append(directories, dto.ProjectDirectory{Project: arguments[0]})
	}
	moves, uncovered, err := plannedNoteMoves(dataDirectory, before, after, directories)
	if err != nil {
		return ownerWrite{}, err
	}
	switch {
	case len(uncovered) > 0 && options.moveTo == "":
		var lines []string
		for _, planned := range uncovered {
			lines = append(lines, fmt.Sprintf("%s of %s under %s", noteCount(planned), planned.project, planned.from))
		}
		return ownerWrite{}, fmt.Errorf("no single owner would cover the project of %s after this change; --move-to <owner> refiles them, or nothing is written", strings.Join(lines, "; "))
	case len(uncovered) == 0 && options.moveTo != "":
		return ownerWrite{}, errors.New("--move-to is for notes no single owner would cover after this change, and none would be; nothing written")
	}
	if len(moves) > 0 && !options.move {
		var lines []string
		for _, planned := range moves {
			lines = append(lines, fmt.Sprintf("%s of %s under %s, which would move to %s", noteCount(planned), planned.project, planned.from, planned.to))
		}
		return ownerWrite{}, fmt.Errorf("this change strands %s; --move refiles them, or nothing is written", strings.Join(lines, "; "))
	}
	for _, planned := range uncovered {
		if planned.from != options.moveTo {
			planned.to = options.moveTo
			moves = append(moves, planned)
		}
	}
	// Checked before any moves, so a clash leaves every note where it was.
	for _, planned := range moves {
		for _, note := range planned.notes {
			if _, err = os.Lstat(filepath.Join(noteTree(dataDirectory, planned.forgotten), planned.to, note.ID+".md")); err == nil {
				return ownerWrite{}, fmt.Errorf("note %s is filed under both %s and %s; nothing moved or written", note.ID, planned.from, planned.to)
			}
		}
	}
	for _, planned := range moves {
		for _, note := range planned.notes {
			if err = notes.Move(noteTree(dataDirectory, planned.forgotten), note, planned.to); err != nil {
				return ownerWrite{}, err
			}
		}
	}
	if err = ownermap.Save(mapPath, after); err != nil {
		return ownerWrite{}, err
	}
	return ownerWrite{after: after, before: before, described: described, directories: directories, moves: moves}, nil
}

func was(previous string) string {
	if previous == "" {
		return ""
	}
	return " (was " + previous + ")"
}
