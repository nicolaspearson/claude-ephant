package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

type projectScope struct {
	all           []notes.Note
	dataDirectory string
	noteDirectory string
	owner         string
	project       string
}

type unownedError struct {
	directory string
	project   string
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
		return cmp.Or(cmp.Compare(b.Time, a.Time), cmp.Compare(b.ID, a.ID))
	})
	return matched
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
	all, _, err := notes.ReadFor(noteDirectory, resolved)
	if err != nil {
		return projectScope{}, 1, err
	}
	return projectScope{all: all, dataDirectory: dataDirectory, noteDirectory: noteDirectory, owner: resolved, project: project}, 0, nil
}

// A directory no owner covers still shows the everywhere notes, so it may list and change them.
func projectOrEverywhereNotes(ctx context.Context, owner string, projectName string) (projectScope, int, error) {
	scope, code, err := projectNotes(ctx, owner, projectName)
	var unowned unownedError
	if !errors.As(err, &unowned) {
		return scope, code, err
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		return projectScope{}, 1, err
	}
	noteDirectory := filepath.Join(dataDirectory, "notes")
	all, _, err := notes.ReadFor(noteDirectory, "")
	if err != nil {
		return projectScope{}, 1, err
	}
	return projectScope{all: all, dataDirectory: dataDirectory, noteDirectory: noteDirectory, project: unowned.project}, 0, nil
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

func (e unownedError) Error() string {
	return fmt.Sprintf("no owner in owner-map.json covers %s (project %s); add a path rule or project entry, or pass --owner <name>", e.directory, e.project)
}
