package plan

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type Tick struct {
	Commit string
	Date   string
	Title  string
}

// Previous prefers an uncommitted tick; outside git, or before the first commit, there is none.
// A committed tick counts only while its box is still ticked, so a revert hands Previous to the tick before.
func Previous(ctx context.Context, root string, path string) (Tick, bool, error) {
	if _, err := git(ctx, root, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		// --quiet exits 1, silent, where HEAD names no commit; outside a repository git says "not a git repository".
		if exit := (*exec.ExitError)(nil); errors.As(err, &exit) && (exit.ExitCode() == 1 || strings.Contains(err.Error(), "not a git repository")) {
			return Tick{}, false, nil
		}
		return Tick{}, false, err
	}
	diff, err := git(ctx, root, "diff", "--no-color", "--no-ext-diff", "--unified=0", "HEAD", "--", path)
	if err != nil {
		return Tick{}, false, err
	}
	if titles := flipped(diff); len(titles) > 0 {
		return Tick{Title: titles[0]}, true, nil
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return Tick{}, false, err
	}
	ticked := map[string]bool{}
	for line := range strings.SplitSeq(string(content), "\n") {
		if match := checkbox.FindStringSubmatch(strings.TrimSuffix(line, "\r")); match != nil && match[1] != " " {
			ticked[key(strings.TrimSpace(match[2]))] = true
		}
	}
	// A record separator opens each commit, so a diff line cannot be read as a header.
	history, err := git(ctx, root, "log", "--follow", "--no-color", "--no-ext-diff", "--format=%x1e%h %cs", "--unified=0", "--patch", "--", path)
	if err != nil {
		return Tick{}, false, err
	}
	for _, commit := range strings.Split(history, "\x1e")[1:] {
		header, patch, _ := strings.Cut(commit, "\n")
		for _, title := range flipped(patch) {
			if ticked[key(title)] {
				hash, date, _ := strings.Cut(header, " ")
				return Tick{Commit: hash, Date: date, Title: title}, true, nil
			}
		}
	}
	return Tick{}, false, nil
}

// Committed serves a plan its closing action already deleted, the deletion committed or not.
func Committed(ctx context.Context, root string, path string) ([]byte, error) {
	commit, err := git(ctx, root, "rev-list", "-1", "HEAD", "--", path)
	if err != nil {
		return nil, err
	}
	commit = strings.TrimSpace(commit)
	if commit == "" {
		return nil, fmt.Errorf("%s was never committed", path)
	}
	content, err := git(ctx, root, "show", commit+":"+path)
	if err != nil {
		// That commit deleted it.
		content, err = git(ctx, root, "show", commit+"^:"+path)
	}
	return []byte(content), err
}

// Titles flipped to ticked, later in the file first.
func flipped(diff string) []string {
	unticked := map[string]bool{}
	var ticked []string
	for line := range strings.SplitSeq(diff, "\n") {
		match := checkbox.FindStringSubmatch(line[min(len(line), 1):])
		switch {
		case match == nil:
		case strings.HasPrefix(line, "-") && match[1] == " ":
			unticked[key(strings.TrimSpace(match[2]))] = true
		case strings.HasPrefix(line, "+") && match[1] != " ":
			ticked = append(ticked, strings.TrimSpace(match[2]))
		}
	}
	var titles []string
	for _, title := range slices.Backward(ticked) {
		if unticked[key(title)] {
			titles = append(titles, title)
		}
	}
	return titles
}

func git(ctx context.Context, root string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	command.Stdout, command.Stderr = &stdout, &stderr
	// English stderr, so Previous can match "not a git repository".
	command.Env = append(os.Environ(), "LC_ALL=C")
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func key(title string) string {
	return cmp.Or(Name(title), title)
}
