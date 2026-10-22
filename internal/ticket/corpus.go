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
	// Page cap: a pagination bug cannot loop forever; 20 pages outrun any ticket's notes.
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
	comments []string
	err      error
	state    string
	title    string
}

type planEntry struct {
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
	if len(entries) == 0 {
		fmt.Fprintln(stderr, "ele: no tickets written")
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
		fmt.Fprintf(stdout, "#%-5s %-6s %s %s\n", entry.name, entry.state, entry.hash[:12], entry.title)
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
	fmt.Fprintf(stdout, "\n%d ticket(s) in %s, raw bodies in %s\n", len(written), out, cleanPath(out+"/raw"))
	fmt.Fprintf(stdout, "corpus: %s\n", strings.Join(names, " "))
	fmt.Fprintf(stdout, "corpus hash: %s\n", hex.EncodeToString(digest.Sum(nil)))
	if len(options.Plans) == 0 {
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
			if exit := (*exec.ExitError)(nil); !errors.As(err, &exit) {
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
	native, err := nativeChildren(ctx, commands, cli, mapNumber)
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
		remote = strings.ToLower(trimWhitespace(output))
	}
	// Installed is not the same as correct: a GitLab CLI on PATH in a GitHub repo
	// fetches nothing and reports every ticket as a failure.
	for _, tracker := range [][2]string{{"gitlab", "glab"}, {"github", "gh"}} {
		if strings.Contains(remote, tracker[0]) {
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
	if remote == "" {
		remote = "unknown"
	}
	return "", fmt.Errorf("remote '%s' names no tracker and both CLIs are installed — pass --cli glab|gh", remote)
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
		if err := paged(ctx, commands, &notes, "glab", fmt.Sprintf("projects/:id/issues/%d/notes", number)); err != nil {
			return fetched{err: err}
		}
		var comments []string
		for _, note := range notes {
			if !note.System {
				comments = append(comments, note.Body)
			}
		}
		return fetched{body: valueOf(issue.Description), comments: comments, state: stateLabel(issue.State), title: issue.Title}
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
	if err := paged(ctx, commands, &notes, "gh", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/comments", number)); err != nil {
		return fetched{err: err}
	}
	var comments []string
	for _, note := range notes {
		comments = append(comments, note.Body)
	}
	return fetched{body: valueOf(issue.Body), comments: comments, state: stateLabel(issue.State), title: issue.Title}
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
		fmt.Fprintf(&written, "    \"file\": %s,\n    \"hash\": %s,\n    \"key\": %s,\n    \"state\": %s\n  }", dumpsString(entry.file), dumpsString(entry.hash), dumpsString(entry.key), dumpsString(entry.state))
	}
	written.WriteString("\n]\n")
	return []byte(written.String())
}

func nativeChildren(ctx context.Context, commands Commands, cli string, mapNumber int) (map[int]bool, error) {
	children := map[int]bool{}
	if cli != "gh" {
		return children, nil
	}
	var issues []struct {
		Number int `json:"number"`
	}
	if err := paged(ctx, commands, &issues, "gh", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/sub_issues", mapNumber)); err != nil {
		return nil, err
	}
	for _, issue := range issues {
		children[issue.Number] = true
	}
	return children, nil
}

// Every page, not the first: a decision recorded in note 123 is a decision.
func paged[T any](ctx context.Context, commands Commands, items *[]T, cli string, path string) error {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	for page := 1; page <= maximumPages; page++ {
		output, err := commands.Run(ctx, cli, "api", fmt.Sprintf("%s%sper_page=%d&page=%d", path, separator, pageSize, page))
		if err != nil {
			return err
		}
		if output == "" {
			output = "[]"
		}
		var raw any
		if err := json.Unmarshal([]byte(output), &raw); err != nil {
			return fmt.Errorf("%s api %s: %w", cli, path, err)
		}
		listed, ok := raw.([]any)
		if !ok || len(listed) == 0 {
			return nil
		}
		var batch []T
		if err := json.Unmarshal([]byte(output), &batch); err != nil {
			return fmt.Errorf("%s api %s: %w", cli, path, err)
		}
		*items = append(*items, batch...)
		if len(batch) < pageSize {
			return nil
		}
	}
	return nil
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
	repository, relative, err := planLocation(ctx, commands, path)
	if err != nil {
		return nil, err
	}
	stem := strings.TrimSuffix(filepath.Base(path), ".md")
	var entries []corpusEntry
	for index, item := range planItems(text) {
		name := fmt.Sprintf("%s-%02d", stem, index+1)
		key := plan.Name(item.title)
		if key == "" {
			key = item.title
		}
		entries = append(entries, corpusEntry{file: name + ".md", hash: itemHash(item.hashed), key: fmt.Sprintf("%s/%s: %s", repository, relative, key), name: name, raw: item.raw, state: item.state, title: item.title})
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
	for _, line := range splitLines(text, false) {
		if heading, ok := strings.CutPrefix(line, "## "); ok {
			section, current = trimWhitespace(heading), nil
			continue
		}
		if section != "Actions" && section != "Inbox" {
			continue
		}
		if match := planItem.FindStringSubmatch(line); match != nil {
			state := "open"
			if match[1] == "x" || match[1] == "X" {
				state = "closed"
			} else if section == "Inbox" {
				state = "inbox"
			}
			current = &gathering{lines: []string{line}, entry: planEntry{state: state, title: collapseWhitespace(match[2])}}
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

func planLocation(ctx context.Context, commands Commands, plan string) (string, string, error) {
	absolute, err := filepath.Abs(plan)
	if err != nil {
		return "", "", err
	}
	if resolved, resolveErr := filepath.EvalSymlinks(absolute); resolveErr == nil {
		absolute = resolved
	}
	output, err := commands.Run(ctx, "git", "-C", filepath.Dir(absolute), "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", fmt.Errorf("%s is not in a git repository", absolute)
	}
	top := trimWhitespace(output)
	resolvedTop := top
	if resolved, resolveErr := filepath.EvalSymlinks(top); resolveErr == nil {
		resolvedTop = resolved
	}
	relative, err := filepath.Rel(resolvedTop, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, "../") {
		return "", "", fmt.Errorf("%s is not inside %s", absolute, resolvedTop)
	}
	return filepath.Base(top), filepath.ToSlash(relative), nil
}

// An empty corpus directory, or a refusal: files left by an earlier run would be
// read as part of this corpus, and two maps in one read as one map contradicting itself.
func prepare(out string, clean bool) error {
	if err := os.MkdirAll(filepath.Join(out, "raw"), 0o755); err != nil {
		return fmt.Errorf("cannot write to %s: %w", out, err)
	}
	var stale []string
	for _, directory := range []string{out, filepath.Join(out, "raw")} {
		listed, err := os.ReadDir(directory)
		if err != nil {
			return fmt.Errorf("cannot write to %s: %w", out, err)
		}
		for _, entry := range listed {
			if strings.HasSuffix(entry.Name(), ".md") {
				stale = append(stale, filepath.Join(directory, entry.Name()))
			}
		}
	}
	if _, err := os.Lstat(filepath.Join(out, manifestName)); err == nil {
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
				break
			}
			for _, issue := range batch {
				hits = append(hits, searchHit{body: valueOf(issue.Description), number: issue.IID, state: stateLabel(issue.State)})
			}
			if len(batch) < pageSize {
				break
			}
		}
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
		fmt.Fprintf(stderr, "warning: hit the %d-issue search cap — pass --tickets for the full set\n", limit)
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
		return "", fmt.Errorf("cannot read a project path from remote '%s'", remote)
	}
	return match[1], nil
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
