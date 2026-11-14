# Dev

`/ele:dev <task>` takes one development task too small for a plan from intake
to checkpoint in one session. A task too small for a plan otherwise gets none of
ele's structure: no agreed done-when, no grounding before questions, and no
reviewer checking the change against what was asked. The shape, explore in
parallel subagents, ask, compare designs from separate architects, build,
review, is the feature-dev plugin's `/feature-dev`, put in ele's own words and
tools, since another plugin's names change between versions.

## Size

- A task needing more than one done-when, or more than one session, stops at
  intake: dev tells the user to run `/ele:plan new <name>`, since `/ele:plan`
  is invoked only by the user
- It stops the same way in the question rounds, on a done-when the explorers'
  findings show would need to be two or on an answer that splits the task:
  the size check at intake sees only the task's wording
- A size stop proposes no handoff: `/ele:plan new` launches its own explorers
  before its first round, so what dev's explorers found would only aim their
  briefs, and a handoff holds three lines. Follow-ups held for the mini plan's
  inbox are still proposed, since no mini plan is written to keep them
- Dev writes no plan file in the repository. A task that needs no plan would
  otherwise leave one to close and delete

## Mini plan

- Once the design is settled, dev writes what was agreed to a mini plan in the
  session scratchpad: `## Ask`, the task in the user's words; `## Requirements`,
  numbered, the done-when and each answer the user gave; `## Decisions`, the
  design and each turned down with its reason; and `## Inbox`, follow-ups found
  while building or reviewing. The checkpoint's comparison of what was built
  against what was asked needs every answer, and the conversation alone loses
  them to compaction; a session that compacted re-reads the file
- The scratchpad, not the repository: a file there is nothing to commit, close
  or delete, and is never on the checkpoint's change list. Not doing: a file in
  the repository, deleted at wrap or kept, since one is a plan to close and the
  other a claim about code that goes stale
- Work found that the task does not take goes under `## Inbox`, never done
  unasked or only said; at wrap, or at a mid-task stop, each line is proposed
  as deferred work, an active plan's inbox item or a `Plan candidate:` question
  note, since the scratchpad ends with the session. Proposed together, not one
  by one, so the build and review run uninterrupted; a session ended with
  neither loses them
- The checkpoint's brief quotes the mini plan less its inbox: an inbox line is
  work the task left out, and quoted as asked it would read as missing

## Approvals

- Three, then the build: the user confirms the done-when, the answers to the
  question rounds, and the design where there is a real choice. After the last,
  the build runs unasked
- Nothing is edited or written before the design is picked, but the mini plan
  in the scratchpad: what is built before then is the design chosen for the
  user
- A real choice is two or more designs that each meet the done-when and differ
  in what the change costs or leaves behind: its files, its interfaces, its
  behaviour beyond the done-when. With none, dev states the one design and goes
  on
- The done-when is proposed in the first question round, after the explorers
  report, not at intake: before them its recommendation could rest only on the
  task's wording, and which tests exist or which page describes the behaviour
  is theirs to find
- The done-when names the outcome, never the mechanism, so it holds whichever
  design is picked
- Questions run in rounds as `/ele:plan`'s do: each round asks every question
  whose prerequisites are settled; facts are the explorers' to find, so the
  user is asked for decisions only
- A question whose answers are alternative designs is never asked in the
  rounds: it goes to the design step, where the architects design each side.
  Answered in a round, it leaves the design step nothing to weigh and the
  architects never run
- Each question's recommended answer is its first option, labelled
  `(Recommended)`, and each option's description carries its evidence: a file
  and identifier the explorers cited, a recall finding, or a convention. A
  recommendation in the question text alone leaves the options unranked, and
  one resting on the task's wording alone gives the user nothing to weigh. A
  question the findings cannot ground gets another explorer first
- The design step always says whether there is a real choice and names the
  design, so the mini plan has a design to record and a session cannot pass
  from the answers to its first edit with neither

## Agents

- 2 or 3 `ele:explorer` agents, one per angle, in one message, before dev reads
  any code: a session that reads the code itself launches no explorer, and so
  runs no recall. Each is briefed as `/ele:plan`'s Grounding briefs one, plus
  the task in the user's words
- 2 `ele:architect` agents only where the design is a real choice, each under a
  constraint pulling it its own way, briefed as `/ele:plan`'s Designs briefs
  them. None otherwise, since each costs an opus run
- Where an agent type is not listed, as in a session started before a plugin
  update, dev launches `general-purpose` with the same brief

## Review

- `ele:checkpoint` reviews the change. Its brief quotes the mini plan whole,
  less its inbox, as what was asked: the done-when alone would leave every file
  the design touches reading as unasked, and an answer left out is never
  checked. The spec reviewer gives each numbered requirement its own verdict
- The mini plan's Decisions are proposed as one `decision` note through
  `/ele:remember` on every run: the design built and, where dev weighed
  designs, each rejected with its reason, so a run with no real choice still
  leaves its design on record
- Dev then runs `ele:wrap`. The user typing `/ele:dev` is the ask wrap's own
  description requires; wrap still asks before a recipe and, unless
  `auto_note_writes` is on, before each note and handoff it writes; asking
  "wrap up now?" after the checkpoint would ask again what typing `/ele:dev`
  already asked
- A session stopping before the checkpoint, by the user's word or because the
  task is blocked, proposes `/ele:handoff`, quoting the task, the done-when and
  the chosen design and saying the answers are to be confirmed again: the
  session after `/clear` has its own scratchpad, and a handoff names no path
  in one. A size stop proposes none, as Size says

## In the plugin

- `plugin/skills/dev/SKILL.md`, `disable-model-invocation: true`: a flow
  launching several agents should not start itself, as `/ele:plan` does not
- The agents are `ele:explorer` and `ele:architect`, their roles in
  `plugin/reference/explorer.md` and `plugin/reference/architect.md`
- `plugin/evals/dev-size-gate` hands dev a task spanning several sessions and
  grades its reply for `/ele:plan new` and its trace for no Agent launch and
  no Write, Edit not being granted there: intake's size check is the one stop
  an eval can reach, since every later one follows a question round
- Not doing: calling feature-dev's skill or agents. ele would depend on that
  plugin, and its names change between versions
- Not doing: an eval of the question rounds. AskUserQuestion does not load in
  the eval sandbox (as of 2026-10-29), so an eval case cannot answer them
