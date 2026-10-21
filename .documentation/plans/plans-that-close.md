---
kind: plan
created: 2026-10-21
spec: .documentation/specs/design.md
---

# Plans that close

## Why

ele plans stay accurate and close with little overhead. A plan's state (its frontier, waits, stale claims and inbox) is worked out and checked by ele rather than kept by hand, and plans replace tracker maps for planning, so no tickets rot on a platform. As of 2026-10-21 four plans are active and closing them is the struggle: the mining-reports audit found an item no `blocked:` line held back and a wait that no item would ever end, and each tick or progress line costs a full checkpoint.

## Context

- As of 2026-10-21: an item is known by its line number and checkbox text only (`Item` in `internal/plan`), and its `blocked:` reasons are kept as plain strings
- As of 2026-10-21: the audit ledger keys a plan item by repository, plan path and checkbox text, so a reworded item loses its rulings; the mining-reports audit and its checkpoint reworded 4 of its 5 bullets
- As of 2026-10-21: `ele plan` has `done`, `list`, `show` and `start`
- As of 2026-10-21: a checkpoint over a 22-line diff that touched only plan files ran small mode, `task check FORCE=1` twice and a re-review
- As of 2026-10-21: `design.md`'s Plans section says "Not a tracker: no item ids, no dependency graph" and that a `blocked:` reason is prose ele never resolves; this plan's names and edges reverse both, which the closing fold rewrites
- As of 2026-10-21: taken from mattpocock-skills 1.3.1's wayfinder: a destination fixing scope, items referred to by name, the plan as an index with each decision in one place, and a frontier worked out from blocking edges

## Decisions

- The plan is the source of truth for planning: ele writes no tracker tickets, so none rot on a platform. ticket-audit still audits tracker maps that exist, as in work repositories whose team uses one
- Each item leads with a short kebab name after its checkbox, `- [ ] judge-blocks: …`, written once and never changed, unique across active and closed plans: it reads as a name, grep finds it, and the ledger keys on it, so rewording keeps rulings
- `blocked:` leads with item names, `blocked: judge-blocks` or `blocked: mining-reports/judge-blocks`, an optional reason after them; `after: <date>` holds an item until then. `ele plan show` works out the frontier from both and flags a blocker that is done or does not exist. A prose reason alone is no edge
- `ele plan done` stores the closed plan's item names and states, so a cross-plan edge still resolves after the plan is deleted, done if its item was ticked, and names stay unique
- Mechanical checks live in the binary, in `ele plan show` and a new `ele plan check`, so every session gets them free: a blocker done or missing, an `after:` date passed, an `as of` line past an age, inbox age, an action with no done-when, readiness to close. Judgment lives in a new `ele:plan-audit` skill on ticket-audit's engine, and ticket-audit's `PLAN.md` moves into it
- A checkpoint whose change list holds only `.documentation/plans/*.md` runs `mode: plan`: no gate, since plan files cannot fail one, and no bugs reviewer; `ele plan check`, then one `ele:checkpoint-reviewer` on sonnet with the spec and conventions-prose files and a check that each claim holds, and a re-review only where a fix changes a claim. A change that only flips checkboxes and adds at most one tick, `as of` or `next:` line per item gets no reviewer: `ele plan check` and the tick judged inline. Decided 2026-10-21 over a reviewer-free plan mode: that day's plan-only checkpoint had six findings confirmed, four of them, a missing folder, an overstated count, an ambiguous date and a misattributed figure, out of any mechanical check's reach
- Work past a plan's why leaves it at once as a question note starting `Plan candidate:`, which plan-audit lists grouped by theme; no section in the plan holds it, so nothing past the why blocks a close
- No "not yet specified" section, as wayfinder's map has: the inbox also holds work not sharp enough to be an item, and triage leaves it there until it is
- plan-audit proposes a home for each open question note, a plan's inbox, a plan candidate, or staying a question, and supersedes a note that moves
- Every action carries a `done when` line, which the plan skill writes; `ele plan check` warns on one without it and never refuses. The closing action, fixed by the template, carries neither a name nor a done-when: nothing blocks on it, and its text is its done-when
- Items in the active plans get names in the item-ids item, losing mining-reports' five ledger records keyed on bullet text: all five rulings were applied, so none is pending
- config.md stays active beside this plan; whether ele limits active plans is for plan-audit's item to settle

## Actions

- [x] item-ids: items lead with a kebab name, parsed by `internal/plan`, unique across active plans; the plan skill and `template.md` write a name and a `done when` line for each action but the closing one; the ledger keys on the name where there is one; the active plans' items named
  - done when `ele plan show` prints each item's name, a duplicate name across active plans is refused with both paths, a reworded item keeps its ledger rulings in a test seen red, every active plan's items but the closing ones carry names, and the plan skill and template say to write a name and a done-when
  - asserted in the item 1 commit: ele plan show prints names (built binary over the active plans); a name twice in a plan fails Parse, and plan start refuses one another active plan holds with both paths; a reworded named item keeps its ledger rulings (corpus→record→reword→rulings test); every new test seen red; task check FORCE=1 green
- [x] edges: `blocked:` leading with item names, local or `<plan>/<name>`, and `after: <date>`; the frontier worked out from them; a blocker done or missing flagged; `ele plan done` stores the plan's item names and states, and a name a closed plan used is refused
  - done when tests seen red show a blocked item leaving the frontier once its blocker is ticked, one held by `after:` until its date, a blocker in a closed plan resolving from the store, a missing one flagged, and a name a closed plan used refused; and the active plans' `blocked:` lines name items
  - asserted in the edges commit: blocked: names resolve locally and as <plan>/<name> against active plans, then closed plans' stores; after: holds by date or UTC minute; done or missing or abandoned blockers flagged; ele plan done stores names and states (before deletion; a deleted plan falls back to git and warns); a closed plan's names refused at start; 14 mutations red; active plans' blocked: lines name items; task check FORCE=1 green
- [x] plan-check: `ele plan check [<path>]`, warnings also summarised by `ele plan show`: a blocker done or missing, an `after:` date passed, an `as of` line past an age settled in the item, inbox items past an age, an action with no done-when other than the closing one, a plan ready to close
  - settled 2026-10-21 with the user: an `as of` line warns past 14 days, outside done items and `## Out of scope`; an inbox item past 7 days from its leading date, and an undated one always; ready to close means every action but the closing one is ticked, saying to triage the inbox first when it holds items; ruled at checkpoint: done actions need no done-when, and check also lists name clashes
  - done when each warning has a test seen red, the closing action raises none and `ele plan check` run over the active plans prints what they hold
  - asserted in the item 3 commit: ele plan check lists each warning with its line (built binary over the 4 active plans: 0, 4, 1 and 3 warnings); show and each summary line count them; each warning and the closing-action exemption has a test seen red, 27 mutations red; task check FORCE=1 green
- [ ] light-checkpoint: a change list of only `.documentation/plans/*.md` runs checkpoint's `mode: plan` as Decisions say, with its tick-only case; `ele checkpoint log` takes `mode: plan` and `/ele:checkpoint-stats` reports it
  - done when real checkpoints from the installed plugin run `mode: plan` over a plan-only change with one sonnet reviewer and no gate, and over a tick-only change with no reviewer, each logged; and a change list with one other file still runs small or full
- [ ] plan-audit: an `ele:plan-audit` skill on ticket-audit's engine, `PLAN.md` moved into it, over every active plan by default: a corpus taking several plans, waits that nothing ends, inbox triage, `Plan candidate:` notes grouped by theme, a home proposed for each open question note, the newest handoff checked against the plans, readiness to close, whether ele limits active plans; the plan skill's inbox triage moving work past the why out as a `Plan candidate:` note; and exit conditions for the plan skill's steps
  - captured 2026-10-21 from skill-lessons' inbox: /ele:plan asks ticket-audit for all drafts and the spec as one corpus, but `ele ticket corpus --plan` takes one file
  - captured 2026-10-20 from skill-lessons' inbox: every skill step ends with an exit condition; as of 2026-10-20 the plan skill's Pickup step 6 has one
  - done when a real run over the active plans proposes each finding class it names, ticket-audit no longer holds `PLAN.md`, the plan skill's triage names `Plan candidate:` notes and each of its steps ends with an exit condition, and whether ele limits active plans is in Decisions
- [ ] Fold what this plan built, and its out-of-scope lines as `Not doing:`, into the spec, run `ele plan done .documentation/plans/plans-that-close.md`, and delete this plan

## Inbox

- 2026-10-21, from the user while this plan's first checkpoint ran: working items one at a time takes long; group frontier items that fit one session, each keeping its own done-when and tick line, so pickup offers a group and one checkpoint judges and ticks each item on its own requirements
- 2026-10-21, from the user's question whether finished plans become specs: the fold carries only decisions with their reasons, `Not doing:` lines and constraints the code cannot show; it drops "Built:" narration and progress lines, which the code and git history hold, and rewrites any spec line the plan made untrue. Open: whether pruning `design.md`'s existing plan residue is this plan's fold or a plan candidate
- 2026-10-21, from the user: `/ele:plan new` audits its draft before `ele plan start`, as `convert` does: contradictions between decisions, actions and the spec, each decision traced to an action, each artefact's producer and consumer, Context lines re-derived; not a gate in the binary, which cannot judge prose. This plan started unaudited and its checkpoint found 18 findings, most of those classes
- 2026-10-21, from item-ids' checkpoint: `ele plan start` skips an active plan missing from the checkout, on another branch, so a name clash with it passes silently; say that plan could not be checked
- 2026-10-21, from item-ids' checkpoint: the ticket corpus reads a name from an Inbox item too, while `internal/plan` names only Actions; key on names in Actions only
- 2026-10-21, from the user's question whether plan-audit names older plans: as of 2026-10-21 only this repository's active plans carry item names, none of the nine plans in ~/Work's other repositories; proposed: `ele plan check` warns on an unnamed action other than the closing one, and plan-audit proposes a name for each, written on approval
- 2026-10-21, from the user's question whether a plan-grill skill is needed: no new skill; `/ele:plan new`'s rounds question a draft harder, drawing on mattpocock-skills' grilling, beside the draft audit before `ele plan start` already in this inbox
- 2026-10-21, from the user, who wants a clean repository once work ends: `ele plan done` archives the whole plan text beside its stored item names, readable through `ele recall` or `ele plan show --closed <name>`; the closing fold then carries only what changes the spec's design, a decision's reasons staying in the archive. Plans stay files in the repository while active

## Out of scope
