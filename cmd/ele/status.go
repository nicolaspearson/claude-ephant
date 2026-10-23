package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/config"
	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/profile"
)

const (
	pluginName    = "ele@claude-ephant"
	syncLogHeader = "--- background sync "
	syncLogLocked = "--- locked "
	// Two failed runs (header, lock and a one-line error each) and a running third's header and lock.
	syncLogTail = 8
)

func age(elapsed time.Duration) string {
	days, hours, minutes := int(elapsed/(24*time.Hour)), int(elapsed/time.Hour)%24, int(elapsed/time.Minute)%60
	var parts []string
	for _, part := range []struct {
		count int
		unit  string
	}{{days, "d"}, {hours, "h"}, {minutes, "m"}} {
		if part.count > 0 {
			parts = append(parts, fmt.Sprintf("%d%s", part.count, part.unit))
		}
	}
	return strings.Join(parts[:min(len(parts), 2)], "")
}

func configState(dataDirectory string) string {
	settings, err := config.Load(configPath(dataDirectory))
	if err != nil {
		return err.Error() + "; every option reads as its default"
	}
	scope, err := resolveConfigScope("", true)
	if err != nil {
		return err.Error()
	}
	changed := offDefaults(settings, scope.owner)
	if len(changed) == 0 {
		return "defaults"
	}
	return strings.Join(changed, ", ")
}

func dirSize(directory string) (int64, error) {
	var total int64
	err := filepath.WalkDir(directory, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		total += info.Size()
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	return total, err
}

func isSyncLogHeader(line string) bool {
	return strings.HasPrefix(line, syncLogHeader)
}

func listCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "none"
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if counts[names[i]] != counts[names[j]] {
			return counts[names[i]] > counts[names[j]]
		}
		return names[i] < names[j]
	})
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s %d", name, counts[name]))
	}
	return strings.Join(parts, ", ")
}

func pluginState(root profile.Root) string {
	content, err := os.ReadFile(filepath.Join(root.Directory, "settings.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return "no settings.json"
	}
	if err != nil {
		return "settings.json unreadable"
	}
	var settings struct {
		EnabledPlugins map[string]any `json:"enabledPlugins"`
	}
	if err = json.Unmarshal(content, &settings); err != nil {
		return "settings.json does not parse"
	}
	enabled, listed := settings.EnabledPlugins[pluginName]
	switch {
	case !listed:
		return "not in settings.json"
	case enabled != true:
		return "disabled"
	}
	return "enabled"
}

func plugins(home string) string {
	roots, err := profile.Discover(home)
	if err != nil {
		return err.Error()
	}
	if len(roots) == 0 {
		return "no profiles found"
	}
	states := make([]string, 0, len(roots))
	for _, root := range roots {
		states = append(states, root.Name+" "+pluginState(root))
	}
	return strings.Join(states, ", ")
}

// Runs dated after last have failed or are still running, newest last. A run dates from its lock line, or from its header
// when it never got the lock; lines before the first header count as older.
func sinceLastSync(log string, last time.Time) [][]string {
	var runs [][]string
	for line := range strings.SplitSeq(strings.TrimRight(log, "\n"), "\n") {
		switch {
		case isSyncLogHeader(line):
			runs = append(runs, []string{line})
		case len(runs) > 0:
			runs[len(runs)-1] = append(runs[len(runs)-1], line)
		}
	}
	var kept [][]string
	for _, run := range runs {
		// A time that does not parse is the zero time, before any stamp.
		at, _ := time.Parse(time.RFC3339, strings.TrimPrefix(run[0], syncLogHeader))
		if len(run) > 1 {
			if locked, isLock := strings.CutPrefix(run[1], syncLogLocked); isLock {
				at, _ = time.Parse(time.RFC3339, locked)
			}
		}
		if at.After(last) {
			kept = append(kept, run)
		}
	}
	return kept
}

func stats(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("stats", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	if err := writeStats(ctx, stdout); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}

func status(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("status", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	if err := writeStatus(stdout, time.Now()); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}

// Stat first: TryLock creates the archive directory and lock file.
func syncRunning(archiveDirectory string) string {
	if _, err := os.Stat(filepath.Join(archiveDirectory, ".lock")); errors.Is(err, fs.ErrNotExist) {
		return "no"
	}
	unlock, held, err := archive.TryLock(archiveDirectory)
	switch {
	case err != nil:
		return "unknown: " + err.Error()
	case held:
		unlock()
		return "no"
	}
	return "yes"
}

func where(dataDirectory string) string {
	ownerMap, err := ownermap.Load(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		return err.Error()
	}
	directory, err := os.Getwd()
	if err != nil {
		return err.Error()
	}
	project, owner := index.NewProjects(ownerMap).Resolve(directory)
	if owner != "" {
		return present(owner, project)
	}
	if unlisted := ownerMap.Projects[project]; unlisted != "" {
		return fmt.Sprintf("project %q maps to owner %q, which owner-map.json does not list in owners; ele owner add %s lists it", project, unlisted, unlisted)
	}
	if unlisted, ok := ownerMap.OwnerOfPath(directory); ok && unlisted != "" {
		return fmt.Sprintf("a path rule maps %s to owner %q, which owner-map.json does not list in owners; ele owner add %s lists it", directory, unlisted, unlisted)
	}
	return fmt.Sprintf("no owner covers %s; ele owner add-project %s <owner>, or ele owner add-path <dir> <owner> for a directory above it, maps it", directory, project)
}

func writeStats(ctx context.Context, stdout io.Writer) error {
	dataDirectory, err := datadir.Path()
	if err != nil {
		return err
	}
	ownerMap, err := ownermap.Load(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		return err
	}
	var indexStats dto.IndexStats
	database := filepath.Join(dataDirectory, "ele.db")
	if _, statErr := os.Stat(database); !errors.Is(statErr, fs.ErrNotExist) {
		searchIndex, openErr := index.OpenChecked(database)
		if openErr != nil {
			return openErr
		}
		indexStats, err = searchIndex.Stats(ctx)
		searchIndex.Close()
		if err != nil {
			return err
		}
	}

	bySource, byKind, byOwner := map[string]int{}, map[string]int{}, map[string]int{}
	total := 0
	for _, count := range indexStats.Counts {
		owner := count.Owner
		if owner == "" {
			owner = "(no owner)"
		}
		bySource[count.Source] += count.Entries
		byKind[count.Kind] += count.Entries
		byOwner[owner] += count.Entries
		total += count.Entries
	}
	switch {
	case total == 0:
		fmt.Fprintln(stdout, "The index is empty.")
	case indexStats.First == "":
		fmt.Fprintln(stdout, "No entry carries a date.")
	default:
		fmt.Fprintf(stdout, "Covers %s to %s.\n", indexStats.First, indexStats.Last)
	}
	table := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(table, "Entries\t%d\n", total)
	fmt.Fprintf(table, "  by source\t%s\n", listCounts(bySource))
	fmt.Fprintf(table, "  by kind\t%s\n", listCounts(byKind))
	fmt.Fprintf(table, "  by owner\t%s\n", listCounts(byOwner))
	fmt.Fprintf(table, "Sessions\t%s\n", listCounts(indexStats.Sessions))

	noteDirectory := filepath.Join(dataDirectory, "notes")
	for _, owner := range ownerMap.Owners {
		all, invalid, readErr := notes.ReadOwner(noteDirectory, owner)
		if readErr != nil {
			return readErr
		}
		superseded := map[string]bool{}
		for _, note := range all {
			for _, target := range note.Supersedes {
				superseded[target] = true
			}
		}
		kinds := map[string]int{}
		for _, note := range all {
			kind := note.Kind
			switch {
			case kind == notes.KindQuestion && superseded[note.ID]:
				kind = "answered question"
			case kind == notes.KindQuestion:
				kind = "open question"
			}
			kinds[kind]++
		}
		if invalid > 0 {
			kinds["invalid"] = invalid
		}
		fmt.Fprintf(table, "Notes, %s\t%s\n", owner, listCounts(kinds))
	}

	archiveSize, err := dirSize(filepath.Join(dataDirectory, "archive"))
	if err != nil {
		return err
	}
	indexSize := int64(0)
	for _, name := range []string{"ele.db", "ele.db-wal"} {
		if info, statErr := os.Stat(filepath.Join(dataDirectory, name)); statErr == nil {
			indexSize += info.Size()
		}
	}
	notesSize, err := dirSize(noteDirectory)
	if err != nil {
		return err
	}
	fmt.Fprintf(table, "On disk\tarchive %s, index %s, notes %s\n", humanBytes(archiveSize), humanBytes(indexSize), humanBytes(notesSize))
	return table.Flush()
}

// Each row reports its own failure, so one broken input does not hide the rest.
func writeStatus(stdout io.Writer, now time.Time) error {
	dataDirectory, err := datadir.Path()
	if err != nil {
		return err
	}
	table := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(table, "Data\t%s\n", dataDirectory)

	stamp, readErr := os.ReadFile(filepath.Join(dataDirectory, "last-sync"))
	last, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(string(stamp)))
	switch {
	case readErr != nil:
		fmt.Fprintln(table, "Last sync\tnone recorded")
	case parseErr != nil:
		fmt.Fprintln(table, "Last sync\tstamp malformed")
	case now.Before(last):
		fmt.Fprintf(table, "Last sync\t%s (in the future; check the clock)\n", last.Format(time.RFC3339))
	case now.Sub(last) < time.Minute:
		fmt.Fprintf(table, "Last sync\t%s (just now)\n", last.Format(time.RFC3339))
	default:
		fmt.Fprintf(table, "Last sync\t%s (%s ago)\n", last.Format(time.RFC3339), age(now.Sub(last)))
	}
	fmt.Fprintf(table, "Sync running\t%s\n", syncRunning(filepath.Join(dataDirectory, "archive", "transcripts")))
	fmt.Fprintf(table, "Here\t%s\n", where(dataDirectory))
	fmt.Fprintf(table, "Config\t%s\n", configState(dataDirectory))
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(table, "Plugin %s\t%v\n", pluginName, err)
	} else {
		fmt.Fprintf(table, "Plugin %s\t%s\n", pluginName, plugins(home))
	}
	if err = table.Flush(); err != nil {
		return err
	}

	log, err := os.ReadFile(filepath.Join(dataDirectory, "sync.log"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		fmt.Fprintln(stdout, "\nNo background sync has written sync.log.")
	case err != nil:
		fmt.Fprintf(stdout, "\nsync.log unreadable: %v\n", err)
	case strings.TrimSpace(string(log)) == "":
		fmt.Fprintln(stdout, "\nsync.log is empty.")
	case readErr != nil || parseErr != nil:
		lines := strings.Split(strings.TrimRight(string(log), "\n"), "\n")
		fmt.Fprintln(stdout, "\nTail of sync.log, newest last:")
		for _, line := range lines[max(0, len(lines)-syncLogTail):] {
			fmt.Fprintf(stdout, "  %s\n", line)
		}
	default:
		runs := sinceLastSync(string(log), last)
		if len(runs) == 0 {
			fmt.Fprintln(stdout, "\nsync.log holds nothing since the last successful sync.")
			return nil
		}
		// Whole runs, newest first, so every line shows under its run's header; the newest alone may be cut.
		var shown []string
		for _, run := range slices.Backward(runs) {

			if len(shown)+len(run) > syncLogTail {
				if len(shown) == 0 {
					shown = append([]string{run[0]}, run[len(run)-syncLogTail+1:]...)
				}
				break
			}
			shown = append(slices.Clone(run), shown...)
		}
		fmt.Fprintln(stdout, "\nsync.log since the last successful sync, newest last:")
		for _, line := range shown {
			fmt.Fprintf(stdout, "  %s\n", line)
		}
		if newest := runs[len(runs)-1]; len(newest) == 1 || (len(newest) == 2 && strings.HasPrefix(newest[1], syncLogLocked)) {
			fmt.Fprintln(stdout, "  (no output)")
		}
	}
	return nil
}
