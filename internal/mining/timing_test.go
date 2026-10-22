package mining

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

func timedRun(t *testing.T) string {
	t.Helper()
	archiveDirectory := t.TempDir()
	report := "Ready to commit.\n\n**Bugs reviewer:** no findings.\n**Prose reviewer:** one comment cut."
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"),
		transcriptLine(t, "10:00:00", "assistant", toolUse("tu0", "Skill", map[string]any{"skill": "checkpoint"})),
		transcriptLine(t, "10:01:00", "assistant", toolUse("tu1", "Agent", map[string]any{"description": "Bugs review", "prompt": "Read reviewers/bugs.md first"})),
		strings.TrimSuffix(transcriptLine(t, "10:01:01", "user", toolResult("tu1", "done", false)), "}\n")+`,"toolUseResult":{"agentId":"a1"}}`+"\n",
		transcriptLine(t, "10:02:00", "assistant", toolUse("tu3", "Bash", map[string]any{"command": "go test ./..."})),
		transcriptLine(t, "10:05:00", "user", toolResult("tu3", "ok", false)),
		transcriptLine(t, "10:05:30", "assistant", toolUse("tu4", "Bash", map[string]any{"command": "ls"})),
		transcriptLine(t, "10:05:40", "user", toolResult("tu4", "go.mod", false)),
		transcriptLine(t, "10:06:00", "assistant", []map[string]any{{"text": report, "type": "text"}}),
		transcriptLine(t, "10:20:00", "assistant", toolUse("tu5", "Bash", map[string]any{"command": "go test ./..."})),
		transcriptLine(t, "10:30:00", "user", toolResult("tu5", "ok", false)))
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1", "subagents", "agent-a1.jsonl.gz"),
		transcriptLine(t, "10:01:05", "user", "Read reviewers/bugs.md"),
		transcriptLine(t, "10:01:10", "assistant", toolUse("st1", "Read", map[string]any{"file_path": "/home/u/.claude/CLAUDE.md"})),
		transcriptLine(t, "10:01:20", "assistant", toolUse("st2", "Bash", map[string]any{"command": "go vet ./..."})),
		transcriptLine(t, "10:01:50", "user", toolResult("st2", "ok", false)),
		transcriptLine(t, "10:03:05", "assistant", []map[string]any{{"text": "No findings.", "type": "text"}}))
	return archiveDirectory
}

func TestTimeEndsAtTheTimeTheJudgmentSaysTheCheckpointReported(t *testing.T) {
	unit := dto.MiningUnit{Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z"}

	timings := Time(timedRun(t), []TimedUnit{{Analysis: "checkpoint-runs", ReportedAt: "2026-10-16T10:05:00Z", Unit: unit}})

	timing := timings[UnitKey{Analysis: "checkpoint-runs", Session: "s1", Timestamp: "2026-10-16T10:00:00Z"}]
	require.Equal(t, 300.0, timing.Seconds)
	require.Equal(t, 180.0, timing.ParentBash, "the 10:05:30 call falls after the judged report")
}

func TestTimeReadsBashAndReviewersInARunsWindow(t *testing.T) {
	unit := dto.MiningUnit{Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z"}

	missing := dto.MiningUnit{Session: "s2", Source: "claude/-gateway/gone.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z"}

	timings := Time(timedRun(t), []TimedUnit{{Analysis: "checkpoint-runs", Unit: unit}, {Analysis: "checkpoint-runs", Unit: missing}})

	reviewerSeconds := 120.0
	require.Equal(t, map[UnitKey]Timing{{Analysis: "checkpoint-runs", Session: "s1", Timestamp: "2026-10-16T10:00:00Z"}: {
		ParentBash:   190,
		ParentGate:   180,
		Reviewers:    []ReviewerTiming{{ConventionReads: 1, GateRuns: 1, Role: "bugs", Round: 1, Seconds: &reviewerSeconds, Tools: 2}},
		Seconds:      360,
		SubagentBash: 30,
		SubagentGate: 30,
	}}, timings, "the window closes at the report, so the 10:20 gate is not this run's, and the unit with no transcript is left untimed")
}

func TestTimeStartsACycleAtItsFirstLaunch(t *testing.T) {
	archiveDirectory := t.TempDir()
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t2.jsonl.gz"),
		transcriptLine(t, "09:55:00", "user", "fix the retry bug"),
		transcriptLine(t, "09:56:00", "assistant", toolUse("tu1", "Bash", map[string]any{"command": "go test ./..."})),
		transcriptLine(t, "09:58:00", "user", toolResult("tu1", "ok", false)),
		transcriptLine(t, "10:00:00", "assistant", toolUse("tu2", "Agent", map[string]any{"description": "Re-review fixes", "prompt": "Read reviewers/bugs.md and reviewers/prose.md"})),
		strings.TrimSuffix(transcriptLine(t, "10:00:01", "user", toolResult("tu2", "done", false)), "}\n")+`,"toolUseResult":{"agentId":"a2"}}`+"\n",
		transcriptLine(t, "10:02:00", "assistant", toolUse("tu3", "Bash", map[string]any{"command": "ls"})),
		transcriptLine(t, "10:03:00", "user", toolResult("tu3", "go.mod", false)))
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t2", "subagents", "agent-a2.jsonl.gz"),
		transcriptLine(t, "10:00:05", "user", "Read reviewers/bugs.md"),
		transcriptLine(t, "10:01:05", "assistant", []map[string]any{{"text": "No findings.", "type": "text"}}))
	unit := dto.MiningUnit{Session: "s1", Source: "claude/-gateway/t2.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z"}

	timings := Time(archiveDirectory, []TimedUnit{{Analysis: "review-cycles", Unit: unit}})

	reviewerSeconds := 60.0
	require.Equal(t, map[UnitKey]Timing{{Analysis: "review-cycles", Session: "s1", Timestamp: "2026-10-16T10:00:00Z"}: {
		ParentBash: 60,
		Reviewers:  []ReviewerTiming{{Role: "combined", Round: 2, Seconds: &reviewerSeconds}},
		Seconds:    180,
	}}, timings, "the 09:56 gate is the cycle's lead-in, not its time")
}

func TestUpperQuartileInterpolatesByTheExclusiveMethod(t *testing.T) {
	for _, test := range []struct {
		values []float64
		want   float64
	}{
		{values: []float64{5, 1, 4, 2, 3}, want: 4.5},
		{values: []float64{10, 20}, want: 22.5},
	} {
		quartile, ok := upperQuartile(test.values)
		require.True(t, ok, test.values)
		require.Equal(t, test.want, quartile, test.values)
	}
	_, ok := upperQuartile([]float64{1})
	require.False(t, ok)
}
