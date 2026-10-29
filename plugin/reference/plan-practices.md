# Plan practices

The planning, spec and decision-record practices of published workflows, read
from their primary sources as of 2026-10-29, each with ele's verdict:

- **Adopted**: a plan Decision or the user's ruling sets it; it names the
  item that will apply it until a skill section does, or the Inbox line
  holding it
- **Proposed**: no Decision sets it; the item named rules on it before
  applying it
- **Rejected**: with the reason
- **Already**: naming where ele does it today

Written for the plan reviewer, `/ele:plan new` and the closing fold to read.
It covers the practices that bear on how a plan is written, reviewed and
closed and how a decision is recorded. Left out: each tool's mechanics, such as
its CLI flags, length bounds, file names, placeholders and editor keys, and
code-review practices, such as code smells, which are `ele:checkpoint`'s and
`ele:review`'s. A source moves, so re-read it before quoting
it again: Nygard's post was read through a summariser, not as raw text, and
IEEE 1028's anomaly classes are paywalled and not used here.

## Sources

- OpenSpec: [concepts](https://github.com/Fission-AI/OpenSpec/blob/main/docs/concepts.md),
  [writing specs](https://github.com/Fission-AI/OpenSpec/blob/main/docs/writing-specs.md),
  [reviewing changes](https://github.com/Fission-AI/OpenSpec/blob/main/docs/reviewing-changes.md),
  [CLI](https://github.com/Fission-AI/OpenSpec/blob/main/docs/cli.md),
  [commands](https://github.com/Fission-AI/OpenSpec/blob/main/docs/commands.md),
  [verify skill](https://github.com/Fission-AI/OpenSpec/blob/main/skills/openspec-verify-change/SKILL.md),
  [validator](https://github.com/Fission-AI/OpenSpec/blob/main/src/core/validation/constants.ts)
- GitHub spec-kit: [command prompts](https://github.com/github/spec-kit/tree/main/templates/commands),
  [spec template](https://github.com/github/spec-kit/blob/main/templates/spec-template.md),
  [plan template](https://github.com/github/spec-kit/blob/main/templates/plan-template.md),
  [tasks template](https://github.com/github/spec-kit/blob/main/templates/tasks-template.md),
  [constitution template](https://github.com/github/spec-kit/blob/main/templates/constitution-template.md)
- ADRs: [Nygard, 2011](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions),
  [MADR template](https://github.com/adr/madr/blob/develop/template/adr-template.md),
  [MADR docs](https://github.com/adr/madr/blob/develop/docs/index.md),
  [adr.github.io](https://adr.github.io/)
- mattpocock/skills: [grilling](https://github.com/mattpocock/skills/blob/main/skills/productivity/grilling/SKILL.md),
  [to-spec](https://github.com/mattpocock/skills/blob/main/skills/engineering/to-spec/SKILL.md),
  [to-tickets](https://github.com/mattpocock/skills/blob/main/skills/engineering/to-tickets/SKILL.md),
  [implement-spec](https://github.com/mattpocock/skills/blob/main/skills/engineering/implement-spec/SKILL.md),
  [domain-modeling](https://github.com/mattpocock/skills/blob/main/skills/engineering/domain-modeling/SKILL.md),
  [ADR format](https://github.com/mattpocock/skills/blob/main/skills/engineering/domain-modeling/ADR-FORMAT.md)
- Kiro: [specs](https://kiro.dev/docs/specs/),
  [feature specs](https://kiro.dev/docs/specs/feature-specs/),
  [analyze requirements](https://kiro.dev/docs/specs/analyze-requirements/),
  [best practices](https://kiro.dev/docs/specs/best-practices/);
  EARS from [Alistair Mavin](https://alistairmavin.com/ears/)
- BMAD: [PRD validation checklist](https://github.com/bmad-code-org/BMAD-METHOD/blob/main/skills/bmad-prd/assets/prd-validation-checklist.md),
  [PRD validate](https://github.com/bmad-code-org/BMAD-METHOD/blob/main/skills/bmad-prd/references/validate.md),
  [adversarial lens](https://github.com/bmad-code-org/BMAD-METHOD/blob/main/skills/bmad-review/references/lens-adversarial.md)
- Claude Code: [best practices](https://code.claude.com/docs/en/best-practices),
  [plan mode](https://code.claude.com/docs/en/permission-modes#analyze-before-you-edit-with-plan-mode)
- Review labels: [Conventional Comments](https://conventionalcomments.org/),
  [Google eng-practices](https://google.github.io/eng-practices/review/reviewer/comments.html)

## Shape of a plan

- **A change carries its why, design and tasks together** (OpenSpec's change
  folder; Kiro's requirements, design and tasks files). Already: one plan
  file holds `## Why`, `## Decisions` and `## Actions`
- **One intent you can say in a sentence** (OpenSpec). Already: `## Why`
- **Size limits**: a requirement over 500 characters, more than 10 deltas a
  change (OpenSpec's validator). Rejected: a count says nothing a reviewer
  reading the plan does not see
- **Structural validation** (OpenSpec `validate`). Already: `ele plan check`
- **Fixed sections, a section that does not apply removed** (spec-kit).
  Rejected: ele's sections stay, empty if need be, since `ele plan check` and
  `show` parse them
- **Mandatory spec sections**: user scenarios, success criteria,
  assumptions, edge cases, key entities (spec-kit). Rejected: a spec's shape
  is its repository's; a plan holds Why, Context, Decisions and Actions
- **Problem, solution, implementation and testing decisions, out of scope**
  (to-spec). Already: `## Why`, `## Decisions`, done-when lines,
  `## Out of scope`
- **Requirements first or design first** (Kiro). Rejected as a fixed order:
  rounds settle each question once its prerequisites are, in either order
- **Plan mode: research and propose without editing** (Claude Code).
  Rejected as the vehicle: plan mode lasts a session, and a plan file
  outlives it; `new` writes nothing but the plan before the start
- **A plan's specs kept in the repository** (Kiro). Already:
  `.documentation/plans/` and `.documentation/specs/`

## Requirements

- **Numbered requirements** (spec-kit `FR-001`). Adopted by review-loop: a
  plan's `## Requirements` is numbered, so the coverage table can name each
- **Normative form**: SHALL or MUST with RFC 2119 strengths (OpenSpec's
  validator), `System MUST` (spec-kit), EARS's six patterns (Kiro documents
  one). Rejected: requirements stay in the user's words, and a rewrite into
  a dialect is an edit the user did not make
- **User stories, long and numbered, prioritised P1 to P3, each
  independently testable** (to-spec, spec-kit). Rejected: requirements are in
  the user's words; order is file order and `blocked:` edges
- **Observable requirement**: "could a tester who's never seen the code tell
  whether it passed?" (OpenSpec). Adopted by plan-reviewer: its test for an
  untestable requirement or done-when line
- **Success criteria measurable, technology-agnostic and verifiable**
  (spec-kit). Adopted by plan-reviewer: a done-when line not observable is a
  finding
- **Keep the how out of requirements** (OpenSpec). Proposed for
  plan-reviewer: a requirement naming an implementation reported as a
  question
- **Scenarios**, GIVEN/WHEN/THEN under each requirement (OpenSpec, spec-kit,
  whose `/converge` keys on them). Rejected: an item's `done when` line
  carries what is observable, and a second form beside it would drift
- **No file paths in a spec** (to-spec). Rejected for plans: an item names
  the code it changes, built within days, and pickup re-derives each claim;
  Context lines are dated

## Clarification

- **Questions in rounds over the frontier**, numbered, each with a
  recommended answer that yes accepts; facts are the agent's, decisions the
  user's; done when the frontier is empty and the user confirms (grilling).
  Already: the plan skill's Asking in rounds
- **A question budget**: at most 3 `[NEEDS CLARIFICATION]` markers in
  `/specify`, ranked scope over security over experience over technical, and
  5 questions in `/clarify`, one at a time (spec-kit). Rejected: a cap turns
  the rest into guesses the plan then rests on; rounds ask only what is open
- **Questions ranked by impact times uncertainty** (spec-kit `/clarify`).
  Rejected with the budget: rounds ask every open question, so none is ranked
  out
- **Informed guesses recorded as assumptions in place of questions**
  (spec-kit `/specify`). Rejected: facts are looked up, a fact not checked
  is marked "not checked", and a decision is the user's
- **Open questions marked inline in the spec, and planning refused while
  one is open** (spec-kit `/specify`, `/plan`). Rejected for the marking: a
  plan starts with no question open; one raised later goes to `## Inbox`.
  Already, the refusal: `new` writes nothing until the rounds end
- **Skipping clarification warns of rework** (spec-kit `/clarify`). Rejected:
  `new` has no skip; its rounds end only when the user confirms the whole
- **Clarifications logged** as dated `Q → A`, replacing the statement they
  contradict (spec-kit `/clarify`). Already: a decision goes in `## Decisions`
  with its reason, in one place, which pickup checks
- **A completion report**: each category resolved, deferred, clear or
  outstanding (spec-kit `/clarify`). Already: rounds end when no question is
  open and the user confirms the whole
- **A fixed coverage taxonomy** for gaps (spec-kit `/clarify`). Rejected: its
  categories fit product features; the reviewer checks against the plan's
  requirements, spec and conventions instead
- **Synthesise without interviewing** (to-spec). Rejected: `new` settles
  every open question with the user before writing
- **Requirement analysis returned as questions with suggested fixes**, the
  user picking, answering or dismissing; its categories logical
  inconsistencies, ambiguities, conflicting constraints, unstated assumptions
  and missing edge cases (Kiro). Adopted by plan-reviewer and review-loop: an
  ambiguous or untestable requirement, or a finding against one, goes to the
  user as a question; an unverified Context line says "not checked"

## Review

- **A fresh reviewer that has not seen the author's reasoning** (Claude
  Code). Adopted by plan-reviewer: its own agent, briefed by path only
- **Review before building and again after** (OpenSpec's two moments, the
  first before `/opsx:apply`, the second `/opsx:verify`). Adopted by
  review-loop for the first; already, the second: `ele:checkpoint`
- **Review against the spec, report what affects correctness, skip style**;
  "a reviewer that's asked to find problems will always find *something*"
  (Claude Code). Adopted by plan-reviewer and review-loop: findings carry a
  severity, and each gets a verdict before any edit
- **A finding quota**, "find at least ten issues", and "if you have zero
  findings, re-check" (BMAD adversarial lens). Rejected: it manufactures
  findings, the failure Claude Code warns of
- **Two review axes, standards and spec** (mattpocock code-review). Already:
  `ele:checkpoint`'s conventions and spec reviewers; for a plan, adopted by
  plan-reviewer, which checks conventions and requirements
- **Simplicity and anti-abstraction gates** (spec-kit's spec-driven.md).
  Rejected as fixed gates: a repository's CLAUDE.md states its own, which the
  reviewer reads
- **Read in order and stop at the first wrong part**: proposal, deltas,
  design, tasks (OpenSpec). Rejected: each pass costs a fresh reviewer, so it
  reports everything it finds
- **Read-only analysis; remediation offered, never applied** (spec-kit
  `/analyze`). Adopted by drafts-mode: no edits, no ledger writes
- **Duplication, ambiguity, underspecification and inconsistency passes**
  (spec-kit `/analyze`). Adopted by plan-reviewer, for a draft against its
  requirements, spec and ADRs, and by drafts-mode, for items against each
  other and other plans
- **Requirement-to-task coverage table and coverage %**, with requirements
  that have no task and tasks that have no requirement (spec-kit `/analyze`;
  OpenSpec's "a task with no matching requirement" and "a task that touches
  something outside the scope you just approved"). Adopted by plan-reviewer:
  the coverage table, and an item no requirement asks for
- **"The most valuable catch of all: what's missing"** (OpenSpec). Adopted by
  plan-reviewer: requirements with no item, and missing practices
- **A conventions gate before research and again after design** (spec-kit
  `/plan`). Adopted by plan-reviewer and review-loop: each pass's reviewer
  reports an item breaking a CLAUDE.md rule or contradicting an accepted ADR
- **A kept violation justified with the simpler alternative rejected**
  (spec-kit's Complexity Tracking). Adopted by review-loop: the verdict
  deliberate, with its reason
- **A versioned constitution with an impact report per amendment**
  (spec-kit). Rejected: conventions live in CLAUDE.md files, whose history is
  git
- **A quality loop capped at 3 iterations, what still fails reported**
  (spec-kit `/specify`). Adopted by review-loop: 2 to 3 passes, what is open
  after the last going to the user
- **Stable finding ids across reruns** (spec-kit `/analyze`). Proposed for
  review-loop: a later pass's brief carries earlier findings with their
  verdicts, which an id would key
- **A findings cap**, 50 (spec-kit `/analyze`). Rejected: the pass cap bounds
  the loop, and a cap hides the findings past it
- **Findings with location and a suggested fix** (BMAD). Adopted by
  plan-reviewer: each finding with severity and the line it rests on
- **Dimension grades and an overall grade**, strong to broken, excellent to
  poor (BMAD). Rejected: a grade beside the severity scale says the same
  twice; the loop's gate reads severities
- **Checklists as unit tests for requirements**, "if your spec is code
  written in English, the checklist is its unit test suite", at least 80% of
  items citing a spec section (spec-kit `/checklist`). Rejected: the
  reviewer's coverage table and done-when checks do that work, so neither the
  checklist nor `/implement`'s gate on its unticked items is needed; a generated
  checklist per plan would repeat them
- **Severity when unsure: prefer the lower level** (OpenSpec verify).
  Proposed for plan-reviewer, with its scale
- **Severity ranks impact, not the size of the fix** (BMAD). Adopted by
  plan-reviewer: the plan's Decisions define each level by its effect
- **Label what does not block, so not every comment reads as mandatory**
  (Google eng-practices; Conventional Comments' `(non-blocking)`). Adopted by
  review-loop: minors listed without blocking. Already in `ele:review`, whose
  voice file uses Conventional Comments labels
- **Completeness, correctness and coherence after building; a skipped check
  never scored as passing** (OpenSpec verify). Already: `ele:checkpoint`'s
  reviewers, and a check that cannot run reported unverified

## Severity scales seen

| Source | Levels | Gate |
| --- | --- | --- |
| spec-kit `/analyze` | CRITICAL, HIGH, MEDIUM, LOW | CRITICAL resolved before implement; LOW and MEDIUM may proceed |
| OpenSpec verify | CRITICAL, WARNING, SUGGESTION | "Must fix before archive", "Should fix", "Nice to fix" |
| BMAD PRD | critical, high, medium, low | none per level |
| OpenSpec `validate` | error, warning | an error fails validation; a warning does not |
| Conventional Comments | `(blocking)`, `(non-blocking)`, `(if-minor)` | blocking holds acceptance until resolved; if-minor only if the fix is small |
| Google eng-practices | Nit, Optional, FYI | none; unlabelled means required |

spec-kit's definitions, verbatim:

- CRITICAL: "Violates constitution MUST, missing core spec artifact, or
  requirement with zero coverage that blocks baseline functionality"
- HIGH: "Duplicate or conflicting requirement, ambiguous security/performance
  attribute, untestable acceptance criterion"
- MEDIUM: "Terminology drift, missing non-functional task coverage,
  underspecified edge case"
- LOW: "Style/wording improvements, minor redundancy not affecting execution
  order"

plan-reviewer writes ele's scale here, as the plan's Decisions set it.

## Items

- **Each item states how it is verified** (OpenSpec tasks). Already: each
  action's `done when` line, and the tick line saying what was asserted at
  which boundary
- **Demoable or verifiable on its own** (to-tickets). Already: the done-when
  line says what will be observable
- **Sized for one session, or one fresh context window** (OpenSpec,
  to-tickets); "one giant 'implement the feature' task" a red flag
  (OpenSpec). Already: one item a session, ele carrying the place across
  `/clear`
- **Specific enough to complete without other context** (spec-kit
  `/tasks`). Already: a session after `/clear` works from the item alone, and
  `convert` carries the spec's detail into it
- **A constraint quoted verbatim into the task** (spec-kit `/tasks`).
  Already: `convert`'s items carry the spec's detail
- **Tests and docs land in the group whose work needs them, not a catch-up
  group** (OpenSpec). Already: a behaviour change updates its spec and
  `docs/` page in the same change, and done-when lines name them
- **Tests written first and seen failing** (spec-kit `/tasks`). Rejected as
  an order: what counts is a test that can fail, so `ele:checkpoint` sees
  every new or changed test go red once by mutating the code it covers
- **Test seams sketched first, the highest one** (to-spec). Rejected for
  plans: a test design choice, made when the item is built
- **Blocking edges between items, worked as a frontier** (to-tickets,
  implement-spec). Already: `blocked:` and `after:`, the next item derived
- **Phases: setup, foundation, one per story, polish, each ending in a
  checkpoint** (spec-kit). Rejected: `blocked:` edges carry order, and each
  item ends in its own checkpoint
- **Prefactoring first; expand-contract for a wide refactor** (to-tickets).
  Rejected as rules: ordering and technique are the author's, set in the
  rounds and held by `blocked:` edges
- **Vertical slices through every layer** (to-tickets). Rejected as a review
  rule: slice shape is the plan author's call, settled in the rounds
- **Numbered tasks**, `1.1` (OpenSpec) or `T001` with a story label in
  place of a requirement id (spec-kit). Rejected: items carry kebab names,
  which survive rewording; the reviewer's coverage table traces requirements
- **Tickets with "Blocked by" and acceptance criteria** (to-tickets).
  Already: `blocked:` and `done when` lines
- **Tasks grouped under headings** (OpenSpec). Already: `###` subheadings
  inside `## Actions` count, in file order
- **Parallel markers and waves** (spec-kit `[P]`, Kiro; implement-spec's
  worktrees). Rejected: running items in parallel changes how a plan runs,
  and is a plan of its own
- **Iterate on the breakdown until the user approves** (to-tickets). Already:
  rounds, then the user confirms the whole
- **Remaining work appended, never rewriting existing tasks** (spec-kit
  `/converge`). Already: mid-session work goes to `## Inbox`, items keep their
  names
- **Tasks re-synced to changed requirements** (Kiro's Sync Files). Rejected:
  a plan amended mid-way is reviewed by pickup's premise check and the
  checkpoint, not re-synced
- **A reviewer, not the author, judges done** (spec-kit `/checklist`, whose
  `[x]` a reviewer sets on a requirements-quality item). Already: no session
  judges its own tick; the checkpoint's reviewers check it
- **Tasks ticked as completed; evidence for done, the command run and its
  output** (OpenSpec, spec-kit `/implement`, Claude Code). Already: the tick
  line
- **Build in a fresh session from the written spec** (Claude Code). Already:
  one item a session after `/clear`, from the plan file
- **Explicit approval between phases** (Kiro's standard specs). Already:
  `new` shows the plan and starts it on approval

## Closing and archive

- **Fold the change into the living spec when done** (OpenSpec archive).
  Already: the closing action
- **Keep the change folder in a dated archive** (OpenSpec). Rejected: git is
  the archive, and the decision fold keeps the reasons
- **Delta sections**, ADDED, MODIFIED, REMOVED, RENAMED, a capability
  retired only on a flag, deltas merged without archiving (OpenSpec).
  Rejected: ele borrows no spec dialect
- **A change with no spec delta fails unless it says so** (OpenSpec).
  Already: a plan with no `spec:` proposes its out-of-scope lines as
  `decision` notes in its closing action instead
- **Refuse or warn on closing with unticked tasks** (OpenSpec `validate
  --archived` and `/opsx:archive`). Adopted by the user's ruling, held by an
  Inbox line: `ele plan
  check` warns on a closing action ticked while other actions are open, as
  it does with inbox items

## Decision records

- **Write an ADR only when the decision is hard to reverse, surprising
  without context, and the result of a real trade-off** (domain-modeling).
  Proposed for decision-fold: the test that sorts a Decision as
  architectural
- **ADRs for design decisions too** (adr.github.io). Rejected: a design
  Decision's reason goes into the spec; ADRs hold architectural ones
- **Considered options and the chosen one "because …"** (MADR; spec-kit's
  `research.md` "Alternatives considered"). Adopted by review-loop: a
  `rejected: <option>, because …` line under an architectural Decision for
  each option turned down, not only those worth remembering
  (domain-modeling); MADR's pros and cons of each option where the
  repository's format holds them
- **Sections: context of value-neutral forces, a decision in active voice,
  status, every consequence, good and bad** (Nygard; MADR's "Good, because"
  and "Bad, because", its decision drivers). Adopted by decision-fold, where
  the repository's own format holds those sections
- **Confirmation: how compliance with the ADR is checked** (MADR). Rejected
  as ele's addition: the ADR is written at close, after the done-when lines
  confirmed it; it carries the section where the repository's format does
- **Decision-makers, consulted and informed** (MADR). Rejected as ele's
  addition: the repository's format sets its fields
- **Numbered sequentially, never reused, per category where a repository
  numbers that way; a superseded record kept, pointing at its replacement**
  (Nygard, MADR). Adopted by decision-fold: the next number in the
  repository's series; superseding is its own question
- **Status values** proposed, accepted, deprecated, superseded, and MADR's
  rejected (Nygard, MADR). Proposed for plan-reviewer: only an accepted ADR binds a draft
- **A minimal record: context, the decision and why, in a few sentences**
  (MADR's minimal template; domain-modeling's 1-3 sentences). Adopted by
  decision-fold where the repository's format is that short; otherwise its
  format rules
- **One decision per record, one or two pages, written for a future
  developer** (Nygard). Adopted by decision-fold: one ADR per architectural
  Decision
