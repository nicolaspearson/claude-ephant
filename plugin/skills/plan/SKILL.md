---
name: plan
description: Work through a plan file one item a session, with ele carrying the place across /clear; pick up the active plan, write a new one, or convert a spec's open items into plans.
argument-hint: "[new <name> | convert <spec>]"
disable-model-invocation: true
---

# Plan

A plan is a Markdown file under `.documentation/plans/` in the repository whose code it changes. The file is the list and its checkboxes the state; ele stores only which plans are active. `ele plan show` parses and reports; every judgement below is yours. A plan is shared, so never write an ele note id or session into it.

The format:

- Front matter: `kind: plan`, `created`, and `spec` when the plan implements a spec in the same repository. Nothing derived, such as counts
- Sections: `## Why`, `## Context` (facts, dated), `## Decisions`, `## Actions`, `## Inbox`, `## Out of scope`
- An item is a `- [ ]` or `- [x]` line and its indented body. Each action but the closing one leads with a short lowercase kebab name, `- [ ] judge-blocks: …`, written once and never changed, unique across active plans, and carries a `done when …` line in its body: what will be observable once it is done. `ele plan start` refuses a name another active plan holds, and an item starting `word: ` is read as named `word`. Progress goes in the body as dated lines, `as of <date>: …` or `next: …`, so a session stopping mid-item needs no handoff. A tick carries one line saying what was asserted, at which boundary and at which commit. A `blocked: <reason>` line holds an item back
- The last action folds what the plan built, and its out-of-scope lines as a `Not doing:` list in the section the plan implemented, into its spec; with no spec, it proposes each out-of-scope line as a `decision` through `/ele:remember`. Then it deletes the plan and runs `ele plan done <path>`. Never tick it while the inbox holds an item; `ele plan show` warns only once it is ticked with items there
- `## Inbox` takes work captured mid-session, each item dated with where it came from
- `## Out of scope` holds what triage ruled out, one line each: `- <gist>: <why> (ruled <date>, from <source>)`. A ruled-out item returns only if the plan's why is redrawn

## Asking in rounds

`new` and `convert` settle questions in rounds. A round asks every question whose prerequisites are settled, numbered, each with your recommended answer, worded so that yes accepts it; a question that hangs on one still open waits for a later round, since asked early it is asked again. Facts are yours to find, `ele recall` on the topic first, then the code: ask the user for decisions, and for a fact neither settles. A decision goes in `## Decisions`, with its reason, of each plan that relies on it; a fact about an item goes under that item as a dated line. The rounds end when no question is open and the user confirms the whole; only then does the step that asked them go on. A question `convert` step 1 leaves as a note is not asked

## new <name>

1. Settle in rounds whatever of the why, the items and their design questions the conversation has not already settled; each decision it already made goes in `## Decisions` with its reason
2. Write `.documentation/plans/<name>.md` from `template.md` beside this file, with today's date and the plan's own path in the closing action. With no spec, drop the `spec` line and start the closing action "Propose each out-of-scope line as a `decision` through `/ele:remember`, delete this plan," in place of its fold
3. Show the file, then on approval run `ele plan start .documentation/plans/<name>.md`

## convert <spec>

1. List the spec's open items with their sections. Leave out those already in a plan, and those gated on a measurement or an undecided question: they stay notes, a `question` proposed through `/ele:remember` where none holds one
2. Check each item's status against the code or data it names rather than the spec's word for it, and ask in rounds about those you cannot check
3. Draft plans grouped by what ships together and split by repository, each written as `new` writes one but not started, its items carrying the spec's detail. Audit the drafts with the `ele:ticket-audit` skill, all drafts and the spec as one corpus: contradictions between items, and artefacts no item produces or consumes. Resolve what the audit finds, or carry it as a question
4. Propose the plans, each with its items; where every open item went, left-out ones included; and the audit's open questions
5. After the user answers, delete every draft not approved, all of them on a rejection. Apply the user's changes to the rest, re-auditing a draft whose items changed, and in the change that lands them remove the spec's "still to build" and "not built" prose for their items only, leaving the design. Start the plans in this repository; for another, print the `ele plan start` command to run there

## Pickup

With no argument. Made for a return after days away, so trust nothing the plan says about the code until it is re-derived.

1. **Which plan.** `ele plan show`. With several active it lists each with its counts and next item: ask which, then `ele plan show <path>`. None active: say so and stop
2. **Already done?** Check the next item against the commits and uncommitted changes since the plan's last tick: `git log` and `git diff` on the paths the item names. Work found that delivers it: propose the tick with its tick line, write it on approval, and repeat on the new next item until one has no work done
3. **Task card.** Before any work, show the plan's path, the item's title and body, its blocked reasons, and related notes from `ele recall` on two or three of the item's rarer words, since every word must match. Ask to start. A wrong plan is caught here, before work begins. Propose no edit to the plan here: a stale premise seen now waits for the premise check
4. **Premise check.** On a yes, before building:
   - Re-derive each claim the item makes about code: cite `file:line`, or mark it "not checked"
   - Re-read each blocked item's reasons and propose removing those now settled
   - Check that a decision the item relies on lives in one place, and that a status or count it states is dated or derived
   - Write what you find under the item as dated lines
   - Where a finding makes the item's title or a `## Context` line wrong, propose the reword as its own question, the old text beside the new, and write it on approval. The yes to start is not approval for it. Declined, the finding stays a dated line only
5. **Inbox.** Propose for each inbox item: move it into `## Actions`, rule it out of scope with its line, adding `## Out of scope` as the plan's last section if it has none, delete it as a duplicate or already done, or leave it. Apply what the user approves
6. **Work the item.** It is done when its tick line, per the format, can be written. Then propose the tick with that line; a stop short of it gets a dated progress line instead

A session that compacted mid-item re-reads the item from the file before carrying on.
