package ticket

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/nicolaspearson/claude-ephant/internal/plan"
)

const (
	corpusWorkers = 8
	manifestName  = "manifest.json"
	// Page cap: a pagination bug cannot loop forever; a list reaching it warns.
	maximumPages = 20
	pageSize     = 100
)

var (
	// One line only: refs on the lines below a "Blocked by" heading are warned about, not read.
	blockedByLine     = regexp.MustCompile(`(?im)^[ \t>*_|#-]*(?:\p{Nd}+\.[ \t]*)?(?:\[[ xX]\][ \t]*)?[*_]*blocked by[*_]*[ \t]*:?[ \t|*_]*(.*)$`)
	corpusHeader      = regexp.MustCompile(`^# [^\t-\r\x1c-\x1f\x85\p{Z}]+ \[(?:open|closed|inbox)\] `)
	leadingReferences = regexp.MustCompile(`(?i)^(?:(?:[\s,&;/|]|\band\b|\bor\b)*(?:[\w.-]+/[\w./-]+)?#\d+(?:\s*\([^)]*\))?)+`)
	markdownLink      = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	parenthetical     = regexp.MustCompile(`\([^)]*\)`)
	placeholders      = []string{"Part of #MAP", "Part of nothing"}
	planItem          = regexp.MustCompile(`^- (?:\[([ xX])\] )?(.*)$`)
	reference         = regexp.MustCompile(`(?:[\w.-]+/[\w./-]+)?#\d+`)
	remoteProject     = regexp.MustCompile(`^(?:[\p{L}\p{N}_]+://[^/]+/|[^@/]+@[^:]+:)(.+?)(?:\.git)?/?$`)
	urlScheme         = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*$`)
)

type Commands struct {
	LookPath func(file string) (string, error)
	Run      func(ctx context.Context, name string, args ...string) (string, error)
}

type CorpusOptions struct {
	CLI     string
	Clean   bool
	Map     int
	Out     string
	Plans   []string
	Tickets []int
}

type corpusEntry struct {
	alias    string
	blockers []string
	file     string
	hash     string
	key      string
	name     string
	raw      string
	state    string
	title    string
}

type fetched struct {
	body     string
	capped   bool
	comments []string
	err      error
	state    string
	title    string
}

type planEntry struct {
	action bool
	hashed string
	raw    string
	state  string
	title  string
}

type searchHit struct {
	body   string
	number int
	state  string
}

// Compare reads a ticket back after a write: 0 when its body, or with note one
// of its comments, matches the file; 1 when none does, with a diff against the
// closest if there is one; 2 when the ticket or the file cannot be read. Never
// both body and comments: a note quoting a body edit that never landed would
// otherwise pass it.
func Compare(ctx context.Context, stdout io.Writer, stderr io.Writer, commands Commands, cli string, number int, expectPath string, note bool) int {
	expectPath = cleanPath(expectPath)
	cli, err := detectCLI(ctx, commands, cli)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 2
	}
	written, err := readText(expectPath)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 2
	}
	expected := comparableText(written)
	ticket := fetch(ctx, commands, cli, number)
	if ticket.err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", ticket.err)
		return 2
	}
	if ticket.capped {
		warnCapped(stderr, number, "comments")
	}
	type candidate struct{ label, text string }
	candidates := []candidate{{"body", ticket.body}}
	if note {
		candidates = nil
		for index, comment := range ticket.comments {
			candidates = append(candidates, candidate{fmt.Sprintf("comment %d", index+1), comment})
		}
	}
	for _, each := range candidates {
		if comparableText(each.text) == expected {
			fmt.Fprintf(stdout, "#%d: %s matches %s\n", number, each.label, expectPath)
			return 0
		}
	}
	if len(candidates) == 0 {
		fmt.Fprintf(stdout, "#%d: no comments to match %s\n", number, expectPath)
		return 1
	}
	closest, closestRatio := candidates[0], -1.0
	for _, each := range candidates {
		if ratio := newSequenceMatcher([]rune(comparableText(each.text)), []rune(expected)).ratio(); ratio > closestRatio {
			closest, closestRatio = each, ratio
		}
	}
	fmt.Fprintf(stdout, "#%d: nothing matches %s; closest is the %s:\n", number, expectPath, closest.label)
	writeUnifiedDiff(stdout, splitLines(expected, true), splitLines(comparableText(closest.text), true), expectPath, fmt.Sprintf("#%d %s", number, closest.label))
	fmt.Fprintln(stdout)
	return 1
}

// Corpus writes each item twice, `<name>.md` with whitespace collapsed so a
// wrapped phrase still matches, and `raw/<name>.md` as read, plus the manifest
// of keys and hashes. Exit 2 is a partial corpus, which reports an artefact as
// unproduced when only its producer is missing; 1 is a refusal or a failure
// before a usable corpus.
func Corpus(ctx context.Context, stdout io.Writer, stderr io.Writer, commands Commands, options CorpusOptions) int {
	out := cleanPath(options.Out)
	if err := prepare(out, options.Clean); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	var entries []corpusEntry
	var failed []int
	var err error
	if len(options.Plans) > 0 {
		for _, path := range options.Plans {
			var planned []corpusEntry
			if planned, err = planEntries(ctx, commands, cleanPath(path)); err != nil {
				break
			}
			entries = append(entries, planned...)
		}
	} else {
		entries, failed, err = ticketEntries(ctx, stderr, commands, options)
	}
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	if duplicates := repeated(entries, func(entry corpusEntry) string { return entry.key }); len(duplicates) > 0 {
		fmt.Fprintf(stderr, "ele: two items share a key, so a ruling could land on either: %s\n", strings.Join(duplicates, "; "))
		return 1
	}
	if duplicates := repeated(entries, func(entry corpusEntry) string { return entry.file }); len(duplicates) > 0 {
		fmt.Fprintf(stderr, "ele: two plans share a file name, so one would overwrite the other: %s\n", strings.Join(duplicates, "; "))
		return 1
	}
	planned := len(options.Plans) > 0
	if len(entries) == 0 {
		nothing := "tickets"
		if planned {
			nothing = "plan items"
		}
		fmt.Fprintf(stderr, "ele: no %s written\n", nothing)
		return 1
	}
	written := map[string][]byte{}
	for _, entry := range entries {
		header := fmt.Sprintf("# %s [%s] %s\n", entry.name, entry.state, entry.title)
		collapsed := []byte(header + collapseWhitespace(entry.raw) + "\n")
		if err := os.WriteFile(filepath.Join(out, entry.file), collapsed, 0o644); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		if err := os.WriteFile(filepath.Join(out, "raw", entry.file), []byte(header+entry.raw+"\n"), 0o644); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		written[entry.file] = collapsed
		if planned {
			fmt.Fprintf(stdout, "%s %-6s %s %s\n", entry.name, entry.state, entry.hash[:12], entry.title)
		} else {
			fmt.Fprintf(stdout, "#%-5s %-6s %s %s\n", entry.name, entry.state, entry.hash[:12], entry.title)
		}
	}
	if err := os.WriteFile(filepath.Join(out, manifestName), manifest(entries), 0o644); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	// Byte order, so `LC_ALL=C cat <out>/*.md` reproduces this hash.
	names := slices.Sorted(maps.Keys(written))
	digest := sha1.New()
	for _, name := range names {
		digest.Write(written[name])
	}
	counted := "ticket(s)"
	switch {
	case planned && len(written) == 1:
		counted = "plan item"
	case planned:
		counted = "plan items"
	}
	fmt.Fprintf(stdout, "\n%d %s in %s, raw bodies in %s\n", len(written), counted, out, cleanPath(out+"/raw"))
	fmt.Fprintf(stdout, "corpus: %s\n", strings.Join(names, " "))
	fmt.Fprintf(stdout, "corpus hash: %s\n", hex.EncodeToString(digest.Sum(nil)))
	if !planned {
		fmt.Fprintln(stdout, "\nblocked by, from each ticket's 'Blocked by' lines; native links are not fetched:")
		fmt.Fprintln(stdout, strings.Join(blockerTable(entries), "\n"))
	}
	if len(failed) > 0 {
		var listed []string
		for _, number := range failed {
			listed = append(listed, fmt.Sprintf("#%d", number))
		}
		fmt.Fprintf(stdout, "\nFAILED to fetch: %s\n", strings.Join(listed, ", "))
		fmt.Fprintln(stdout, "the corpus is partial — an artefact can read as unproduced because its producer is missing")
		return 2
	}
	return 0
}

func SystemCommands() Commands {
	return Commands{LookPath: exec.LookPath, Run: func(ctx context.Context, name string, args ...string) (string, error) {
		var stdout, stderr bytes.Buffer
		command := exec.CommandContext(ctx, name, args...)
		command.Stdout, command.Stderr = &stdout, &stderr
		if err := command.Run(); err != nil {
			reason := trimWhitespace(stderr.String())
			if exit := (*exec.ExitError)(nil); !errors.As(err, &exit) || reason == "" {
				reason = err.Error()
			}
			return "", fmt.Errorf("%s failed: %s", strings.Join(append([]string{name}, args...), " "), reason)
		}
		return stdout.String(), nil
	}}
}

// Prose after the leading refs is not an edge: "Blocked by: nothing — see #177"
// blocks on nothing. Also counts "Blocked by" lines with nothing after them, whose
// refs sit on the lines below where no table reads them.
func blockedBy(text string, project string) ([]string, int) {
	var references []string
	unread := 0
	for _, match := range blockedByLine.FindAllStringSubmatch(text, -1) {
		rest := trimWhitespace(strings.NewReplacer("*", "", "`", "").Replace(markdownLink.ReplaceAllString(match[1], "${1}")))
		if rest == "" {
			unread++
			continue
		}
		runes := []rune(rest)
		lead := leadingReferences.FindStringIndex(referenceShadow(runes))
		if lead == nil {
			continue
		}
		bare := []rune(parenthetical.ReplaceAllString(string(runes[:lead[1]]), ""))
		for _, span := range reference.FindAllStringIndex(referenceShadow(bare), -1) {
			references = append(references, string(bare[span[0]:span[1]]))
		}
	}
	own := []rune(strings.ToLower(project + "#"))
	var unique []string
	for _, found := range references {
		if strings.HasPrefix(strings.ToLower(found), string(own)) {
			found = "#" + string([]rune(found)[len(own):])
		}
		if !slices.Contains(unique, found) {
			unique = append(unique, found)
		}
	}
	return unique, unread
}

func blockerTable(entries []corpusEntry) []string {
	states := map[string]string{}
	for _, entry := range entries {
		states["#"+entry.name] = entry.state
	}
	var rows []string
	for _, entry := range entries {
		var blockers []string
		for _, blocker := range entry.blockers {
			state, ok := states[blocker]
			if !ok {
				state = "?"
			}
			blockers = append(blockers, blocker+" "+state)
		}
		listed := strings.Join(blockers, ", ")
		if listed == "" {
			listed = "-"
		}
		rows = append(rows, fmt.Sprintf("#%-5s %-6s %s", entry.name, entry.state, listed))
	}
	return rows
}

func childrenOf(ctx context.Context, stderr io.Writer, commands Commands, cli string, mapNumber int) ([]int, map[int]string, error) {
	native, err := nativeChildren(ctx, stderr, commands, cli, mapNumber)
	if err != nil {
		return nil, nil, err
	}
	text, err := textChildren(ctx, stderr, commands, cli, mapNumber)
	if err != nil {
		return nil, nil, err
	}
	gaps := map[int]string{}
	if cli == "gh" {
		for number := range native {
			if !text[number] {
				gaps[number] = fmt.Sprintf("is a sub-issue but never says 'Part of #%d'", mapNumber)
			}
		}
		for number := range text {
			if !native[number] {
				gaps[number] = fmt.Sprintf("says 'Part of #%d' but is not linked as a sub-issue", mapNumber)
			}
		}
	}
	maps.Copy(native, text)
	delete(native, mapNumber)
	return slices.Sorted(maps.Keys(native)), gaps, nil
}

// No empty or `.` parts; a leading `//` is kept, `///` collapses to `/`.
func cleanPath(path string) string {
	root := ""
	switch {
	case strings.HasPrefix(path, "//") && !strings.HasPrefix(path, "///"):
		root = "//"
	case strings.HasPrefix(path, "/"):
		root = "/"
	}
	var parts []string
	for part := range strings.SplitSeq(path, "/") {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	if root == "" && len(parts) == 0 {
		return "."
	}
	return root + strings.Join(parts, "/")
}

func collapseWhitespace(text string) string {
	var collapsed strings.Builder
	spaced := false
	for _, character := range trimWhitespace(text) {
		if isWhitespace(character) {
			spaced = true
			continue
		}
		if spaced {
			collapsed.WriteByte(' ')
			spaced = false
		}
		collapsed.WriteRune(character)
	}
	return collapsed.String()
}

// Corpus header and surrounding whitespace stripped: either reads as a failed write.
func comparableText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if lines := strings.Split(text, "\n"); corpusHeader.MatchString(lines[0]) {
		text = strings.Join(lines[1:], "\n")
	}
	return trimWhitespace(text)
}

func detectCLI(ctx context.Context, commands Commands, explicit string) (string, error) {
	if explicit != "" {
		if _, err := commands.LookPath(explicit); err != nil {
			return "", fmt.Errorf("%s is not on PATH", explicit)
		}
		return explicit, nil
	}
	remote := ""
	if output, err := commands.Run(ctx, "git", "remote", "get-url", "origin"); err == nil {
		remote = trimWhitespace(output)
	}
	shown, host := readRemote(remote)
	// Installed is not the same as correct: a GitLab CLI on PATH in a GitHub repo
	// fetches nothing and reports every ticket as a failure.
	for _, tracker := range [][2]string{{"gitlab", "glab"}, {"github", "gh"}} {
		if strings.Contains(host, tracker[0]) {
			if _, err := commands.LookPath(tracker[1]); err != nil {
				return "", fmt.Errorf("remote is %s but %s is not on PATH", tracker[0], tracker[1])
			}
			return tracker[1], nil
		}
	}
	var available []string
	for _, cli := range []string{"glab", "gh"} {
		if _, err := commands.LookPath(cli); err == nil {
			available = append(available, cli)
		}
	}
	switch len(available) {
	case 0:
		return "", errors.New("neither glab nor gh is on PATH")
	case 1:
		return available[0], nil
	}
	if shown == "" {
		shown = "unknown"
	}
	return "", fmt.Errorf("remote '%s' names no tracker and both CLIs are installed — pass --cli glab|gh", shown)
}

func fetch(ctx context.Context, commands Commands, cli string, number int) fetched {
	if cli == "glab" {
		var issue struct {
			Description *string `json:"description"`
			State       string  `json:"state"`
			Title       string  `json:"title"`
		}
		if err := runJSON(ctx, commands, &issue, "glab", "issue", "view", strconv.Itoa(number), "-F", "json"); err != nil {
			return fetched{err: err}
		}
		var notes []struct {
			Body   string `json:"body"`
			System bool   `json:"system"`
		}
		capped, err := paged(ctx, commands, &notes, "glab", fmt.Sprintf("projects/:id/issues/%d/notes", number))
		if err != nil {
			return fetched{err: err}
		}
		var comments []string
		for _, note := range notes {
			if !note.System {
				comments = append(comments, note.Body)
			}
		}
		return fetched{body: valueOf(issue.Description), capped: capped, comments: comments, state: stateLabel(issue.State), title: issue.Title}
	}
	var issue struct {
		Body  *string `json:"body"`
		State string  `json:"state"`
		Title string  `json:"title"`
	}
	if err := runJSON(ctx, commands, &issue, "gh", "issue", "view", strconv.Itoa(number), "--json", "title,state,body"); err != nil {
		return fetched{err: err}
	}
	var notes []struct {
		Body string `json:"body"`
	}
	capped, err := paged(ctx, commands, &notes, "gh", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/comments", number))
	if err != nil {
		return fetched{err: err}
	}
	var comments []string
	for _, note := range notes {
		comments = append(comments, note.Body)
	}
	return fetched{body: valueOf(issue.Body), capped: capped, comments: comments, state: stateLabel(issue.State), title: issue.Title}
}

func isWhitespace(character rune) bool {
	return unicode.IsSpace(character) || character >= 0x1c && character <= 0x1f
}

func isWordRune(character rune) bool {
	return character == '_' || unicode.IsLetter(character) || unicode.IsNumber(character)
}

func itemHash(text string) string {
	sum := sha1.Sum([]byte(collapseWhitespace(text)))
	return hex.EncodeToString(sum[:])
}

func manifest(entries []corpusEntry) []byte {
	var written strings.Builder
	written.WriteString("[")
	for index, entry := range entries {
		if index > 0 {
			written.WriteString(",")
		}
		written.WriteString("\n  {\n")
		if entry.alias != "" {
			fmt.Fprintf(&written, "    \"alias\": %s,\n", dumpsString(entry.alias))
		}
		fmt.Fprintf(&written, "    \"file\": %s,\n    \"hash\": %s,\n    \"key\": %s,\n    \"state\": %s\n  }", dumpsString(entry.file), dumpsString(entry.hash), dumpsString(entry.key), dumpsString(entry.state))
	}
	written.WriteString("\n]\n")
	return []byte(written.String())
}

func nativeChildren(ctx context.Context, stderr io.Writer, commands Commands, cli string, mapNumber int) (map[int]bool, error) {
	children := map[int]bool{}
	if cli != "gh" {
		return children, nil
	}
	var issues []struct {
		Number int `json:"number"`
	}
	capped, err := paged(ctx, commands, &issues, "gh", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/sub_issues", mapNumber))
	if err != nil {
		return nil, err
	}
	if capped {
		warnCapped(stderr, mapNumber, "sub-issues")
	}
	for _, issue := range issues {
		children[issue.Number] = true
	}
	return children, nil
}

// Every page, not the first: a decision recorded in note 123 is a decision.
// True when every page up to the cap came back full, so more may follow unread.
func paged[T any](ctx context.Context, commands Commands, items *[]T, cli string, path string) (bool, error) {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	for page := 1; page <= maximumPages; page++ {
		output, err := commands.Run(ctx, cli, "api", fmt.Sprintf("%s%sper_page=%d&page=%d", path, separator, pageSize, page))
		if err != nil {
			return false, err
		}
		if output == "" {
			output = "[]"
		}
		var raw any
		if err := json.Unmarshal([]byte(output), &raw); err != nil {
			return false, fmt.Errorf("%s api %s: %w", cli, path, err)
		}
		listed, ok := raw.([]any)
		if !ok || len(listed) == 0 {
			return false, nil
		}
		var batch []T
		if err := json.Unmarshal([]byte(output), &batch); err != nil {
			return false, fmt.Errorf("%s api %s: %w", cli, path, err)
		}
		*items = append(*items, batch...)
		if len(batch) < pageSize {
			return false, nil
		}
	}
	return true, nil
}

func placeholder(text string) string {
	for _, term := range placeholders {
		for offset := 0; ; {
			at := strings.Index(text[offset:], term)
			if at < 0 {
				break
			}
			end := offset + at + len(term)
			if next, _ := utf8.DecodeRuneInString(text[end:]); end == len(text) || !isWordRune(next) {
				return term
			}
			offset += at + 1
		}
	}
	return ""
}

// Open tickets only: a closed one holding the placeholder needs no map.
func placeholderTickets(ctx context.Context, stderr io.Writer, commands Commands, cli string) (map[int]string, error) {
	found := map[int]string{}
	for _, term := range placeholders {
		hits, err := search(ctx, stderr, commands, cli, term)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			if hit.state == "open" && placeholder(hit.body) == term {
				found[hit.number] = term
			}
		}
	}
	return found, nil
}

func planEntries(ctx context.Context, commands Commands, path string) ([]corpusEntry, error) {
	text, err := readText(path)
	if err != nil {
		return nil, err
	}
	repository, folder, relative, err := planLocation(ctx, commands, path)
	if err != nil {
		return nil, err
	}
	stem := strings.TrimSuffix(filepath.Base(path), ".md")
	var entries []corpusEntry
	for index, item := range planItems(text) {
		name := fmt.Sprintf("%s-%02d", stem, index+1)
		key := ""
		if item.action {
			key = plan.Name(item.title)
		}
		if key == "" {
			key = item.title
		}
		entry := corpusEntry{file: name + ".md", hash: itemHash(item.hashed), key: fmt.Sprintf("%s/%s: %s", repository, relative, key), name: name, raw: item.raw, state: item.state, title: item.title}
		if folder != repository {
			entry.alias = fmt.Sprintf("%s/%s: %s", folder, relative, key)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// The hashed text drops the checkbox marker, so ticking an item is no change to
// it. An unticked Inbox item is state inbox: not yet work, so never on the frontier.
func planItems(text string) []planEntry {
	type gathering struct {
		body  []string
		entry planEntry
		lines []string
	}
	var items []*gathering
	var current *gathering
	section := ""
	lines := splitLines(text, false)
	fenced := plan.Fenced(lines)
	for index, line := range lines {
		if fenced[index] {
			if current != nil {
				current.lines = append(current.lines, line)
				current.body = append(current.body, line)
			}
			continue
		}
		if level, title, ok := plan.Heading(line); ok {
			current = nil
			switch level {
			case 1:
				section = ""
			case 2:
				section = title
			}
			continue
		}
		if section != "Actions" && section != "Inbox" {
			continue
		}
		// Under Actions only a checkbox line is an item, as plan.Parse reads it.
		if match := planItem.FindStringSubmatch(line); match != nil && (section == "Inbox" || match[1] != "") {
			state := "open"
			if match[1] == "x" || match[1] == "X" {
				state = "closed"
			} else if section == "Inbox" {
				state = "inbox"
			}
			current = &gathering{lines: []string{line}, entry: planEntry{action: section == "Actions", state: state, title: collapseWhitespace(match[2])}}
			items = append(items, current)
		} else if current != nil && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || trimWhitespace(line) == "") {
			current.lines = append(current.lines, line)
			current.body = append(current.body, line)
		} else if trimWhitespace(line) != "" {
			current = nil
		}
	}
	entries := make([]planEntry, 0, len(items))
	for _, item := range items {
		item.entry.raw = strings.TrimRightFunc(strings.Join(item.lines, "\n"), isWhitespace)
		item.entry.hashed = strings.Join(append([]string{item.entry.title}, item.body...), "\n")
		entries = append(entries, item.entry)
	}
	return entries
}

func planLocation(ctx context.Context, commands Commands, path string) (string, string, string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", "", "", err
	}
	if resolved, resolveErr := filepath.EvalSymlinks(absolute); resolveErr == nil {
		absolute = resolved
	}
	output, err := commands.Run(ctx, "git", "-C", filepath.Dir(absolute), "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", "", fmt.Errorf("%s is not in a git repository", absolute)
	}
	top := trimWhitespace(output)
	resolvedTop := top
	if resolved, resolveErr := filepath.EvalSymlinks(top); resolveErr == nil {
		resolvedTop = resolved
	}
	relative, err := filepath.Rel(resolvedTop, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, "../") {
		return "", "", "", fmt.Errorf("%s is not inside %s", absolute, resolvedTop)
	}
	// The origin's project path, so a worktree or a clone under another folder keeps its rulings.
	if remote, remoteErr := commands.Run(ctx, "git", "-C", top, "remote", "get-url", "origin"); remoteErr == nil {
		if match := remoteProject.FindStringSubmatch(trimWhitespace(remote)); match != nil {
			return match[1], filepath.Base(top), filepath.ToSlash(relative), nil
		}
	}
	return filepath.Base(top), filepath.Base(top), filepath.ToSlash(relative), nil
}

// An empty corpus directory, or a refusal: files left by an earlier run would be
// read as part of this corpus, and two maps in one read as one map contradicting itself.
func prepare(out string, clean bool) error {
	if err := os.MkdirAll(filepath.Join(out, "raw"), 0o755); err != nil {
		return fmt.Errorf("cannot write to %s: %w", out, err)
	}
	written, err := manifestFiles(out)
	if err != nil {
		return err
	}
	var foreign, stale []string
	for _, directory := range []string{out, filepath.Join(out, "raw")} {
		listed, err := os.ReadDir(directory)
		if err != nil {
			return fmt.Errorf("cannot write to %s: %w", out, err)
		}
		for _, entry := range listed {
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			if written[entry.Name()] {
				stale = append(stale, filepath.Join(directory, entry.Name()))
			} else {
				foreign = append(foreign, filepath.Join(directory, entry.Name()))
			}
		}
	}
	// Refused with or without --clean: the corpus hash is over <out>/*.md.
	if len(foreign) > 0 {
		return fmt.Errorf("%s holds %d .md file(s) no corpus manifest there lists, which --clean keeps — remove them or name an empty directory", out, len(foreign))
	}
	if written != nil {
		stale = append(stale, filepath.Join(out, manifestName))
	}
	if len(stale) > 0 && !clean {
		return fmt.Errorf("%s already holds %d corpus file(s) — pass --clean, or name an empty directory", out, len(stale))
	}
	for _, path := range stale {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

// nil, not empty, with no manifest: prepare tells them apart.
func manifestFiles(out string) (map[string]bool, error) {
	content, err := os.ReadFile(filepath.Join(out, manifestName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", filepath.Join(out, manifestName), err)
	}
	var entries []struct {
		File string `json:"file"`
	}
	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, fmt.Errorf("%s is not a corpus manifest: %w", filepath.Join(out, manifestName), err)
	}
	files := map[string]bool{}
	for _, entry := range entries {
		if filepath.Base(entry.File) != entry.File || !strings.HasSuffix(entry.File, ".md") {
			return nil, fmt.Errorf("%s lists %q, not a corpus file name", filepath.Join(out, manifestName), entry.File)
		}
		files[entry.File] = true
	}
	return files, nil
}

// \r\n and a lone \r both read as \n; invalid UTF-8 is refused.
func readText(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", path, err)
	}
	if !utf8.Valid(content) {
		return "", fmt.Errorf("cannot read %s: not UTF-8", path)
	}
	return strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(string(content)), nil
}

// One ASCII byte per rune, so offsets index the rune slice: non-ASCII decimal
// digits become '0', other letters and numbers 'q', whitespace ' ', the rest
// '\x01'. Go's \b, \d, \s and \w are ASCII-only, and 'q' is in no pattern's literal.
func referenceShadow(runes []rune) string {
	shadow := make([]byte, len(runes))
	for index, character := range runes {
		switch {
		case character == '\v' || character >= 0x1c && character <= 0x1f:
			shadow[index] = ' '
		case character <= unicode.MaxASCII:
			shadow[index] = byte(character)
		case unicode.Is(unicode.Nd, character):
			shadow[index] = '0'
		case isWordRune(character):
			shadow[index] = 'q'
		case isWhitespace(character):
			shadow[index] = ' '
		default:
			shadow[index] = '\x01'
		}
	}
	return string(shadow)
}

func repeated(entries []corpusEntry, field func(corpusEntry) string) []string {
	counts := map[string]int{}
	for _, entry := range entries {
		counts[field(entry)]++
	}
	var duplicates []string
	for value, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, value)
		}
	}
	slices.Sort(duplicates)
	return duplicates
}

func runJSON(ctx context.Context, commands Commands, value any, name string, args ...string) error {
	output, err := commands.Run(ctx, name, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(output), value); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// The literal, checked locally: a search hit need only mention the words.
func saysPartOf(body string, mapNumber int) bool {
	literal := fmt.Sprintf("Part of #%d", mapNumber)
	for offset := 0; ; {
		at := strings.Index(body[offset:], literal)
		if at < 0 {
			return false
		}
		end := offset + at + len(literal)
		if next, _ := utf8.DecodeRuneInString(body[end:]); end == len(body) || !unicode.Is(unicode.Nd, next) {
			return true
		}
		offset += at + 1
	}
}

// Server-side search narrows, a local check decides: one call against fetching
// every issue in the project. gh's term is quoted to search it as a phrase.
func search(ctx context.Context, stderr io.Writer, commands Commands, cli string, term string) ([]searchHit, error) {
	var hits []searchHit
	if cli == "glab" {
		for page := 1; page <= maximumPages; page++ {
			var batch []struct {
				Description *string `json:"description"`
				IID         int     `json:"iid"`
				State       string  `json:"state"`
			}
			output, err := commands.Run(ctx, "glab", "issue", "list", "--all", "--search", term, "--per-page", strconv.Itoa(pageSize), "--page", strconv.Itoa(page), "-O", "json")
			if err != nil {
				return nil, err
			}
			if output != "" {
				if err := json.Unmarshal([]byte(output), &batch); err != nil {
					return nil, fmt.Errorf("glab issue list: %w", err)
				}
			}
			if len(batch) == 0 {
				return hits, nil
			}
			for _, issue := range batch {
				hits = append(hits, searchHit{body: valueOf(issue.Description), number: issue.IID, state: stateLabel(issue.State)})
			}
			if len(batch) < pageSize {
				return hits, nil
			}
		}
		warnSearchCapped(stderr, maximumPages*pageSize)
		return hits, nil
	}
	limit := pageSize * 5
	var batch []struct {
		Body   *string `json:"body"`
		Number int     `json:"number"`
		State  string  `json:"state"`
	}
	output, err := commands.Run(ctx, "gh", "issue", "list", "--state", "all", "--search", `"`+term+`"`, "--limit", strconv.Itoa(limit), "--json", "number,state,body")
	if err != nil {
		return nil, err
	}
	if output != "" {
		if err := json.Unmarshal([]byte(output), &batch); err != nil {
			return nil, fmt.Errorf("gh issue list: %w", err)
		}
	}
	for _, issue := range batch {
		hits = append(hits, searchHit{body: valueOf(issue.Body), number: issue.Number, state: stateLabel(issue.State)})
	}
	if len(batch) == limit {
		warnSearchCapped(stderr, limit)
	}
	return hits, nil
}

// A line ends at \r\n or at any one of \n \r \v \f \x1c \x1d \x1e \x85 \u2028 \u2029.
func splitLines(text string, keepEnds bool) []string {
	var lines []string
	start := 0
	for index := 0; index < len(text); {
		character, size := utf8.DecodeRuneInString(text[index:])
		end := index + size
		switch character {
		case '\r':
			if strings.HasPrefix(text[end:], "\n") {
				end++
			}
		case '\n', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
		default:
			index = end
			continue
		}
		if keepEnds {
			lines = append(lines, text[start:end])
		} else {
			lines = append(lines, text[start:index])
		}
		start, index = end, end
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

// One vocabulary across trackers, so counting open tickets is one grep.
func stateLabel(raw string) string {
	if lowered := strings.ToLower(raw); lowered == "closed" || lowered == "merged" {
		return "closed"
	}
	return "open"
}

func textChildren(ctx context.Context, stderr io.Writer, commands Commands, cli string, mapNumber int) (map[int]bool, error) {
	hits, err := search(ctx, stderr, commands, cli, fmt.Sprintf("Part of #%d", mapNumber))
	if err != nil {
		return nil, err
	}
	children := map[int]bool{}
	for _, hit := range hits {
		if saysPartOf(hit.body, mapNumber) {
			children[hit.number] = true
		}
	}
	return children, nil
}

// The hash covers comments too: a decision recorded in a note changes the ticket.
func ticketEntries(ctx context.Context, stderr io.Writer, commands Commands, options CorpusOptions) ([]corpusEntry, []int, error) {
	cli, err := detectCLI(ctx, commands, options.CLI)
	if err != nil {
		return nil, nil, err
	}
	project, err := trackerProject(ctx, commands)
	if err != nil {
		return nil, nil, err
	}
	var numbers []int
	gaps, found := map[int]string{}, map[int]string{}
	if options.Map != 0 {
		children, childGaps, err := childrenOf(ctx, stderr, commands, cli, options.Map)
		if err != nil {
			return nil, nil, err
		}
		numbers, gaps = append([]int{options.Map}, children...), childGaps
		if found, err = placeholderTickets(ctx, stderr, commands, cli); err != nil {
			return nil, nil, err
		}
	}
	var unique []int
	for _, number := range append(numbers, options.Tickets...) {
		if !slices.Contains(unique, number) {
			unique = append(unique, number)
		}
	}
	results := make([]fetched, len(unique))
	limit := make(chan struct{}, corpusWorkers)
	var waiting sync.WaitGroup
	for index, number := range unique {
		waiting.Go(func() {
			limit <- struct{}{}
			defer func() { <-limit }()
			results[index] = fetch(ctx, commands, cli, number)
		})
	}
	waiting.Wait()

	var entries []corpusEntry
	var failed []int
	states := map[int]string{}
	for index, number := range unique {
		result := results[index]
		if result.err != nil {
			fmt.Fprintf(stderr, "#%d: could not fetch — %v\n", number, result.err)
			failed = append(failed, number)
			continue
		}
		if result.capped {
			warnCapped(stderr, number, "comments")
		}
		text := strings.Join(append([]string{result.body}, result.comments...), "\n\n")
		blockers, unread := blockedBy(text, project)
		name := strconv.Itoa(number)
		entries = append(entries, corpusEntry{blockers: blockers, file: name + ".md", hash: itemHash(result.title + "\n" + text), key: project + "#" + name, name: name, raw: text, state: result.state, title: result.title})
		states[number] = result.state
		if unread > 0 {
			fmt.Fprintf(stderr, "warning: #%d has %d 'Blocked by' line(s) with no ref on them — read raw/%d.md\n", number, unread, number)
		}
		if term := placeholder(result.body); result.state == "open" && term != "" {
			found[number] = term
		}
	}
	for _, number := range slices.Sorted(maps.Keys(gaps)) {
		if states[number] != "closed" {
			fmt.Fprintf(stderr, "note: #%d %s\n", number, gaps[number])
		}
	}
	for _, number := range slices.Sorted(maps.Keys(found)) {
		where := "not in this corpus — pass it with --tickets if it belongs"
		if _, ok := states[number]; ok {
			where = "in this corpus"
		}
		fmt.Fprintf(stderr, "warning: #%d says the placeholder '%s', %s\n", number, found[number], where)
	}
	return entries, failed, nil
}

func trackerProject(ctx context.Context, commands Commands) (string, error) {
	output, err := commands.Run(ctx, "git", "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	remote := trimWhitespace(output)
	match := remoteProject.FindStringSubmatch(remote)
	if match == nil {
		shown, _ := readRemote(remote)
		return "", fmt.Errorf("cannot read a project path from remote '%s'", shown)
	}
	return match[1], nil
}

// readRemote returns the remote cut after its last @, safe for an error to
// print even where url.Parse fails, a path holding @ cut too, and its
// lowercased host.
func readRemote(remote string) (shown, host string) {
	scheme, rest, hasScheme := strings.Cut(remote, "://")
	// A token can hold ://, so only a URL scheme stays in front of the cut.
	if hasScheme = hasScheme && urlScheme.MatchString(scheme); !hasScheme {
		rest = remote
	}
	rest = rest[strings.LastIndex(rest, "@")+1:]
	if !hasScheme {
		head, _, found := strings.Cut(rest, ":")
		if found && !strings.Contains(head, "/") {
			host = head
		}
		return rest, strings.ToLower(host)
	}
	if parsed, err := url.Parse(remote); err == nil {
		host = parsed.Hostname()
	}
	return scheme + "://" + rest, strings.ToLower(host)
}

func trimWhitespace(text string) string {
	return strings.TrimFunc(text, isWhitespace)
}

func valueOf(text *string) string {
	if text == nil {
		return ""
	}
	return *text
}

func warnCapped(stderr io.Writer, number int, listed string) {
	fmt.Fprintf(stderr, "warning: #%d's %s reach the %d-item cap, so any beyond were not read\n", number, listed, maximumPages*pageSize)
}

func warnSearchCapped(stderr io.Writer, limit int) {
	fmt.Fprintf(stderr, "warning: hit the %d-issue search cap — pass --tickets for the full set\n", limit)
}
