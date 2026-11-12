---
kind: plan
created: 2026-11-11
---

# ele:dev

## Why

A development task too small for a plan gets none of ele's structure: no agreed
done-when, no grounding before questions, and no reviewer checking the change
against what was asked. The feature-dev plugin's `/feature-dev` shows a shape
that works for such a task (explore in parallel subagents, ask, compare designs
from separate architects, build, review), but it keeps no state across
`/clear`, ends in its own review rather than ele's checkpoint, and its agents
ask for documented code. This plan builds `ele:dev` on that shape with ele's
tools, gives ele its own explorer and architect agents, and lets the plan skill
use them where it reads code or weighs designs in the main session.

## Requirements

1. An `ele:dev` skill, similar to `/feature-dev`, for a development task that
   does not need a plan, using ele's tooling
2. ele's own `ele:explorer` and `ele:architect` agents, following the
   `CLAUDE.md` files their brief names, with the rules that hold in any
   repository (no comment restating code, no file in a language the
   repository does not use) built in
3. `/ele:plan`'s Grounding, architectural Decisions and pickup premise check
   use the new agents

## Context

- As of 2026-11-11: checkpoint launches its spec reviewer only when the brief
  quotes a plan item or fetched ticket (`plugin/skills/checkpoint/SKILL.md`
  §4's Spec paragraph, and `reviewers/spec.md` reads only an item or ticket; `.documentation/specs/checkpoint-and-ticket-audit.md`'s
  spec reviewer line)
- As of 2026-11-11: `plugin/skills/checkpoint/SKILL.md` is 499 lines, and the
  skill lint (`internal/skilllint`, run by `task test`) fails a `SKILL.md` of
  500 lines or more
- As of 2026-11-11: `plugin/agents/` holds three thin agents, each told to
  follow the file its brief names: `checkpoint-reviewer` (sonnet; Bash, Glob,
  Grep, Read), `plan-reviewer` (opus; Glob, Grep, Read) and `review-reader`
  (inherit; Glob, Grep, Read)
- As of 2026-11-11: `.documentation/specs/design.md`'s Principles say ele's
  skills put an idea taken from another plugin in their own words rather than
  call that plugin's skill
- As of 2026-11-11: `/ele:plan`'s Grounding and pickup premise check read the
  code in the main session; `ele:ticket-audit` and `ele:plan-audit` already
  brief subagents for their methods
- As of 2026-11-11: `ele events --kind agent` records each launch's subagent
  type and the reviewer files its prompt names (design.md's Events section)
- As of 2026-11-11: AskUserQuestion does not load in the eval sandbox
  (CONTRIBUTING's Evals section, dated 2026-10-29), so an eval case cannot
  answer ele:dev's question rounds
- As of 2026-11-11: claude-ephant keeps no ADRs; its specs carry each
  decision's reason

## Decisions

- `/ele:dev <task>` is invoked only by the user (`disable-model-invocation`),
  as `/ele:plan` is: a flow launching several agents should not start itself
- Size gate: at intake, a task needing more than one done-when or more than one
  session stops ele:dev, which tells the user to run `/ele:plan new <name>`,
  since `/ele:plan` is invoked only by the user
- Three approvals, then build: the user confirms the done-when, the answers to
  the question rounds, and the design where there is a real choice; after the
  last, the build runs unasked. Review is `ele:checkpoint`, then `ele:wrap`
- A done-when the user confirmed in the session counts for checkpoint as a plan
  item does, so the spec reviewer runs; the brief quotes the task in the
  user's words, the done-when and the chosen design together, since the
  done-when alone would leave every file the design touches reading as
  unasked
  - rejected: writing a one-item plan file per task, because a task that needs
    no plan would then leave one to close and delete
- Stopping mid-task proposes `/ele:handoff`, quoting the done-when and the
  chosen design, since no plan file carries the place across `/clear`
- The chosen design and its rejected options are proposed as one `decision`
  note through `/ele:remember`
- ele ships its own `ele:explorer` and `ele:architect` agents (ruled
  2026-11-11)
  - rejected: briefing the built-in Explore and Plan agents, because the user's
    conventions would hold only where every brief repeated them
  - rejected: feature-dev's code-explorer and code-architect, because ele
    would depend on that plugin, its names change between versions, and its
    architect asks for well-documented code, against the user's comment rules
- Each agent file stays thin; its role lives in `plugin/reference/explorer.md`
  and `plugin/reference/architect.md`, since `reference/` holds files several
  skills read
  - rejected: the role inside the agent file, because a skill briefing it could
    not name the file its rules come from, as the reviewer agents' briefs do
- Each agent's `description` says it is launched by ele skills with a brief
  naming its reference file, as the reviewer agents' do, so no session picks
  one unprompted
- Where an agent type is not listed, as in a session started before a plugin
  update, the skill launches `general-purpose` with the reference file in the
  brief, as checkpoint does for its reviewer
- `ele:explorer`: sonnet; Bash, Glob, Grep, Read; edits nothing. Its reference
  file names the Bash it runs, `ele recall`, `ele context`, `ele show` and
  git's read commands, and what it never runs: a git write, or a cluster, cloud
  or other live-infrastructure command. It reports `file:line` cites, the 5 to
  10 files to read, "not checked" on what it did not verify, the grep behind
  any claim of absence, and recall findings as what was observed, never a note
  or session id, since a plan must not cite ele data; only the convention
  files its brief names bind it
- `ele:architect`: opus; Glob, Grep, Read, no Bash; commits to one design under
  the constraint its brief names; the design follows the repository's
  conventions: no comment restating code, no file in a language the
  repository does not use
- ele:dev launches 2 to 3 explorers, each on a different angle, and 2
  architects only where the design is a real choice, the same test as its
  design approval; none otherwise, since each costs an opus run
- The plan skill takes explorers in `new`/`convert` Grounding, an explorer in
  pickup's premise check, and two architects before the round that puts a
  design question meeting plan-audit Close mode's architectural test (hard to
  reverse, surprising without context, a real trade-off), so its `rejected:`
  lines come from designs the session did not author
- One eval case covers the size gate; the question rounds are not evaled,
  since AskUserQuestion does not load in the eval sandbox. The user runs
  `task eval`
- A live run named in a done-when runs in a `task dev` session, since
  `task install` replaces only the binary and the agents and skills load from
  the plugin

## Actions

- [x] explorer-agent: Write `plugin/agents/explorer.md` and
  `plugin/reference/explorer.md`, and add the agent to design.md's Names table
  and Integration list
  - done when, in a `task dev` session, `ele:explorer` is listed as an agent
    type, and one launch over a named question about this repository whose
    answer has a recall hit returns `file:line` cites, a list of files to
    read, a "not checked" line or the statement that it has none, and the
    recall finding with no note or session id
  - as of 2026-11-11: the three agents' frontmatter matches Context
    (`plugin/agents/*.md`, lines 4–5 of each); design.md's Names table holds
    no agents row, so adding one lists all four agents; its Integration list
    is the `plugin/` file line under Integration
  - as of 2026-11-11: an `agent` event records the subagent type, and as
    reviewer files only `reviewers/<name>.md` (`internal/transcript`'s
    `reviewerFile`), so `reference/explorer.md` in a brief is not recorded;
    the done-when needs only the type
  - as of 2026-11-11: stop-check counts a subagent's Edit, NotebookEdit and
    Write as work, not its Bash (`cmd/ele`'s `subagentWorkTools`), so an
    explorer's reads raise no notes ask
  - as of 2026-11-11: no `docs/` page lists the agents outside the skill using
    each, so this item changes no user doc
  - asserted in the explorer-agent commit: in a `task dev` session
    `ele:explorer` resolved as an agent type and its `agent` event was
    recorded; its report on how ele records agent launches gave `file:line`
    cites, 8 files to read, a Not checked list and 7 dated recall findings
    with no note or session id; the run predates the Commands rewording,
    which the done-when does not cover
- [x] architect-agent: Write `plugin/agents/architect.md` and
  `plugin/reference/architect.md`, and add the agent to design.md's Names table
  and Integration list
  - done when, in a `task dev` session, `ele:architect` is listed as an agent
    type, and one launch with a named constraint, on a task in this Go
    repository that invites a helper script and doc comments, returns a single
    committed design naming the files it changes, with no file in a language
    the repository does not use and no comment restating code
  - as of 2026-11-11: design.md's Names table has an Agents row
    (`.documentation/specs/design.md:17`) and the Integration list is its
    `plugin/` file line (`:181`), each listing `ele:explorer` alphabetically,
    so `ele:architect` goes first in both; `plugin/agents/explorer.md` and
    `plugin/reference/explorer.md` are the pattern to follow
  - asserted in the architect-agent commit: in a `task dev` session
    `ele:architect` resolved as an agent type and its `agent` event was
    recorded; given a minimal-change constraint and a task to regenerate
    design.md's Agents row with documented exported functions, it returned one
    design in an existing Go package naming the files it changes, no new file,
    and two one-line doc comments each stating a contract; it used only Glob,
    Grep and Read; the run predates the checkpoint's rewording of the reference
    file's Rules and Costs and of the agent body, which the done-when does not
    cover
- [ ] checkpoint-done-when: Checkpoint quotes a done-when the user confirmed in
  the session as it quotes a plan item, with the task and chosen design, and
  runs the spec reviewer on it; update `checkpoint/SKILL.md` §2, §3's What was
  asked and §4's Spec paragraph, `reviewers/spec.md`'s Read section, the
  checkpoint spec and `docs/checkpoint-and-ticket-audit.md`
  - blocked: v1-release/cut-v1; v1-release's open items also edit
    `checkpoint/SKILL.md`, which has one line under the lint's limit
  - room in `checkpoint/SKILL.md` comes from moving a case only some runs need
    into its `references/`, as the skill lint's message says
  - done when `checkpoint/SKILL.md` stays under 500 lines with `task test`
    passing, the spec and docs page say a confirmed done-when runs the spec
    reviewer, and one checkpoint over a change briefed with such a done-when
    shows a `reviewers/spec.md` launch in `ele events --kind agent` whose
    brief, read with `ele show`, quotes the task, the done-when and the
    chosen design
- [ ] dev-skill: Write `plugin/skills/dev/SKILL.md`,
  `.documentation/specs/dev.md` and `docs/dev.md`, and add `/ele:dev` to
  design.md's Names table and Integration list, the README's skill list and
  `docs/commands.md`
  - blocked: checkpoint-done-when
  - done when `task check FORCE=1` passes, and one `/ele:dev` run in a
    `task dev` session, on a small task in this repository with a real design
    choice, goes from intake to checkpoint with `ele:explorer`,
    `ele:architect` and `reviewers/spec.md` launches in
    `ele events --kind agent` for that session; `ele show` of it holds no
    Edit or Write before the user picks the design, a `decision` note
    proposed, and an `ele:wrap` after the checkpoint; the trial change is
    reverted unless the user keeps it
- [ ] dev-size-eval: Add an eval case under `plugin/evals/` in which
  `/ele:dev` is handed a task spanning several sessions and tells the user to
  run `/ele:plan new`, with no Edit or Write
  - blocked: dev-skill
  - done when the case exists and the user's `task eval -- ./plugin` run passes
    its graders
- [x] plan-grounding-explorers: `/ele:plan` `new` and `convert` launch
  `ele:explorer` agents for the facts each round rests on; update the Plans
  section of design.md and `docs/plans.md`
  - done when one `/ele:plan new` run in a `task dev` session shows
    `ele:explorer` launches before its first round in
    `ele events --kind agent`, and the user's `task eval -- ./plugin` run
    passes `plan-new-review`'s graders
  - as of 2026-11-11: `/ele:plan`'s Grounding and convert step 2, design.md's
    Plans section and `docs/plans.md` send the rounds' fact-finding to
    `ele:explorer` agents, at most 3, `general-purpose` where the type is not
    listed; `plan-new-review` gains an `explores` grader requiring an
    `ele:explorer` launch; `task check FORCE=1` and `task audit` pass
  - as of 2026-11-11: the user's `task eval -- ./plugin` run failed
    `plan-new-review`'s `explores` grader, the session reading the scaffold's
    few files itself and launching no `ele:explorer`; its other five graders
    passed. Grounding now launches explorers however small the repository,
    the session reading no code before the first round
  - as of 2026-11-11: at 985b956 the user's `task eval -- ./plugin --case
    plan-new-review` passed all six graders, `explores` with one
    `ele:explorer` launch; the live `task dev` run is still to do
  - asserted in the plan-grounding-explorers commit: in a `task dev`
    session, `/ele:plan new blocked-colon` launched two `ele:explorer`
    agents, one on the code and one on the specs and docs, before its first
    round's reply, by that session's agent and reply events in `ele events`;
    at 985b956 the user's `task eval -- ./plugin --case plan-new-review`
    passed all six graders, run as that one case since no other case runs
    `/ele:plan`
- [x] plan-decision-architects: `/ele:plan` `new` and `convert` launch two
  `ele:architect` agents under different constraints before the round that
  puts a design question meeting the architectural test, the designs not
  chosen becoming `rejected:` lines; update the Plans section of design.md and
  `docs/plans.md`
  - done when one `/ele:plan new` run in a `task dev` session with an
    architectural Decision shows two `ele:architect` launches in
    `ele events --kind agent`, each design they returned appears in the draft
    as the chosen design or a `rejected:` line, and the user's `task eval -- ./plugin` run passes
    `plan-new-review`'s graders
  - as of 2026-11-12, before this item's change: the architectural test is
    plan-audit Close mode's Architectural bullet, adopted from
    domain-modeling in `plugin/reference/plan-practices.md`'s Decision
    records; the plan skill's Grounding, format and `new` step 2 are where a
    launch and its `rejected:` lines fit; design.md's Plans section carries
    Grounding in its `/ele:plan` bullets, and `docs/plans.md`'s Starting one
    named explorers but no architects
  - as of 2026-11-12: `plugin/reference/architect.md` already says another
    architect may design the same task under another constraint and the
    session compares them, and its Report ends in Costs, the source of a
    `rejected:` line's "because"; `plugin/agents/architect.md` is opus with
    Glob, Grep and Read
  - as of 2026-11-12: `plan-new-review`'s tally scenario
    (`plugin/evals/plan-new-review/case.yaml`) holds no decision passing the
    architectural test (`--lines` and `--json` are reversible, `--url` is
    ruled out by the ADR), so its graders guard against regression only and
    the live run is what shows the architects
  - asserted in the plan-decision-architects commit: in a `task dev` session,
    `/ele:plan new cross-repo-blockers` launched two `ele:architect` agents in
    one message after its first round, under a smallest-change and a
    robust-identity constraint, by that session's agent events; the
    smallest-change design became the draft's first Decision and the
    remote-identity design its `rejected:` line; a run on a reversible parsing
    fix launched none; the user's `task eval -- ./plugin --case
    plan-new-review` passed all six graders, its scenario holding no
    architectural question
- [x] pickup-premise-explorer: `/ele:plan` pickup's premise check launches an
  `ele:explorer` to re-derive the item's claims about code; update the Plans
  section of design.md and `docs/plans.md`
  - done when one pickup in a `task dev` session shows an `ele:explorer`
    launch in
    `ele events --kind agent` and the item's dated premise lines carry its
    cites
  - as of 2026-11-12, before this item's change: pickup's premise check
    re-derived claims in the main session (`plugin/skills/plan/SKILL.md`
    Pickup step 4's first bullet);
    design.md's Plans section said so in its pickup bullet and
    `docs/plans.md`'s Working through one in its pickup paragraph
  - asserted in the pickup-premise-explorer commit: in a `task dev` session on
    this change, `/ele:plan` pickup of v1-release's scratch-log launched one
    `ele:explorer` after the yes to start, by that session's agent events, its
    brief naming `plugin/reference/explorer.md`, both `CLAUDE.md` files, the
    plan and the item quoted whole; the premise lines it wrote under
    scratch-log cite packages and identifiers, this repository refusing line
    numbers in a plan, and mark one claim not checked; with only this change
    on HEAD, `task check FORCE=1`'s lint-docs, unused, tidy-check and test
    passed, and its lint in the working tree
- [ ] explorer-commands: Reword `plugin/reference/explorer.md`'s Commands,
  which says "their sync writes ele's own data" of `ele recall`, `ele context`
  and `ele show` though only `ele recall` syncs, to "recall's sync writes ele's
  own data, not the repository"
  - from the explorer-agent checkpoint's re-review, 2026-11-11
  - done when the file says so, and one `ele:explorer` launch in a `task dev`
    session uses only Glob, Grep, Read and the Bash commands Commands lists
- [ ] architect-language-rule: Drop "fixtures and generated files" from
  `plugin/reference/architect.md`'s language rule, which an agent with only
  Glob cannot tell by path; vendored code, `node_modules` and `testdata` stay
  - from the architect-agent checkpoint's re-review, 2026-11-11
  - done when the rule's exclusions name only what a path shows
- [ ] Fold each Decision through `/ele:plan-audit close
  .documentation/plans/dev-skill.md`, propose each out-of-scope line as a
  `decision` through `/ele:remember`, run `ele plan done
  .documentation/plans/dev-skill.md`, and delete this plan

## Inbox

- 2026-11-12, from the plan-decision-architects checkpoint: say in
  `/ele:plan`'s Designs that it covers only `new` step 1's and `convert` step
  2's rounds, before drafting; a design question raised later, as an `open`
  review-loop finding or in `convert` step 5's re-audit round, is asked
  without architects (ruled 2026-11-12)
- 2026-11-12, from the pickup-premise-explorer checkpoint:
  `plugin/reference/explorer.md`'s Commands needs a fallback for a session
  without Grep or Glob, and says a note id takes `ele show --note`

## Out of scope

- `ele:ticket-audit` and `ele:plan-audit` moving to `ele:explorer`: their
  subagent briefs are already specific to their methods (ruled 2026-11-11,
  from the plan's rounds)
- `ele:checkpoint` taking more from feature-dev: its code-reviewer's ideas
  went into checkpoint on 2026-10-07 (ruled 2026-11-11, from the review loop)
- `ele:review` using `ele:explorer`: it has its own `ele:review-reader`
  (ruled 2026-11-11, from the review loop)
- wrap, remember, recall, recipe, handoff, flag and setup using the new
  agents: none of them explores code (ruled 2026-11-11, from the review loop)
- A second eval case running a small task through ele:dev's rounds: the
  dev-skill item's live run covers the flow (ruled 2026-11-11, from the
  review loop)
