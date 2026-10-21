package ticket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

const (
	maximumTokenLength = 80
	minimumTokenLength = 4
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
func WriteClaims(ctx context.Context, stdout io.Writer, stderr io.Writer, directory string, numbers []int, quiet bool) {
	root, err := run(ctx, directory, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		fmt.Fprintln(stderr, "Warning: the ticket claim check needs a git repository.")
		return
	}
	root = strings.TrimSpace(root)
	// Nothing to check against without a tracker, and asking costs a network call.
	if !hasTracker(ctx, root) {
		if !quiet {
			fmt.Fprintln(stdout, "No GitHub remote here, so there are no tickets to check.")
		}
		return
	}
	tickets, err := openTickets(ctx, root, numbers)
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
		if !quiet {
			fmt.Fprintln(stdout, "No open ticket is mistyped or grants itself execution.")
		}
		return
	}
	slices.SortStableFunc(flagged, func(left Ticket, right Ticket) int { return left.Number - right.Number })
	fmt.Fprintf(stdout, "%d open tickets are flagged.\n\n", len(flagged))
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

func hasTracker(ctx context.Context, root string) bool {
	if _, err := exec.LookPath("gh"); err != nil {
		return false
	}
	remotes, _ := run(ctx, root, "git", "remote", "-v")
	return strings.Contains(remotes, "github.com")
}

func openTickets(ctx context.Context, root string, numbers []int) ([]Ticket, error) {
	listed, err := run(ctx, root, "gh", "issue", "list", "--state", "open", "--limit", "200", "--json", "number,title,body,labels")
	if err != nil {
		return nil, err
	}
	var tickets []Ticket
	if err := json.Unmarshal([]byte(listed), &tickets); err != nil {
		return nil, fmt.Errorf("gh printed no ticket list: %w", err)
	}
	if len(numbers) > 0 {
		tickets = slices.DeleteFunc(tickets, func(ticket Ticket) bool { return !slices.Contains(numbers, ticket.Number) })
	}
	return tickets, nil
}

func run(ctx context.Context, directory string, name string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, name, args...)
	command.Dir, command.Stdout, command.Stderr = directory, &stdout, &stderr
	if err := command.Run(); err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return "", fmt.Errorf("%s", message)
		}
		return "", err
	}
	return stdout.String(), nil
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
