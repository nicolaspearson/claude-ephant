package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

func TestResetRebuildsAnEventsTableOfAnOlderShape(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	write(t, filepath.Join(projects, "-w", "s1.jsonl"), `{"type":"assistant","sessionId":"s1","timestamp":"2026-10-15T10:00:00Z","uuid":"u1","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}}`+"\n")
	index := open(t, filepath.Join(root, "ele.db"))
	_, err := index.db.Exec(`DROP TABLE events; CREATE TABLE events (attributes TEXT NOT NULL, id INTEGER PRIMARY KEY, kind TEXT NOT NULL, name TEXT NOT NULL, owner TEXT NOT NULL, profile TEXT NOT NULL, project TEXT NOT NULL, session TEXT NOT NULL, source TEXT NOT NULL, subagent INTEGER NOT NULL, timestamp TEXT NOT NULL, uuid TEXT NOT NULL)`)
	require.NoError(t, err)
	require.NoError(t, index.SetSetting(context.Background(), "index-version", "2"))

	require.NoError(t, index.Reset(context.Background()))
	summary := update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{})
	require.Equal(t, 1, summary.Events)
	version, err := index.Setting(context.Background(), "index-version")
	require.NoError(t, err)
	require.Equal(t, "2", version, "Reset keeps settings")
}

func TestUpdateIndexesEventsThatEntriesLack(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "w", "gateway")
	require.NoError(t, os.MkdirAll(filepath.Join(repository, ".git"), 0o700))
	projects := filepath.Join(root, "projects")
	line := func(entrypoint string, minute string, body string) string {
		return `{"cwd":"` + repository + `","entrypoint":"` + entrypoint + `","sessionId":"s1","timestamp":"2026-10-15T10:` + minute + `:00Z","uuid":"u` + minute + `",` + body + "}\n"
	}
	content := line("cli", "00", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}`) +
		line("cli", "01", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Agent","input":{"description":"review ghp_abcdefghijklmnopqrstuvwxyz0123456789","prompt":"Read reviewers/bugs.md","subagent_type":"general-purpose"}}]}`) +
		line("cli", "02", `"type":"user","isMeta":true,"message":{"content":"Stop hook feedback:\nFiles changed this turn."}`) +
		line("sdk-ts", "03", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"automated"}}]}`) +
		line("cli", "04", `"type":"user","isCompactSummary":true,"message":{"content":"continued"}`) +
		line("cli", "05", `"type":"user","message":{"content":"<command-name>/ghp_abcdefghijklmnopqrstuvwxyz0123456789</command-name>"}`)
	write(t, filepath.Join(projects, "-gateway", "s1.jsonl"), content)
	// The same lines archived a second time, as a transcript rewritten in place would be.
	write(t, filepath.Join(projects, "-gateway-copy", "s1.jsonl"), content)
	index := open(t, filepath.Join(root, "ele.db"))
	ownerMap := ownermap.OwnerMap{Owners: []string{"corp"}, Projects: map[string]string{"gateway": "corp"}}
	summary := update(t, index, projects, filepath.Join(root, "archive"), ownerMap)
	require.Equal(t, 5, summary.Events)

	events, total, err := index.Events(context.Background(), dto.EventQuery{})
	require.NoError(t, err)
	require.Equal(t, 5, total)
	require.Equal(t, []dto.Event{
		{Kind: "skill", Name: "checkpoint", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Timestamp: "2026-10-15T10:00:00Z"},
		{Attributes: map[string]any{"description": "review [redacted:github]", "reviewers": []any{"bugs"}}, Kind: "agent", Name: "general-purpose", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Timestamp: "2026-10-15T10:01:00Z"},
		{Attributes: map[string]any{"reason": "Files changed this turn."}, Kind: "hook-block", Name: "Stop", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Timestamp: "2026-10-15T10:02:00Z"},
		{Kind: "compaction", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Timestamp: "2026-10-15T10:04:00Z"},
		{Kind: "command", Name: "[redacted:github]", Owner: "corp", Profile: "claude", Project: "gateway", Session: "s1", Source: "claude/-gateway/s1.jsonl.gz", Timestamp: "2026-10-15T10:05:00Z"},
	}, events)

	events, total, err = index.Events(context.Background(), dto.EventQuery{Kind: "skill", Name: "checkpoint", Since: "2026-10-15"})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, events, 1)
	events, total, err = index.Events(context.Background(), dto.EventQuery{Limit: 2})
	require.NoError(t, err)
	require.Equal(t, 5, total)
	require.Equal(t, []string{"compaction", "command"}, []string{events[0].Kind, events[1].Kind}, "the newest two, oldest first")
	events, _, err = index.Events(context.Background(), dto.EventQuery{Since: "2026-10-15T10:05:00Z"})
	require.NoError(t, err)
	require.Len(t, events, 1)
	events, _, err = index.Events(context.Background(), dto.EventQuery{Since: "2026-10-16"})
	require.NoError(t, err)
	require.Empty(t, events)

	require.NoError(t, index.Reset(context.Background()))
	_, total, err = index.Events(context.Background(), dto.EventQuery{})
	require.NoError(t, err)
	require.Zero(t, total)
}

func TestUpdateKeysEventsByLineAndPlace(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	keyless := `{"type":"assistant","sessionId":"s1","timestamp":"2026-10-15T10:00:00Z","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}}` + "\n"
	twice := `{"type":"assistant","sessionId":"s1","timestamp":"2026-10-15T10:01:00Z","uuid":"u1","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}},{"type":"tool_use","name":"Skill","input":{"skill":"checkpoint"}}]}}` + "\n"
	write(t, filepath.Join(projects, "-w", "s1.jsonl"), keyless+keyless+twice)
	index := open(t, filepath.Join(root, "ele.db"))
	summary := update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{})
	require.Equal(t, 4, summary.Events, "two lines without an id, and two calls on one line")
}

func TestUpdateNamesAToolErrorByItsCall(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	transcript := filepath.Join(projects, "-w", "s1.jsonl")
	line := func(minute string, uuid string, body string) string {
		return `{"sessionId":"s1","timestamp":"2026-10-15T10:` + minute + `:00Z","uuid":"` + uuid + `",` + body + "}\n"
	}
	call := func(minute string, uuid string, command string) string {
		return line(minute, uuid, `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"`+command+`"}}]}`)
	}
	edit := func(minute string, uuid string) string {
		return line(minute, uuid, `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/w/a.go","old_string":"`+uuid+`"}}]}`)
	}
	failure := func(minute string, uuid string, source string, text string) string {
		return line(minute, uuid, `"type":"user"`+source+`,"message":{"content":[{"type":"tool_result","is_error":true,"content":"`+text+`"}]}`)
	}
	long := "echo " + strings.Repeat("y", 400)
	write(t, transcript, call("00", "a1", "ele checkpoint log /w")+failure("01", "r1", `,"sourceToolAssistantUUID":"a1"`, "Exit code 2")+call("02", "a2", "ele checkpoint log /w"))
	index := open(t, filepath.Join(root, "ele.db"))
	update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{})
	// Resumed past the call: the second failure's call was indexed by the first sync.
	appendTo(t, transcript, failure("03", "r2", `,"sourceToolAssistantUUID":"a2"`, "Exit code 2 ghp_abcdefghijklmnopqrstuvwxyz0123456789")+call("04", "a3", "ele checkpoint log --help /w")+failure("05", "r3", "", "Exit code 1")+
		edit("06", "e1")+failure("07", "r4", `,"sourceToolAssistantUUID":"e1"`, "String to replace not found")+edit("08", "e2")+
		call("09", "b1", "echo first")+call("09", "b2", "echo sibling")+failure("10", "r5", `,"sourceToolAssistantUUID":"b1"`, "Exit code 1")+call("11", "b3", "echo first")+
		call("12", "c1", long)+failure("13", "r6", `,"sourceToolAssistantUUID":"c1"`, "Exit code 1")+call("14", "c2", long+"z")+
		call("15", "d1", "echo last")+failure("16", "r7", `,"sourceToolAssistantUUID":"d1"`, "Exit code 1"))
	update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{})

	events, _, err := index.Events(context.Background(), dto.EventQuery{Kind: "tool-error"})
	require.NoError(t, err)
	place := dto.Event{Kind: "tool-error", Profile: "claude", Session: "s1", Source: "claude/-w/s1.jsonl.gz"}
	at := func(minute string, name string, attributes map[string]any) dto.Event {
		event := place
		event.Attributes, event.Name, event.Timestamp = attributes, name, "2026-10-15T10:"+minute+":00Z"
		return event
	}
	require.Equal(t, []dto.Event{
		at("01", "Bash", map[string]any{"call": "Bash: ele checkpoint log /w", "callLine": "a1", "error": "Exit code 2", "retried": true}),
		at("03", "Bash", map[string]any{"call": "Bash: ele checkpoint log /w", "callLine": "a2", "error": "Exit code 2 [redacted:github]", "retried": false}),
		at("05", "", map[string]any{"error": "Exit code 1"}),
		at("07", "Edit", map[string]any{"call": "Edit: /w/a.go", "callLine": "e1", "error": "String to replace not found"}),
		at("10", "Bash", map[string]any{"call": "Bash: echo first", "callLine": "b1", "error": "Exit code 1", "retried": true}),
		at("13", "Bash", map[string]any{"call": "Bash: " + long[:300] + "…", "callLine": "c1", "error": "Exit code 1"}),
		at("16", "Bash", map[string]any{"call": "Bash: echo last", "callLine": "d1", "error": "Exit code 1"}),
	}, events)
}

func TestEventsFollowAHookBlockToItsReply(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	line := func(minute string, body string) string {
		return `{"sessionId":"s1","timestamp":"2026-10-15T10:` + minute + `:00Z","uuid":"u` + minute + `",` + body + "}\n"
	}
	said := func(minute string, sidechain bool, text string) string {
		flag := ""
		if sidechain {
			flag = `"isSidechain":true,`
		}
		return line(minute, flag+`"type":"assistant","message":{"content":[{"type":"text","text":"`+text+`"}]}`)
	}
	block := func(minute string) string {
		return line(minute, `"type":"user","isMeta":true,"message":{"content":"Stop hook feedback:\nNo checkpoint was reported.\n\nChanged: a.go"}`)
	}
	prompt := func(minute string, text string) string {
		return line(minute, `"type":"user","origin":{"kind":"human"},"message":{"content":"`+text+`"}`)
	}
	write(t, filepath.Join(projects, "-w", "s1.jsonl"), said("00", false, "All done.")+
		block("01")+said("02", false, "This isn't a checkpoint; the handler is unwired.")+prompt("03", "go on")+
		block("04")+line("05", `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"ele:checkpoint"}}]}`)+said("06", false, "Not done until the review is in.")+
		block("07")+prompt("08", "go on")+said("09", false, "Answering the prompt.")+
		block("10")+said("11", true, "subagent prose, not a checkpoint")+said("12", false, "Checkpointing now.")+
		prompt("13", "go on")+block("14")+prompt("15", `<command-name>/ele:checkpoint</command-name>`)+
		line("16", `"type":"user","isMeta":true,"message":{"content":"Stop hook feedback:\nPropose notes now."}`)+said("17", false, "Nothing worth keeping, mid-task."))
	index := open(t, filepath.Join(root, "ele.db"))
	update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{})

	events, _, err := index.Events(context.Background(), dto.EventQuery{Kind: "hook-block"})
	require.NoError(t, err)
	var follow []map[string]any
	for _, event := range events {
		follow = append(follow, event.Attributes)
	}
	reason := "No checkpoint was reported."
	changed := []any{"a.go"}
	require.Equal(t, []map[string]any{
		{"changed": changed, "checkpointed": false, "disputed": true, "reason": reason, "reply": "This isn't a checkpoint; the handler is unwired.", "replyAt": "2026-10-15T10:02:00Z"},
		{"changed": changed, "checkpointed": true, "disputed": false, "reason": reason, "reply": "Not done until the review is in.", "replyAt": "2026-10-15T10:06:00Z"},
		{"changed": changed, "checkpointed": false, "reason": reason},
		{"changed": changed, "checkpointed": false, "disputed": false, "reason": reason, "reply": "Checkpointing now.", "replyAt": "2026-10-15T10:12:00Z"},
		{"changed": changed, "checkpointed": true, "reason": reason},
		{"reason": "Propose notes now."},
	}, follow)
}

func TestEventsDuringKeepsOnlyARunsEvents(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	line := func(session string, at string, body string) string {
		return `{"sessionId":"` + session + `","timestamp":"2026-10-15T` + at + `Z","uuid":"` + session + at + `",` + body + "}\n"
	}
	skill := func(session string, at string, name string) string {
		return line(session, at, `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"`+name+`"}}]}`)
	}
	command := func(session string, at string, name string) string {
		return line(session, at, `"type":"user","message":{"content":"<command-name>/`+name+`</command-name>"}`)
	}
	prompt := func(at string, text string) string {
		return line("s1", at, `"type":"user","message":{"content":"`+text+`"}`)
	}
	bash := func(at string, command string) string {
		return line("s1", at, `"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"`+command+`"}}]}`)
	}
	failure := func(session string, at string) string {
		return line(session, at, `"type":"user","message":{"content":[{"type":"tool_result","is_error":true,"content":"failed at `+at+`"}]}`)
	}
	write(t, filepath.Join(projects, "-w", "s1.jsonl"), failure("s1", "09:59:00")+skill("s1", "10:00:00", "ele:checkpoint")+failure("s1", "10:01:00")+
		prompt("10:03:00", "yes, fix them")+failure("s1", "10:04:00")+bash("10:06:00", "cd /w; ele checkpoint log /w")+failure("s1", "10:06:30")+
		prompt("10:07:00", "now the next item")+failure("s1", "10:08:00")+
		command("s1", "11:00:00", "ele:checkpoint")+failure("s1", "11:01:00")+prompt("11:02:00", "never mind")+failure("s1", "11:03:00")+
		skill("s1", "14:00:00", "ele:checkpoint")+failure("s1", "14:01:00")+skill("s1", "14:02:00", "ele:remember")+failure("s1", "14:03:00")+
		skill("s1", "14:04:00", "checkpoint")+failure("s1", "16:04:00")+failure("s1", "16:04:01")+
		command("s1", "17:00:00", "ticket-audit")+failure("s1", "17:01:00")+prompt("17:02:00", "thanks")+failure("s1", "17:03:00")+
		skill("s1", "18:00:00", "ele:checkpoint")+failure("s1", "18:01:00")+prompt("18:05:00", "never mind")+failure("s1", "18:10:00")+
		command("s1", "19:00:00", "checkpoint")+failure("s1", "19:10:00")+bash("19:30:00", "ele checkpoint log /w")+failure("s1", "19:31:00")+prompt("19:32:00", "next")+failure("s1", "19:33:00")+
		skill("s1", "21:00:00", "checkpoint")+failure("s1", "21:01:00")+prompt("21:02:00", "yes")+failure("s1", "21:03:00")+
		bash("21:04:00", "~/.claude/hooks/checkpoint-log.sh /w")+prompt("21:05:00", "next")+failure("s1", "21:06:00")+
		skill("s1", "22:00:00", "ele:plan-audit")+failure("s1", "22:01:00")+prompt("22:02:00", "yes, apply them")+failure("s1", "22:03:00")+
		bash("22:04:00", "ele ticket ledger record --manifest m --records r")+prompt("22:05:00", "next")+failure("s1", "22:06:00"))
	write(t, filepath.Join(projects, "-w", "s1", "subagents", "agent-1.jsonl"), failure("s1", "10:02:00")+skill("s1", "10:02:30", "ele:remember")+failure("s1", "10:02:40"))
	write(t, filepath.Join(projects, "-w", "s2.jsonl"), failure("s2", "10:01:00"))
	index := open(t, filepath.Join(root, "ele.db"))
	update(t, index, projects, filepath.Join(root, "archive"), ownermap.OwnerMap{})

	during := func(skill string) []string {
		events, total, err := index.Events(context.Background(), dto.EventQuery{During: skill, Kind: "tool-error"})
		require.NoError(t, err)
		var errors []string
		for _, event := range events {
			errors = append(errors, event.Attributes["error"].(string))
		}
		require.Len(t, errors, total)
		return errors
	}
	require.Equal(t, []string{
		"failed at 10:01:00", "failed at 10:02:00", "failed at 10:02:40", "failed at 10:04:00", "failed at 10:06:30",
		"failed at 11:01:00",
		"failed at 14:01:00", "failed at 16:04:00",
		"failed at 18:01:00", "failed at 19:10:00", "failed at 19:31:00",
		"failed at 21:01:00", "failed at 21:03:00",
	}, during("checkpoint"), "a prompt ends a run only after its own closing call, not a later run's, or where it has none; a subagent's skill call never does")
	require.Equal(t, during("checkpoint"), during("ele:checkpoint"))
	require.Equal(t, []string{"failed at 17:01:00"}, during("ticket-audit"))
	require.Equal(t, []string{"failed at 22:01:00", "failed at 22:03:00"}, during("plan-audit"))
	require.Empty(t, during("point"), "a name matches whole, with or without a plugin")
}
