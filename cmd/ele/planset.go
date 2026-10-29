package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/plan"
	"github.com/nicolaspearson/claude-ephant/internal/profile"
)

const namesUnique = "item names are unique across active and closed plans"

// Tests fix it, so a dated fixture does not age into a warning.
var planClock = time.Now

// A plan that cannot be loaded is left out and named in unchecked: it resolves no edge and clashes with nothing.
type planSet struct {
	active        map[string]plan.Plan
	activePaths   []string
	closed        []plan.Stored
	dataDirectory string
	unchecked     []uncheckedPlan
}

type uncheckedPlan struct {
	path   string
	reason string
}

func loadPlanSet(root string, all []notes.Note, project string, noteDirectory string) planSet {
	set := planSet{active: map[string]plan.Plan{}, dataDirectory: filepath.Dir(noteDirectory)}
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
	roots, err := profile.Discover(home)
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
	resolved := s.resolve(parsed)
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

// The first plan of that name holding the item answers: active plans, then closed ones newest first.
func (s planSet) lookup(edge plan.Edge) plan.Blocker {
	for _, relative := range s.activePaths {
		if planBase(relative) != edge.Plan {
			continue
		}
		if at := slices.IndexFunc(s.active[relative].Actions, func(item plan.Item) bool { return item.Name == edge.Name }); at >= 0 {
			if s.active[relative].Actions[at].Done {
				return plan.BlockerDone
			}
			return plan.BlockerOpen
		}
	}
	for _, stored := range s.closed {
		if planBase(stored.Path) != edge.Plan {
			continue
		}
		if at := slices.IndexFunc(stored.Items, func(item plan.StoredItem) bool { return item.Name == edge.Name }); at >= 0 {
			if stored.Items[at].Done {
				return plan.BlockerDone
			}
			return plan.BlockerAbandoned
		}
	}
	return plan.BlockerMissing
}

func (s planSet) resolve(parsed plan.Plan) plan.Plan {
	return parsed.Resolve(planClock(), s.lookup)
}
