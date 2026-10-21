---
name: ticket-audit
description: Audit a ticket map or ele plan against the code for stale premises and contradictions. Use before picking up a ticket or closing a map, or when asked to audit tickets, a map or a plan.
---

A ticket is a claim about code, and a claim about code is stale the moment the
code moves. Nothing says so. The cost is never a failing test — it is work done
twice.

Tracker writes are the user's to run; §9 says how.

## What this is for

The audit's subject is what tickets say and how they link, not labels,
milestones or boards.

A wrong ticket is mostly an adherence problem rather than a specification one:
the rules of whatever skill or template wrote the tickets usually already forbid
what went wrong. Audit against the rules that exist before concluding the rules
are missing.

The audit is also a source of errors. Its output gets the same scrutiny as the
tickets it read — §9 says how.

`/ele:ticket-claims` is not an audit. It matches phrasing and greps for the tokens a
ticket names, but never re-derives a premise; never report a tracker consistent
on its word.

Auditing an ele plan rather than a tracker map: read `PLAN.md` beside this file
first; it changes what each section below means.

## Questions and rulings

Before asking the user anything, `ele recall --project <name>` the question's
key terms, `<name>` being the basename of the repository the tracker belongs
to, and once §1 has fetched it, grep the corpus — tickets, their notes, linked
plans and ADRs. A hit is quoted, not asked: as of 2026-10-15, five of the
eight already-answered questions mined from past audits were answered in a
ticket, a tracker note or a plan.

A question nothing answers is asked once. Its answer is one of two things:

- **A decision or fact** that would outlive the item: propose it per the
  `ele:remember` skill's Propose section, with `--project <name>`.
- **A ruling** on one item's finding that would not outlive the item: recorded
  in the ledger at §10. A ruling that commits the item to an edit not made in
  this audit is also drafted at §9, as a note on the ticket: the builder reads
  the ticket, never the ledger.

Earlier audits' rulings come back in §1, each labelled text unchanged or
CHANGED since. A ruling never removes a finding or a method: the code may have
moved under an unchanged ticket. When a method raises the finding again, report
it with the ruling attached — unchanged asks the user to confirm it, changed to
rule again, quoting the note an earlier audit drafted for it where the ticket
holds one: that note is itself a change.

## 0. What is already known

Before fetching anything, load what is known per the `ele:recall` skill's
Project context section, `<topic>` being `<map number> audit`. Where that
skill is not listed, run `ele recall --project <name> "<topic>"`, then
`ele context --project <name>`.

The recall finds earlier audits of this map, their findings and the user's
rulings on them. The context prints the project's live notes: decisions and
standing facts — which repositories are ours, which are read-only — and open
questions.

## 1. Scope and corpus

Name the map and every ticket under it, open and closed. Closed tickets are in
scope: a closed ticket's decision is what the open ones were built on.

Fetch each ticket's **body and its notes together**, with whitespace normalised
before any matching. Run it in the repository the tracker belongs to — `glab` and
`gh` both read the remote to know which project they are talking about:

```sh
AUDIT=$(mktemp -d)
ele ticket corpus --map <id> --out "$AUDIT"
```

It takes children from two places: GitHub's native sub-issues, and any ticket
whose body says the literal `Part of #N`. A `note:` line names each ticket not
known closed that only one of them knows about, which is a linking gap worth reporting. A
`warning:` line names each open ticket still holding a template's `Part of #MAP`
or `Part of nothing`: a child no search finds, so ask whether it belongs and
re-run with it in `--tickets`. Read the list it
prints: a tracker using another convention still comes back short, and
`--tickets` takes an explicit set. It refuses a directory that already holds a
corpus unless given `--clean`, because two maps in one directory read as one map
contradicting itself. Exit 2 means the corpus is partial and no method can run
on it yet.

Do this once and grep the corpus, never the tracker. Line-wrapped prose turns an
existing phrase into an absent one: it broke across two lines, nothing says so,
and the audit reports clean on the thing it was run to find.

Each ticket lands twice — `<n>.md` with whitespace collapsed for matching, and
`raw/<n>.md` as written. Match against the first; anything about where a line
sits reads the second, because the collapsed copy is one line.

`manifest.json` beside them holds each ticket's ledger key and the hash of its
title, body and comments as fetched. Then the user's earlier rulings on these
tickets, read per Questions and rulings above:

```sh
ele ticket ledger rulings --manifest "$AUDIT/manifest.json"
```

If the work came from a parent effort with its own map, fetch that too. A
contradiction between two maps reads exactly like a contradiction inside one.

Add every plan, ADR and design document the map or its tickets link to. A
decision recorded there and not in the ticket is still a decision.

List the repositories the map's journeys cross. For each: whether it is ours or
read-only, and whether the local clone is current — ask the user rather than
assert it. As of 2026-10-09 a stale sibling clone hid an audit's biggest finding.

Derive the frontier — the open tickets no open ticket blocks — from the blocker
table the script prints after the corpus hash and the tracker's native blocking
links, which the corpus does not fetch. The table takes only the refs leading
each `Blocked by` line, so "Blocked by: nothing — see #177" is no edge. A
`warning:` names each ticket with a `Blocked by` line it could not read, such as
a heading over a list: read those in `raw/<n>.md`, and likewise any ticket whose
prose says it waits on something the table does not show. §3's premises method
runs over the frontier.

A blocker marked `?` is outside the corpus and has no state in it; fetch it
from the repository it names (a bare `#N` names this one) before letting it hold
a ticket off the frontier. Count `OPEN` (gh) and `opened` (glab) as open:

```sh
gh issue view <n> -R <owner/repo> --json state --jq .state
glab issue view <n> -R <host/group/project> -F json | jq -r .state
```

On GitHub, each ticket's native blockers, closed ones included, with their state:

```sh
gh api --paginate repos/{owner}/{repo}/issues/<n>/dependencies/blocked_by --jq '.[] | "\(.number) \(.state)"'
```

On GitLab, each ticket's links with their state and type; the blockers are the
`is_blocked_by` ones:

```sh
glab api 'projects/:id/issues/<n>/links' | jq -r '.[] | "\(.iid) \(.state) \(.link_type)"'
```

No ticket in the map coming back with a `blocks` or `is_blocked_by` link means
the map uses no native blocking, and the `Blocked by` lines are the whole graph.

## 2. Decide which methods to run

Every subagent this audit launches gets the same brief: the corpus directory,
what §0 recalled, §1's earlier rulings with their labels, the user's rulings and
answers so far, the repository list with which are ours and which clones are
current, the scope the user approved (proposed below), and for a plan,
`PLAN.md`'s absolute path, resolved from this skill's base directory. It hands
back questions instead of asking the user; put them to the user, then resume the
same subagent with the answers rather than starting another. §9 and §10 stay
here.

Before each method, check whether this session has compacted since the audit
began: ele's start block, shown again after a compaction, opens with a line
saying when. No ele start block in context at all means this check cannot be
made here: say so. If it has compacted, run the remaining methods in a fresh
subagent rather than from this session's memory of the tickets; a summary of a
ticket is not the ticket.

Each method below finds a different class. A clean result from one says nothing
about the others.

No result carries over from an earlier run, this session's included. As of
2026-10-15 a corpus-wide hash never once let a method stand: every run, writes or
commits since the last one had changed the corpus.

Propose the scope — the tickets, the methods, and why the others are left out —
and wait for the user's go before running any method. §3's premises method over
the frontier is never left out — it finds the most and, done shallowly, misses
the most. Over about fifteen open tickets, split it across subagents by cluster,
each with the brief above, and spot-check each one's costliest claims yourself
before filing any of them.

## 3. Method — premises against the code

The one that finds what would have shipped. For every claim each open ticket on
the frontier makes about the codebase, open the code and re-derive it. Do not
read for agreement; read for what the ticket did not ask.

Every claim gets a `file:line` or "not checked". "Premises hold" without them is
a claim nobody opened — as of 2026-10-15, all but one premise the audits passed
that a build later found wrong was one not opened, opened too shallowly, or
sitting in a repository not read. Read the ticket's comments as well as its
body. Where the data is in hand — a dump, a query result — run the ticket's
query or backfill against it.

- **Claims of absence.** "There is no X", "nothing reads Y", "the row has no
  field for it". True the day it was typed and indistinguishable from the truth
  afterwards. Each one is a grep.
- **Claims of mechanism.** "The endpoint takes only an email", "the column is an
  enum". Open it. An endpoint that also hardcodes a verified flag, or a column
  that is a plain varchar, changes what the ticket is asking for.
- **Work the ticket calls for that is already done, or never needed.** A bullet
  demanding a migration for a column that needs none produces an empty migration
  and a confused implementer.
- **Work the ticket does not call for that the change requires.** The gate it
  names is rarely the only gate. Follow the journey end to end and count them.
- **Phrases nobody can source.** A term that appears in a ticket, a note and a
  document but in no code and no upstream is usually one agent's invention
  propagating. Find its origin or retire it.

Report each as: the claim, where it is written, what the code says, and what that
changes about the work.

## 4. Method — contradiction between tickets

Pairwise, over the normalised corpus: every pair of tickets naming the same
artefact, population or condition. Done when each such pair is compared and the
report lists the pairs, contradicting or not.

Before calling one stale, check **position**, in `raw/<n>.md` rather than the
collapsed copy. A correction that exists but sits far below the claim it corrects
is not a stale ticket — it is a mis-ordered one, and the fix is to move or merge,
not to rewrite.

A ticket that opens with a premise and later says "the premise above is wrong" is
working as intended, provided the warning is at the top. If it is not, move it.

## 5. Method — artefacts without a producer or a consumer

List every artefact the tickets name: a column, an endpoint, a record, a flag, an
email template, a state. For each, find the ticket that produces it and the
tickets that read it.

- **No producer:** something is to be read that nothing creates.
- **No consumer:** something is to be built that nothing reads. The wire-field
  trap at ticket scale.
- **Two producers:** two tickets will both build it, differently.
- **Named differently by producer and consumer:** the implementer of the
  consumer searches the producer for a term that is not there and concludes it
  was never decided. Align the vocabulary or say they are the same thing.

## 6. Method — the dependency graph

- Every `Blocked by` names a ticket that exists and is open. A blocker on a
  closed ticket is noise that hides a real one.
- No cycles. No ticket blocked on a descendant.
- §1's frontier is what someone will actually pick up. Check it is what the map
  intends.
- Where the tracker supports native blocking links, prefer them and link rather
  than restate. Where it does not, the prose line is the graph and is checked by
  hand.
- Re-derive the build order from the edges rather than reading the order section,
  then compare the two.

## 7. Method — where decisions live

- **A decision in a note while the body still asks the question.** Whoever picks
  the ticket up reads the body. Either the body carries the answer or it says
  plainly that the answer is below.
- **A decision in two places.** The map is an index: a decision lives in exactly
  one ticket and the map gists it and links. Two copies are two things to rot,
  and they will.
- **A decision the map records and no ticket implements.** The build will not
  happen; nothing is tracking it.

## 8. Method — status claims and self-granted licence

- Every status sentence in the map, checked against the children. "Reached",
  "every decision closed", "ready to hand to a build session" — count the open
  tickets and compare. A map that tells readers the work is ready when it is not
  is the most consequential thing this audit can find.
- Counts and lists go stale on the next edit. Prefer a link or a query; where a
  figure must be written, date it.
- **Does the map grant itself permissions?** A line in the map's own notes
  licensing execution, scope or exemption is written by the party it constrains
  and read back later as authority. Quote it and ask.

## 9. Fixing

Order the findings by what they cost: a premise that would mis-build first, a
contradiction that would stall someone second, prose last.

For each, draft the exact tracker write. Edit the body where the ticket's own
premise is wrong; add a note where a decision is being recorded. Where a
correction lands in a body, put it at the top, not after the thing it corrects.
Before drafting a note, check the ticket does not already hold the same one.

Before handing the drafts over, audit them as you audited the tickets — they are
claims written in a hurry by the party least able to see their errors:

- §3's premises method over every premise a draft adds or changes, and every
  decision taken while grilling the user, with the same `file:line` rule.
- §6's graph method over the graph as the drafts leave it: cycles, a blocker on
  a closed ticket, a descendant blocking its ancestor.
- A "Blocked by" line holds tickets only. Prose after it gets parsed as an edge.

Hand the writes over as one script the user runs with `!`; the agent never runs
it. Make it resumable: each write records itself in a file beside the script — a
create its new ticket's ID, a note or edit its step — and a rerun skips every
step already recorded, so a partial failure posts nothing twice.

After the user runs it, re-fetch and re-read. A write that did not land reads
exactly like one that did. Save each drafted body and note to a file, and read
each write back against its file — a body edit plainly, a note with `--note`:

```sh
ele ticket corpus --compare <n> --expect <file> [--note]
```

Exit 0 names the body or comment that matches; 1 prints a diff against the
closest; 2 means the ticket or file could not be read, which says nothing about
the write. A body edit is never matched against comments, so a note quoting it
cannot pass it. A `diff` against `raw/<n>.md` instead fails on the header line
the corpus adds and on trailing newlines, which reads as a write that did not
land.

## 10. Report

- Per method: the calls or output that show it ran, or "not run" and why.
- The premises checked, each with its `file:line`, and those left unchecked.
- The repositories read, and at which commit.
- Per finding: what was claimed, what is true, where it is written, what it
  changes, and whether it is fixed, drafted or deliberately left.
- The frontier, as the audit now understands it.
- Anything that could not be settled from the code and needs a person.

"Audited, no contradictions" has run one method and reported on six.

Last, after §9's re-read, record the audit. Write `$AUDIT/records.json`, one
record per ticket in scope, naming only the methods that reached that ticket and
the user's rulings on its findings:

```json
[{"key": "<manifest key>", "methods": ["contradiction", "premises"],
  "rulings": [{"finding": "<what was raised>", "ruling": "<what the user ruled, and why>"}]}]
```

Methods are `premises`, `contradiction`, `artefacts`, `graph`, `decisions` and
`status`, §3 to §8. Then:

```sh
ele ticket ledger record --manifest "$AUDIT/manifest.json" --records "$AUDIT/records.json"
```

It stamps each line with the hash the audit read and refuses the whole set, writing
nothing, on a key not in the manifest or an unknown method. Never write the
ledger by hand.
