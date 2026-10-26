# Checkpoint and ticket-audit

## Checkpoint

The session that wrote a change knows what every line was meant to do, so it
reads its own diff as confirmation. `/ele:checkpoint` hands the change to
reviewers in subagents that have seen only a brief, before the work is reported
done or committed.

Ask for one, or let Claude run it before saying the work is done. A run:

1. **Scopes** the change: uncommitted work by default, only what changed since
   the last checkpoint when one reviewed part of it, or the whole branch when you
   ask
2. **Reviews** it. A small change to tracked files in one repository, up to 40
   changed lines or 100 in Markdown or text, gets one reviewer. Any other change
   gets a bugs reviewer and a conventions
   and prose reviewer in parallel. When the brief quotes a plan item or ticket, a
   spec reviewer checks the change against what it asked for
3. **Verifies** each finding against the code before fixing it, and breaks the
   code each new or changed test covers to watch the test fail
4. **Runs the project's checks**, the ones its CI runs, again after the fixes. A
   red test is fixed in the code, never changed to pass; what a test asserts
   changes only for a behaviour change the work sets out to make, and the report
   names it
5. **Re-reviews** the fixes, and proposes a plan tick or progress line
6. **Reports** each reviewer's findings, fixed, refuted or left, and prints the
   commit commands for you to run. It never commits

The conventions reviewer reads your `CLAUDE.md`, under `$CLAUDE_CONFIG_DIR` or
`~/.claude`, and the repository's.

### The Stop hook gate

When a turn changed files and claims the work is done, but no checkpoint was
reported, ele's Stop hook blocks the turn once and asks for `/ele:checkpoint`.
Mid-task work, an investigation or a turn waiting on you is not a checkpoint,
and Claude says so rather than running one.

When you say a change needs no checkpoint, Claude exempts its paths for the
session with `ele checkpoint optout`. To turn the gate off entirely:

```sh
ele config set checkpoint_gate false
```

### What checkpoints found

Each run is logged to `~/.claude/logs/checkpoints.jsonl`, under `~/.claude`
whatever `CLAUDE_CONFIG_DIR` says, as are the ticket-audit ledger and the copies
of reviewed files.
`/ele:checkpoint-stats [days]` reports from it: which reviewer found what, how
much was refuted, what the re-review and the checks caught, and how long runs
took. Use it to tune the skill, not to judge the code. Reviews of a merge
request or branch are logged there too and reported in a section of their own,
with how often you skipped what the reviewers found.

## Ticket-audit

`/ele:ticket-audit` audits a tracker's ticket map, a parent ticket and its
children on GitHub or GitLab, against the code and against itself:

- premises the code no longer bears out
- tickets that contradict each other
- something one ticket produces that nothing reads, or reads that nothing
  produces
- blockers on closed or missing tickets, and cycles
- decisions recorded in two places, or in a comment while the body still asks
- status claims the open tickets contradict, and a map granting itself licence
  to act

It needs `gh` or `glab`, logged in, and `jq` for GitLab. Run it before picking
up a ticket or closing a map. It drafts every tracker change and hands it to you
to run: nothing is written to the tracker by the audit itself.

Your rulings on its findings go in a ledger, `~/.claude/logs/ticket-audits.jsonl`.
A later audit of the same map raises a ruled finding again with your ruling
beside it, and says whether the ticket's text changed since.

For ele plans, use [`/ele:plan-audit`](plans.md#auditing-plans), which runs the
same methods over plans.

### Ticket claims

`/ele:ticket-claims` is for a GitHub tracker that labels its parent tickets
`wayfinder:map` and its work tickets `wayfinder:task`; on a tracker without those
labels it finds nothing. Of the first 200 open tickets, it flags a map granting
itself execution and a task naming a source path. Each flag is a judgement for
you to confirm, not a defect.
