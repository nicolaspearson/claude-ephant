# Plans

A plan is a fixed list of work, done one item a session, with ele carrying your
place across `/clear`. It is a Markdown file in the repository it changes, under
`.documentation/plans/`, so it is reviewed and versioned with the code. The
file is the list and its checkboxes are the state: ele stores only which plans
are active.

A plan is not a tracker. It has no assignees, and ele writes no tickets.

## Starting one

`/ele:plan new <name>` writes a plan with you. Claude asks what the conversation
has not settled, in rounds, each question with a recommended answer, then
writes the file and, on your approval, starts it.

`/ele:plan convert <spec>` turns a spec's open items into plans: it checks each
item against the code, drafts plans grouped by what ships together, audits the
drafts and proposes them.

To adopt a plan you wrote yourself:

```sh
ele plan start .documentation/plans/<name>.md
```

It refuses a file without `kind: plan` in its front matter and a checkbox under
`## Actions`.

## The format

```markdown
---
kind: plan
created: 2026-10-23
---

# Faster sync

## Why

## Context

## Decisions

## Actions

- [ ] batch-writes: Write index rows in one transaction per file
  - done when a sync of 1,000 transcripts takes under 10 seconds
- [ ] drop-rescans: Skip files whose size has not changed
  - blocked: batch-writes
  - done when …
- [ ] Propose each out-of-scope line as a `decision` through `/ele:remember`, run `ele plan done .documentation/plans/faster-sync.md`, and delete this plan

## Inbox

## Out of scope
```

- **Front matter** holds `kind: plan`, `created`, and `spec` when the plan
  implements a spec in the same repository
- **Items** are `- [ ]` lines with an indented body. Each but the last leads
  with a short kebab-case name, written once and never changed, and carries a
  `done when` line saying what will be observable once it is done
- **`blocked:`** names the items an item waits on, in this plan or
  `<plan>/<name>` in another; a reason may follow a `;`. **`after: <date>`**
  holds an item until then
- **Progress** goes in the item's body as dated lines, `as of <date>: …` or
  `next: …`, so a session stopping mid-item needs no handoff. A tick carries
  one line saying what was checked, and at which commit
- **`## Inbox`** takes work found mid-session, each line dated with where it came
  from. **`## Out of scope`** keeps what was ruled out, and why
- **The last action** closes the plan: with a spec, fold what the plan built into
  it; without one, propose each out-of-scope line as a decision note. Then
  `ele plan done`, and delete the file. Git history is its archive

## Working through one

At startup and after `/clear`, the session opens with each active plan and its
next item: the first open item nothing blocks. Claude asks before starting
anything.

`/ele:plan` with no argument picks up where you left off. It checks whether the
next item is already done in the commits since the last tick, shows the item
with related notes, and asks to start. Once you say yes, it re-checks what the
item claims about the code before building on it, and triages the inbox.

When a change completes an item, `/ele:checkpoint` proposes the tick with its
evidence line; otherwise a dated progress line. Both are written only on your
approval.

```sh
ele plan show               # the active plan: counts, previous item, blocked items, next item
ele plan list               # every active plan of the project
ele plan check              # warnings: stale dates, done blockers, old inbox items, missing done-when
```

## Auditing plans

`/ele:plan-audit` checks every active plan against the code and against each
other: premises that no longer hold, items that contradict, waits nothing will
end, inbox items to triage, and plans ready to close. Run it before picking up a
plan after time away, or when plans feel stuck.

## Closing one

Once the last action is the only one open and the inbox is empty, do the last
action. `ele plan done <path>`, run from the plan's checkout, closes the plan
and keeps its item names, so a `blocked:` line in another plan still resolves
and names stay unique after the file is deleted. It closes whatever it is given,
so check the inbox is empty first; `ele plan check` says when only the last
action is left.
