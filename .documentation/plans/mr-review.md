---
kind: plan
created: 2026-10-26
spec: .documentation/specs/review.md
---

# MR review

## Why

Review an MR, or the user's own branch before one exists, with checkpoint's
adversarial reviewers, keep only the major findings, and let the user publish
each as a comment worded as they would write it, or take its Markdown to post
themselves.

## Context

- As of 2026-10-26: checkpoint's reviewers are `plugin/skills/checkpoint/reviewers/bugs.md`, `conventions-prose.md` and `spec.md`; bugs runs as `general-purpose`, the other two as `ele:checkpoint-reviewer` on sonnet with Bash, Glob, Grep and Read. Only `bugs.md` scores findings; `conventions-prose.md` has no confidence threshold, and `spec.md` reports a requirement needing an unobserved observation as missing
- As of 2026-10-26: `ele checkpoint log` validates `scope` as one of `branch`, `delta`, `uncommitted`, `severity` as one of `blocker`, `defect`, `nit`, `quality`, and `verdict` as one of `deliberate`, `fixed`, `open`, `refuted`; it requires `mode`, `change`, `mutations`, `notes` and `rereview`, and per finding `category`, `source` and `summary` (`Problems` and `findingProblems` in `internal/checkpoint`)
- As of 2026-10-26: `Append` in `internal/checkpoint` adds `duration` from the session's open run and then removes that run's `.started` clock file, so a line logged mid-checkpoint takes and stops that run's clock
- As of 2026-10-26: `internal/config` declares one option kind, `KindBoolean`; the options are `auto_note_writes`, `checkpoint_gate`, `colour` and `reminders`
- As of 2026-10-26: the plugin's shared reference holds `plugin/reference/errors.md` and `review-voice.md`
- As of 2026-10-26, from the GitLab and GitHub API docs: a GitLab draft note takes a `position` as a discussion does, and is published one by id or all of the user's at once with `bulk_publish`; GitHub allows one pending review per user per PR, its REST API cannot add a comment to a pending review, and a review's `comments[]` take no `subject_type`

## Decisions

- A separate `ele:review` skill, not a checkpoint mode, decided 2026-10-26 by the user: checkpoint fixes, commits and records; review only reports
- Targets are an MR or PR through glab or gh, a local branch, and a commit, decided 2026-10-26 by the user
- Reviewers read an archive of the head tree extracted into the scratchpad, from git when the SHA is local, else the forge, decided 2026-10-26 by the user: the bugs reviewer greps for callers, which a forge's file API cannot
- A local branch with uncommitted work is reviewed, the work left out and said so, decided 2026-10-26 by the user: the archive of HEAD already shows what an MR would
- A closed or merged MR is refused unless the user says otherwise; a draft is reviewed; state and head are re-read before publishing, decided 2026-10-26 by the user
- The spec is the issue the MR closes, or an explicit `spec:<ref>`, which wins; other trackers' keys are not fetched, decided 2026-10-26 by the user
- Bugs and spec reviewers as checkpoint runs them; conventions runs its Conventions pass only, without Shape, on the repository's convention files from the base SHA, the user's own `CLAUDE.md` left out, decided 2026-10-26 by the user
- The brief's Observed field is the head pipeline; the parent turns a spec finding missing only for want of an observation into a question, decided 2026-10-26 by the user
- Findings are filtered by the bugs reviewer's confidence and the published exclusions, deduplicated against the MR's discussions, classified, and only blockers and defects verified and shown, decided 2026-10-26 by the user
- Severity takes the log's names with review's own, wider definitions, stats reporting review runs apart, decided 2026-10-26 by the user
- A breach of a rule the repository states as required, the rule quoted, is a defect, decided 2026-10-26 by the user
- A table first, then one finding at a time: Add, Raw Markdown, Learn more, Skip, with Other as a redraft, each body shown verbatim, decided 2026-10-26 by the user
- Added findings are held by the skill and sent together on one final approval, with no approval state, decided 2026-10-26 by the user: nothing unapproved reaches the author, and on GitHub, and on GitLab when the user holds no other drafts there, it arrives as one review; GitHub's REST API cannot grow a pending review, so nothing waits on the forge
- A body opens with a Conventional Comments label, the kind as a decoration, decided 2026-10-26 by the user, reversing an earlier no-label decision
- A suggestion block only where applying it resolves the finding whole, decided 2026-10-26 by the user
- MR text is stripped of hidden markup, reviewers run as `ele:review-reader` with Read, Grep and Glob only, nobody runs the tree's code, and every published word is approved, decided 2026-10-26 by the user: the MR is its author's input to an agent holding the user's credentials
- The parent drafts every body from a voice file; reviewers never see it, decided 2026-10-26 by the user, so the reviewer files stay shared unchanged
- The voice file holds general rules with neutral examples and names no client, employer or person, decided 2026-10-26 by the user: the repository is public
- `review_voice`, a path option settable per owner, replaces the shipped voice file, decided 2026-10-26 by the user
- Review runs log through `ele checkpoint log` with their own scopes, filter counts and a per-finding outcome, touching no run state, decided 2026-10-26 by the user
- The skill is user-invoked only, decided 2026-10-26 by the user: it posts under the user's name
- No `claude -p` shell-out, decided 2026-10-26 by the user: subagents give the same separation
- This plan adds the rule that a test is never weakened to make it pass: once bugs-test-weakening lands, `bugs.md` reports one and checkpoint's Verify and Gate sections forbid it, decided 2026-10-26 by the user: review and checkpoint both read `bugs.md`, and the session making the gate green is the likeliest to loosen a test

## Actions

- [x] review-voice: Write `plugin/reference/review-voice.md` from the spec's Voice rules, each rule with a neutral Write and Not example, the Conventional Comments labels and decorations the spec names included
  - done when the file holds every rule the spec's Voice section lists, it names no client, employer or person, and `task check FORCE=1` passes
  - as of 2026-10-26: the spec's Voice section lists nine rules; the labels it names are in Walk-through, `issue (blocking, <kind>):`, `issue (non-blocking, <kind>):` and `question:`, the kinds bug, convention, security, spec and test; `plugin/reference/` held only `errors.md` before this item
  - asserted in the review-voice commit: `plugin/reference/review-voice.md` holds the spec's nine Voice rules, each with a Write and Not example, and the Walk-through's labels and kinds, mapped one-to-one by the spec reviewer; a grep for client, employer and person names finds none; `task check FORCE=1` and `task audit` pass
- [ ] review-voice-option: Add `review_voice`, a path option, default empty, settable per owner, the first non-boolean option kind
  - done when `ele config set review_voice <path>` and `--owner` round-trip through `ele config get`, a relative path is refused, a `config.json` holding a non-string value reports the option's kind, a test seen red covers each, `ele config list` describes it, and design.md's Configuration section names it
- [ ] review-log: Let `ele checkpoint log` take the spec's review record: scopes `mr` and `local-branch`, `target`, `reviewed` and `dropped`, no `mutations` or `rereview` on them, the verdict `confirmed`, a per-finding `outcome` on confirmed and open findings, and neither a `duration` nor a change to run state; `ele checkpoint stats` reports review scopes apart
  - done when a review record validates and a checkpoint record missing `mutations` still fails; `outcome`, `confirmed` and `dropped` are refused on a checkpoint scope and `outcome` on a refuted finding; a review logged during an open checkpoint run carries no `duration` and leaves that run's clock, the test going red when the review-scope guard in `Append` is removed; stats print review runs in their own section with the skip rate and the dropped counts; each tested and seen red; and the checkpoint-and-ticket-audit spec's Log line names the new fields
- [ ] review-skill: Write `plugin/skills/review/SKILL.md` to the spec, with the `ele:review-reader` agent in `plugin/agents/review-reader.md`
  - blocked: review-voice-option, review-log
  - done when, observed by hand with the clone's build installed (`task install`), `/ele:review` on a throwaway MR of the user's on each forge, GitLab and GitHub, shows the table, sends nothing until the final question, then publishes one added finding at its line with its label; prints another as Raw Markdown with a permalink; sends nothing when publishing is declined; stops on a pending review the user already holds on GitHub; on a local branch with no MR offers no Add; and the run's log line validates. `SKILL.md` states the blocker, defect and question labels as its own rule, so a `review_voice` file that leaves them out cannot drop them. Not observed by hand: the head-moved and closed-at-publish paths, left to the eval or a later run. `docs/review.md` and `docs/commands.md` describe it, each command checked against its `--help`
- [ ] review-eval: An eval on a local-branch fixture with one planted defect: the table names it, and nothing reaches a forge without an Add and the final publish
  - blocked: review-skill
  - done when the eval goes red with the planted defect removed and green with it in place; run only when the user asks, since `task eval` costs model calls
- [ ] bugs-test-weakening: Add to `reviewers/bugs.md` that a test changed to pass, rather than to match changed behaviour, is a finding: a weakened assertion, a widened tolerance, a skipped case; and to checkpoint's `SKILL.md` Verify and Gate sections that a red test is fixed in the code, and a test changes only for a behaviour change this change intends, which the report names
  - done when, in real sessions, a checkpoint over a change that loosens an assertion to make it pass reports it under the Bugs reviewer, and a checkpoint whose gate goes red on a test the change did not set out to change fixes the code and leaves the test unchanged; and the checkpoint-and-ticket-audit spec's Checkpoint section states both rules
- [ ] Fold what this plan built into `.documentation/specs/review.md`, run `ele plan done .documentation/plans/mr-review.md`, and delete this plan

## Inbox

## Out of scope

- A review of uncommitted work: checkpoint's
- A summary comment: the findings are the review
- Approving or requesting changes: the published review carries no state, so the user sets it on the forge
- Fetching specs from trackers other than the forge: the user passes a path or URL
- Running the tree's code to reproduce a finding: the tree is the MR author's input
