---
name: wrap
description: Wrap up a session by asking what to keep as notes, plan inbox items or progress lines, which due reminders to close, and whether to hand off. Use when the user asks to wrap up or end a session, or asks "anything to remember or hand off?"; never start one unasked.
argument-hint: "[what the next session is for]"
---

# Wrap

Lose nothing the next session would re-derive, and record nothing it can already see. Write only what the user approves, except notes and the handoff when `auto_note_writes` is on, per step 3.

## 1. Unfinished work

In each repository the session touched, run `git status --porcelain`; where it lists changes, run `ele checkpoint check <root>`, as it reports a clean tree `not covered`. Changes the last checkpoint does not cover are unfinished. Then list what the conversation left open.

- An unfinished item of an active plan: propose a dated `as of <date>: …` line under it, saying where it stands and what remains
- Other unfinished work: say what it is and ask whether to finish it now or wrap anyway, before proposing anything else. Finish now: stop here, dropping what was drafted, and wrap again once it is done. Wrap anyway: it becomes deferred work in step 2

Done when every open piece of work is finished, has a progress line proposed, or is deferred.

## 2. Collect proposals

Leave out what the next session can already see: the code, commits, plan ticks and lines, live notes, and the active plans' next items, which the start block shows.

- **Deferred work**: an inbox item of the active plan whose `## Why` it serves, `- <date>, from <source>: <words>`, naming the plan in the question when several could take it, and never a note id or session. Serving none, a `question` note starting `Plan candidate:`
- **Notes**: what `/ele:remember`'s Propose section looks for
- **Due reminders**: `ele reminder list`, not the start block, which a compaction may have dropped. Propose closing only one whose subject this session finished; say when it is a nudge, which closing re-arms
- **Handoff**: propose one unless the start block alone sends the next session to the right place, meaning either the session has no next step, or it worked no issue or merge request and its next step is the next item of the only active plan, and nothing else is left: no change other than a plan file uncommitted in a repository the session touched, no printed command to run, no failed approach or unverified guess, and no next task the user named as an argument. Worded by `/ele:handoff`'s Write section, naming a progress line, inbox item or note proposed here rather than restating it. An argument names the task, not approval: the handoff still goes through step 3

Nothing left: say so in one line and skip to step 4. Done when every proposal is drafted with its exact words.

## 3. Ask

Run `ele config get auto_note_writes`. When it prints `true`, write the notes and the handoff without asking, the handoff once the asked proposals are answered.

Ask the rest with AskUserQuestion, loading it through ToolSearch if it is deferred: one call, one question per proposal, most costly to lose first, a second call in the same turn past the tool's four. Each `question` holds the proposal's exact words and any ids it supersedes; `header` its kind (`inbox`, `progress`, a note kind, `reminder`, `handoff`); options `Write` and `Skip`, or `Close` and `Keep` for a reminder. An `Other` that rewords one is written as reworded; for any other `Other`, ask again before writing or skipping that one.

Write each approved one yourself, never printing the command for the user:

- Inbox item: append it under the plan's `## Inbox`. Progress line: add it under its item
- Note: `ele remember` as `/ele:remember` writes one. Handoff: `ele remember --kind handoff`, with `--project` as `/ele:handoff` says. Write it last, restating any skipped item it names
- Reminder: `ele reminder close <id>`

Done when every proposal is written, skipped or closed as answered.

## 4. Report

- Each id and path written, and how many were proposed and written
- The words of each one written without asking, so a wrong one can be undone with `ele forget <id>`
- Each plan file edited and left uncommitted

End with one line. It names first any plan file left to commit, then any open work skipped, as `Open, not recorded: <gist>.`, then:

- a handoff written: `/clear, then continue`
- otherwise, a progress line written: `/clear, then /ele:plan`
- otherwise, work deferred to an inbox item or a note: `Deferred: <gist> (<note id or plan>); safe to end or /clear`
- otherwise, with no open work skipped: `Nothing pending; safe to end or /clear`

Print these lines only here, as a wrap's last line.
