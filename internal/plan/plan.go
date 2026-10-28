package plan

import (
	"bufio"
	"bytes"
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
	eleShow   = regexp.MustCompile(`\bele show(?: --(?:around|at|context) (?:"[^"]*"|'[^']*'|\S+))* (?:--note (\d{8}-[0-9a-f]{6}|[0-9a-f]{8,})|([0-9a-f]{8,}))\b`)
	heading   = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
	inboxDate = regexp.MustCompile(`^- (\d{4}-\d{2}-\d{2})\b`)
	itemName  = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*): `)
	noteID    = regexp.MustCompile(`\b\d{8}-[0-9a-f]{6}\b`)
	sessionID = regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
)

type Blocker int

const (
	// First, so a name absent from a lookup map reads as missing.
	BlockerMissing Blocker = iota
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
	After    time.Time
	Blocked  []string
	Body     []string
	Done     bool
	DoneWhen bool
	Edges    []Edge
	Flags    []string
	Held     bool
	Line     int
	Name     string
	Prose    []string
	Title    string
}

type Lookup func(edge Edge) Blocker

type Plan struct {
	Actions   []Item
	Citations []Citation
	Claims    []Dated
	Created   string
	Inbox     []Dated
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
	number := 1
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
			return invalid("unknown front matter key %q", key)
		}
		if seen[key] {
			return invalid("front matter key %q appears twice", key)
		}
		seen[key] = true
		*field = strings.TrimSpace(value)
	}
	switch {
	case !closed:
		return invalid("front matter never closes")
	case kind != "plan":
		return invalid("kind %q is not plan", kind)
	}
	if _, err := time.Parse(time.DateOnly, plan.Created); plan.Created != "" && err != nil {
		return invalid("created %q is not YYYY-MM-DD", plan.Created)
	}

	section, fence, actions := "", "", false
	var item *Item
	for scanner.Scan() {
		number++
		line := scanner.Text()
		if references := cited(line); len(references) > 0 {
			plan.Citations = append(plan.Citations, Citation{Line: number, References: references})
		}
		inside := fence != ""
		fence = stepFence(fence, line)
		fenceLine := inside || fence != ""
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
		bodyFence := ""
		for offset, line := range action.Body {
			inside := bodyFence != ""
			if bodyFence = stepFence(bodyFence, line); inside || bodyFence != "" {
				continue
			}
			trimmed := strings.TrimPrefix(strings.TrimSpace(line), "- ")
			action.DoneWhen = action.DoneWhen || strings.HasPrefix(strings.ToLower(trimmed), "done when")
			if reason, ok := strings.CutPrefix(trimmed, "blocked:"); ok {
				reason = strings.TrimSpace(reason)
				action.Blocked = append(action.Blocked, reason)
				if found := edges(reason); found != nil {
					action.Edges = append(action.Edges, found...)
				} else {
					action.Prose = append(action.Prose, reason)
				}
			}
			if value, ok := strings.CutPrefix(trimmed, "after:"); ok {
				number := action.Line + offset + 1
				if !action.After.IsZero() {
					return invalid("a second after: on line %d", number)
				}
				after, err := parseAfter(strings.TrimSpace(value))
				if err != nil {
					return invalid("after %q on line %d is neither YYYY-MM-DD nor YYYY-MM-DD HH:MMZ", strings.TrimSpace(value), number)
				}
				action.After = after
			}
		}
		// Unresolved until Resolve: every edge and date still holds the item back.
		action.Held = !action.Done && (len(action.Blocked) > 0 || !action.After.IsZero())
	}
	return plan, nil
}

func (e Edge) String() string {
	if e.Plan == "" {
		return e.Name
	}
	return e.Plan + "/" + e.Name
}

func Name(title string) string {
	if match := itemName.FindStringSubmatch(title); match != nil {
		return match[1]
	}
	return ""
}

// Check runs on a resolved plan: it reports the plan's flags beside its own warnings.
func (p Plan) Check(now time.Time) []Warning {
	var warnings []Warning
	last := len(p.Actions) - 1
	for at, action := range p.Actions {
		label := action.Name
		if label == "" {
			label = "action"
		}
		for _, flag := range action.Flags {
			warnings = append(warnings, Warning{Line: action.Line, Text: label + ": " + flag})
		}
		if !action.Done && at != last && !action.DoneWhen {
			warnings = append(warnings, Warning{Line: action.Line, Text: label + ": no done when line"})
		}
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

func (p Plan) Next() (Item, bool) {
	for _, action := range p.Actions {
		if !action.Done && !action.Held {
			return action, true
		}
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
	actions := slices.Clone(p.Actions)
	for at := range actions {
		action := &actions[at]
		action.Flags, action.Held = nil, false
		if action.Done {
			continue
		}
		action.Held = len(action.Prose) > 0 || now.Before(action.After)
		if !action.After.IsZero() && !now.Before(action.After) {
			action.Flags = append(action.Flags, fmt.Sprintf("after: %s has passed; drop it", action.After.Local().Format("2006-01-02 15:04 -07:00")))
		}
		for _, edge := range action.Edges {
			state := local[edge.Name]
			if edge.Plan != "" {
				state = lookup(edge)
			}
			switch state {
			case BlockerMissing:
				action.Held = true
				action.Flags = append(action.Flags, fmt.Sprintf("blocker %s does not exist", edge))
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
func stepFence(open string, line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || (trimmed[0] != '`' && trimmed[0] != '~') {
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
