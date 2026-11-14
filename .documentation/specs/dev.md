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
- Dev writes no plan file. A task that needs no plan would otherwise leave one
  to close and delete

## Approvals

- Three, then the build: the user confirms the done-when, the answers to the
  question rounds, and the design where there is a real choice. After the last,
  the build runs unasked
- Nothing is edited or written before the design is picked: what is built
  before then is the design chosen for the user
- A real choice is two or more designs that each meet the done-when and differ
  in what the change costs or leaves behind: its files, its interfaces, its
  behaviour beyond the done-when. With none, dev states the one design and goes
  on
- The done-when is proposed in the first question round, after the explorers
  report, not at intake: before them its recommendation could rest only on the
  task's wording, and which tests exist or which page describes the behaviour
  is theirs to find. Intake runs the size check on the task as stated, and an
  answer that fails it stops dev there too
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
  design, so the checkpoint has a design to quote and a session cannot pass
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

- `ele:checkpoint` reviews the change. Its brief quotes the task in the user's
  words, the confirmed done-when and the chosen design together, since the
  done-when alone would leave every file the design touches reading as unasked,
  and a confirmed done-when runs the spec reviewer as a plan item does
- Where dev weighed designs, the chosen one and each rejected, with its reason,
  are proposed as one `decision` note through `/ele:remember`
- Dev then runs `ele:wrap`. The user typing `/ele:dev` is the ask wrap's own
  description requires; wrap still asks before a recipe and, unless
  `auto_note_writes` is on, before each note and handoff it writes
- A session stopping before the checkpoint proposes `/ele:handoff`, quoting the
  task, the done-when and the chosen design, since no plan file carries the
  place across `/clear`

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
