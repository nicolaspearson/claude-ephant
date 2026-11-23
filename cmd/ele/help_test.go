package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
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

func usageLines() []string {
	var lines []string
	for _, line := range strings.Split(usage, "\n")[1:] {
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func usagePaths() [][]string {
	word := regexp.MustCompile(`^[a-z][a-z-]*$`)
	var paths [][]string
	for _, line := range usageLines() {
		var path []string
		for _, field := range strings.Fields(line)[1:] {
			if !word.MatchString(field) {
				break
			}
			path = append(path, field)
		}
		if !slices.ContainsFunc(paths, func(seen []string) bool { return slices.Equal(seen, path) }) {
			paths = append(paths, path)
		}
	}
	return paths
}

func TestCommandsDocHoldsEveryUsageLine(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "docs", "commands.md"))
	require.NoError(t, err)
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	synopses := usageLines()
	for _, synopsis := range synopses {
		require.True(t, slices.Contains(lines, synopsis), "%q is in usage but not a line of docs/commands.md", synopsis)
	}
	fenced := false
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
			continue
		}
		if fenced && strings.HasPrefix(line, "ele ") {
			require.True(t, slices.Contains(synopses, line), "%q is in docs/commands.md's code fences but not in usage", line)
		}
	}
}

func TestRunAnswersHelpForEveryCommand(t *testing.T) {
	paths := usagePaths()
	for name := range commands {
		require.True(t, slices.ContainsFunc(paths, func(path []string) bool { return len(path) > 0 && path[0] == name }), "%s has no usage line", name)
	}
	for _, invocation := range paths {
		full := strings.Join(invocation, " ")
		// A path with no purposes entry, such as ticket ledger record, answers under its nearest parent's name.
		name := full
		for depth := len(invocation); depth > 0; depth-- {
			if prefix := strings.Join(invocation[:depth], " "); purposes[prefix] != "" {
				name = prefix
				break
			}
		}
		stdout, stderr, code := runSplit(t, append(invocation, "--help")...)
		require.Equal(t, 0, code, full)
		require.Empty(t, stderr, full)
		require.Regexp(t, `^(ele `+regexp.QuoteMeta(name)+`( .*)?\n)+[A-Z][^\n]*\.\n(  --\S.*\n)*$`, stdout, full)
		synopsis, _, _ := strings.Cut(stdout, "\n"+purposes[name])
		for _, row := range regexp.MustCompile(`(?m)^  (--\S+ \S+) {2,}`).FindAllStringSubmatch(stdout, -1) {
			require.Regexp(t, regexp.QuoteMeta(row[1])+`(\]|\s|$)`, synopsis, "%s: a flag's placeholder is its synopsis's, whole", full)
			require.Equal(t, strings.Count(row[1], "["), strings.Count(row[1], "]"), "%s: %s cut inside its brackets", full, row[1])
		}
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
		`--since <YYYY-MM-DD\|Nd>   only entries from this local day`,
	} {
		require.Regexp(t, `(?m)^  `+line, stdout)
	}
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

func TestRunPrintsSynopsisAndPurposeForACommandWithoutFlags(t *testing.T) {
	stdout, stderr, code := runSplit(t, "stats", "--help")
	require.Equal(t, 0, code)
	require.Empty(t, stderr)
	require.Equal(t, "ele stats\nIndex counts and disk use.\n", stdout)
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

func TestRunReadsHelpAndVersionTheConventionalWays(t *testing.T) {
	version, _, versionCode := runSplit(t, "version")
	require.Equal(t, 0, versionCode)
	for _, flag := range []string{"--version", "-v"} {
		stdout, stderr, code := runSplit(t, flag)
		require.Equal(t, 0, code, stderr)
		require.Equal(t, version, stdout, flag)
	}
	for _, command := range [][]string{{"recall"}, {"checkpoint", "stats"}} {
		want, _, _ := runSplit(t, append(slices.Clone(command), "--help")...)
		stdout, stderr, code := runSplit(t, append([]string{"help"}, command...)...)
		require.Equal(t, 0, code, stderr)
		require.Equal(t, want, stdout, command)
	}
	overview, _, _ := runSplit(t, "help")
	for _, asked := range []string{"--help", "-h", "-v", "help"} {
		stdout, stderr, code := runSplit(t, "help", asked)
		require.Equal(t, 0, code, stderr)
		require.Equal(t, overview, stdout, asked)
	}
	_, stderr, unknownCode := runSplit(t, "help", "bogus")
	require.Equal(t, 2, unknownCode)
	require.Equal(t, "ele: unknown command \"bogus\"\n"+usage+"\n", stderr)
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
