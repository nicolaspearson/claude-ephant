package ticket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	closingText = "Each is a judgement, not a defect. A claim made in a ticket's text is\ninvisible to these label checks and still needs reading.\n"
	mapFinding  = "execution-map    grants itself execution, which later sessions read back as licence"
	taskSuffix  = " — a task unblocks a decision, it does not deliver one"
)

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
	root := trackedRepository(t, []Ticket{
		labelled("touch internal/x.go", "wayfinder:task"),
		{Body: "we carry execution", Labels: []Label{{Name: "wayfinder:map"}}, Number: 3, Title: "Map"},
		{Body: "touch internal/y.go", Labels: []Label{{Name: "bug"}}, Number: 9, Title: "Bug"},
	})

	stdout, stderr := writeClaims(t, root, nil, false)

	require.Empty(t, stderr)
	require.Equal(t, "2 open tickets are flagged.\n\n  #3 Map\n    "+mapFinding+"\n\n  #7 Task\n    typed-as-task    names internal/x.go"+taskSuffix+"\n\n"+closingText, stdout)

	stdout, _ = writeClaims(t, root, []int{7, 9}, true)
	require.True(t, strings.HasPrefix(stdout, "1 open tickets are flagged.\n\n  #7 Task\n"), stdout)
}

func TestWriteClaimsNamesEachAskedNumberItCouldNotCheck(t *testing.T) {
	root := trackedRepository(t, []Ticket{
		{Body: "clean", Number: 9, Title: "Bug"},
		{Body: "touch internal/x.go", Labels: []Label{{Name: "wayfinder:task"}}, Number: 12, State: "CLOSED", Title: "Done"},
		{Body: "clean", Number: 14, State: "MERGED", Title: "A pull request"},
		{Body: "touch internal/x.go", Labels: []Label{{Name: "wayfinder:task"}}, Number: 7, Title: "Task"},
	})

	stdout, stderr := writeClaims(t, root, []int{9, 12, 99, 14, 12}, false)

	require.Empty(t, stdout, "no all-clear past a number left unchecked")
	require.Equal(t, "Warning: #12 is closed, so its claims were not checked.\n"+
		"Warning: #99 was not checked: GraphQL: Could not resolve to an issue or pull request with the number of 99.\n"+
		"Warning: #14 is merged, so its claims were not checked.\n", stderr, "each number once")

	stdout, _ = writeClaims(t, root, []int{7, 7}, true)
	require.True(t, strings.HasPrefix(stdout, "1 open tickets are flagged.\n\n  #7 Task\n"), stdout)

	stdout, stderr = writeClaims(t, root, []int{9}, false)
	require.Equal(t, "No open ticket is mistyped or grants itself execution.\n", stdout+stderr)
}

func TestWriteClaimsWarnsWhenTheListReachesItsCap(t *testing.T) {
	tickets := make([]Ticket, ticketListLimit)
	for index := range tickets {
		tickets[index] = Ticket{Body: "clean", Number: index + 1, Title: "Bug"}
	}
	root := trackedRepository(t, tickets)

	stdout, stderr := writeClaims(t, root, nil, true)

	require.Empty(t, stdout)
	require.Equal(t, "Warning: gh listed 500 open tickets, its cap, so any beyond were not checked; pass their numbers to check them.\n", stderr)
}

func TestWriteClaimsSaysWhyNothingWasChecked(t *testing.T) {
	root := trackedRepository(t, nil)

	stdout, stderr := writeClaims(t, root, nil, false)
	require.Equal(t, "No open ticket is mistyped or grants itself execution.\n", stdout+stderr)
	stdout, stderr = writeClaims(t, root, nil, true)
	require.Empty(t, stdout+stderr)

	t.Setenv("GH_FAIL", "HTTP 401: Bad credentials")
	stdout, stderr = writeClaims(t, root, nil, true)
	require.Empty(t, stdout)
	require.Equal(t, "Warning: the ticket claim check did not run: HTTP 401: Bad credentials\n", stderr)

	for _, remote := range []string{"git@gitlab.com:x/y.git", "https://gitlab.com/mirrors/github.com-y.git"} {
		runGit(t, root, "remote", "set-url", "origin", remote)
		stdout, stderr = writeClaims(t, root, nil, false)
		require.Equal(t, "No GitHub remote here, so there are no tickets to check.\n", stdout+stderr, remote)
	}
	t.Setenv("GH_FAIL", "")
	runGit(t, root, "remote", "set-url", "origin", "git@github.com-work:x/y.git")
	stdout, stderr = writeClaims(t, root, nil, false)
	require.Equal(t, "No open ticket is mistyped or grants itself execution.\n", stdout+stderr, "an ssh alias of github.com")

	stdout, stderr = writeClaims(t, t.TempDir(), nil, true)
	require.Empty(t, stdout)
	require.Equal(t, "Warning: the ticket claim check needs a git repository.\n", stderr)
}

func labelled(body string, labels ...string) Ticket {
	ticket := Ticket{Body: body, Number: 7, Title: "Task"}
	for _, name := range labels {
		ticket.Labels = append(ticket.Labels, Label{Name: name})
	}
	return ticket
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
	require.NoError(t, err, string(output))
}

// A gh on PATH that lists the open tickets and views any by number, or fails
// with $GH_FAIL on stderr. A ticket with no state is open.
func trackedRepository(t *testing.T, tickets []Ticket) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "remote", "add", "origin", "git@github.com:x/y.git")
	bin := t.TempDir()
	var open []Ticket
	for _, ticket := range tickets {
		if ticket.State == "" {
			ticket.State = "OPEN"
		}
		if ticket.State == "OPEN" {
			open = append(open, ticket)
		}
		viewed, err := json.Marshal(ticket)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(bin, fmt.Sprintf("%d.json", ticket.Number)), viewed, 0o600))
	}
	listed, err := json.Marshal(open)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(bin, "tickets.json"), listed, 0o600))
	script := "#!/bin/sh\n[ -n \"$GH_FAIL\" ] && { echo \"$GH_FAIL\" >&2; exit 1; }\n" +
		"if [ \"$2\" = view ]; then\n" +
		"  [ -f '" + bin + "'/\"$3\".json ] || { echo \"GraphQL: Could not resolve to an issue or pull request with the number of $3.\" >&2; exit 1; }\n" +
		"  exec cat '" + bin + "'/\"$3\".json\n" +
		"fi\n" +
		"cat '" + filepath.Join(bin, "tickets.json") + "'\n"
	require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return root
}

func writeClaims(t *testing.T, directory string, numbers []int, quiet bool) (string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	WriteClaims(context.Background(), &stdout, &stderr, directory, numbers, quiet)
	return stdout.String(), stderr.String()
}
