package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/mining"
)

func TestRunMineDigestClosesACycleAtItsSessionsNextCycle(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	line := func(clock string, body string) string {
		return `{"cwd":"` + gateway + `","sessionId":"mn2","timestamp":"2026-10-16T` + clock + `Z","uuid":"m` + clock + `",` + body + "}\n"
	}
	reviewer := func(id string, description string) string {
		return `"type":"assistant","message":{"content":[{"type":"tool_use","id":"` + id + `","name":"Agent","input":{"description":"` + description + `","prompt":"Read ~/.agents/skills/checkpoint/reviewers/bugs.md"}}]}`
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "mn2.jsonl"),
		line("09:00:00", reviewer("a1", "First review"))+
			line("10:00:00", reviewer("a2", "Second review")))
	var stdout, stderr bytes.Buffer
	digests := filepath.Join(home, "digests")
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--digests", digests, "review-cycles"}, &stdout, &stderr), stderr.String())
	digest, err := os.ReadFile(filepath.Join(digests, "mn2-2026-10-16T090000Z.txt"))
	require.NoError(t, err)
	require.Contains(t, string(digest), "AGENT First review")
	require.NotContains(t, string(digest), "Second review")
}

func TestRunMineReportReadsBothAnalysesAndTakesNothingElse(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ELE_HOME", home)
	cache := filepath.Join(home, "mining", "cache")
	_, err := mining.Store(cache, "checkpoint-runs", []mining.Judgment{{Classifier: "c", Era: "subagent", Result: json.RawMessage(`{"mode":"small"}`), Session: "s1", Timestamp: "2026-10-08T10:00:00Z"}})
	require.NoError(t, err)
	_, err = mining.Store(cache, "review-cycles", []mining.Judgment{{Classifier: "c", Result: json.RawMessage(`{"mode":"full"}`), Session: "s1", Timestamp: "2026-10-08T11:00:00Z"}})
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--report"}, &stdout, &stderr), stderr.String())
	require.Contains(t, stdout.String(), "Subagent era: run judgments 1, review-cycle judgments 1")
	require.Contains(t, stdout.String(), "Mode: full 1, small 1.")

	for _, args := range [][]string{{"mine", "--report", "checkpoint-runs"}, {"mine", "--report", "--no-sync"}} {
		stdout.Reset()
		stderr.Reset()
		require.Equal(t, 2, run(context.Background(), args, &stdout, &stderr), args)
		require.Empty(t, stdout.String(), args)
	}
}

func TestRunMinePrintsUnitsPastTheWatermarkAndCountsTheRest(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	line := func(clock string, body string) string {
		return `{"cwd":"` + gateway + `","sessionId":"mn1","timestamp":"2026-10-16T` + clock + `Z","uuid":"m` + clock + `",` + body + "}\n"
	}
	reviewer := func(id string, description string) string {
		return `"type":"assistant","message":{"content":[{"type":"tool_use","id":"` + id + `","name":"Agent","input":{"description":"` + description + `","prompt":"Read ~/.agents/skills/checkpoint/reviewers/bugs.md"}}]}`
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "mn1.jsonl"),
		line("09:00:00", `"type":"user","message":{"content":"<command-name>/checkpoint</command-name>"}`)+
			line("11:00:00", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}`)+
			line("11:01:00", reviewer("a1", "Bugs review"))+
			line("12:00:00", reviewer("a2", "Bugs review"))+
			line("12:20:00", reviewer("a3", "Re-review fixes")))
	cache := filepath.Join(home, "data", "mining", "cache")
	for _, analysis := range []string{"checkpoint-runs", "review-cycles"} {
		require.NoError(t, mining.Advance(cache, analysis, time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)))
	}

	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, run(context.Background(), []string{"mine", "checkpoint-runs"}, &stdout, &stderr), stderr.String())
	require.Equal(t, []dto.MiningUnit{{Owner: "corp", Project: "gateway", Session: "mn1", Timestamp: "2026-10-16T11:00:00Z"}}, decodeUnits(t, stdout.String()))
	require.Equal(t, "ele: unjudged units in checkpoint-runs at or before the watermark 2026-10-16T10:00:00Z: 1\n", stderr.String())

	stdout.Reset()
	stderr.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--no-sync", "review-cycles"}, &stdout, &stderr), stderr.String())
	require.Equal(t, []dto.MiningUnit{{Owner: "corp", Project: "gateway", Rereviews: 1, Reviewers: 1, Session: "mn1", Timestamp: "2026-10-16T12:00:00Z"}}, decodeUnits(t, stdout.String()), "the 11:01 cycle is the 11:00 run's")
	require.Empty(t, stderr.String())

	stdout.Reset()
	digests := filepath.Join(home, "digests")
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--no-sync", "--digests", digests, "checkpoint-runs"}, &stdout, &stderr), stderr.String())
	path := filepath.Join(digests, "mn1-2026-10-16T110000Z.txt")
	require.Equal(t, []dto.MiningUnit{{Digest: path, Owner: "corp", Project: "gateway", Session: "mn1", Timestamp: "2026-10-16T11:00:00Z"}}, decodeUnits(t, stdout.String()))
	digest, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(digest), `"session":"mn1","timestamp":"2026-10-16T11:00:00Z"`)
	require.Contains(t, string(digest), "2026-10-16T11:01:00Z AGENT Bugs review")

	stdout.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--no-sync", "--digests", digests, "review-cycles"}, &stdout, &stderr), stderr.String())
	path = filepath.Join(digests, "mn1-2026-10-16T120000Z.txt")
	require.Equal(t, []dto.MiningUnit{{Digest: path, Owner: "corp", Project: "gateway", Rereviews: 1, Reviewers: 1, Session: "mn1", Timestamp: "2026-10-16T12:00:00Z"}}, decodeUnits(t, stdout.String()))
	digest, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(digest), "2026-10-16T12:20:00Z AGENT Re-review fixes")
}

func TestRunMineListsRunsUnderThePluginName(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	line := func(clock string, body string) string {
		return `{"cwd":"` + gateway + `","sessionId":"mn5","timestamp":"2026-10-16T` + clock + `Z","uuid":"m` + clock + `",` + body + "}\n"
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "mn5.jsonl"),
		line("11:00:00", `"type":"user","message":{"content":"<command-name>/ele:checkpoint</command-name>"}`)+
			line("13:00:00", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"ele:checkpoint"}}]}`)+
			line("15:00:00", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}`))

	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, run(context.Background(), []string{"mine", "checkpoint-runs"}, &stdout, &stderr), stderr.String())
	require.Equal(t, []dto.MiningUnit{
		{Owner: "corp", Project: "gateway", Session: "mn5", Timestamp: "2026-10-16T11:00:00Z"},
		{Owner: "corp", Project: "gateway", Session: "mn5", Timestamp: "2026-10-16T13:00:00Z"},
		{Owner: "corp", Project: "gateway", Session: "mn5", Timestamp: "2026-10-16T15:00:00Z"},
	}, decodeUnits(t, stdout.String()))
}

func TestRunMineRefusesBadInput(t *testing.T) {
	recallFixture(t)
	for _, test := range []struct {
		args    []string
		message string
	}{
		{args: []string{"mine"}, message: "ele mine [--digests"},
		{args: []string{"mine", "--judgments", "j.jsonl", "--prompt", "p.md", "checkpoint-runs"}, message: "ele mine [--digests"},
		{args: []string{"mine", "--judgments", "j.jsonl", "checkpoint-runs"}, message: "ele mine [--digests"},
		{args: []string{"mine", "--prompt", "p.md", "checkpoint-runs"}, message: "ele mine [--digests"},
		{args: []string{"mine", "--units", "u.jsonl", "--digests", "d", "--judgments", "j.jsonl", "--prompt", "p.md", "checkpoint-runs"}, message: "ele mine [--digests"},
		{args: []string{"mine", "checkpoint-runs", "review-cycles"}, message: "ele mine [--digests"},
		{args: []string{"mine", "hook-blocks"}, message: `ele: unknown analysis "hook-blocks"; use checkpoint-runs, review-cycles`},
	} {
		output, code := runCommand(t, test.args...)
		require.Equal(t, 2, code, test.args)
		require.Contains(t, output, test.message, test.args)
	}
}

func TestRunMineStoresJudgmentsAndListsOnlyTheUnitsLeft(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	line := func(clock string) string {
		return `{"cwd":"` + gateway + `","sessionId":"mn3","timestamp":"2026-10-16T` + clock + `Z","uuid":"m` + clock + `","type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}}` + "\n"
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "mn3.jsonl"), line("09:00:00")+line("10:00:00")+line("11:00:00"))
	prompt := filepath.Join(home, "classify.md")
	writeFile(t, prompt, "Classify the run.\n")
	judgments := filepath.Join(home, "judgments.jsonl")
	writeFile(t, judgments, `{"session":"mn3","timestamp":"2026-10-16T09:00:00Z","mode":"full"}`+"\n"+`{"session":"mn3","timestamp":"2026-10-16T11:00:00Z","mode":"small"}`+"\n")
	digests := filepath.Join(home, "digests")

	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--digests", digests, "checkpoint-runs"}, &stdout, &stderr), stderr.String())
	stdout.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--no-sync", "--judgments", judgments, "--prompt", prompt, "--digests", digests, "checkpoint-runs"}, &stdout, &stderr), stderr.String())
	require.Equal(t, "Stored 2 of 2 judgments in checkpoint-runs under classifier "+mining.Classifier([]byte("Classify the run.\n"))[:12]+"; watermark 2026-10-16T09:00:00Z\n", stdout.String())

	stdout.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--no-sync", "checkpoint-runs"}, &stdout, &stderr), stderr.String())
	require.Equal(t, []dto.MiningUnit{{Owner: "corp", Project: "gateway", Session: "mn3", Timestamp: "2026-10-16T10:00:00Z"}}, decodeUnits(t, stdout.String()))
	require.Empty(t, stderr.String())
}

func TestRunMineListsTheNamedUnitsAndKeysAJudgmentByTheDigestFileItRead(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	line := func(clock string) string {
		return `{"cwd":"` + gateway + `","sessionId":"mn4","timestamp":"2026-10-16T` + clock + `Z","uuid":"m` + clock + `","type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}}` + "\n"
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "mn4.jsonl"), line("09:00:00")+line("10:00:00"))
	prompt := filepath.Join(home, "classify.md")
	writeFile(t, prompt, "Classify the run.\n")
	judgments := filepath.Join(home, "judgments.jsonl")
	writeFile(t, judgments, `{"session":"mn4","timestamp":"2026-10-16T09:00:00Z","mode":"full"}`+"\n")
	digests := filepath.Join(home, "digests")
	output, code := runCommand(t, "mine", "--judgments", judgments, "--prompt", prompt, "--digests", digests, "checkpoint-runs")
	require.Equal(t, 1, code)
	require.Contains(t, output, "ele: judgments line 1: open "+filepath.Join(digests, "mn4-2026-10-16T090000Z.txt"), "a judgment with no digest written stores nothing")

	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--no-sync", "--digests", digests, "checkpoint-runs"}, &stdout, &stderr), stderr.String())
	path := filepath.Join(digests, "mn4-2026-10-16T090000Z.txt")
	written, err := os.ReadFile(path)
	require.NoError(t, err)
	classified := append(written, "a later sync moved the window\n"...)
	require.NoError(t, os.WriteFile(path, classified, 0o600))
	stdout.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--no-sync", "--judgments", judgments, "--prompt", prompt, "--digests", digests, "checkpoint-runs"}, &stdout, &stderr), stderr.String())
	sum := sha256.Sum256(classified)
	held, err := mining.Load(filepath.Join(home, "data", "mining", "cache"), "checkpoint-runs")
	require.NoError(t, err)
	require.Len(t, held, 1)
	require.Equal(t, hex.EncodeToString(sum[:]), held[0].Digest, "the judgment is keyed by the digest file a classifier read, not one rebuilt at store time")

	units := filepath.Join(home, "units.jsonl")
	writeFile(t, units, `{"session":"mn4","timestamp":"2026-10-16T09:00:00Z"}`+"\n")
	stdout.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--no-sync", "--units", units, "--digests", digests, "checkpoint-runs"}, &stdout, &stderr), stderr.String())
	require.Equal(t, []dto.MiningUnit{{Digest: path, Owner: "corp", Project: "gateway", Session: "mn4", Timestamp: "2026-10-16T09:00:00Z"}}, decodeUnits(t, stdout.String()), "a judged unit behind the watermark is listed when named")
	require.Empty(t, stderr.String())

	stdout.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"mine", "--no-sync", "--judgments", judgments, "--prompt", prompt, "--digests", digests, "checkpoint-runs"}, &stdout, &stderr), stderr.String())
	require.Contains(t, stdout.String(), "Stored 1 of 1 judgments", "the rewritten digest differs from the one judged, so its re-judge is kept beside it")

	writeFile(t, units, `{"session":"mn4","timestamp":"2026-10-16T11:00:00Z"}`+"\n")
	output, code = runCommand(t, "mine", "--no-sync", "--units", units, "checkpoint-runs")
	require.Equal(t, 1, code)
	require.Contains(t, output, `ele: units line 1: no unit has session "mn4" and timestamp "2026-10-16T11:00:00Z"`)
}

func decodeUnits(t *testing.T, output string) []dto.MiningUnit {
	t.Helper()
	var units []dto.MiningUnit
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		var unit dto.MiningUnit
		require.NoError(t, json.Unmarshal([]byte(line), &unit))
		units = append(units, unit)
	}
	return units
}
