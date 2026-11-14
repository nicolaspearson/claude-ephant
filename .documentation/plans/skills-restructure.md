---
kind: plan
created: 2026-11-13
---

# Skills restructure

## Why

A writing-for-agents audit of `plugin/` on 2026-11-13 found the ele skills
carrying material only some runs reach in their main files, rules written in
several places, and wording that steers less well than it could. Its wrong-now
findings were fixed in cfbc9dd or refuted; this plan works the findings
Context lists, re-derived against the code before drafting, so each skill
loads what its run needs and each rule changes in one place.

## Requirements

1. Each section Context names as reached only on one branch moves to a
   reference loaded on that branch
2. Each copy Context lists lives in one place and the others point at it
3. Each wording finding Context lists as of 2026-11-13 is fixed
4. No SKILL.md crosses the 500-line lint

## Context

- As of 2026-11-13: SKILL.md line counts: checkpoint 499, review 483,
  ticket-audit 370, plan-audit 237, recipe 95, plan 87; `internal/skilllint`
  fails a SKILL.md of 500 lines or more, allows only `assets`, `references`,
  `reviewers` and `scripts` beside it, and checks that a code span
  `references/<name>.md` names a file in the skill. A file shared by skills
  sits in the plugin root's `reference/` and is named as
  "`reference/<name>.md` in the plugin root"
- As of 2026-11-13, sections reached only on one branch: plan-audit's Drafts
  mode (29 lines) and Close mode (35), each run alone, routed from Which
  plans; recipe's Audit (33 lines), run only with `audit`; review's step 3
  Perl strip (about 30 lines with its explanation), run only on text fetched
  from a forge; checkpoint's outside-git rules, in §1, §3, §4 and §8; and §1's
  Go-linter notes, which only Go repositories need. review fetches forge text
  on an MR run and for a `spec:` forge issue on any run
- As of 2026-11-13, copies: checkpoint's held tick defined in §7 and
  `references/PLAN.md` and restated in §8, §10 and §11; the red-test rule in
  §5 and restated in §6; the same Errors section in `reviewers/bugs.md` and
  `conventions-prose.md` and none in `spec.md`, while checkpoint §3's and
  review step 4's briefs each carry an Errors path; the intro's exceptions to
  "Every review runs fresh", stated again in §1 and `references/DELTA.md`;
  `reference/explorer.md`'s Commands repeating `agents/explorer.md`'s four tool
  rules; plan-audit restating ticket-audit §1's `mktemp -d` and the
  no-ele-data rule plan's format states; ticket-audit §1 restating `ele ticket
  corpus --help` and its "What this is for" repeating its description; plan's
  `blocked:` bullet copying parser edge cases `ele plan check` warns on;
  review's four-questions-per-call AskUserQuestion limit stated in
  steps 8, 9 and 10
- As of 2026-11-13, wording: checkpoint's description says "done or ready to
  commit" twice; flag's lists three phrases for one branch; setup's names two
  signals for one; recipe's describes audit twice and, with wrap's, ends
  "never start one unasked"; recall has no done-when and checkpoint's optout
  none; checkpoint §8's re-reviewer brief and wrap §2's handoff condition are
  each one long sentence; checkpoint's `references/LOG.md` says "failed for
  the reason stated" and "did not bite" where §5 says red, plan's Asking in
  rounds spells out a frontier, ticket-audit spells out a premise carrying its
  `file:line` in several sections; `reference/architect.md`'s Rules open with
  prohibitions
- As of 2026-11-13: `ele checkpoint freeze` reports only a count of dirty
  paths past 200 left unfingerprinted, naming none (`internal/checkpoint`
  `Freeze`'s summary); no skill handles that output, and §8's fingerprint
  comparison cannot see an edit to such a path. `docs/commands.md` describes
  the count; no other `docs/` page covers the cap
- As of 2026-11-13: the evals grade tool calls, not skill text; a moved
  section can add a Read inside a case's turn budget, `plan-new-review`'s
  being 150 and `recipe-audit`'s 25. No eval covers checkpoint, review or
  plan-audit text
- As of 2026-11-13: review's preamble rests on a Claude Code claim dated
  2026-10-26, that text printed between calls does not reliably reach the
  user; not re-observed. Regrouping its AskUserQuestion rules does not change
  what they require
- As of 2026-11-13: other active plans edit the same skills:
  `cross-repo-blockers`' `foreign-blockers` edits plan and plan-audit text,
  and `candidates-in-start-block`'s `count-line` edits plan-audit's Plan
  candidates section

## Decisions

- Actions run disclosure, the freeze cap, single source, then wording:
  `checkpoint/SKILL.md` has no line to spare, the moves and single-source
  cuts free lines, and the freeze cap and wording add them
- No blocker on `foreign-blockers` or `count-line`: whichever lands second
  rebases onto the other's text
- A pointer from one skill into another's moved section names the skill and
  the section in prose, "plan-audit's Close mode", never a `references/` code
  span, since `internal/skilllint` checks such a span against the pointing
  skill's own directory
- The reviewers' Errors line, "report an error it does not clear under
  Questions", goes into both briefs that launch them, checkpoint §3's and
  review step 4's, since review reads the reviewer files unchanged
- A moved section leaves a pointer that names its trigger, as checkpoint's
  `references/BRANCH.md` and `DELTA.md` pointers do; a heading another file
  names keeps its name or each naming file is updated in the same change
- A freeze reporting unfingerprinted paths is handled as a change with a file
  outside git: it is full mode and §8 always re-reviews, since the fingerprint
  cannot see their edits and freeze names none of them
- Each action updates the spec and `docs/` page that describe the text or
  behaviour it changes, in the same change, as the repository's `CLAUDE.md`
  requires, and passes `task format`, `task check FORCE=1` and `task audit`

## Actions

- [ ] skill-disclose: Move each one-branch section Context names to a
  reference loaded on its branch: plan-audit's Drafts and Close modes to
  `references/DRAFTS.md` and `references/CLOSE.md`, Drafts' exceptions phrased
  as what it runs; recipe's Audit to `references/AUDIT.md`; review's Perl strip
  to a reference read whenever text is fetched from the forge, on an MR run or
  for a `spec:` forge issue; checkpoint's outside-git rules to a reference §1
  points at, §4's mode rule and §8's skip condition staying inline and naming
  it, and §1's Go-linter notes to one read for a Go repository
  - done when:
    - each moved section is grepped absent from its main file and present in
      its reference
    - each pointer resolves under `internal/skilllint` and names its trigger,
      review's naming both forge cases
    - every file naming a moved heading names it where it now lives, and
      `.documentation/specs/checkpoint-and-ticket-audit.md`'s list of skill
      references names each new one
    - the added Reads leave `recipe-audit` within its 25 turns and
      `plan-new-review` within its 150, or the case's budget is raised in the
      same change
    - every SKILL.md is under 500 lines, the specs and `docs/` pages
      describing the moved text match it, and `task format`, `task check
      FORCE=1` and `task audit` pass
- [ ] freeze-cap: Checkpoint handles a freeze reporting dirty paths past 200
  left unfingerprinted as a change with a file outside git: full mode, and §8
  always re-reviews
  - done when §1's pointer to the outside-git reference names a capped freeze
    as a trigger, the reference and §4 and §8 cover it,
    `.documentation/specs/checkpoint-and-ticket-audit.md` says how a capped
    freeze is handled, `docs/checkpoint-and-ticket-audit.md`'s mode rule says
    a capped freeze gets full review, `checkpoint/SKILL.md` is under 500
    lines, and `task format`, `task check FORCE=1` and `task audit` pass
- [ ] skill-single-source: Keep each copy Context lists once: checkpoint's
  held tick in `references/PLAN.md`, §7, §8, §10 and §11 naming it; the
  red-test rule in §5, §6 citing it; the reviewers' Errors line in both
  briefs, cut from `bugs.md` and `conventions-prose.md`, and
  `conventions-prose.md`'s Output reworded to match; the intro's exceptions
  cut where §1 and `references/DELTA.md` state them; `reference/explorer.md`'s
  Commands keeping only what `agents/explorer.md` lacks; plan-audit citing
  ticket-audit §1 for `mktemp -d` and plan's format for the no-ele-data rule;
  ticket-audit §1 keeping only what `ele ticket corpus --help` lacks, and its
  "What this is for" cut, the `file:line` rule kept in §3; plan's `blocked:`
  bullet keeping the syntax and leaving parser edge cases to `ele plan
  check`'s warnings; review's four-per-call limit stated once, under a
  heading steps 8, 9 and 10 cite
  - done when each removed copy is grepped absent, each kept rule and its
    pointers grepped present, each pointer names a heading that exists, every
    SKILL.md is under 500 lines, the specs and `docs/` pages quoting the text
    match it, and `task format`, `task check FORCE=1` and `task audit` pass
- [ ] skill-wording: Fix each wording finding Context lists: checkpoint's,
  flag's, setup's, recipe's and wrap's descriptions, one trigger per branch,
  recipe and wrap ending on "Use only when the user asks"; a done-when for
  recall and for checkpoint's optout; checkpoint §8's re-reviewer brief and
  wrap §2's handoff condition as lists; red in checkpoint's
  `references/LOG.md`, frontier in plan's Asking in rounds, opened in
  ticket-audit for a premise with its `file:line`; `reference/architect.md`'s
  Rules leading with the positive target
  - done when each finding's old wording is grepped absent and its new text
    grepped present, including a done-when line in recall and under
    checkpoint's optout, every SKILL.md is under 500 lines, the specs and
    `docs/` pages quoting the text match it, and `task format`, `task check
    FORCE=1` and `task audit` pass
- [ ] Fold each Decision through `/ele:plan-audit close
  .documentation/plans/skills-restructure.md`, offer the user a `task eval`
  run of `recipe-audit` and `plan-new-review`, propose each out-of-scope line
  as a `decision` through `/ele:remember`, run `ele plan done
  .documentation/plans/skills-restructure.md`, and delete this plan

## Inbox

- 2026-11-14, from a plan audit of plan candidates, first raised 2026-10-23
  in a session's feedback: batch pickup. `/ele:plan pickup` works one item a
  session; propose instead every startable item, plus inbox items touching the
  same files, that would ship as one commit, with one checkpoint over the batch
  and a tick line per item. Also: plan-only edits mid-session ride along with
  the next code commit instead of their own checkpoint, which needs the Stop
  gate or checkpoint's plan-only path to allow it
- 2026-11-14, from a plan audit of plan candidates, first asked 2026-10-26 in
  a by-hand local review run: a review fix step. The user asked to "fix it
  now" or "fix all" on a finding. Not in the walk-through, since an edit moves
  later findings' lines and the review reads a frozen copy; instead a step
  after it, on a local run or an MR whose head is the checked-out branch, that
  applies the findings the user picked, then runs ele:checkpoint. It cuts
  across the decision that review only reports and checkpoint fixes. Open:
  should review hand the picked findings to checkpoint?

## Out of scope

- Moving plan's Asking in rounds, Grounding, Designs and Review loop to a
  reference: about 24 of 87 lines, and Pickup uses Grounding (ruled
  2026-11-13, from this plan's premise check)
- Moving review's step 13 log schema: every review past step 2 logs (ruled
  2026-11-13, from this plan's premise check)
- Moving checkpoint §1's "The run": it applies to every run (ruled
  2026-11-13, from this plan's premise check)
- Splitting `plan-practices.md`, its unread Sources and "Severity scales seen"
  included: it is no skill file, the plan reviewer reads its adopted practices
  across most sections, and the ledger's read date would break the
  repository's `as of` rule in a spec (ruled 2026-11-13, from this plan's
  review loop)
- `commands/checkpoint-stats.md`'s Duration and Reviews: `ele checkpoint stats
  --help` does not carry what they say, so they are no copy of it (ruled
  2026-11-13, from this plan's premise check)
- remember's opening paragraph: it also carries the approval rule (ruled
  2026-11-13, from this plan's premise check)
- Generating mine's two classifier prompts from one template: each prompt's
  bytes are its classifier hash; worth doing only when a schema next changes
  (ruled 2026-11-13, from the audit)
- One home for the ask-the-user protocol in remember's Propose: wrap and
  recipe each keep rules Propose lacks (recipe always asks and skips on the
  user's exact words; wrap auto-writes a handoff and offers Close/Keep for
  reminders), what they share is three short rules, and `recipe-asks` and
  `wrap-asks` allow no Read, so citing Propose costs a Skill load inside a
  15-turn eval (ruled 2026-11-13, from this plan's checkpoint)
