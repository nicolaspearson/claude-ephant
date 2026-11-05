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

const session = "this-session"

func TestCloseEndsOnlyThisSessionsRun(t *testing.T) {
	root := project(t)
	writeFile(t, root, "main.go", "package main // one\n")
	freeze(t, root, FreezeOptions{}, "main.go")
	freeze(t, root, FreezeOptions{Session: "another-session"}, "main.go")

	require.NoError(t, Close(context.Background(), root, session))

	require.False(t, Open(context.Background(), root, session))
	require.True(t, Open(context.Background(), root, "another-session"))
	require.Error(t, Close(context.Background(), root, ""))
}

func TestFingerprintHashesASymlinkByItsLinkText(t *testing.T) {
	root := project(t)
	writeFile(t, root, "a.txt", "same\n")
	writeFile(t, root, "b.txt", "same\n")
	require.NoError(t, os.Symlink("a.txt", filepath.Join(root, "link")))
	runGit(t, root, "add", "a.txt", "b.txt", "link")
	runGit(t, root, "commit", "-qm", "link")
	require.NoError(t, os.Remove(filepath.Join(root, "link")))
	require.NoError(t, os.Symlink("b.txt", filepath.Join(root, "link")))
	runGit(t, root, "add", "link")
	require.Equal(t, []StatePath{{Hash: strings.TrimSpace(gitOutput(t, root, "rev-parse", ":link")), Path: "link"}}, state(t, root).Paths, "the link's blob, as git hashes it")

	require.NoError(t, os.Symlink("nowhere", filepath.Join(root, "dangling")))
	before := fingerprint(t, root)
	require.NoError(t, os.Remove(filepath.Join(root, "dangling")))
	require.NoError(t, os.Symlink("elsewhere", filepath.Join(root, "dangling")))
	require.NotEqual(t, before, fingerprint(t, root), "a dangling link retargeted")
}

func TestFingerprintHoldsAPathWithANewline(t *testing.T) {
	root := project(t)
	names := []string{"\"quoted.txt", "Icon\r", "new\nline.txt"}
	for index, name := range names {
		writeFile(t, root, name, fmt.Sprintf("%d\n", index))
	}

	now := state(t, root)

	var want []StatePath
	for _, name := range names {
		want = append(want, StatePath{Hash: strings.TrimSpace(gitOutput(t, root, "hash-object", "--", name)), Path: name})
	}
	require.Equal(t, want, now.Paths)
	require.Equal(t, now, parseState(string(now.Bytes())))
}

func TestFingerprintSeesStagedFilesBeforeTheFirstCommit(t *testing.T) {
	root, _ := repository(t)
	runGit(t, root, "update-ref", "-d", "HEAD")
	writeFile(t, root, "staged.go", "package staged\n")
	runGit(t, root, "add", "staged.go")

	paths, _ := DirtyPaths(context.Background(), root)

	require.Equal(t, []string{"staged.go"}, paths)
}

func TestFingerprintSurvivesStagingAndMovesOnAnEdit(t *testing.T) {
	root := project(t)
	writeFile(t, root, "main.go", "package main // one\n")
	before := fingerprint(t, root)

	runGit(t, root, "add", "main.go")
	require.Equal(t, before, fingerprint(t, root))

	writeFile(t, root, "main.go", "package main // two\n")
	require.NotEqual(t, before, fingerprint(t, root))
}

func TestMatchGlobMatchesAsBashDoes(t *testing.T) {
	for glob, cases := range map[string]map[string]bool{
		"*.md":        {"a.md": true, "deck/a.md": true, "a.go": false},
		"[!]]":        {"]": false, "x": true},
		"[!q]*":       {"q": false, "x": true},
		"[]a]":        {"]": true, "a": true, "b": false},
		"[a-c]?":      {"bz": true, "dz": false, "b": false},
		"[[:digit:]]": {"7": true, "x": false},
		`\*`:          {"*": true, "x": false},
		"[x":          {"[x": true, "x": false},
	} {
		for text, want := range cases {
			require.Equal(t, want, MatchGlob(glob, text), "%q against %q", glob, text)
		}
	}
}

func TestMovedPathsIgnoresTheUsersGitActionsAndSeesTheTurnsEdits(t *testing.T) {
	for name, test := range map[string]struct {
		before func(root string)
		during func(root string)
		moved  []string
	}{
		"agent reverted a dirty file": {
			before: func(root string) { writeFile(t, root, "main.go", "package main // dirty\n") },
			during: func(root string) { runGit(t, root, "checkout", "-q", "--", "main.go") },
			moved:  []string{"main.go"},
		},
		"untracked file content changed": {
			before: func(root string) { writeFile(t, root, "notes.txt", "one\n") },
			during: func(root string) { writeFile(t, root, "notes.txt", "two\n") },
			moved:  []string{"notes.txt"},
		},
		"user committed a dirty file": {
			before: func(root string) { writeFile(t, root, "main.go", "package main // dirty\n") },
			during: func(root string) { runGit(t, root, "commit", "-qam", "wip") },
		},
		"user staged a dirty file": {
			before: func(root string) { writeFile(t, root, "main.go", "package main // dirty\n") },
			during: func(root string) { runGit(t, root, "add", "main.go") },
		},
		"user staged an untracked file": {
			before: func(root string) { writeFile(t, root, "new.go", "new\n") },
			during: func(root string) { runGit(t, root, "add", "new.go") },
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := project(t)
			test.before(root)
			baseline := state(t, root)
			test.during(root)

			require.Equal(t, test.moved, MovedPaths(baseline, state(t, root)))
		})
	}
}

func TestOptedOutNeedsEveryMovedPathExempted(t *testing.T) {
	root := project(t)
	require.NoError(t, Optout(context.Background(), root, session, []string{"deck/*"}))

	require.True(t, OptedOut(context.Background(), root, session, []string{"deck/one.md", "deck/two/three.md"}))
	require.False(t, OptedOut(context.Background(), root, session, []string{"deck/one.md", "main.go"}))
	require.False(t, OptedOut(context.Background(), root, "another-session", []string{"deck/one.md"}))
	require.ErrorContains(t, Optout(context.Background(), root, "", []string{"deck/*"}), "CLAUDE_CODE_SESSION_ID is unset")
}

func TestRecordCopiesOnlyWhatTheRunFroze(t *testing.T) {
	root := project(t)
	writeFile(t, root, "main.go", "package main // one\n")
	writeFile(t, root, "other.go", "off the list\n")
	freeze(t, root, FreezeOptions{}, "main.go")

	require.NoError(t, Record(context.Background(), root, session))

	file := reviewedFile(t, root)
	require.FileExists(t, filepath.Join(file+".d", "main.go"))
	require.NoFileExists(t, filepath.Join(file+".d", "other.go"))
	_, ok := ReviewedBytes(context.Background(), root, "other.go")
	require.False(t, ok, "a dirty path no run froze is no record")
	require.False(t, Reviewed(context.Background(), root), "dirt no reviewer saw")
	require.False(t, Covered(context.Background(), root, state(t, root), []string{"other.go"}), "dirt no reviewer saw")

	require.NoError(t, os.Remove(filepath.Join(root, "other.go")))
	require.True(t, Reviewed(context.Background(), root))
}

func TestRecordCoversTheFrozenBytesUntilTheyMove(t *testing.T) {
	root := project(t)
	require.False(t, Reviewed(context.Background(), root), "no checkpoint ever recorded")
	writeFile(t, root, "main.go", "package main // one\n")
	writeFile(t, root, "other.go", "package main\n")
	freeze(t, root, FreezeOptions{}, "main.go", "other.go")
	require.NoError(t, Record(context.Background(), root, session))
	require.True(t, Reviewed(context.Background(), root))

	runGit(t, root, "add", "other.go")
	runGit(t, root, "commit", "-qm", "partial")
	require.True(t, Reviewed(context.Background(), root), "user committed part of the reviewed work")

	writeFile(t, root, "main.go", "package main // two\n")
	require.False(t, Reviewed(context.Background(), root), "edited after the record")

	runGit(t, root, "checkout", "-q", "--", ".")
	require.False(t, Reviewed(context.Background(), root), "clean tree")
}

func TestRecordRefusesARunThatIsNotThisSessionsOpenOne(t *testing.T) {
	root := project(t)
	writeFile(t, root, "main.go", "package main // abandoned\n")
	freeze(t, root, FreezeOptions{}, "main.go")
	run, err := RunFile(context.Background(), root, session)
	require.NoError(t, err)
	old := time.Now().Add(-3 * time.Hour)
	require.NoError(t, os.Chtimes(run+".open", old, old))

	err = Record(context.Background(), root, session)
	require.ErrorContains(t, err, "freeze again, then record")
	require.False(t, Reviewed(context.Background(), root), "the abandoned run's tree")

	freeze(t, root, FreezeOptions{Session: "another-session"}, "main.go")
	require.ErrorContains(t, Record(context.Background(), root, session), "no open run of this session")
	require.False(t, Reviewed(context.Background(), root), "another session's tree")
	require.True(t, Open(context.Background(), root, "another-session"), "the refused record leaves the other run open")

	require.ErrorContains(t, Record(context.Background(), root, ""), "CLAUDE_CODE_SESSION_ID unset")
}

func TestRecordRefusesATreeThatMovedSinceTheFreeze(t *testing.T) {
	root := project(t)
	writeFile(t, root, "main.go", "package main // reviewed\n")
	freeze(t, root, FreezeOptions{}, "main.go")
	writeFile(t, root, "main.go", "package main // unreviewed\n")

	require.ErrorContains(t, Record(context.Background(), root, session), "changed since the last freeze, so no reviewer saw it; have the change re-reviewed")
	require.False(t, Reviewed(context.Background(), root))

	run, err := RunFile(context.Background(), root, session)
	require.NoError(t, err)
	require.NoError(t, os.Remove(run+".state"))
	require.ErrorContains(t, Record(context.Background(), root, session), "has no frozen fingerprint, as one opened before ele saved it, so no reviewer is known to have seen the tree; have the change re-reviewed")

	writeFile(t, root, "other.go", "another session's work\n")
	freeze(t, root, FreezeOptions{}, "main.go")
	writeFile(t, root, "other.go", "another session's work, moved\n")
	require.NoError(t, Record(context.Background(), root, session), "a path off the change list moving is not this run's")
}

func TestRunsOfTwoSessionsOnOneRepositoryKeepTheirOwnStateAndClock(t *testing.T) {
	root := project(t)
	writeFile(t, root, "main.go", "package main // one\n")
	freeze(t, root, FreezeOptions{StartsRun: true}, "main.go")
	ours, err := RunFile(context.Background(), root, session)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(ours+".started", []byte("1\n"), 0o600))

	freeze(t, root, FreezeOptions{Session: "another-session", StartsRun: true}, "main.go")
	require.True(t, Open(context.Background(), root, session), "another session's freeze leaves this run open")
	freeze(t, root, FreezeOptions{}, "main.go")
	requireStarted(t, ours, "1")

	require.NoError(t, Record(context.Background(), root, "another-session"))
	require.True(t, Open(context.Background(), root, session), "another session's record leaves this run open")
}

func TestStatePastThePathCapCoversNothing(t *testing.T) {
	root := project(t)
	for index := range pathCap {
		writeFile(t, root, fmt.Sprintf("a/%03d.txt", index), "x\n")
	}
	under := make([]string, 0, pathCap)
	for index := range pathCap {
		under = append(under, fmt.Sprintf("a/%03d.txt", index))
	}
	freeze(t, root, FreezeOptions{}, under...)
	require.NoError(t, Record(context.Background(), root, session))
	require.True(t, Reviewed(context.Background(), root), "every dirty path reviewed, none past the cap")

	writeFile(t, root, "main.go", "package main // past the cap\n")
	summary := freeze(t, root, FreezeOptions{}, append(under, "main.go")...)
	require.Contains(t, summary, "dirty paths past 200, unfingerprinted: 1")
	require.NoError(t, Record(context.Background(), root, session))

	now := state(t, root)
	require.Equal(t, 1, now.Capped)
	require.Equal(t, now, parseState(string(now.Bytes())), "the cap survives a baseline on disk")
	require.False(t, Reviewed(context.Background(), root), "main.go reviewed too, but past the cap no hash says so")
	require.False(t, Covered(context.Background(), root, now, []string{"main.go"}))
}

func TestValidateOptoutRefusesGlobsThatMatchNothingOrEverything(t *testing.T) {
	for _, glob := range []string{"*", "**", "**/*", "*.*", "*/*", "?*", "[!q]*", "/repo/deck/*", "", "../deck/*", "./deck/*", "deck/", "deck/../x", "deck/./x", "deck//x"} {
		require.Error(t, ValidateOptout(glob), glob)
	}
	for _, glob := range []string{"*.md", ".github/*", "deck/*"} {
		require.NoError(t, ValidateOptout(glob), glob)
	}
}

func fingerprint(t *testing.T, root string) string {
	t.Helper()
	return state(t, root).Fingerprint()
}

func freeze(t *testing.T, root string, options FreezeOptions, paths ...string) string {
	t.Helper()
	options.Root, options.Paths = root, paths
	if options.Against == "" {
		options.Against = "HEAD"
	}
	if options.OutDir == "" {
		options.OutDir = t.TempDir()
	}
	if options.Session == "" {
		options.Session = session
	}
	summary, err := Freeze(context.Background(), options)
	require.NoError(t, err)
	return summary
}

// project is a repository holding a committed main.go, its run state kept under the test.
func project(t *testing.T) string {
	t.Helper()
	root, _ := repository(t)
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("CHECKPOINT_OPEN_MINUTES", "")
	writeFile(t, root, "main.go", "package main\n")
	runGit(t, root, "add", "main.go")
	runGit(t, root, "commit", "-qm", "main")
	return root
}

func requireStarted(t *testing.T, run string, want string) {
	t.Helper()
	started, err := os.ReadFile(run + ".started")
	require.NoError(t, err)
	require.Equal(t, want+"\n", string(started))
}

func reviewedFile(t *testing.T, root string) string {
	t.Helper()
	file, err := ReviewedFile(context.Background(), root)
	require.NoError(t, err)
	return file
}

func state(t *testing.T, root string) State {
	t.Helper()
	current, err := WriteState(context.Background(), root)
	require.NoError(t, err)
	return current
}
