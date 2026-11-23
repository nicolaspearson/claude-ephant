package skilllint

import (
	"encoding/json"
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

func TestCheckDocs(t *testing.T) {
	const commandsRow = "| `/ele:good` | Do a thing |\n"
	const readmeRow = "| `/ele:good` | Do a thing | [Good](docs/good.md) |\n"
	const statsCommandsRow = "| `/ele:stats` | Stats |\n"
	const statsReadmeRow = "| `/ele:stats` | Stats | [Stats](docs/stats.md) |\n"
	tests := []struct {
		commandFiles []string
		commands     string
		hooks        string
		name         string
		problems     []string
		readme       string
	}{
		{name: "README prose line holding pipes", commands: commandsRow, readme: "Type | `/ele:good` | to start\n", problems: []string{"good: no row under ## Skills at a glance in README.md"}},
		{name: "README prose mention only", commands: commandsRow, readme: "Type `/ele:good` to start.\n", problems: []string{"good: no row under ## Skills at a glance in README.md"}},
		{name: "command with no row", commandFiles: []string{"stats.md"}, commands: commandsRow, readme: readmeRow, problems: []string{"stats.md: no row under ## Skills and slash commands in commands.md", "stats.md: no row under ## Skills at a glance in README.md"}},
		{name: "command with rows", commandFiles: []string{"stats.md"}, commands: commandsRow + statsCommandsRow, readme: readmeRow + statsReadmeRow},
		{name: "file beside the commands that is not Markdown", commandFiles: []string{"notes.txt"}, commands: commandsRow, readme: readmeRow},
		{name: "hooks name with arguments", commands: commandsRow, hooks: "| `ele x` | `/ele:good <words>` |\n", readme: readmeRow},
		{name: "hooks row escaping a pipe, naming present names", commandFiles: []string{"stats.md"}, commands: commandsRow + statsCommandsRow, hooks: "| `ele stats [a\\|b]` | `/ele:good` and `/ele:stats` |\n", readme: readmeRow + statsReadmeRow},
		{name: "row for a longer name only", commands: "| `/ele:good-audit` | Audit |\n", readme: "| `/ele:good-audit` | Audit | [Audit](docs/audit.md) |\n", problems: []string{"good: no row under ## Skills and slash commands in commands.md", "good: no row under ## Skills at a glance in README.md", "commands.md: `/ele:good-audit` under ## Skills and slash commands names no skill or command", "README.md: `/ele:good-audit` under ## Skills at a glance names no skill or command"}},
		{name: "row in README.md only", readme: readmeRow, problems: []string{"good: no row under ## Skills and slash commands in commands.md"}},
		{name: "row in another section only", hooks: commandsRow, readme: readmeRow, problems: []string{"good: no row under ## Skills and slash commands in commands.md"}},
		{name: "row in both", commands: commandsRow, readme: readmeRow},
		{name: "row in commands.md only", commands: commandsRow, problems: []string{"good: no row under ## Skills at a glance in README.md"}},
		{name: "row in neither", problems: []string{"good: no row under ## Skills and slash commands in commands.md", "good: no row under ## Skills at a glance in README.md"}},
		{name: "rows under subheadings of the section", commands: commandsRow, readme: "### Memory\n\n" + readmeRow},
		{name: "stale name second in a two-name hooks cell", commands: commandsRow, hooks: "| `ele x` | `/ele:good` and `/ele:gone` |\n", readme: readmeRow, problems: []string{"commands.md: `/ele:gone` under ## Used by the skills and hooks names no skill or command"}},
		{name: "stale row in README.md", commands: commandsRow, readme: readmeRow + "| `/ele:gone` | Gone | [Gone](docs/gone.md) |\n", problems: []string{"README.md: `/ele:gone` under ## Skills at a glance names no skill or command"}},
		{name: "stale row in commands.md", commands: commandsRow + "| `/ele:gone` | Gone |\n", readme: readmeRow, problems: []string{"commands.md: `/ele:gone` under ## Skills and slash commands names no skill or command"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", "good"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(root, "skills", "good", "SKILL.md"), []byte(goodSkill), 0o600))
			for _, file := range test.commandFiles {
				require.NoError(t, os.MkdirAll(filepath.Join(root, "commands"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(root, "commands", file), []byte("---\ndescription: Do a thing.\n---\n"), 0o600))
			}
			commands := filepath.Join(root, "commands.md")
			readme := filepath.Join(root, "README.md")
			require.NoError(t, os.WriteFile(commands, []byte("# Commands\n\n## Skills and slash commands\n\n| Command | What it does |\n| --- | --- |\n"+test.commands+"\n## Used by the skills and hooks\n\n| Command | Called by |\n| --- | --- |\n"+test.hooks+"\n## Notes\n"), 0o600))
			require.NoError(t, os.WriteFile(readme, []byte("# ele\n\n## Skills at a glance\n\n"+test.readme+"\n## Prerequisites\n"), 0o600))
			problems, err := CheckDocs(root, commands, readme)
			require.NoError(t, err)
			require.Len(t, problems, len(test.problems))
			for index, problem := range test.problems {
				require.Contains(t, problems[index], problem)
			}
		})
	}
}

func TestCheckDocsMissingSection(t *testing.T) {
	tests := []struct {
		commands string
		err      string
		name     string
		readme   string
	}{
		{name: "README.md", commands: "## Skills and slash commands\n## Used by the skills and hooks\n", readme: "# ele\n", err: "README.md has no ## Skills at a glance section"},
		{name: "commands.md", commands: "# Commands\n", readme: "## Skills at a glance\n", err: "commands.md has no ## Skills and slash commands section"},
		{name: "hooks table", commands: "## Skills and slash commands\n", readme: "## Skills at a glance\n", err: "commands.md has no ## Used by the skills and hooks section"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", "good"), 0o755))
			commands := filepath.Join(root, "commands.md")
			readme := filepath.Join(root, "README.md")
			require.NoError(t, os.WriteFile(commands, []byte(test.commands), 0o600))
			require.NoError(t, os.WriteFile(readme, []byte(test.readme), 0o600))
			_, err := CheckDocs(root, commands, readme)
			require.ErrorContains(t, err, test.err)
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

func TestPluginDescriptionMatchesTheMarketplaceEntry(t *testing.T) {
	var plugin struct {
		Description string `json:"description"`
	}
	content, err := os.ReadFile(filepath.Join("..", "..", "plugin", ".claude-plugin", "plugin.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(content, &plugin))
	var marketplace struct {
		Plugins []struct {
			Description string `json:"description"`
			Name        string `json:"name"`
		} `json:"plugins"`
	}
	content, err = os.ReadFile(filepath.Join("..", "..", ".claude-plugin", "marketplace.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(content, &marketplace))
	require.Len(t, marketplace.Plugins, 1)
	require.Equal(t, "ele", marketplace.Plugins[0].Name)
	require.NotEmpty(t, plugin.Description)
	require.Equal(t, plugin.Description, marketplace.Plugins[0].Description, "marketplace.json's description of ele differs from plugin.json's")
}

func TestPluginSkills(t *testing.T) {
	problems, err := Check(filepath.Join("..", "..", "plugin", "skills"))
	require.NoError(t, err)
	require.Empty(t, problems)
}

func TestPluginSkillsDocumented(t *testing.T) {
	problems, err := CheckDocs(filepath.Join("..", "..", "plugin"), filepath.Join("..", "..", "docs", "commands.md"), filepath.Join("..", "..", "README.md"))
	require.NoError(t, err)
	require.Empty(t, problems)
}
