package checkpoint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAppendGitFailsOnAnErrorThatNoIndexExits1For(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(directory, "target"), 0o700))
	require.NoError(t, os.Symlink("target", filepath.Join(directory, "alias")))

	err := appendGit(context.Background(), directory, filepath.Join(t.TempDir(), "out.diff"), "diff", "--no-index", "--", devNull, "alias")

	require.ErrorContains(t, err, "Could not access")
}

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

func TestFreezeDiffsASymlinkByItsLinkText(t *testing.T) {
	root := project(t)
	writeFile(t, root, "a.txt", "same\n")
	writeFile(t, root, "b.txt", "same\n")
	runGit(t, root, "add", "a.txt", "b.txt")
	runGit(t, root, "commit", "-qm", "targets")
	require.NoError(t, os.Symlink("a.txt", filepath.Join(root, "link")))
	freeze(t, root, FreezeOptions{}, "link")
	require.NoError(t, Record(context.Background(), root, session))
	copied, ok := ReviewedBytes(context.Background(), root, "link")
	require.True(t, ok)
	target, err := os.Readlink(copied)
	require.NoError(t, err, "the record keeps the link, not its target's bytes")
	require.Equal(t, "a.txt", target)
	require.Contains(t, freeze(t, root, FreezeOptions{}, "link"), "0 changed since the last checkpoint", "the link as recorded")

	require.NoError(t, os.Remove(filepath.Join(root, "link")))
	require.NoError(t, os.Symlink("b.txt", filepath.Join(root, "link")))
	out := t.TempDir()

	require.Contains(t, freeze(t, root, FreezeOptions{OutDir: out}, "link"), "1 changed since the last checkpoint")
	delta := readFile(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".delta.diff"))
	require.Contains(t, delta, "+b.txt")
	require.NotContains(t, delta, "+same")
}

func TestFreezeDiffsTheBytesNotATextconv(t *testing.T) {
	root := project(t)
	runGit(t, root, "config", "diff.upper.textconv", "tr a-z A-Z <")
	writeFile(t, root, ".gitattributes", "*.up diff=upper\n")
	writeFile(t, root, "tracked.up", "old\n")
	runGit(t, root, "add", ".gitattributes", "tracked.up")
	runGit(t, root, "commit", "-qm", "textconv")
	writeFile(t, root, "tracked.up", "new\n")
	writeFile(t, root, "untracked.up", "fresh\n")
	out := t.TempDir()

	freeze(t, root, FreezeOptions{OutDir: out}, "tracked.up", "untracked.up")

	diff := readFile(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".diff"))
	require.Contains(t, diff, "+new\n")
	require.Contains(t, diff, "+fresh\n")
	require.NotContains(t, diff, "NEW")
	require.NotContains(t, diff, "FRESH")
}

func TestFreezeKeepsTheLastDiffWhenGitFails(t *testing.T) {
	root := project(t)
	writeFile(t, root, "other.go", "package other\n")
	runGit(t, root, "add", "other.go")
	runGit(t, root, "commit", "-qm", "other")
	blob := strings.TrimSpace(gitOutput(t, root, "rev-parse", "HEAD:other.go"))
	writeFile(t, root, "main.go", "package main // one\n")
	writeFile(t, root, "other.go", "package other // one\n")
	out := t.TempDir()
	prefix := filepath.Join(out, "checkpoint-"+filepath.Base(root))
	freeze(t, root, FreezeOptions{OutDir: out}, "main.go")
	require.NoError(t, Record(context.Background(), root, session))
	writeFile(t, root, "main.go", "package main // two\n")
	freeze(t, root, FreezeOptions{OutDir: out}, "main.go")
	diff, delta, hash := readFile(t, prefix+".diff"), readFile(t, prefix+".delta.diff"), readFile(t, prefix+".hash")
	require.NoError(t, os.Remove(filepath.Join(root, ".git", "objects", blob[:2], blob[2:])))

	for _, paths := range [][]string{{"other.go"}, {"main.go", "other.go"}} {
		_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: out, Paths: paths, Root: root, Session: session})
		require.ErrorContains(t, err, "git diff --no-ext-diff --no-color --no-textconv HEAD", "%v", paths)
		require.Equal(t, diff, readFile(t, prefix+".diff"), "%v", paths)
		require.Equal(t, delta, readFile(t, prefix+".delta.diff"), "%v", paths)
		require.Equal(t, hash, readFile(t, prefix+".hash"), "%v", paths)
	}
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

func TestFreezeRefusesAPathHoldingANewline(t *testing.T) {
	root := project(t)
	writeFile(t, root, "new\nline.txt", "x\n")

	_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"new\nline.txt"}, Root: root, Session: session})

	require.ErrorContains(t, err, "holds a newline, which the run's path list cannot; rename it")
}

func TestFreezeRefusesAPathInsideANestedRepository(t *testing.T) {
	root := project(t)
	writeFile(t, root, "kept/t.txt", "tracked\n")
	runGit(t, root, "add", "kept/t.txt")
	runGit(t, root, "commit", "-qm", "kept")
	runGit(t, root, "init", "-q", "kept")
	writeFile(t, root, "kept/t.txt", "tracked // one\n")
	runGit(t, root, "init", "-q", "vendor/clone")
	runGit(t, root, "init", "-q", "vendor/clone/inner")
	writeFile(t, root, "vendor/clone/a.go", "package clone\n")
	writeFile(t, root, "vendor/clone/inner/c.go", "package inner\n")
	runGit(t, root, "init", "-q", "--separate-git-dir", filepath.Join(t.TempDir(), "module.git"), "module")
	writeFile(t, root, "module/deep/b.go", "package deep\n")
	writeFile(t, root, "main.go", "package main // one\n")
	for path, nested := range map[string]string{
		"module/deep/b.go":        "module",
		"vendor/clone/a.go":       "vendor/clone",
		"vendor/clone/inner/c.go": "vendor/clone/inner",
	} {
		_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"main.go", path}, Root: root, Session: session})
		require.ErrorContains(t, err, path+" is inside the repository at "+nested+"; checkpoint it from there", path)
	}

	require.Contains(t, freeze(t, root, FreezeOptions{}, "kept/t.txt"), "froze 1 paths", "a path the parent tracks")
}

func TestFreezeRefusesARepositoryWithNoCommit(t *testing.T) {
	root, _ := repository(t)
	runGit(t, root, "update-ref", "-d", "HEAD")
	writeFile(t, root, "staged.go", "package staged\n")
	runGit(t, root, "add", "staged.go")

	_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"staged.go"}, Root: root, Session: session})

	require.ErrorContains(t, err, root+" has no commit yet; commit once, then checkpoint")
}

func TestFreezeRefusesAnOutDirAnotherRootFroze(t *testing.T) {
	var roots []string
	for range 2 {
		directory := filepath.Join(t.TempDir(), "same")
		runGit(t, t.TempDir(), "init", "-q", directory)
		runGit(t, directory, "commit", "-q", "--allow-empty", "-m", "first")
		writeFile(t, directory, "main.go", "package main\n")
		root, err := Root(context.Background(), directory)
		require.NoError(t, err)
		roots = append(roots, root)
	}
	t.Setenv("CHECKPOINT_REVIEWED_DIR", filepath.Join(t.TempDir(), "reviewed"))
	out := t.TempDir()
	freeze(t, roots[0], FreezeOptions{OutDir: out}, "main.go")
	require.Equal(t, roots[0]+"\n", readFile(t, filepath.Join(out, "checkpoint-same.root")))
	diff := readFile(t, filepath.Join(out, "checkpoint-same.diff"))

	_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: out, Paths: []string{"main.go"}, Root: roots[1], Session: session})
	require.ErrorContains(t, err, out+" holds the checkpoint files of "+roots[0]+"; freeze "+roots[1]+" into another directory")
	require.Equal(t, diff, readFile(t, filepath.Join(out, "checkpoint-same.diff")))
	freeze(t, roots[0], FreezeOptions{OutDir: out}, "main.go")
}

func TestFreezeRefusesAnUntrackedSymlinkToADirectory(t *testing.T) {
	root := project(t)
	writeFile(t, root, "pkg/a.go", "package pkg\n")
	require.NoError(t, os.Symlink("pkg", filepath.Join(root, "alias")))
	writeFile(t, root, "main.go", "package main // one\n")

	_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"main.go", "alias"}, Root: root, Session: session})

	require.ErrorContains(t, err, "alias is an untracked symlink to a directory, which git cannot diff; stage it with `git add`, then freeze")
}

func TestFreezeRefusesDirectoriesAndPathsNotAsGitPrintsThem(t *testing.T) {
	root := project(t)
	writeFile(t, root, "pkg/a.go", "package pkg\n")
	runGit(t, root, "add", "pkg/a.go")
	runGit(t, root, "commit", "-qm", "pkg")
	writeFile(t, root, "pkg/a.go", "package pkg // one\n")
	writeFile(t, root, "main.go", "package main // one\n")
	writeFile(t, root, "fresh/b.go", "package fresh\n")
	writeFile(t, filepath.Dir(root), "outside.go", "outside\n")
	for path, want := range map[string]string{
		"../outside.go":   "../outside.go is not a path relative to " + root + " as git prints it",
		"./main.go":       "./main.go is not a path relative to " + root + " as git prints it",
		"fresh":           "fresh is a directory; name the files in it",
		"main.go/":        "main.go/ is not a path relative to " + root + " as git prints it",
		"pkg":             "pkg is a directory; name the files in it",
		"pkg/../main.go":  "pkg/../main.go is not a path relative to " + root + " as git prints it",
		"pkg//a.go":       "pkg//a.go is not a path relative to " + root + " as git prints it",
		root + "/main.go": root + "/main.go is not a path relative to " + root + " as git prints it",
	} {
		_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"main.go", path}, Root: root, Session: session})
		require.ErrorContains(t, err, want, path)
	}

	require.NoError(t, os.RemoveAll(filepath.Join(root, "pkg")))
	_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"pkg"}, Root: root, Session: session})
	require.ErrorContains(t, err, "pkg is a directory; name the files in it", "a tracked directory deleted from the working tree")

	runGit(t, root, "update-index", "--add", "--cacheinfo", "160000,"+head(t, root)+",sub")
	runGit(t, root, "commit", "-qm", "sub")
	_, err = Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"sub"}, Root: root, Session: session})
	require.ErrorContains(t, err, "sub is a directory; name the files in it", "a submodule with no directory")

	writeFile(t, root, "staged/c.go", "package staged\n")
	runGit(t, root, "add", "staged/c.go")
	require.NoError(t, os.RemoveAll(filepath.Join(root, "staged")))
	_, err = Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"staged"}, Root: root, Session: session})
	require.ErrorContains(t, err, "staged is neither tracked nor a file", "a directory of staged files deleted from the working tree")
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

func TestFreezeSweepsFilesUntouchedForAWeek(t *testing.T) {
	root := project(t)
	reviewed := filepath.Dir(reviewedFile(t, root))
	scratch := filepath.Join(os.TempDir(), "claude-checkpoint")
	old := time.Now().Add(-8 * 24 * time.Hour)
	record, fresh := filepath.Join(reviewed, strings.Repeat("a", 40)), filepath.Join(reviewed, strings.Repeat("f", 40))
	stale := []string{record + ".def.started", record + ".def.open", record + ".def.frozen", record + ".def.state", record + ".tmp-1", filepath.Join(scratch, "abc.optout"), filepath.Join(scratch, "abc")}
	kept := []string{record, record + ".paths", filepath.Join(record+".d", "main.go"), filepath.Join(reviewed, "not-ele.state"), fresh + ".def.started", filepath.Join(scratch, "fresh.optout")}
	for _, path := range append(slices.Clone(stale), kept...) {
		writeFile(t, filepath.Dir(path), filepath.Base(path), "x\n")
		if !strings.HasPrefix(path, fresh) && !strings.HasSuffix(path, "fresh.optout") {
			require.NoError(t, os.Chtimes(path, old, old))
		}
	}
	require.NoError(t, os.Chtimes(record+".d", old, old))
	writeFile(t, root, "main.go", "package main // one\n")

	freeze(t, root, FreezeOptions{}, "main.go")

	for _, path := range stale {
		require.NoFileExists(t, path)
	}
	for _, path := range kept {
		require.FileExists(t, path)
	}
}

func TestFreezeTakesAPathLiterally(t *testing.T) {
	root := project(t)
	for _, path := range []string{"pages/[id].tsx", "pages/i.tsx"} {
		writeFile(t, root, path, "page\n")
		runGit(t, root, "add", path)
	}
	runGit(t, root, "commit", "-qm", "pages")
	writeFile(t, root, "pages/i.tsx", "page // one\n")
	_, err := Freeze(context.Background(), FreezeOptions{Against: "HEAD", OutDir: t.TempDir(), Paths: []string{"pages/[id].tsx"}, Root: root, Session: session})
	require.ErrorContains(t, err, "pages/[id].tsx has no change against HEAD")

	writeFile(t, root, "pages/[id].tsx", "page // one\n")
	out := t.TempDir()
	require.Contains(t, freeze(t, root, FreezeOptions{OutDir: out}, "pages/[id].tsx"), "froze 1 paths, 2 lines")
	require.NotContains(t, readFile(t, filepath.Join(out, "checkpoint-"+filepath.Base(root)+".diff")), "pages/i.tsx")
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
			if fileOrLink(filepath.Join(root, "other.go")) {
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
