package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
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
	var first dto.Event
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &first))
	require.Equal(t, dto.Event{Kind: "skill", Name: "checkpoint", Owner: "corp", Profile: "claude", Project: "gateway", Session: "ev1", Timestamp: "2026-10-15T11:00:00Z"}, first)
	require.Empty(t, stderr.String())

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
