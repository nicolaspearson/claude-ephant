package mining

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func judgment(era string, session string, timestamp string, result string) Judgment {
	return Judgment{Classifier: "c", Era: era, Result: json.RawMessage(result), Session: session, Timestamp: timestamp}
}

func TestReportLeavesOutACycleInsideAJudgedRunsWindow(t *testing.T) {
	runs := []Judgment{judgment("subagent", "s1", "2026-10-08T10:00:00Z", `{"report_ts":"2026-10-08T10:30:00Z"}`)}
	cycles := []Judgment{
		judgment("", "s1", "2026-10-08T10:30:00Z", `{"started_by":"inside"}`),
		judgment("", "s1", "2026-10-08T10:40:00Z", `{"started_by":"after"}`),
		judgment("", "s2", "2026-10-08T10:10:00Z", `{"started_by":"elsewhere"}`),
	}
	var report bytes.Buffer

	require.NoError(t, Report(&report, runs, cycles))

	require.Contains(t, report.String(), "run judgments 1, review-cycle judgments 2, cycles inside a judged run's window left out 1, checkpoints 3.")
	require.Contains(t, report.String(), "Review cycles started by after 1, elsewhere 1.")
}

func TestReportLeavesOutAnInlineRunJudgedWithoutPerPassCounts(t *testing.T) {
	runs := []Judgment{
		judgment("inline", "s1", "2026-09-10T10:00:00Z", `{"findings":{"quality":{"fixed":2,"refuted_or_left":1}},"credited_earlier_passes":true}`),
		judgment("inline", "s1", "2026-09-11T10:00:00Z", `{"findings":[]}`),
		judgment("inline", "s1", "2026-09-12T10:00:00Z", `{"is_checkpoint":false,"findings":{},"after":{"user_said_checkpoint_unwanted":true}}`),
	}
	var report bytes.Buffer

	require.NoError(t, Report(&report, runs, nil))

	require.Contains(t, report.String(), "Inline era: run judgments 2, checkpoints 1.")
	require.Contains(t, report.String(), "Inline-era runs judged under the subagent-era prompt, with no per-pass counts, left out: 1.")
	require.Contains(t, report.String(), "Credited passes run earlier in the session: 1. Fixed nothing: 0.")
	require.Contains(t, report.String(), "quality             2                1")
	require.Contains(t, report.String(), "Said a checkpoint was unwanted: units 1.", "a unit judged no checkpoint still counts")
}

func TestReportReadsTheLastStoredJudgmentPerUnit(t *testing.T) {
	runs := []Judgment{
		judgment("subagent", "s1", "2026-10-08T10:00:00Z", `{"mode":"full","findings":[{"source":"bugs","stage":"review","verdict":"refuted","category":"naming"}]}`),
		judgment("subagent", "s1", "2026-10-08T10:00:00Z", `{"mode":"small","findings":[]}`),
	}
	var report bytes.Buffer

	require.NoError(t, Report(&report, runs, nil))

	require.Contains(t, report.String(), "Subagent era: run judgments 1, review-cycle judgments 0")
	require.Contains(t, report.String(), "Mode: full 0, small 1.")
	require.Contains(t, report.String(), "Refuted: 0 of 0 findings")
}
