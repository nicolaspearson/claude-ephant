package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/checkpoint"
)

type hookInput struct {
	SessionID      string          `json:"session_id"`
	StopHookActive json.RawMessage `json:"stop_hook_active"`
	TranscriptPath string          `json:"transcript_path"`
}

func checkpointCommand(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	subcommands := map[string]func(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int{
		"base":     checkpointBase,
		"check":    checkpointCheck,
		"close":    checkpointClose,
		"density":  checkpointDensity,
		"freeze":   checkpointFreeze,
		"gate":     checkpointGate,
		"hash":     checkpointHash,
		"log":      checkpointLog,
		"optout":   checkpointOptout,
		"record":   checkpointRecord,
		"snapshot": checkpointSnapshot,
		"stats":    checkpointStats,
	}
	if len(args) == 0 || subcommands[args[0]] == nil {
		return misuse("checkpoint", args, stdout, stderr)
	}
	return subcommands[args[0]](ctx, args[1:], stdin, stdout, stderr)
}

func checkpointBase(ctx context.Context, args []string, _ io.Reader, stdout io.Writer, stderr io.Writer) int {
	return withCheckpointRoot(ctx, "checkpoint base", args, stdout, stderr, func(root string) int {
		base, err := checkpoint.Base(ctx, root)
		if err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, base)
		return 0
	})
}

func checkpointCheck(ctx context.Context, args []string, _ io.Reader, stdout io.Writer, stderr io.Writer) int {
	return withCheckpointRoot(ctx, "checkpoint check", args, stdout, stderr, func(root string) int {
		if !checkpoint.Reviewed(ctx, root) {
			fmt.Fprintf(stdout, "not covered: %s has changed since the last checkpoint, or none ran\n", root)
			return 1
		}
		fmt.Fprintf(stdout, "covered: every changed path in %s is what the last checkpoint reviewed\n", root)
		return 0
	})
}

func checkpointClose(ctx context.Context, args []string, _ io.Reader, stdout io.Writer, stderr io.Writer) int {
	return withCheckpointRoot(ctx, "checkpoint close", args, stdout, stderr, func(root string) int {
		if err := checkpoint.Close(ctx, root, checkpoint.Session()); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "closed: %s\n", root)
		return 0
	})
}

func checkpointDensity(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("checkpoint density", stderr)
	ref := flags.String("ref", "HEAD", "measure the diff against this `ref`")
	positional, err := parseInterspersed(flags, args)
	if err != nil || len(positional) > 1 {
		return rejectCall(flags, err, stdout, stderr)
	}
	root, ok := checkpointRoot(ctx, positional, stderr)
	if !ok {
		return 2
	}
	if err := checkpoint.Density(ctx, stdout, root, *ref); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}

// checkpointFreeze parses by hand: --base and --start come after the two positionals and before --, which the flag package cannot express.
func checkpointFreeze(ctx context.Context, args []string, _ io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("checkpoint freeze", stderr)
	if len(args) > 0 && isHelpFlag(args[0]) {
		return rejectCall(flags, flag.ErrHelp, stdout, stderr)
	}
	if len(args) < 2 {
		return rejectCall(flags, nil, stdout, stderr)
	}
	options := checkpoint.FreezeOptions{Against: "HEAD", OutDir: args[1], Session: checkpoint.Session()}
	rest := args[2:]
	for len(rest) > 0 {
		if rest[0] == "--start" {
			options.StartsRun, rest = true, rest[1:]
			continue
		}
		if rest[0] != "--base" {
			break
		}
		if len(rest) < 2 {
			return rejectCall(flags, nil, stdout, stderr)
		}
		options.Against, rest = rest[1], rest[2:]
	}
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
	}
	if info, err := os.Stat(options.OutDir); len(rest) == 0 || err != nil || !info.IsDir() {
		return rejectCall(flags, nil, stdout, stderr)
	}
	options.Paths = rest
	return withCheckpointRoot(ctx, "checkpoint freeze", args[:1], stdout, stderr, func(root string) int {
		options.Root = root
		summary, err := checkpoint.Freeze(ctx, options)
		if err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, summary)
		return 0
	})
}

// checkpointGate is the Stop hook. However it exits, the turn is over, so the next prompt takes a fresh baseline; clearing it is also what lets the snapshot tell a new turn from a message sent partway through one.
func checkpointGate(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("checkpoint gate", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	input, root, stateFile, ok := readCheckpointHook(ctx, stdin)
	if !ok {
		return 0
	}
	defer func() { _ = os.Remove(stateFile) }()
	if checkpointGateOff() {
		return 0
	}
	stopHookActive := string(input.StopHookActive) == "true" || string(input.StopHookActive) == `"true"`
	reason := checkpoint.Gate(ctx, checkpoint.GateInput{Root: root, Session: input.SessionID, StopHookActive: stopHookActive, TranscriptPath: input.TranscriptPath}, stateFile)
	if reason == "" {
		return 0
	}
	checkpoint.HookLog("block", "checkpoint", filepath.Base(root), time.Now())
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(stopOutput{Decision: "block", Reason: reason}); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}

// checkpointGateOff leaves the gate on where config cannot be read, as a config.json that does not parse.
func checkpointGateOff() bool {
	scope, err := resolveConfigScope("", true)
	if err != nil {
		return false
	}
	value, err := loadConfig(scope.dataDirectory, io.Discard).Value("checkpoint_gate", scope.owner)
	return err == nil && value.Value == false
}

func checkpointHash(ctx context.Context, args []string, _ io.Reader, stdout io.Writer, stderr io.Writer) int {
	return withCheckpointRoot(ctx, "checkpoint hash", args, stdout, stderr, func(root string) int {
		state, err := checkpoint.WriteState(ctx, root)
		if err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, state.Fingerprint())
		return 0
	})
}

func checkpointLog(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("checkpoint log", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() > 1 {
		return rejectCall(flags, err, stdout, stderr)
	}
	root, ok := checkpointRoot(ctx, flags.Args(), stderr)
	if !ok {
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

func checkpointOptout(ctx context.Context, args []string, _ io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("checkpoint optout", stderr)
	if len(args) > 0 && isHelpFlag(args[0]) {
		return rejectCall(flags, flag.ErrHelp, stdout, stderr)
	}
	if len(args) < 2 {
		return rejectCall(flags, nil, stdout, stderr)
	}
	return withCheckpointRoot(ctx, "checkpoint optout", args[:1], stdout, stderr, func(root string) int {
		if err := checkpoint.Optout(ctx, root, checkpoint.Session(), args[1:]); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "opted out for this session: %s\n", strings.Join(args[1:], " "))
		return 0
	})
}

func checkpointRecord(ctx context.Context, args []string, _ io.Reader, stdout io.Writer, stderr io.Writer) int {
	return withCheckpointRoot(ctx, "checkpoint record", args, stdout, stderr, func(root string) int {
		if err := checkpoint.Record(ctx, root, checkpoint.Session()); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "recorded: %s\n", root)
		return 0
	})
}

func checkpointRoot(ctx context.Context, positional []string, stderr io.Writer) (string, bool) {
	directory, named := ".", "the working directory"
	if len(positional) == 1 && positional[0] != "" {
		directory, named = positional[0], positional[0]
	}
	root, err := checkpoint.Root(ctx, directory)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %s is not inside a git repository\n", named)
		return "", false
	}
	return root, true
}

// checkpointSnapshot is the prompt hook. The baseline spans a stop-to-stop cycle, not a prompt: a message sent while the turn runs would otherwise re-baseline over edits already made.
func checkpointSnapshot(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("checkpoint snapshot", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	_, root, stateFile, ok := readCheckpointHook(ctx, stdin)
	if !ok {
		return 0
	}
	// Off clears the baseline too, so turning the gate back on starts from the next prompt, not a stale one.
	if checkpointGateOff() {
		_ = os.Remove(stateFile)
		return 0
	}
	if _, err := os.Lstat(stateFile); err == nil {
		return 0
	}
	state, err := checkpoint.WriteState(ctx, root)
	if err != nil {
		return 0
	}
	// Renamed into place: a partial baseline reads every missing path as moved.
	if os.WriteFile(stateFile+".tmp", state.Bytes(), 0o600) == nil {
		_ = os.Rename(stateFile+".tmp", stateFile)
	}
	return 0
}

func checkpointStats(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("checkpoint stats", stderr)
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
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}

func readCheckpointHook(ctx context.Context, stdin io.Reader) (hookInput, string, string, bool) {
	var input hookInput
	content, _ := io.ReadAll(stdin)
	_ = json.Unmarshal(content, &input)
	root, err := checkpoint.Root(ctx, ".")
	if err != nil || root == "" {
		return input, "", "", false
	}
	stateFile, err := checkpoint.StateFile(input.SessionID, root)
	if err != nil {
		return input, "", "", false
	}
	return input, root, stateFile, true
}

func withCheckpointRoot(ctx context.Context, name string, args []string, stdout io.Writer, stderr io.Writer, run func(root string) int) int {
	flags := newFlagSet(name, stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() > 1 {
		return rejectCall(flags, err, stdout, stderr)
	}
	root, ok := checkpointRoot(ctx, flags.Args(), stderr)
	if !ok {
		return 2
	}
	return run(root)
}
