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
	sectionActions = "Actions"
	sectionInbox   = "Inbox"
)

var (
	ErrInvalid = errors.New("invalid plan")

	checkbox = regexp.MustCompile(`^- \[([ xX])\] (.*)$`)
	edgeName = regexp.MustCompile(`^(?:([a-z0-9]+(?:-[a-z0-9]+)*)/)?([a-z0-9]+(?:-[a-z0-9]+)*)$`)
	heading  = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
	itemName = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*): `)
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

type Counts struct {
	Blocked int
	Done    int
	Inbox   int
	Open    int
}

type Edge struct {
	Name string
	Plan string
}

type Item struct {
	After   time.Time
	Blocked []string
	Body    []string
	Done    bool
	Edges   []Edge
	Flags   []string
	Held    bool
	Line    int
	Name    string
	Prose   []string
	Title   string
}

type Lookup func(edge Edge) Blocker

type Plan struct {
	Actions []Item
	Created string
	Inbox   int
	Spec    string
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
		inside := fence != ""
		fence = stepFence(fence, line)
		fenceLine := inside || fence != ""
		if item != nil && (strings.TrimSpace(line) == "" || fenceLine || line != strings.TrimLeft(line, " \t")) {
			item.Body = append(item.Body, line)
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
			continue
		}
		switch {
		case section == sectionInbox && strings.HasPrefix(line, "- "):
			plan.Inbox++
		case section == sectionActions:
			if match := checkbox.FindStringSubmatch(line); match != nil {
				title := strings.TrimSpace(match[2])
				plan.Actions = append(plan.Actions, Item{Done: match[1] != " ", Line: number, Name: Name(title), Title: title})
				item = &plan.Actions[len(plan.Actions)-1]
			}
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

func (p Plan) Counts() Counts {
	counts := Counts{Inbox: p.Inbox}
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
