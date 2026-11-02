package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/index"
)

func TestRunEventsPrintsOneJSONObjectPerLine(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	line := func(minute string, body string) string {
		return `{"cwd":"` + gateway + `","sessionId":"ev1","timestamp":"2026-10-15T11:` + minute + `:00Z","uuid":"e` + minute + `",` + body + "}\n"
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "ev1.jsonl"),
		line("00", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}`)+
			line("01", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"recall"}}]}`))

	var stdout, stderr bytes.Buffer
	require.Equal(t, 0, run(context.Background(), []string{"events", "--kind", "skill"}, &stdout, &stderr), stderr.String())
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 2)
	// Literal, not decoded back through dto.Event's tags: the keys are an API at 1.0, and a renamed tag would read itself back.
	require.Equal(t, `{"kind":"skill","name":"checkpoint","owner":"corp","profile":"claude","project":"gateway","session":"ev1","timestamp":"2026-10-15T11:00:00Z"}`, lines[0])
	require.Empty(t, stderr.String())

	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "ev1", "subagents", "agent-a1.jsonl"),
		line("02", `"isSidechain":true,"type":"assistant","message":{"id":"msg1","model":"claude-test","usage":{"input_tokens":9007199254740993,"output_tokens":1},"content":[{"type":"text","text":"done"}]}`))
	stdout.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"events", "--kind", "tokens"}, &stdout, &stderr), stderr.String())
	require.Equal(t, `{"attributes":{"cacheRead":0,"cacheWrite":0,"input":9007199254740993,"output":1},"kind":"tokens","name":"claude-test","owner":"corp","profile":"claude","project":"gateway","session":"ev1","subagent":true,"timestamp":"2026-10-15T11:02:00Z"}`+"\n", stdout.String(), "an integer past 2^53 prints whole")

	stdout.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"events", "--no-sync", "--kind", "skill", "--limit", "1"}, &stdout, &stderr))
	require.Equal(t, 1, strings.Count(stdout.String(), "\n"))
	require.Contains(t, stdout.String(), `"name":"recall"`, "the limit keeps the newest")
	require.Equal(t, "ele: 1 of 2 events shown, the newest; --limit 0 prints all, --since narrows\n", stderr.String())

	stdout.Reset()
	stderr.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"events", "--no-sync", "--name", "recall"}, &stdout, &stderr))
	require.Contains(t, stdout.String(), `"name":"recall"`)
	require.NotContains(t, stdout.String(), `"name":"checkpoint"`)

	stdout.Reset()
	require.Equal(t, 0, run(context.Background(), []string{"events", "--no-sync", "--during", "checkpoint"}, &stdout, &stderr))
	require.Contains(t, stdout.String(), `"name":"checkpoint"`)
	require.NotContains(t, stdout.String(), `"name":"recall"`, "another skill ends the run")
}

func TestRunEventsRefusesBadInput(t *testing.T) {
	recallFixture(t)
	for _, test := range []struct {
		args    []string
		message string
	}{
		{args: []string{"events", "--kind", "tool"}, message: `ele: unknown kind "tool"; use agent, command, compaction, hook-block, injection, reply, skill, tokens`},
		{args: []string{"events", "--limit", "-1"}, message: "ele events [--during"},
		{args: []string{"events", "--owner", "nobody"}, message: `ele: unknown owner "nobody"`},
		{args: []string{"events", "--since", "yesterday"}, message: "--since takes YYYY-MM-DD or a number of days"},
		{args: []string{"events", "stray"}, message: "ele events [--during"},
	} {
		output, code := runCommand(t, test.args...)
		require.Equal(t, 2, code, test.args)
		require.Contains(t, output, test.message, test.args)
	}
}

func TestRunEventsWarnsWhileTheIndexRebuilds(t *testing.T) {
	home := recallFixture(t)
	searchIndex, err := index.Open(filepath.Join(home, "data", "ele.db"))
	require.NoError(t, err)
	require.NoError(t, searchIndex.SetSetting(context.Background(), "index-version", "1"))
	require.NoError(t, searchIndex.Close())

	output, code := runCommand(t, "events", "--no-sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "ele: the index is not complete for this version of ele, so these events may be incomplete; run ele sync and wait for it")

	output, code = runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	output, code = runCommand(t, "events", "--no-sync")
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "incomplete")
}
