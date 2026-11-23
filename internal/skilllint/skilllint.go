package skilllint

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	commandsHeading = "## Skills and slash commands"
	hooksHeading    = "## Used by the skills and hooks"
	maxSkillLines   = 500
	readmeHeading   = "## Skills at a glance"
)

// SKILL.md and the directories a skill may keep beside it; reviewers/ is allowed by name.
var allowedEntries = []string{"SKILL.md", "assets", "references", "reviewers", "scripts"}

var (
	codeSpan = regexp.MustCompile("`([^`]*)`")
	eleName  = regexp.MustCompile("`/ele:([a-z0-9-]+)`")
	fence    = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
	link     = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	// Skills point at their own files in code spans, not links.
	skillPath = regexp.MustCompile(`^(?:assets|references|reviewers|scripts)/[^\s*<>:#]+\.md$`)
)

type invocable struct {
	name string
	path string
}

// Check returns one problem per broken rule across the skill directories under root.
func Check(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		found, err := checkSkill(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}
		problems = append(problems, found...)
	}
	return problems, nil
}

// CheckDocs returns a problem for each skill or slash command under pluginRoot with no row in a doc's skills section, and for each row there, or /ele: span in the hooks table, naming neither; a missing section is an error.
func CheckDocs(pluginRoot string, commands string, readme string) ([]string, error) {
	commandRows, err := sectionRows(commands, commandsHeading)
	if err != nil {
		return nil, err
	}
	hookRows, err := sectionRows(commands, hooksHeading)
	if err != nil {
		return nil, err
	}
	readmeRows, err := sectionRows(readme, readmeHeading)
	if err != nil {
		return nil, err
	}
	invocables, err := pluginInvocables(pluginRoot)
	if err != nil {
		return nil, err
	}
	hasRow := func(rows []string, cell string) bool {
		return slices.ContainsFunc(rows, func(row string) bool { return firstCell(row) == cell })
	}
	var problems []string
	for _, invocable := range invocables {
		cell := "`/ele:" + invocable.name + "`"
		if !hasRow(commandRows, cell) {
			problems = append(problems, fmt.Sprintf("%s: no row under %s in %s", invocable.path, commandsHeading, filepath.Base(commands)))
		}
		if !hasRow(readmeRows, cell) {
			problems = append(problems, fmt.Sprintf("%s: no row under %s in %s", invocable.path, readmeHeading, filepath.Base(readme)))
		}
	}
	present := func(name string) bool {
		return slices.ContainsFunc(invocables, func(invocable invocable) bool { return invocable.name == name })
	}
	for _, section := range []struct {
		heading string
		path    string
		rows    []string
	}{{commandsHeading, commands, commandRows}, {readmeHeading, readme, readmeRows}} {
		for _, row := range section.rows {
			cell := firstCell(row)
			if match := eleName.FindStringSubmatch(cell); match != nil && match[0] == cell && !present(match[1]) {
				problems = append(problems, fmt.Sprintf("%s: %s under %s names no skill or command", section.path, cell, section.heading))
			}
		}
	}
	// Every span in the row: a cell names one or two, and an escaped \| shifts the columns.
	for _, row := range hookRows {
		for _, match := range eleName.FindAllStringSubmatch(row, -1) {
			if !present(match[1]) {
				problems = append(problems, fmt.Sprintf("%s: %s under %s names no skill or command", commands, match[0], hooksHeading))
			}
		}
	}
	return problems, nil
}

func checkDescription(path string, content string) []string {
	fields, ok := frontMatter(content)
	if !ok {
		return []string{path + ": no front matter"}
	}
	if strings.EqualFold(strings.Trim(fields["disable-model-invocation"], `"'`), "true") {
		return nil
	}
	if !strings.Contains(fields["description"], "Use when") {
		return []string{path + `: description has no "Use when", and the model invokes this skill from its description`}
	}
	return nil
}

func checkEntries(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, entry := range entries {
		// .DS_Store and editor files are the filesystem's, not the skill's.
		if !strings.HasPrefix(entry.Name(), ".") && !slices.Contains(allowedEntries, entry.Name()) {
			problems = append(problems, fmt.Sprintf("%s: not under references/, scripts/, assets/ or reviewers/", filepath.Join(dir, entry.Name())))
		}
	}
	return problems, nil
}

func checkLinks(skillDir string, path string, content string) []string {
	var problems []string
	var open string
	for number, line := range strings.Split(content, "\n") {
		if marker := fence.FindStringSubmatch(line); marker != nil {
			switch {
			case open == "":
				open = marker[1]
			case marker[1][0] == open[0] && len(marker[1]) >= len(open) && strings.TrimSpace(line) == marker[1]:
				open = ""
			}
			continue
		}
		if open != "" {
			continue
		}
		for _, span := range codeSpan.FindAllStringSubmatch(line, -1) {
			if !skillPath.MatchString(span[1]) {
				continue
			}
			if _, err := os.Stat(filepath.Join(skillDir, span[1])); err != nil {
				problems = append(problems, fmt.Sprintf("%s:%d: %s is not in the skill", path, number+1, span[1]))
			}
		}
		for _, match := range link.FindAllStringSubmatch(codeSpan.ReplaceAllString(line, ""), -1) {
			target, _, _ := strings.Cut(match[1], "#")
			if target == "" || strings.Contains(target, ":") || filepath.IsAbs(target) {
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(path), target)); err != nil {
				problems = append(problems, fmt.Sprintf("%s:%d: link to %s does not resolve", path, number+1, match[1]))
			}
		}
	}
	return problems
}

func checkSkill(dir string) ([]string, error) {
	skill := filepath.Join(dir, "SKILL.md")
	raw, err := os.ReadFile(skill)
	if errors.Is(err, fs.ErrNotExist) {
		return []string{dir + ": no SKILL.md"}, nil
	}
	if err != nil {
		return nil, err
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	problems := checkDescription(skill, content)
	if lines := len(strings.Split(strings.TrimSuffix(content, "\n"), "\n")); lines >= maxSkillLines {
		problems = append(problems, fmt.Sprintf("%s: %d lines, and a SKILL.md stays under %d; move a case only some runs need into references/", skill, lines, maxSkillLines))
	}
	found, err := checkEntries(dir)
	if err != nil {
		return nil, err
	}
	problems = append(problems, found...)
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		markdown, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		problems = append(problems, checkLinks(dir, path, strings.ReplaceAll(string(markdown), "\r\n", "\n"))...)
		return nil
	})
	return problems, err
}

func firstCell(row string) string {
	cells := strings.Split(row, "|")
	return strings.TrimSpace(cells[1])
}

func frontMatter(content string) (map[string]string, bool) {
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		return nil, false
	}
	block, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return nil, false
	}
	fields := map[string]string{}
	for line := range strings.SplitSeq(block, "\n") {
		if key, value, found := strings.Cut(line, ":"); found {
			fields[key] = strings.TrimSpace(value)
		}
	}
	return fields, true
}

func pluginInvocables(pluginRoot string) ([]invocable, error) {
	skills, err := os.ReadDir(filepath.Join(pluginRoot, "skills"))
	if err != nil {
		return nil, err
	}
	var invocables []invocable
	for _, entry := range skills {
		if entry.IsDir() {
			invocables = append(invocables, invocable{name: entry.Name(), path: filepath.Join(pluginRoot, "skills", entry.Name())})
		}
	}
	commands, err := os.ReadDir(filepath.Join(pluginRoot, "commands"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for _, entry := range commands {
		if name, ok := strings.CutSuffix(entry.Name(), ".md"); ok && !entry.IsDir() {
			invocables = append(invocables, invocable{name: name, path: filepath.Join(pluginRoot, "commands", entry.Name())})
		}
	}
	return invocables, nil
}

func sectionRows(path string, heading string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	start := slices.Index(lines, heading)
	if start < 0 {
		return nil, fmt.Errorf("%s has no %s section", path, heading)
	}
	var rows []string
	for _, line := range lines[start+1:] {
		if strings.HasPrefix(line, "## ") {
			break
		}
		if strings.HasPrefix(line, "|") && strings.Count(line, "|") > 1 {
			rows = append(rows, line)
		}
	}
	return rows, nil
}
