# Handoffs and reminders

## Handoffs

A long session pays for its whole context on every turn, so the natural stopping
point is the time to start fresh. A handoff is a note of up to three lines that
the next session in the project opens with. Its first line points at the next
task: a plan item, as `Next: <plan path> <item name>`, or an issue or merge
request, as `Next: <url>`; otherwise `Next:` and the task in words. The rest says where the
work stands.

1. Ask for one with `/ele:handoff`, or say yes when `/ele:wrap` proposes one
2. Run `/clear`
3. Type `continue`

The handoff shows only at startup or after `/clear`, never on resume or after a
compaction, so it is not used up by a session that did not start the work
afresh.

- A lone pending handoff is marked shown when the session opens with it, and
  does not show again
- With several pending, the session lists them and asks which to pick up. The
  ones you pick or drop are closed with `ele handoff close <id>`; the rest show
  again at the next start, so a second stream of work is not lost

```sh
ele handoff list            # pending handoffs, newest first
ele handoff list --all      # with closed and superseded ones
ele handoff reopen <id>     # show a closed one again at the next start
```

### When to hand off

As a session's context grows past 150k tokens, ele's Stop hook suggests a
fresh start, once per further 100k:

- Mid-task, `/compact`, with the active plan's next item as its focus
- Otherwise, a handoff with `/ele:handoff`, then `/clear`

`ele context --size` prints the running session's context in tokens.

## Reminders

A reminder is something to come back to on a date. It shows at every session
start in its project from its due date until you close it:

```sh
ele remember --kind reminder --due 2026-11-01 "Renew the staging certificate"
```

Or say "remind me on the first of November to renew the staging certificate",
and `/ele:remember` works out the date.

```sh
ele reminder list           # open reminders, due or not, soonest due first
ele reminder close <id>     # done or dropped
ele reminder reopen <id>    # open it again
```

A reminder saved with `--global` shows in every project of the owner.

### Nudges

A nudge is a reminder that comes back. `--every <days>` makes one; closing it
hides it for that many days, then it is due again:

```sh
ele remember --kind reminder --every 7 "Review open dependency updates"
```

With no `--due`, a nudge is due today. `--done-file <path>` ties it to a file
that something else writes when the work is done, such as a weekly report: the
nudge is next due that many days after the file last changed. A nudge never
closes for good; `ele forget <id>` removes it.

The `reminders` option, on by default, turns every reminder off when set to
`false`. They can still be saved, listed and closed.
