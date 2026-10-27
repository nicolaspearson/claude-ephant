package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type replayLines struct {
	lines  []string
	second int
}

func (l *replayLines) add(format string, arguments ...any) *replayLines {
	l.second++
	stamp := fmt.Sprintf("2026-10-20T10:%02d:%02d.000Z", l.second/60, l.second%60)
	l.lines = append(l.lines, fmt.Sprintf(`{"timestamp":"%s","uuid":"u%d",`, stamp, l.second)+fmt.Sprintf(format, arguments...)+"}")
	return l
}

func (l *replayLines) block(reason string) *replayLines {
	return l.add(`"type":"user","message":{"content":"Stop hook feedback:\n%s"}`, reason)
}

func (l *replayLines) content() []byte {
	return []byte(strings.Join(l.lines, "\n") + "\n")
}

func (l *replayLines) prompt() *replayLines {
	return l.add(`"type":"user","message":{"content":"go on"}`)
}

func (l *replayLines) stamp() string {
	return fmt.Sprintf("2026-10-20T10:%02d:%02d.000Z", l.second/60, l.second%60)
}

func (l *replayLines) stop() *replayLines {
	return l.add(`"type":"system","subtype":"stop_hook_summary","isSidechain":false`)
}

func (l *replayLines) tool(name string, input string) *replayLines {
	return l.add(`"type":"assistant","message":{"id":"m%d","content":[{"type":"tool_use","name":%q,"input":%s}]}`, l.second+1, name, input)
}

func replayed(t *testing.T, lines *replayLines, edits ...string) []replayStop {
	t.Helper()
	stops := replayTranscript(lines.content(), edits, map[string]bool{})
	for position := range stops {
		stops[position].key = ""
	}
	return stops
}

func TestReplayTranscriptJudgesEachRecordedStop(t *testing.T) {
	const ask = "Follow /ele:remember's Propose section now"
	lines := &replayLines{}
	lines.prompt().tool("Edit", `{"file_path":"a.go"}`).block(ask).stop()
	asked := lines.stamp()
	lines.add(`"type":"assistant","message":{"id":"r","content":[{"type":"text","text":"Nothing to keep."}]}`).stop()
	active := lines.stamp()
	lines.prompt().tool("Bash", `{"command":"git status"}`).stop()
	read := lines.stamp()
	lines.prompt().tool("Bash", `{"command":"ele remember --kind fact x"}`).stop()
	noted := lines.stamp()
	lines.prompt().tool("Skill", `{"skill":"ele:checkpoint"}`).tool("Edit", `{"file_path":"b.go"}`).stop()
	checkpoint := lines.stamp()
	lines.prompt().tool("Edit", `{"file_path":"c.go"}`).stop()
	afterCheckpoint := lines.stamp()
	lines.add(`"type":"system","subtype":"stop_hook_summary","isSidechain":true`)

	require.Equal(t, []replayStop{
		{Asked: true, Asks: true, Stop: stopRecorded, Timestamp: asked, Trigger: []string{"edit"}},
		{Stop: stopRecorded, Timestamp: active, WhyNot: "hook-active"},
		{Stop: stopRecorded, Timestamp: read, WhyNot: "nothing-pending"},
		{Stop: stopRecorded, Timestamp: noted, WhyNot: "proposal"},
		{Stop: stopRecorded, Timestamp: checkpoint, WhyNot: "checkpoint"},
		{Asks: true, Stop: stopRecorded, Timestamp: afterCheckpoint, Trigger: []string{"edit"}},
	}, replayed(t, lines), "a block's feedback line is not a prompt, and a subagent's stop line is not the session's")

	lines = &replayLines{}
	lines.prompt().tool("Edit", `{"file_path":"a.go"}`).stop()
	first := lines.stamp()
	lines.prompt().tool("Edit", `{"file_path":"b.go"}`).stop()
	require.Equal(t, "stretch", replayed(t, lines)[1].WhyNot, "a stretch asks once")
	require.Equal(t, first, replayed(t, lines)[0].Timestamp)

	lines = &replayLines{}
	lines.prompt().tool("Bash", `{"command":"rm x"}`).tool("AskUserQuestion", `{"questions":[]}`).tool("Bash", `{"command":"git status"}`).stop()
	require.Equal(t, []string{"bash", "question"}, replayed(t, lines)[0].Trigger, "the work that set it pending, not a read")

	lines = &replayLines{}
	lines.prompt().tool("Agent", `{"subagent_type":"ele:review-reader","description":"read"}`).tool("Edit", `{"file_path":"a.go"}`).stop()
	require.Equal(t, "review", replayed(t, lines)[0].WhyNot)
}

func TestReplayTranscriptAppliesSubagentEditsAtTheFirstStopAfterThem(t *testing.T) {
	lines := &replayLines{}
	lines.prompt().tool("Agent", `{"description":"build"}`).stop()
	first := lines.stamp()
	lines.prompt().tool("Agent", `{"description":"build more"}`).stop()
	second := lines.stamp()
	lines.prompt().tool("Bash", `{"command":"git status"}`).stop()
	stops := replayed(t, lines, "", first, second)
	require.Equal(t, "nothing-pending", stops[0].WhyNot, "at its first stop the hook has no state and reads no subagent file, unstamped edits included")
	require.Equal(t, []string{"subagent-edit"}, stops[1].Trigger, "an edit at a later stop's stamp counts there")
	require.True(t, stops[1].Asks)
	require.Equal(t, "nothing-pending", stops[2].WhyNot, "each edit counts once")

	lines = &replayLines{}
	lines.prompt().stop()
	lines.prompt().tool("Agent", `{"description":"build"}`).tool("Bash", `{"command":"ele remember --kind fact x"}`).stop()
	require.Equal(t, "proposal", replayed(t, lines, "2026-10-20T10:00:03.500Z")[1].WhyNot, "an edit stamped before a proposal is cleared by it, though read after")
}

func TestReplayTranscriptPlacesStopsWhereNoneIsRecorded(t *testing.T) {
	const ask = "Follow /ele:remember's Propose section now"
	lines := &replayLines{}
	lines.prompt().tool("Edit", `{"file_path":"a.go"}`).block("ele checkpoint gate failed")
	blocked := lines.stamp()
	lines.block(ask).add(`"type":"assistant","message":{"id":"r","content":[{"type":"text","text":"Fixed."}]}`).prompt()
	beforePrompt := lines.stamp()
	lines.tool("Edit", `{"file_path":"b.go"}`)
	last := lines.stamp()

	require.Equal(t, []replayStop{
		{Asked: true, Asks: true, Stop: stopPlaced, Timestamp: blocked, Trigger: []string{"edit"}},
		{Stop: stopPlaced, Timestamp: beforePrompt, WhyNot: "hook-active"},
		{Stop: stopPlaced, Timestamp: last, WhyNot: "stretch"},
	}, replayed(t, lines), "blocks within 2 s are one stop; a prompt after output, and the last line, are stops")
}

func TestReplayTranscriptJudgesNoPlacedStopItHasSeen(t *testing.T) {
	const ask = "Follow /ele:remember's Propose section now"
	lines := &replayLines{}
	lines.prompt().tool("Edit", `{"file_path":"a.go"}`).block(ask)
	lines.block("ele checkpoint gate failed").add(`"type":"assistant","message":{"id":"r","content":[{"type":"text","text":"Done."}]}`).prompt()
	own := lines.stamp()
	stops := replayTranscript(lines.content(), nil, map[string]bool{"u3": true})
	require.Equal(t, []replayStop{{Stop: stopPlaced, Timestamp: own, WhyNot: "hook-active", key: "u6"}}, stops, "a copied block group is skipped whole, its later blocks included")
}

func TestStopCheckReplaysTheArchiveOnceAStopInTheWindow(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	original := &replayLines{}
	original.prompt().tool("Edit", `{"file_path":"a.go"}`).stop()
	resumed := &replayLines{lines: original.lines, second: 30}
	resumed.prompt().tool("Edit", `{"file_path":"b.go"}`).stop().prompt().stop()
	until := resumed.stamp()
	write := func(session string, lines *replayLines) {
		content := strings.ReplaceAll(string(lines.content()), `"uuid"`, `"cwd":"`+gateway+`","sessionId":"`+session+`","uuid"`)
		writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", session+".jsonl"), content)
	}
	write("orig", original)
	write("res2", resumed)
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)

	output, code = runCommand(t, "stop-check", "--replay", "--since", "2026-10-20", "--until", until)
	require.Equal(t, 0, code, output)
	var stops []replayStop
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		var stop replayStop
		require.NoError(t, json.Unmarshal([]byte(line), &stop), line)
		stops = append(stops, stop)
	}
	require.Equal(t, []replayStop{
		{Asks: true, Owner: "corp", Project: "gateway", Session: "orig", Stop: stopRecorded, Timestamp: "2026-10-20T10:00:03.000Z", Trigger: []string{"edit"}},
		{Asks: true, Owner: "corp", Project: "gateway", Session: "res2", Stop: stopRecorded, Timestamp: "2026-10-20T10:00:33.000Z", Trigger: []string{"edit"}},
	}, stops, "a stop copied into a resumed transcript is the original's, and the hook ran nothing there; a stop at --until is left out")
}

func TestStopCheckRejectsAReplayWindowItCannotRead(t *testing.T) {
	for _, args := range [][]string{{"--replay"}, {"--replay", "--since", "2026-10-20", "x"}, {"--since", "2026-10-20"}, {"--until", "2026-10-21"}} {
		_, code := runCommand(t, append([]string{"stop-check"}, args...)...)
		require.Equal(t, 2, code, args)
	}
	output, code := runCommand(t, "stop-check", "--replay", "--since", "last week")
	require.Equal(t, 2, code)
	require.Contains(t, output, `--since: takes YYYY-MM-DD or an RFC 3339 time like 2026-10-20T04:42:00Z, not "last week"`)
}

func TestReadArchivedReadsAGzippedTranscript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.jsonl.gz")
	file, err := os.Create(path)
	require.NoError(t, err)
	writer := gzip.NewWriter(file)
	_, err = writer.Write([]byte("{}\n"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
	content, err := readArchived(path)
	require.NoError(t, err)
	require.Equal(t, "{}\n", string(content))
}
