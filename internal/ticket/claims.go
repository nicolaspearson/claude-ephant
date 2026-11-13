package ticket

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const (
	maximumTokenLength = 80
	minimumTokenLength = 4
	ticketListLimit    = 500
)

var (
	// Tickets backtick inconsistently, so bare prose is scanned too.
	bareToken       = regexp.MustCompile(`\b(?:[A-Za-z_][\w-]*(?:[./][\w-]+)+|[a-z]+(?:_[a-z]+)+|[A-Z][a-z]+(?:[A-Z][a-z]+)+)\b`)
	camelCase       = regexp.MustCompile(`[a-z][A-Z]`)
	executionGrants = []string{"carries execution", "carry execution", "execution, not only decisions"}
	quotedSpan      = regexp.MustCompile("`([^`\n]+)`")
	sourceRoots     = []string{"backend/", "frontend/", "internal/", "manifests/", "packages/", "proto/"}
	tokenShape      = regexp.MustCompile(`^[\w./-]+$`)
)

type Label struct {
	Name string `json:"name"`
}

type Ticket struct {
	Body   string  `json:"body"`
	Labels []Label `json:"labels"`
	Number int     `json:"number"`
	State  string  `json:"state"`
	Title  string  `json:"title"`
}

func Findings(ticket Ticket) []string {
	var found []string
	if hasLabel(ticket, "wayfinder:map") {
		lowered := strings.ToLower(ticket.Body)
		if slices.ContainsFunc(executionGrants, func(grant string) bool { return strings.Contains(lowered, grant) }) {
			found = append(found, "execution-map    grants itself execution, which later sessions read back as licence")
		}
	}
	if hasLabel(ticket, "wayfinder:task") {
		var named []string
		for _, token := range tokens(ticket.Body) {
			if slices.ContainsFunc(sourceRoots, func(root string) bool { return strings.HasPrefix(token, root) }) {
				named = append(named, token)
			}
		}
		slices.Sort(named)
		if named = slices.Compact(named); len(named) > 0 {
			found = append(found, fmt.Sprintf("typed-as-task    names %s — a task unblocks a decision, it does not deliver one", strings.Join(named, ", ")))
		}
	}
	return found
}

// WriteClaims never fails; it warns on stderr when the check cannot run.
func WriteClaims(ctx context.Context, stdout io.Writer, stderr io.Writer, commands Commands, numbers []int, quiet bool) {
	if _, err := commands.Run(ctx, "git", "rev-parse", "--show-toplevel"); err != nil {
		fmt.Fprintln(stderr, "Warning: the ticket claim check needs a git repository.")
		return
	}
	// Nothing to check against without a tracker, and asking costs a network call.
	if !hasTracker(ctx, commands) {
		if !quiet {
			fmt.Fprintln(stdout, "No GitHub remote here, so there are no tickets to check.")
		}
		return
	}
	tickets, complete, err := openTickets(ctx, stderr, commands, numbers)
	if err != nil {
		fmt.Fprintf(stderr, "Warning: the ticket claim check did not run: %v\n", err)
		return
	}
	var flagged []Ticket
	for _, ticket := range tickets {
		if len(Findings(ticket)) > 0 {
			flagged = append(flagged, ticket)
		}
	}
	if len(flagged) == 0 {
		if !quiet && complete {
			fmt.Fprintln(stdout, "No open ticket is mistyped or grants itself execution.")
		}
		return
	}
	slices.SortStableFunc(flagged, func(left Ticket, right Ticket) int { return left.Number - right.Number })
	if len(flagged) == 1 {
		fmt.Fprint(stdout, "1 open ticket is flagged.\n\n")
	} else {
		fmt.Fprintf(stdout, "%d open tickets are flagged.\n\n", len(flagged))
	}
	for _, ticket := range flagged {
		fmt.Fprintf(stdout, "  #%d %s\n", ticket.Number, ticket.Title)
		for _, line := range Findings(ticket) {
			fmt.Fprintf(stdout, "    %s\n", line)
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintln(stdout, "Each is a judgement, not a defect. A claim made in a ticket's text is")
	fmt.Fprintln(stdout, "invisible to these label checks and still needs reading.")
}

// Go's \w and \b are ASCII-only. Non-ASCII letters and numbers become '0' (a
// word character), other non-ASCII runes '\x01' (not one). One byte per rune,
// so offsets index the rune slice.
func asciiShadow(runes []rune) string {
	shadow := make([]byte, len(runes))
	for index, character := range runes {
		switch {
		case character <= unicode.MaxASCII:
			shadow[index] = byte(character)
		case unicode.IsLetter(character) || unicode.IsNumber(character):
			shadow[index] = '0'
		default:
			shadow[index] = '\x01'
		}
	}
	return string(shadow)
}

func codeShaped(shadow string) bool {
	if len(shadow) < minimumTokenLength || len(shadow) > maximumTokenLength {
		return false
	}
	if !tokenShape.MatchString(shadow) || !strings.ContainsFunc(shadow, unicode.IsLetter) {
		return false
	}
	// Two humps is a product name — TikTok, StarRocks. Three is a declaration.
	return strings.ContainsAny(shadow, "._/-") || len(camelCase.FindAllString(shadow, -1)) >= 2
}

func hasLabel(ticket Ticket, name string) bool {
	return slices.ContainsFunc(ticket.Labels, func(label Label) bool { return label.Name == name })
}

func hasTracker(ctx context.Context, commands Commands) bool {
	if _, err := commands.LookPath("gh"); err != nil {
		return false
	}
	remotes, _ := commands.Run(ctx, "git", "remote", "-v")
	for line := range strings.Lines(remotes) {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if _, host := readRemote(fields[1]); strings.Contains(host, "github") {
			return true
		}
	}
	return false
}

func openTickets(ctx context.Context, stderr io.Writer, commands Commands, numbers []int) ([]Ticket, bool, error) {
	if len(numbers) == 0 {
		listed, err := commands.Run(ctx, "gh", "issue", "list", "--state", "open", "--limit", strconv.Itoa(ticketListLimit), "--json", "number,title,body,labels")
		if err != nil {
			return nil, false, err
		}
		var tickets []Ticket
		if err := json.Unmarshal([]byte(listed), &tickets); err != nil {
			return nil, false, fmt.Errorf("gh printed no ticket list: %w", err)
		}
		if len(tickets) >= ticketListLimit {
			fmt.Fprintf(stderr, "Warning: gh listed %d open tickets, its cap, so any beyond were not checked; pass their numbers to check them.\n", ticketListLimit)
			return tickets, false, nil
		}
		return tickets, true, nil
	}
	var tickets []Ticket
	complete := true
	for index, number := range numbers {
		if slices.Contains(numbers[:index], number) {
			continue
		}
		viewed, err := commands.Run(ctx, "gh", "issue", "view", strconv.Itoa(number), "--json", "number,title,body,labels,state")
		var ticket Ticket
		if err == nil {
			if err = json.Unmarshal([]byte(viewed), &ticket); err != nil {
				err = fmt.Errorf("gh printed no ticket: %w", err)
			}
		}
		switch {
		case err != nil:
			fmt.Fprintf(stderr, "Warning: #%d was not checked: %v\n", number, err)
			complete = false
		case ticket.State != "OPEN":
			fmt.Fprintf(stderr, "Warning: #%d is %s, so its claims were not checked.\n", number, cmp.Or(strings.ToLower(ticket.State), "not open"))
			complete = false
		default:
			tickets = append(tickets, ticket)
		}
	}
	return tickets, complete, nil
}

func tokens(text string) []string {
	runes := []rune(text)
	shadow := asciiShadow(runes)
	var found []string
	for _, span := range quotedSpan.FindAllStringSubmatchIndex(shadow, -1) {
		if codeShaped(shadow[span[2]:span[3]]) {
			found = append(found, string(runes[span[2]:span[3]]))
		}
	}
	stripped := slices.Clone(runes)
	for _, span := range slices.Backward(quotedSpan.FindAllStringIndex(shadow, -1)) {
		stripped = slices.Replace(stripped, span[0], span[1], ' ')
	}
	strippedShadow := asciiShadow(stripped)
	for _, span := range bareToken.FindAllStringIndex(strippedShadow, -1) {
		if codeShaped(strippedShadow[span[0]:span[1]]) {
			found = append(found, string(stripped[span[0]:span[1]]))
		}
	}
	return found
}
