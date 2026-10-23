package checkpoint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBaseTakesTheUpstreamElseOriginsDefaultBranch(t *testing.T) {
	root := project(t)
	_, err := Base(context.Background(), root)
	require.ErrorContains(t, err, "has no upstream and no origin/HEAD")

	pushed := head(t, root)
	runGit(t, root, "update-ref", "refs/remotes/origin/main", pushed)
	runGit(t, root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	writeFile(t, root, "branch.go", "package main\n")
	runGit(t, root, "add", "branch.go")
	runGit(t, root, "commit", "-qm", "branch")
	require.Equal(t, pushed, base(t, root))

	runGit(t, root, "commit", "-q", "--allow-empty", "-m", "ahead")
	runGit(t, root, "branch", "-q", "upstream")
	branch := strings.TrimSpace(gitOutput(t, root, "branch", "--show-current"))
	runGit(t, root, "config", "branch."+branch+".remote", ".")
	runGit(t, root, "config", "branch."+branch+".merge", "refs/heads/upstream")
	require.Equal(t, head(t, root), base(t, root))

	runGit(t, root, "config", "--unset", "branch."+branch+".remote")
	runGit(t, root, "config", "--unset", "branch."+branch+".merge")
	orphan := strings.TrimSpace(gitOutput(t, root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "commit-tree", "-m", "orphan", strings.TrimSpace(gitOutput(t, root, "hash-object", "-w", "-t", "tree", "/dev/null"))))
	runGit(t, root, "update-ref", "refs/remotes/origin/main", orphan)
	_, err = Base(context.Background(), root)
	require.ErrorContains(t, err, "shares no history")
}

func TestFreezeAgainstABaseDiffsTheCommitsAndWritesNoDelta(t *testing.T) {
	root := project(t)
	pushed := head(t, root)
	writeFile(t, root, "branch.go", "package main\n")
	runGit(t, root, "add", "branch.go")
	runGit(t, root, "commit", "-qm", "branch")
	out := t.TempDir()

	summary := freeze(t, root, FreezeOptions{Against: pushed, OutDir: out}, "branch.go")
	require.Contains(t, summary, "froze 1 paths, 1 lines against "+pushed+", small mode")
	require.Contains(t, readFile(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".diff")), "+++ b/branch.go")

	_, err := Freeze(context.Background(), FreezeOptions{Against: pushed, OutDir: out, Paths: []string{"main.go"}, Root: root})
	require.ErrorContains(t, err, "main.go has no change against "+pushed)
	_, err = Freeze(context.Background(), FreezeOptions{Against: "nowhere", OutDir: out, Paths: []string{"branch.go"}, Root: root})
	require.ErrorContains(t, err, "nowhere is not a commit")

	writeFile(t, root, "branch.go", "package main\n// again\n")
	freeze(t, root, FreezeOptions{}, "branch.go")
	require.NoError(t, Record(context.Background(), root, session))
	writeFile(t, root, "branch.go", "package main\n// again and again\n")
	freeze(t, root, FreezeOptions{Against: pushed, OutDir: out}, "branch.go")
	require.NoFileExists(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".delta.diff"))
}

func TestFreezeChoosesTheModeFromLinesAndKind(t *testing.T) {
	for name, test := range map[string]struct {
		files map[string]int
		mode  string
	}{
		"100 lines of Markdown":             {files: map[string]int{"notes.md": 100}, mode: "small"},
		"100 lines of text":                 {files: map[string]int{"notes.txt": 100}, mode: "small"},
		"101 lines of Markdown":             {files: map[string]int{"notes.md": 101}, mode: "full"},
		"40 lines of code":                  {files: map[string]int{"code.go": 40}, mode: "small"},
		"41 lines of code":                  {files: map[string]int{"code.go": 41}, mode: "full"},
		"docs and code past the code limit": {files: map[string]int{"main.go": 2, "notes.md": 39}, mode: "full"},
	} {
		t.Run(name, func(t *testing.T) {
			root := project(t)
			var paths []string
			for path, lines := range test.files {
				writeFile(t, root, path, "")
				runGit(t, root, "add", path)
				runGit(t, root, "commit", "-qm", path)
				writeFile(t, root, path, numbered(lines))
				paths = append(paths, path)
			}

			require.Contains(t, freeze(t, root, FreezeOptions{}, paths...), ", "+test.mode+" mode")
		})
	}
}

func TestFreezeCountsABinaryOrUntrackedPathAsFullMode(t *testing.T) {
	root := project(t)
	writeFile(t, root, "blob.bin", "\x00\x01")
	runGit(t, root, "add", "blob.bin")
	runGit(t, root, "commit", "-qm", "blob")
	writeFile(t, root, "blob.bin", "\x00\x02")
	require.Contains(t, freeze(t, root, FreezeOptions{}, "blob.bin"), "full mode")

	writeFile(t, root, "main.go", "package main // one\n")
	writeFile(t, root, "new file.go", "new\n")
	out := t.TempDir()
	summary := freeze(t, root, FreezeOptions{OutDir: out}, "main.go", "new file.go")
	require.Contains(t, summary, "froze 2 paths, 3 lines, full mode")
	diff := readFile(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".diff"))
	require.Contains(t, diff, "+++ b/new file.go")
	require.Contains(t, diff, "+++ b/main.go")
	require.Equal(t, fingerprint(t, root)+"\n", readFile(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".hash")))
}

func TestFreezeDiffsAPathReviewedAsDeletedFromNothing(t *testing.T) {
	root := project(t)
	writeFile(t, root, "other.go", "other\n")
	runGit(t, root, "add", "other.go")
	runGit(t, root, "commit", "-qm", "other")
	out := t.TempDir()
	delta := filepath.Join(out, "checkpoint-"+filepath.Base(root)+".delta.diff")
	runGit(t, root, "rm", "-q", "--cached", "other.go")
	require.NoError(t, os.Remove(filepath.Join(root, "other.go")))
	freeze(t, root, FreezeOptions{OutDir: out}, "other.go")
	require.NoError(t, Record(context.Background(), root, session))

	runGit(t, root, "checkout", "-q", "HEAD", "--", "other.go")
	writeFile(t, root, "other.go", "back\n")
	freeze(t, root, FreezeOptions{OutDir: out}, "other.go")
	require.Contains(t, readFile(t, delta), "+back")
	require.NotContains(t, readFile(t, delta), "-other")

	writeFile(t, root, "other.go", "")
	require.Contains(t, freeze(t, root, FreezeOptions{OutDir: out}, "other.go"), "1 changed since the last checkpoint, 0 lines")
}

func TestFreezeKeepsTheRunsClockUnlessItStartsARun(t *testing.T) {
	root := project(t)
	writeFile(t, root, "main.go", "package main // one\n")
	run, err := RunFile(context.Background(), root, session)
	require.NoError(t, err)

	freeze(t, root, FreezeOptions{}, "main.go")
	require.FileExists(t, run+".started", "the first freeze starts the clock")
	require.NoError(t, os.WriteFile(run+".started", []byte("1\n"), 0o600))
	freeze(t, root, FreezeOptions{}, "main.go")
	requireStarted(t, run, "1")

	require.NoError(t, Record(context.Background(), root, session))
	requireStarted(t, run, "1")
	writeFile(t, root, "main.go", "package main // two\n")
	freeze(t, root, FreezeOptions{}, "main.go")
	requireStarted(t, run, "1")

	old := time.Now().Add(-3 * time.Hour)
	require.NoError(t, os.Chtimes(run+".open", old, old))
	freeze(t, root, FreezeOptions{}, "main.go")
	requireStarted(t, run, "1")

	freeze(t, root, FreezeOptions{StartsRun: true}, "main.go")
	started := readFile(t, run+".started")
	require.NotEqual(t, "1\n", started, "--start opens a new run")

	_, err = Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"main.go"}, Root: root})
	require.NoError(t, err, "a freeze with no session still writes its diff")
}

func TestFreezePairsAReviewedPathRenamedSince(t *testing.T) {
	for name, paths := range map[string][2]string{
		"a path git quotes": {"ünï.go", "möved.go"},
		"a plain path":      {"rename.go", "moved.go"},
	} {
		t.Run(name, func(t *testing.T) {
			root := project(t)
			runGit(t, root, "config", "core.quotePath", "true")
			writeFile(t, root, paths[0], "a\nb\nc\nd\ne\n")
			runGit(t, root, "add", paths[0])
			runGit(t, root, "commit", "-qm", "rename")
			writeFile(t, root, paths[0], "a\nb\nc\nd\ne\nf\n")
			freeze(t, root, FreezeOptions{}, paths[0])
			require.NoError(t, Record(context.Background(), root, session))
			runGit(t, root, "mv", paths[0], paths[1])
			writeFile(t, root, paths[1], "a\nb\nc\nd\ne\nf\ng\n")
			out := t.TempDir()

			require.Contains(t, freeze(t, root, FreezeOptions{OutDir: out}, paths[0], paths[1]), "2 changed since the last checkpoint, 2 lines")
			require.Contains(t, readFile(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".delta.diff")), "rename from")
		})
	}
}

func TestFreezePairsAStagedRenameAndTakesAStagedDeletion(t *testing.T) {
	root := project(t)
	writeFile(t, root, "other.go", "package other\n")
	runGit(t, root, "add", "other.go")
	runGit(t, root, "commit", "-qm", "other")
	runGit(t, root, "mv", "other.go", "renamed.go")
	out := t.TempDir()
	freeze(t, root, FreezeOptions{OutDir: out}, "other.go", "renamed.go")
	require.Contains(t, readFile(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".diff")), "rename from other.go")

	runGit(t, root, "reset", "-q", "--hard")
	runGit(t, root, "rm", "-q", "other.go")
	require.Contains(t, freeze(t, root, FreezeOptions{}, "other.go"), "froze 1 paths, 1 lines")
}

func TestFreezeRefusesPathsItCannotDiff(t *testing.T) {
	root := project(t)
	writeFile(t, root, "untouched.go", "")
	runGit(t, root, "add", "untouched.go")
	runGit(t, root, "commit", "-qm", "untouched")
	writeFile(t, root, "main.go", "package main // one\n")
	for name, paths := range map[string][]string{
		"missing path":                        {"gone.go"},
		"unchanged path":                      {"untouched.go"},
		"unchanged path beside a changed one": {"main.go", "untouched.go"},
	} {
		_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: paths, Root: root, Session: session})
		require.Error(t, err, name)
	}
	plain := t.TempDir()
	writeFile(t, plain, "f.md", "x\n")
	_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"f.md"}, Root: plain, Session: session})
	require.ErrorContains(t, err, "is not a git repository")
}

func TestFreezeVouchesOnlyForPathsTheLastFreezeNamed(t *testing.T) {
	for name, setup := range map[string]func(root string){
		"a deletion off the frozen list": func(root string) {
			writeFile(t, root, "other.go", "other\n")
			runGit(t, root, "add", "other.go")
			runGit(t, root, "commit", "-qm", "other")
			require.NoError(t, os.Remove(filepath.Join(root, "other.go")))
			freeze(t, root, FreezeOptions{}, "main.go")
		},
		"a path an earlier freeze in the run named": func(root string) {
			writeFile(t, root, "other.go", "refrozen\n")
			freeze(t, root, FreezeOptions{}, "other.go")
			freeze(t, root, FreezeOptions{}, "main.go")
		},
		"a path dirty but never frozen": func(root string) {
			writeFile(t, root, "other.go", "off the list\n")
			freeze(t, root, FreezeOptions{}, "main.go")
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := project(t)
			writeFile(t, root, "main.go", "package main\n// one\n")
			setup(root)
			require.NoError(t, Record(context.Background(), root, session))
			out := t.TempDir()
			if regularFile(filepath.Join(root, "other.go")) {
				writeFile(t, root, "other.go", "changed again\n")
			}

			freeze(t, root, FreezeOptions{OutDir: out}, "other.go")

			require.NoFileExists(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".delta.diff"))
		})
	}
}

func TestFreezeWritesTheDeltaSinceTheLastRecord(t *testing.T) {
	root := project(t)
	out := t.TempDir()
	delta := filepath.Join(out, "checkpoint-"+filepath.Base(root)+".delta.diff")
	writeFile(t, root, "main.go", "package main\n// one\n")
	freeze(t, root, FreezeOptions{OutDir: out}, "main.go")
	require.NoFileExists(t, delta, "no delta without a record")
	require.NoError(t, Record(context.Background(), root, session))

	writeFile(t, root, "main.go", "package main\n// one\n// two\n")
	require.Contains(t, freeze(t, root, FreezeOptions{OutDir: out}, "main.go"), "; 1 changed since the last checkpoint, 1 lines, small mode")
	require.Contains(t, readFile(t, delta), "+// two")
	require.NotContains(t, readFile(t, delta), "+// one")
	require.Contains(t, readFile(t, delta), "+++ b/main.go")
	require.Contains(t, readFile(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".diff")), "+// one")

	writeFile(t, root, "main.go", "package main\n// one\n")
	writeFile(t, root, "new.go", "new\n")
	require.Contains(t, freeze(t, root, FreezeOptions{OutDir: out}, "main.go", "new.go"), "1 changed since the last checkpoint, 1 lines, full mode")
	require.NotContains(t, readFile(t, delta), "main.go")
	require.Contains(t, readFile(t, delta), "+++ b/new.go")

	require.Contains(t, freeze(t, root, FreezeOptions{OutDir: out}, "main.go"), "0 changed since the last checkpoint, 0 lines")
	require.Empty(t, readFile(t, delta))

	require.NoError(t, os.Remove(filepath.Join(root, "main.go")))
	freeze(t, root, FreezeOptions{OutDir: out}, "main.go")
	require.Contains(t, readFile(t, delta), "+++ /dev/null")
	require.Contains(t, readFile(t, delta), "-// one")

	writeFile(t, root, "main.go", "package main\n// one\n// two\n")
	writeFile(t, reviewedFile(t, root)+".d", "main.go", "tampered\n")
	freeze(t, root, FreezeOptions{OutDir: out}, "main.go")
	require.NoFileExists(t, delta, "a copy that no longer matches the record is no record")
}

func base(t *testing.T, root string) string {
	t.Helper()
	commit, err := Base(context.Background(), root)
	require.NoError(t, err)
	return commit
}

func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	output, err := git(context.Background(), root, args...)
	require.NoError(t, err)
	return output
}

func head(t *testing.T, root string) string {
	t.Helper()
	return strings.TrimSpace(gitOutput(t, root, "rev-parse", "HEAD"))
}

func numbered(lines int) string {
	var text strings.Builder
	for line := 1; line <= lines; line++ {
		fmt.Fprintf(&text, "%s\n", strings.Repeat("x", line%7+1))
	}
	return text.String()
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
