package ticket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	closingText = "Each is a judgement, not a defect. A claim made in a ticket's text is\ninvisible to these label checks and still needs reading.\n"
	mapFinding  = "execution-map    grants itself execution, which later sessions read back as licence"
	taskSuffix  = " — a task unblocks a decision, it does not deliver one"
)

type claimsTracker struct {
	failure string
	missing string
	outside bool
	remote  string
	tickets []Ticket
}

func TestFindingsFlagsAMapGrantingItselfExecutionInAnyCase(t *testing.T) {
	for body, want := range map[string][]string{
		"Execution, Not Only Decisions": {mapFinding},
		"no grant here":                 nil,
		"this map Carries Execution.":   {mapFinding},
		"we carry execution":            {mapFinding},
	} {
		require.Equal(t, want, Findings(labelled(body, "wayfinder:map")), body)
	}
	require.Empty(t, Findings(labelled("we carry execution", "wayfinder:task")), "only a map can grant itself execution")
}

func TestFindingsNamesEachSourcePathOnceSortedQuotedOrBare(t *testing.T) {
	ticket := labelled("Touch `internal/foo/bar.go`, internal/foo/bar.go and backend/api/x.go; also internal/a- and proto/fooBarBaz", "wayfinder:map", "wayfinder:task")

	require.Equal(t, []string{"typed-as-task    names backend/api/x.go, internal/a, internal/foo/bar.go, proto/fooBarBaz" + taskSuffix}, Findings(ticket))
}

func TestTokensKeepsOnlyCodeShapedOnes(t *testing.T) {
	for text, want := range map[string][]string{
		"`a.b` `12/34` `abc` `abcd`":                 nil,
		"`internal/" + strings.Repeat("a", 71) + "`": {"internal/" + strings.Repeat("a", 71)},
		"`internal/" + strings.Repeat("a", 72) + "`": nil,
		"`internal/x y` `code\nspan`":                nil,
		"`two` TikTok StarRocks CamelCaseThing":      {"CamelCaseThing"},
		"snake_case_name and file.go":                {"snake_case_name", "file.go"},
	} {
		require.Equal(t, want, tokens(text), text)
	}
}

func TestTokensTreatsNonASCIILettersAsWordCharacters(t *testing.T) {
	require.Equal(t, []string{"backend/é_thing.go", "naïve_backend/x.go", "y.go", "internal/日本/x.go"}, tokens("naïve_backend/x.go ébackend/y.go `backend/é_thing.go` —internal/日本/x.go—"))
}

func TestWriteClaimsListsFlaggedTicketsByNumber(t *testing.T) {
	tracker := newClaimsTracker(
		labelled("touch internal/x.go", "wayfinder:task"),
		Ticket{Body: "we carry execution", Labels: []Label{{Name: "wayfinder:map"}}, Number: 3, Title: "Map"},
		Ticket{Body: "touch internal/y.go", Labels: []Label{{Name: "bug"}}, Number: 9, Title: "Bug"},
	)

	stdout, stderr := writeClaims(t, tracker, nil, false)

	require.Empty(t, stderr)
	require.Equal(t, "2 open tickets are flagged.\n\n  #3 Map\n    "+mapFinding+"\n\n  #7 Task\n    typed-as-task    names internal/x.go"+taskSuffix+"\n\n"+closingText, stdout)

	stdout, _ = writeClaims(t, tracker, []int{7, 9}, true)
	require.True(t, strings.HasPrefix(stdout, "1 open ticket is flagged.\n\n  #7 Task\n"), stdout)
}

func TestWriteClaimsNamesEachAskedNumberItCouldNotCheck(t *testing.T) {
	tracker := newClaimsTracker(
		Ticket{Body: "clean", Number: 9, Title: "Bug"},
		Ticket{Body: "touch internal/x.go", Labels: []Label{{Name: "wayfinder:task"}}, Number: 12, State: "CLOSED", Title: "Done"},
		Ticket{Body: "clean", Number: 14, State: "MERGED", Title: "A pull request"},
		Ticket{Body: "touch internal/x.go", Labels: []Label{{Name: "wayfinder:task"}}, Number: 7, Title: "Task"},
	)

	stdout, stderr := writeClaims(t, tracker, []int{9, 12, 99, 14, 12}, false)

	require.Empty(t, stdout, "no all-clear past a number left unchecked")
	require.Equal(t, "Warning: #12 is closed, so its claims were not checked.\n"+
		"Warning: #99 was not checked: gh issue view 99 --json number,title,body,labels,state failed: GraphQL: Could not resolve to an issue or pull request with the number of 99.\n"+
		"Warning: #14 is merged, so its claims were not checked.\n", stderr, "each number once")

	stdout, _ = writeClaims(t, tracker, []int{7, 7}, true)
	require.True(t, strings.HasPrefix(stdout, "1 open ticket is flagged.\n\n  #7 Task\n"), stdout)

	stdout, stderr = writeClaims(t, tracker, []int{9}, false)
	require.Equal(t, "No open ticket is mistyped or grants itself execution.\n", stdout+stderr)
}

func TestWriteClaimsSaysWhyNothingWasChecked(t *testing.T) {
	tracker := newClaimsTracker()

	stdout, stderr := writeClaims(t, tracker, nil, false)
	require.Equal(t, "No open ticket is mistyped or grants itself execution.\n", stdout+stderr)
	stdout, stderr = writeClaims(t, tracker, nil, true)
	require.Empty(t, stdout+stderr)

	tracker.failure = "HTTP 401: Bad credentials"
	stdout, stderr = writeClaims(t, tracker, nil, true)
	require.Empty(t, stdout)
	require.Equal(t, "Warning: the ticket claim check did not run: gh issue list --state open --limit 500 --json number,title,body,labels failed: HTTP 401: Bad credentials\n", stderr)

	for _, remote := range []string{"git@gitlab.com:x/y.git", "https://gitlab.com/mirrors/github.com-y.git"} {
		tracker.remote = remote
		stdout, stderr = writeClaims(t, tracker, nil, false)
		require.Equal(t, "No GitHub remote here, so there are no tickets to check.\n", stdout+stderr, remote)
	}
	tracker.failure = ""
	tracker.remote = "git@github.com-work:x/y.git"
	stdout, stderr = writeClaims(t, tracker, nil, false)
	require.Equal(t, "No open ticket is mistyped or grants itself execution.\n", stdout+stderr, "an ssh alias of github.com")

	tracker.missing = "gh"
	stdout, stderr = writeClaims(t, tracker, nil, false)
	require.Equal(t, "No GitHub remote here, so there are no tickets to check.\n", stdout+stderr, "no gh to ask")

	tracker.outside = true
	stdout, stderr = writeClaims(t, tracker, nil, true)
	require.Empty(t, stdout)
	require.Equal(t, "Warning: the ticket claim check needs a git repository.\n", stderr)
}

func TestWriteClaimsWarnsWhenTheListReachesItsCap(t *testing.T) {
	tickets := make([]Ticket, ticketListLimit)
	for index := range tickets {
		tickets[index] = Ticket{Body: "clean", Number: index + 1, Title: "Bug"}
	}

	stdout, stderr := writeClaims(t, newClaimsTracker(tickets...), nil, true)

	require.Empty(t, stdout)
	require.Equal(t, "Warning: gh listed 500 open tickets, its cap, so any beyond were not checked; pass their numbers to check them.\n", stderr)
}

func (tracker *claimsTracker) commands() Commands {
	return Commands{
		LookPath: func(file string) (string, error) {
			if file == tracker.missing {
				return "", exec.ErrNotFound
			}
			return "/bin/" + file, nil
		},
		Run: tracker.run,
	}
}

func (tracker *claimsTracker) run(_ context.Context, name string, args ...string) (string, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	switch {
	case call == "git rev-parse --show-toplevel" && tracker.outside:
		return "", errors.New("fatal: not a git repository")
	case call == "git rev-parse --show-toplevel":
		return "/repository\n", nil
	case call == "git remote -v":
		return "origin\t" + tracker.remote + " (fetch)\norigin\t" + tracker.remote + " (push)\n", nil
	case name == "gh" && tracker.failure != "":
		return "", fmt.Errorf("%s failed: %s", call, tracker.failure)
	case name == "gh" && len(args) > 2 && args[0] == "issue" && args[1] == "view":
		for _, ticket := range tracker.tickets {
			if strconv.Itoa(ticket.Number) == args[2] {
				return encode(ticket), nil
			}
		}
		return "", fmt.Errorf("%s failed: GraphQL: Could not resolve to an issue or pull request with the number of %s.", call, args[2])
	case name == "gh" && len(args) > 1 && args[0] == "issue" && args[1] == "list":
		open := []Ticket{}
		for _, ticket := range tracker.tickets {
			if ticket.State == "OPEN" {
				open = append(open, ticket)
			}
		}
		return encode(open), nil
	}
	return "", fmt.Errorf("unexpected call: %s", call)
}

func labelled(body string, labels ...string) Ticket {
	ticket := Ticket{Body: body, Number: 7, Title: "Task"}
	for _, name := range labels {
		ticket.Labels = append(ticket.Labels, Label{Name: name})
	}
	return ticket
}

// A ticket with no state is open.
func newClaimsTracker(tickets ...Ticket) *claimsTracker {
	tracker := &claimsTracker{remote: "git@github.com:x/y.git"}
	for _, ticket := range tickets {
		if ticket.State == "" {
			ticket.State = "OPEN"
		}
		tracker.tickets = append(tracker.tickets, ticket)
	}
	return tracker
}

func writeClaims(t *testing.T, tracker *claimsTracker, numbers []int, quiet bool) (string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	WriteClaims(context.Background(), &stdout, &stderr, tracker.commands(), numbers, quiet)
	return stdout.String(), stderr.String()
}
