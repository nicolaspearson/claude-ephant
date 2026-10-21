package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/checkpoint"
)

func checkpointCommand(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	subcommands := map[string]func(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int{
		"density": checkpointDensity,
		"log":     checkpointLog,
		"stats":   checkpointStats,
	}
	if len(args) == 0 || subcommands[args[0]] == nil {
		return misuse("checkpoint", args, stdout, stderr)
	}
	return subcommands[args[0]](ctx, args[1:], stdin, stdout, stderr)
}

func checkpointDensity(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("checkpoint density", stderr)
	ref := flags.String("ref", "HEAD", "measure the diff against this `ref`")
	positional, err := parseInterspersed(flags, args)
	if err != nil || len(positional) > 1 {
		return rejectCall(flags, err, stdout, stderr)
	}
	directory, named := ".", "the working directory"
	if len(positional) == 1 && positional[0] != "" {
		directory, named = positional[0], positional[0]
	}
	root, err := checkpoint.Root(ctx, directory)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %s is not inside a git repository\n", named)
		return 2
	}
	if err := checkpoint.Density(ctx, stdout, root, *ref); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}

func checkpointLog(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("checkpoint log", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() > 1 {
		return rejectCall(flags, err, stdout, stderr)
	}
	directory, named := ".", "the working directory"
	if flags.Arg(0) != "" {
		directory, named = flags.Arg(0), flags.Arg(0)
	}
	root, err := checkpoint.Root(ctx, directory)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %s is not inside a git repository\n", named)
		return 2
	}
	record, problems, err := checkpoint.Problems(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "ele: the record is not JSON: %v\n", err)
		return 2
	}
	if len(problems) > 0 {
		fmt.Fprintln(stderr, "ele: record refused:")
		for _, problem := range problems {
			fmt.Fprintf(stderr, "  %s\n", problem)
		}
		return 2
	}
	logPath, err := checkpoint.LogPath()
	if err == nil {
		err = checkpoint.Append(ctx, root, record, logPath, time.Now())
	}
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "logged: %s\n", root)
	return 0
}

func checkpointStats(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("checkpoint stats", stderr)
	baseline := flags.Bool("baseline", false, "also report the checkpoints mined from transcripts before the log, whatever the window")
	positional, err := parseInterspersed(flags, args)
	if err != nil || len(positional) > 1 {
		return rejectCall(flags, err, stdout, stderr)
	}
	days := 30
	if len(positional) == 1 {
		if days, err = strconv.Atoi(positional[0]); err != nil {
			fmt.Fprintf(stderr, "ele: days %q is not a whole number\n", positional[0])
			return 2
		}
	}
	logPath, err := checkpoint.LogPath()
	if err == nil {
		err = checkpoint.WriteStats(stdout, logPath, time.Now().UTC().AddDate(0, 0, -days))
	}
	if err == nil && *baseline {
		var baselinePath string
		if baselinePath, err = checkpoint.BaselinePath(); err == nil {
			if err = checkpoint.WriteBaseline(stdout, baselinePath); errors.Is(err, checkpoint.ErrNoBaseline) {
				fmt.Fprintf(stdout, "\nNo baseline at %s.\n", baselinePath)
				return 1
			}
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}
