package mining

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

const secret = "glpat-abcdefghijklmnopqrstuvwx"

func toolResult(id string, content string, isError bool) []map[string]any {
	return []map[string]any{{"content": content, "is_error": isError, "tool_use_id": id, "type": "tool_result"}}
}

func toolUse(id string, name string, input map[string]any) []map[string]any {
	return []map[string]any{{"id": id, "input": input, "name": name, "type": "tool_use"}}
}

func transcriptLine(t *testing.T, clock string, kind string, content any) string {
	t.Helper()
	encoded, err := json.Marshal(map[string]any{"message": map[string]any{"content": content}, "sessionId": "s1", "timestamp": "2026-10-16T" + clock + "Z", "type": kind})
	require.NoError(t, err)
	return string(encoded) + "\n"
}

func writeArchived(t *testing.T, path string, lines ...string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	file, err := os.Create(path)
	require.NoError(t, err)
	writer := gzip.NewWriter(file)
	_, err = writer.Write([]byte(strings.Join(lines, "")))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
}

func TestDigestOfACheckpointRunLinksItsSubagentsAndStopsAtItsReport(t *testing.T) {
	archiveDirectory := t.TempDir()
	report := "Ready to commit.\n\n**Bugs reviewer:** no findings.\n**Prose reviewer:** one comment cut."
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"),
		transcriptLine(t, "09:00:00", "user", "add retries to the client"),
		transcriptLine(t, "09:59:00", "user", "Stop hook feedback:\nno checkpoint was reported"),
		`{"type":"user","timestamp":"2026-10-16T09:59:30Z","message":`+"\n",
		transcriptLine(t, "10:00:00", "assistant", toolUse("tu0", "Skill", map[string]any{"skill": "checkpoint"})),
		transcriptLine(t, "10:01:00", "assistant", toolUse("tu1", "Agent", map[string]any{"description": "Checkpoint bugs review", "prompt": strings.Repeat("x", 1490) + " " + secret})),
		strings.TrimSuffix(transcriptLine(t, "10:01:01", "user", toolResult("tu1", "No findings, though the report quotes agentId: zz", false)), "}\n")+`,"toolUseResult":{"agentId":"a1"}}`+"\n",
		transcriptLine(t, "10:01:02", "assistant", toolUse("tu2", "Agent", map[string]any{"description": "Checkpoint prose review " + secret, "subagent_type": "checkpoint-reviewer"})),
		transcriptLine(t, "10:01:03", "user", toolResult("tu2", "Agent type 'checkpoint-reviewer' not found", true)),
		transcriptLine(t, "10:01:30", "assistant", toolUse("tu5", "Edit", map[string]any{"file_path": "/tmp/" + secret})),
		transcriptLine(t, "10:02:00", "assistant", toolUse("tu3", "Bash", map[string]any{"command": "go test ./..."})),
		transcriptLine(t, "10:05:00", "user", toolResult("tu3", "ok\ncovered: retries", false)),
		transcriptLine(t, "10:06:00", "assistant", []map[string]any{{"text": report, "type": "text"}}),
		transcriptLine(t, "10:07:00", "user", "looks good, thanks"),
		transcriptLine(t, "10:08:00", "assistant", []map[string]any{{"text": "after the report", "type": "text"}}),
		transcriptLine(t, "15:00:00", "user", "<command-name>/checkpoint</command-name>"))
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1", "subagents", "agent-a1.jsonl.gz"),
		transcriptLine(t, "10:01:05", "user", "Read reviewers/bugs.md"),
		transcriptLine(t, "10:01:10", "assistant", toolUse("st1", "Read", map[string]any{"file_path": "client.go"})),
		transcriptLine(t, "10:03:05", "assistant", []map[string]any{{"text": "No findings.", "type": "text"}}))

	digest, err := Digest(archiveDirectory, "checkpoint-runs", dto.MiningUnit{Owner: "corp", Project: "gateway", Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z"})
	require.NoError(t, err)

	encoded, _, _ := strings.Cut(digest, "\n")
	var header map[string]any
	require.NoError(t, json.Unmarshal([]byte(encoded), &header))
	require.Equal(t, map[string]any{
		"agentDescriptions": []any{"Checkpoint bugs review", "Checkpoint prose review [redacted:gitlab]"},
		"agentSeconds":      []any{120.0, nil},
		"agents":            2.0,
		"bashCommands":      1.0,
		"covered":           true,
		"durationSeconds":   360.0,
		"edits":             1.0,
		"end":               "2026-10-16T10:06:00Z",
		"era":               "subagent",
		"gateCommands":      1.0,
		"hookBlocks":        0.0,
		"nextHuman":         "2026-10-16T10:07:00Z",
		"owner":             "corp",
		"project":           "gateway",
		"session":           "s1",
		"timestamp":         "2026-10-16T10:00:00Z",
		"triggeredByHook":   true,
	}, header)
	require.Contains(t, digest, "=== PRIOR USER ASK 2026-10-16T09:00:00Z\nadd retries to the client\n")
	require.Contains(t, digest, "--- subagent duration 120s, tools:\n  Read client.go\n--- final\nNo findings.\n")
	require.Contains(t, digest, "--- launch failed\nAgent type 'checkpoint-reviewer' not found\n")
	require.Contains(t, digest, "=== NEXT HUMAN 2026-10-16T10:07:00Z\nlooks good, thanks\n")
	require.NotContains(t, digest, "after the report", "the window ends at the report")
	require.NotContains(t, digest, "glpat-", "a secret is redacted before its text is cut, and in descriptions and paths")
}

func TestDigestOfAReviewCycleCutsAtAReportOnlyOnceEveryReviewerHasHandedBack(t *testing.T) {
	archiveDirectory := t.TempDir()
	progress := "**Bugs reviewer:** no findings. The prose reviewer is still out."
	report := "The checkpoint is recorded and logged.\n\n**Bugs reviewer:** no findings."
	launched := func(clock string, id string, agent string) string {
		line := transcriptLine(t, clock, "user", toolResult(id, "Async agent launched successfully.\nagentId: "+agent, false))
		return strings.TrimSuffix(line, "}\n") + `,"toolUseResult":{"agentId":"` + agent + `","status":"async_launched"}}` + "\n"
	}
	lines := []string{
		transcriptLine(t, "11:19:00", "user", "review it"),
		transcriptLine(t, "11:20:00", "assistant", toolUse("tu1", "Agent", map[string]any{"description": "Bugs review", "prompt": "Read reviewers/bugs.md"})),
		launched("11:20:01", "tu1", "a1"),
		transcriptLine(t, "11:20:02", "assistant", toolUse("tu2", "Agent", map[string]any{"description": "Prose review", "prompt": "Read reviewers/prose.md"})),
		launched("11:20:03", "tu2", "a2"),
		transcriptLine(t, "11:20:04", "assistant", toolUse("tu4", "Agent", map[string]any{"description": "Conventions review", "prompt": "Read reviewers/conventions.md"})),
		launched("11:20:05", "tu4", "a3"),
		transcriptLine(t, "11:30:00", "user", "Another Claude session sent a message:\n<agent-message from=\"a1\">\nNo findings."),
		`{"type":"queue-operation","operation":"enqueue","timestamp":"2026-10-16T11:31:00Z","content":"<task-notification>\n<task-id>a2</task-id>\n<status>completed</status>"}` + "\n",
		transcriptLine(t, "11:31:20", "assistant", []map[string]any{{"text": "Waiting on <task-id>a3</task-id>.", "type": "text"}}),
		transcriptLine(t, "11:31:30", "assistant", []map[string]any{{"text": progress, "type": "text"}}),
		`{"type":"attachment","timestamp":"2026-10-16T11:32:00Z","attachment":{"type":"queued_command","prompt":"<task-notification>\n<task-id>a3</task-id>\n<status>completed</status>"}}` + "\n",
	}
	after := []string{
		transcriptLine(t, "11:55:00", "user", "commit it"),
		transcriptLine(t, "11:56:00", "assistant", toolUse("tu3", "Agent", map[string]any{"description": "Explore the next slice", "prompt": "Find the handlers"})),
		transcriptLine(t, "11:57:00", "assistant", []map[string]any{{"text": "after the report", "type": "text"}}),
	}
	unit := dto.MiningUnit{Reviewers: 3, Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T11:20:00Z"}

	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"), append(lines, after...)...)
	digest, err := Digest(archiveDirectory, "review-cycles", unit)
	require.NoError(t, err)
	require.Contains(t, digest, "TEXT after the report", "a message written while reviewers still run is not the report")

	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"), append(append(lines, transcriptLine(t, "11:50:00", "assistant", []map[string]any{{"text": report, "type": "text"}})), after...)...)
	digest, err = Digest(archiveDirectory, "review-cycles", unit)
	require.NoError(t, err)
	require.Contains(t, digest, "=== NEXT HUMAN 2026-10-16T11:55:00Z\ncommit it\n")
	require.NotContains(t, digest, "after the report", "a launch naming no reviewer file does not reopen the window")
}

func TestDigestOfACheckpointRunClosesAtTheNextCycle(t *testing.T) {
	archiveDirectory := t.TempDir()
	report := "Ready to commit.\n\n**Bugs reviewer:** no findings.\n**Prose reviewer:** one comment cut."
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"),
		transcriptLine(t, "10:00:00", "assistant", toolUse("tu0", "Skill", map[string]any{"skill": "checkpoint"})),
		transcriptLine(t, "10:05:00", "assistant", []map[string]any{{"text": "The checkpoint is done.", "type": "text"}}),
		transcriptLine(t, "10:20:00", "assistant", toolUse("tu1", "Agent", map[string]any{"description": "Next checkpoint", "prompt": "Read reviewers/bugs.md"})),
		transcriptLine(t, "10:30:00", "assistant", []map[string]any{{"text": report, "type": "text"}}))

	digest, err := Digest(archiveDirectory, "checkpoint-runs", dto.MiningUnit{Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z", Until: "2026-10-16T10:20:00Z"})
	require.NoError(t, err)
	require.Contains(t, digest, `"end":"2026-10-16T10:05:00Z"`)
	require.NotContains(t, digest, "Next checkpoint")
	require.NotContains(t, digest, "Bugs reviewer", "a later checkpoint's report is not the run's")
}

func TestDigestOfACheckpointRunClosesAtTheNextTypedPluginInvocation(t *testing.T) {
	archiveDirectory := t.TempDir()
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"),
		transcriptLine(t, "09:59:00", "user", "add retries to the client"),
		transcriptLine(t, "10:00:00", "user", "<command-name>/ele:checkpoint</command-name>"),
		transcriptLine(t, "10:01:00", "user", "<command-name>/ele:checkpoint-stats</command-name>"),
		transcriptLine(t, "10:02:00", "assistant", toolUse("tu1", "Skill", map[string]any{"skill": "ele:checkpoint-stats"})),
		transcriptLine(t, "10:05:00", "assistant", []map[string]any{{"text": "first run", "type": "text"}}),
		transcriptLine(t, "11:00:00", "user", "<command-name>/ele:checkpoint</command-name>"),
		transcriptLine(t, "11:05:00", "assistant", []map[string]any{{"text": "second run", "type": "text"}}))

	first, err := Digest(archiveDirectory, "checkpoint-runs", dto.MiningUnit{Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z"})
	require.NoError(t, err)
	require.Contains(t, first, "first run", "another skill whose name starts checkpoint is not a checkpoint")
	require.NotContains(t, first, "second run")

	second, err := Digest(archiveDirectory, "checkpoint-runs", dto.MiningUnit{Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T11:00:00Z"})
	require.NoError(t, err)
	require.Contains(t, second, "add retries to the client")
	require.NotContains(t, second, "2026-10-16T10:00:00Z", "an invocation is not the user's asking")
}

func TestDigestOfACheckpointRunEndsTheTurnItRecordsIn(t *testing.T) {
	progress := "The bugs reviewer found two problems and the gate passes; fixing both before the commit. " + strings.Repeat("x", 800)
	launched := strings.TrimSuffix(transcriptLine(t, "10:01:01", "user", toolResult("tu1", "Async agent launched successfully.\nagentId: a1", false)), "}\n") + `,"toolUseResult":{"agentId":"a1","status":"async_launched"}}` + "\n"
	for _, command := range []string{
		`P=/plugin/premigration; "$P/checkpoint-reviewed.sh" record /repo`,
		"~/.claude/hooks/checkpoint-log.sh /repo <<'EOF'\n{}\nEOF",
		"R=/repo; ele checkpoint log $R <<'EOF'\n{}\nEOF",
		"ele checkpoint freeze /repo /tmp -- a.go && ele checkpoint record /repo",
	} {
		archiveDirectory := t.TempDir()
		writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"),
			transcriptLine(t, "10:00:00", "assistant", toolUse("tu0", "Skill", map[string]any{"skill": "ele:checkpoint"})),
			transcriptLine(t, "10:01:00", "assistant", toolUse("tu1", "Agent", map[string]any{"description": "Bugs review", "prompt": "Read reviewers/bugs.md"})),
			launched,
			`{"type":"queue-operation","operation":"enqueue","timestamp":"2026-10-16T10:02:20Z","content":"<task-notification>\n<task-id>a1</task-id>\n<status>completed</status>"}`+"\n",
			transcriptLine(t, "10:02:30", "assistant", toolUse("tu2", "Bash", map[string]any{"command": "find ~/.claude -name 'checkpoint-log.sh'"})),
			transcriptLine(t, "10:02:40", "user", "keep going"),
			transcriptLine(t, "10:02:55", "assistant", []map[string]any{{"text": progress, "type": "text"}}),
			transcriptLine(t, "10:03:00", "assistant", toolUse("tu3", "Bash", map[string]any{"command": command})),
			transcriptLine(t, "10:03:30", "assistant", []map[string]any{{"text": "Recorded. Logging the run.", "type": "text"}}),
			transcriptLine(t, "10:04:00", "assistant", []map[string]any{{"text": "Item 3 is built and ready to commit.", "type": "text"}}),
			transcriptLine(t, "10:05:00", "user", "thanks"),
			transcriptLine(t, "10:06:00", "assistant", []map[string]any{{"text": "after the report, the checkpoint is recorded", "type": "text"}}))

		digest, err := Digest(archiveDirectory, "checkpoint-runs", dto.MiningUnit{Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z"})
		require.NoError(t, err)
		require.Contains(t, digest, `"end":"2026-10-16T10:04:00Z"`, "%s: a script named but not run and long progress are not the report; the turn that records ends at it", command)
		require.NotContains(t, digest, "after the report", command)
	}
}

func TestDigestOfACheckpointRunOpensAtItsCallNotALineSharingItsTime(t *testing.T) {
	archiveDirectory := t.TempDir()
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"),
		`{"type":"attachment","attachment":{"type":"deferred_tools_record"},"sessionId":"s1","timestamp":"2026-10-16T10:00:00Z"}`+"\n",
		transcriptLine(t, "10:00:00", "assistant", toolUse("tu0", "Skill", map[string]any{"skill": "checkpoint"})),
		transcriptLine(t, "10:01:00", "assistant", toolUse("tu1", "Bash", map[string]any{"command": "ele checkpoint log /repo <<'EOF'\n{}\nEOF"})),
		transcriptLine(t, "10:02:00", "assistant", []map[string]any{{"text": "Item 3 is built and ready to commit.", "type": "text"}}))
	unit := dto.MiningUnit{Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z"}

	digest, err := Digest(archiveDirectory, "checkpoint-runs", unit)
	require.NoError(t, err)
	require.Contains(t, digest, `"end":"2026-10-16T10:02:00Z"`, "the run opens at its Skill call, not the attachment written at the same time")

	file, err := readTranscript(filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"))
	require.NoError(t, err)
	timing, err := timeUnit(file, TimedUnit{Analysis: "checkpoint-runs", Unit: unit})
	require.NoError(t, err)
	require.Equal(t, 120.0, timing.Seconds, "timing opens at the call too")
}

func TestDigestOfACheckpointRunReadsACompactionAsNoPrompt(t *testing.T) {
	archiveDirectory := t.TempDir()
	compaction := strings.TrimSuffix(transcriptLine(t, "10:03:30", "user", "This session is being continued from a previous conversation that ran out of context."), "}\n") + `,"isCompactSummary":true,"isVisibleInTranscriptOnly":true}` + "\n"
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"),
		transcriptLine(t, "10:00:00", "assistant", toolUse("tu0", "Skill", map[string]any{"skill": "ele:checkpoint"})),
		transcriptLine(t, "10:03:00", "assistant", toolUse("tu1", "Bash", map[string]any{"command": "ele checkpoint log /repo <<'EOF'\n{}\nEOF"})),
		compaction,
		transcriptLine(t, "10:04:00", "assistant", []map[string]any{{"text": "Item 3 is built and ready to commit.", "type": "text"}}))

	digest, err := Digest(archiveDirectory, "checkpoint-runs", dto.MiningUnit{Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z"})
	require.NoError(t, err)
	require.Contains(t, digest, `"end":"2026-10-16T10:04:00Z"`, "a compaction between the record and the report is not the user speaking")
}

func TestDigestOfACheckpointRunTakesNoReportWhileAReviewerIsOut(t *testing.T) {
	archiveDirectory := t.TempDir()
	launched := strings.TrimSuffix(transcriptLine(t, "10:01:01", "user", toolResult("tu1", "Async agent launched successfully.\nagentId: a1", false)), "}\n") + `,"toolUseResult":{"agentId":"a1","status":"async_launched"}}` + "\n"
	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"),
		transcriptLine(t, "10:00:00", "assistant", toolUse("tu0", "Skill", map[string]any{"skill": "ele:checkpoint"})),
		transcriptLine(t, "10:01:00", "assistant", toolUse("tu1", "Agent", map[string]any{"description": "Bugs review", "prompt": "Read reviewers/bugs.md"})),
		launched,
		transcriptLine(t, "10:02:00", "assistant", []map[string]any{{"text": "**Bugs reviewer:** no findings yet; it is still out.", "type": "text"}}),
		`{"type":"queue-operation","operation":"enqueue","timestamp":"2026-10-16T10:03:00Z","content":"<task-notification>\n<task-id>a1</task-id>\n<status>completed</status>"}`+"\n",
		transcriptLine(t, "10:04:00", "assistant", []map[string]any{{"text": "fixing what it found", "type": "text"}}))

	digest, err := Digest(archiveDirectory, "checkpoint-runs", dto.MiningUnit{Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T10:00:00Z"})
	require.NoError(t, err)
	require.Contains(t, digest, "fixing what it found", "a labelled text written while a reviewer is out is progress, so with no report the window runs on")
}

func TestDigestOfAReviewCycleOpensFifteenMinutesBackAndClosesAtTheNextCycle(t *testing.T) {
	archiveDirectory := t.TempDir()
	report := "The checkpoint is recorded. **Bugs reviewer:** one issue, fixed."
	lines := []string{
		transcriptLine(t, "09:00:00", "user", "the first ask"),
		transcriptLine(t, "09:10:00", "assistant", []map[string]any{{"text": "first work", "type": "text"}}),
		transcriptLine(t, "10:50:00", "user", "an earlier ask"),
		transcriptLine(t, "11:05:00", "assistant", []map[string]any{{"text": "old work", "type": "text"}}),
		transcriptLine(t, "11:10:00", "assistant", []map[string]any{{"text": "starting the review", "type": "text"}}),
		transcriptLine(t, "11:20:00", "assistant", toolUse("tu1", "Agent", map[string]any{"description": "Bugs review", "prompt": "Read reviewers/bugs.md"})),
		transcriptLine(t, "11:20:30", "user", toolResult("tu1", "One issue.", false)),
		transcriptLine(t, "11:21:00", "assistant", toolUse("tu2", "Bash", map[string]any{"command": "go test ./..."})),
		transcriptLine(t, "11:22:00", "assistant", toolUse("tu3", "Bash", map[string]any{"command": "ls internal"})),
		transcriptLine(t, "11:23:00", "assistant", toolUse("tu5", "Bash", map[string]any{"command": "~/.claude/hooks/checkpoint-reviewed.sh freeze ."})),
		transcriptLine(t, "11:24:00", "assistant", toolUse("tu6", "Bash", map[string]any{"command": "ele checkpoint density ."})),
		transcriptLine(t, "11:25:00", "user", "a question mid-cycle"),
	}
	unit := dto.MiningUnit{Reviewers: 1, Session: "s1", Source: "claude/-gateway/t1.jsonl.gz", Timestamp: "2026-10-16T11:20:00Z", Until: "2026-10-16T12:00:00Z"}
	next := []string{
		transcriptLine(t, "11:59:00", "user", "ship it"),
		transcriptLine(t, "12:00:00", "assistant", toolUse("tu4", "Agent", map[string]any{"description": "Next cycle", "prompt": "Read reviewers/bugs.md"})),
		transcriptLine(t, "12:01:00", "user", "during the next cycle"),
	}

	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"), append(lines, next...)...)
	digest, err := Digest(archiveDirectory, "review-cycles", unit)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(digest, `{"rereviews":0,"reviewers":1,"session":"s1","timestamp":"2026-10-16T11:20:00Z"}`), digest)
	require.Contains(t, digest, "TEXT old work", "the walk back stops at the first line 15 minutes before")
	require.NotContains(t, digest, "an earlier ask")
	require.Contains(t, digest, "BASH go test ./...")
	require.NotContains(t, digest, "ls internal", "only gate commands are shown")
	require.Contains(t, digest, "BASH ~/.claude/hooks/checkpoint-reviewed.sh freeze")
	require.Contains(t, digest, "BASH ele checkpoint density", "a ported script shows as its script did")
	require.Contains(t, digest, "HUMAN ship it")
	require.NotContains(t, digest, "Next cycle")
	require.NotContains(t, digest, "=== NEXT HUMAN", "with no report, nothing comes after")

	writeArchived(t, filepath.Join(archiveDirectory, "claude", "-gateway", "t1.jsonl.gz"), append(append(lines, transcriptLine(t, "11:30:00", "assistant", []map[string]any{{"text": report, "type": "text"}})), next...)...)
	digest, err = Digest(archiveDirectory, "review-cycles", unit)
	require.NoError(t, err)
	require.NotContains(t, digest, "HUMAN ship it\n")
	require.Contains(t, digest, "=== NEXT HUMAN 2026-10-16T11:59:00Z\nship it\n")
	require.NotContains(t, digest, "=== NEXT HUMAN 2026-10-16T11:25:00Z", "next humans start after the report")
}

func TestDigestRefusesAUnitItCannotPlace(t *testing.T) {
	archiveDirectory := t.TempDir()
	writeArchived(t, filepath.Join(archiveDirectory, "t1.jsonl.gz"), transcriptLine(t, "10:00:00", "user", "hello"))

	_, err := Digest(archiveDirectory, "checkpoint-runs", dto.MiningUnit{Session: "s1", Timestamp: "2026-10-16T10:00:00Z"})
	require.ErrorContains(t, err, "no archived transcript")
	_, err = Digest(archiveDirectory, "checkpoint-runs", dto.MiningUnit{Session: "s1", Source: "t1.jsonl.gz", Timestamp: "2026-10-16T11:00:00Z"})
	require.ErrorContains(t, err, "no line at that time in t1.jsonl.gz")
}
