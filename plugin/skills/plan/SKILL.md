---
name: plan
description: Work through a plan file one item a session, with ele carrying the place across /clear; pick up the active plan, write a new one, or convert a spec's open items into plans.
argument-hint: "[new <name> | convert <spec>]"
disable-model-invocation: true
---

# Plan

A plan is a Markdown file under `.documentation/plans/` in the repository whose code it changes. The file is the list and its checkboxes the state; ele stores only which plans are active. `ele plan show` parses and reports, and `ele plan check` lists the warnings `show` counts; every judgement below is yours. A plan is shared, so never write an ele note id or session into it.

The format:

- Front matter: `kind: plan`, `created`, and `spec` when the plan implements a spec in the same repository. Nothing derived, such as counts
- Sections: `## Why`, `## Requirements`, `## Context` (facts, dated), `## Decisions`, `## Actions`, `## Inbox`, `## Out of scope`
- `## Requirements`, optional, numbered, in the user's words: what the plan must deliver, the list the review loop checks the items against. Only the user changes it. A plan written before the section has none
- A Context line not verified against the code or data says "not checked"
- An architectural Decision carries a `rejected: <option>, because …` line for each option it turned down, so an ADR drafted at the close has its considered options
- An item is a `- [ ]` or `- [x]` line and its indented body
  - Each action but the closing one leads with a short lowercase kebab name, `- [ ] judge-blocks: …`, written once and never changed, unique across active and closed plans. `ele plan start` refuses a name another active plan holds or a closed plan used, and an action starting `word: ` is read as named `word`; an Inbox line is never named: its leading `word: ` or date is not a name
  - Each action but the closing one carries a `done when …` line in its body: what will be observable once it is done
  - Progress goes in the body as dated lines, `as of <date>: …` or `next: …`, so a session stopping mid-item needs no handoff
  - A tick carries one line saying what was asserted, at which boundary and at which commit
  - A `blocked:` line holds an item back: `blocked: judge-blocks, mining-reports/report-figures; <reason>` names items in this plan or `<plan>/<name>` in another, `<plan>` being its file name without `.md`, and `ele plan show` frees the item once each is ticked and flags one done, missing, or left unticked by a closed plan; a line with anything but names before its `;` is prose and holds the item until removed by hand
  - `after: 2026-10-22` or `after: 2026-10-22 19:28Z` (UTC) holds an item until then, and `ele plan show` flags it to drop once passed
- The last action folds each Decision through `/ele:plan-audit close <path>`, then what the plan built, and its out-of-scope lines as a `Not doing:` list in the section the plan implemented, into its spec; with no spec, it proposes each out-of-scope line as a `decision` through `/ele:remember`. Then it runs `ele plan done <path>`, which stores the plan's item names and whether each was ticked, and only then deletes the plan, since `done` reads the file. Never tick it while the inbox holds an item; `ele plan check` says when every other action is done, and warns when it is ticked with items there
- `## Inbox` takes work captured mid-session, each item dated with where it came from
- `## Out of scope` holds what triage ruled out, one line each: `- <gist>: <why> (ruled <date>, from <source>)`. A ruled-out item returns only if the plan's why is redrawn

## Asking in rounds

`new` and `convert` settle questions in rounds. A round asks every question whose prerequisites are settled, numbered, each with your recommended answer, worded so that yes accepts it; a question that hangs on one still open waits for a later round, since asked early it is asked again. Facts are yours to find, `ele recall` on the topic first, then the code: ask the user for decisions, and for a fact neither settles. A decision goes in `## Decisions`, with its reason, of each plan that relies on it; a fact about an item goes under that item as a dated line. The rounds end when no question is open and the user confirms the whole; only then does the step that asked them go on. A question `convert` step 1 leaves as a note is not asked

## Grounding

Before the first round of `new` or `convert`, read the repository's ADRs, found by the ADRs section of `reference/plan-practices.md` in the plugin root, the directory holding `skills/`. An accepted one is a settled constraint: a question it answers is not asked, and a wish it rules out is said to be so, superseding it being a question of its own. Another status is context. Before each round, look up the facts its questions rest on, `ele recall` first, then the code; a fact that goes in `## Context` unverified says "not checked".

## new <name>

1. Ground the rounds, then settle in rounds whatever of the why, the requirements, the items and their design questions the conversation has not already settled; each decision it already made goes in `## Decisions` with its reason. Done when the user has confirmed the whole
2. Write `.documentation/plans/<name>.md` from `references/template.md`, with today's date and the plan's own path in the closing action. `## Requirements` holds what the rounds confirmed the plan must deliver, numbered, in the user's words; each architectural Decision its `rejected:` lines. With no spec, drop the `spec` line and replace the closing action's "then what this plan built and its out-of-scope lines as `Not doing:` into the spec" with "propose each out-of-scope line as a `decision` through `/ele:remember`". Done when `ele plan check` on the file lists no warning
3. Run the review loop over the file. Done when the loop's summary is shown and its questions answered
4. Show the file, then on approval run `ele plan start .documentation/plans/<name>.md`. Done when it prints the plan as started, or the user has declined

## convert <spec>

1. List the spec's open items with their sections. Leave out those already in a plan, and those gated on a measurement or an undecided question: they stay notes, a `question` proposed through `/ele:remember` where none holds one. Done when every open item is listed as to plan, already planned, or left as a note
2. Ground the rounds, then check each item's status against the code or data it names rather than the spec's word for it, and ask in rounds about those you cannot check, confirming the spec's words each draft will quote as its requirements. Done when each item's status is checked or ruled by the user, and the requirements confirmed
3. Draft plans grouped by what ships together and split by repository, each written as `new` writes one but not started, its items carrying the spec's detail and its `## Requirements` quoting the spec items it plans, as confirmed. Run the review loop over every draft at once, so the drafts and the spec they name are one corpus: contradictions between items, and artefacts no item produces or consumes. Done when the loop's summary is in hand
4. Propose the plans, each with its items; where every open item went, left-out ones included; and the loop's summary and open questions. Done when the user has answered
5. After the user answers, delete every draft not approved, all of them on a rejection. Apply the user's changes to the rest. Where any draft's items changed, re-audit every remaining draft with `ele:plan-audit`'s Drafts mode, as the loop's Audit step runs it, and put its findings and questions to the user in a round before starting; and in the change that lands them remove the spec's "still to build" and "not built" prose for their items only, leaving the design. Start the plans in this repository; for another, print the `ele plan start` command to run there. Done when no unapproved draft remains, every re-audit finding is answered, and every approved plan is started or its command printed

## Review loop

`new` and `convert` run it over their unstarted drafts, before any is shown for approval: an author reads its own draft as confirmation. At least 2 passes and at most 3. Before pass 1, copy each draft into `BASE=$(mktemp -d)`, for the diff at the end. Each pass:

1. **Audit.** Run the `ele:plan-audit` skill as `drafts <path>...`, every draft's path, marking this checkout's repository ours and current; a draft for another repository, from `convert`'s split, leaves its repository's question to the audit. It adds every active plan of the repository itself and asks nothing. Invoke it afresh every pass: a revision is unaudited until the skill has run over it, and its text loaded in an earlier pass is not a run. Done when its report is in hand
2. **Review.** Launch one `ele:plan-reviewer` agent a draft, all in one message. Brief each by absolute path only: `reviewers/plan.md` beside this file, the reviewer file to follow; the draft; its `spec:`; the user's `CLAUDE.md`, under `$CLAUDE_CONFIG_DIR` when set, else `~/.claude`, and the repository's; each ADR Grounding found; `reference/plan-practices.md`; and from pass 2 the settled file below. Never this session's reasoning, its account of the draft or what changed: the review leans on the reviewer's own reading. Done when every reviewer has reported
3. **Ids.** Number each finding `A<pass>.<n>` for the audit's and `R<pass>.<n>` for the reviewers', in report order across the pass. A finding matching a settled one keeps its id and its level, unless it names evidence that pass did not have. Done when every finding has an id and a level
4. **Verdicts.** Before any edit, give each finding a verdict with its reason: `fixed`, naming the revision; `refuted`, naming the evidence it is wrong; `deliberate`, saying why it stays; or `open`, a question for the user. A finding against a requirement, and a requirement question, is `open`: the loop never edits `## Requirements`, since editing one to clear a finding makes the check circular. Append each to `$BASE/settled.md` with its id, level, verdict and reason. Done when every finding is in the file with its verdict
5. **Revise.** Make the `fixed` revisions, then `ele plan check` each draft. Done when the drafts carry every `fixed` revision and the check lists no warning

After pass 2, stop when no blocker or major is `open` and pass 2 fixed no blocker or major, such a fix being unreviewed until pass 3; else run pass 3, the last. Then show the user every finding as a table of id, level, verdict and reason, minors listed without blocking; ask the `open` ones in a round, by the rules of Asking in rounds; and show `diff -u` of each draft from its copy in `$BASE`. An answer that changes a requirement is the user's edit; one that changes an item is made and shown in a second diff. Done when the user has seen the table and diffs, and every `open` finding is answered

## Pickup

With no argument. Made for a return after days away, so trust nothing the plan says about the code until it is re-derived.

1. **Which plan.** `ele plan show`. With several active it lists each with its counts and next item: ask which, then `ele plan show <path>`. None active: say so and stop. Done when one plan is chosen
2. **Already done?** Check the next item against the commits and uncommitted changes since the plan's last tick: `git log` and `git diff` on the paths the item names. Work found that delivers it: propose the tick with its tick line, write it on approval, and repeat on the new next item until one has no work done. Done when the item in hand has no work found that delivers it
3. **Task card.** Before any work, show the plan's path, the item's title and body, its blocked reasons, and related notes from `ele recall` on two or three of the item's rarer words, since every word must match. Ask to start. A wrong plan is caught here, before work begins. Propose no edit to the plan here: a stale premise seen now waits for the premise check. Done when the user says to start, or names another item
4. **Premise check.** On a yes, before building:
   - Re-derive each claim the item makes about code: cite `file:line`, or mark it "not checked"
   - Re-read each blocked item's reasons and propose removing those now settled
   - Run `ele plan check <path>` and propose a fix for each warning: re-measure a stale `as of` line, drop a done blocker or a passed `after:`, add a missing name or done-when, untick a closing action ticked while another is open, triage an old inbox item at step 5, rewrite a line citing ele data (delete provenance, describe what was observed, move a dependency into the plan)
   - Check that a decision the item relies on lives in one place, and that a status or count it states is dated or derived
   - Write what you find under the item as dated lines
   - Where a finding makes the item's title or a `## Context` line wrong, propose the reword as its own question, the old text beside the new, and write it on approval. The yes to start is not approval for it. Declined, the finding stays a dated line only
   - Done when every claim is cited or marked "not checked", every `ele plan check` warning has a proposed fix, and the findings are written under the item
5. **Inbox.** Triage this plan's inbox by the Inbox triage section of `plan-audit/SKILL.md`, in the skills directory holding this skill's base directory, and apply what the user approves. Done when every inbox item has a proposal and each approved one is applied
6. **Work the item.** It is done when its tick line, per the format, can be written. Then propose the tick with that line; a stop short of it gets a dated progress line instead

A session that compacted mid-item re-reads the item from the file before carrying on.
