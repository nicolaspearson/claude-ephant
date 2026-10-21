package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/ticket"
)

func ticketCommand(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	subcommands := map[string]func(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int{
		"claims": ticketClaims,
		"corpus": ticketCorpus,
		"ledger": ticketLedger,
	}
	if len(args) == 0 || subcommands[args[0]] == nil {
		return misuse("ticket", args, stdout, stderr)
	}
	return subcommands[args[0]](ctx, args[1:], stdout, stderr)
}

// Quoted paths arrive with ~ unexpanded.
func expandHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	return home + strings.TrimPrefix(path, "~")
}

func ticketClaims(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("ticket claims", stderr)
	quiet := flags.Bool("quiet", false, "print nothing when no ticket is flagged")
	positional, err := parseInterspersed(flags, args)
	if err != nil {
		return rejectCall(flags, err, stdout, stderr)
	}
	var numbers []int
	for _, argument := range positional {
		number, err := strconv.Atoi(argument)
		if err != nil {
			fmt.Fprintf(stderr, "ele: ticket %q is not a number\n", argument)
			return 2
		}
		numbers = append(numbers, number)
	}
	ticket.WriteClaims(ctx, stdout, stderr, ".", numbers, *quiet)
	return 0
}

func ticketCorpus(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("ticket corpus", stderr)
	clean := flags.Bool("clean", false, "empty --out first")
	cli := flags.String("cli", "", "the `glab|gh` tracker CLI to use, over the one the remote names")
	compare := flags.String("compare", "", "the ticket `number` to read back after a write")
	expect := flags.String("expect", "", "with --compare: the `file` holding the text as written")
	mapNumber := flags.String("map", "", "the map ticket's `number`; its children are discovered")
	note := flags.Bool("note", false, "with --compare: match a comment, not the body")
	out := flags.String("out", "", "the writable `directory` to build the corpus in")
	plan := flags.String("plan", "", "an ele plan `file`; each Actions and Inbox item stands for a ticket")
	tickets := flags.String("tickets", "", "comma-separated ticket `numbers`")
	positional, err := parseInterspersed(flags, args)
	if err != nil || len(positional) > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	if *cli != "" && *cli != "glab" && *cli != "gh" {
		fmt.Fprintf(stderr, "ele: --cli %q is not glab or gh\n", *cli)
		return 2
	}
	mapped := 0
	if *mapNumber != "" {
		if mapped, err = strconv.Atoi(strings.TrimSpace(*mapNumber)); err != nil {
			fmt.Fprintf(stderr, "ele: --map %q is not a number\n", *mapNumber)
			return 2
		}
	}
	comparing := false
	flags.Visit(func(set *flag.Flag) { comparing = comparing || set.Name == "compare" })
	if comparing {
		number, err := strconv.Atoi(strings.TrimSpace(*compare))
		if err != nil {
			fmt.Fprintf(stderr, "ele: --compare %q is not a number\n", *compare)
			return 2
		}
		if *expect == "" || *out != "" || *plan != "" || mapped != 0 || *tickets != "" || *clean {
			fmt.Fprintln(stderr, "ele: --compare takes --expect and no corpus options")
			return 2
		}
		return ticket.Compare(ctx, stdout, stderr, ticket.SystemCommands(), *cli, number, expandHome(*expect), *note)
	}
	if *note || *expect != "" {
		fmt.Fprintln(stderr, "ele: --note and --expect go with --compare")
		return 2
	}
	if (*plan != "") == (mapped != 0 || *tickets != "") {
		fmt.Fprintln(stderr, "ele: one of --plan, or --map and --tickets, is required")
		return 2
	}
	if *out == "" {
		fmt.Fprintln(stderr, "ele: --out is required")
		return 2
	}
	var numbers []int
	for listed := range strings.SplitSeq(*tickets, ",") {
		if strings.TrimSpace(listed) == "" {
			continue
		}
		number, err := strconv.Atoi(strings.TrimSpace(listed))
		if err != nil {
			fmt.Fprintf(stderr, "ele: ticket %q is not a number\n", listed)
			return 2
		}
		numbers = append(numbers, number)
	}
	options := ticket.CorpusOptions{CLI: *cli, Clean: *clean, Map: mapped, Out: expandHome(*out), Plan: expandHome(*plan), Tickets: numbers}
	return ticket.Corpus(ctx, stdout, stderr, ticket.SystemCommands(), options)
}

func ticketLedger(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("ticket ledger", stderr)
	ledgerPath := flags.String("ledger", "", "the ledger `file`, by default ~/.claude/logs/ticket-audits.jsonl")
	manifestPath := flags.String("manifest", "", "the manifest.json `file` the ticket corpus wrote")
	recordsPath := flags.String("records", "", "record: the JSON list of records, as a `file`")
	positional, err := parseInterspersed(flags, args)
	if err != nil || len(positional) != 1 || positional[0] != "record" && positional[0] != "rulings" {
		return rejectCall(flags, err, stdout, stderr)
	}
	if *manifestPath == "" {
		fmt.Fprintln(stderr, "ele: ticket ledger needs --manifest")
		return 2
	}
	if *ledgerPath == "" {
		if *ledgerPath, err = ticket.LedgerPath(); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
	}
	if positional[0] == "rulings" {
		err = ticket.WriteRulings(stdout, expandHome(*manifestPath), expandHome(*ledgerPath))
	} else if *recordsPath == "" {
		fmt.Fprintln(stderr, "ele: ticket ledger record needs --records")
		return 2
	} else {
		err = ticket.Record(stdout, expandHome(*manifestPath), expandHome(*recordsPath), expandHome(*ledgerPath), time.Now())
	}
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}
