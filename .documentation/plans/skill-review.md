---
kind: plan
created: 2026-10-23
spec: .documentation/specs/checkpoint-and-ticket-audit.md
---

# Skill review

## Why

A writing-for-agents review of `plugin/skills/` on 2026-10-23 found meanings kept in several drifted copies, branch-only material sitting in always-read files, a Propose cap with no rule past it, and steps with no condition that ends them. Each copy is a place the next edit misses; fixing them now keeps one edit per behaviour.

## Context

- As of 2026-10-23: the error-retry rule and the shell and guard rows sit in four files, `checkpoint/SKILL.md` and `ticket-audit/SKILL.md` as tables, `reviewers/bugs.md` and `reviewers/conventions-prose.md` as prose; checkpoint holds all eight general rows, the rest five each, the reviewer files lacking git guard, slow command and stale Edit, ticket-audit slow command, oversized Read and stale Edit
- As of 2026-10-23: inbox triage is written twice, `plan/SKILL.md` pickup step 5 with 5 options and `plan-audit/SKILL.md` Inbox triage with 7; `blocked:` and `after:` syntax is in `plan/SKILL.md` and restated in plan-audit's Frontier
- As of 2026-10-23: mining's `reviewerFile` regex in `internal/transcript/events.go` matches `reviewers/([a-z-]+)\.md`, so a lowercase shared file under `reviewers/` would count as a reviewer
- As of 2026-10-23: the Propose ask is worded in `remember/SKILL.md`, `rememberReason` in `cmd/ele/stopcheck.go`, its test in `cmd/ele/stopcheck_test.go` and `.documentation/specs/stop-check.md`; `advance` does not count an AskUserQuestion while proposing as work, so several ask calls need no hook change beyond wording
- As of 2026-10-22: AskUserQuestion does not load in eval runs; remember evals grade the reply with an llm grader

## Decisions

- One shared `plugin/reference/errors.md` holds the retry rule and the general rows (zsh `===`, `Exit code 1`, unmatched glob, chained `cd`, git guard, slow command piped to tail/head, oversized Read, stale Edit): the rows are shell and guard errors every skill hits, and four copies had drifted. This overturns the spec's line keeping a table in each SKILL.md and reviewer file
- Checkpoint keeps its own rows (`not covered` from `check`, agent type not found, `command not found` in a §2 check) in a short local table: no other skill hits them
- No `Seen` column: the counts change nothing the agent does; the spec keeps the `ele events --during <skill> --kind tool-error` command that produces them
- Reviewers find the errors file through a brief field, `Errors: <absolute path>, read when a call fails`: they read only what the brief names
- Inbox triage lives only in plan-audit's Inbox triage section; plan pickup points to it, gaining its Finding and other-plan's-inbox options
- `blocked:` and `after:` syntax lives only in `plan/SKILL.md`; plan-audit's Frontier keeps how those lines become edges
- Checkpoint §7's body moves to `checkpoint/PLAN.md`, per the spec's rule that a case only some runs meet sits in a file beside SKILL.md
- Propose asks at most four Write/Skip questions per AskUserQuestion call and another call for more, so no note is dropped silently; the Stop hook keeps restating the rule, since a session can act on the hook's reason without loading the skill
- A wording-only item is done when a fixed-string grep finds each moved passage in one file, every pointer resolves to an existing file, and `ele plan check` and `task check` pass

## Actions

- [ ] shared-errors: Write `plugin/reference/errors.md` with the retry rule and the general rows; cut checkpoint's table to its own rows plus a pointer; replace ticket-audit's and both reviewer files' Errors sections with a pointer; add the Errors field to checkpoint §3's brief; reword the spec's Errors line
  - done when the wording-only checks pass, and one real checkpoint run's brief carries the Errors path
  - as of 2026-10-23: §8's launch paragraph in `checkpoint/SKILL.md` passes the re-reviewer the brief's Settled and Report only fields, and Observed when the spec reviewer ran or §7 wrote, but no Errors field; the Errors field joins them
  - as of 2026-10-23: the done-when's real run reads the installed plugin, so it needs a release or a local plugin install first
- [ ] plan-format-once: Point plan pickup step 5 at plan-audit's Inbox triage; cut plan-audit's Frontier to the edge mapping with a pointer to `plan/SKILL.md` for the syntax; split the item bullet in `plan/SKILL.md`'s format into sub-bullets, one per concept, same words
  - done when the wording-only checks pass, Inbox triage's Finding option no longer says `under §7`, and the format's item bullet has one sub-bullet per concept
  - as of 2026-10-23: Inbox triage's Finding option says "report it under §7", an audit method; reword it so it also reads right from plan pickup, such as "report it as contradicting that decision"
- [ ] checkpoint-plan-file: Move checkpoint §7's body to `checkpoint/PLAN.md`, leaving the plan listing and a pointer read when the change worked on an active plan's item; add `PLAN.md` to the spec's list of files beside SKILL.md
  - done when the wording-only checks pass
  - as of 2026-10-23: not measured how many runs reach §7's body; the checkpoint log holds no plan field. Runs in repositories with no active plan never do
- [ ] remember-batches: Propose asks four Write/Skip questions per AskUserQuestion call and another call for more; the same wording in `rememberReason`, its test and `stop-check.md`; a `remember-many` eval proposing five decisions
  - done when the `rememberReason` test has been seen red on the old wording, and `remember-many` passes through `task eval`, graded on a reply that proposes all five notes and writes none
- [ ] mine-done-whens: Add a done-when to `mine/SKILL.md` §1 (each analysis's list and `.err` read), §2 (every batch launched), §3 (`ele mine --judgments` prints what it stored, or no batch wrote output and every unit is reported unjudged) and §4 (every report field given per analysis)
  - done when each of `mine/SKILL.md`'s four numbered sections ends with a `Done when` line, and the wording-only checks pass
- [ ] step-exits: Each step of `handoff/SKILL.md` and `remember/SKILL.md` ends with a done-when, as checkpoint's and ticket-audit's do; `recall`'s sections are alternatives picked per query, not a sequence that ends
  - done when handoff's Write and remember's Write and Propose each end with a `Done when` line, and the wording-only checks pass
- [ ] Fold what this plan built, and its out-of-scope lines as `Not doing:`, into the spec, run `ele plan done .documentation/plans/skill-review.md`, and delete this plan

## Inbox

## Out of scope

- Dropping the recall fallback in checkpoint §2 and ticket-audit §0: a skill run inside a subagent gets no skill list, per the spec's cost section (ruled 2026-10-23, from the skill review)
- Merging the two `mine/classify-*.md` prompts: stored judgments are keyed by each prompt's hash (ruled 2026-10-23, from the skill review)
- Trimming the disambiguating clauses in the handoff and ticket-audit descriptions: each steers away from a neighbouring skill (ruled 2026-10-23, from the skill review)
