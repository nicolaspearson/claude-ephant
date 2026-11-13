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
- Dev explores before it proposes the done-when, which opens the first
  question round; every question's recommended answer is its first option,
  labelled `(Recommended)`, each option carrying its evidence from the
  explorers' findings, since a trial run's done-when rested on the task's
  wording alone and its recommendations sat in the question text with the
  options unranked (ruled 2026-11-13)
  - rejected: proposing the done-when at intake, flagged as resting on the
    task's wording, because the tests and pages that would show it done are
    the explorers' to find
- The user typing `/ele:dev` is the ask wrap's "never start one unasked"
  needs, so dev runs `ele:wrap` once the checkpoint ends; wrap still asks
  before each note, handoff or recipe it writes (ruled 2026-11-13)
  - rejected: asking "wrap up now?" after the checkpoint, because the flow the
    user started already ends there
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
- [x] checkpoint-done-when: Checkpoint quotes a done-when the user confirmed in
  the session as it quotes a plan item, with the task and chosen design, and
  runs the spec reviewer on it; update `checkpoint/SKILL.md` §2 to §5 and §8
  where they name a quoted item, `reviewers/spec.md` throughout, the
  checkpoint spec and `docs/checkpoint-and-ticket-audit.md`
  - room in `checkpoint/SKILL.md` comes from moving a case only some runs need
    into its `references/`, as the skill lint's message says
  - done when `checkpoint/SKILL.md` stays under 500 lines with `task test`
    passing, the spec and docs page say a confirmed done-when runs the spec
    reviewer, and one checkpoint over a change briefed with such a done-when
    shows a `reviewers/spec.md` launch in `ele events --kind agent` whose
    brief, read with `ele show`, quotes the task, the done-when and the
    chosen design
  - as of 2026-11-13: `checkpoint/SKILL.md` was 499 lines; `internal/skilllint`
    `maxSkillLines` is 500 and fails at 500 or more, so §2's plan-files gate
    moved to `references/PLAN-GATE.md` to make room
  - as of 2026-11-13: the spec rule's "quoted item" wording also sits in §3's
    Observed bullet, §5's spec-finding check and §8's re-review brief, and in
    `reviewers/spec.md`'s intro, Hunt and Output beyond its Read section
  - as of 2026-11-13: `ele events --kind agent` records the reviewer files a
    launch named (`spec` among them) but not the brief; `ele show <session>
    --all` prints a launch's brief whole, `--at <timestamp> --context 1`
    truncates it
  - as of 2026-11-13: checkpoint quotes a confirmed done-when with the task in
    the user's words and the chosen design, and runs the spec reviewer on it
    (`checkpoint/SKILL.md` §2 to §5 and §8, `reviewers/spec.md`, the spec and
    docs page); §2's plan-files gate moved to `references/PLAN-GATE.md`,
    leaving `SKILL.md` at 499 lines; `task check FORCE=1` and `task audit` pass
  - asserted in the checkpoint-done-when commit and 103b032: at 103b032
    `checkpoint/SKILL.md` is 499 lines and `task check FORCE=1` passes; the
    spec and docs page say a confirmed done-when runs the spec reviewer; in a
    `task dev` session a checkpoint over a `cmd/ele/plan.go` change briefed with
    a done-when the user confirmed logged a launch in `ele events --kind agent`
    with `spec` among its reviewers, whose brief, read with `ele show --all`,
    quotes the task, the done-when and the design in the user's words
- [ ] dev-skill: Write `plugin/skills/dev/SKILL.md`,
  `.documentation/specs/dev.md` and `docs/dev.md`, and add `/ele:dev` to
  design.md's Names table and Integration list, the README's skill list and
  `docs/commands.md`
  - done when `task check FORCE=1` passes, and one `/ele:dev` run in a
    `task dev` session, on a small task in this repository with a real design
    choice, goes from intake to checkpoint with `ele:explorer`,
    `ele:architect` and `reviewers/spec.md` launches in
    `ele events --kind agent` for that session; `ele show` of it holds no
    Edit or Write before the user picks the design, a `decision` note
    proposed, and an `ele:wrap` after the checkpoint; the trial change is
    reverted unless the user keeps it
  - as of 2026-11-13: none of the three files exists; design.md's Names
    table Skills row and Integration list also leave out `/ele:review`,
    `/ele:setup` and `/ele:flag`, and its "Two skills call `ele`" list names
    only checkpoint and ticket-audit; `docs/commands.md`'s table is complete
    and alphabetical; the README's "Skills at a glance" is free-form
  - as of 2026-11-13: `internal/skilllint` exempts a
    `disable-model-invocation` skill from "Use when", allows only
    `references/`, `reviewers/`, `scripts/` and `assets/` beside `SKILL.md`,
    and checks their code-span paths exist; no test lists skills against the
    docs or design.md, and `plugin.json` lists no skills
  - as of 2026-11-13: `.documentation/specs/review.md` is the one-skill spec
    to follow, ending in `## In the plugin` with `Not doing:` lines;
    `docs/review.md` the one-skill docs page
  - as of 2026-11-13: no skill runs `ele:wrap`, whose description says never
    to start one unasked; `cmd/ele` `proposalNames` counts an `ele:wrap`
    skill event as a proposal for the Stop check
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
- [x] explorer-commands: Reword `plugin/reference/explorer.md`'s Commands,
  which says "their sync writes ele's own data" of `ele recall`, `ele context`
  and `ele show` though only `ele recall` syncs, to "recall's sync writes ele's
  own data, not the repository"
  - from the explorer-agent checkpoint's re-review, 2026-11-11
  - done when the file says so, and one `ele:explorer` launch in a `task dev`
    session uses only Glob, Grep, Read and the Bash commands Commands lists
  - asserted in the explorer-commands commit: Commands says recall's sync
    writes ele's own data, lists `grep -rn`, `find` and `ls` as the fallback
    where Grep or Glob is missing, and names `ele show --note <id>`; one
    `ele:explorer` launch in a `task dev` session (sonnet, the agent definition
    carrying the tool rules, before the review's rewording of the same rules)
    used Read, `ele recall` first, and grep in Bash with absolute paths after
    Grep proved missing: no `cd`, `sed`, `cat` or `head`
- [x] architect-language-rule: Drop "fixtures and generated files" from
  `plugin/reference/architect.md`'s language rule, which an agent with only
  Glob cannot tell by path; vendored code, `node_modules` and `testdata` stay
  - from the architect-agent checkpoint's re-review, 2026-11-11
  - done when the rule's exclusions name only what a path shows
  - asserted in the architect-language-rule commit: the language rule's
    exclusions read `vendor`, `node_modules` and `testdata`, each a path Glob
    shows; "vendored code" became `vendor`, by the user's ruling, to meet the
    done-when; a grep of plugin/, docs/ and .documentation/ found no other copy
    of the list
- [x] designs-rounds-scope: Say in `/ele:plan`'s Designs, design.md's Plans
  line on it and `docs/plans.md`, that architects run only before `new` step
  1's and `convert` step 2's rounds; a design question raised later, as an
  `open` review-loop finding or in `convert` step 5's re-audit round, is asked
  without architects
  - from the plan-decision-architects checkpoint, ruled 2026-11-12
  - done when `plugin/skills/plan/SKILL.md`'s Designs,
    `.documentation/specs/design.md` and `docs/plans.md` all say so
  - asserted in the designs-rounds-scope commit: `/ele:plan`'s Designs and
    design.md's Plans line say architects run only in `new` step 1's and
    `convert` step 2's rounds, and that a design question raised once a draft
    exists (an `open` review-loop finding, `convert` step 5's re-audit) is
    asked without them, both giving the reason; docs/plans.md states the
    rule without step numbers; `convert` step 2 now settles the items' design
    questions in SKILL.md and design.md, so its rounds can launch architects
    as docs/plans.md says
- [x] reviewer-tool-rules: List a recent `ele:checkpoint-reviewer` launch's
  tool calls from its transcript; where it read files through `cat` or `sed`,
  used `cd` or gave relative paths, add tool rules to
  `plugin/agents/checkpoint-reviewer.md` fitting its Bash needs, such as
  `git diff <ref>`, which differ from the explorer's
  - from the architect-language-rule pickup, 2026-11-12
  - done when the item describes what the listed launch ran, and either rules
    are added and one launch keeps to them, or the launch broke none and the
    item says so
  - asserted in the reviewer-tool-rules commit: of three
    `ele:checkpoint-reviewer` launches on the installed release, one used only
    Read, one ran `cd` twice and read with `sed -n` on relative paths, one read
    with `sed -n` on absolute paths, the session's subagents finding Grep
    missing; the
    agent definition now carries the explorer's Read, search-fallback and
    absolute-path rules, plus a line, by the user's ruling, allowing a scratch
    probe's Bash writes; one `ele:checkpoint-reviewer` launch in a headless `claude -p --plugin-dir` run with the clone's
    plugin used Read, `git -C` with an absolute path and Grep, no `cd`, `sed`
    or `cat`; the Grep-missing fallback and a probe were not exercised
- [x] designs-rounds-wording: In `plugin/skills/plan/SKILL.md`'s Designs and
  design.md's Plans line, replace "since it refines a design already chosen"
  and "since by then it refines a design already chosen", which overclaim,
  because such a question can concern a design not yet in `## Decisions`, with
  "since by then the user has seen a draft whose design the question can be
  weighed against, and each architect is a full design run"; in
  `docs/plans.md`'s `convert` paragraph, say `convert` asks the items' design
  questions in rounds; in this plan, reword designs-rounds-scope's tick line:
  rewrap "`convert` step 2", say "Plans line" as its item does, and replace
  "says the last" with "states the rule"
  - from the reviewer-tool-rules launch check's review of the
    designs-rounds-scope commit, 2026-11-12
  - done when the four files say so
  - asserted in the designs-rounds-wording commit: SKILL.md's Designs and
    design.md's Plans line give the reason "since by then a draft exists whose
    design the question can be weighed against", dropping, by the user's
    rulings, the cost half (the sentence before states it) and "the user has
    seen" (the review loop asks `open` findings before showing a draft);
    docs/plans.md's `convert` paragraph says it asks the items' design
    questions in rounds before drafting; designs-rounds-scope's tick line is
    rewrapped, says "Plans line" and "states the rule"; a grep of plugin/,
    docs/ and .documentation/ found no other copy of the old reason
- [x] architect-search-fallback: Say in `plugin/reference/architect.md`'s
  language rule that where Glob and Grep are both missing, leaving only Read,
  the design names no new file in a language it has not seen in the files it
  read, and lists the language check under Not checked
  - from the architect-language-rule pickup, 2026-11-12
  - done when the rule says so
  - asserted in the architect-search-fallback commit:
    `plugin/reference/architect.md`'s language rule says that where Glob and
    Grep are both missing, leaving only Read, the design names a new file only
    in a language seen in the files read and lists the language check under
    Not checked, the Report's field; a text property, with no launch run
- [x] architect-search-tools: In `plugin/reference/architect.md`'s language
  rule, say that where Glob is missing, the extension search runs through
  Grep's glob filter, and where a launch has Bash, as the `general-purpose`
  architect `/ele:plan` launches where `ele:architect` is not listed, through
  `find` in Bash with absolute paths, for searching only; the Read-only
  fallback then holds only where none of the three is there
  - from the architect-search-fallback checkpoint, 2026-11-12
  - done when the rule says so
  - asserted in the architect-search-tools commit:
    `plugin/reference/architect.md`'s language rule searches for a new file's
    extension with Glob, with Grep's glob filter where Glob is missing, and
    with `find` in Bash, by absolute paths and for searching only, where both
    are missing and the launch has Bash; the Read-only fallback holds only
    where none of the three is there; a text property, with no launch run
- [ ] Fold each Decision through `/ele:plan-audit close
  .documentation/plans/dev-skill.md`, propose each out-of-scope line as a
  `decision` through `/ele:remember`, run `ele plan done
  .documentation/plans/dev-skill.md`, and delete this plan

## Inbox

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
