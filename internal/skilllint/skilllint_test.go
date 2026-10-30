package skilllint

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const goodSkill = "---\nname: good\ndescription: Do a thing. Use when asked.\n---\n\nSee [the guide](references/guide.md#start).\n"

func TestCheck(t *testing.T) {
	tests := []struct {
		files   map[string]string
		name    string
		problem string
	}{
		{name: "clean skill", files: map[string]string{}},
		{name: "code span naming a file the skill has", files: map[string]string{"references/guide.md": "Read `references/guide.md`, `reviewers/*.md`, `references/<name>.md` and `scripts/build.sh`.\n"}},
		{name: "code span naming a file the skill lacks", problem: "guide.md:1: references/gone.md is not in the skill", files: map[string]string{"references/guide.md": "Read `references/gone.md`.\n"}},
		{name: "CRLF line endings", files: map[string]string{"SKILL.md": strings.ReplaceAll(goodSkill, "\n", "\r\n")}},
		{name: "description without Use when", problem: `description has no "Use when"`, files: map[string]string{"SKILL.md": "---\nname: good\ndescription: Do a thing.\n---\n"}},
		{name: "description without Use when on a quoted user-only skill", files: map[string]string{"SKILL.md": "---\nname: good\ndescription: Do a thing.\ndisable-model-invocation: \"True\"\n---\n"}},
		{name: "description without Use when on a user-only skill", files: map[string]string{"SKILL.md": "---\nname: good\ndescription: Do a thing.\ndisable-model-invocation: true\n---\n"}},
		{name: "extra directory at the root", problem: "notes: not under references/", files: map[string]string{"notes/a.md": "# A\n"}},
		{name: "extra file at the root", problem: "extra.md: not under references/", files: map[string]string{"extra.md": "# Extra\n"}},
		{name: "files under the allowed directories", files: map[string]string{"assets/a.png": "", "reviewers/bugs.md": "# Bugs\n", "scripts/run.sh": ""}},
		{name: "hidden file at the root", files: map[string]string{".DS_Store": ""}},
		{name: "link after a longer fence holding a shorter one", problem: "guide.md:6: link to after.md does not resolve", files: map[string]string{"references/guide.md": "````\n```\n[a](gone.md)\n```\n````\n[b](after.md)\n"}},
		{name: "link after a line indented past a fence", problem: "guide.md:2: link to gone.md does not resolve", files: map[string]string{"references/guide.md": "    ```\n[a](gone.md)\n"}},
		{name: "link from a reference that does not resolve", problem: "guide.md:1: link to other.md does not resolve", files: map[string]string{"references/guide.md": "Read [other](other.md).\n"}},
		{name: "link that does not resolve", problem: "SKILL.md:6: link to references/gone.md does not resolve", files: map[string]string{"SKILL.md": strings.Replace(goodSkill, "guide.md#start", "gone.md", 1)}},
		{name: "links skipped in code, to anchors, to absolute paths and to URLs", files: map[string]string{"references/guide.md": "`[a](gone.md)` [b](#start) [c](https://example.com/x) [d](mailto:a@b.c) [e](/abs/gone.md)\n```\n[f](gone.md)\n```\n~~~\n[g](gone.md)\n~~~\n"}},
		{name: "nested description key ignored", files: map[string]string{"SKILL.md": "---\nname: good\ndescription: Do a thing. Use when asked.\nmetadata:\n  description: Other.\n---\n"}},
		{name: "no front matter", problem: "SKILL.md: no front matter", files: map[string]string{"SKILL.md": "# Good\n"}},
		{name: "SKILL.md at 499 lines", files: map[string]string{"SKILL.md": goodSkill + strings.Repeat("line\n", 499-6)}},
		{name: "SKILL.md at 500 lines", problem: "SKILL.md: 500 lines", files: map[string]string{"SKILL.md": goodSkill + strings.Repeat("line\n", 500-6)}},
		{name: "SKILL.md at 500 lines without a final newline", problem: "SKILL.md: 500 lines", files: map[string]string{"SKILL.md": goodSkill + strings.Repeat("line\n", 500-7) + "line"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{"SKILL.md": goodSkill, "references/guide.md": "# Guide\n"}
			maps.Copy(files, test.files)
			for name, content := range files {
				path := filepath.Join(root, "good", name)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
			}
			problems, err := Check(root)
			require.NoError(t, err)
			if test.problem == "" {
				require.Empty(t, problems)
				return
			}
			require.Len(t, problems, 1)
			require.Contains(t, problems[0], test.problem)
		})
	}
}

func TestCheckMissingSkill(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "empty"), 0o755))
	problems, err := Check(root)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(root, "empty") + ": no SKILL.md"}, problems)
}

func TestCheckReadsEverySkill(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"b", "a"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, name), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, name, "SKILL.md"), []byte("# No front matter\n"), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("# Skills\n"), 0o600))
	problems, err := Check(root)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(root, "a", "SKILL.md") + ": no front matter", filepath.Join(root, "b", "SKILL.md") + ": no front matter"}, problems)
}

func TestPluginSkills(t *testing.T) {
	problems, err := Check(filepath.Join("..", "..", "plugin", "skills"))
	require.NoError(t, err)
	require.Empty(t, problems)
}
