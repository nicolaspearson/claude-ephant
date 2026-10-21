package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/plan"
)

const namesUnique = "item names are unique across active and closed plans"

// Tests fix it, so a dated fixture does not age into a warning.
var planClock = time.Now

// A plan missing from the checkout or not parsing is left out: it resolves no edge and clashes with nothing.
type planSet struct {
	active      map[string]plan.Plan
	activePaths []string
	closed      []plan.Stored
}

func loadPlanSet(root string, all []notes.Note, project string, noteDirectory string) planSet {
	set := planSet{active: map[string]plan.Plan{}}
	for _, note := range activePlans(all, project) {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(note.Body)))
		if err != nil {
			continue
		}
		if parsed, err := plan.Parse(content); err == nil {
			set.active[note.Body] = parsed
			set.activePaths = append(set.activePaths, note.Body)
		}
	}
	superseded := supersededBy(all)
	for _, note := range newestOfKind(all, notes.KindPlan, project) {
		if note.Shown == "" || superseded[note.ID] != "" {
			continue
		}
		if stored, ok, err := plan.ReadStored(planStore(noteDirectory), note.ID); err == nil && ok {
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

func (s planSet) check(relative string, parsed plan.Plan) (plan.Plan, []plan.Warning) {
	resolved := s.resolve(parsed)
	warnings := resolved.Check(planClock())
	for _, clash := range s.clashes(relative, parsed) {
		warnings = append(warnings, plan.Warning{Text: clash + "; " + namesUnique})
	}
	return resolved, warnings
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
