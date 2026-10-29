---
kind: plan
created: 2026-10-29
---

# Eval graders off LLM judges

## Why

An LLM judge's verdict is not stable: in setup-maps a Haiku judge passed a reply on 3 of 3 votes, then failed a reply meeting every criterion on 2 of 3, and the grader was replaced by a `tool_used` regex. The suite's other `type: llm` graders carry the same risk, most of them judging several claims at once. A grader should be deterministic wherever the trace or the reply allows, and judged only on one claim where neither does, and the rule should be written down so new evals follow it.

## Requirements

1. Convert each `type: llm` grader under `plugin/evals` that checks a fact the trace holds to `tool_used`, and one that checks a literal the reply must or must not hold to `regex`
2. Narrow the rest to one claim each
3. Write the rule into CONTRIBUTING's eval section

## Context

- As of 2026-10-29: 23 `type: llm` graders in 14 of the 17 evals under `plugin/evals`, 10 of them in recipe-audit; `plan-new-review` and `setup-maps` hold none, `recall` none
- As of 2026-10-29: most llm-graded cases already have a `tool_used` grader for the command or write their llm grader names (`ele flag bad`, `ele remember` or its `max: 0`, `ele handoff list --all`, the `SKILL.md` Write, `ln -s`), and the llm grader mostly judges claims about the reply: reported as saved, asks write or skip, does not claim it was written, names a finding. Not all: recipe-writes' grader says `recipe_dir` was unset, a read no grader checks, and `wrap-deferred/open-work` and `wrap-recipe-off/no-recipe` name no command
- As of 2026-10-29: 15 of the 17 cases set `runs: 1`; `wrap-asks` and `wrap-deferred` set `runs: 3`
- As of 2026-10-29, from plan-workflow's new-eval probe on claude 2.1.308: AskUserQuestion does not load in the eval sandbox, so an ask is a claim in the reply, never a tool call
- As of 2026-10-29, from the `claude plugin eval` docs: `regex` takes `pattern`, `flags`, `match` (`not_contains`, `count:N`) and `target` (`last_message` by default, `trace`, `files`, a workspace file); an `llm` grader passes on 2 of 3 judge votes, Haiku by default
- As of 2026-10-29: CONTRIBUTING's Evals section says how a case is laid out and run, and nothing about which grader type to use; no spec under `.documentation/specs/` describes evals, so this plan has none
- As of 2026-10-29: ele-followups' `grader-alternation` edits recipe-audit's `no-run-step`, `no-run-script` and `no-run-runner` graders, none of them `type: llm`; the two plans touch different files in that case

## Decisions

- A clause of an llm grader that a sibling `tool_used` grader already asserts is deleted from the llm grader, and that counts as converting it: the trace fact is graded once, by the deterministic grader
- A literal the reply must or must not hold, such as a path or a command name, is graded by `regex` on `last_message`, not by a judge
  - rejected: keeping such claims as `llm`, because a literal in the reply needs no judge
- Narrowing keeps one `llm` grader per claim the case exists to test, the behaviour its prompt is built around, each in its own named file; a secondary claim, one a deterministic grader already makes unlikely to fail, such as "does not claim it was saved" beside a `max: 0` write grader, is dropped and listed under its item with the reason. `task eval` runs at the default `--threshold` of 1.0, so one failed grader in one run fails the case, and every judge grader is a vote that can flip
  - rejected: splitting out every claim, because each extra judge grader is another vote that can fail the case and adds 3 judge calls a run
  - rejected: keeping only each grader's main claim with no review of the rest, because it drops checks the case was written for unexamined
  - rejected: lowering `--threshold`, because it hides a real failure along with a flaky one
- Each new `tool_used` or `regex` pattern is hand-probed in node, over a JSON-encoded tool input or reply text, matching the passing shape and seen red on a failing one; each grader item also runs `task eval -- ./plugin --case <case> --runs 3` once per case it changes, started only on the user's go, since it costs model calls; `--runs 3` because most cases run once, and a split verdict shows only across runs. With the go withheld, the item stays open with a dated `next:` line
- The rule is written before the conversions, so each conversion follows written text

## Actions

- [ ] grader-rule: CONTRIBUTING's Evals section states which grader type checks what
  - done when the section says: a fact the trace holds, a command run, a file written or a tool called, is graded with `tool_used`; a literal the reply must or must not hold with `regex` on `last_message`; an `llm` grader judges one claim neither can check, and only one the case exists to test, each with its reason; and `task format`, `task check FORCE=1` and `task audit` pass
- [ ] recipe-audit-graders: recipe-audit's 10 `type: llm` graders follow the rule
  - blocked: grader-rule; the conversions follow the rule as written
  - as of 2026-10-29: the eval run also scores `no-run-step`, `no-run-script` and `no-run-runner`, which ele-followups/grader-alternation finds matching a quoted `grep -E` alternation as a run; such a failure is traced to that grader, not this item's
  - done when each of `reports-audit-mode`, `reports-findings`, `reports-no-false`, `reports-not-checked`, `reports-partial-help`, `reports-path`, `reports-runners`, `reports-script`, `reports-script-holds` and `reports-stale-flag` is a `tool_used` or `regex` grader or one-claim `llm` graders, no remaining `llm` grader judging a claim a `tool_used` or `regex` grader could check, or one the case does not exist to test, per the rule grader-rule wrote; each dropped claim listed under this item with its reason, each new pattern seen red by hand probe; `task format`, `task check FORCE=1` and `task audit` pass; and recipe-audit's eval run, as the Decisions set it, on the user's go, has its per-grader results written under this item, naming any grader whose verdict split across a case's runs; each failed or split verdict traced to the reply or the grader, a grader at fault fixed and its case run again, a reply at fault written down as the model's failure with its reason
- [ ] remember-graders: the `type: llm` graders of flag-saves, handoff-list, remember-auto-off, remember-auto-on, remember-many and remember-reminder follow the rule
  - blocked: grader-rule; the conversions follow the rule as written
  - done when each of `flag-saves/reports`, `handoff-list/shows`, `remember-auto-off/asks`, `remember-auto-on/reports`, `remember-many/asks` and `remember-reminder/reports` is a `tool_used` or `regex` grader or one-claim `llm` graders, no remaining `llm` grader judging a claim a `tool_used` or `regex` grader could check, or one the case does not exist to test, per the rule grader-rule wrote; each dropped claim listed under this item with its reason, each new pattern seen red by hand probe; `task format`, `task check FORCE=1` and `task audit` pass; and each of those six cases' eval runs, as the Decisions set them, on the user's go, has its per-grader results written under this item, naming any grader whose verdict split across a case's runs; each failed or split verdict traced to the reply or the grader, a grader at fault fixed and its case run again, a reply at fault written down as the model's failure with its reason
- [ ] recipe-wrap-graders: the `type: llm` graders of recipe-asks, recipe-writes, recipe-writes-dir, wrap-asks, wrap-deferred, wrap-recipe and wrap-recipe-off follow the rule
  - blocked: grader-rule; the conversions follow the rule as written
  - done when each of `recipe-asks/asks`, `recipe-writes/reports`, `recipe-writes-dir/reports`, `wrap-asks/asks`, `wrap-deferred/open-work`, `wrap-recipe/proposes` and `wrap-recipe-off/no-recipe` is a `tool_used` or `regex` grader or one-claim `llm` graders, no remaining `llm` grader judging a claim a `tool_used` or `regex` grader could check, or one the case does not exist to test, per the rule grader-rule wrote; each dropped claim listed under this item with its reason, each new pattern seen red by hand probe; `task format`, `task check FORCE=1` and `task audit` pass; and each of those seven cases' eval runs, as the Decisions set them, on the user's go, has its per-grader results written under this item, naming any grader whose verdict split across a case's runs; each failed or split verdict traced to the reply or the grader, a grader at fault fixed and its case run again, a reply at fault written down as the model's failure with its reason
- [ ] Propose each out-of-scope line as a `decision` through `/ele:remember`, run `ele plan done .documentation/plans/eval-graders-off-judges.md`, and delete this plan

## Inbox

## Out of scope
