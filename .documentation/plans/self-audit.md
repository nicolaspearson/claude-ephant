---
kind: plan
created: 2026-11-22
---

# Self-audit

## Why

ele's own records show where ele misbehaves: Stop-hook blocks and the replies
to them, the notes ask the Stop hook would make, the checkpoint log, and the
mining judgments. Reading them is spread over separate commands and the
maintainer's private recipes, so it happens rarely and its findings stay in
chat. A skill in this repository reads them together over a window and turns
what it finds into plan candidates.

## Requirements

1. A skill in this repository, run by contributors from the checkout, reading
   ele's records through the ele CLI only, never transcript or log files
   directly
2. It reads Stop-hook blocks with their replies and disputes, the notes-ask
   replay, the checkpoint log, and the stored mining report with its age
3. It proposes only plan candidates, through `/ele:remember`, skipping a
   finding whose cause an open plan candidate, an active plan's item or an
   inbox line already names, read from `ele context` and the plan files; each
   skip listed with what held it
4. It takes a window, by default the last 7 days
5. It judges checkpoint-gate false blocks, moved in from the
   checkpoint-block-attribution recipe, and that recipe is then retired

## Context

- As of 2026-11-22: `ele events --kind hook-block` returns each Stop-hook
  block, attachment and user-text forms alike, with `reason`, `changed`, and
  from internal/index `followBlock` the `reply`, `checkpointed` and `disputed`
  attributes; 1,572 blocks indexed, all on Stop
- As of 2026-11-22: `ele stop-check --replay --since <t>` prints one JSON line
  per Stop with `asked`, `asks`, `trigger` and `whyNot`; 852 Stops since
  2026-11-15, 86 of them asking
- As of 2026-11-22: `ele checkpoint stats [--since]` aggregates
  `logs/checkpoints.jsonl`, 1,261 checkpoints in the 30 days to that date
- As of 2026-11-22: `ele mine --report` reads the mining cache, last mined to
  a 2026-10-16 watermark; `/ele:mine` is typed only and costs model calls
- As of 2026-11-22: the replies to a notes ask are not an event; no ele
  command reads `~/.claude/logs/hooks-*.log`, whose only ele-written rows are
  the checkpoint gate's (internal/checkpoint `HookLog`); no flag exists
- As of 2026-11-22: the checkpoint-block-attribution recipe lives in
  `~/.agents/skills/`, linked from `~/.claude/skills/`. It pulls checkpoint
  blocks from `ele events`, but reads each block's turn (the last reply before
  it, the turn's edits and Bash commands) from the transcript files under
  `~/.claude*/projects/`, and runs the Bash commands through cmd/ele
  `commandIsWork` in a throwaway `_test.go`
- As of 2026-11-22: `ele show <session> --at <time>` prints a session's
  entries with tool calls as `tool  Bash: <command>`, `tool  Edit: <path>` and
  `tool  Write: <path>` lines, truncated, and `--all` prints them whole
- As of 2026-11-22: `.claude/` exists in the repository with no skills in it
- As of 2026-11-22: `/ele:plan-audit`'s Plan candidates section matches a
  candidate by its text starting `plan candidate` in any case, then optional
  spaces and `:`, `(` or `,`, read from `ele context`

## Decisions

- A repository skill under `.claude/skills/`, not a shipped `/ele:` skill or
  a private recipe: it proposes fixes to ele into this repository's notes,
  where a user's project has no use for them, and contributors can run it
  from the checkout
- cmd/ele `commandIsWork` classifies a block's Bash commands as a throwaway
  `_test.go` in the checkout, removed afterwards: it reads no record, only
  runs the checkout's own code over commands `ele show` printed
  - rejected: an ele command exposing the classification, because it adds a
    command and its docs for one skill's use
- Mining is reported with its watermark's date, never run by the audit:
  `/ele:mine` is typed only and costs model calls
- Only the checkpoint gate's half of `/hook-stats` comes in, read from
  `ele events`; the guard-hook firings in `~/.claude/logs/hooks-*.log` are
  the maintainer's own hooks, not ele's
- Plan candidates only: eval cases have no proposing mechanism, and an
  ask-behaviour failure cannot be graded in the eval sandbox; a candidate can
  say an eval is needed

## Actions

- [ ] audit-skill: write `.claude/skills/self-audit/SKILL.md`, taking a window
      (default the last 7 days), reading Stop-hook blocks with replies and
      disputes, the notes-ask replay, checkpoint stats and the mining report
      with its watermark's age, and proposing each finding as a plan candidate
      through `/ele:remember`, skipping one an open candidate or an active
      plan's item or inbox already holds. It prints `ele version` with its
      report, and gives `ele stop-check --replay --since` the window's first
      day as a date, which that flag takes in place of `Nd`. A source that
      returns nothing or fails is named with its command's output, and the
      others still reported
  - done when a real run in the maintainer's session over the last 7 days
    prints a figure from each of the four sources, the blocks' replies and
    disputed count, and the mining age, and proposes candidates, the window
    widened until at least one is proposed and written; a second
    run over the same window, once the first run's candidates are written,
    proposes none of them again and names each as skipped; neither run
    writes a note but plan candidates; and
    `task format`, `task check FORCE=1` and `task audit` pass
- [ ] gate-blocks: add the checkpoint-gate false-block judging from the
      checkpoint-block-attribution recipe to the skill, its turns read through
      `ele show`, run the recipe and the skill over the same window, then
      delete the recipe from `~/.agents/skills/` and its `~/.claude/skills/`
      link
  - done when a real run sorts each checkpoint block with `checkpointed`
    false in the window into real or false by cause, at least one, the
    window widened until there is one, and each false block is proposed as a
    plan candidate or named as skipped as already held; the recipe sorts the
    same window the
    same way before it is deleted; the throwaway `_test.go` is gone from
    `git status` afterwards; neither recipe path exists; and `task format`,
    `task check FORCE=1` and `task audit` pass
- [ ] contributor-docs: say in CONTRIBUTING.md what the self-audit skill reads
      and proposes and when to run it
  - done when CONTRIBUTING.md names the skill, its window and its output, and
    `task format`, `task check FORCE=1` and `task audit` pass
- [ ] Fold each Decision through
      `/ele:plan-audit close .documentation/plans/self-audit.md`, then propose
      each out-of-scope line as a `decision` through `/ele:remember`, run
      `ele plan done .documentation/plans/self-audit.md`, and delete this plan

## Inbox

## Out of scope

- Shipping the self-audit as an `/ele:` skill: its plan candidates are about
  ele, of no use in a user's project (ruled 2026-11-22, from
  `/ele:plan new self-audit`)
- Eval-case proposals: no mechanism proposes them, and an ask failure cannot
  be graded in the sandbox (ruled 2026-11-22, from `/ele:plan new self-audit`)
- Flags and heads-ups as sources: no flag exists, and a heads-up is text with
  no record of what the receiver did (ruled 2026-11-22, from
  `/ele:plan new self-audit`)
- The guard-hook half of `/hook-stats`: the maintainer's own hooks, not ele's
  (ruled 2026-11-22, from `/ele:plan new self-audit`)
- Removing the checkpoint-block counts from `/hook-stats`: it is the
  maintainer's own tooling, whose checkpoint count also feeds its check
  against the hooks log (ruled 2026-11-22, from `/ele:plan new self-audit`)
- The skill lint over `.claude/skills/`: one maintainer-run file, and a
  second root would need a rule exempting it from docs rows (ruled
  2026-11-22, from `/ele:plan new self-audit`)
