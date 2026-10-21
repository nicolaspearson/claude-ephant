package plan

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
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
	heading  = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
)

type Counts struct {
	Blocked int
	Done    int
	Inbox   int
	Open    int
}

type Item struct {
	Blocked []string
	Body    []string
	Done    bool
	Line    int
	Title   string
}

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
				plan.Actions = append(plan.Actions, Item{Done: match[1] != " ", Line: number, Title: strings.TrimSpace(match[2])})
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
	for at := range plan.Actions {
		action := &plan.Actions[at]
		for len(action.Body) > 0 && strings.TrimSpace(action.Body[len(action.Body)-1]) == "" {
			action.Body = action.Body[:len(action.Body)-1]
		}
		bodyFence := ""
		for _, line := range action.Body {
			inside := bodyFence != ""
			if bodyFence = stepFence(bodyFence, line); inside || bodyFence != "" {
				continue
			}
			if reason, ok := strings.CutPrefix(strings.TrimPrefix(strings.TrimSpace(line), "- "), "blocked:"); ok {
				action.Blocked = append(action.Blocked, strings.TrimSpace(reason))
			}
		}
	}
	return plan, nil
}

func (p Plan) Counts() Counts {
	counts := Counts{Inbox: p.Inbox}
	for _, action := range p.Actions {
		switch {
		case action.Done:
			counts.Done++
		case len(action.Blocked) > 0:
			counts.Blocked++
		default:
			counts.Open++
		}
	}
	return counts
}

func (p Plan) Next() (Item, bool) {
	for _, action := range p.Actions {
		if !action.Done && len(action.Blocked) == 0 {
			return action, true
		}
	}
	return Item{}, false
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
