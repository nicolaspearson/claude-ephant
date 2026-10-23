package checkpoint

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Small mode: one reviewer. Docs carry no logic for a bugs reviewer to trace.
const (
	smallDocLines = 100
	smallLines    = 40
)

type FreezeOptions struct {
	Against   string
	OutDir    string
	Paths     []string
	Root      string
	Session   string
	StartsRun bool
}

type measure struct {
	docs    bool
	lines   int
	numstat []string
	tracked bool
}

func Base(ctx context.Context, root string) (string, error) {
	upstream, err := git(ctx, root, "rev-parse", "-q", "--verify", "@{upstream}")
	if err != nil {
		if upstream, err = git(ctx, root, "rev-parse", "-q", "--verify", "refs/remotes/origin/HEAD"); err != nil {
			return "", fmt.Errorf("%s has no upstream and no origin/HEAD; name the base", root)
		}
	}
	upstream = strings.TrimRight(upstream, "\n")
	base, err := git(ctx, root, "merge-base", "HEAD", upstream)
	if err != nil {
		return "", fmt.Errorf("HEAD shares no history with %s; name the base", upstream)
	}
	return strings.TrimRight(base, "\n"), nil
}

// Freeze writes the diff for the reviewers, the delta since the last record and the fingerprint for the re-review check, and opens the run. A freeze inside an open run keeps its clock unless StartsRun, so a re-review's freeze, or one after the record, does not reset the duration.
func Freeze(ctx context.Context, options FreezeOptions) (string, error) {
	root := options.Root
	if _, err := ReviewedFile(ctx, root); err != nil {
		return "", fmt.Errorf("%s is not a git repository", root)
	}
	against := options.Against
	if _, err := git(ctx, root, "rev-parse", "-q", "--verify", against+"^{commit}"); err != nil {
		return "", fmt.Errorf("%s is not a commit in %s", against, root)
	}
	sizes := measure{docs: true, tracked: true}
	for _, path := range options.Paths {
		// A staged rename or deletion has left the working tree.
		if known(ctx, root, against, path) {
			if _, err := git(ctx, root, "diff", against, "--quiet", "--", path); err == nil {
				return "", fmt.Errorf("%s has no change against %s", path, against)
			}
		} else if !regularFile(filepath.Join(root, path)) {
			return "", fmt.Errorf("%s is neither tracked nor a file in %s", path, root)
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".txt") {
			sizes.docs = false
		}
	}
	name := filepath.Base(root)
	prefix := options.OutDir + "/checkpoint-" + name
	diff, delta, staging, hash := prefix+".diff", prefix+".delta.diff", prefix+".delta", prefix+".hash"
	if err := os.WriteFile(diff, nil, 0o600); err != nil {
		return "", err
	}
	for _, path := range []string{delta, staging} {
		if err := os.RemoveAll(path); err != nil {
			return "", err
		}
	}
	if err := appendDiff(ctx, root, against, diff, &sizes, options.Paths); err != nil {
		return "", err
	}
	if info, err := os.Stat(diff); err != nil || info.Size() == 0 {
		return "", fmt.Errorf("the diff of %d paths is empty", len(options.Paths))
	}
	mode := sizes.mode()
	summary := fmt.Sprintf("froze %d paths, %d lines", len(options.Paths), sizes.lines)
	if against != "HEAD" {
		summary += " against " + against
	}
	reviewed, err := false, error(nil)
	// A branch review covers commits, which no record's copies describe.
	if against == "HEAD" {
		if reviewed, mode, summary, err = freezeDelta(ctx, root, options.Paths, delta, staging, sizes, mode, summary); err != nil {
			return "", err
		}
	}
	state, err := WriteState(ctx, root)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(hash, []byte(state.Fingerprint()+"\n"), 0o600); err != nil {
		return "", err
	}
	if err := openRun(ctx, root, options); err != nil {
		return "", err
	}
	if reviewed {
		return fmt.Sprintf("%s, %s mode for this root alone: %s, delta %s", summary, mode, diff, delta), nil
	}
	return fmt.Sprintf("%s, %s mode for this root alone: %s", summary, mode, diff), nil
}

// appendDiff diffs paths git knows in one call, so a rename given by both paths pairs instead of reading as a delete and an add.
func appendDiff(ctx context.Context, root string, against string, out string, sizes *measure, paths []string) error {
	var tracked, untracked []string
	for _, path := range paths {
		if known(ctx, root, against, path) {
			tracked = append(tracked, path)
		} else {
			untracked = append(untracked, path)
		}
	}
	if len(tracked) > 0 {
		if err := appendGit(ctx, root, out, append([]string{"diff", against, "-M", "--"}, tracked...)...); err != nil {
			return err
		}
		numstat, _ := git(ctx, root, append([]string{"diff", against, "-M", "--numstat", "--"}, tracked...)...)
		sizes.numstat = append(sizes.numstat, strings.Split(numstat, "\n")...)
	}
	for _, path := range untracked {
		sizes.tracked = false
		if err := appendGit(ctx, root, out, "diff", "--no-index", "--", devNull, path); err != nil {
			return err
		}
		numstat, _ := git(ctx, root, "diff", "--no-index", "--numstat", "--", devNull, path)
		sizes.numstat = append(sizes.numstat, strings.Split(numstat, "\n")...)
	}
	sizes.count()
	return nil
}

// appendGit ignores git's exit status: a diff that finds changes exits 1.
func appendGit(ctx context.Context, directory string, out string, args ...string) error {
	file, err := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, "git", append([]string{"-C", directory}, args...)...)
	command.Stdout = file
	var exitError *exec.ExitError
	if err = command.Run(); err != nil && !errors.As(err, &exitError) {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func freezeDelta(ctx context.Context, root string, paths []string, delta string, staging string, sizes measure, mode string, summary string) (bool, string, string, error) {
	renamed, err := renamedPaths(ctx, root, paths)
	if err != nil {
		return false, "", "", err
	}
	sizes.numstat, sizes.tracked = nil, true
	var fresh []string
	reviewed, changed := false, 0
	for _, path := range paths {
		baseline, ok := ReviewedBytes(ctx, root, path)
		if !ok {
			fresh = append(fresh, path)
			continue
		}
		reviewed = true
		if slices.Contains(renamed, path) {
			fresh = append(fresh, path)
			continue
		}
		current := devNull
		if regularFile(filepath.Join(root, path)) {
			current = filepath.Join(root, path)
		}
		if baseline == current || (baseline != devNull && current != devNull && sameBytes(baseline, current)) {
			continue
		}
		changed++
		before, after := devNull, devNull
		if baseline != devNull {
			before = "a/" + path
			if err = copyFile(baseline, filepath.Join(staging, before)); err != nil {
				return false, "", "", err
			}
		}
		if current != devNull {
			after = "b/" + path
			if err = copyFile(current, filepath.Join(staging, after)); err != nil {
				return false, "", "", err
			}
		}
		if err = appendGit(ctx, staging, delta, "diff", "--no-index", "--no-prefix", "--", before, after); err != nil {
			return false, "", "", err
		}
		numstat, _ := git(ctx, staging, "diff", "--no-index", "--numstat", "--", before, after)
		sizes.numstat = append(sizes.numstat, strings.Split(numstat, "\n")...)
	}
	if !reviewed {
		return false, mode, summary, nil
	}
	touched, err := os.OpenFile(delta, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return false, "", "", err
	}
	if err = touched.Close(); err != nil {
		return false, "", "", err
	}
	if err := appendDiff(ctx, root, "HEAD", delta, &sizes, fresh); err != nil {
		return false, "", "", err
	}
	summary += fmt.Sprintf("; %d changed since the last checkpoint, %d lines", changed+len(fresh), sizes.lines)
	return true, sizes.mode(), summary, nil
}

func known(ctx context.Context, root string, against string, path string) bool {
	if _, err := git(ctx, root, "cat-file", "-e", against+":"+path); err == nil {
		return true
	}
	_, err := git(ctx, root, "ls-files", "--error-unmatch", "--", path)
	return err == nil
}

func openRun(ctx context.Context, root string, options FreezeOptions) error {
	if options.Session == "" {
		return nil
	}
	run, err := RunFile(ctx, root, options.Session)
	if err != nil {
		return err
	}
	if _, err := os.Stat(run + ".started"); options.StartsRun || err != nil {
		if err := os.WriteFile(run+".started", []byte(strconv.FormatInt(time.Now().Unix(), 10)+"\n"), 0o600); err != nil {
			return err
		}
	}
	if err := os.WriteFile(run+".frozen", []byte(strings.Join(options.Paths, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	return os.WriteFile(run+".open", []byte(options.Session+"\n"), 0o600)
}

// renamedPaths reads -z: otherwise git quotes a non-ASCII, tab or newline path, and the rename never matches.
func renamedPaths(ctx context.Context, root string, paths []string) ([]string, error) {
	output, err := git(ctx, root, append([]string{"diff", "HEAD", "-M", "--name-status", "-z", "--"}, paths...)...)
	if err != nil {
		return nil, err
	}
	fields := strings.Split(strings.TrimSuffix(output, "\x00"), "\x00")
	var renamed []string
	for at := 0; at+1 < len(fields); at += 2 {
		if strings.HasPrefix(fields[at], "R") && at+2 < len(fields) {
			renamed = append(renamed, fields[at+1], fields[at+2])
			at++
		}
	}
	return renamed, nil
}

func sameBytes(left string, right string) bool {
	leftContent, leftErr := os.ReadFile(left)
	rightContent, rightErr := os.ReadFile(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftContent, rightContent)
}

func (m *measure) count() {
	m.lines = 0
	for _, line := range m.numstat {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == "-" {
			m.tracked = false
			continue
		}
		added, _ := strconv.Atoi(fields[0])
		removed, _ := strconv.Atoi(fields[1])
		m.lines += added + removed
	}
}

func (m measure) mode() string {
	if m.tracked && (m.lines <= smallLines || (m.docs && m.lines <= smallDocLines)) {
		return "small"
	}
	return "full"
}
