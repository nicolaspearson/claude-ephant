package ticket

import (
	"bytes"
	"context"
	"encoding/json"
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

	runGit(t, root, "remote", "set-url", "origin", "git@gitlab.com:x/y.git")
	stdout, stderr = writeClaims(t, root, nil, false)
	require.Equal(t, "No GitHub remote here, so there are no tickets to check.\n", stdout+stderr)

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

// A gh on PATH that prints the tickets, or fails with $GH_FAIL on stderr.
func trackedRepository(t *testing.T, tickets []Ticket) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "remote", "add", "origin", "git@github.com:x/y.git")
	listed, err := json.Marshal(tickets)
	require.NoError(t, err)
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "tickets.json"), listed, 0o600))
	script := "#!/bin/sh\n[ -n \"$GH_FAIL\" ] && { echo \"$GH_FAIL\" >&2; exit 1; }\ncat '" + filepath.Join(bin, "tickets.json") + "'\n"
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
