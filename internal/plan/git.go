package plan

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

type Tick struct {
	Commit string
	Date   string
	Title  string
}

// Previous prefers an uncommitted tick; outside git, or before the first commit, there is none.
func Previous(ctx context.Context, root string, path string) (Tick, bool, error) {
	if _, err := git(ctx, root, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		return Tick{}, false, nil
	}
	diff, err := git(ctx, root, "diff", "--no-color", "--no-ext-diff", "--unified=0", "HEAD", "--", path)
	if err != nil {
		return Tick{}, false, err
	}
	if title, ok := flipped(diff); ok {
		return Tick{Title: title}, true, nil
	}
	// A record separator opens each commit, so a diff line cannot be read as a header.
	history, err := git(ctx, root, "log", "--no-color", "--no-ext-diff", "--format=%x1e%h %cs", "--unified=0", "--patch", "--", path)
	if err != nil {
		return Tick{}, false, err
	}
	for _, commit := range strings.Split(history, "\x1e")[1:] {
		header, patch, _ := strings.Cut(commit, "\n")
		if title, ok := flipped(patch); ok {
			hash, date, _ := strings.Cut(header, " ")
			return Tick{Commit: hash, Date: date, Title: title}, true, nil
		}
	}
	return Tick{}, false, nil
}

func flipped(diff string) (string, bool) {
	unticked := map[string]bool{}
	var ticked []string
	for line := range strings.SplitSeq(diff, "\n") {
		match := checkbox.FindStringSubmatch(line[min(len(line), 1):])
		switch {
		case match == nil:
		case strings.HasPrefix(line, "-") && match[1] == " ":
			unticked[strings.TrimSpace(match[2])] = true
		case strings.HasPrefix(line, "+") && match[1] != " ":
			ticked = append(ticked, strings.TrimSpace(match[2]))
		}
	}
	for _, title := range slices.Backward(ticked) {
		if unticked[title] {
			return title, true
		}
	}
	return "", false
}

func git(ctx context.Context, root string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
