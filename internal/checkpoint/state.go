package checkpoint

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
)

const (
	// Path lines hold a tab; this line never does.
	cappedPrefix       = "capped "
	defaultOpenMinutes = 120
	deleted            = "-"
	devNull            = "/dev/null"
	// Every path is hashed, so the count bounds the hook's runtime.
	pathCap         = 200
	sessionVariable = "CLAUDE_CODE_SESSION_ID"
	// Far past the run window; a session idle this long loses its opt-outs.
	staleAfter = 7 * 24 * time.Hour
)

// ele's names only, `<sha1>.`: the reviewed directory can be shared.
var elesName = regexp.MustCompile(`^[0-9a-f]{40}\.`)

// State is a working-tree fingerprint, hashed by on-disk content rather than index or diff, so the user staging or committing between turns moves nothing.
type State struct {
	Capped int
	Head   string
	Paths  []StatePath
}

type StatePath struct {
	Hash string
	Path string
}

func (s State) Bytes() []byte {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "%s\n", s.Head)
	for _, path := range s.Paths {
		name := path.Path
		// One path a line; a leading quote reads as quoting.
		if strings.Contains(name, "\n") || strings.HasPrefix(name, `"`) {
			name = strconv.Quote(name)
		}
		fmt.Fprintf(&buffer, "%s\t%s\n", path.Hash, name)
	}
	if s.Capped > 0 {
		fmt.Fprintf(&buffer, "%s%d\n", cappedPrefix, s.Capped)
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
	return endRun(ctx, root, session, ".frozen", ".open", ".started", ".state")
}

// Covered ignores dirt outside paths: another session's work does not hold a reviewed turn.
func Covered(ctx context.Context, root string, now State, paths []string) bool {
	if now.Capped > 0 {
		return false
	}
	file, err := ReviewedFile(ctx, root)
	if err != nil {
		return false
	}
	record, err := ReadState(file)
	if err != nil {
		return false
	}
	current := now.hashes()
	held := record.hashes()
	for _, path := range paths {
		if hash, dirty := current[path]; dirty && held[path] != hash {
			return false
		}
	}
	return true
}

func DirtyPaths(ctx context.Context, root string) ([]string, int) {
	changed, _ := git(ctx, root, "diff", diffBase(ctx, root), "--name-only", "--no-renames", "-z")
	untracked, _ := git(ctx, root, "ls-files", "--others", "--exclude-standard", "-z")
	var paths []string
	for path := range strings.SplitSeq(changed+untracked, "\x00") {
		if path != "" {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	paths = slices.Compact(paths)
	// Before the cap: ELE_HOME inside a checkout holds enough files to crowd out the repository's own.
	if directory, err := datadir.Path(); err == nil {
		paths = outside(paths, root, directory)
	}
	if len(paths) > pathCap {
		return paths[:pathCap], len(paths) - pathCap
	}
	return paths, 0
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
	info, err := os.Stat(run + ".open")
	if err != nil || lapsed(info.ModTime()) {
		return false
	}
	owner, err := os.ReadFile(run + ".open")
	return err == nil && strings.TrimRight(string(owner), "\n") == session
}

func lapsed(since time.Time) bool {
	minutes := defaultOpenMinutes
	if setting := os.Getenv("CHECKPOINT_OPEN_MINUTES"); setting != "" {
		var err error
		if minutes, err = strconv.Atoi(setting); err != nil {
			return true
		}
	}
	return time.Since(since) >= time.Duration(minutes)*time.Minute
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
	if err := ValidateOptout(ctx, root, globs...); err != nil {
		return err
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

// Record holds only what the run froze, as the last freeze saw it: dirt off the change list was never reviewed, and neither was an edit since that freeze.
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
	frozen, err := os.ReadFile(run + ".frozen")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	frozenPaths := slices.DeleteFunc(strings.Split(string(frozen), "\n"), func(path string) bool { return path == "" })
	state, err := PathState(ctx, root, frozenPaths)
	if err != nil {
		return err
	}
	saw, err := ReadState(run + ".state")
	switch {
	case errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("the run in %s has no frozen fingerprint, as one opened before ele saved it, so no reviewer is known to have seen the tree; have the change re-reviewed, then freeze again and record", root)
	case err != nil:
		return err
	case !slices.Equal(saw.Paths, state.Paths):
		return fmt.Errorf("a frozen path in %s changed since the last freeze, so no reviewer saw it; have the change re-reviewed, then freeze again and record", root)
	}
	// Tmp names unique to this call: a concurrent record writes into none of them.
	directory, base := filepath.Dir(file), filepath.Base(file)
	copies, err := os.MkdirTemp(directory, base+".d.tmp-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(copies) }()
	for _, path := range state.Paths {
		if path.Hash == deleted || !fileOrLink(filepath.Join(root, path.Path)) {
			continue
		}
		if err = copyFile(filepath.Join(root, path.Path), filepath.Join(copies, path.Path)); err != nil {
			return err
		}
	}
	stateFile, err := writeTemp(directory, base+".tmp-", state.Bytes())
	defer func() { _ = os.Remove(stateFile) }()
	if err != nil {
		return err
	}
	pathsFile, err := writeTemp(directory, base+".paths.tmp-", frozen)
	defer func() { _ = os.Remove(pathsFile) }()
	if err != nil {
		return err
	}
	retired, err := os.MkdirTemp(directory, base+".d.tmp-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(retired) }()
	steps := []func() error{
		func() error { return os.Remove(retired) },
		func() error {
			if err := os.Rename(file+".d", retired); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		},
		func() error { return os.Rename(copies, file+".d") },
		func() error { return os.Rename(pathsFile, file+".paths") },
		// Last: ReviewedBytes reads a copy that does not hash to the record as no record, so a crash before this never reads as covered.
		func() error { return os.Rename(stateFile, file) },
		func() error { return endRun(ctx, root, session, ".frozen", ".open", ".state") },
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
	if err != nil || len(now.Paths) == 0 || now.Capped > 0 {
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
	if !fileOrLink(copied) {
		return "", false
	}
	hash, err := hashPath(ctx, root, copied, path)
	if err != nil || hash != record.Paths[index].Hash {
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

// ValidateOptout refuses an absolute glob or one with a `.`, `..` or empty segment, one matching an arbitrary probe path, and one matching over half the repository's paths.
func ValidateOptout(ctx context.Context, root string, globs ...string) error {
	for _, glob := range globs {
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
	}
	// --deduplicate: an unmerged path is listed once per stage otherwise.
	listed, err := git(ctx, root, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--deduplicate")
	if err != nil {
		return err
	}
	paths := slices.DeleteFunc(strings.Split(listed, "\x00"), func(path string) bool { return path == "" })
	for _, glob := range globs {
		matched := 0
		for _, path := range paths {
			if MatchGlob(glob, path) {
				matched++
			}
		}
		if matched*2 > len(paths) {
			return fmt.Errorf("'%s' matches %d of the repository's %d paths, over half; name the paths", glob, matched, len(paths))
		}
	}
	return nil
}

// PathState hashes the named paths, dirty or not and uncapped, a deleted one as `-`.
func PathState(ctx context.Context, root string, paths []string) (State, error) {
	state := State{Head: "none"}
	if head, err := git(ctx, root, "rev-parse", "-q", "--verify", "HEAD"); err == nil {
		state.Head = strings.TrimRight(head, "\n")
	}
	var files, others, singles []string
	for _, path := range paths {
		info, err := os.Lstat(filepath.Join(root, path))
		// --stdin-paths reads a symlink's target, splits a name at a newline, drops a carriage return before one and unquotes a leading quote.
		switch {
		case err == nil && info.Mode().IsRegular() && !strings.ContainsAny(path, "\n\r") && !strings.HasPrefix(path, `"`):
			files = append(files, path)
		case err == nil && (info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0):
			singles = append(singles, path)
		default:
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
	for _, path := range singles {
		hash, err := hashPath(ctx, root, filepath.Join(root, path), path)
		if err != nil {
			return State{}, err
		}
		state.Paths = append(state.Paths, StatePath{Hash: hash, Path: path})
	}
	against := ""
	for _, path := range others {
		if _, err := os.Lstat(filepath.Join(root, path)); err != nil {
			state.Paths = append(state.Paths, StatePath{Hash: deleted, Path: path})
			continue
		}
		if against == "" {
			against = diffBase(ctx, root)
		}
		diff, _ := git(ctx, root, "diff", against, "--", path)
		hash, err := gitInput(ctx, root, diff, "hash-object", "--stdin")
		if err != nil {
			return State{}, err
		}
		state.Paths = append(state.Paths, StatePath{Hash: strings.TrimRight(hash, "\n"), Path: path})
	}
	return state, nil
}

func WriteState(ctx context.Context, root string) (State, error) {
	paths, capped := DirtyPaths(ctx, root)
	state, err := PathState(ctx, root, paths)
	state.Capped = capped
	return state, err
}

func copyFile(source string, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		var link string
		if link, err = os.Readlink(source); err != nil {
			return err
		}
		if err = os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return os.Symlink(link, target)
	}
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(target, content, info.Mode().Perm())
}

// diffBase is HEAD, or the empty tree before the first commit, so staged files still diff.
func diffBase(ctx context.Context, root string) string {
	if _, err := git(ctx, root, "rev-parse", "-q", "--verify", "HEAD^{commit}"); err == nil {
		return "HEAD"
	}
	tree, err := git(ctx, root, "hash-object", "-t", "tree", devNull)
	if err != nil {
		return "HEAD"
	}
	return strings.TrimSpace(tree)
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

func fileOrLink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && (info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0)
}

func git(ctx context.Context, directory string, args ...string) (string, error) {
	return gitInput(ctx, directory, "", args...)
}

func gitInput(ctx context.Context, directory string, input string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	// A path is a name, never a glob: `pages/[id].tsx` is one file.
	command := exec.CommandContext(ctx, "git", append([]string{"-C", directory, "--literal-pathspecs"}, args...)...)
	command.Stdin = strings.NewReader(input)
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return stdout.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// A symlink hashes by its link text, as git stores it: hash-object on the path would read the target.
func hashPath(ctx context.Context, root string, file string, path string) (string, error) {
	info, err := os.Lstat(file)
	if err != nil {
		return "", err
	}
	var hash string
	if info.Mode()&os.ModeSymlink == 0 {
		hash, err = git(ctx, root, "hash-object", "--path="+path, "--", file)
		return strings.TrimSpace(hash), err
	}
	link, err := os.Readlink(file)
	if err != nil {
		return "", err
	}
	hash, err = gitInput(ctx, root, link, "hash-object", "--stdin")
	return strings.TrimSpace(hash), err
}

func outside(paths []string, root string, directory string) []string {
	prefix := repositoryPath(root, directory)
	if filepath.IsAbs(prefix) || strings.HasPrefix(prefix, "..") {
		return paths
	}
	return slices.DeleteFunc(paths, func(path string) bool {
		return strings.HasPrefix(path, prefix+"/")
	})
}

func parseState(content string) State {
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	state := State{Head: lines[0]}
	for _, line := range lines[1:] {
		hash, path, found := strings.Cut(line, "\t")
		if count, capped := strings.CutPrefix(line, cappedPrefix); !found && capped {
			state.Capped, _ = strconv.Atoi(count)
			continue
		}
		if unquoted, err := strconv.Unquote(path); err == nil && strings.HasPrefix(path, `"`) {
			path = unquoted
		}
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

func sha1Hex(text string) string {
	sum := sha1.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

func sweep(directory string, cutoff time.Time, stale func(name string) bool) {
	entries, _ := os.ReadDir(directory)
	for _, entry := range entries {
		if info, err := entry.Info(); err == nil && stale(entry.Name()) && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(directory, entry.Name()))
		}
	}
}

// sweepStale removes what nothing else does: the files of runs abandoned without a record or close, a crashed record's tmp names, and ended sessions' opt-outs and baselines.
func sweepStale(ctx context.Context, root string) {
	file, err := ReviewedFile(ctx, root)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-staleAfter)
	sweep(filepath.Dir(file), cutoff, func(name string) bool {
		return elesName.MatchString(name) && (slices.Contains([]string{".frozen", ".open", ".started", ".state"}, filepath.Ext(name)) || strings.Contains(name, ".tmp"))
	})
	sweep(filepath.Join(os.TempDir(), "claude-checkpoint"), cutoff, func(string) bool { return true })
}

func writeTemp(directory string, pattern string, content []byte) (string, error) {
	file, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return "", err
	}
	if _, err = file.Write(content); err != nil {
		_ = file.Close()
		return file.Name(), err
	}
	return file.Name(), file.Close()
}
