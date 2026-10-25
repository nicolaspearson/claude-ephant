---
name: plan-audit
description: Audit ele plans against the code and each other for stale premises, contradictions, waits nothing ends, inbox items to triage and plans ready to close. Use before picking up or closing a plan, when plans feel stuck, or when asked for a plan audit.
---

A plan audit runs ticket-audit's engine over plans. `<ticket-audit>` is
`ticket-audit/SKILL.md` in the skills directory holding this skill's base
directory. Read it first and follow its sections, with each item standing for a
ticket and the differences below. Then run this skill's own methods.

Plan files are repository files: every fix is an edit made on approval, never a
script. Notes go through the `ele:remember` skill's Propose section.

## Which plans

With paths, those plans; `/ele:plan convert` passes its drafts, which are not
yet active. With none, every active plan of the repository:

```sh
ele plan list --project <name>
```

`<name>` is the repository's basename. Then, per plan, from the root of the
repository holding it, since `ele plan check` resolves a path in the working
directory's checkout:

```sh
ele plan check <path>
```

Its warnings are mechanical and already found: a blocker done or missing, an
`after:` date passed, an `as of` line past 14 days, an old or undated inbox
item, an open action with no done-when, a name another plan holds, a plan ready
to close. Report them as found by `ele plan check`; no method re-derives them.

Done when every plan in scope is listed with its path and its check warnings, or
"no warnings".

## Ticket-audit's sections, for plans

- §0: `<topic>` is the plans' file names and `audit`. Also read the newest
  handoff, `ele handoff list --all --project <name>`, and keep `ele context`'s
  open question notes for Open questions below
- §1: one corpus for every plan in scope:

  ```sh
  AUDIT=$(mktemp -d)
  ele ticket corpus --plan <path> --plan <path> ... --out "$AUDIT"
  ```

  Each Actions and Inbox item lands as `<plan>-<nn>.md` and `raw/<plan>-<nn>.md`,
  `<plan>` being the file name without `.md`, `<nn>` its position in the file.
  Add each plan's `spec` from its front matter, and read its Context and
  Decisions as the plan's notes. Name an item `<plan>/<name>`, the kebab name
  after its checkbox; an unnamed one, an inbox item or the closing action, by
  plan and title. Its ledger key is the repository, the plan's path and its
  name, so rewording keeps its rulings; an unnamed item's key is its title.
  Two plans of one file name, drafts in two repositories, are refused: audit
  each repository's plans in a run of its own
- Frontier: per plan, the open actions `ele plan show <path>` does not list as
  blocked. A `blocked:` line naming items is an edge, `<name>` in its plan or
  `<plan>/<name>` in another; a line with prose before its `;` is no edge and
  holds the item until removed by hand. An `after:` line holds an item until
  its date
- §2: the subagent brief also carries this file's absolute path, which a
  subagent reads after `<ticket-audit>`
- §3 runs over the frontier of every plan in scope
- §4 and §5 always run, and across plans: a producer in one plan and its
  consumer in another is what a single-plan audit misses
- §6: the edges above. `ele plan check` already flags a blocker done or missing;
  check that file order builds, an item reading what a later item produces being
  out of order
- §7: a decision lives once, in a plan's Decisions or its spec; an item or
  another plan restating it is a second copy
- §8: each `as of` line under its age limit still needs its figure re-counted
  where §3 reaches it; a plan's status sentence is checked against its ticks
- §9: propose each edit, apply it on approval, and read the file back. A ruling
  that commits an item to an edit not made in this audit is written as a
  `next:` line in the item's body. Skip the script, the `!` and `--compare`
- §10: the ledger takes §3 to §8's methods only; this skill's own methods are
  reported, not recorded
- Elsewhere: `<n>.md` is `<plan>-<nn>.md`; a ticket's comments are its item's
  body; the map's notes are the plans' Context and Decisions

## Waits nothing ends

Every `blocked:` line, `after:` line, and decision or item saying it waits
until, once or after something: name the item, date or event that ends the
wait. A wait on an item no plan holds, on an event nothing produces, or on prose
nobody will remove is a finding: the item sits off the frontier for good.

Done when every wait in scope is listed with what ends it, or flagged.

## Inbox triage

For each inbox item of every plan in scope, with its age from its leading date,
propose one:

- **Delete**: an action already covers it, or it is done
- **Finding**: it contradicts a decision; report it under §7
- **Move to an action** in its plan, named, with a done-when
- **Move to another plan's inbox**: it serves that plan's why
- **Plan candidate**: it serves no active plan's why; a `question` note starting
  `Plan candidate:`, then delete the line
- **Out of scope**: `- <gist>: <why> (ruled <date>, from <source>)` under the
  plan's `## Out of scope`
- **Leave**: not sharp enough to be an action yet

A line moved into a plan cites no ele data, which resolves only in its author's
data directory: delete a note id or session that records where it came from,
describe what was observed in place of one cited as evidence, and move a note it
depends on into the plan, superseding the note.

Done when every inbox item has a proposal and each approved one is applied.

## Plan candidates

From `ele context --project <name>`, every open `question` note starting
`Plan candidate:`, grouped by theme. For each group, propose: a new plan through
`/ele:plan new`, an active plan's inbox, or waiting. A group becoming a plan
or moving into an inbox supersedes its notes.

Done when every plan candidate note is in a group with a proposal.

## Open questions

Every other open `question` note of the project: propose a home. A plan's inbox,
when an active plan's why covers it; a plan candidate, reworded to start
`Plan candidate:`; or staying a question, when it waits on a measurement or a
person. A note that moves is superseded through `ele remember --supersedes`, on
approval. Related questions become one candidate note, repeating `--supersedes`
once per question.

Done when every open question note has a proposed home.

## Handoff

The newest handoff from §0: check each claim it makes about a plan, an item's
state, a commit or a next step against the plan files and `git log`. A claim the
files no longer bear out is a finding; the next session starts from it.

Done when each of the handoff's claims about plans is marked holds or stale, or
there is no handoff.

## Closing

- Per plan that `ele plan check` calls ready to close: its inbox empty, or each
  item triaged above; its out-of-scope lines ready for the fold. Propose closing
  it next
- How many plans are active. Over 3, flag it and propose which to close first:
  the one nearest done, or one whose why another plan now covers

Done when each plan's readiness is stated and, over 3 active, an order to close
them is proposed.

## Report

ticket-audit's §10 report, plus:

- Each plan with its `ele plan check` warnings
- Per method of this skill: what it found, or "nothing" with what it read
- Every proposal, applied, declined or open

Done when every plan in scope and every method above appears in the report.
