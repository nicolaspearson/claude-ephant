package checkpoint

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const valid = `{"change":{"files":3,"lines":40},"findings":[{"category":"stale-comment","origin":"fix","severity":"quality","source":"prose","stage":"rereview","summary":"Comment names the old flag.","verdict":"fixed"}],"mode":"full","mutations":{"failed":4,"inconclusive":1,"run":5},"notes":{"approved":0,"proposed":1},"rereview":"ran","scope":"delta"}`

func TestAppendAddsDateHeadRepositoryAndKeepsTheRecord(t *testing.T) {
	root, logPath := repository(t)
	record, problems, err := Problems(strings.NewReader(valid))
	require.NoError(t, err)
	require.Empty(t, problems)

	require.NoError(t, Append(context.Background(), root, record, logPath, time.Date(2026, 10, 20, 9, 5, 7, 0, time.FixedZone("x", 3600))))
	require.NoError(t, Append(context.Background(), root, record, logPath, time.Now()))

	lines := logLines(t, logPath)
	require.Len(t, lines, 2, "appended, not overwritten")
	head, err := git(context.Background(), root, "rev-parse", "HEAD")
	require.NoError(t, err)
	require.Equal(t, "2026-10-20T08:05:07Z", lines[0]["date"])
	require.Equal(t, strings.TrimSpace(head), lines[0]["head"])
	require.Equal(t, root, lines[0]["repository"])
	require.Equal(t, "rereview", lines[0]["findings"].([]any)[0].(map[string]any)["stage"])
	require.NotContains(t, lines[0], "duration", "no duration without a freeze time")
}

func TestAppendKeepsTheFreezeTimeWhenTheLineIsNotWritten(t *testing.T) {
	root, _ := repository(t)
	marker, err := ReviewedFile(context.Background(), root)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(marker+".started", []byte("1\n"), 0o600))

	require.Error(t, Append(context.Background(), root, map[string]any{}, t.TempDir(), time.Now()))

	require.FileExists(t, marker+".started")
}

func TestAppendKeysTheLineSortedAndCompact(t *testing.T) {
	root, logPath := repository(t)
	record, _, err := Problems(strings.NewReader(`{"z":1.0,"a":{"y":"<&>","b":1e2}}`))
	require.NoError(t, err)

	require.NoError(t, Append(context.Background(), root, record, logPath, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)))

	written, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Regexp(t, `^\{"a":\{"b":1e2,"y":"<&>"\},"date":"2026-10-20T00:00:00Z","head":"[0-9a-f]{40}","repository":"[^"]+","z":1.0\}\n$`, string(written))
}

func TestAppendLeavesHeadEmptyBeforeTheFirstCommit(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	root, err := Root(context.Background(), root)
	require.NoError(t, err)
	t.Setenv("CHECKPOINT_REVIEWED_DIR", filepath.Join(t.TempDir(), "reviewed"))
	logPath := filepath.Join(t.TempDir(), "checkpoints.jsonl")

	require.NoError(t, Append(context.Background(), root, map[string]any{}, logPath, time.Now()))

	require.Equal(t, "", logLines(t, logPath)[0]["head"])
}

func TestAppendTimesTheRunFromTheFirstFreezeAndEndsIt(t *testing.T) {
	root, logPath := repository(t)
	t.Setenv(sessionVariable, "this-session")
	run, err := RunFile(context.Background(), root, "this-session")
	require.NoError(t, err)
	other, err := RunFile(context.Background(), root, "another-session")
	require.NoError(t, err)
	now := time.Now()
	for _, started := range []string{"", "0x10\n", "12\n34\n"} {
		require.NoError(t, os.WriteFile(run+".started", []byte(started), 0o600))
		require.NoError(t, Append(context.Background(), root, map[string]any{}, logPath, now))
	}
	require.NoError(t, os.WriteFile(other+".started", []byte(strconv.FormatInt(now.Unix()-5, 10)+"\n"), 0o600))
	require.NoError(t, os.WriteFile(run+".started", []byte(strconv.FormatInt(now.Unix()-90, 10)+"\n"), 0o600))
	require.NoError(t, Append(context.Background(), root, map[string]any{}, logPath, now))

	lines := logLines(t, logPath)
	for _, line := range lines[:3] {
		require.NotContains(t, line, "duration", "an unreadable freeze time is logged without a duration")
	}
	require.Equal(t, json.Number("90"), lines[3]["duration"], "another session's run on the same repository keeps its own clock")
	require.NoFileExists(t, run+".started", "freeze time removed once logged")
	require.FileExists(t, other+".started")
}

func TestProblemsAcceptsAValidRecord(t *testing.T) {
	for name, record := range map[string]string{
		"a finding raised by two reviewers": with(t, func(record map[string]any) { finding(record)["also_raised_by"] = []any{"bugs"} }),
		"empty findings":                    with(t, func(record map[string]any) { record["findings"] = []any{} }),
		"valid":                             valid,
	} {
		t.Run(name, func(t *testing.T) {
			_, problems, err := Problems(strings.NewReader(record))
			require.NoError(t, err)
			require.Empty(t, problems)
		})
	}
}

func TestProblemsRefusesARecordTheStatsWouldMisread(t *testing.T) {
	for _, test := range []struct {
		name    string
		record  string
		problem string
	}{
		{"also_raised_by empty", with(t, func(record map[string]any) { finding(record)["also_raised_by"] = []any{} }), "findings[0].also_raised_by is not a list of sources"},
		{"also_raised_by names a source twice", with(t, func(record map[string]any) { finding(record)["also_raised_by"] = []any{"bugs", "bugs"} }), "findings[0].also_raised_by names a source twice"},
		{"also_raised_by not a list", with(t, func(record map[string]any) { finding(record)["also_raised_by"] = "bugs" }), "findings[0].also_raised_by is not a list of sources"},
		{"also_raised_by repeats the source, first", with(t, func(record map[string]any) { finding(record)["also_raised_by"] = []any{"prose", "bugs"} }), `findings[0].also_raised_by repeats its source "prose"`},
		{"also_raised_by with an empty source", with(t, func(record map[string]any) { finding(record)["also_raised_by"] = []any{""} }), "findings[0].also_raised_by is not a list of sources"},
		{"category not kebab-case", with(t, func(record map[string]any) { finding(record)["category"] = "Stale Comment" }), `findings[0].category "Stale Comment" is not kebab-case`},
		{"category with a trailing newline", with(t, func(record map[string]any) { finding(record)["category"] = "stale-comment\n" }), `findings[0].category "stale-comment\n" is not kebab-case`},
		{"change missing", with(t, func(record map[string]any) { delete(record, "change") }), "change.files is not a whole number"},
		{"duration supplied", with(t, func(record map[string]any) { record["duration"] = 60 }), "duration is derived from the first freeze; leave it out"},
		{"empty stdin", "", "no record on stdin"},
		{"finding not an object", with(t, func(record map[string]any) { record["findings"] = []any{"x"} }), "findings[0] is not an object"},
		{"finding without source", with(t, func(record map[string]any) { finding(record)["source"] = "" }), "findings[0] has no source"},
		{"finding without summary", with(t, func(record map[string]any) { delete(finding(record), "summary") }), "findings[0] has no summary"},
		{"findings not a list", with(t, func(record map[string]any) { record["findings"] = "x" }), "findings is not a list"},
		{"fractional count", with(t, func(record map[string]any) { record["change"].(map[string]any)["lines"] = 1.5 }), "change.lines is not a whole number"},
		{"more mutations failed than run", with(t, func(record map[string]any) { record["mutations"].(map[string]any)["failed"] = 9 }), "mutations.failed plus mutations.inconclusive exceeds mutations.run"},
		{"mutations not an object", with(t, func(record map[string]any) { record["mutations"] = "x" }), "mutations.failed is not a whole number"},
		{"mutations one over run", with(t, func(record map[string]any) { record["mutations"].(map[string]any)["failed"] = 5 }), "mutations.failed plus mutations.inconclusive exceeds mutations.run"},
		{"negative count", with(t, func(record map[string]any) { record["mutations"].(map[string]any)["run"] = -1 }), "mutations.run is not a whole number"},
		{"not an object", "[]", "the record is not an object"},
		{"notes missing", with(t, func(record map[string]any) { delete(record, "notes") }), "notes.approved is not a whole number"},
		{"rereview missing", with(t, func(record map[string]any) { delete(record, "rereview") }), "rereview null is not one of ran, skipped"},
		{"scope missing", with(t, func(record map[string]any) { delete(record, "scope") }), "scope null is not one of branch, delta, uncommitted"},
		{"severity missing", with(t, func(record map[string]any) { delete(finding(record), "severity") }), "findings[0].severity null is not one of blocker, defect, nit, quality"},
		{"two records", valid + " " + valid, "2 records on stdin; send one per checkpoint"},
		{"unknown mode", with(t, func(record map[string]any) { record["mode"] = "medium" }), `mode "medium" is not one of full, small`},
		{"unknown origin", with(t, func(record map[string]any) { finding(record)["origin"] = "unknown" }), `findings[0].origin "unknown" is not one of change, earlier, fix`},
		{"unknown scope", with(t, func(record map[string]any) { record["scope"] = "staged" }), `scope "staged" is not one of branch, delta, uncommitted`},
		{"unknown severity", with(t, func(record map[string]any) { finding(record)["severity"] = "major" }), `findings[0].severity "major" is not one of blocker, defect, nit, quality`},
		{"unknown stage", with(t, func(record map[string]any) { finding(record)["stage"] = "lint" }), `findings[0].stage "lint" is not one of gate, mutation, rereview, review, user`},
		{"unknown verdict", with(t, func(record map[string]any) { finding(record)["verdict"] = "wontfix" }), `findings[0].verdict "wontfix" is not one of deliberate, fixed, open, refuted`},
		{"whitespace stdin", "  ", "no record on stdin"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, problems, err := Problems(strings.NewReader(test.record))
			require.NoError(t, err)
			require.Contains(t, problems, test.problem)
		})
	}
}

func TestProblemsReportsInputThatIsNotJSON(t *testing.T) {
	_, _, err := Problems(strings.NewReader(`{"mode":`))
	require.Error(t, err)
}

func TestReviewedFileKeysByTheRepositoryRootAsRecordsWrittenBeforeThePort(t *testing.T) {
	root, _ := repository(t)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sub"), 0o700))

	fromRoot, err := ReviewedFile(context.Background(), root)
	require.NoError(t, err)
	fromSubdirectory, err := ReviewedFile(context.Background(), filepath.Join(root, "sub"))
	require.NoError(t, err)

	require.Equal(t, fromRoot, fromSubdirectory)
	shasum, err := exec.Command("sh", "-c", `printf '%s' "$1" | shasum | cut -d' ' -f1`, "sh", root).Output()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(os.Getenv("CHECKPOINT_REVIEWED_DIR"), strings.TrimSpace(string(shasum))), fromRoot)
}

func TestRootKeepsATrailingSpaceInThePath(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "repository ")
	require.NoError(t, os.Mkdir(directory, 0o700))
	runGit(t, directory, "init", "-q")

	root, err := Root(context.Background(), directory)

	require.NoError(t, err)
	require.True(t, strings.HasSuffix(root, "repository "), root)
}

func finding(record map[string]any) map[string]any {
	return record["findings"].([]any)[0].(map[string]any)
}

func logLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	written, err := os.ReadFile(path)
	require.NoError(t, err)
	var lines []map[string]any
	for line := range strings.SplitSeq(strings.TrimSuffix(string(written), "\n"), "\n") {
		decoder := json.NewDecoder(strings.NewReader(line))
		decoder.UseNumber()
		var decoded map[string]any
		require.NoError(t, decoder.Decode(&decoded))
		lines = append(lines, decoded)
	}
	return lines
}

func repository(t *testing.T) (string, string) {
	t.Helper()
	directory := t.TempDir()
	runGit(t, directory, "init", "-q")
	runGit(t, directory, "commit", "-q", "--allow-empty", "-m", "first")
	root, err := Root(context.Background(), directory)
	require.NoError(t, err)
	t.Setenv("CHECKPOINT_REVIEWED_DIR", filepath.Join(t.TempDir(), "reviewed"))
	return root, filepath.Join(t.TempDir(), "logs", "checkpoints.jsonl")
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func with(t *testing.T, change func(record map[string]any)) string {
	t.Helper()
	var record map[string]any
	require.NoError(t, json.Unmarshal([]byte(valid), &record))
	change(record)
	encoded, err := json.Marshal(record)
	require.NoError(t, err)
	return string(encoded)
}
