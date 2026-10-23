package checkpoint

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOpenMinutes = 120
	deleted            = "-"
	devNull            = "/dev/null"
	// Every path is hashed, so the count bounds the hook's runtime.
	pathCap         = 200
	sessionVariable = "CLAUDE_CODE_SESSION_ID"
)

// State is a working-tree fingerprint, hashed by on-disk content rather than index or diff, so the user staging or committing between turns moves nothing.
type State struct {
	Head  string
	Paths []StatePath
}

type StatePath struct {
	Hash string
	Path string
}

func (s State) Bytes() []byte {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "%s\n", s.Head)
	for _, path := range s.Paths {
		fmt.Fprintf(&buffer, "%s\t%s\n", path.Hash, path.Path)
	}
	return buffer.Bytes()
}

func (s State) Fingerprint() string {
	return sha1Hex(string(s.Bytes()))
}

func (s State) hashes() map[string]string {
	hashes := make(map[string]string, len(s.Paths))
	for _, path := range s.Paths {
		hashes[path.Path] = path.Hash
	}
	return hashes
}

// Close ends a run that stops without a record, so it stops the clock too; a record leaves the clock for the log.
func Close(ctx context.Context, root string, session string) error {
	return endRun(ctx, root, session, ".frozen", ".open", ".started")
}

func DirtyPaths(ctx context.Context, root string) []string {
	changed, _ := git(ctx, root, "diff", "HEAD", "--name-only", "--no-renames", "-z")
	untracked, _ := git(ctx, root, "ls-files", "--others", "--exclude-standard", "-z")
	var paths []string
	for path := range strings.SplitSeq(changed+untracked, "\x00") {
		if path != "" {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	paths = slices.Compact(paths)
	if len(paths) > pathCap {
		paths = paths[:pathCap]
	}
	return paths
}

// MovedPaths lists paths whose content moved between the baseline and now. A path that went clean counts only while HEAD stayed put: clean under a new HEAD is a commit.
func MovedPaths(baseline State, now State) []string {
	was := baseline.hashes()
	current := now.hashes()
	var moved []string
	for path, hash := range current {
		if previous, seen := was[path]; !seen || previous != hash {
			moved = append(moved, path)
		}
	}
	if baseline.Head == now.Head {
		for path := range was {
			if _, still := current[path]; !still {
				moved = append(moved, path)
			}
		}
	}
	slices.Sort(moved)
	return slices.Compact(moved)
}

// Open covers a run frozen and not yet recorded or closed: background reviewers end turns mid-run, and those turns fall outside the Stop hook's view of the skill call.
func Open(ctx context.Context, root string, session string) bool {
	if session == "" {
		return false
	}
	run, err := RunFile(ctx, root, session)
	if err != nil {
		return false
	}
	minutes := defaultOpenMinutes
	if setting := os.Getenv("CHECKPOINT_OPEN_MINUTES"); setting != "" {
		if minutes, err = strconv.Atoi(setting); err != nil {
			return false
		}
	}
	info, err := os.Stat(run + ".open")
	if err != nil {
		return false
	}
	if math.Ceil(time.Since(info.ModTime()).Minutes()) >= float64(minutes) {
		return false
	}
	owner, err := os.ReadFile(run + ".open")
	return err == nil && strings.TrimRight(string(owner), "\n") == session
}

func OptedOut(ctx context.Context, root string, session string, moved []string) bool {
	file, err := OptoutFile(ctx, root, session)
	if err != nil {
		return false
	}
	globs, err := readLines(file)
	if err != nil {
		return false
	}
	for _, path := range moved {
		if path != "" && !slices.ContainsFunc(globs, func(glob string) bool { return MatchGlob(glob, path) }) {
			return false
		}
	}
	return true
}

func Optout(ctx context.Context, root string, session string, globs []string) error {
	if session == "" {
		return fmt.Errorf("%s is unset", sessionVariable)
	}
	for _, glob := range globs {
		if err := ValidateOptout(glob); err != nil {
			return err
		}
	}
	file, err := OptoutFile(ctx, root, session)
	if err != nil {
		return err
	}
	optouts, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err = optouts.WriteString(strings.Join(globs, "\n") + "\n"); err != nil {
		_ = optouts.Close()
		return err
	}
	return optouts.Close()
}

func OptoutFile(ctx context.Context, root string, session string) (string, error) {
	top, err := Root(ctx, root)
	if err != nil {
		return "", err
	}
	file, err := StateFile(session, top)
	if err != nil {
		return "", err
	}
	return file + ".optout", nil
}

func ReadState(path string) (State, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return State{}, err
	}
	return parseState(string(content)), nil
}

// Record copies only what the run froze: a dirty path off the change list is in the record for `check`, but was never reviewed, so no delta may skip it.
func Record(ctx context.Context, root string, session string) error {
	file, err := ReviewedFile(ctx, root)
	if err != nil {
		return err
	}
	if !Open(ctx, root, session) {
		return fmt.Errorf("no open run of this session in %s (none, expired, closed, or %s unset); freeze again, then record", root, sessionVariable)
	}
	run, err := RunFile(ctx, root, session)
	if err != nil {
		return err
	}
	state, err := WriteState(ctx, root)
	if err != nil {
		return err
	}
	frozen, err := os.ReadFile(run + ".frozen")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	frozenPaths := strings.Split(string(frozen), "\n")
	if err := os.RemoveAll(file + ".d.tmp"); err != nil {
		return err
	}
	if err := os.MkdirAll(file+".d.tmp", 0o700); err != nil {
		return err
	}
	for _, path := range state.Paths {
		if path.Hash == deleted || !regularFile(filepath.Join(root, path.Path)) || !slices.Contains(frozenPaths, path.Path) {
			continue
		}
		if err := copyFile(filepath.Join(root, path.Path), filepath.Join(file+".d.tmp", path.Path)); err != nil {
			return err
		}
	}
	steps := []func() error{
		func() error { return os.WriteFile(file+".tmp", state.Bytes(), 0o600) },
		func() error { return os.WriteFile(file+".paths.tmp", frozen, 0o600) },
		func() error { return os.RemoveAll(file + ".d") },
		func() error { return os.Rename(file+".d.tmp", file+".d") },
		func() error { return os.Rename(file+".paths.tmp", file+".paths") },
		func() error { return os.Rename(file+".tmp", file) },
		func() error { return endRun(ctx, root, session, ".frozen", ".open") },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// Reviewed ignores HEAD: committing part of reviewed work leaves the rest covered.
func Reviewed(ctx context.Context, root string) bool {
	file, err := ReviewedFile(ctx, root)
	if err != nil {
		return false
	}
	record, err := ReadState(file)
	if err != nil {
		return false
	}
	now, err := WriteState(ctx, root)
	if err != nil || len(now.Paths) == 0 {
		return false
	}
	for _, path := range now.Paths {
		if !slices.Contains(record.Paths, path) {
			return false
		}
	}
	return true
}

// ReviewedBytes is false where the copy no longer hashes to what the record says.
func ReviewedBytes(ctx context.Context, root string, path string) (string, bool) {
	file, err := ReviewedFile(ctx, root)
	if err != nil {
		return "", false
	}
	paths, err := readLines(file + ".paths")
	if err != nil || !slices.Contains(paths, path) {
		return "", false
	}
	record, err := ReadState(file)
	if err != nil {
		return "", false
	}
	index := slices.IndexFunc(record.Paths, func(entry StatePath) bool { return entry.Path == path })
	switch {
	case index < 0 || record.Paths[index].Hash == "":
		return "", false
	case record.Paths[index].Hash == deleted:
		return devNull, true
	}
	copied := filepath.Join(file+".d", path)
	if !regularFile(copied) {
		return "", false
	}
	hash, err := git(ctx, root, "hash-object", "--path="+path, "--", copied)
	if err != nil || strings.TrimSpace(hash) != record.Paths[index].Hash {
		return "", false
	}
	return copied, true
}

// ReviewedFile is the latest checkpoint per repository, not per session: identical bytes were reviewed whichever session asks.
func ReviewedFile(ctx context.Context, directory string) (string, error) {
	// Keyed by git's own root: a symlinked or /private path names the same repo.
	root, err := Root(ctx, directory)
	if err != nil {
		return "", err
	}
	reviewed := os.Getenv("CHECKPOINT_REVIEWED_DIR")
	if reviewed == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		reviewed = filepath.Join(home, ".claude", "cache", "checkpoint-reviewed")
	}
	if err := os.MkdirAll(reviewed, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(reviewed, sha1Hex(root)), nil
}

func Root(ctx context.Context, directory string) (string, error) {
	root, err := git(ctx, directory, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(root, "\n"), nil
}

// RunFile is per session as well as per repository: two sessions checkpointing one repository each keep their own run open, frozen and timed.
func RunFile(ctx context.Context, directory string, session string) (string, error) {
	if session == "" {
		return "", fmt.Errorf("%s is unset", sessionVariable)
	}
	file, err := ReviewedFile(ctx, directory)
	if err != nil {
		return "", err
	}
	return file + "." + sha1Hex(session), nil
}

func Session() string {
	return os.Getenv(sessionVariable)
}

func StateFile(session string, root string) (string, error) {
	if session == "" || root == "" {
		return "", errors.New("a state file needs a session and a root")
	}
	directory := filepath.Join(os.TempDir(), "claude-checkpoint")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(directory, sha1Hex(session+"\n"+root)), nil
}

// ValidateOptout refuses a glob that matches no repository path or nearly every one.
func ValidateOptout(glob string) error {
	if strings.HasPrefix(glob, "/") {
		return fmt.Errorf("'%s' is absolute; name paths relative to the repository root", glob)
	}
	wrapped := "/" + glob + "/"
	if MatchGlob("*/./*", wrapped) || MatchGlob("*/../*", wrapped) || MatchGlob("*//*", wrapped) {
		return fmt.Errorf("'%s' has a \".\", \"..\" or empty path segment, which no repository path does; write paths as git prints them, e.g. deck/*", glob)
	}
	for _, probe := range []string{"q9x", "Z_0", "q9x/q9x/q9x.q9x", "Z_0/Z_0/Z_0.Z_0"} {
		if MatchGlob(glob, probe) {
			return fmt.Errorf("'%s' is too broad; name the paths", glob)
		}
	}
	return nil
}

// WriteState hashes a deleted path as `-`.
func WriteState(ctx context.Context, root string) (State, error) {
	state := State{Head: "none"}
	if head, err := git(ctx, root, "rev-parse", "-q", "--verify", "HEAD"); err == nil {
		state.Head = strings.TrimRight(head, "\n")
	}
	var files, others []string
	for _, path := range DirtyPaths(ctx, root) {
		if regularFile(filepath.Join(root, path)) {
			files = append(files, path)
		} else {
			others = append(others, path)
		}
	}
	if len(files) > 0 {
		hashes, err := gitInput(ctx, root, strings.Join(files, "\n")+"\n", "hash-object", "--stdin-paths")
		if err != nil {
			return State{}, err
		}
		for index, hash := range strings.Split(strings.TrimRight(hashes, "\n"), "\n") {
			if index < len(files) {
				state.Paths = append(state.Paths, StatePath{Hash: hash, Path: files[index]})
			}
		}
	}
	for _, path := range others {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			state.Paths = append(state.Paths, StatePath{Hash: deleted, Path: path})
			continue
		}
		diff, _ := git(ctx, root, "diff", "HEAD", "--", path)
		hash, err := gitInput(ctx, root, diff, "hash-object", "--stdin")
		if err != nil {
			return State{}, err
		}
		state.Paths = append(state.Paths, StatePath{Hash: strings.TrimRight(hash, "\n"), Path: path})
	}
	return state, nil
}

func copyFile(source string, target string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	return os.WriteFile(target, content, info.Mode().Perm())
}

func endRun(ctx context.Context, root string, session string, suffixes ...string) error {
	run, err := RunFile(ctx, root, session)
	if err != nil {
		return err
	}
	for _, suffix := range suffixes {
		if err := os.Remove(run + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func git(ctx context.Context, directory string, args ...string) (string, error) {
	return gitInput(ctx, directory, "", args...)
}

func gitInput(ctx context.Context, directory string, input string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, "git", append([]string{"-C", directory}, args...)...)
	command.Stdin = strings.NewReader(input)
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return stdout.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func parseState(content string) State {
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	state := State{Head: lines[0]}
	for _, line := range lines[1:] {
		hash, path, _ := strings.Cut(line, "\t")
		state.Paths = append(state.Paths, StatePath{Hash: hash, Path: path})
	}
	return state
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(nil, 1<<20)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func sha1Hex(text string) string {
	sum := sha1.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}
