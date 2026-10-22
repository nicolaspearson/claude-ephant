package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/mining"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/profile"
)

const usage = `usage:
  ele backup <dir>
  ele checkpoint density [--ref <ref>] [<root>]
  ele checkpoint log [<root>] < <record.json>
  ele checkpoint stats [<days>]
  ele context [--owner <name>] [--project <name>]
  ele context --size
  ele events [--during <skill>] [--kind <kind>] [--name <name>] [--owner <name>] [--project <name>] [--session <id>] [--since <YYYY-MM-DD|Nd>] [--limit <n>] [--no-sync]
  ele forget [--owner <name>] [--restore] <id>...
  ele handoff close [--owner <name>] [--project <name>] <id>...
  ele handoff list [--all] [--owner <name>] [--project <name>]
  ele handoff reopen [--owner <name>] [--project <name>] <id>...
  ele import mining <run>
  ele mine [--digests <dir>] [--units <file>] [--no-sync] checkpoint-runs|review-cycles
  ele mine --judgments <file> --prompt <file> --digests <dir> [--no-sync] checkpoint-runs|review-cycles
  ele mine --report [--no-sync]
  ele owner add [--move] [--move-to <owner>] [--no-sync] <name>
  ele owner add-path [--move] [--move-to <owner>] [--no-sync] <dir> <owner>
  ele owner add-project [--move] [--move-to <owner>] [--no-sync] <name> <owner>
  ele owner remove-path [--move] [--move-to <owner>] [--no-sync] <dir>
  ele owner remove-project [--move] [--move-to <owner>] [--no-sync] <name>
  ele plan check [<path>]
  ele plan done [--owner <name>] [--project <name>] <path|id>...
  ele plan list [--all] [--owner <name>] [--project <name>]
  ele plan show [<path>]
  ele plan start <path>
  ele recall [--all | --owner <name>] [--kind <k,k>] [--project <name>] [--since <YYYY-MM-DD|Nd>] [--limit <n>] [--no-sync] <query>
  ele reindex
  ele remember --kind decision|fact|handoff|question [--global] [--owner <name>] [--project <name>] [--session <id>] [--supersedes <id>] <words>
  ele session-start [--no-sync]
  ele show [--around <text>] [--context <n>] <session>
  ele show --note <id> [--context <n>]
  ele stats
  ele status
  ele stop-check [--hint-at <tokens>]
  ele sync [--background]
  ele ticket claims [--quiet] [<number>...]
  ele ticket corpus --compare <number> --expect <file> [--note] [--cli glab|gh]
  ele ticket corpus --map <number> [--tickets <numbers>] --out <directory> [--clean] [--cli glab|gh]
  ele ticket corpus --plan <file> [--plan <file>...] --out <directory> [--clean]
  ele ticket corpus --tickets <numbers> --out <directory> [--clean] [--cli glab|gh]
  ele ticket ledger record --manifest <file> --records <file> [--ledger <file>]
  ele ticket ledger rulings --manifest <file> [--ledger <file>]
  ele usage [--owner <name>] [--project <name>] [--since <YYYY-MM-DD|Nd>] [--no-sync]
  ele version`

// Set by startSync for the sync it runs, whose output goes to sync.log.
const backgroundSyncVariable = "ELE_BACKGROUND_SYNC"

var commands = map[string]func(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int{
	"backup": backup,
	"checkpoint": func(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
		return checkpointCommand(ctx, args, os.Stdin, stdout, stderr)
	},
	"context":  projectContext,
	"events":   events,
	"forget":   forget,
	"handoff":  handoffCommand,
	"import":   importCommand,
	"mine":     mine,
	"owner":    ownerCommand,
	"plan":     planCommand,
	"recall":   recall,
	"reindex":  reindex,
	"remember": remember,
	"session-start": func(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
		return sessionStart(ctx, args, os.Stdin, stdout, stderr)
	},
	"show":  show,
	"stats": stats,
	"status": func(_ context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
		return status(args, stdout, stderr)
	},
	"stop-check": func(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
		return stopCheck(ctx, args, os.Stdin, stdout, stderr)
	},
	"sync":    syncCommand,
	"ticket":  ticketCommand,
	"usage":   usageReport,
	"version": versionCommand,
}

var errSyncBusy = errors.New("another sync holds the archive lock")

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func humanBytes(count int64) string {
	const unit = 1024
	if count < unit {
		return fmt.Sprintf("%d B", count)
	}
	value, suffix := float64(count)/unit, "KiB"
	for _, next := range []string{"MiB", "GiB", "TiB"} {
		// Compare after rounding to one decimal, or 1048575 B prints as "1024.0 KiB".
		if value < unit-0.05 {
			break
		}
		value, suffix = value/unit, next
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}

func importCommand(_ context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "mining" {
		return importMining(args[1:], stdout, stderr)
	}
	return misuse("import", args, stdout, stderr)
}

func importMining(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("import mining", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() != 1 {
		return rejectCall(flags, err, stdout, stderr)
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	cache := filepath.Join(dataDirectory, "mining", "cache")
	imported, err := mining.Import(flags.Arg(0), cache)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Cached in %s:\n", cache)
	table := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "ANALYSIS\tPROMPT\tCLASSIFIER\tJUDGMENTS\tADDED")
	for _, family := range imported {
		fmt.Fprintf(table, "%s\t%s\t%s\t%d\t%d\n", family.Analysis, family.Prompt, family.Classifier[:12], family.Judgments, family.Added)
	}
	_ = table.Flush()
	fmt.Fprintln(stdout, "Watermarks:")
	table = tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	printed := map[string]bool{}
	for _, family := range imported {
		if printed[family.Analysis] {
			continue
		}
		printed[family.Analysis] = true
		watermark, watermarkErr := mining.Watermark(cache, family.Analysis)
		if watermarkErr != nil {
			fmt.Fprintf(stderr, "ele: %v\n", watermarkErr)
			return 1
		}
		fmt.Fprintf(table, "%s\t%s\n", family.Analysis, watermark.Format(time.RFC3339Nano))
	}
	_ = table.Flush()
	return 0
}

func reindex(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("reindex", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	return sync(ctx, stdout, stderr, true)
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "help" || isHelpFlag(args[0]))) {
		return writeOverview(stdout)
	}
	if command, ok := commands[args[0]]; ok {
		return command(ctx, args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "ele: unknown command %q\n%s\n", args[0], usage)
	return 2
}

func sync(ctx context.Context, stdout io.Writer, stderr io.Writer, rebuild bool) int {
	if err := syncAll(ctx, stdout, true, rebuild); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}

// syncAll returns errSyncBusy without waiting when wait is false and another sync holds the lock.
func syncAll(ctx context.Context, stdout io.Writer, wait bool, rebuild bool) error {
	background := os.Getenv(backgroundSyncVariable) != ""
	// Before anything can fail, so every line a background run writes has a time above it.
	if background {
		fmt.Fprintf(stdout, "%s%s\n", syncLogHeader, time.Now().UTC().Format(time.RFC3339Nano))
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	roots, err := profile.Discover(home)
	if err != nil {
		return err
	}
	sources := make([]archive.Source, 0, len(roots))
	for _, root := range roots {
		sources = append(sources, archive.Source{Directory: filepath.Join(root.Directory, "projects"), Profile: root.Name})
	}
	ownerMap, owners, err := ownermap.Read(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		return err
	}

	archiveDirectory := filepath.Join(dataDirectory, "archive", "transcripts")
	var unlock func()
	if wait {
		unlock, err = archive.Lock(archiveDirectory)
	} else {
		var held bool
		unlock, held, err = archive.TryLock(archiveDirectory)
		if err == nil && !held {
			return errSyncBusy
		}
	}
	if err != nil {
		return err
	}
	defer unlock()
	// A run that queued behind a success dates from here, after that success's stamp.
	if background {
		fmt.Fprintf(stdout, "%s%s\n", syncLogLocked, time.Now().UTC().Format(time.RFC3339Nano))
	}
	archived, err := archive.Sync(sources, archiveDirectory)
	if err != nil {
		return err
	}
	searchIndex, err := index.Open(filepath.Join(dataDirectory, "ele.db"))
	if err != nil {
		return err
	}
	defer searchIndex.Close()
	// Owners are resolved once, at indexing, so a changed owner map means rebuilding the whole index.
	indexedUnder, err := searchIndex.Setting(ctx, "owner-map")
	if err != nil {
		return err
	}
	ownersChanged := indexedUnder != "" && indexedUnder != owners
	// An absent version counts as older. No owner-map setting: no sync has finished, so the index is treated as new.
	version, err := searchIndex.Setting(ctx, "index-version")
	if err != nil {
		return err
	}
	versionChanged := indexedUnder != "" && version != index.Version
	problems := searchIndex.SchemaProblems()
	schemaChanged := len(problems) > 0
	if rebuild || ownersChanged || versionChanged || schemaChanged {
		if err = searchIndex.Reset(ctx); err != nil {
			return err
		}
	}
	indexed, err := searchIndex.Update(ctx, archiveDirectory, index.NewProjects(ownerMap))
	if err != nil {
		return err
	}
	remembered, err := searchIndex.UpdateClaudeMem(ctx, filepath.Join(dataDirectory, "archive", "claude-mem"), ownerMap)
	if err != nil {
		return err
	}
	noted, err := searchIndex.UpdateNotes(ctx, filepath.Join(dataDirectory, "notes"))
	if err != nil {
		return err
	}
	if err = searchIndex.SetSetting(ctx, "owner-map", owners); err != nil {
		return err
	}
	if err = searchIndex.SetSetting(ctx, "index-version", index.Version); err != nil {
		return err
	}
	stateErr := removeStaleStopState(dataDirectory, time.Now())
	stamp := filepath.Join(dataDirectory, "last-sync")
	if err := os.WriteFile(stamp, []byte(time.Now().UTC().Format(time.RFC3339Nano)+"\n"), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", stamp, err)
	}

	switch {
	case rebuild:
		fmt.Fprint(stdout, "Rebuilt the index from the archive and notes. ")
	case ownersChanged:
		fmt.Fprint(stdout, "owner-map.json changed, so the index was rebuilt. ")
	case versionChanged:
		fmt.Fprint(stdout, "The index was built by another version of ele, so it was rebuilt. ")
	case schemaChanged:
		fmt.Fprintf(stdout, "The index's tables did not match this version of ele (%s), so it was rebuilt. ", strings.Join(problems, "; "))
	}
	fmt.Fprintf(stdout, "Archived %s from %d transcripts", humanBytes(archived.Bytes), archived.Files)
	if archived.Generations > 0 {
		fmt.Fprintf(stdout, " (%d rewritten in place, kept as new generations)", archived.Generations)
	}
	fmt.Fprintf(stdout, "; indexed %d entries from %d files", indexed.Entries, indexed.Files)
	if indexed.Automated > 0 {
		fmt.Fprintf(stdout, ", left out %d entries from automated SDK sessions", indexed.Automated)
	}
	if indexed.Malformed > 0 {
		fmt.Fprintf(stdout, ", skipped %d malformed lines", indexed.Malformed)
	}
	if remembered.Files > 0 {
		fmt.Fprintf(stdout, "; re-indexed %d entries from %d claude-mem tables", remembered.Entries, remembered.Files)
		if remembered.Malformed > 0 {
			fmt.Fprintf(stdout, ", skipped %d malformed rows", remembered.Malformed)
		}
	}
	if noted.Entries > 0 {
		fmt.Fprintf(stdout, "; indexed %s", plural(noted.Entries, "note", "notes"))
	}
	fmt.Fprintln(stdout, ".")
	for _, problem := range noted.Problems {
		fmt.Fprintf(stdout, "Skipped a note: %v\n", problem)
	}
	if stateErr != nil {
		fmt.Fprintf(stdout, "Could not remove old stop-check state: %v\n", stateErr)
	}
	return nil
}

func syncCommand(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("sync", stderr)
	background := flags.Bool("background", false, "start the sync detached and return")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	if !*background {
		return sync(ctx, stdout, stderr, false)
	}
	dataDirectory, err := datadir.Path()
	if err == nil {
		err = startSync(dataDirectory)
	}
	if err != nil {
		fmt.Fprintf(stderr, "ele: could not start a background sync: %v\n", err)
		return 1
	}
	return 0
}
