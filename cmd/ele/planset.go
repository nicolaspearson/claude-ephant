package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/plan"
	"github.com/nicolaspearson/claude-ephant/internal/profile"
)

const namesUnique = "item names are unique across active and closed plans"

// Tests fix it, so a dated fixture does not age into a warning.
var planClock = time.Now

// Read once a blocker names another project, so a set with none never opens the index.
type foreignPlans struct {
	directories      []dto.ProjectDirectory
	directoriesError error
	directoriesRead  bool
	planNotes        map[string][]notes.Note
}

// A plan that cannot be loaded is left out and named in unchecked: it resolves no edge and clashes with nothing.
type planSet struct {
	active        map[string]plan.Plan
	activePaths   []string
	closed        []plan.Stored
	dataDirectory string
	foreign       *foreignPlans
	project       string
	unchecked     []uncheckedPlan
}

type uncheckedPlan struct {
	path   string
	reason string
}

func activeState(parsed plan.Plan, name string) (plan.Blocker, bool) {
	at := slices.IndexFunc(parsed.Actions, func(item plan.Item) bool { return item.Name == name })
	switch {
	case at < 0:
		return plan.BlockerMissing, false
	case parsed.Actions[at].Done:
		return plan.BlockerDone, true
	}
	return plan.BlockerOpen, true
}

func closedState(stored plan.Stored, name string) (plan.Blocker, bool) {
	at := slices.IndexFunc(stored.Items, func(item plan.StoredItem) bool { return item.Name == name })
	switch {
	case at < 0:
		return plan.BlockerMissing, false
	case stored.Items[at].Done:
		return plan.BlockerDone, true
	}
	return plan.BlockerAbandoned, true
}

func loadPlanSet(root string, all []notes.Note, project string, noteDirectory string) planSet {
	set := planSet{active: map[string]plan.Plan{}, dataDirectory: filepath.Dir(noteDirectory), foreign: &foreignPlans{}, project: project}
	leaveOut := func(note notes.Note, format string, arguments ...any) {
		set.unchecked = append(set.unchecked, uncheckedPlan{path: note.Body, reason: fmt.Sprintf(format, arguments...)})
	}
	for _, note := range activePlans(all, project) {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(note.Body)))
		if errors.Is(err, fs.ErrNotExist) {
			leaveOut(note, "active plan %s, missing from the checkout", note.Body)
			continue
		}
		if err != nil {
			leaveOut(note, "active plan %s, which does not read", note.Body)
			continue
		}
		parsed, err := plan.Parse(content)
		if err != nil {
			leaveOut(note, "active plan %s, which does not parse", note.Body)
			continue
		}
		set.active[note.Body] = parsed
		set.activePaths = append(set.activePaths, note.Body)
	}
	// The hint names the id: given the path, ele plan done takes an active plan restarted there first.
	for _, note := range closedPlans(all, project) {
		stored, ok, err := plan.ReadStored(planStore(noteDirectory), note.ID)
		switch {
		case err != nil:
			leaveOut(note, "closed plan %s, whose stored record does not read (ele plan done %s rewrites it)", note.Body, note.ID)
		case !ok:
			leaveOut(note, "closed plan %s, with no stored record (ele plan done %s writes it)", note.Body, note.ID)
		default:
			set.closed = append(set.closed, stored)
		}
	}
	return set
}

func planBase(relative string) string {
	return strings.TrimSuffix(path.Base(relative), ".md")
}

func planStore(noteDirectory string) string {
	return filepath.Join(filepath.Dir(noteDirectory), "plans")
}

// A session not yet synced, the running one included, has only its transcript.
func transcribed(session string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return true
	}
	roots, _, err := profile.Discover(home, os.Getenv("CLAUDE_CONFIG_DIR"))
	if err != nil {
		return true
	}
	for _, root := range roots {
		projects, err := os.ReadDir(filepath.Join(root.Directory, "projects"))
		if err != nil {
			return true
		}
		for _, project := range projects {
			if !project.IsDir() {
				continue
			}
			if _, err = os.Stat(filepath.Join(root.Directory, "projects", project.Name(), session+".jsonl")); !errors.Is(err, fs.ErrNotExist) {
				return true
			}
		}
	}
	return false
}

func (s planSet) check(ctx context.Context, relative string, parsed plan.Plan) (plan.Plan, []plan.Warning) {
	resolved := s.resolve(ctx, parsed)
	resolved.Citations = s.cited(ctx, parsed.Citations)
	warnings := resolved.Check(planClock())
	for _, clash := range s.clashes(relative, parsed) {
		warnings = append(warnings, plan.Warning{Text: clash + "; " + namesUnique})
	}
	return resolved, warnings
}

// Citations ele resolves; all of them when the index cannot be read.
func (s planSet) cited(ctx context.Context, citations []plan.Citation) []plan.Citation {
	if len(citations) == 0 || s.dataDirectory == "" {
		return citations
	}
	database := filepath.Join(s.dataDirectory, "ele.db")
	if _, err := os.Stat(database); err != nil {
		return citations
	}
	searchIndex, err := index.OpenChecked(database)
	if err != nil {
		return citations
	}
	defer searchIndex.Close()
	owners, ownersErr := os.ReadDir(filepath.Join(s.dataDirectory, "notes"))
	resolves := func(reference plan.Reference) bool {
		if reference.Kind == plan.ReferenceNote {
			if ownersErr != nil && !errors.Is(ownersErr, fs.ErrNotExist) {
				return true
			}
			return slices.ContainsFunc(owners, func(owner fs.DirEntry) bool {
				_, err := os.Stat(filepath.Join(s.dataDirectory, "notes", owner.Name(), reference.ID+".md"))
				return err == nil || !errors.Is(err, fs.ErrNotExist)
			})
		}
		sessions, err := searchIndex.Sessions(ctx, reference.ID, 1)
		return err != nil || len(sessions) > 0 || transcribed(reference.ID)
	}
	return slices.DeleteFunc(slices.Clone(citations), func(citation plan.Citation) bool {
		return !slices.ContainsFunc(citation.References, resolves)
	})
}

// A closed plan reopened at its own path may reuse its names.
func (s planSet) clashes(relative string, parsed plan.Plan) []string {
	var clashes []string
	for _, action := range parsed.Actions {
		if action.Name == "" {
			continue
		}
		for _, other := range s.activePaths {
			if other != relative && slices.ContainsFunc(s.active[other].Actions, func(item plan.Item) bool { return item.Name == action.Name }) {
				clashes = append(clashes, fmt.Sprintf("item name %q is also in %s", action.Name, other))
			}
		}
		for _, stored := range s.closed {
			if stored.Path != relative && slices.ContainsFunc(stored.Items, func(item plan.StoredItem) bool { return item.Name == action.Name }) {
				clashes = append(clashes, fmt.Sprintf("item name %q was used by closed plan %s", action.Name, stored.Path))
			}
		}
	}
	return clashes
}

// The other checkout's working tree is the truth: a tick on a branch not checked out there does not count.
func (s planSet) foreignActive(ctx context.Context, edge plan.Edge, owner string, relative string) (plan.Blocker, string) {
	directories, err := s.foreign.indexed(ctx, s.dataDirectory)
	if err != nil {
		return plan.BlockerUnreachable, err.Error()
	}
	ownerMap, err := ownermap.Load(filepath.Join(s.dataDirectory, "owner-map.json"))
	if err != nil {
		return plan.BlockerUnreachable, fmt.Sprintf("owner-map.json does not read: %v", err)
	}
	var checkouts, holding []string
	for _, directory := range directories {
		if directory.Project != edge.Project {
			continue
		}
		// The root of a deleted directory walks up to whatever ancestor repository holds it.
		if info, statErr := os.Stat(directory.Directory); statErr != nil || !info.IsDir() {
			continue
		}
		// A linked worktree reads its main checkout, so one open beside it is no second checkout.
		root := index.RepositoryRoot(directory.Directory)
		// Path rules match the root as reached; one checkout reached through a symlink and its target counts once.
		resolved, evalErr := filepath.EvalSymlinks(root)
		if evalErr != nil || slices.Contains(checkouts, resolved) || ownerMap.Owner(edge.Project, root) != owner {
			continue
		}
		checkouts = append(checkouts, resolved)
		if _, statErr := os.Stat(filepath.Join(resolved, filepath.FromSlash(relative))); !errors.Is(statErr, fs.ErrNotExist) {
			holding = append(holding, resolved)
		}
	}
	switch {
	case len(checkouts) == 0:
		return plan.BlockerUnreachable, fmt.Sprintf("no checkout of %s the index lists is still there; a session there syncs it", edge.Project)
	case len(holding) == 0:
		return plan.BlockerUnreachable, fmt.Sprintf("%s is missing from %s", relative, strings.Join(checkouts, " and "))
	case len(holding) > 1:
		return plan.BlockerUnreachable, fmt.Sprintf("%s is checked out in %s, and ele cannot tell which", edge.Project, strings.Join(holding, " and "))
	}
	content, err := os.ReadFile(filepath.Join(holding[0], filepath.FromSlash(relative)))
	if err != nil {
		return plan.BlockerUnreachable, fmt.Sprintf("%s in %s does not read", relative, holding[0])
	}
	parsed, err := plan.Parse(content)
	if err != nil {
		return plan.BlockerUnreachable, fmt.Sprintf("%s in %s does not parse", relative, holding[0])
	}
	blocker, _ := activeState(parsed, edge.Name)
	return blocker, ""
}

// A blocker naming another project is the ask to cross owners: every owner's plan notes for it are read, and only its state and reason are printed.
func (s planSet) foreignLookup(ctx context.Context, edge plan.Edge) (plan.Blocker, string) {
	noteDirectory := filepath.Join(s.dataDirectory, "notes")
	byOwner, err := s.foreign.ownerNotes(noteDirectory)
	if err != nil {
		return plan.BlockerUnreachable, fmt.Sprintf("the plan notes do not read: %v", err)
	}
	var owners []string
	var active, closed []notes.Note
	for _, owner := range slices.Sorted(maps.Keys(byOwner)) {
		otherPlan := func(note notes.Note) bool { return planBase(note.Body) != edge.Plan }
		ownerActive := slices.DeleteFunc(activePlans(byOwner[owner], edge.Project), otherPlan)
		ownerClosed := slices.DeleteFunc(closedPlans(byOwner[owner], edge.Project), otherPlan)
		if len(ownerActive)+len(ownerClosed) > 0 {
			owners = append(owners, owner)
			active, closed = ownerActive, ownerClosed
		}
	}
	switch {
	case len(owners) == 0:
		return plan.BlockerUnreachable, fmt.Sprintf("%s has no plan named %s started", edge.Project, edge.Plan)
	case len(owners) > 1:
		return plan.BlockerUnreachable, fmt.Sprintf("a plan named %s is started in %s under owners %s, and ele cannot tell which", edge.Plan, edge.Project, strings.Join(owners, ", "))
	}
	// A plan restarted at its path may lack an item its closed run held.
	if len(active) > 0 {
		if blocker, reason := s.foreignActive(ctx, edge, owners[0], active[0].Body); blocker != plan.BlockerMissing {
			return blocker, reason
		}
	}
	for _, note := range closed {
		stored, ok, readErr := plan.ReadStored(planStore(noteDirectory), note.ID)
		switch {
		case readErr != nil:
			return plan.BlockerUnreachable, fmt.Sprintf("closed plan %s's stored record does not read", note.Body)
		case !ok:
			return plan.BlockerUnreachable, fmt.Sprintf("closed plan %s has no stored record", note.Body)
		}
		if blocker, found := closedState(stored, edge.Name); found {
			return blocker, ""
		}
	}
	return plan.BlockerMissing, ""
}

// The first plan of that name holding the item answers: active plans, then closed ones newest first.
// An unchecked plan still counts as a plan of this project; a set never loaded counts every plan as known, so none reads as absent.
func (s planSet) lookup(ctx context.Context, edge plan.Edge) (plan.Blocker, string) {
	if edge.Project != "" && edge.Project != s.project {
		if s.foreign == nil {
			return plan.BlockerMissing, ""
		}
		return s.foreignLookup(ctx, edge)
	}
	known := s.active == nil || slices.ContainsFunc(s.unchecked, func(unchecked uncheckedPlan) bool { return planBase(unchecked.path) == edge.Plan })
	for _, relative := range s.activePaths {
		if planBase(relative) != edge.Plan {
			continue
		}
		known = true
		if blocker, found := activeState(s.active[relative], edge.Name); found {
			return blocker, ""
		}
	}
	for _, stored := range s.closed {
		if planBase(stored.Path) != edge.Plan {
			continue
		}
		known = true
		if blocker, found := closedState(stored, edge.Name); found {
			return blocker, ""
		}
	}
	if !known {
		return plan.BlockerNoPlan, ""
	}
	return plan.BlockerMissing, ""
}

func (s planSet) resolve(ctx context.Context, parsed plan.Plan) plan.Plan {
	return parsed.Resolve(planClock(), func(edge plan.Edge) (plan.Blocker, string) { return s.lookup(ctx, edge) })
}

// A missing index is refused like an unfinished one: with no sync, no checkout is known.
func (f *foreignPlans) indexed(ctx context.Context, dataDirectory string) ([]dto.ProjectDirectory, error) {
	if !f.directoriesRead {
		f.directoriesRead = true
		if _, err := os.Stat(filepath.Join(dataDirectory, "ele.db")); errors.Is(err, fs.ErrNotExist) {
			f.directoriesError = errors.New("no sync has completed; run ele sync, then retry")
		} else {
			f.directories, f.directoriesError = indexedDirectories(ctx, dataDirectory)
		}
	}
	return f.directories, f.directoriesError
}

func (f *foreignPlans) ownerNotes(noteDirectory string) (map[string][]notes.Note, error) {
	if f.planNotes != nil {
		return f.planNotes, nil
	}
	entries, err := os.ReadDir(noteDirectory)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	read := map[string][]notes.Note{}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == notes.OwnerEverywhere {
			continue
		}
		if read[entry.Name()], _, err = notes.ReadOwner(noteDirectory, entry.Name()); err != nil {
			return nil, err
		}
	}
	f.planNotes = read
	return read, nil
}
