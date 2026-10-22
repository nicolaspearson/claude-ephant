package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
)

const overview = `ele: memory across Claude Code sessions.

Notes
  remember        save a decision, fact, question or handoff
  context         the project's live notes and pending handoffs, unbudgeted
  handoff list    pending handoffs; --all adds closed and superseded
  handoff close   mark handoffs picked up or dropped
  handoff reopen  make closed handoffs pending again
  forget          move mistaken notes out of the index; --restore brings them back

Plans
  plan start      work through a plan file, one item per session
  plan show       a plan's counts, previous item, blocked items and next item
  plan list       active plans; --all adds done and superseded
  plan done       close a plan once its last item is done
  plan check      a plan's warnings: blockers, dates, inbox, missing done-when, name clashes, ready to close

History
  recall          search past sessions and notes
  show            a session's entries, or a note's discussion
  events          agent, command, hook, skill and other session events
  usage           tool calls, injected bytes and tokens across sessions

Index
  sync            archive transcripts and update the index
  reindex         rebuild the index from the archive and notes
  status          data directory, sync state, owner here, plugins
  stats           index counts and disk use
  version         the release tag this binary was built from, the version Go stamped, or devel
  backup          copy the archive, notes, forgotten notes, mining runs, owner map and index elsewhere
  owner           add an owner, or add or remove a path rule or project entry

Mining
  mine            list and digest units to judge, or store judgments
  import          load a mining run

Checkpoint
  checkpoint density  comment density of the diff against its file and its peers
  checkpoint log      append a checkpoint's record to the log
  checkpoint stats    what checkpoints found and which stage found it; <days> back, default 30

Ticket
  ticket claims   open tickets mistyped as a task or granting themselves execution
  ticket corpus   an audit's corpus from a map's tickets, a list of tickets or a plan's items, or read a ticket back
  ticket ledger   record what an audit ran against each item, or show earlier rulings

Hooks
  session-start   the SessionStart block
  stop-check      ask for notes once per stretch of work; suggest /compact or a handoff as context grows

ele <command> --help says what each flag does.`

var purposes = map[string]string{
	"backup":               "Copy the archive, notes, forgotten notes, mining runs, owner map and the index into a new or empty directory.",
	"checkpoint density":   "Per file in the diff since --ref, the share of added lines that are comments, beside the file's own share and the median of its nearest peers of the same suffix; flags the outliers.",
	"checkpoint log":       "Read a checkpoint's record on stdin, refuse one the stats would misread, and append it with its date, HEAD, repository and duration since the first freeze.",
	"checkpoint stats":     "What the log's checkpoints found and which stage found it over the last <days> days, 30 by default.",
	"context":              "The project's live notes and pending handoffs, with no budget.",
	"events":               "Agent, command, hook, skill and other events from session transcripts.",
	"forget":               "Move mistaken notes out of the index into forgotten/, or bring them back with --restore.",
	"handoff close":        "Mark pending handoffs picked up or dropped, so the start block stops showing them.",
	"handoff list":         "Pending handoffs for the project, newest first.",
	"handoff reopen":       "Make closed handoffs pending, so the next startup or /clear shows them.",
	"import mining":        "Cache a mining run's judgments and report each analysis's watermark.",
	"mine":                 "List an analysis's unjudged units as JSON Lines, store judgments made from their digests, or report figures from the stored judgments.",
	"owner add":            "List a new owner in owner-map.json.",
	"owner add-path":       "Map a directory and everything under it to a listed owner, printing the projects that change owner.",
	"owner add-project":    "Map a project to a listed owner, ahead of any path rule, printing the projects that change owner.",
	"owner remove-path":    "Drop a path rule, printing the projects that change owner.",
	"owner remove-project": "Drop a project entry, printing the projects that change owner.",
	"plan check":           "A plan's warnings, each with its line: a blocker done or missing, an after: date passed, an as of line past 14 days, an inbox item past 7 days or undated, an open action with no done when line, a name another plan holds, readiness to close; with no path, every active plan. Exits 0 on warnings.",
	"plan done":            "Close active plans once their last item is done.",
	"plan list":            "Active plans for the project, newest first.",
	"plan show":            "A plan's counts, previous item, blocked items and next item; with no path, the project's active plan.",
	"plan start":           "Start working through a plan file in this checkout.",
	"recall":               "Search past sessions and notes.",
	"reindex":              "Rebuild the index from the archive and notes.",
	"remember":             "Save a note that later sessions start with.",
	"session-start":        "Print the SessionStart block and start a background sync.",
	"show":                 "A session's entries, or the discussion a note came from.",
	"stats":                "Index counts and disk use.",
	"status":               "Data directory, sync state, the owner here, plugins and the sync log.",
	"stop-check":           "Read a Stop hook's input. Block once per stretch between checkpoints to ask for notes on unproposed work, and suggest /compact or a handoff once per 100k of context past --hint-at.",
	"sync":                 "Archive new transcripts and update the index.",
	"ticket claims":        "Flag open GitHub tickets labelled wayfinder:map that grant themselves execution, and wayfinder:task tickets naming a source path; with numbers, only those tickets.",
	"ticket corpus":        "Write each ticket of a map or a list, or each Actions and Inbox item of one or more plans, twice, whitespace collapsed and raw, with a manifest of keys and hashes and a table of each ticket's blockers; with --compare, read a ticket back after a write.",
	"ticket ledger":        "With record, append a line per audited item carrying the hash of the text the audit read, refusing the whole set on a key or method the audit did not read or run; with rulings, print each earlier ruling on a manifest item and whether its text has changed since.",
	"usage":                "Tool calls, injected bytes and tokens across sessions, main and subagents.",
	"version":              "The release tag stamped at build; otherwise the version Go stamped, or devel where it stamped none.",
}

func isHelpFlag(argument string) bool {
	return slices.Contains([]string{"--h", "--help", "-h", "-help"}, argument)
}

// A group such as `ele handoff` has no FlagSet, so --help and a bad subcommand are caught here.
func misuse(group string, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) > 0 && isHelpFlag(args[0]) {
		writeCommandHelp(stdout, group, nil)
		return 0
	}
	if len(args) > 0 {
		fmt.Fprintf(stderr, "ele: unknown %s subcommand %q\n", group, args[0])
	}
	writeCommandHelp(stderr, group, nil)
	return 2
}

// Empty Usage: flag prints only the parse error; rejectCall adds the help.
func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	return flags
}

func rejectCall(flags *flag.FlagSet, err error, stdout io.Writer, stderr io.Writer) int {
	if errors.Is(err, flag.ErrHelp) {
		writeCommandHelp(stdout, flags.Name(), flags)
		return 0
	}
	writeCommandHelp(stderr, flags.Name(), flags)
	return 2
}

func synopses(name string) []string {
	var matched []string
	for _, line := range strings.Split(usage, "\n")[1:] {
		synopsis := strings.TrimSpace(line)
		if synopsis == "ele "+name || strings.HasPrefix(synopsis, "ele "+name+" ") {
			matched = append(matched, synopsis)
		}
	}
	return matched
}

func writeCommandHelp(w io.Writer, name string, flags *flag.FlagSet) {
	matched := synopses(name)
	fmt.Fprintln(w, strings.Join(matched, "\n"))
	if purpose := purposes[name]; purpose != "" {
		fmt.Fprintln(w, purpose)
	}
	if flags == nil {
		return
	}
	var names, descriptions []string
	flags.VisitAll(func(defined *flag.Flag) {
		placeholder, description := flag.UnquoteUsage(defined)
		flagName := "--" + defined.Name
		if placeholder != "" {
			if match := regexp.MustCompile(`--` + regexp.QuoteMeta(defined.Name) + ` ([^\s\]]+)`).FindStringSubmatch(strings.Join(matched, "\n")); match != nil {
				placeholder = match[1]
			} else {
				placeholder = "<" + placeholder + ">"
			}
			flagName += " " + placeholder
		}
		names = append(names, flagName)
		descriptions = append(descriptions, description)
	})
	width := 0
	for _, flagName := range names {
		width = max(width, len(flagName))
	}
	for at, flagName := range names {
		fmt.Fprintf(w, "  %-*s   %s\n", width, flagName, descriptions[at])
	}
}

func writeOverview(stdout io.Writer) int {
	fmt.Fprintf(stdout, "%s\n\n%s\n", overview, usage)
	return 0
}
