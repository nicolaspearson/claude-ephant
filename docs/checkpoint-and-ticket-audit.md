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
reported, ele's Stop hook keeps the turn going once and points Claude at
`/ele:checkpoint`. It goes out the same way as the notes ask, which your
terminal shows as `Stop hook feedback:`, not as a hook error.
Mid-task work, an investigation or a turn waiting on you is not a checkpoint,
and Claude says so rather than running one.

The gate weighs only what the turn itself wrote, so another session editing
the same checkout does not block a turn that edited files. A turn that edited files is judged on
those files alone, and passes once a checkpoint has recorded them as they
stand. A turn that edited nothing is judged on everything that moved only when
it ran a Bash command that may write. The gate never weighs ele's own data
directory, even when `ELE_HOME` puts it inside the checkout. Offering a commit
counts as saying the work is done, unless the commit is offered for later, as
in "once the suite passes, you can commit". "Now done; two are left" reports
progress, not done work.

When you say a change needs no checkpoint, Claude exempts its paths for the
session with `ele checkpoint optout`. To turn the gate off entirely:

```sh
ele config set checkpoint_gate false
```

### What checkpoints found

Each run is logged to `logs/checkpoints.jsonl` in the
[data directory](data-and-privacy.md#storage), next to the ticket-audit ledger.
The copies of reviewed files stay under `~/.claude/cache/`, whatever
`CLAUDE_CONFIG_DIR` says.
`/ele:checkpoint-stats [YYYY-MM-DD|Nd]` reports from it: which reviewer found
what, how much was refuted, what the re-review and the checks caught, and how
long runs took. Use it to tune the skill, not to judge the code. Reviews of a merge
request or branch are logged there too and reported in a section of their own,
with how often you skipped what the reviewers found and how many anchored
comments each forge kept, moved or refused. A review logged again after a
later finding is counted once, from the later line.

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

Your rulings on its findings go in a ledger, `logs/ticket-audits.jsonl` in the
data directory.
A later audit of the same map raises a ruled finding again with your ruling
beside it, and says whether the ticket's text changed since. A plan item's
rulings are keyed on its repository's origin, so a worktree or another clone
of it sees them too; rulings recorded under the folder's name before then
still show, and move to the new key at the next record.

For ele plans, use [`/ele:plan-audit`](plans.md#auditing-plans), which runs the
same methods over plans.

### Ticket claims

`/ele:ticket-claims` is for a GitHub tracker that labels its parent tickets
`wayfinder:map` and its work tickets `wayfinder:task`; on a tracker without those
labels it finds nothing. Of the first 500 open tickets, or the ones you number,
it flags a map granting itself execution and a task naming a source path. It
warns when the list reaches 500, and names each numbered ticket that is not
open or could not be read, rather than leave it out unsaid. Each flag is a judgement for you
to confirm, not a defect.
