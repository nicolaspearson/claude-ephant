---
kind: plan
created: 2026-10-20
spec: .documentation/specs/design.md
---

# Lessons from the mattpocock skills

## Why

A comparison with the mattpocock-skills plugin (wayfinder, grilling, writing-for-agents, diagnosing-bugs, code-review) found ideas worth taking, though not a dependency on that plugin: a record of what a plan ruled out, design questions asked in rounds, a Stop hook that sees decision-only sessions and subagent work, a checkpoint reviewer that checks the diff against the item it delivers, and a size pass on the two largest skills.

## Context

- As of 2026-10-20 stop-check counts Bash, Edit, NotebookEdit and Write as work (`cmd/ele/stopcheck.go:41`), one check setting both the notes ask and mid-task (`:69`)
- As of 2026-10-20 subagent transcripts live in `<session>/subagents/agent-*.jsonl`, and stop-check reads only the parent transcript
- As of 2026-10-20 the plan parser ignores an unknown `##` section's lines (`internal/plan/plan.go:114-126`), pinned by `TestParseReadsActionsInFileOrder`
- As of 2026-10-20 checkpoint has two reviewers, bugs and conventions-prose, and neither checks the diff against the ticket or plan item; only §10's tick judgement does
- As of 2026-10-20 a warm stop-check that shows a message takes 0.8-1s on a 153 MB transcript, and a stop with nothing to show 0.01s

## Decisions

- This plan is separate from `move-checkpoint-into-ele.md`; the checkpoint and ticket-audit items wait for its phase 1, so those skills are edited once, inside ele
- ele's skills say an idea from another plugin in their own words rather than call that plugin's skill: its names change between versions, as grill-me became grilling
- Inbox triage either rules an item out of scope, which leaves a line, or deletes it, for a duplicate or an item already done. An item ruled out returns only if the plan's why is redrawn
- `## Out of scope` is a plan's last section, empty in the template; a line is `- <gist>: <why> (ruled <date>, from <source>)`. `ele plan show` neither counts nor shows it
- At close, out-of-scope lines fold into the spec section the plan implemented as a `Not doing:` list; with no spec, each is proposed as a `decision` note
- `/ele:plan new` and `convert` ask design questions in rounds: every question whose prerequisites are settled, numbered, each with a recommended answer; facts are looked up, `ele recall` first, never asked. Pickup's approvals stay one question each
- AskUserQuestion is work for the notes ask only, not mid-task: a question changes no files. One between a remember block and the next prompt is not work. A declined one still counts, so the transcript parser needs no tool results
- A subagent's Edit, NotebookEdit and Write calls are work for both the notes ask and mid-task, read from its own transcript with an offset per file. Not its Bash, decided 2026-10-20: as of that date this session's Explore agent made 21 Bash calls and its reviewers 4 to 17 each, all reads, so subagent Bash would count every research run. Bar: no slower than today beyond noise on the 153 MB session; if not met, only subagent files changed since the state was saved are scanned
- Checkpoint gets a spec reviewer, `reviewers/spec.md`, run as `checkpoint-reviewer` only when the change names a plan item or fetched ticket, its text quoted in the brief: a third parallel reviewer in full mode, a file added to the single reviewer's list in small mode. It reports what the item asked that is missing or partial, what the diff does unasked, and what looks implemented wrong, each quoting the item's line, in its own section, unranked against the others. Its findings go through §5 Verify, and a verified missing or partial finding turns §10's tick into a progress line
- The writing-for-agents pass on checkpoint and ticket-audit produces findings first, applied on approval, with no line-count target: SKILL.md keeps what every run needs

## Actions

- [x] Plans get `## Out of scope`: pickup's inbox triage splits drop into rule out and delete; the section in `template.md` and the plan skill's format; the closing action folds the lines into the spec as `Not doing:`, or proposes decision notes without one; a parser test, seen red, that the section's lines are not counted; the design spec's Plans section updated
  - asserted in the item 1 commit: TestParseReadsActionsInFileOrder's sample ends with an `## Out of scope` section holding a plain line and a `- [ ]` line; counting either as inbox or action turned it red (Inbox 4, an extra 'Not an action' item); skill, template and design spec carry the section, the rule-out/delete split and the closing fold; task check green
- [x] `/ele:plan new` and `convert` ask in rounds, look facts up and recall before asking, in the plan skill's own words; the design spec updated
  - asserted in the item 2 commit: the plan skill's 'Asking in rounds' section, cited by `new` step 1 and `convert` step 2, asks settled-prerequisite questions numbered with a recommended answer, recall then code before asking, decisions to `## Decisions` and item facts as dated lines; design spec matches; read-checked by two reviews, no real session yet
- [x] stop-check: AskUserQuestion sets the notes ask but not mid-task; one after a remember block and before the next prompt is not work; table tests for each, seen red; `stop-check.md` updated
  - asserted in the item 3 commit: TestAdvance and TestRememberBlock rows plus TestRunStopCheckCountsAQuestionForNotesOnly, run through stopCheck on a written transcript: a question alone blocks with the handoff hint (not mid-task); a question after a remember block or skill, skipped, leaves nothing pending; after the next prompt a question counts again; six mutations each turned them red; task check FORCE=1 green. Not yet seen in a real session: needs task install
- [x] stop-check reads `subagents/agent-*.jsonl` for work tools with an offset per file; tests, seen red; timed on the 153 MB session against the bar in Decisions; `stop-check.md` and the design spec updated
  - asserted in the item 4 commit: TestScanSubagentsCountsFileChangesOnly, TestRunStopCheckCountsASubagentsEdits and TestRunStopCheckKeepsASubagentsEditBeforeACheckpoint, through stopCheck on written transcripts: a subagent's Edit blocks, its Bash does not, reviewers' edits during a checkpoint do not, an edit stamped before the checkpoint does, no saved state skips history; ten mutations each red; timed on the 153 MB session (181 MB subagents): no saved state 0.69-0.76s vs HEAD 0.68-0.73s, with state 0.04-0.05s both; task check FORCE=1 green. Not yet seen in a real session: needs task install
- [ ] Checkpoint spec reviewer, as in Decisions
  - blocked: waits for phase 1 of `.documentation/plans/move-checkpoint-into-ele.md`
- [ ] writing-for-agents findings for checkpoint and ticket-audit, applied on approval
  - blocked: waits for phase 1 of `.documentation/plans/move-checkpoint-into-ele.md`
- [ ] Fold what this plan built into the spec, delete this plan, and run `ele plan done .documentation/plans/skill-lessons.md`

## Inbox

- 2026-10-20, from item 4's re-review: a subagent edit, a checkpoint and a notes proposal in one turn with no stop between re-ask for notes after the next prompt, since the subagent file is first read after the proposal cleared Pending. Also untested: a subagent edit with no timestamp, or one stamped equal to the checkpoint's start
- 2026-10-20, from the install of 8596ec2 at 10:45: from 2026-10-27, judge a week of stop-check notes asks: per Stop hook-block event, the trigger rebuilt from the tool calls before it (Bash, question, subagent edit), notes written after it in that session, asks repeated without new work, and sessions with questions or subagent edits that got no ask
- 2026-10-20, from the comparison with another repository's skill tooling: a skill lint in Go under `task check`, after its lint script: a model-invoked skill's description says when to use it, relative links resolve, SKILL.md under 500 lines, extra files under `references/`, `scripts/` or `assets/`, the rest moved or allowed by name. As of 2026-10-20 `plan/template.md` and `mine/classify-*.md` sit at the skill root, and phase 1 of `.documentation/plans/move-checkpoint-into-ele.md` brings checkpoint's `BRANCH.md`, `HEAD-EXPORT.md` and `reviewers/` and ticket-audit's `PLAN.md`. Run it after that phase, whose rename to `ele:` names is what breaks links
- 2026-10-20, from the same comparison: every skill step ends with an exit condition, as that repository's walkthrough (`Done when`) and CI-template migration (`Completion:`) skills do; for checkpoint and ticket-audit inside the writing-for-agents item, and for handoff, mine, plan, recall and remember. As of 2026-10-20 ticket-audit's §4, checkpoint's §1 and §2 and plan's Pickup step 6 have one
- 2026-10-20, from the same comparison: checkpoint and ticket-audit take that repository's loki skill's error rule wherever they run a tool: quote the error verbatim, name its class from a table of known errors and their fixes, apply exactly one fix and run once more, never the same call unchanged; then stop and report, per the user's two-attempts rule. Each skill needs its own table. Inside the writing-for-agents item

## Out of scope
