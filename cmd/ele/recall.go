package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

// About 2k tokens at four bytes each: the most one recall or show adds to a session's context, every line counted.
const outputBudget = 8000

const (
	anchorLimit  = 2000
	regionLimit  = 3
	snippetLimit = 220
	textLimit    = 600
	trailerRoom  = 200
)

var commonWords = map[string]bool{"about": true, "after": true, "also": true, "been": true, "before": true, "could": true, "does": true, "each": true, "every": true, "from": true, "have": true, "into": true, "more": true, "note": true, "only": true, "should": true, "than": true, "that": true, "them": true, "then": true, "there": true, "they": true, "this": true, "what": true, "when": true, "which": true, "with": true, "would": true}

var relativeDays = regexp.MustCompile(`^(\d+)d$`)

// Entries low to high, inclusive.
type discussionRegion struct {
	anchor int
	high   int
	low    int
}

func clip(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if runes := []rune(text); len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return text
}

// One owner's work never surfaces in another's session unasked.
func currentOwner(ownerMap ownermap.OwnerMap) (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	project, owner := index.NewProjects(ownerMap).Resolve(directory)
	if owner == "" {
		return "", fmt.Errorf("no owner in owner-map.json covers %s (project %s); add a path rule or project entry naming a listed owner, or pass --owner <name> or --all", directory, project)
	}
	return owner, nil
}

// Note words weigh by rarity among entries up to the note, so a term the whole session repeats counts for little. Up to
// regionLimit anchors, best first, each more than separation apart. matched is false when no entry shares a word; the
// anchor is then the last entry at or before the note. None: nothing precedes it.
func discussionRegions(entries []index.Result, note notes.Note, separation int) (anchors []int, matched bool) {
	limit, err := time.Parse(time.RFC3339, note.Time)
	if err != nil {
		day, _ := time.ParseInLocation(time.DateOnly, note.Date, time.Local)
		limit = day.AddDate(0, 0, 1).Add(-time.Second)
	}
	var candidates []int
	for i, entry := range entries {
		at, parseErr := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if entry.Kind != index.KindNote && parseErr == nil && !at.After(limit) {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return nil, false
	}
	words := distinctiveWords(note.Body)
	texts := make(map[int]string, len(candidates))
	frequency := map[string]int{}
	for _, i := range candidates {
		texts[i] = strings.ToLower(entries[i].Text)
		for _, word := range words {
			if strings.Contains(texts[i], word) {
				frequency[word]++
			}
		}
	}
	scores := map[int]float64{}
	for _, i := range candidates {
		for _, word := range words {
			if strings.Contains(texts[i], word) {
				scores[i] += math.Log(float64(len(candidates)+1) / float64(frequency[word]))
			}
		}
	}
	ranked := slices.Clone(candidates)
	slices.SortStableFunc(ranked, func(a, b int) int { return cmp.Compare(scores[b], scores[a]) })
	for _, i := range ranked {
		// As of 2026-10-16 a quarter of the best kept every true part of three real notes' discussions.
		if scores[i] <= 0 || scores[i]*4 < scores[ranked[0]] || len(anchors) == regionLimit {
			break
		}
		if !slices.ContainsFunc(anchors, func(j int) bool { return max(i-j, j-i) <= separation }) {
			anchors = append(anchors, i)
		}
	}
	if len(anchors) == 0 {
		return []int{candidates[len(candidates)-1]}, false
	}
	return anchors, true
}

func distinctiveWords(text string) []string {
	seen := map[string]bool{}
	var words []string
	for _, word := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' }) {
		if utf8.RuneCountInString(word) < 4 || commonWords[word] || seen[word] {
			continue
		}
		seen[word] = true
		words = append(words, word)
	}
	return words
}

func findNote(id string) (notes.Note, error) {
	if id == "" || filepath.Base(id) != id {
		return notes.Note{}, fmt.Errorf("%q is not a note id", id)
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		return notes.Note{}, err
	}
	directory := filepath.Join(dataDirectory, "notes")
	owners, err := os.ReadDir(directory)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return notes.Note{}, err
	}
	for _, owner := range owners {
		if !owner.IsDir() {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(directory, owner.Name(), id+".md"))
		if errors.Is(readErr, fs.ErrNotExist) {
			continue
		}
		if readErr != nil {
			return notes.Note{}, readErr
		}
		return notes.Parse(filepath.Join("notes", owner.Name(), id+".md"), content)
	}
	return notes.Note{}, fmt.Errorf("no note %s in %s", id, directory)
}

func matchLine(result index.Result) string {
	kind := result.Kind
	if result.NoteKind != "" {
		kind = "note " + result.NoteKind
	}
	if result.Subagent {
		kind = "subagent " + kind
	}
	text := clip(result.Text, snippetLimit)
	if result.NoteID != "" {
		text = result.NoteID + " " + text
	}
	if result.SupersededBy != "" {
		text = "(superseded by " + result.SupersededBy + ") " + text
	}
	return fmt.Sprintf("  %-22s %s\n", kind, text)
}

func openIndex() (*index.Index, error) {
	dataDirectory, err := datadir.Path()
	if err != nil {
		return nil, err
	}
	return index.OpenChecked(filepath.Join(dataDirectory, "ele.db"))
}

// Flags may follow words: recall pydantic --all. Everything after "--" is words.
func parseInterspersed(flags *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	before, after, separated := args, []string(nil), false
	if at := slices.Index(args, "--"); at >= 0 {
		before, after, separated = args[:at], args[at+1:], true
	}
	for {
		if err := flags.Parse(before); err != nil {
			return nil, err
		}
		if flags.NArg() == 0 {
			break
		}
		positional = append(positional, flags.Arg(0))
		before = flags.Args()[1:]
	}
	if separated {
		positional = append(positional, after...)
	}
	return positional, nil
}

func parseSince(value string, now time.Time) (string, error) {
	if value == "" {
		return "", nil
	}
	if match := relativeDays.FindStringSubmatch(value); match != nil {
		days, err := strconv.Atoi(match[1])
		if err != nil {
			return "", err
		}
		return now.AddDate(0, 0, -days).UTC().Format(time.DateOnly), nil
	}
	if _, err := time.Parse(time.DateOnly, value); err != nil {
		return "", fmt.Errorf("--since takes YYYY-MM-DD or a number of days like 7d, not %q", value)
	}
	return value, nil
}

func plural(count int, one string, many string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, one)
	}
	return fmt.Sprintf("%d %s", count, many)
}

func present(parts ...string) string {
	var kept []string
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, " · ")
}

func recall(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("recall", stderr)
	all := flags.Bool("all", false, "search every owner")
	kinds := flags.String("kind", "", "comma-separated kinds: "+strings.Join(index.Kinds, ", "))
	limit := flags.Int("limit", 20, "most matches to return")
	noSync := flags.Bool("no-sync", false, "search without syncing first")
	owner := flags.String("owner", "", "search this owner instead of the project's or the current directory's")
	project := flags.String("project", "", "only this project, searched under its owner unless --owner or --all")
	since := flags.String("since", "", "only entries from this date (YYYY-MM-DD) or this many days back (7d)")
	words, err := parseInterspersed(flags, args)
	if err != nil {
		return rejectCall(flags, err, stdout, stderr)
	}
	text := strings.Join(words, " ")
	if strings.TrimSpace(text) == "" {
		return rejectCall(flags, nil, stdout, stderr)
	}
	if *all && *owner != "" {
		fmt.Fprintln(stderr, "ele: pass --all or --owner, not both")
		return 2
	}
	from, err := parseSince(*since, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 2
	}
	query := index.Query{Limit: *limit, Project: *project, Since: from, Text: text}
	for kind := range strings.SplitSeq(*kinds, ",") {
		kind = strings.TrimSpace(kind)
		if kind == "" {
			continue
		}
		if !slices.Contains(index.Kinds, kind) {
			fmt.Fprintf(stderr, "ele: unknown kind %q; use %s\n", kind, strings.Join(index.Kinds, ", "))
			return 2
		}
		query.Kinds = append(query.Kinds, kind)
	}

	dataDirectory, err := datadir.Path()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	ownerMap, err := ownermap.Load(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	if *owner != "" && !slices.Contains(ownerMap.Owners, *owner) {
		listed := "no owners"
		if len(ownerMap.Owners) > 0 {
			listed = strings.Join(ownerMap.Owners, ", ")
		}
		fmt.Fprintf(stderr, "ele: unknown owner %q; owner-map.json lists %s\n", *owner, listed)
		return 2
	}

	if !*noSync {
		switch syncErr := syncAll(ctx, io.Discard, false, false); {
		case errors.Is(syncErr, errSyncBusy):
		case syncErr != nil:
			fmt.Fprintf(stderr, "ele: sync failed, searching what is already indexed: %v\n", syncErr)
		}
	}
	searchIndex, err := openIndex()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	defer searchIndex.Close()
	switch {
	case *owner != "":
		query.Owner = *owner
	case *all:
	case *project != "":
		query.Owner, err = projectOwner(ctx, dataDirectory, ownerMap, *project)
		if err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
	default:
		query.Owner, err = currentOwner(ownerMap)
		if err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
	}

	results, err := searchIndex.Search(ctx, query)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	writeResults(stdout, text, query, results)
	return 0
}

func show(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("show", stderr)
	at := flags.String("at", "", "centre on the first entry at or after this time, such as an event's timestamp")
	around := flags.String("around", "", "centre on the first entry containing this text")
	window := flags.Int("context", 4, "entries either side of --around or --at, or of each part of a note's discussion (1 there unless set); when nothing matches a note, twice this many before it")
	noteID := flags.String("note", "", "open the session a note came from, at its discussion")
	positional, err := parseInterspersed(flags, args)
	if err != nil {
		return rejectCall(flags, err, stdout, stderr)
	}
	if (*noteID == "" && len(positional) != 1) || (*noteID != "" && (len(positional) != 0 || *around != "" || *at != "")) || (*around != "" && *at != "") {
		return rejectCall(flags, nil, stdout, stderr)
	}
	if *window < 0 {
		fmt.Fprintln(stderr, "ele: --context must be 0 or more")
		return 2
	}
	searchIndex, err := openIndex()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	defer searchIndex.Close()

	var note notes.Note
	session := ""
	if *noteID != "" {
		if note, err = findNote(*noteID); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
		if note.Session == "" {
			fmt.Fprintf(stderr, "ele: note %s records no session\n", note.ID)
			return 1
		}
		session = note.Session
	} else {
		session = positional[0]
	}
	var entries []index.Result
	if *noteID != "" {
		entries, err = searchIndex.SessionExactly(ctx, session)
	} else {
		entries, err = searchIndex.Session(ctx, session)
	}
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	switch {
	case *noteID != "" && len(entries) == 0:
		fmt.Fprintf(stderr, "ele: note %s's session %s is not indexed\n", note.ID, note.Session)
		return 1
	case len(entries) == 0:
		fmt.Fprintf(stderr, "ele: no indexed session starts with %q\n", session)
		return 1
	}
	if *noteID != "" {
		neighbours := 1
		flags.Visit(func(set *flag.Flag) {
			if set.Name == "context" {
				neighbours = *window
			}
		})
		anchors, matched := discussionRegions(entries, note, 2*neighbours)
		if len(anchors) == 0 {
			fmt.Fprintf(stderr, "ele: no entry in session %s precedes note %s; the session id may be incomplete, or its transcript not yet synced (ele sync)\n", note.Session, note.ID)
			return 1
		}
		var regions []discussionRegion
		for _, anchor := range anchors {
			regions = append(regions, discussionRegion{anchor: anchor, high: min(anchor+neighbours, len(entries)-1), low: max(anchor-neighbours, 0)})
		}
		// Nothing matched: the discussion led up to the note.
		if !matched {
			regions[0].high, regions[0].low = anchors[0], max(anchors[0]-2**window, 0)
		}
		writeDiscussion(stdout, entries, note, regions)
		return 0
	}
	start, end := 0, len(entries)
	if *around != "" {
		needle := strings.ToLower(*around)
		// A note shares words with its discussion; prefer the discussion.
		found, firstNote := -1, -1
		for i, entry := range entries {
			if !strings.Contains(strings.ToLower(entry.Text), needle) {
				continue
			}
			if entry.Kind != index.KindNote {
				found = i
				break
			}
			if firstNote < 0 {
				firstNote = i
			}
		}
		if found < 0 {
			found = firstNote
		}
		if found < 0 {
			fmt.Fprintf(stderr, "ele: no entry in session %s contains %q\n", entries[0].Session, *around)
			return 1
		}
		start, end = max(found-*window, 0), min(found+*window+1, len(entries))
	}
	if *at != "" {
		found := slices.IndexFunc(entries, func(entry index.Result) bool { return entry.Timestamp >= *at })
		if found < 0 {
			fmt.Fprintf(stderr, "ele: no entry in session %s is at or after %s\n", entries[0].Session, *at)
			return 1
		}
		start, end = max(found-*window, 0), min(found+*window+1, len(entries))
	}

	first := entries[0]
	heading := present(first.Session, first.Owner, first.Project, first.Profile)
	written := len(fmt.Sprintf("Session %s · entries %d–%d of %d\n", heading, len(entries), len(entries), len(entries)))
	line := func(i int) string {
		return fmt.Sprintf("%s  %-12s %s\n", entries[i].Timestamp, entries[i].Kind, clip(entries[i].Text, textLimit))
	}
	var lines []string
	for i := start; i < end; i++ {
		next := line(i)
		if written+len(next) > outputBudget-trailerRoom {
			break
		}
		lines = append(lines, next)
		written += len(next)
	}
	fmt.Fprintf(stdout, "Session %s · entries %d–%d of %d\n", heading, start+1, start+len(lines), len(entries))
	fmt.Fprint(stdout, strings.Join(lines, ""))
	if hidden := end - start - len(lines); hidden > 0 {
		fmt.Fprintf(stdout, "… %s not shown; narrow with --around and --context\n", plural(hidden, "more entry", "more entries"))
	}
	return 0
}

// Anchors first, each in up to anchorLimit characters and an equal share of the budget's bytes, so wide characters cannot
// overrun it; then each region grows by one entry a side per round, best region first, while the budget lasts.
func writeDiscussion(stdout io.Writer, entries []index.Result, note notes.Note, regions []discussionRegion) {
	first := entries[0]
	heading := fmt.Sprintf("Session %s · %d entries · note %s's discussion\n", present(first.Session, first.Owner, first.Project, first.Profile), len(entries), note.ID)
	render := func(i int, limit int) string {
		return fmt.Sprintf("%s  %-12s %s\n", entries[i].Timestamp, entries[i].Kind, clip(entries[i].Text, limit))
	}
	const regionHeadingLength = len("— entries 00000–00000\n")
	budget := outputBudget - trailerRoom - len(heading)
	share := budget/len(regions) - regionHeadingLength
	shown := map[int]string{}
	bounds := map[int]discussionRegion{}
	kept := make([]discussionRegion, 0, len(regions))
	for _, region := range regions {
		limit := anchorLimit
		text := render(region.anchor, limit)
		for len(text) > share && limit > 0 {
			limit = max(limit*share/len(text)-1, 0)
			text = render(region.anchor, limit)
		}
		shown[region.anchor] = text
		budget -= len(text) + regionHeadingLength
		bounds[region.anchor] = region
		kept = append(kept, discussionRegion{anchor: region.anchor, high: region.anchor, low: region.anchor})
	}
	for grew := true; grew; {
		grew = false
		for i := range kept {
			limits := bounds[kept[i].anchor]
			if kept[i].low > limits.low {
				if text := render(kept[i].low-1, textLimit); len(text) <= budget {
					kept[i].low--
					shown[kept[i].low], budget, grew = text, budget-len(text), true
				}
			}
			if kept[i].high < limits.high {
				if text := render(kept[i].high+1, textLimit); len(text) <= budget {
					kept[i].high++
					shown[kept[i].high], budget, grew = text, budget-len(text), true
				}
			}
		}
	}

	slices.SortFunc(kept, func(a, b discussionRegion) int { return cmp.Compare(a.anchor, b.anchor) })
	var out strings.Builder
	out.WriteString(heading)
	hidden := 0
	for _, region := range kept {
		limits := bounds[region.anchor]
		hidden += (region.low - limits.low) + (limits.high - region.high)
		fmt.Fprintf(&out, "— entries %d–%d\n", region.low+1, region.high+1)
		for i := region.low; i <= region.high; i++ {
			out.WriteString(shown[i])
		}
	}
	if hidden > 0 {
		fmt.Fprintf(&out, "… %s not shown (output limit); ele show %s --around <text> reaches them\n", plural(hidden, "more entry", "more entries"), note.Session[:min(len(note.Session), sessionPrefix)])
	}
	fmt.Fprint(stdout, out.String())
}

func writeResults(stdout io.Writer, text string, query index.Query, results []index.Result) {
	scope := "all owners"
	if query.Owner != "" {
		scope = "owner " + query.Owner
	}
	if len(results) == 0 {
		fmt.Fprintf(stdout, "No matches for %q in %s.\n", clip(text, snippetLimit), scope)
		return
	}

	var order []string
	groups := map[string][]index.Result{}
	for i, result := range results {
		// One heading per session-less result; a shared "" group would show the first one's date and project above the rest.
		key := result.Session
		if key == "" {
			key = fmt.Sprintf("\x00%d", i)
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], result)
	}

	count := plural(len(results), "match", "matches")
	if limit := query.Limit; limit > 0 && len(results) == limit {
		count = "Top " + count + " (--limit)"
	}
	header := fmt.Sprintf("%s for %q in %s, best first:\n", count, clip(text, snippetLimit), scope)
	fmt.Fprint(stdout, header)
	written, shown := len(header), 0
sessions:
	for _, session := range order {
		group := groups[session]
		first := group[0]
		heading := "\n" + present(first.Session, first.Timestamp[:min(len(first.Timestamp), 10)], first.Owner, first.Project) + "\n"
		line := matchLine(group[0])
		if written+len(heading)+len(line) > outputBudget-trailerRoom {
			break
		}
		fmt.Fprint(stdout, heading)
		written += len(heading)
		for _, result := range group {
			line := matchLine(result)
			if written+len(line) > outputBudget-trailerRoom {
				break sessions
			}
			fmt.Fprint(stdout, line)
			written += len(line)
			shown++
		}
	}
	if shown < len(results) {
		fmt.Fprintf(stdout, "\n… %s not shown; narrow with --kind, --project or --since\n", plural(len(results)-shown, "more match", "more matches"))
	}
	if slices.ContainsFunc(results, func(result index.Result) bool { return result.Session != "" }) {
		fmt.Fprintln(stdout, "\nExpand a session: ele show <session> --around <word>")
	}
}
