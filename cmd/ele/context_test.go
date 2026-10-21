package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

func TestRunContextListsOnlyLiveNotes(t *testing.T) {
	home := recallFixture(t)
	elsewhere := filepath.Join(home, "elsewhere")
	require.NoError(t, os.MkdirAll(elsewhere, 0o700))
	t.Chdir(elsewhere)

	output, code := runCommand(t, "remember", "--kind", "decision", "--project", "gateway", "Pin pydantic below 2.14.")
	require.Equal(t, 0, code, output)
	old := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "decision", "--project", "gateway", "--supersedes", old.ID, "Pin pydantic below 2.15.")
	require.Equal(t, 0, code, output)
	live := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "question", "--project", "gateway", "Does the wrapper need pydantic at all?")
	require.Equal(t, 0, code, output)
	question := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "billing", "--owner", "corp", "--global", "Every corp repository uses Go 1.25.")
	require.Equal(t, 0, code, output)
	global := readRemembered(t, output)
	output, code = runCommand(t, "remember", "--kind", "fact", "--project", "billing", "--owner", "corp", "Another project's fact.")
	require.Equal(t, 0, code, output)
	elsewhereNote := readRemembered(t, output)

	output, code = runCommand(t, "context", "--project", "gateway")
	require.Equal(t, 0, code, output)
	require.True(t, strings.HasPrefix(output, "Live notes from ele for corp · gateway:\n"), output)
	require.Contains(t, output, live.ID)
	require.Contains(t, output, question.ID)
	require.Contains(t, output, global.ID)
	require.NotContains(t, output, old.ID)
	require.NotContains(t, output, elsewhereNote.ID)
	require.Less(t, strings.Index(output, question.ID), strings.Index(output, live.ID))

	output, code = runCommand(t, "context", "--project", "nowhere", "--owner", "personal")
	require.Equal(t, 0, code, output)
	require.Equal(t, "ele has no live notes for personal · nowhere.\n", output)
}

func TestRunContextNamesAPendingHandoff(t *testing.T) {
	home := recallFixture(t)
	t.Chdir(filepath.Join(home, "w", "corp", "gateway"))
	output, code := runCommand(t, "remember", "--kind", "handoff", "--session", "", "Next: item 3.")
	require.Equal(t, 0, code, output)
	handoff := readRemembered(t, output)

	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.Equal(t, "ele has no live notes for corp · gateway.\nHandoff for the next startup or /clear:\n- "+handoff.Date+" handoff "+handoff.ID+": Next: item 3.\n", output)

	older := notes.Note{Body: "Next: item 2.", Date: "2026-10-01", ID: "20261001-aaaaaa", Kind: notes.KindHandoff, Owner: "corp", Project: "gateway", Scope: notes.ScopeProject}
	_, err := notes.Write(filepath.Join(home, "data", "notes"), older)
	require.NoError(t, err)
	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.True(t, strings.HasSuffix(output, "\nHandoffs for the next startup or /clear, newest first:\n- "+handoff.Date+" handoff "+handoff.ID+": Next: item 3.\n- 2026-10-01 handoff 20261001-aaaaaa: Next: item 2.\n"), output)

	require.NoError(t, notes.MarkShown(filepath.Join(home, "data", "notes"), handoff, time.Now()))
	require.NoError(t, notes.MarkShown(filepath.Join(home, "data", "notes"), older, time.Now()))
	output, code = runCommand(t, "context")
	require.Equal(t, 0, code, output)
	require.Equal(t, "ele has no live notes for corp · gateway.\n", output)
}

func TestRunContextRefusesAsRememberDoes(t *testing.T) {
	home := recallFixture(t)
	for _, owner := range []string{"corp", "personal"} {
		repository := filepath.Join(home, "w", owner, "shared")
		require.NoError(t, os.MkdirAll(filepath.Join(repository, ".git"), 0o700))
		writeFile(t, filepath.Join(home, ".claude", "projects", "-shared-"+owner, "s-"+owner+".jsonl"),
			`{"type":"user","cwd":"`+repository+`","sessionId":"s-`+owner+`","timestamp":"2026-10-15T09:00:00Z","uuid":"u-`+owner+`","message":{"content":"shared work"}}`+"\n")
	}
	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	elsewhere := filepath.Join(home, "elsewhere")
	require.NoError(t, os.MkdirAll(elsewhere, 0o700))
	t.Chdir(elsewhere)

	for _, args := range [][]string{{"--project", " "}, {"--project", "gateway", "--owner", "nobody"}, {"--project", "nowhere"}, {"--project", "shared"}} {
		contextOutput, contextCode := runCommand(t, append([]string{"context"}, args...)...)
		rememberOutput, rememberCode := runCommand(t, append(append([]string{"remember", "--kind", "fact"}, args...), "Never written.")...)
		require.NotZero(t, contextCode, args)
		require.Equal(t, rememberCode, contextCode, args)
		require.Equal(t, rememberOutput, contextOutput, args)
	}
	output, _ = runCommand(t, "context", "--project", "shared")
	require.Contains(t, output, `project "shared" has entries under owners corp, personal; pass --owner <name>`)
}

func TestRunContextSizeReadsTheNewestMainThreadResponse(t *testing.T) {
	home := recallFixture(t)
	assistant := func(sidechain string, input int, cacheWrite int, cacheRead int, output int) string {
		return fmt.Sprintf(`{"type":"assistant","isSidechain":%s,"message":{"id":"m","usage":{"input_tokens":%d,"cache_creation_input_tokens":%d,"cache_read_input_tokens":%d,"output_tokens":%d},"content":[]}}`+"\n", sidechain, input, cacheWrite, cacheRead, output)
	}
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "sz1.jsonl"),
		assistant("false", 1, 9, 80, 10)+
			assistant("false", 1, 99, 150, 50)+
			assistant("true", 1, 9000, 999, 9)+
			assistant("false", 0, 0, 0, 0)+
			`{"type":"user","message":{"content":"next"}}`)
	writeFile(t, filepath.Join(home, ".claude", "projects", "-gateway", "sz2.jsonl"), `{"type":"user","message":{"content":"hello"}}`+"\n")

	t.Setenv("CLAUDE_CODE_SESSION_ID", "sz1")
	output, code := runCommand(t, "context", "--size")
	require.Equal(t, 0, code, output)
	require.Equal(t, "300\n", output, "input, cache write, cache read and output of the last main-thread response with usage")

	output, code = runCommand(t, "context", "--size", "--project", "gateway")
	require.Equal(t, 2, code)
	require.Contains(t, output, "ele context --size\n")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "sz2")
	output, code = runCommand(t, "context", "--size")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: session sz2 has no response with token usage yet\n", output)
	t.Setenv("CLAUDE_CODE_SESSION_ID", "nowhere")
	output, code = runCommand(t, "context", "--size")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: found 0 transcripts for session nowhere, not one\n", output)
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	output, code = runCommand(t, "context", "--size")
	require.Equal(t, 1, code)
	require.Equal(t, "ele: --size needs CLAUDE_CODE_SESSION_ID to name the running session; it is \"\"\n", output)
}
