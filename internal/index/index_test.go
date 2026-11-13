package index

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/hookversion"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

type row struct {
	Kind     string
	Owner    string
	Profile  string
	Project  string
	Session  string
	Source   string
	Subagent bool
	Text     string
}

func TestResolveFallsBackToTheDirectoryOutsideARepository(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "scratch", "notes")
	require.NoError(t, os.MkdirAll(directory, 0o700))
	projects := NewProjects(ownermap.OwnerMap{Owners: []string{"personal"}, Paths: map[string]string{filepath.Dir(directory): "personal"}})

	project, owner := projects.Resolve(directory)
	require.Equal(t, "notes", project)
	require.Equal(t, "personal", owner)

	project, owner = projects.Resolve("")
	require.Empty(t, project)
	require.Empty(t, owner)
}

func TestResolveNamesALinkedWorktreeAfterItsMainRepository(t *testing.T) {
	base := t.TempDir()
	repository := filepath.Join(base, "claude-ephant")
	worktreeGit := filepath.Join(repository, ".git", "worktrees", "feature")
	require.NoError(t, os.MkdirAll(worktreeGit, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(worktreeGit, "commondir"), []byte("../..\n"), 0o600))
	worktree := filepath.Join(base, "feature-checkout")
	require.NoError(t, os.MkdirAll(filepath.Join(worktree, "cmd"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: "+worktreeGit+"\n"), 0o600))
	relativeGit := filepath.Join(repository, ".git", "worktrees", "relative")
	require.NoError(t, os.MkdirAll(relativeGit, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(relativeGit, "commondir"), []byte(filepath.Join(repository, ".git")), 0o600))
	relative := filepath.Join(base, "relative-checkout")
	require.NoError(t, os.MkdirAll(relative, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(relative, ".git"), []byte("gitdir: ../claude-ephant/.git/worktrees/relative\r\n"), 0o600))
	bareGit := filepath.Join(base, "mirror.git", "worktrees", "bare")
	require.NoError(t, os.MkdirAll(bareGit, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(bareGit, "commondir"), []byte("../..\n"), 0o600))
	bare := filepath.Join(base, "bare-checkout")
	require.NoError(t, os.MkdirAll(bare, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(bare, ".git"), []byte("gitdir: "+bareGit+"\n"), 0o600))
	submoduleGit := filepath.Join(repository, ".git", "modules", "vendored")
	require.NoError(t, os.MkdirAll(submoduleGit, 0o700))
	submodule := filepath.Join(repository, "vendored")
	require.NoError(t, os.MkdirAll(submodule, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(submodule, ".git"), []byte("gitdir: ../.git/modules/vendored\n"), 0o600))
	projects := NewProjects(ownermap.OwnerMap{})

	project, _ := projects.Resolve(filepath.Join(worktree, "cmd"))
	require.Equal(t, "claude-ephant", project)

	project, _ = projects.Resolve(relative)
	require.Equal(t, "claude-ephant", project)

	project, _ = projects.Resolve(bare)
	require.Equal(t, "bare-checkout", project)

	project, _ = projects.Resolve(submodule)
	require.Equal(t, "vendored", project)

	project, _ = projects.Resolve(repository)
	require.Equal(t, "claude-ephant", project)
}

func TestResolveNamesAWorktreeOfALinkedCheckoutAfterTheRealFolder(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	checkout := filepath.Join(base, "real-name")
	require.NoError(t, os.MkdirAll(checkout, 0o700))
	runGit(t, checkout, "init", "--quiet")
	runGit(t, checkout, "commit", "--quiet", "--allow-empty", "-m", "initial")
	link := filepath.Join(base, "link-name")
	require.NoError(t, os.Symlink(checkout, link))
	worktree := filepath.Join(base, "feature")
	runGit(t, link, "worktree", "add", "--quiet", worktree)
	projects := NewProjects(ownermap.OwnerMap{})

	project, _ := projects.Resolve(link)
	require.Equal(t, "link-name", project)

	project, _ = projects.Resolve(worktree)
	require.Equal(t, "real-name", project)
}

func TestUpdateIndexesNewArchiveBytesWithOwners(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "w", "corp", "gateway")
	require.NoError(t, os.MkdirAll(filepath.Join(repository, ".git"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(repository, "app"), 0o700))
	projects := filepath.Join(root, "projects")
	session := filepath.Join(projects, "-gateway", "s1.jsonl")
	write(t, session,
		line("user", filepath.Join(repository, "app"), `"export GITLAB_TOKEN=abc123def please"`)+
			`{"type":`+"\n"+
			line("assistant", repository, `[{"type":"text","text":"Pinned the pydantic version."}]`))
	write(t, filepath.Join(projects, "-gateway", "s1", "subagents", "agent-1.jsonl"),
		line("assistant", filepath.Join(root, "w", "personal", "notes"), `[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]`))
	write(t, filepath.Join(projects, "-observer", "s2.jsonl"),
		strings.Replace(line("user", repository, `"Condense the tool payload"`), `"cwd"`, `"entrypoint":"sdk-ts","cwd"`, 1))

	ownerMap := ownermap.OwnerMap{
		Owners:   []string{"corp", "personal"},
		Paths:    map[string]string{filepath.Join(root, "w", "personal"): "personal"},
		Projects: map[string]string{"gateway": "corp"},
	}
	archiveDirectory := filepath.Join(root, "archive")
	index := open(t, filepath.Join(root, "ele.db"))

	summary := update(t, index, projects, archiveDirectory, ownerMap)
	require.Equal(t, Summary{Automated: 1, Entries: 3, Events: 1, Files: 3, Malformed: 1}, summary)
	require.Equal(t, []row{
		{Kind: "prompt", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Text: "export GITLAB_TOKEN=[redacted:assignment] please"},
		{Kind: "conversation", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Text: "Pinned the pydantic version."},
		{Kind: "tool", Owner: "personal", Profile: "claude", Project: "notes", Session: "s1", Source: "claude/-gateway/s1/subagents/agent-1.jsonl.gz", Subagent: true, Text: "Bash: go test ./..."},
	}, rows(t, index))
	require.Equal(t, []string{"Pinned the pydantic version."}, match(t, index, "pydantic"))
	require.Empty(t, match(t, index, "abc123def"))

	require.Equal(t, Summary{}, update(t, index, projects, archiveDirectory, ownerMap))

	appendTo(t, session, line("user", repository, `"and the next one"`))
	summary = update(t, index, projects, archiveDirectory, ownerMap)
	require.Equal(t, Summary{Entries: 1, Files: 1}, summary)
	require.Len(t, rows(t, index), 4)
}

func TestUpdateIndexesThroughASymlinkedArchiveDirectory(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	archiveDirectory := filepath.Join(root, "archive")
	write(t, filepath.Join(projects, "-a", "s1.jsonl"), line("user", "/a", `"first prompt"`))
	unlock, err := archive.Lock(archiveDirectory)
	require.NoError(t, err)
	_, err = archive.Sync([]archive.Source{{Directory: projects, Profile: "claude"}}, archiveDirectory)
	unlock()
	require.NoError(t, err)
	link := filepath.Join(root, "linked")
	require.NoError(t, os.Symlink(archiveDirectory, link))
	index := open(t, filepath.Join(root, "ele.db"))

	summary, err := index.Update(context.Background(), link, NewProjects(ownermap.OwnerMap{}), hookversion.Open(filepath.Join(root, hookversion.Directory)))
	require.NoError(t, err)

	require.Equal(t, Summary{Entries: 1, Files: 1}, summary)
	require.Equal(t, []string{"first prompt"}, match(t, index, "first"))
}

func TestUpdateSkipsAnArchiveFileShorterThanItIndexedAndIndexesTheRest(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	archiveDirectory := filepath.Join(root, "archive")
	write(t, filepath.Join(projects, "-a", "s1.jsonl"), line("user", "/a", `"first prompt"`)+line("user", "/a", `"second prompt"`))
	index := open(t, filepath.Join(root, "ele.db"))
	update(t, index, projects, archiveDirectory, ownermap.OwnerMap{})
	// As restored from an older backup: shorter than what the index has read.
	short := archive.Path(archiveDirectory, "claude/-a/s1.jsonl", 1)
	file, err := os.Create(short)
	require.NoError(t, err)
	compressed := gzip.NewWriter(file)
	_, err = compressed.Write([]byte("x\n"))
	require.NoError(t, err)
	require.NoError(t, compressed.Close())
	require.NoError(t, file.Close())
	write(t, filepath.Join(projects, "-b", "s2.jsonl"), line("user", "/b", `"other prompt"`))
	write(t, filepath.Join(archiveDirectory, "claude", "-c", "s3.jsonl.gz"), "not gzip")

	summary := update(t, index, projects, archiveDirectory, ownermap.OwnerMap{})

	require.Equal(t, 1, summary.Files)
	require.Len(t, summary.Skipped, 2)
	require.Contains(t, summary.Skipped[0], "claude/-a/s1.jsonl")
	require.Contains(t, summary.Skipped[0], "ele reindex rebuilds from the archive")
	require.Contains(t, summary.Skipped[1], "claude/-c/s3.jsonl.gz", "a corrupt archive file")
	require.Equal(t, []string{"other prompt"}, match(t, index, "other"))
}

func TestUpdateStopsOnACancelledContextRatherThanSkippingTheRest(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	archiveDirectory := filepath.Join(root, "archive")
	write(t, filepath.Join(projects, "-a", "s1.jsonl"), line("user", "/a", `"first"`))
	write(t, filepath.Join(projects, "-b", "s2.jsonl"), line("user", "/b", `"second"`))
	unlock, err := archive.Lock(archiveDirectory)
	require.NoError(t, err)
	_, err = archive.Sync([]archive.Source{{Directory: projects, Profile: "claude"}}, archiveDirectory)
	unlock()
	require.NoError(t, err)
	index := open(t, filepath.Join(root, "ele.db"))
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	summary, err := index.Update(cancelled, archiveDirectory, NewProjects(ownermap.OwnerMap{}), hookversion.Open(t.TempDir()))

	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, summary.Skipped)
}

func TestUpdateKeepsNoFragmentOfASecretTheCutSplits(t *testing.T) {
	const secret = "ghp_abcdefghijklmnopqrstuvwxyz0123456789"
	// The secret starts before the 300-rune cut and ends after it.
	straddling := strings.Repeat("a", 281) + " " + secret + " beyond"
	toolLine := func(name, input string) string {
		return line("assistant", "/w", `[{"type":"tool_use","name":"`+name+`","input":{`+input+`}}]`)
	}
	for name, transcriptLine := range map[string]string{
		"agent description":        toolLine("Agent", `"description":"`+straddling+`","subagent_type":"general-purpose"`),
		"hook block changed paths": strings.Replace(line("user", "/w", `"Stop hook feedback:\nreason\nChanged: a.go `+secret+`"`), `"cwd"`, `"isMeta":true,"cwd"`, 1),
		"hook block reason":        strings.Replace(line("user", "/w", `"Stop hook feedback:\n`+straddling+`"`), `"cwd"`, `"isMeta":true,"cwd"`, 1),
		"tool input":               toolLine("Bash", `"command":"`+straddling+`"`),
		"tool input as JSON":       toolLine("Mystery", `"payload":"`+straddling+`"`),
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			projects := filepath.Join(root, "projects")
			write(t, filepath.Join(projects, "-w", "s1.jsonl"), transcriptLine)
			index := open(t, filepath.Join(root, "ele.db"))
			update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{})

			var stored []string
			for _, entry := range rows(t, index) {
				stored = append(stored, entry.Text)
			}
			events, _, err := index.Events(context.Background(), dto.EventQuery{})
			require.NoError(t, err)
			for _, event := range events {
				for _, value := range event.Attributes {
					switch typed := value.(type) {
					case string:
						stored = append(stored, typed)
					case []any:
						for _, element := range typed {
							text, _ := element.(string)
							stored = append(stored, text)
						}
					}
				}
			}
			require.NotEmpty(t, stored)
			for _, text := range stored {
				require.NotContains(t, text, "ghp_", "a fragment of the secret is indexed")
				require.NotContains(t, text, "beyond", "the cut no longer falls inside the secret")
			}
			require.Contains(t, strings.Join(stored, "\n"), "[redac", "the secret's place holds its placeholder, whole or cut")
		})
	}
}

func TestUpdateKeysEntriesByLineAndPlace(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	keyed := func(session string, uuid string, body string) string {
		return `{"sessionId":"` + session + `","timestamp":"2026-10-15T10:00:00Z","uuid":"` + uuid + `",` + body + "}\n"
	}
	prompt := `"type":"user","message":{"content":"fix the build"}`
	tools := `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/x"}},{"type":"tool_use","name":"Bash","input":{"command":"go test"}}]}`
	keyless := `{"sessionId":"p1","timestamp":"2026-10-15T10:01:00Z","type":"user","message":{"content":"no id here"}}` + "\n"
	write(t, filepath.Join(projects, "-w", "p1.jsonl"), keyed("p1", "a", prompt)+keyed("p1", "b", tools)+keyless+keyless)
	// A fork opens with copies of its parent's lines, under its own session.
	write(t, filepath.Join(projects, "-w", "p2.jsonl"), keyed("p2", "a", prompt)+keyed("p2", "b", tools)+keyed("p2", "c", `"type":"user","message":{"content":"now the lint"}`))
	index := open(t, filepath.Join(root, "ele.db"))
	summary := update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{})
	require.Equal(t, 6, summary.Entries, "two calls on one line, two lines without an id, and the fork's own line")

	usage, err := index.Usage(context.Background(), dto.UsageQuery{})
	require.NoError(t, err)
	require.Equal(t, 4, usage.Prompts)
	require.Equal(t, []dto.UsageCount{{Main: 1, Name: "Bash"}, {Main: 1, Name: "Read"}}, usage.Tools)
	results, err := index.Search(context.Background(), Query{Text: "fix the build"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "p1", results[0].Session, "the parent's copy, indexed first")
}

func appendTo(t *testing.T, path string, content string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = file.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, file.Close())
}

func line(kind string, workingDirectory string, content string) string {
	return `{"type":"` + kind + `","cwd":"` + workingDirectory + `","sessionId":"s1","timestamp":"2026-10-15T10:00:00Z","uuid":"` + lineID(kind, workingDirectory, content) + `","message":{"content":` + content + "}}\n"
}

// Transcript lines carry unique ids, and the index keeps one entry per id and place.
func lineID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:8])
}

func match(t *testing.T, index *Index, query string) []string {
	t.Helper()
	result, err := index.db.Query(`SELECT e.text FROM entries_fts f JOIN entries e ON e.id = f.rowid WHERE entries_fts MATCH ? ORDER BY e.id`, query)
	require.NoError(t, err)
	defer result.Close()
	var texts []string
	for result.Next() {
		var text string
		require.NoError(t, result.Scan(&text))
		texts = append(texts, text)
	}
	require.NoError(t, result.Err())
	return texts
}

func open(t *testing.T, path string) *Index {
	t.Helper()
	index, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, index.Close()) })
	return index
}

func rows(t *testing.T, index *Index) []row {
	t.Helper()
	result, err := index.db.Query(`SELECT kind, owner, profile, project, session, source, subagent, text FROM entries ORDER BY source, id`)
	require.NoError(t, err)
	defer result.Close()
	var entries []row
	for result.Next() {
		var entry row
		require.NoError(t, result.Scan(&entry.Kind, &entry.Owner, &entry.Profile, &entry.Project, &entry.Session, &entry.Source, &entry.Subagent, &entry.Text))
		entries = append(entries, entry)
	}
	require.NoError(t, result.Err())
	return entries
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
	require.NoError(t, err, string(output))
}

func update(t *testing.T, index *Index, projects string, archiveDirectory string, ownerMap ownermap.OwnerMap) Summary {
	t.Helper()
	unlock, err := archive.Lock(archiveDirectory)
	require.NoError(t, err)
	defer unlock()
	_, err = archive.Sync([]archive.Source{{Directory: projects, Profile: "claude"}}, archiveDirectory)
	require.NoError(t, err)
	summary, err := index.Update(context.Background(), archiveDirectory, NewProjects(ownerMap), hookversion.Open(filepath.Join(filepath.Dir(archiveDirectory), hookversion.Directory)))
	require.NoError(t, err)
	return summary
}

func write(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
