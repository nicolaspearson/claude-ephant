# Drafts mode

With `drafts` and paths: the paths are unstarted drafts, audited for a review
loop. It reports, asking the user nothing and changing nothing. In scope are
the drafts and every active plan of the repository, listed by `ele plan list`
run at its root.

Only these run:

- `SKILL.md`'s Which plans, for each plan's `ele plan check` warnings
- Ticket-audit's §1 to §8, as `SKILL.md`'s Ticket-audit's sections, for plans
  adapts them, and with these:
  - §1: read each draft's `## Requirements` with its Context and Decisions. A
    repository the caller does not mark ours or read-only, and current, is a
    question
  - §2: launch the subagents with no scope to approve, the brief also
    carrying this file's absolute path, which a subagent reads after
    `SKILL.md`
  - §3 over every open action of the drafts, the closing one included; §4 to
    §8 across every plan in scope, so a draft contradicting an active plan is
    found
  - Each finding takes a level from the Severity section of
    `reference/plan-practices.md` in the plugin root, the directory holding
    `skills/`, and the plan path and line it rests on, in each plan for a
    contradiction between two
- The report below

A question §1 or Questions and rulings would ask, after its recall and grep,
goes in the report, as does a subagent's: the caller puts it to the user.

The report: ticket-audit's §10 report without its record, plus each plan's
`ele plan check` warnings, the findings blockers first, each with its level,
and the open questions.

Done when every finding carries a level and a line, every question is in the
report, and no plan file or ledger was written.
