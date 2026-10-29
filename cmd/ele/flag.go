package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/profile"
	"github.com/nicolaspearson/claude-ephant/internal/redact"
)

func flagCommand(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		return misuse("flag", args, stdout, stderr)
	}
	if args[0] == "list" {
		return listFlags(ctx, args[1:], stdout, stderr)
	}
	if !slices.Contains(notes.Verdicts, args[0]) {
		return misuse("flag", args, stdout, stderr)
	}
	return saveFlag(ctx, args[0], args[1:], stdout, stderr)
}

func listFlags(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("flag list", stderr)
	owner := flags.String("owner", "", "list this owner's flags instead of the resolved one's")
	projectName := flags.String("project", "", "list this project's flags instead of the working directory's")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	scope, code, err := projectNotes(ctx, *owner, *projectName)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return code
	}
	superseded := supersededBy(scope.all)
	var lines []string
	for _, note := range newestOfKind(scope.all, notes.KindFlag, scope.project) {
		if superseded[note.ID] != "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %-4s  %s  session %s  note %s: %s\n", note.Verdict, note.Time, note.Session[:min(len(note.Session), sessionPrefix)], note.ID, strings.Join(strings.Fields(redact.Redact(note.Body)), " ")))
	}
	where := present(scope.owner, scope.project)
	if len(lines) == 0 {
		fmt.Fprintf(stdout, "ele has no flags for %s.\n", where)
		return 0
	}
	fmt.Fprintf(stdout, "Flags for %s, newest first:\n%sele show <session> --at <time> opens the session at a flag.\n", where, strings.Join(lines, ""))
	return 0
}

func saveFlag(ctx context.Context, verdict string, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("flag "+verdict, stderr)
	at := flags.String("at", "", "RFC 3339 time of the moment in the --session, such as an entry's timestamp from ele show; its last entry by default")
	owner := flags.String("owner", "", "file under this owner instead of the resolved one")
	projectName := flags.String("project", "", "file under this project instead of the working directory's")
	session := flags.String("session", "", "id or unique prefix of an indexed session to flag instead of the running one")
	positional, err := parseInterspersed(flags, args)
	if err != nil {
		return rejectCall(flags, err, stdout, stderr)
	}
	body := strings.TrimSpace(strings.Join(positional, " "))
	if body == "" {
		return rejectCall(flags, nil, stdout, stderr)
	}
	if lines := strings.Count(body, "\n") + 1; lines > bodyLineLimit {
		fmt.Fprintf(stderr, "ele: a flag is at most %d lines, not %d; keep what happened and why it was %s\n", bodyLineLimit, lines, verdict)
		return 2
	}
	if *at != "" && *session == "" {
		fmt.Fprintln(stderr, "ele: --at names a moment of an earlier session; pass --session <id> too")
		return 2
	}
	now := time.Now()
	flagged, moment := os.Getenv("CLAUDE_CODE_SESSION_ID"), now.UTC().Format(time.RFC3339)
	if *session != "" {
		code := 0
		if flagged, moment, code, err = indexedMoment(ctx, *session, *at); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return code
		}
	}
	if flagged == "" {
		fmt.Fprintln(stderr, "ele: no running session to flag; pass --session <id>")
		return 2
	}
	if !notes.ValidSession(flagged) {
		fmt.Fprintf(stderr, "ele: session %q holds characters other than letters, digits, - and _\n", flagged)
		return 2
	}
	scope, code, err := projectNotes(ctx, *owner, *projectName)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return code
	}
	id, err := notes.NewID(now)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	path, err := notes.Write(scope.noteDirectory, notes.Note{
		Body:    body,
		Date:    now.Format(time.DateOnly),
		ID:      id,
		Kind:    notes.KindFlag,
		Owner:   scope.owner,
		Profile: profile.Current(os.Getenv("CLAUDE_CONFIG_DIR")),
		Project: scope.project,
		Scope:   notes.ScopeProject,
		Session: flagged,
		Time:    moment,
		Verdict: verdict,
	})
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Flagged session %s %s, note %s: %s\n", flagged[:min(len(flagged), sessionPrefix)], verdict, id, path)
	return 0
}

// The moment is an indexed entry's own timestamp, so ele show --at compares it as a string and lands on that entry.
func indexedMoment(ctx context.Context, prefix string, at string) (session string, moment string, code int, err error) {
	var wanted time.Time
	if at != "" {
		if wanted, err = time.Parse(time.RFC3339Nano, at); err != nil {
			return "", "", 2, fmt.Errorf("--at %q is not an RFC 3339 time, such as 2026-10-29T03:11:39Z", at)
		}
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		return "", "", 1, err
	}
	if _, err = os.Stat(filepath.Join(dataDirectory, "ele.db")); err != nil {
		return "", "", 1, fmt.Errorf("no index to find session %q in; run ele sync", prefix)
	}
	searchIndex, err := openIndex()
	if err != nil {
		return "", "", 1, err
	}
	defer searchIndex.Close()
	sessions, err := searchIndex.Sessions(ctx, prefix, 2)
	if err != nil {
		return "", "", 1, err
	}
	switch len(sessions) {
	case 0:
		return "", "", 1, fmt.Errorf("no indexed session starts with %q; ele sync indexes a recent one", prefix)
	case 2:
		return "", "", 1, fmt.Errorf("more than one indexed session starts with %q; give more of its id", prefix)
	}
	entries, err := searchIndex.SessionExactly(ctx, sessions[0])
	if err != nil {
		return "", "", 1, err
	}
	var first, last string
	var started time.Time
	for _, entry := range entries {
		stamped, parseErr := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if entry.Kind == index.KindNote || parseErr != nil {
			continue
		}
		if first == "" {
			first, started = entry.Timestamp, stamped
		}
		last = entry.Timestamp
		if at != "" && moment == "" && !stamped.Before(wanted) {
			moment = entry.Timestamp
		}
	}
	switch {
	case last == "":
		return "", "", 1, fmt.Errorf("session %s has no timestamped entry to flag", sessions[0])
	case at == "":
		return sessions[0], last, 0, nil
	case moment == "" || wanted.Before(started):
		return "", "", 1, fmt.Errorf("--at %s is outside session %s, which runs from %s to %s", at, sessions[0], first, last)
	}
	return sessions[0], moment, 0, nil
}
