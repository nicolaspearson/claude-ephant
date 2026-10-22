---
name: ticket-audit
description: Audit a tracker's ticket map against the code for stale premises and contradictions. Use before picking up a ticket, before closing a map, or when asked for a ticket audit; for ele plans, use ele:plan-audit.
---

Tracker writes are the user's to run; §9 says how.

## What this is for

The audit's subject is what tickets say and how they link.

A premise counts as checked only with §3's `file:line`. `/ele:ticket-claims`
output is a label check on wayfinder tickets, not that.

## 0. What is already known

Before fetching anything, load what is known per the `ele:recall` skill's
Project context section, `<topic>` being `<map number> audit` and `<name>` the
basename of the repository the tracker belongs to. Where that skill is not
listed, run `ele recall --project <name> "<topic>"`, then
`ele context --project <name>`.

The recall finds earlier audits of this map, their findings and the user's
rulings on them. The context prints the project's live notes: decisions and
standing facts — which repositories are ours, which are read-only — and open
questions.

Done when you can name each earlier audit of this map with its date and
rulings, or have said there are none.

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
whose body says the literal `Part of #N`; on GitLab, only the second. On GitHub
a `note:` line names each ticket not known closed that only one of them knows
about, which is a linking gap worth reporting. A `warning:` line names each open
ticket in the project still holding a template's `Part of #MAP` or `Part of
nothing`. One "not in this corpus" is a child no search finds: ask whether it
belongs and re-run with it in `--tickets`. One "in this corpus" is reported.
Read the list it prints: a tracker using another convention still comes back
short, and `--tickets` takes an explicit set. It refuses a directory that
already holds a corpus unless given `--clean`, because two maps in one directory
read as one map contradicting itself. Exit 2 with `FAILED to fetch` means the
corpus is partial and no method can run on it yet; exit 2 with an `ele:` line is
a usage error; exit 1 is a refusal, such as a used directory.

Grep the corpus, never the tracker: wrapped prose hides a phrase from a tracker
search, and the audit reports clean on the thing it was run to find.

Each ticket lands twice — `<n>.md` with whitespace collapsed for matching, and
`raw/<n>.md` as written. Each holds the body, then its notes. Match against the
first; anything about where a line sits reads the second, because the collapsed
copy is one line.

`manifest.json` beside them holds each ticket's ledger key and the hash of its
title, body and comments as fetched. Then the user's earlier rulings on these
tickets, read per Questions and rulings below:

```sh
ele ticket ledger rulings --manifest "$AUDIT/manifest.json"
```

If the work came from a parent effort with its own map, fetch that too. A
contradiction between two maps reads exactly like a contradiction inside one.

Add every plan, ADR and design document the map or its tickets link to. A
decision recorded there and not in the ticket is still a decision.

Add the template or skill that wrote the tickets. A wrong ticket mostly broke a
rule that already exists: audit against the rules that exist before concluding
a rule is missing.

List the repositories the map's journeys cross. For each: whether it is ours or
read-only, and whether the local clone is current — ask the user rather than
assert it. As of 2026-10-09 a stale sibling clone hid an audit's biggest finding.

The frontier is the open tickets no open ticket blocks. Its edges come from two
places: the blocker table `ele ticket corpus` prints after the corpus hash, and
the tracker's native blocking links, which the corpus does not fetch.

The table takes only the refs leading each `Blocked by` line, so "Blocked by:
nothing — see #177" is no edge, and it reads notes as well as bodies, so a
`Blocked by` line in a note is one. A `warning:` names each ticket with a
`Blocked by` line it could not read, such as a heading over a list: read those
in `raw/<n>.md`, and likewise any ticket whose prose says it waits on something
the table does not show.

A blocker marked `?` is outside the corpus and has no state in it; a bare `#N`
names this repository. Fetch each `?` blocker's state and each ticket's native
blockers per `TRACKERS.md` beside this file, before letting a blocker hold a
ticket off the frontier. No ticket in the map coming back with a `blocks` or
`is_blocked_by` link means the map uses no native blocking, and the `Blocked by`
lines are the whole graph.

Then write the frontier out. §3's premises method runs over it, except under a
pickup (§2).

Done when the corpus exited 0; every `note:` and `warning:` line is resolved or
listed for the report; the earlier rulings are printed; every linked plan, ADR
and design document is in the corpus; each repository is marked ours or
read-only and current per the user; every `?` blocker has a fetched state; and
the frontier is written as a list of ticket numbers.

## Questions and rulings

Before asking the user anything, `ele recall --project <name>` the question's
key terms, and grep the corpus — tickets, their notes, linked plans and ADRs. A
hit is quoted, not asked: as of 2026-10-15, five of the eight already-answered
questions mined from past audits were answered in a ticket, a tracker note or a
plan.

A question nothing answers is asked once. Its answer is one of two things:

- **A decision or fact** that would outlive the item: propose it per the
  `ele:remember` skill's Propose section, with `--project <name>`.
- **A ruling** on one item's finding that would not outlive the item: recorded
  in the ledger at §10. A ruling that commits the item to an edit not made in
  this audit is also drafted at §9, as a note on the ticket: the builder reads
  the ticket, never the ledger.

Earlier audits' rulings come back in §1, each labelled text unchanged or
CHANGED since. Every method runs and every finding is reported whatever the
ruling: the code may have moved under an unchanged ticket. When a method raises
the finding again, report it with the ruling attached — unchanged asks the user
to confirm it, changed to rule again, quoting the note an earlier audit drafted
for it where the ticket holds one: that note is itself a change.

## 2. Decide which methods to run

§3's **premises scope** is the frontier. Picking up one ticket narrows it: the
corpus is still the ticket's map, or the ticket alone where it has none, and the
premises scope is that ticket and its blockers, closed ones included.

Every subagent this audit launches gets the same brief: the corpus directory,
what §0 recalled, §1's earlier rulings with their labels, the user's rulings and
answers so far, the repository list with which are ours and which clones are
current, and the scope the user approved (proposed below). It hands back
questions instead of asking the user; put them to the user, then resume the
same subagent with the answers rather than starting another. §9 and §10 stay
here.

Every method runs fresh on this corpus, earlier runs this session included. As
of 2026-10-15 a corpus-wide hash never once let a method stand: every run,
writes or commits since the last one had changed the corpus.

Propose the scope — the tickets, the methods, and why the others are left out —
and wait for the user's go before running any method. §3's premises method over
the premises scope is always in scope — it finds the most and, done shallowly,
misses the most. Over about fifteen open tickets, split it across subagents by
cluster, each with the brief above, and spot-check each one's costliest claims
yourself before filing any of them.

Done when the user has approved a scope naming each ticket, each method, and why
each method left out is left out.

## Methods, §3 to §8

Before each method, look in ele's start block for a line `This session compacted
at <time>`; with no ele start block in context, say this check cannot be made.
After a compaction, run the remaining methods in a fresh subagent: a summary of
a ticket is not the ticket.

## 3. Method — premises against the code

The one that finds what would have shipped. For every claim each ticket in the
premises scope (§2) makes about the codebase, open the code and re-derive it.
Read adversarially: for what the ticket did not ask.

Every claim gets a `file:line` or "not checked". "Premises hold" without them is
a claim nobody opened — as of 2026-10-15, all but one premise the audits passed
that a build later found wrong was one not opened, opened too shallowly, or
sitting in a repository not read. Where the data is in hand — a dump, a query
result — run the ticket's query or backfill against it.

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
- **Phrases without provenance.** A term that appears in a ticket, a note and a
  document but in no code and no upstream is usually one agent's invention
  propagating. Find its origin or retire it.

Report each as: the claim, where it is written, what the code says, and what that
changes about the work.

Done when every claim of every ticket in the premises scope, body and notes, is
listed with its `file:line`, or "not checked" and why.

## 4. Method — contradiction between tickets

Pairwise, over the normalised corpus: every pair of tickets naming the same
artefact, population or condition. Done when each such pair is compared and the
report lists the pairs, contradicting or not.

Before calling one stale, check **position**, in `raw/<n>.md` rather than the
collapsed copy. A correction that sits below the claim it corrects breaks §9's
lede rule: the ticket is mis-ordered, and the fix is to move or merge.

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

Done when every named artefact is listed with its producer and consumers, or the
gap class it falls in.

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

Done when every edge is checked against the rules above, and the build order
re-derived from the edges sits beside the map's stated order with each
difference listed.

## 7. Method — where decisions live

- **A decision in a note while the body still asks the question.** Whoever picks
  the ticket up reads the body. Either the body carries the answer or it says
  plainly that the answer is below.
- **A decision in two places.** The map is an index: a decision lives in exactly
  one ticket and the map gists it and links. Two copies are two things to rot,
  and they will.
- **A decision the map records and no ticket implements.** The build will not
  happen; nothing is tracking it.

Done when every decision found in a note, the map or a linked document is listed
with the one ticket body holding it, or flagged.

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

Done when every status sentence in the map is quoted with the count that
confirms or refutes it, and every self-granted permission is quoted and asked.

## 9. Fixing

Order the findings by what they cost: a premise that would mis-build first, a
contradiction that would stall someone second, prose last.

For each, draft the exact tracker write, and a note for each ruling per
Questions and rulings. Edit the body where the ticket's own premise is wrong;
add a note where a decision is being recorded. The correction is the lede: where
one lands in a body, it opens the body. Before drafting a note, check the ticket
does not already hold the same one.

Before handing the drafts over, audit them as you audited the tickets — they are
claims written in a hurry by the party least able to see their errors:

- §3's premises method over every premise a draft adds or changes, and every
  decision taken while grilling the user, with the same `file:line` rule.
- §6's graph method over the graph as the drafts leave it: cycles, a blocker on
  a closed ticket, a descendant blocking its ancestor.
- A `Blocked by` line leads with its refs; a blocker named only in prose after
  them is no edge, and a `Blocked by` line in a note counts as one.

Hand the writes over as one script the user runs with `!`; the agent never runs
it. Make it idempotent: each write records itself in a file beside the script —
a create its new ticket's ID, a note or edit its step — and a rerun skips every
step already recorded.

After the user runs it, re-fetch and re-read: a write has landed only once read
back. Save each drafted body and note to a file, and read each write back
against its file — a body edit plainly, a note with `--note`:

```sh
ele ticket corpus --compare <n> --expect <file> [--note]
```

Exit 0 names the body or comment that matches; 1 prints a diff against the
closest; 2 means the ticket or file could not be read, which says nothing about
the write. A body edit is never matched against comments, so a note quoting it
cannot pass it. A `diff` against `raw/<n>.md` instead fails on the header line
the corpus adds, on the notes it appends and on trailing newlines, which reads
as a write that did not land.

Done when every write the user ran reads back exit 0 from `--compare`, or its
mismatch is in the report.

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

It stamps each line with the hash the audit read and refuses the whole set,
writing nothing, on any bad record. Write the ledger only through this command.

Done when the report carries every bullet above, each method's line naming its
calls or "not run" and why, and the record prints `recorded <n> item(s)` with n
equal to the tickets in scope.

## Errors

When a call fails, quote its error verbatim, find its row below, apply that
row's one fix and run once more — never the same call unchanged. An error in no
row, or a second failure: stop and report, quoting the error.

Seen counts are tool errors inside ticket-audit runs, as of 2026-10-21
(`ele events --during ticket-audit --kind tool-error`), classed by each error's
first and last 300 characters; `gh`, `glab` and `ele ticket` had none.

| Error contains | Class | One fix | Seen |
| --- | --- | --- | --- |
| `not found` after a word starting with `=`, such as `=== not found` | zsh took the word, often an `echo ====` separator, for a command and dropped the rest of the line: nothing after it ran | Quote it, `echo '===='`, or use `echo ----`; rerun what came after it | 14 |
| `Exit code 1` with no error text at the end | Usually a `grep` that matched nothing, or a command with stderr sent to `/dev/null`; the output cannot tell which. Output ending in `Traceback`, an `…Error:` line, `no such file`, `FAIL`, `panic`, `fatal:` or `command not found` is a real failure, not this row | Read the output; if what is missing matters, rerun that part alone with stderr kept | 5 |
| `Blocked: git writes are the user's to run` | Guard hook: only listed git reads pass, with or without `-C`, and one unlisted git command blocks the whole call; `branch -a`, `branch -r`, `branch --contains`, `tag --contains`, `tag --sort` and `reflog` are not listed | Run the listed reads apart from the rest; for the unlisted, `git for-each-ref --contains <sha> refs/heads refs/remotes refs/tags`, `git for-each-ref refs/heads refs/remotes`, or `git log -g`; a real write goes to the user | 6 |
| `no matches found: <glob>` | zsh expanded an unquoted glob that matched nothing; that command did not run | Quote the glob | 4 |
| `Blocked: cd chained with an edit` | Guard hook | Absolute paths, no `cd` | 1 |
