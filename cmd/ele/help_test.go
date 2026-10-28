package main

import (
	"bytes"
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func runSplit(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), args, &stdout, &stderr)
	return stdout.String(), stderr.String(), code
}

func TestRunPrintsACommandsHelpFromItsFlags(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		stdout, stderr, code := runSplit(t, "handoff", "list", flag)
		require.Equal(t, 0, code)
		require.Empty(t, stderr)
		require.Equal(t, `ele handoff list [--all] [--owner <name>] [--project <name>]
Pending handoffs for the project, newest first.
  --all              add closed and superseded handoffs
  --owner <name>     list this owner's handoffs instead of the resolved one's
  --project <name>   list this project's handoffs instead of the working directory's
`, stdout)
	}
}

func TestRunPrintsTheOverviewNamingEveryCommand(t *testing.T) {
	for _, args := range [][]string{{}, {"--help"}, {"-h"}, {"help"}} {
		stdout, stderr, code := runSplit(t, args...)
		require.Equal(t, 0, code, args)
		require.Empty(t, stderr, args)
		for name := range commands {
			require.Regexp(t, `(?m)^  `+name+`( \S+)? {2,}\S`, stdout, args)
		}
		require.Contains(t, stdout, "\n  ele remember --kind decision|fact|handoff|question|reminder [--done-file <path>] [--due <YYYY-MM-DD[THH:MM]>] [--every <interval>] [--everywhere | --global] [--owner <name>] [--project <name>] [--session <id>] [--supersedes <id>...] <words>\n", "every flag shows without a second call")
	}
	for _, row := range regexp.MustCompile(`(?m)^  ([a-z-]+)(?: [a-z-]+)? {2,}\S`).FindAllStringSubmatch(overview, -1) {
		require.Contains(t, commands, row[1], "the overview names only commands that run")
	}
}

func TestRunAnswersHelpForEveryCommand(t *testing.T) {
	invocations := [][]string{{"checkpoint", "density"}, {"checkpoint", "log"}, {"checkpoint", "stats"}, {"config", "get"}, {"config", "list"}, {"config", "set"}, {"config", "unset"}, {"handoff", "close"}, {"handoff", "list"}, {"handoff", "reopen"}, {"owner", "add"}, {"owner", "add-path"}, {"owner", "add-project"}, {"owner", "remove-path"}, {"owner", "remove-project"}, {"plan", "done"}, {"plan", "list"}, {"plan", "show"}, {"plan", "start"}, {"reminder", "close"}, {"reminder", "list"}, {"reminder", "reopen"}, {"ticket", "claims"}, {"ticket", "corpus"}, {"ticket", "ledger"}}
	for name := range commands {
		if !slices.Contains([]string{"checkpoint", "config", "handoff", "owner", "plan", "reminder", "ticket"}, name) {
			invocations = append(invocations, []string{name})
		}
	}
	for _, invocation := range invocations {
		full := strings.Join(invocation, " ")
		stdout, stderr, code := runSplit(t, append(invocation, "--help")...)
		require.Equal(t, 0, code, full)
		require.Empty(t, stderr, full)
		require.Regexp(t, `^(ele `+regexp.QuoteMeta(full)+`( .*)?\n)+[A-Z][^\n]*\.\n(  --\S.*\n)*$`, stdout, full)
	}
}

func TestRunListsEachRecallFlagInItsHelp(t *testing.T) {
	stdout, _, code := runSplit(t, "recall", "-h")
	require.Equal(t, 0, code)
	for _, line := range []string{
		`--all {21}search every owner`,
		`--kind <k,k> {14}comma-separated kinds: note`,
		`--limit <n> {15}most matches to return`,
		`--no-sync {17}search without syncing first`,
		`--owner <name> {12}search this owner`,
		`--project <name> {10}only this project`,
		`--since <YYYY-MM-DD\|Nd>   only entries from this date`,
	} {
		require.Regexp(t, `(?m)^  `+line, stdout)
	}
}

func TestRunPrintsSynopsisAndPurposeForACommandWithoutFlags(t *testing.T) {
	stdout, stderr, code := runSplit(t, "stats", "--help")
	require.Equal(t, 0, code)
	require.Empty(t, stderr)
	require.Equal(t, "ele stats\nIndex counts and disk use.\n", stdout)
}

func TestRunRejectsAMisusedCommandWithOnlyItsHelp(t *testing.T) {
	stdout, stderr, code := runSplit(t, "backup")
	require.Equal(t, 2, code)
	require.Empty(t, stdout)
	require.Equal(t, "ele backup <dir>\n"+purposes["backup"]+"\n", stderr)
}

func TestRunRejectsAMisusedGroupWithItsSynopses(t *testing.T) {
	stdout, stderr, code := runSplit(t, "handoff", "bogus")
	require.Equal(t, 2, code)
	require.Empty(t, stdout)
	require.Equal(t, `ele: unknown handoff subcommand "bogus"
ele handoff close [--owner <name>] [--project <name>] <id>...
ele handoff list [--all] [--owner <name>] [--project <name>]
ele handoff reopen [--owner <name>] [--project <name>] <id>...
`, stderr)

	for _, flag := range []string{"--help", "-h"} {
		stdout, stderr, code = runSplit(t, "handoff", flag)
		require.Equal(t, 0, code, flag)
		require.Empty(t, stderr, flag)
		require.True(t, strings.HasPrefix(stdout, "ele handoff close "), stdout)
	}
}

func TestRunRejectsAnUnknownCommandWithTheSynopsisList(t *testing.T) {
	stdout, stderr, code := runSplit(t, "bogus")
	require.Equal(t, 2, code)
	require.Empty(t, stdout)
	require.Equal(t, "ele: unknown command \"bogus\"\n"+usage+"\n", stderr)
}

func TestRunRejectsAnUnknownFlagWithThatCommandsHelp(t *testing.T) {
	stdout, stderr, code := runSplit(t, "recall", "--bogus", "pydantic")
	require.Equal(t, 2, code)
	require.Empty(t, stdout)
	require.True(t, strings.HasPrefix(stderr, "flag provided but not defined: -bogus\nele recall [--all"), stderr)
	require.Contains(t, stderr, "  --limit <n>")
	require.NotContains(t, stderr, "ele backup")
}
