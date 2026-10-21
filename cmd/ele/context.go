package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/profile"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

// The live transcript, not the index: a sync can be a session behind.
func contextSize(stdout io.Writer, stderr io.Writer) int {
	session := os.Getenv("CLAUDE_CODE_SESSION_ID")
	if session == "" || !notes.ValidSession(session) {
		fmt.Fprintf(stderr, "ele: --size needs CLAUDE_CODE_SESSION_ID to name the running session; it is %q\n", session)
		return 1
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	roots, err := profile.Discover(home)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	var paths []string
	for _, root := range roots {
		matches, globErr := filepath.Glob(filepath.Join(root.Directory, "projects", "*", session+".jsonl"))
		if globErr != nil {
			fmt.Fprintf(stderr, "ele: %v\n", globErr)
			return 1
		}
		paths = append(paths, matches...)
	}
	if len(paths) != 1 {
		fmt.Fprintf(stderr, "ele: found %d transcripts for session %s, not one\n", len(paths), session)
		return 1
	}
	file, err := os.Open(paths[0])
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	defer file.Close()
	tokens := 0
	lines := bufio.NewReaderSize(file, 1<<20)
	for {
		line, readErr := lines.ReadBytes('\n')
		if total, ok := transcript.ContextTokens(line); ok {
			tokens = total
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			fmt.Fprintf(stderr, "ele: read %s: %v\n", paths[0], readErr)
			return 1
		}
	}
	if tokens == 0 {
		fmt.Fprintf(stderr, "ele: session %s has no response with token usage yet\n", session)
		return 1
	}
	fmt.Fprintln(stdout, tokens)
	return 0
}

func projectContext(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("context", stderr)
	owner := flags.String("owner", "", "read this owner's notes instead of the resolved one")
	projectName := flags.String("project", "", "read this project instead of the working directory's")
	size := flags.Bool("size", false, "print the running session's context in tokens instead")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 || (*size && *owner+*projectName != "") {
		return rejectCall(flags, err, stdout, stderr)
	}
	if *size {
		return contextSize(stdout, stderr)
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
	project, resolved, code, err := resolveOwner(ctx, dataDirectory, ownerMap, *projectName, *owner)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return code
	}
	all, invalid, err := notes.ReadOwner(filepath.Join(dataDirectory, "notes"), resolved)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	questions, others := liveNotes(all, project)
	var out strings.Builder
	if len(questions)+len(others) == 0 {
		fmt.Fprintf(&out, "ele has no live notes for %s.\n", present(resolved, project))
	} else {
		fmt.Fprintf(&out, "Live notes from ele for %s:\n", present(resolved, project))
	}
	for _, note := range append(questions, others...) {
		out.WriteString(noteLine(note))
	}
	if pending := pendingHandoffs(all, project); len(pending) > 0 {
		header := "Handoff for the next startup or /clear:\n"
		if len(pending) > 1 {
			header = "Handoffs for the next startup or /clear, newest first:\n"
		}
		out.WriteString(header)
		for _, handoff := range pending {
			out.WriteString(noteLine(handoff))
		}
	}
	if invalid > 0 {
		out.WriteString(plural(invalid, "invalid note", "invalid notes") + " skipped; ele sync names each.\n")
	}
	fmt.Fprint(stdout, out.String())
	return 0
}
