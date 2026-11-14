---
kind: plan
created: 2026-11-13
---

# Plan candidates in the start block

## Why

A plan candidate is an open `question` note whose text starts `Plan
candidate`, and the start block lists open questions newest first until its
budget cuts the oldest. With dozens of open questions a project's candidates
fall in the cut, so work set aside as a candidate is never seen again, and
while shown they take room from the real questions. The start block should
count candidates in one line and give their room back to the questions.

## Requirements

1. The start block shows plan candidates as one line with their count and
   newest date, naming `/ele:plan-audit` and `ele context`
2. Candidates no longer take room from the open-question list
3. The prefix forms in use today are recognised; the skills that write
   candidates name one form, `Plan candidate:`

## Context

- As of 2026-11-13: in claude-ephant, 53 open questions, 17 of them
  candidates: 8 start `Plan candidate:` and 9 start `Plan candidate
  (<topic>),`; one live horizon candidate starts `Plan candidate, <date>`,
  a third form. The start block listed 15 of the 53, from 2026-11-06 on, their
  lines about 95% of the block's bytes; hyve-acquisition-gateway-api held 97
  open questions, 14 of them candidates, and horizon 46, 7 of them candidates
- As of 2026-11-13: `internal/notes`' `problem` refuses a kind outside
  `notes.Kinds`, so an `ele` that predates a kind reads its notes as invalid,
  as `.documentation/specs/design.md` records for the `flag` kind
- As of 2026-11-13: `cmd/ele` `liveNotes` sends every live `question` note to
  the questions list and the rest of the shown kinds to the others list, both
  newest first; `writeNotes` adds questions while they fit the room left under
  `blockBudget`, 9,500 UTF-16 units, then others under `contextBudget`, 6,000
  bytes, and its closing line counts "older questions not shown, ele context
  lists them". `ele context` reads `liveNotes` too; `ele status` counts note
  kinds itself, a question as open or answered
- As of 2026-11-13: no Go code reads the `Plan candidate` text; the
  convention lives in `plugin/skills/plan-audit/SKILL.md` (Inbox triage, Plan
  candidates, Open questions), `plugin/skills/wrap/SKILL.md` (Deferred work),
  `.documentation/specs/design.md` (Plans) and nowhere in `docs/`
- As of 2026-11-13: `writeNotes`' header line counts toward the budget it
  shares with the notes, so a line added to it is never cut

## Decisions

- A candidate stays a `question` note. `cmd/ele` recognises one when its
  text, trimmed, starts with `plan candidate` in any case, then optional
  spaces and `:`, `(` or `,`, which takes the three forms in use and no
  question that only mentions the phrase; plan-audit's Plan candidates
  section states the same rule. New candidates are written `Plan candidate:`
  - rejected: a `candidate` note kind, migrated from the prefix on sync, with
    its own `ele context` block, `ele status` split and `ele plan start
    --supersedes`, because an older `ele` reads every candidate as invalid,
    which this repository's own `task upgrade` after `task install` does, and
    a plugin from `main` ahead of the binary leaves plan-audit finding none
- The start block lists no candidate and shows one line first under the notes
  header, `<n> plan candidates, newest <date>; /ele:plan-audit groups them,
  ele context lists them.`, omitted when there are none; the closing line's
  "older questions not shown" counts real questions only. It counts every live
  candidate the block would have listed as a question, whatever its scope, and
  with candidates as the only notes the header and the line still print
  - rejected: the line before the closing line, because the first line of the
    notes is where it is read
- `ele context` and `ele status` are unchanged: `ele context` still lists
  every candidate among the open questions, and `ele status` still counts them
  as open questions, so the two counts differ; count-line records the
  difference in the spec's SessionStart injection section

## Actions

- [x] count-line: `cmd/ele` gets `isPlanCandidate`, and `writeNotes` moves
  candidates out of the questions into the count line under its header; the
  same change states the match in plan-audit's Plan candidates section and
  updates `.documentation/specs/design.md`'s SessionStart injection and Plans
  sections and `docs/notes-and-recall.md`'s "What a session starts with"
  - done when `writeNotes` tests assert, each case seen red under the
    mutation named for it:
    - given candidates in all three forms, case variants and two questions
      that only mention the phrase: the exact count line with the newest
      candidate's date, and no candidate line (red with `isPlanCandidate`
      returning false)
    - with a room small enough that the candidates would crowd out a
      question: every real question shown, and an "older questions not shown"
      count without candidates (red with `isPlanCandidate` returning false)
    - with candidates as the only notes: the header and the count line (red
      with `isPlanCandidate` returning false)
    - the two questions that only mention the phrase listed as questions (red
      with `isPlanCandidate` matching on `strings.Contains`)
    - no count line with no candidates (red with the line printed at a count
      of 0)
  - and, outside the tests, the clone build, `echo '{"source":"resume"}' | go
    run ./cmd/ele session-start --no-sync` in this repository, prints the
    count line and no candidate's text
  - asserted in the count-line commit: in `cmd/ele` tests,
    TestWriteNotesCountsPlanCandidates (three forms, case variants, two
    mentions listed as questions), TestWriteNotesCountsPlanCandidatesAlone and
    TestWriteNotesKeepsQuestionsOverPlanCandidates went red with
    `isPlanCandidate` returning false, the first also with it matching on
    `strings.Contains`, and TestWriteNotesNamesEachNotesSession went red with
    the line printed at a count of 0; `go run ./cmd/ele session-start
    --no-sync` in this repository printed `16 plan candidates, newest
    2026-11-12` under the notes header and no candidate's text
  - as of 2026-11-14, premise check: `liveNotes` routes as Context says
    (`cmd/ele` `liveNotes`); `writeNotes` adds questions against the UTF-16
    room only, never `contextBudget`; before this item its header became "ele
    has no notes" when questions and others were both empty, so that test had
    to count candidates too; the header's length seeds `written` and `units`,
    and `render` always writes it, so a count line written beside it is never
    cut; `ele context` reads `liveNotes`, `ele status` counts kinds itself;
    before this item no Go code read the `Plan candidate` text
  - as of 2026-11-14: claude-ephant holds 16 live candidates, 9 `Plan
    candidate (` and 7 `Plan candidate:`, none with a comma, newest
    2026-11-12; the horizon comma form not checked
- [ ] Fold each Decision through `/ele:plan-audit close
  .documentation/plans/candidates-in-start-block.md`, propose each
  out-of-scope line as a `decision` through `/ele:remember`, run `ele plan
  done .documentation/plans/candidates-in-start-block.md`, and delete this plan

## Inbox

## Out of scope

- A `candidate` note kind: an older `ele` would read every candidate as
  invalid; worth it only once a typed feature needs it (ruled 2026-11-13, from
  this plan's design round)
- Grouping candidates by theme in `ele`: plan-audit groups them (ruled
  2026-11-13, from this plan's design round)
