package checkpoint

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var statsNow = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

func TestReportListsTheFifteenCommonestLateCategoriesTiesInFirstSeenOrder(t *testing.T) {
	var findings []string
	for index := range 17 {
		findings = append(findings, strings.Replace(statsFinding(``, "bugs", ``, "fixed"), `"category":"x"`, fmt.Sprintf(`"category":"c%d"`, index), 1))
	}
	findings = append(findings, strings.Replace(statsFinding(``, "bugs", ``, "fixed"), `"category":"x"`, `"category":"c16"`, 1))
	record := strings.ReplaceAll(statsRecord(``, "["+strings.Join(findings, ",")+"]", "full"), `"stage":"review"`, `"stage":"gate"`)

	output := writeStats(t, writeStatsLog(t, record), 30)

	_, late, _ := strings.Cut(output, "Caught after review (gate, mutation, user), by category\n")
	late, _, _ = strings.Cut(late, "\n\n")
	lines := strings.Split(late, "\n")
	require.Len(t, lines, 15, late)
	require.Equal(t, fmt.Sprintf("  %-9s %-32s 2", "gate", "c16"), lines[0])
	require.Equal(t, fmt.Sprintf("  %-9s %-32s 1", "gate", "c0"), lines[1])
	require.Equal(t, fmt.Sprintf("  %-9s %-32s 1", "gate", "c13"), lines[14])
}

func TestWriteBaselineRefusesAMissingFileOrADirectory(t *testing.T) {
	directory := t.TempDir()

	for _, path := range []string{filepath.Join(directory, "missing.jsonl"), directory} {
		var output bytes.Buffer
		require.ErrorIs(t, WriteBaseline(&output, path), ErrNoBaseline, path)
		require.Empty(t, output.String())
	}
}

func TestWriteBaselineReportsOnlyTheBaselineWhateverItsDates(t *testing.T) {
	baselinePath := writeStatsLog(t,
		strings.NewReplacer(`"lines":10`, `"lines":30`, statsNow.Format(time.RFC3339), "2026-01-01T00:00:00Z").Replace(statsRecord(`"confidence":"high",`, "[]", "small")),
		strings.NewReplacer(`"lines":10`, `"lines":0`, statsNow.Format(time.RFC3339), "2026-01-02T00:00:00.000Z").Replace(statsRecord(`"change_unknown":true,"confidence":"low",`, "[]", "small")),
	)
	var output bytes.Buffer

	require.NoError(t, WriteBaseline(&output, baselinePath))

	require.True(t, strings.HasPrefix(output.String(), "\n\nBaseline, whatever the window: 2 checkpoints mined from transcripts, not logged, 2026-01-01 to 2026-01-02; confidence high 1, medium 0, low 1.\n\nMode: full 0, small 2. Median change 30 lines added plus removed, 1 of unknown size left out.\n"), output.String())
}

func TestWriteStatsCountsVerdictsRereviewsOlderFindingsMutationsAndNotes(t *testing.T) {
	findings := []string{
		statsFinding(``, "bugs", ``, "refuted"),
		statsFinding(``, "bugs", ``, "refuted"),
		statsFinding(``, "bugs", ``, "open"),
		strings.NewReplacer(`"category":"x"`, `"category":"z"`, `"origin":"change"`, `"origin":"earlier"`).Replace(statsFinding(``, "lint", ``, "refuted")),
		strings.NewReplacer(`"origin":"change"`, `"origin":"fix"`, `"stage":"review"`, `"stage":"rereview"`).Replace(statsFinding(``, "prose", ``, "fixed")),
		strings.NewReplacer(`"category":"x"`, `"category":"y"`, `"origin":"change"`, `"origin":"earlier"`, `"stage":"review"`, `"stage":"rereview"`).Replace(statsFinding(``, "prose", ``, "deliberate")),
	}
	loggedRun := strings.NewReplacer(`"failed":0,"inconclusive":0,"run":0`, `"failed":1,"inconclusive":1,"run":3`, `"approved":0,"proposed":0`, `"approved":1,"proposed":2`).Replace(statsRecord(``, "["+strings.Join(findings, ",")+"]", "full"))
	skipped := strings.Replace(statsRecord(``, "[]", "small"), `"rereview":"ran"`, `"rereview":"skipped"`, 1)

	output := writeStats(t, writeStatsLog(t, loggedRun, skipped), 30)

	require.Contains(t, output, "\nrereview   prose          full        1        0     0           1         0%\nreview     bugs           full        0        2     1           0        66%\n")
	require.Contains(t, output, "\nRe-reviews: ran in 1 of 2 checkpoints (skipped when nothing changed after the first review)\n  found something confirmed: 1 of 1\n  broken by a fix: 1\n  missed by the first review: 1\n")
	require.Contains(t, output, "\nOlder than the change (origin earlier), by category\n  y                                1\n")
	require.Contains(t, output, "\nMutations: 3 run, 1 failed as meant, 1 did not compile or did not land (33%)\nNotes: 2 proposed, 1 approved\n")
}

func TestWriteStatsCreditsSeverityAndSharedFindingsAndTimesRuns(t *testing.T) {
	logPath := writeStatsLog(t,
		statsRecord(`"scope":"branch",`, "["+statsFinding(``, "bugs", ``, "fixed")+"]", "full"),
		statsRecord(`"duration":600,`, "["+statsFinding(`"also_raised_by":["prose"],`, "bugs", `"severity":"defect",`, "fixed")+","+statsFinding(``, "prose", `"severity":"nit",`, "refuted")+","+statsFinding(``, "conventions", `"severity":"blocker",`, "deliberate")+"]", "full"),
		statsRecord(`"duration":120,"scope":"delta",`, "[]", "small"),
	)

	output := writeStats(t, logPath, 30)

	for _, line := range []string{
		"bugs                  0       1        0    0        1       1",
		"conventions           1       0        0    0        0       0",
		"prose                 0       1        0    0        0       1",
		"Scope: uncommitted 1, delta 1, branch 1.",
		"Duration, first freeze to log line: median 6.0m over 2 of 3 checkpoints (full 10.0m, small 2.0m); longest 10.0m",
	} {
		require.Contains(t, strings.Split(output, "\n"), line, output)
	}
}

func TestWriteStatsNamesTheLogWhenTheWindowIsEmpty(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "checkpoints.jsonl")

	output := writeStats(t, logPath, 30)

	require.Equal(t, "Since 2026-09-20: 0 checkpoints logged. The log has no readable line.\n\nNothing logged in the window. The log is "+logPath+".\n", output)
}

func TestWriteStatsSaysWhenNoRunIsTimed(t *testing.T) {
	output := writeStats(t, writeStatsLog(t, statsRecord(``, "[]", "full")), 30)

	require.Contains(t, output, "\nDuration: no checkpoint carries one\n")
}

func TestWriteStatsSkipsUnreadableLinesAndRunsBeforeTheWindow(t *testing.T) {
	old := strings.Replace(statsRecord(``, "[]", "small"), statsNow.Format(time.RFC3339), "2026-09-01T00:00:00Z", 1)
	logPath := writeStatsLog(t,
		old,
		"",
		"not json",
		strings.Replace(statsRecord(``, "[]", "full"), `"mode":"full"`, `"mode":1`, 1),
		strings.Replace(statsRecord(``, "[]", "full"), statsNow.Format(time.RFC3339), "2026-10-20", 1),
		statsRecord(``, `[{"category":"x"}]`, "full"),
		statsRecord(``, "[]", "full"),
	)

	output := writeStats(t, logPath, 30)

	require.True(t, strings.HasPrefix(output, "Since 2026-09-20: 1 checkpoint logged. The log starts 2026-09-01.\n\n5 log lines unreadable or missing a field the report reads; skipped.\n\nMode: full 1, small 0."), output)
}

func statsFinding(extra string, source string, severity string, verdict string) string {
	return fmt.Sprintf(`{%s"category":"x","origin":"change",%s"source":"%s","stage":"review","summary":"s","verdict":"%s"}`, extra, severity, source, verdict)
}

func statsRecord(extra string, findings string, mode string) string {
	return fmt.Sprintf(`{"change":{"files":1,"lines":10},"date":"%s",%s"findings":%s,"mode":"%s","mutations":{"failed":0,"inconclusive":0,"run":0},"notes":{"approved":0,"proposed":0},"rereview":"ran"}`, statsNow.Format(time.RFC3339), extra, findings, mode)
}

func writeStats(t *testing.T, logPath string, days int) string {
	t.Helper()
	var output bytes.Buffer
	require.NoError(t, WriteStats(&output, logPath, statsNow.AddDate(0, 0, -days)))
	return output.String()
}

func writeStatsLog(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "checkpoints.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600))
	return path
}
