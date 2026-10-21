package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

const bytesPerToken = 4

func cachedShare(tokens dto.UsageTokens) string {
	input := tokens.Input + tokens.CacheWrite + tokens.CacheRead
	if input == 0 {
		return "no input"
	}
	return fmt.Sprintf("%.1f%%", float64(tokens.CacheRead)*100/float64(input))
}

func grouped(count int) string {
	digits := strconv.Itoa(count)
	for at := len(digits) - 3; at > 0; at -= 3 {
		digits = digits[:at] + "," + digits[at:]
	}
	return digits
}

func usageReport(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("usage", stderr)
	noSync := flags.Bool("no-sync", false, "read without syncing first")
	owner := flags.String("owner", "", "only this owner")
	project := flags.String("project", "", "only this project")
	since := flags.String("since", "", "only from this date (YYYY-MM-DD) or this many days back (7d)")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	from, err := parseSince(*since, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 2
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	ownerMap, err := ownermap.Load(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	if *owner != "" && !slices.Contains(ownerMap.Owners, *owner) {
		fmt.Fprintf(stderr, "ele: unknown owner %q; owner-map.json lists %s\n", *owner, strings.Join(ownerMap.Owners, ", "))
		return 2
	}

	searchIndex, err := syncedIndex(ctx, *noSync, stderr, "these figures")
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	defer searchIndex.Close()
	report, err := searchIndex.Usage(ctx, dto.UsageQuery{Owner: *owner, Project: *project, Since: from})
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	writeUsage(stdout, report, *owner, *project, from)
	return 0
}

func writeUsage(stdout io.Writer, report dto.Usage, owner string, project string, since string) {
	scope := ""
	if owner != "" {
		scope += " for " + owner
	}
	if project != "" {
		scope += " in " + project
	}
	if since != "" {
		scope += " since " + since
	}
	fmt.Fprintf(stdout, "%s, %s%s.\n", plural(report.Sessions, "session", "sessions"), plural(report.Prompts, "prompt", "prompts"), scope)
	if report.Sessions == 0 {
		return
	}

	table := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	if len(report.Tools) > 0 {
		fmt.Fprintln(table, "\nTool calls\tmain\tsubagents")
	}
	for _, tool := range report.Tools {
		fmt.Fprintf(table, "%s\t%d\t%d\n", tool.Name, tool.Main, tool.Subagent)
	}
	if len(report.Injections) > 0 {
		fmt.Fprintln(table, "\nInjected into context\tmain\tsubagents\t≈ tokens per session, main + subagents")
	}
	for _, injection := range report.Injections {
		tokens := "<1"
		if perSession := (injection.Main + injection.Subagent) / bytesPerToken / report.Sessions; perSession > 0 {
			tokens = strconv.Itoa(perSession)
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", injection.Name, humanBytes(int64(injection.Main)), humanBytes(int64(injection.Subagent)), tokens)
	}
	main, subagents := report.MainTokens, report.SubagentTokens
	tokensReported := main != (dto.UsageTokens{}) || subagents != (dto.UsageTokens{})
	if tokensReported {
		fmt.Fprintln(table, "\nTokens\tmain\tsubagents\tper session, main + subagents")
		for _, row := range []struct {
			main     int
			name     string
			subagent int
		}{
			{main.Input, "input, uncached", subagents.Input},
			{main.CacheWrite, "cache write", subagents.CacheWrite},
			{main.CacheRead, "cache read", subagents.CacheRead},
			{main.Output, "output", subagents.Output},
		} {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", row.name, grouped(row.main), grouped(row.subagent), grouped((row.main+row.subagent)/report.Sessions))
		}
	}
	table.Flush()
	if tokensReported {
		fmt.Fprintf(stdout, "\nInput read from cache: main %s, subagents %s.\n", cachedShare(main), cachedShare(subagents))
	}

	// Sessions are counted from prompts, so Prompts is never 0 here.
	fmt.Fprintf(stdout, "\nReplies: %d characters in %d text blocks, %d per prompt.\n", report.ReplyCharacters, report.Replies, report.ReplyCharacters/report.Prompts)
}
