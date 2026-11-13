package plan

import (
	"bufio"
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	claimAge          = 14 * 24 * time.Hour
	inboxAge          = 7 * 24 * time.Hour
	sectionActions    = "Actions"
	sectionInbox      = "Inbox"
	sectionOutOfScope = "Out of scope"
)

var (
	ErrInvalid = errors.New("invalid plan")

	asOf     = regexp.MustCompile(`(?i)\bas of (\d{4}-\d{2}-\d{2})`)
	checkbox = regexp.MustCompile(`^- \[([ xX])\] (.*)$`)
	edgeName = regexp.MustCompile(`^(?:([a-z0-9]+(?:-[a-z0-9]+)*)/)?([a-z0-9]+(?:-[a-z0-9]+)*)$`)
	// Eight hex digits for a session prefix, so prose such as "ele show added" passes.
	eleShow    = regexp.MustCompile(`\bele show(?: --(?:around|at|context) (?:"[^"]*"|'[^']*'|\S+))* (?:--note (\d{8}-[0-9a-f]{6}|[0-9a-f]{8,})|([0-9a-f]{8,}))\b`)
	heading    = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
	inboxDate  = regexp.MustCompile(`^- (\d{4}-\d{2}-\d{2})\b`)
	itemName   = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*): `)
	listMarker = regexp.MustCompile(`^(?:[-*+]|\d{1,9}[.)]) +`)
	// A deleted line can join an item's header onto the end of the line before.
	mergedBox = regexp.MustCompile(`\S.*- \[[ xX]\] `)
	noteID    = regexp.MustCompile(`\b\d{8}-[0-9a-f]{6}\b`)
	otherBox  = regexp.MustCompile(`^(?:[*+]|\d+[.)]) \[[ xX]\] `)
	otherList = regexp.MustCompile(`^(?:[*+]|\d+[.)]) `)
	sessionID = regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
)

type Blocker int

const (
	// First, so a name absent from a lookup map reads as missing.
	BlockerMissing Blocker = iota
	// A plan name this project has not started, closed or failed to read: a draft, or a plan in another repository.
	BlockerNoPlan
	// An item a closed plan never ticked.
	BlockerAbandoned
	BlockerDone
	BlockerOpen
)

type Citation struct {
	Line       int
	References []Reference
}

type Counts struct {
	Blocked int
	Done    int
	Inbox   int
	Open    int
}

// Dated has a zero Date for none, or for one that does not parse.
type Dated struct {
	Date time.Time
	Line int
}

type Edge struct {
	Name string
	Plan string
}

type Item struct {
	After time.Time
	// The malformed value as written, two quote characters when empty; non-empty holds the item.
	AfterInvalid string
	Blocked      []string
	Body         []string
	Done         bool
	DoneWhen     bool
	Edges        []Edge
	Flags        []string
	Held         bool
	Line         int
	Name         string
	Prose        []string
	Title        string
}

type Lookup func(edge Edge) Blocker

type Plan struct {
	Actions   []Item
	Citations []Citation
	Claims    []Dated
	Created   string
	Inbox     []Dated
	Malformed []Warning
	Spec      string
}

// Reference is an id as cited; a session's may be a prefix.
type Reference struct {
	ID   string
	Kind ReferenceKind
}

type ReferenceKind int

const (
	ReferenceNote ReferenceKind = iota
	ReferenceSession
)

type Warning struct {
	Line int
	Text string
}

// The content column of each list item a line may sit in, outermost first.
type containers []int

// enter reads a line outside a fence and returns the column its fence indent counts from.
func (c *containers) enter(line string) int {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return c.top()
	}
	indent := column(line)
	for len(*c) > 0 && indent < (*c)[len(*c)-1] {
		*c = (*c)[:len(*c)-1]
	}
	base := c.top()
	if marker := listMarker.FindString(trimmed); marker != "" {
		*c = append(*c, indent+len(marker))
	}
	return base
}

func (c containers) top() int {
	if len(c) == 0 {
		return 0
	}
	return c[len(c)-1]
}

func Parse(content []byte) (Plan, error) {
	invalid := func(format string, arguments ...any) (Plan, error) {
		return Plan{}, fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, arguments...))
	}
	scanner := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))))
	scanner.Buffer(nil, 1<<20)
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return invalid("first line is not ---")
	}
	var plan Plan
	kind := ""
	fields := map[string]*string{"created": &plan.Created, "kind": &kind, "spec": &plan.Spec}
	seen := map[string]bool{}
	closed := false
	number, createdLine := 1, 0
	for scanner.Scan() {
		number++
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return invalid("front matter line %q is not key: value", line)
		}
		key = strings.TrimSpace(key)
		field, known := fields[key]
		if !known {
			plan.Malformed = append(plan.Malformed, Warning{Line: number, Text: fmt.Sprintf("unknown front matter key %q; ignored", key)})
			continue
		}
		if seen[key] {
			return invalid("front matter key %q appears twice", key)
		}
		seen[key] = true
		*field = strings.TrimSpace(value)
		if key == "created" {
			createdLine = number
		}
	}
	switch {
	case !closed:
		return invalid("front matter never closes")
	case kind != "plan":
		return invalid("kind %q is not plan", kind)
	}
	if _, err := time.Parse(time.DateOnly, plan.Created); plan.Created != "" && err != nil {
		plan.Malformed = append(plan.Malformed, Warning{Line: createdLine, Text: fmt.Sprintf("created %q is not YYYY-MM-DD; ignored", plan.Created)})
		plan.Created = ""
	}

	section, fence, actions := "", "", false
	var item *Item
	var lists containers
	for scanner.Scan() {
		number++
		line := scanner.Text()
		if references := cited(line); len(references) > 0 {
			plan.Citations = append(plan.Citations, Citation{Line: number, References: references})
		}
		inside := fence != ""
		base := lists.top()
		if !inside {
			base = lists.enter(line)
		}
		fence = stepFence(fence, line, base)
		fenceLine := inside || fence != ""
		if section == sectionActions && !fenceLine && mergedBox.MatchString(withoutCode(strings.TrimSpace(line))) {
			plan.Malformed = append(plan.Malformed, Warning{Line: number, Text: "a checkbox follows other text, so ele reads no item there; put it on a line of its own"})
		}
		if item != nil && (strings.TrimSpace(line) == "" || fenceLine || line != strings.TrimLeft(line, " \t")) {
			item.Body = append(item.Body, line)
			if !item.Done && !fenceLine {
				plan.claim(line, number)
			}
			continue
		}
		item = nil
		if fenceLine {
			continue
		}
		if match := heading.FindStringSubmatch(line); match != nil {
			switch len(match[1]) {
			case 1:
				section = ""
			case 2:
				section = match[2]
				actions = actions || section == sectionActions
			}
			if section != sectionOutOfScope {
				plan.claim(line, number)
			}
			continue
		}
		switch {
		case section == sectionInbox && strings.HasPrefix(line, "- "):
			entry := Dated{Line: number}
			if match := inboxDate.FindStringSubmatch(line); match != nil {
				entry.Date, _ = time.ParseInLocation(time.DateOnly, match[1], time.Local)
			}
			plan.Inbox = append(plan.Inbox, entry)
		case section == sectionInbox && otherList.MatchString(line):
			plan.Malformed = append(plan.Malformed, Warning{Line: number, Text: "inbox item is not a - line, so ele does not count it"})
		case section == sectionActions && otherBox.MatchString(line):
			plan.Malformed = append(plan.Malformed, Warning{Line: number, Text: "item is not a - [ ] or - [x] line, so ele does not read it"})
		case section == sectionActions:
			if match := checkbox.FindStringSubmatch(line); match != nil {
				title := strings.TrimSpace(match[2])
				plan.Actions = append(plan.Actions, Item{Done: match[1] != " ", Line: number, Name: Name(title), Title: title})
				item = &plan.Actions[len(plan.Actions)-1]
			}
		}
		if (item == nil || !item.Done) && section != sectionOutOfScope {
			plan.claim(line, number)
		}
	}
	if err := scanner.Err(); err != nil {
		return Plan{}, err
	}
	if !actions {
		return invalid("no ## %s section", sectionActions)
	}
	named := map[string]int{}
	for _, action := range plan.Actions {
		if action.Name == "" {
			continue
		}
		if line, ok := named[action.Name]; ok {
			return invalid("item name %q on lines %d and %d", action.Name, line, action.Line)
		}
		named[action.Name] = action.Line
	}
	for at := range plan.Actions {
		action := &plan.Actions[at]
		for len(action.Body) > 0 && strings.TrimSpace(action.Body[len(action.Body)-1]) == "" {
			action.Body = action.Body[:len(action.Body)-1]
		}
		label := cmp.Or(action.Name, "action")
		bodyFence := ""
		// The header's checkbox is its list item's content.
		lists := containers{len("- ")}
		for offset := 0; offset < len(action.Body); offset++ {
			line := action.Body[offset]
			inside := bodyFence != ""
			base := lists.top()
			if !inside {
				base = lists.enter(line)
			}
			if bodyFence = stepFence(bodyFence, line, base); inside || bodyFence != "" {
				continue
			}
			number := action.Line + offset + 1
			trimmed := strings.TrimPrefix(strings.TrimSpace(line), "- ")
			action.DoneWhen = action.DoneWhen || strings.HasPrefix(strings.ToLower(trimmed), "done when")
			if reason, ok := cutFold(trimmed, "blocked:"); ok {
				reason = strings.TrimSpace(reason)
				// A formatter wraps a long list after a comma.
				for strings.HasSuffix(reason, ",") && !strings.Contains(reason, ";") && offset+1 < len(action.Body) && continues(action.Body[offset+1]) {
					offset++
					reason += " " + strings.TrimSpace(action.Body[offset])
				}
				action.Blocked = append(action.Blocked, reason)
				head, _, _ := strings.Cut(reason, ";")
				switch found := edges(reason); {
				case found != nil:
					action.Edges = append(action.Edges, found...)
				case strings.TrimSpace(head) == "":
					action.Prose = append(action.Prose, reason)
					plan.Malformed = append(plan.Malformed, Warning{Line: number, Text: label + ": blocked: names no item; it holds an open item until fixed"})
				case strings.Contains(head, ","):
					action.Prose = append(action.Prose, reason)
					plan.Malformed = append(plan.Malformed, Warning{Line: number, Text: fmt.Sprintf("%s: blocked: %q is not a list of item names; it holds an open item until fixed", label, strings.TrimSpace(head))})
				default:
					action.Prose = append(action.Prose, reason)
				}
			}
			if value, ok := cutFold(trimmed, "after:"); ok {
				if !action.After.IsZero() || action.AfterInvalid != "" {
					return invalid("a second after: on line %d", number)
				}
				value = strings.TrimSpace(value)
				after, err := parseAfter(value)
				if err != nil {
					plan.Malformed = append(plan.Malformed, Warning{Line: number, Text: fmt.Sprintf("%s: after %q is neither YYYY-MM-DD nor YYYY-MM-DD HH:MMZ; it holds an open item until fixed", label, value)})
					action.AfterInvalid = cmp.Or(value, `""`)
					continue
				}
				action.After = after
			}
		}
		// Unresolved until Resolve: every edge and date still holds the item back.
		action.Held = !action.Done && (len(action.Blocked) > 0 || !action.After.IsZero() || action.AfterInvalid != "")
	}
	return plan, nil
}

func (e Edge) String() string {
	if e.Plan == "" {
		return e.Name
	}
	return e.Plan + "/" + e.Name
}

// Fenced marks each line Parse reads as a fence or inside one.
func Fenced(lines []string) []bool {
	fenced := make([]bool, len(lines))
	fence := ""
	var lists containers
	for index, line := range lines {
		inside := fence != ""
		base := lists.top()
		if !inside {
			base = lists.enter(line)
		}
		fence = stepFence(fence, line, base)
		fenced[index] = inside || fence != ""
	}
	return fenced
}

// Heading reads a line as Parse reads a heading: its level and its text.
func Heading(line string) (int, string, bool) {
	match := heading.FindStringSubmatch(line)
	if match == nil {
		return 0, "", false
	}
	return len(match[1]), match[2], true
}

func Name(title string) string {
	if match := itemName.FindStringSubmatch(title); match != nil {
		return match[1]
	}
	return ""
}

// Check runs on a resolved plan: it reports the plan's flags beside its own warnings.
func (p Plan) Check(now time.Time) []Warning {
	warnings := slices.Clone(p.Malformed)
	last := len(p.Actions) - 1
	for at, action := range p.Actions {
		label := action.Name
		if label == "" {
			label = "action"
		}
		for _, flag := range action.Flags {
			warnings = append(warnings, Warning{Line: action.Line, Text: label + ": " + flag})
		}
		if !action.Done && at != last && action.Name == "" {
			warnings = append(warnings, Warning{Line: action.Line, Text: "action has no name; lead its title with a kebab-case name and a colon"})
		}
		if !action.Done && at != last && !action.DoneWhen {
			warnings = append(warnings, Warning{Line: action.Line, Text: label + ": no done when line"})
		}
	}
	if last >= 1 && p.Actions[last].Done && slices.ContainsFunc(p.Actions[:last], func(action Item) bool { return !action.Done }) {
		warnings = append(warnings, Warning{Line: p.Actions[last].Line, Text: "the closing action is ticked while other actions are open"})
	}
	for _, citation := range p.Citations {
		warnings = append(warnings, Warning{Line: citation.Line, Text: "cites ele data, which resolves only in its author's data directory; delete provenance, describe what was observed, and move a dependency into the plan, superseding its note"})
	}
	for _, claim := range p.Claims {
		if age := now.Sub(claim.Date); age > claimAge {
			warnings = append(warnings, Warning{Line: claim.Line, Text: fmt.Sprintf("as of %s is %s old; re-measure it", claim.Date.Format(time.DateOnly), daysOld(age))})
		}
	}
	for _, entry := range p.Inbox {
		switch age := now.Sub(entry.Date); {
		case entry.Date.IsZero():
			warnings = append(warnings, Warning{Line: entry.Line, Text: "inbox item has no date; lead it with the date and where it came from"})
		case age > inboxAge:
			warnings = append(warnings, Warning{Line: entry.Line, Text: fmt.Sprintf("inbox item from %s is %s old; triage it", entry.Date.Format(time.DateOnly), daysOld(age))})
		}
	}
	switch {
	case last >= 1 && p.Actions[last].Done && len(p.Inbox) > 0:
		warnings = append(warnings, Warning{Line: p.Actions[last].Line, Text: "the closing action is ticked while the inbox holds items; triage them first"})
	case last >= 1 && !p.Actions[last].Done && !slices.ContainsFunc(p.Actions[:last], func(action Item) bool { return !action.Done }):
		closing := p.Actions[last]
		switch {
		case len(p.Inbox) > 0:
			warnings = append(warnings, Warning{Line: closing.Line, Text: "every action but the closing one is done; ready to close once the inbox is triaged"})
		default:
			warnings = append(warnings, Warning{Line: closing.Line, Text: "every action but the closing one is done; ready to close"})
		}
	}
	slices.SortStableFunc(warnings, func(a Warning, b Warning) int { return a.Line - b.Line })
	return warnings
}

func (p Plan) Counts() Counts {
	counts := Counts{Inbox: len(p.Inbox)}
	for _, action := range p.Actions {
		switch {
		case action.Done:
			counts.Done++
		case action.Held:
			counts.Blocked++
		default:
			counts.Open++
		}
	}
	return counts
}

// Next leaves the closing action until every other action is done, held ones included.
func (p Plan) Next() (Item, bool) {
	last := len(p.Actions) - 1
	for at, action := range p.Actions {
		if action.Done || action.Held {
			continue
		}
		if at == last && slices.ContainsFunc(p.Actions[:last], func(earlier Item) bool { return !earlier.Done }) {
			break
		}
		return action, true
	}
	return Item{}, false
}

// Resolve works out Held and Flags from blockers and after: dates; a missing blocker holds its item, so a typo cannot release it.
func (p Plan) Resolve(now time.Time, lookup Lookup) Plan {
	local := map[string]Blocker{}
	for _, action := range p.Actions {
		switch {
		case action.Name == "":
		case action.Done:
			local[action.Name] = BlockerDone
		default:
			local[action.Name] = BlockerOpen
		}
	}
	state := func(edge Edge) Blocker {
		switch {
		case edge.Plan == "":
			return local[edge.Name]
		case lookup == nil:
			return BlockerMissing
		}
		return lookup(edge)
	}
	waits := map[string][]string{}
	for _, action := range p.Actions {
		for _, edge := range action.Edges {
			if !action.Done && action.Name != "" && edge.Plan == "" {
				waits[action.Name] = append(waits[action.Name], edge.Name)
			}
		}
	}
	actions := slices.Clone(p.Actions)
	for at := range actions {
		action := &actions[at]
		action.Flags, action.Held = nil, false
		if action.Done {
			for _, edge := range action.Edges {
				if state(edge) == BlockerOpen {
					action.Flags = append(action.Flags, fmt.Sprintf("ticked while blocker %s is open", edge))
				}
			}
			continue
		}
		if path := cycle(waits, action.Name); path != nil {
			action.Flags = append(action.Flags, fmt.Sprintf("blocked in a cycle, %s, which no tick can free", strings.Join(path, " → ")))
		}
		for _, reason := range action.Prose {
			head, _, _ := strings.Cut(reason, ";")
			words := strings.Fields(head)
			if len(words) > 1 && !slices.ContainsFunc(words, func(word string) bool { _, ok := local[word]; return !ok }) {
				action.Flags = append(action.Flags, fmt.Sprintf("blocked: %q reads as prose; separate its names with commas", strings.TrimSpace(head)))
			}
		}
		action.Held = len(action.Prose) > 0 || now.Before(action.After) || action.AfterInvalid != ""
		if !action.After.IsZero() && !now.Before(action.After) {
			action.Flags = append(action.Flags, fmt.Sprintf("after: %s has passed; drop it", action.After.Local().Format("2006-01-02 15:04 -07:00")))
		}
		for _, edge := range action.Edges {
			switch state(edge) {
			case BlockerMissing:
				action.Held = true
				action.Flags = append(action.Flags, fmt.Sprintf("blocker %s does not exist", edge))
			case BlockerNoPlan:
				action.Held = true
				action.Flags = append(action.Flags, fmt.Sprintf("blocker %s names no started or closed plan of this project; start that plan, or write a wait on another repository's plan as prose, with no comma before its semicolon", edge))
			case BlockerAbandoned:
				action.Held = true
				action.Flags = append(action.Flags, fmt.Sprintf("blocker %s is in a closed plan and was never done", edge))
			case BlockerDone:
				action.Flags = append(action.Flags, fmt.Sprintf("blocker %s is done; drop it from blocked:", edge))
			case BlockerOpen:
				action.Held = true
			}
		}
	}
	p.Actions = actions
	return p
}

func cited(line string) []Reference {
	var references []Reference
	for _, id := range noteID.FindAllString(line, -1) {
		references = append(references, Reference{ID: id, Kind: ReferenceNote})
	}
	for _, id := range sessionID.FindAllString(line, -1) {
		references = append(references, Reference{ID: id, Kind: ReferenceSession})
	}
	for _, match := range eleShow.FindAllStringSubmatch(line, -1) {
		if match[1] != "" {
			references = append(references, Reference{ID: match[1], Kind: ReferenceNote})
		} else {
			references = append(references, Reference{ID: match[2], Kind: ReferenceSession})
		}
	}
	return references
}

// The oldest date on the line, as the one most in need of re-measuring.
func (p *Plan) claim(line string, number int) {
	var oldest time.Time
	for _, match := range asOf.FindAllStringSubmatch(line, -1) {
		if date, err := time.ParseInLocation(time.DateOnly, match[1], time.Local); err == nil && (oldest.IsZero() || date.Before(oldest)) {
			oldest = date
		}
	}
	if !oldest.IsZero() {
		p.Claims = append(p.Claims, Dated{Date: oldest, Line: number})
	}
}

// A tab reaches the next multiple of four, as CommonMark has it.
func column(line string) int {
	width := 0
	for _, character := range line {
		switch character {
		case ' ':
			width++
		case '\t':
			width += 4 - width%4
		default:
			return width
		}
	}
	return width
}

// No item name holds a colon, so a line with one, such as a bare after:, is not a wrapped name.
func continues(line string) bool {
	trimmed := strings.TrimSpace(line)
	if _, doneWhen := cutFold(trimmed, "done when"); doneWhen || strings.Contains(trimmed, ":") {
		return false
	}
	return trimmed != "" && line != trimmed && !strings.HasPrefix(trimmed, "- ") && !otherList.MatchString(trimmed) && stepFence("", trimmed, 0) == ""
}

func cutFold(line string, prefix string) (string, bool) {
	if len(line) < len(prefix) || !strings.EqualFold(line[:len(prefix)], prefix) {
		return line, false
	}
	return line[len(prefix):], true
}

// The names from start back to itself, or nil when start is in no cycle.
func cycle(waits map[string][]string, start string) []string {
	seen := map[string]bool{}
	var walk func(name string, path []string) []string
	walk = func(name string, path []string) []string {
		for _, next := range waits[name] {
			if next == start {
				return append(path, next)
			}
			if !seen[next] {
				seen[next] = true
				if found := walk(next, append(path, next)); found != nil {
					return found
				}
			}
		}
		return nil
	}
	return walk(start, []string{start})
}

func daysOld(age time.Duration) string {
	if count := int(age / (24 * time.Hour)); count != 1 {
		return fmt.Sprintf("%d days", count)
	}
	return "1 day"
}

// A single bare word is an edge, not prose: held and flagged missing if no item has that name.
func edges(reason string) []Edge {
	head, _, _ := strings.Cut(reason, ";")
	var found []Edge
	for entry := range strings.SplitSeq(head, ",") {
		match := edgeName.FindStringSubmatch(strings.TrimSpace(entry))
		if match == nil {
			return nil
		}
		found = append(found, Edge{Name: match[2], Plan: match[1]})
	}
	return found
}

// A date alone releases at local midnight; a time is UTC.
func parseAfter(value string) (time.Time, error) {
	if after, err := time.ParseInLocation(time.DateOnly, value, time.Local); err == nil {
		return after, nil
	}
	return time.Parse("2006-01-02 15:04Z", value)
}

// A fence closes only on its own character, at least as long, with nothing after; a backtick in a backtick opener's info string makes it inline code.
// Four columns of indent past base, its list item's content, make neither opener nor closer, as CommonMark has it.
func stepFence(open string, line string, base int) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || (trimmed[0] != '`' && trimmed[0] != '~') {
		return open
	}
	if column(line)-base >= 4 {
		return open
	}
	rest := strings.TrimLeft(trimmed, trimmed[:1])
	marker := trimmed[:len(trimmed)-len(rest)]
	switch {
	case len(marker) < 3:
		return open
	case open == "":
		if marker[0] == '`' && strings.Contains(rest, "`") {
			return ""
		}
		return marker
	case marker[0] == open[0] && len(marker) >= len(open) && strings.TrimSpace(rest) == "":
		return ""
	}
	return open
}

// An unclosed backtick run hides the rest of the line: a wrapped code span closes on a later one.
func withoutCode(line string) string {
	var kept strings.Builder
	for line != "" {
		start := strings.IndexByte(line, '`')
		if start < 0 {
			kept.WriteString(line)
			break
		}
		kept.WriteString(line[:start])
		run := line[start : start+len(line[start:])-len(strings.TrimLeft(line[start:], "`"))]
		rest := line[start+len(run):]
		end := strings.Index(rest, run)
		if end < 0 {
			break
		}
		line = rest[end+len(run):]
	}
	return kept.String()
}
