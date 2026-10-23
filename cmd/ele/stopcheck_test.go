package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

const (
	stopCheckEdit    = `{"type":"tool_use","name":"Edit","input":{"file_path":"main.go"}}`
	stopCheckGate    = `{"type":"tool_use","name":"Bash","input":{"command":"task check"}}`
	stopCheckSession = "0123456789abcdef"
	stopCheckText    = `{"type":"text","text":"Done."}`
)

type stopCheckRun struct {
	gateway        string
	home           string
	lines          strings.Builder
	messages       int
	session        string
	t              *testing.T
	transcriptPath string
}

func newStopCheckRun(t *testing.T) *stopCheckRun {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	t.Chdir(gateway)
	return &stopCheckRun{gateway: gateway, home: home, session: stopCheckSession, t: t, transcriptPath: filepath.Join(t.TempDir(), stopCheckSession+".jsonl")}
}

func (run *stopCheckRun) command(name string) {
	run.write(`{"type":"user","sessionId":"` + run.session + `","message":{"content":"<command-name>/` + name + `</command-name>"}}`)
}

func (run *stopCheckRun) prompt() {
	run.write(`{"type":"user","sessionId":"` + run.session + `","message":{"content":"go on"}}`)
}

func (run *stopCheckRun) respond(tokens int, content string) {
	run.messages++
	run.write(fmt.Sprintf(`{"type":"assistant","sessionId":%q,"message":{"id":"m%d","content":[%s],"usage":{"cache_read_input_tokens":%d,"input_tokens":0,"output_tokens":0}}}`, run.session, run.messages, content, tokens))
}

func (run *stopCheckRun) stop(directory string, stopHookActive bool) (string, stopOutput) {
	var stdout, stderr bytes.Buffer
	input := fmt.Sprintf(`{"cwd":%q,"hook_event_name":"Stop","session_id":%q,"stop_hook_active":%t,"transcript_path":%q}`, directory, run.session, stopHookActive, run.transcriptPath)
	code := stopCheck(context.Background(), nil, strings.NewReader(input), &stdout, &stderr)
	require.Equal(run.t, 0, code, stderr.String())
	require.Empty(run.t, stderr.String())
	var output stopOutput
	if stdout.Len() == 0 {
		return "", output
	}
	decoder := json.NewDecoder(strings.NewReader(stdout.String()))
	decoder.DisallowUnknownFields()
	require.NoError(run.t, decoder.Decode(&output), stdout.String())
	return stdout.String(), output
}

func (run *stopCheckRun) write(line string) {
	fmt.Fprintln(&run.lines, line)
	writeFile(run.t, run.transcriptPath, run.lines.String())
}

func TestAdvance(t *testing.T) {
	tool := func(text string) []transcript.Entry {
		return []transcript.Entry{{Kind: transcript.KindTool, Text: text}}
	}
	run := func(kind string, name string) []dto.TranscriptEvent {
		return []dto.TranscriptEvent{{Kind: kind, Name: name}}
	}
	tests := []struct {
		entries []transcript.Entry
		events  []dto.TranscriptEvent
		from    stopState
		name    string
		want    stopState
	}{
		{events: run(transcript.EventSkill, "ele:checkpoint"), from: stopState{MidTask: true, StretchBlocked: true}, name: "a checkpoint skill call starts a stretch", want: stopState{InCheckpoint: true}},
		{entries: tool("Bash: ls"), from: stopState{InCheckpoint: true}, name: "a checkpoint's own runs are not work", want: stopState{InCheckpoint: true}},
		{entries: tool("Bash: grep -rn \"ele remember \" plugin"), name: "a command naming ele remember is work", want: stopState{MidTask: true, Pending: true}},
		{events: run(transcript.EventSkill, "ele:handoff"), from: stopState{MidTask: true, Pending: true}, name: "a handoff proposes, still mid-task", want: stopState{MidTask: true, Proposing: true}},
		{events: run(transcript.EventSkill, "ele:wrap"), from: stopState{MidTask: true, Pending: true}, name: "a wrap proposes, still mid-task", want: stopState{MidTask: true, Proposing: true}},
		{entries: []transcript.Entry{{Kind: transcript.KindPrompt, Text: "go on"}}, from: stopState{Proposing: true}, name: "a prompt ends a proposal", want: stopState{}},
		{entries: []transcript.Entry{{Kind: transcript.KindPrompt, Text: "go on"}}, from: stopState{InCheckpoint: true, StretchBlocked: true}, name: "a prompt ends the checkpoint, not the stretch", want: stopState{StretchBlocked: true}},
		{entries: tool(`AskUserQuestion: {"questions":[]}`), from: stopState{Proposing: true}, name: "a question asking a remember block's notes is not work", want: stopState{Proposing: true}},
		{entries: tool(`AskUserQuestion: {"questions":[]}`), name: "a question is work for notes, not mid-task", want: stopState{Pending: true}},
		{entries: tool(`AskUserQuestion: {"questions":[]}`), from: stopState{MidTask: true, Pending: true, Proposing: true}, name: "a question keeps work pending", want: stopState{MidTask: true, Pending: true, Proposing: true}},
		{entries: tool(`AskUserQuestion: {"questions":[]}`), from: stopState{InCheckpoint: true}, name: "a question mid-checkpoint is not work", want: stopState{InCheckpoint: true}},
		{entries: tool("Read: main.go"), name: "a read is not work", want: stopState{}},
		{entries: tool("Agent: review"), name: "a subagent call is not work", want: stopState{}},
		{events: run(transcript.EventCommand, "checkpoint"), from: stopState{MidTask: true, Pending: true, StretchBlocked: true}, name: "a typed checkpoint starts a stretch, keeping pending work", want: stopState{InCheckpoint: true, Pending: true}},
		{events: run(transcript.EventCommand, "ele:remember"), from: stopState{Pending: true}, name: "a typed remember proposes", want: stopState{Proposing: true}},
		{entries: tool("Edit: main.go"), name: "an edit is work", want: stopState{MidTask: true, Pending: true}},
		{entries: tool("Bash: ele remember fact \"x\""), from: stopState{MidTask: true, Pending: true}, name: "an ele remember command proposes", want: stopState{MidTask: true}},
		{entries: tool("Bash: ele remember fact \"x\""), from: stopState{InCheckpoint: true, Pending: true}, name: "an ele remember command proposes mid-checkpoint", want: stopState{InCheckpoint: true}},
		{entries: tool("Bash: ele config get auto_note_writes"), from: stopState{Proposing: true}, name: "an option read is not work", want: stopState{Proposing: true}},
		{events: run(transcript.EventSkill, "ele:recall"), from: stopState{Pending: true}, name: "another skill changes nothing", want: stopState{Pending: true}},
		{entries: tool("Bash: ls"), name: "any Bash call is work", want: stopState{MidTask: true, Pending: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, advance(test.from, test.entries, test.events))
		})
	}
}

func TestLastContextTokensReadsBackPastALongLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	writeFile(t, path, `{"type":"assistant","message":{"id":"m1","content":[],"usage":{"input_tokens":200000}}}`+"\n"+
		`{"type":"assistant","isSidechain":true,"message":{"id":"m2","content":[],"usage":{"input_tokens":5}}}`+"\n"+
		`{"type":"user","message":{"content":"`+strings.Repeat("x", 3*tailWindow)+`"}}`+"\n"+
		`{"type":"assistant","message":{"id":"m3","usage":{"input_tok`)

	tokens, err := lastContextTokens(path)
	require.NoError(t, err)
	require.Equal(t, 200000, tokens, "the last main-thread response, behind a subagent's, a line longer than the window and a half-written line")

	empty := filepath.Join(t.TempDir(), "empty.jsonl")
	writeFile(t, empty, "")
	tokens, err = lastContextTokens(empty)
	require.NoError(t, err)
	require.Zero(t, tokens)
}

func TestRememberBlock(t *testing.T) {
	tests := []struct {
		block          bool
		from           stopState
		name           string
		stopHookActive bool
		want           stopState
	}{
		{from: stopState{}, name: "no work"},
		{from: stopState{StretchBlocked: true}, name: "no work, blocked this stretch", want: stopState{StretchBlocked: true}},
		{block: true, from: stopState{MidTask: true, Pending: true}, name: "work pending", want: stopState{MidTask: true, Proposing: true, StretchBlocked: true}},
		{from: stopState{Pending: true}, name: "work pending, Stop hook already continuing", stopHookActive: true, want: stopState{Pending: true}},
		{from: stopState{Pending: true, StretchBlocked: true}, name: "work pending, blocked this stretch", want: stopState{Pending: true, StretchBlocked: true}},
		{from: stopState{InCheckpoint: true, Pending: true}, name: "work pending, mid-checkpoint", want: stopState{InCheckpoint: true, Pending: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			block, state := rememberBlock(test.from, test.stopHookActive)
			require.Equal(t, test.block, block)
			require.Equal(t, test.want, state)
		})
	}
}

func TestRemoveStaleStopStateKeepsAWeek(t *testing.T) {
	dataDirectory := t.TempDir()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	for name, age := range map[string]time.Duration{"fresh.json": 6 * 24 * time.Hour, "stale.json": 8 * 24 * time.Hour} {
		path := filepath.Join(dataDirectory, "stop-check", name)
		writeFile(t, path, `{"band":0}`)
		require.NoError(t, os.Chtimes(path, now.Add(-age), now.Add(-age)))
	}

	require.NoError(t, removeStaleStopState(dataDirectory, now))
	require.FileExists(t, filepath.Join(dataDirectory, "stop-check", "fresh.json"))
	require.NoFileExists(t, filepath.Join(dataDirectory, "stop-check", "stale.json"))
	require.NoError(t, removeStaleStopState(t.TempDir(), now), "no state directory yet")
}

func TestRunStopCheckAsksForNotesOncePerStretch(t *testing.T) {
	run := newStopCheckRun(t)
	block := stopOutput{Decision: "block", Reason: rememberReason}

	run.respond(1000, `{"type":"tool_use","name":"Read","input":{"file_path":"main.go"}}`)
	raw, _ := run.stop(run.gateway, false)
	require.Empty(t, raw, "a read is not work")

	run.respond(1000, stopCheckEdit)
	raw, _ = run.stop(run.gateway, true)
	require.Empty(t, raw, "suppressed while a Stop hook is already continuing the turn")
	raw, output := run.stop(run.gateway, false)
	require.Equal(t, block, output, "the suppressed block waits for the next stop")
	require.Equal(t, `{"decision":"block","reason":"Follow /ele:remember's Propose section now: one AskUserQuestion call, one Write/Skip question per note, then write each approved note yourself with ele remember. Never print the command for the user to run. If nothing is worth keeping, say so in one line."}`+"\n", raw)
	raw, _ = run.stop(run.gateway, false)
	require.Empty(t, raw, "no new work since the block")

	run.prompt()
	run.respond(1000, stopCheckGate)
	raw, _ = run.stop(run.gateway, false)
	require.Empty(t, raw, "once per stretch")

	run.respond(1000, `{"type":"tool_use","name":"Skill","input":{"skill":"ele:checkpoint"}}`)
	raw, _ = run.stop(run.gateway, false)
	require.Empty(t, raw, "not while a checkpoint runs")
	run.prompt()
	run.respond(1000, stopCheckText)
	_, output = run.stop(run.gateway, false)
	require.Equal(t, block, output, "a checkpoint starts a stretch and keeps the work before it")

	for _, proposal := range []func(){
		func() { run.command("ele:remember") },
		func() {
			run.respond(1000, `{"type":"tool_use","name":"Bash","input":{"command":"ele remember decision \"Kept\""}}`)
		},
		func() { run.respond(1000, `{"type":"tool_use","name":"Skill","input":{"skill":"ele:handoff"}}`) },
	} {
		run.command("checkpoint")
		run.prompt()
		run.respond(1000, stopCheckEdit)
		proposal()
		raw, _ = run.stop(run.gateway, false)
		require.Empty(t, raw, "a proposal after the work")
	}

	run.prompt()
	run.respond(1000, stopCheckEdit)
	run.write(`{"type":"assistant","sessionId":"` + stopCheckSession + `","message":{"id":"half`)
	_, output = run.stop(run.gateway, false)
	require.Equal(t, block, output, "past a half-written last line")
}

func TestRunStopCheckReasonFollowsAutoNoteWrites(t *testing.T) {
	tests := []struct {
		config string
		name   string
		want   string
	}{
		{config: `{"auto_note_writes": tru`, name: "broken file reads as off", want: rememberReason},
		{name: "default off", want: rememberReason},
		{config: `{"auto_note_writes": true, "owners": {"corp": {"auto_note_writes": false}}}`, name: "off for this owner", want: rememberReason},
		{config: `{"auto_note_writes": true}`, name: "on", want: rememberWriteReason},
		{config: `{"owners": {"personal": {"auto_note_writes": true}}}`, name: "on for another owner", want: rememberReason},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			run := newStopCheckRun(t)
			if test.config != "" {
				writeFile(t, filepath.Join(run.home, "data", "config.json"), test.config)
			}
			run.respond(1000, stopCheckEdit)
			var stdout, stderr bytes.Buffer
			input := fmt.Sprintf(`{"cwd":%q,"session_id":%q,"transcript_path":%q}`, run.gateway, run.session, run.transcriptPath)
			require.Equal(t, 0, stopCheck(context.Background(), nil, strings.NewReader(input), &stdout, &stderr), stderr.String())
			var output stopOutput
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &output), stdout.String())
			require.Equal(t, stopOutput{Decision: "block", Reason: test.want}, output)
		})
	}
}

func TestRunStopCheckColoursTheHint(t *testing.T) {
	coloured := "\033[1mContext 150k.\033[22m Write a handoff (\033[36m/ele:handoff\033[39m), then \033[36m/clear\033[39m"
	plain := "Context 150k. Write a handoff (/ele:handoff), then /clear"
	tests := []struct {
		config  string
		name    string
		noColor string
		want    string
	}{
		{config: `{"colour": tru`, name: "broken file reads as on", want: coloured},
		{config: `{"colour": false}`, name: "colour off", want: plain},
		{config: `{"owners": {"personal": {"colour": false}}}`, name: "colour off for another owner", want: coloured},
		{config: `{"colour": false, "owners": {"corp": {"colour": true}}}`, name: "colour on for this owner", want: coloured},
		{name: "default on", want: coloured},
		{name: "NO_COLOR", noColor: "1", want: plain},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			run := newStopCheckRun(t)
			t.Setenv("NO_COLOR", test.noColor)
			if test.config != "" {
				writeFile(t, filepath.Join(run.home, "data", "config.json"), test.config)
			}
			run.respond(150000, stopCheckText)
			var stdout, stderr bytes.Buffer
			input := fmt.Sprintf(`{"cwd":%q,"session_id":%q,"transcript_path":%q}`, run.gateway, run.session, run.transcriptPath)
			require.Equal(t, 0, stopCheck(context.Background(), nil, strings.NewReader(input), &stdout, &stderr), stderr.String())
			var output stopOutput
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &output), stdout.String())
			require.Equal(t, test.want, output.SystemMessage)
		})
	}
}

func TestRunStopCheckCountsAQuestionForNotesOnly(t *testing.T) {
	run := newStopCheckRun(t)
	question := `{"type":"tool_use","name":"AskUserQuestion","input":{"questions":[{"header":"Scope","question":"Which?"}]}}`

	run.respond(150000, question)
	_, output := run.stop(run.gateway, false)
	require.Equal(t, stopOutput{Decision: "block", Reason: rememberReason, SystemMessage: "Context 150k. Write a handoff (/ele:handoff), then /clear"}, output, "a question asks for notes but is not mid-task")

	run.respond(1000, question)
	run.command("checkpoint")
	run.prompt()
	run.respond(1000, stopCheckText)
	raw, _ := run.stop(run.gateway, false)
	require.Empty(t, raw, "skipping every note the block asked for leaves nothing pending")

	run.respond(1000, `{"type":"tool_use","name":"Skill","input":{"skill":"ele:remember"}}`)
	run.respond(1000, question)
	run.respond(1000, stopCheckText)
	raw, _ = run.stop(run.gateway, false)
	require.Empty(t, raw, "skipping every note the remember skill asked for leaves nothing pending")

	run.prompt()
	run.respond(1000, question)
	_, output = run.stop(run.gateway, false)
	require.Equal(t, "block", output.Decision, "after the next prompt, a question is work again")
}

func TestRunStopCheckCountsASubagentsEdits(t *testing.T) {
	run := newStopCheckRun(t)
	subagent := filepath.Join(subagentDirectory(run.transcriptPath), "agent-a1.jsonl")
	line := func(content string) string {
		return `{"type":"assistant","isSidechain":true,"sessionId":"` + stopCheckSession + `","message":{"id":"s","content":[` + content + `]}}` + "\n"
	}

	run.respond(1000, stopCheckText)
	raw, _ := run.stop(run.gateway, false)
	require.Empty(t, raw, "no subagents yet")

	run.prompt()
	run.respond(1000, `{"type":"tool_use","name":"Agent","input":{"description":"look around"}}`)
	writeFile(t, subagent, line(stopCheckGate))
	raw, _ = run.stop(run.gateway, false)
	require.Empty(t, raw, "a subagent's Bash is not work")

	writeFile(t, subagent, line(stopCheckGate)+line(stopCheckEdit))
	_, output := run.stop(run.gateway, false)
	require.Equal(t, stopOutput{Decision: "block", Reason: rememberReason}, output, "an edit by a subagent first seen after the first stop is work")
}

func TestRunStopCheckKeepsASubagentsEditBeforeACheckpoint(t *testing.T) {
	run := newStopCheckRun(t)
	run.respond(1000, stopCheckText)
	raw, _ := run.stop(run.gateway, false)
	require.Empty(t, raw, "no subagents yet")

	run.prompt()
	run.respond(1000, `{"type":"tool_use","name":"Agent","input":{"description":"build it"}}`)
	writeFile(t, filepath.Join(subagentDirectory(run.transcriptPath), "agent-a1.jsonl"), `{"type":"assistant","isSidechain":true,"timestamp":"2026-10-20T10:00:00.000Z","message":{"id":"s","content":[`+stopCheckEdit+`]}}`+"\n")
	run.write(`{"type":"assistant","sessionId":"` + stopCheckSession + `","timestamp":"2026-10-20T10:00:05.000Z","message":{"id":"c1","content":[{"type":"tool_use","name":"Skill","input":{"skill":"ele:checkpoint"}}]}}`)
	raw, _ = run.stop(run.gateway, false)
	require.Empty(t, raw, "not while the checkpoint runs")

	run.prompt()
	run.respond(1000, stopCheckText)
	_, output := run.stop(run.gateway, false)
	require.Equal(t, stopOutput{Decision: "block", Reason: rememberReason}, output, "a delegated edit made before the checkpoint, read only once it began, is kept")
}

func TestRunStopCheckDecidesEveryCombination(t *testing.T) {
	const threshold = 150000
	home := recallFixture(t)
	elsewhere := filepath.Join(home, "elsewhere")
	require.NoError(t, os.MkdirAll(elsewhere, 0o700))
	for project, titles := range map[string][]string{"lone": {"Next item"}, "none": nil, "several": {"First", "Second"}} {
		directory := filepath.Join(home, "w", "corp", project)
		require.NoError(t, os.MkdirAll(filepath.Join(directory, ".git"), 0o700))
		t.Chdir(directory)
		for index, title := range titles {
			name := fmt.Sprintf("plan%d.md", index)
			writeFile(t, filepath.Join(directory, name), "---\nkind: plan\n---\n\n## Actions\n\n- [ ] "+title+"\n")
			result, code := runCommand(t, "plan", "start", name)
			require.Equal(t, 0, code, result)
		}
	}
	tasks := []struct {
		advice  string
		midTask bool
		name    string
		project string
	}{
		{advice: "Write a handoff (/ele:handoff), then /clear", name: "between tasks with a lone plan", project: "lone"},
		{advice: "At a pause: /compact Next item", midTask: true, name: "mid-task with a lone plan", project: "lone"},
		{advice: "At a pause: /compact", midTask: true, name: "mid-task with no plan", project: "none"},
		{advice: "At a pause: /compact", midTask: true, name: "mid-task with several plans", project: "several"},
	}
	contexts := []struct {
		name   string
		tokens int
	}{{name: "at", tokens: threshold}, {name: "over", tokens: threshold + 110000}, {name: "under", tokens: threshold - 1}}
	cases := 0
	for _, checkpoint := range []string{"none", "skill", "typed"} {
		for _, task := range tasks {
			for _, pending := range []bool{false, true} {
				for _, proposed := range []bool{false, true} {
					for _, size := range contexts {
						for _, hinted := range []bool{false, true} {
							for _, stopHookActive := range []bool{false, true} {
								for _, owned := range []bool{false, true} {
									if (checkpoint == "none" && pending && !task.midTask) || (task.midTask && !pending && !proposed) {
										continue
									}
									cases++
									name := fmt.Sprintf("checkpoint %s, %s, pending %t, proposed %t, %s, hinted %t, stop hook active %t, owned %t", checkpoint, task.name, pending, proposed, size.name, hinted, stopHookActive, owned)
									t.Run(name, func(t *testing.T) {
										run := &stopCheckRun{session: fmt.Sprintf("case%04d", cases), t: t, transcriptPath: filepath.Join(t.TempDir(), "s.jsonl")}
										proposal := `{"type":"tool_use","name":"Skill","input":{"skill":"ele:remember"}}`
										if proposed && pending {
											run.respond(1000, proposal)
										}
										if checkpoint != "none" {
											if pending && !task.midTask {
												run.respond(1000, stopCheckEdit)
											}
											if checkpoint == "skill" {
												run.respond(1000, `{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}`)
											} else {
												run.command("checkpoint")
											}
											run.prompt()
										}
										if task.midTask {
											run.respond(1000, stopCheckEdit)
										}
										if proposed && !pending {
											run.respond(1000, proposal)
										}
										run.respond(size.tokens, stopCheckText)
										if hinted {
											band := max(0, (size.tokens-threshold)/hintBand)
											writeFile(t, filepath.Join(home, "data", "stop-check", run.session+".json"), fmt.Sprintf(`{"band":%d}`, band))
										}
										directory := elsewhere
										if owned {
											directory = filepath.Join(home, "w", "corp", task.project)
										}

										var want stopOutput
										if owned && pending && !stopHookActive {
											want.Decision, want.Reason = "block", rememberReason
										}
										if owned && size.tokens >= threshold && !hinted {
											want.SystemMessage = fmt.Sprintf("Context %dk. %s", size.tokens/1000, task.advice)
										}
										_, output := run.stop(directory, stopHookActive)
										require.Equal(t, want, output)
									})
								}
							}
						}
					}
				}
			}
		}
	}
	require.Equal(t, 3*4*2*2*3*2*2*2-(1*1*1*2*3*2*2*2)-(3*3*1*1*3*2*2*2), cases, "every combination but pending between tasks with no checkpoint, and mid-task with nothing pending and nothing proposed")
}

func TestRunStopCheckHintsOncePerBandAndNamesTheLonePlan(t *testing.T) {
	run := newStopCheckRun(t)
	stop := func() (string, stopOutput) {
		return run.stop(run.gateway, false)
	}

	run.respond(149999, stopCheckText)
	raw, _ := stop()
	require.Empty(t, raw, "under the threshold")

	run.respond(150000, `{"type":"tool_use","name":"Read","input":{"file_path":"main.go"}}`)
	_, output := stop()
	require.Equal(t, stopOutput{SystemMessage: "Context 150k. Write a handoff (/ele:handoff), then /clear"}, output, "a read is not work")
	run.respond(249999, stopCheckText)
	raw, _ = stop()
	require.Empty(t, raw, "the band is hinted once")

	run.respond(250000, stopCheckEdit)
	_, output = stop()
	require.Equal(t, "Context 250k. At a pause: /compact", output.SystemMessage, "mid-task with no active plan")
	require.Equal(t, "block", output.Decision, "the hint and the remember block go out together")

	writeFile(t, filepath.Join(run.gateway, "build.md"), "---\nkind: plan\n---\n\n## Actions\n\n- [x] Parser\n- [ ] Size hint token=abc123secret\n")
	result, code := runCommand(t, "plan", "start", "build.md")
	require.Equal(t, 0, code, result)
	run.respond(350000, stopCheckText)
	raw, output = stop()
	require.Equal(t, stopOutput{SystemMessage: "Context 350k. At a pause: /compact Size hint token=[redacted:assignment]"}, output, "the lone plan's next item is the focus")
	require.Equal(t, `{"systemMessage":"Context 350k. At a pause: /compact Size hint token=[redacted:assignment]"}`+"\n", raw)

	writeFile(t, filepath.Join(run.gateway, "other.md"), "---\nkind: plan\n---\n\n## Actions\n\n- [ ] Other\n")
	result, code = runCommand(t, "plan", "start", "other.md")
	require.Equal(t, 0, code, result)
	run.respond(450000, stopCheckText)
	_, output = stop()
	require.Equal(t, stopOutput{SystemMessage: "Context 450k. At a pause: /compact"}, output, "no focus with several plans")

	run.command("checkpoint")
	run.respond(550000, stopCheckGate)
	_, output = stop()
	require.Equal(t, stopOutput{SystemMessage: "Context 550k. Write a handoff (/ele:handoff), then /clear"}, output, "a typed checkpoint ends the task, its own gate run included")

	run.prompt()
	run.respond(650000, stopCheckEdit)
	run.respond(650000, `{"type":"tool_use","name":"Skill","input":{"skill":"ele:checkpoint"}}`)
	run.respond(650000, stopCheckGate)
	run.respond(650000, stopCheckEdit)
	_, output = stop()
	require.Equal(t, stopOutput{SystemMessage: "Context 650k. Write a handoff (/ele:handoff), then /clear"}, output, "a checkpoint skill call ends the task, its gate run and fixes included")

	run.prompt()
	run.respond(40000, stopCheckGate)
	_, output = stop()
	require.Equal(t, stopOutput{Decision: "block", Reason: rememberReason}, output, "compacted under the threshold, no hint")
	statePath := filepath.Join(run.home, "data", "stop-check", stopCheckSession+".json")
	state, err := readStopState(statePath)
	require.NoError(t, err)
	require.Equal(t, noBand, state.Band, "the band resets")
	writeFile(t, statePath, `{"band":`)
	run.respond(160000, stopCheckText)
	_, output = stop()
	require.Equal(t, "Context 160k. At a pause: /compact", output.SystemMessage, "regrown past the threshold, hinted again past a cut-short state file; a Bash call after a prompt is work")

	elsewhere := filepath.Join(run.home, "elsewhere")
	require.NoError(t, os.MkdirAll(elsewhere, 0o700))
	run.respond(900000, stopCheckText)
	raw, _ = run.stop(elsewhere, false)
	require.Empty(t, raw, "no owner covers the directory")
}

func TestRunStopCheckOnARecordedTranscript(t *testing.T) {
	home := recallFixture(t)
	gateway := filepath.Join(home, "w", "corp", "gateway")
	recorded, err := os.ReadFile(filepath.Join("testdata", "stop-check.jsonl"))
	require.NoError(t, err)
	lines := strings.SplitAfter(string(recorded), "\n")
	transcriptPath := filepath.Join(t.TempDir(), "s.jsonl")
	stop := func(count int, stopHookActive bool) string {
		writeFile(t, transcriptPath, strings.Join(lines[:count], ""))
		stdin := filepath.Join(t.TempDir(), "stdin.json")
		writeFile(t, stdin, fmt.Sprintf(`{"cwd":%q,"hook_event_name":"Stop","session_id":"2be5afc9-9f1e-463a-bc5c-b67c4a969394","stop_hook_active":%t,"transcript_path":%q}`, gateway, stopHookActive, transcriptPath))
		file, err := os.Open(stdin)
		require.NoError(t, err)
		defer file.Close()
		saved := os.Stdin
		os.Stdin = file
		defer func() { os.Stdin = saved }()
		var stdout, stderr bytes.Buffer
		require.Equal(t, 0, run(context.Background(), []string{"stop-check", "--hint-at", "60000"}, &stdout, &stderr), stderr.String())
		require.Empty(t, stderr.String())
		return stdout.String()
	}

	require.Equal(t, `{"decision":"block","reason":"`+rememberReason+`","systemMessage":"Context 63k. At a pause: /compact"}`+"\n", stop(88, false), "the first turn ran Bash and proposed nothing")
	require.Empty(t, stop(len(lines), true), "the stop the block continued, in the same band")
}

func TestRunStopCheckRefusesBadInput(t *testing.T) {
	recallFixture(t)
	for _, input := range []string{"", "not json", `{"cwd":"/w","session_id":"../escape","transcript_path":"/t.jsonl"}`, `{"cwd":"/w","session_id":"","transcript_path":"/t.jsonl"}`, `{"cwd":"/w","session_id":"0123456789abcdef"}`} {
		var stdout, stderr bytes.Buffer
		code := stopCheck(context.Background(), nil, strings.NewReader(input), &stdout, &stderr)
		require.Equal(t, 1, code, input)
		require.Empty(t, stdout.String())
		require.Equal(t, "ele: stop-check reads a Stop hook's JSON on stdin, with session_id, transcript_path and cwd\n", stderr.String())
	}
}

func TestRunSyncReportsStopCheckStateItCannotRemove(t *testing.T) {
	home := recallFixture(t)
	writeFile(t, filepath.Join(home, "data", "stop-check"), "not a directory")

	output, code := runCommand(t, "sync")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "\nCould not remove old stop-check state: ")
	require.FileExists(t, filepath.Join(home, "data", "last-sync"))
}

func TestScanSubagentsCountsFileChangesOnly(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "subagents")
	path := filepath.Join(directory, "agent-a1.jsonl")
	line := func(content string) string {
		return `{"type":"assistant","isSidechain":true,"message":{"id":"s","content":[` + content + `]}}` + "\n"
	}
	reads := line(stopCheckGate) + line(`{"type":"tool_use","name":"Read","input":{"file_path":"main.go"}}`)

	edit := line(stopCheckEdit)
	writeFile(t, path, edit)
	state, err := scanSubagents(directory, stopState{})
	require.NoError(t, err)
	require.Equal(t, stopState{Subagents: map[string]int64{"agent-a1.jsonl": int64(len(edit))}}, state, "with no saved state, existing history is skipped unread")

	state, err = scanSubagents(filepath.Join(t.TempDir(), "subagents"), stopState{})
	require.NoError(t, err)
	require.Equal(t, stopState{Subagents: map[string]int64{}}, state, "no subagents directory")

	writeFile(t, path, reads)
	writeFile(t, filepath.Join(directory, "agent-a1.meta.json"), `{"agentType":"Explore"}`)
	state, err = scanSubagents(directory, state)
	require.NoError(t, err)
	require.Equal(t, stopState{Subagents: map[string]int64{"agent-a1.jsonl": int64(len(reads))}}, state, "a subagent's Bash and reads are not work")

	writeFile(t, path, reads+edit+`{"type":"assistant","isSidechain":true,"message":{"id":"half`)
	state, err = scanSubagents(directory, state)
	require.NoError(t, err)
	require.Equal(t, stopState{MidTask: true, Pending: true, Subagents: map[string]int64{"agent-a1.jsonl": int64(len(reads + edit))}}, state, "an edit is work, and the unterminated line waits")

	state, err = scanSubagents(directory, stopState{InCheckpoint: true, Subagents: map[string]int64{"agent-a1.jsonl": int64(len(reads))}})
	require.NoError(t, err)
	require.Equal(t, stopState{InCheckpoint: true, Subagents: map[string]int64{"agent-a1.jsonl": int64(len(reads + edit))}}, state, "a checkpoint's reviewers are read past, not counted")

	stamped := func(timestamp string) string {
		return `{"type":"assistant","isSidechain":true,"timestamp":"` + timestamp + `","message":{"id":"s","content":[` + stopCheckEdit + `]}}` + "\n"
	}
	checkpoint := stopState{CheckpointStarted: "2026-10-20T10:00:00.000Z", InCheckpoint: true, Subagents: map[string]int64{"agent-a1.jsonl": 0}}
	after := stamped("2026-10-20T10:00:01.000Z")
	writeFile(t, path, after)
	state, err = scanSubagents(directory, checkpoint)
	require.NoError(t, err)
	require.False(t, state.Pending, "an edit made after the checkpoint started is a reviewer's")

	before := stamped("2026-10-20T09:59:59.000Z")
	writeFile(t, path, after+before)
	state, err = scanSubagents(directory, state)
	require.NoError(t, err)
	require.Equal(t, stopState{CheckpointStarted: "2026-10-20T10:00:00.000Z", InCheckpoint: true, Pending: true, Subagents: map[string]int64{"agent-a1.jsonl": int64(len(after + before))}}, state, "an edit made before the checkpoint started is work, not mid-task")

	writeFile(t, path, edit)
	state, err = scanSubagents(directory, stopState{Subagents: map[string]int64{"agent-a1.jsonl": 5000}})
	require.NoError(t, err)
	require.Equal(t, stopState{MidTask: true, Pending: true, Subagents: map[string]int64{"agent-a1.jsonl": int64(len(edit))}}, state, "a shorter file is read again from the start")
}

func TestScanTranscriptConsumesWholeLinesOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	edit := `{"type":"assistant","message":{"id":"m1","content":[` + stopCheckEdit + `]}}` + "\n"
	writeFile(t, path, edit+`{"type":"assistant","message":{"id":"m2","content":[{"type":"tool_use","name":"Skill","input":{"skill":"ele:remember"}}]}}`)

	state, err := scanTranscript(path, stopState{Band: noBand})
	require.NoError(t, err)
	require.Equal(t, stopState{Band: noBand, MidTask: true, Offset: int64(len(edit)), Pending: true}, state, "the unterminated line waits")

	writeFile(t, path, "{}\n")
	state, err = scanTranscript(path, stopState{Band: 2, Offset: 500, StretchBlocked: true})
	require.NoError(t, err)
	require.Equal(t, stopState{Band: 2, Offset: 3}, state, "a shorter transcript is read again from the start, keeping the band")
}

func TestSizeHint(t *testing.T) {
	const threshold = 150000
	tests := []struct {
		band    int
		focus   string
		hinted  int
		message string
		midTask bool
		name    string
		tokens  int
	}{
		{band: 0, hinted: 0, midTask: true, name: "at, band hinted", tokens: 249999},
		{band: 0, hinted: noBand, message: "Context 150k. Write a handoff (/ele:handoff), then /clear", name: "at, between tasks", tokens: 150000},
		{band: 0, focus: "Size hint", hinted: noBand, message: "Context 150k. At a pause: /compact Size hint", midTask: true, name: "at, mid-task with a lone plan", tokens: 150000},
		{band: 0, hinted: noBand, message: "Context 150k. At a pause: /compact", midTask: true, name: "at, mid-task with no lone plan", tokens: 150000},
		{band: 3, hinted: 3, name: "over, band hinted", tokens: 480000},
		{band: 3, hinted: 0, message: "Context 480k. Write a handoff (/ele:handoff), then /clear", name: "over, bands skipped", tokens: 480000},
		{band: 1, hinted: 0, message: "Context 250k. Write a handoff (/ele:handoff), then /clear", name: "over, next band", tokens: 250000},
		{band: noBand, hinted: 2, name: "under resets the band", tokens: 40000},
		{band: noBand, hinted: noBand, name: "under, nothing hinted", tokens: 149999},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message, band := sizeHint(test.tokens, threshold, test.hinted, test.midTask, test.focus)
			require.Equal(t, test.message, message)
			require.Equal(t, test.band, band)
		})
	}
}
