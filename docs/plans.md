# Plans

A plan is a fixed list of work, done one item a session, with ele carrying your
place across `/clear`. It is a Markdown file in the repository it changes, under
`.documentation/plans/`, so it is reviewed and versioned with the code. The
file is the list and its checkboxes are the state: ele stores only which plans
are active.

A plan is not a tracker. It has no assignees, and ele writes no tickets.

## Starting one

`/ele:plan new <name>` writes a plan with you. Claude first reads the
repository's accepted ADRs, which settle what they decide, and looks up the
facts each question rests on. It then asks what the conversation has not
settled, in rounds, each question with a recommended answer. It writes the
file, with what the plan must deliver as `## Requirements` in your words, then
runs the review loop, and on your approval starts the plan.

`/ele:plan convert <spec>` turns a spec's open items into plans: it checks each
item against the code, drafts plans grouped by what ships together, runs the
review loop over the drafts and proposes them.

### The review loop

Before you see a draft, it goes through 2 or 3 passes. Each pass audits the
drafts with `/ele:plan-audit drafts` and has a fresh `ele:plan-reviewer` agent
review each one (see [Auditing plans](#auditing-plans)). Every finding gets an
id, such as `A1.2` from the audit or `R2.1` from a reviewer, and a verdict
before anything is edited:

- `fixed`, with the revision that fixes it
- `refuted`, with the evidence that it is wrong
- `deliberate`, with why the plan keeps it
- `open`, a question for you

A later pass sees the earlier findings with their levels and verdicts, and
raises one again only with new evidence. A finding against a requirement is
always `open`: the loop never edits your requirements. The loop stops after
pass 2 if no blocker or major is open and pass 2 fixed none of them;
otherwise, so that fix is itself reviewed, it runs
a third and last pass. You then see every finding with its id, level and
verdict, minors listed without blocking, are asked the open ones, and see a
diff of each draft from its first version.

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

## Requirements

1. A sync of 1,000 transcripts finishes in under 10 seconds

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
- **`## Requirements`**, optional, lists what the plan must deliver, numbered,
  in your words; the review loop checks the items against it and never edits
  it
- **`## Context`** holds dated facts; one not checked against the code says
  "not checked". A **Decision** that chose an architecture lists each option
  it turned down as `rejected: <option>, because …`
- **Items** are `- [ ]` lines with an indented body. Each but the last leads
  with a short kebab-case name, written once and never changed, and carries a
  `done when` line saying what will be observable once it is done
- **`blocked:`** names the items an item waits on, in this plan or
  `<plan>/<name>` in another; a reason may follow a `;`. **`after: <date>`**
  holds an item until then
- **Progress** goes in the item's body as dated lines, `as of <date>: …` or
  `next: …`, so a session stopping mid-item needs no handoff. A tick carries
  one line saying what was checked, and at which commit
- **No ele data**: a note or session id resolves only in its author's data
  directory, so a plan line describes what was observed, and a note the plan
  depends on moves into it; `ele plan check` warns on a note id, a session
  UUID or an `ele show` of an id that ele resolves, a saved note or a
  session indexed or still only a transcript, so a commit or a repository's
  own UUID passes; with the index unreadable it warns on every such line
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
ele plan check              # warnings: stale dates, done blockers, old inbox items, missing done-when, ele data cited
```

## Auditing plans

`/ele:plan-audit` checks every active plan against the code and against each
other: premises that no longer hold, items that contradict, waits nothing will
end, inbox items to triage, and plans ready to close. Run it before picking up a
plan after time away, or when plans feel stuck.

`/ele:plan-audit drafts <path>...` audits plans not yet started, for a review
before `ele plan start`, adding every active plan itself; the review loop runs
it on each pass. It reports each finding with a level, blocker, major or minor,
and the line it rests on, and returns its questions in the report: it asks you
nothing, edits no plan and records nothing in the audit ledger.

The `ele:plan-reviewer` agent reviews a draft plan before it starts, launched
by the review loop on each pass. It never sees the session that wrote the
draft. It reads the draft, its spec, your and the repository's `CLAUDE.md`,
the repository's accepted ADRs and ele's planning-practices reference, and
reports:

- each requirement against the items that deliver it
- a requirement that is ambiguous, untestable or names an implementation, as a
  question for you
- an item no requirement asks for, or one going past what its requirement asks
- an item contradicting the spec or an accepted ADR, or breaking a `CLAUDE.md`
  rule
- a done-when line that cannot be observed, cannot be met, or could pass while
  its requirement fails
- a practice the plan is missing

Each finding is a blocker (the plan would build the wrong thing or cannot
finish), a major (a requirement could go undelivered, or an item cannot be
ticked as written) or a minor. ADRs are looked for under `docs/adr`,
`.documentation/adr` or `docs/decisions`, or the directory the repository's
`CLAUDE.md` names. An ADR with no status line counts as accepted, and a
partially accepted one binds for the part it says is in force.

## Closing one

Once the last action is the only one open and the inbox is empty, do the last
action. `ele plan done <path>`, run from the plan's checkout, closes the plan
and keeps its item names, so a `blocked:` line in another plan still resolves
and names stay unique after the file is deleted. It closes whatever it is given,
so check the inbox is empty first; `ele plan check` says when only the last
action is left.
