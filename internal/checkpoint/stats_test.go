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

func TestWriteStatsCountsVerdictsRereviewsOlderFindingsAndMutations(t *testing.T) {
	findings := []string{
		statsFinding(``, "bugs", ``, "refuted"),
		statsFinding(``, "bugs", ``, "refuted"),
		statsFinding(``, "bugs", ``, "open"),
		strings.NewReplacer(`"category":"x"`, `"category":"z"`, `"origin":"change"`, `"origin":"earlier"`).Replace(statsFinding(``, "lint", ``, "refuted")),
		strings.NewReplacer(`"origin":"change"`, `"origin":"fix"`, `"stage":"review"`, `"stage":"rereview"`).Replace(statsFinding(``, "prose", ``, "fixed")),
		strings.NewReplacer(`"category":"x"`, `"category":"y"`, `"origin":"change"`, `"origin":"earlier"`, `"stage":"review"`, `"stage":"rereview"`).Replace(statsFinding(``, "prose", ``, "deliberate")),
	}
	loggedRun := strings.NewReplacer(`"failed":0,"inconclusive":0,"run":0`, `"failed":1,"inconclusive":1,"run":3`).Replace(statsRecord(``, "["+strings.Join(findings, ",")+"]", "full"))
	skipped := strings.Replace(statsRecord(``, "[]", "small"), `"rereview":"ran"`, `"rereview":"skipped"`, 1)
	skipped = strings.Replace(skipped, `"notes":{"approved":0,"proposed":0},`, ``, 1)

	output := writeStats(t, writeStatsLog(t, loggedRun, skipped), 30)

	require.Contains(t, output, "\nrereview   prose          full        1        0     0           1         0%\nreview     bugs           full        0        2     1           0        66%\n")
	require.Contains(t, output, "\nRe-reviews: ran in 1 of 2 checkpoints (skipped when nothing changed after the first review)\n  found something confirmed: 1 of 1\n  broken by a fix: 1\n  missed by the first review: 1\n")
	require.Contains(t, output, "\nOlder than the change (origin earlier), by category\n  y                                1\n")
	require.Contains(t, output, "\nMutations: 3 run, 1 failed as meant, 1 did not compile or did not land (33%)\nDuration")
	require.NotContains(t, output, "Notes:")
}

func TestWriteStatsCreditsSeverityAndSharedFindingsAndTimesRuns(t *testing.T) {
	logPath := writeStatsLog(t,
		statsRecord(`"scope":"branch",`, "["+statsFinding(``, "bugs", ``, "fixed")+"]", "full"),
		statsRecord(`"duration":600,`, "["+statsFinding(`"also_raised_by":["prose"],`, "bugs", `"severity":"defect",`, "fixed")+","+statsFinding(``, "prose", `"severity":"nit",`, "refuted")+","+statsFinding(``, "conventions", `"severity":"blocker",`, "deliberate")+"]", "full"),
		statsRecord(`"duration":120,"scope":"delta",`, "[]", "small"),
		statsRecord(`"duration":0,"scope":"delta",`, "[]", "small"),
		statsRecord(`"duration":-30,"scope":"delta",`, "[]", "full"),
	)

	output := writeStats(t, logPath, 30)

	for _, line := range []string{
		"bugs                  0       1        0    0        1       1",
		"conventions           1       0        0    0        0       0",
		"prose                 0       1        0    0        0       1",
		"Scope: uncommitted 1, delta 3, branch 1.",
		"Duration, the freeze that opened the run to log line: median 6.0m over 2 of 5 checkpoints (full 10.0m, small 2.0m); longest 10.0m",
		"Duration of 0 seconds or less, left out of the median: 2 checkpoints; 1 carry none",
	} {
		require.Contains(t, strings.Split(output, "\n"), line, output)
	}
}

func TestWriteStatsLeavesTheMedianOutWhenNoDurationIsAboveZero(t *testing.T) {
	logPath := writeStatsLog(t,
		statsRecord(`"duration":0,`, "[]", "small"),
		statsRecord(``, "[]", "full"),
	)

	output := writeStats(t, logPath, 30)

	require.Contains(t, output, "\nDuration of 0 seconds or less, left out of the median: 1 checkpoints; 1 carry none\n")
	require.NotContains(t, output, "median 0")
	require.NotContains(t, output, "no checkpoint carries one")
}

func TestWriteStatsNamesTheLogWhenTheWindowIsEmpty(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "checkpoints.jsonl")

	output := writeStats(t, logPath, 30)

	require.Equal(t, "Since 2026-09-20: 0 checkpoints logged. The log has no readable line.\n\nNothing logged in the window. The log is "+logPath+".\n", output)
}

func TestWriteStatsReportsReviewsApartWithTheSkipRateAndDroppedCounts(t *testing.T) {
	first := []string{
		statsReviewFinding("bugs", "confirmed", "published"),
		statsReviewFinding("bugs", "confirmed", "skipped"),
		statsReviewFinding("bugs", "open", "raw"),
		strings.Replace(statsReviewFinding("spec", "refuted", ""), `"outcome":"",`, ``, 1),
	}
	second := []string{
		statsReviewFinding("bugs", "confirmed", "discarded"),
		statsReviewFinding("bugs", "confirmed", "unanswered"),
		statsReviewFinding("conventions", "confirmed", "skipped"),
	}
	logPath := writeStatsLog(t,
		statsRecord(``, "["+statsFinding(``, "bugs", ``, "fixed")+"]", "full"),
		statsReview("mr", `{"excluded":3,"minor":5,"raised":1}`, "["+strings.Join(first, ",")+"]"),
		statsReview("local-branch", `{"excluded":2,"minor":1,"raised":0}`, "["+strings.Join(second, ",")+"]"),
	)

	output := writeStats(t, logPath, 30)

	row := func(source string, counts ...any) string {
		return fmt.Sprintf("%-14s %9d %5d %8d %9d %4d %9d %7d %10d %9s", append([]any{source}, counts...)...)
	}
	for _, line := range []string{
		"Since 2026-09-20: 1 checkpoint and 2 reviews logged. The log starts 2026-10-20.",
		"Scope: uncommitted 1, delta 0, branch 0.",
		"Reviews: mr 1, local-branch 1. Median change 10 lines added plus removed.",
		"Dropped before verification: 5 excluded, 6 minor, 1 already raised",
		row("bugs", 4, 1, 0, 1, 1, 1, 1, 1, "25%"),
		row("conventions", 1, 0, 0, 0, 0, 0, 1, 0, "100%"),
		row("spec", 0, 0, 1, 0, 0, 0, 0, 0, "-"),
		row("total", 5, 1, 1, 1, 1, 1, 2, 1, "40%"),
	} {
		require.Contains(t, strings.Split(output, "\n"), line, output)
	}
	_, checkpoints, _ := strings.Cut(output, "Who found what")
	checkpoints, _, _ = strings.Cut(checkpoints, "Reviews:")
	require.NotContains(t, checkpoints, "spec", "review findings stay out of the checkpoint tables")
}

func TestWriteStatsCountsReviewAnchorsByForgeKindAndAnswer(t *testing.T) {
	anchored := func(outcome string, anchor string) string {
		return strings.Replace(statsReviewFinding("bugs", "confirmed", outcome), `{`, `{"anchor":`+anchor+`,`, 1)
	}
	gitlab := []string{
		anchored("published", `{"answer":"kept","kind":"range"}`),
		anchored("published", `{"answer":"moved","kind":"range"}`),
		anchored("discarded", `{"answer":"refused","kind":"range"}`),
		anchored("published", `{"kind":"none"}`),
		statsReviewFinding("bugs", "confirmed", "raw"),
	}
	github := []string{anchored("published", `{"answer":"kept","kind":"line"}`)}
	forge := func(name string, findings []string) string {
		return strings.Replace(statsReview("mr", `{"excluded":0,"minor":0,"raised":0}`, "["+strings.Join(findings, ",")+"]"), `"mode"`, `"forge":"`+name+`","mode"`, 1)
	}
	logPath := writeStatsLog(t, forge("gitlab", gitlab), forge("github", github), statsReview("mr", `{"excluded":0,"minor":0,"raised":0}`, "["+statsReviewFinding("bugs", "confirmed", "published")+"]"))

	output := writeStats(t, logPath, 30)

	row := func(cells ...any) string { return fmt.Sprintf("%-8s %-6s %5d %5d %6d %8d", cells...) }
	for _, line := range []string{
		"Anchors sent, by forge and kind, with the forge's answer",
		fmt.Sprintf("%-8s %-6s %5s %5s %6s %8s", "forge", "kind", "sent", "kept", "moved", "refused"),
		row("github", "line", 1, 1, 0, 0),
		row("gitlab", "none", 1, 0, 0, 0),
		row("gitlab", "range", 3, 1, 1, 1),
	} {
		require.Contains(t, strings.Split(output, "\n"), line, output)
	}
	_, anchors, _ := strings.Cut(output, "Anchors sent")
	require.Equal(t, 5, strings.Count(anchors, "\n"), "the title, a header and three rows: unsent findings and lines logged before anchors stay out\n"+output)
}

func TestWriteStatsCountsAReplacedReviewOnceAtItsLatestOutcomes(t *testing.T) {
	const moved = "1123456789abcdef0123456789abcdef01234567"
	review := func(source string, repository string, target string, reviewed string, outcome string, replaces bool) string {
		line := statsReview("mr", `{"excluded":0,"minor":0,"raised":0}`, "["+statsReviewFinding(source, "confirmed", outcome)+"]")
		line = strings.Replace(line, `"target":"!1"`, `"target":"`+target+`"`, 1)
		line = strings.Replace(line, `"reviewed":"0123456789abcdef0123456789abcdef01234567"`, `"reviewed":"`+reviewed+`"`, 1)
		fields := `"repository":"` + repository + `",`
		if replaces {
			fields += `"replaces":true,`
		}
		return strings.Replace(line, `"reviewed"`, fields+`"reviewed"`, 1)
	}
	head := "0123456789abcdef0123456789abcdef01234567"
	logPath := writeStatsLog(t,
		review("replaced", "/a", "!1", head, "raw", false),
		review("other-head", "/a", "!1", moved, "raw", false),
		review("other-target", "/a", "!2", head, "raw", false),
		review("other-repository", "/b", "!1", head, "raw", false),
		review("replacement", "/a", "!1", head, "skipped", true),
	)

	output := writeStats(t, logPath, 30)

	require.Contains(t, output, "Since 2026-09-20: 0 checkpoints and 4 reviews logged.", output)
	lines := strings.Split(output, "\n")
	row := func(source string, raw int, skipped int, rate string) string {
		return fmt.Sprintf("%-14s %9d %5d %8d %9d %4d %9d %7d %10d %9s", source, 1, 0, 0, 0, raw, 0, skipped, 0, rate)
	}
	for _, source := range []string{"other-head", "other-target", "other-repository"} {
		require.Contains(t, lines, row(source, 1, 0, "0%"), "another head, target or repository stays\n"+output)
	}
	require.Contains(t, lines, row("replacement", 0, 1, "100%"), output)
	require.NotContains(t, output, "replaced ", "only the line of the same repository, target and head drops")
}

func TestWriteStatsSaysWhenNoRunIsTimed(t *testing.T) {
	output := writeStats(t, writeStatsLog(t, statsRecord(``, "[]", "full")), 30)

	require.Contains(t, output, "\nDuration: no checkpoint carries one\n")
}

func TestWriteStatsLeavesOutRunsOverATemporaryRepository(t *testing.T) {
	temporary := t.TempDir()
	resolved, err := filepath.EvalSymlinks(temporary)
	require.NoError(t, err)
	linked := filepath.Join(t.TempDir(), "linked")
	require.NoError(t, os.Symlink(resolved, linked))
	t.Setenv("TMPDIR", linked)
	inRepository := func(repository string) string {
		return statsRecord(`"repository":"`+repository+`",`, "[]", "full")
	}
	old := strings.Replace(inRepository("/tmp/fixture"), statsNow.Format(time.RFC3339), "2026-09-01T00:00:00Z", 1)
	logPath := writeStatsLog(t,
		old,
		inRepository("/private/tmp/claude-501/project/session/scratchpad/fixture"),
		inRepository("/tmp/fixture"),
		inRepository(filepath.Join(resolved, "fixture")),
		inRepository("/tmpfixture"),
		inRepository("/Users/someone/repository"),
	)

	output := writeStats(t, logPath, 30)

	require.True(t, strings.HasPrefix(output, "Since 2026-09-20: 2 checkpoints logged. The log starts 2026-09-01.\n\n3 runs over a repository under a temp directory left out.\n\nMode: full 2, small 0."), output)
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
		strings.Replace(statsReview("mr", `{"excluded":1}`, "[]"), `"dropped":{"excluded":1},`, ``, 1),
	)

	output := writeStats(t, logPath, 30)

	require.True(t, strings.HasPrefix(output, "Since 2026-09-20: 1 checkpoint logged. The log starts 2026-09-01.\n\n6 log lines unreadable or missing a field the report reads; skipped.\n\nMode: full 1, small 0."), output)
}

func statsFinding(extra string, source string, severity string, verdict string) string {
	return fmt.Sprintf(`{%s"category":"x","origin":"change",%s"source":"%s","stage":"review","summary":"s","verdict":"%s"}`, extra, severity, source, verdict)
}

func statsRecord(extra string, findings string, mode string) string {
	return fmt.Sprintf(`{"change":{"files":1,"lines":10},"date":"%s",%s"findings":%s,"mode":"%s","mutations":{"failed":0,"inconclusive":0,"run":0},"notes":{"approved":0,"proposed":0},"rereview":"ran"}`, statsNow.Format(time.RFC3339), extra, findings, mode)
}

func statsReview(scope string, dropped string, findings string) string {
	return fmt.Sprintf(`{"change":{"files":1,"lines":10},"date":"%s","dropped":%s,"findings":%s,"mode":"full","notes":{"approved":0,"proposed":0},"reviewed":"0123456789abcdef0123456789abcdef01234567","scope":"%s","target":"!1"}`, statsNow.Format(time.RFC3339), dropped, findings, scope)
}

func statsReviewFinding(source string, verdict string, outcome string) string {
	return fmt.Sprintf(`{"category":"x","origin":"change","outcome":"%s","severity":"defect","source":"%s","stage":"review","summary":"s","verdict":"%s"}`, outcome, source, verdict)
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
