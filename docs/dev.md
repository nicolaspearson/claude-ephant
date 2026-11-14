# Dev

`/ele:dev` takes one development task too small for a plan from the ask to a
reviewed change in a single session. It agrees what done looks like with you
first, reads the code in subagents before asking you anything, and ends in the
same checkpoint a plan item does.

```text
/ele:dev <task>
```

It runs only when you type it, since it launches several agents. A task that
needs more than one done-when, or more than one session, is a plan: dev says so
and points you at [`/ele:plan new <name>`](plans.md#starting-one). It checks
at the start, and again once the explorers report: a done-when they show would
need to be two, or an answer of yours that splits the task, stops it the same
way. It writes no handoff then, since `/ele:plan new` explores for itself.

## A run

1. **Restates** the task, and stops if it needs a plan
2. **Explores** the code with two or three explorer agents in parallel, each
   on its own angle, such as the code the task changes and the tests and pages
   that cover it. Each runs `ele recall` too, so earlier sessions on the area
   count
3. **Agrees the done-when, then asks** what the findings leave open, in rounds.
   The done-when says what will be observable once the task is done, such as a
   command's output, a test passing or a real-session run. The first option of
   each question is the recommended one, and every option says what in the code
   or in earlier sessions backs it. It asks you for decisions, never for facts
   it can find
4. **Weighs designs** where there is a real choice: two architect agents each
   design under a different constraint, such as the smallest change against
   the shape easiest to extend, and you pick one. With no real choice it states
   the one design and goes on. It then writes a mini plan of what you agreed,
   below. Nothing else is written before this point
5. **Builds** the design you picked, without asking again, and runs what the
   done-when names, or tells you what to run where it names a session of yours.
   Work it finds that the task does not take goes in the mini plan's inbox,
   not into the change
6. **Checkpoints** the change against the mini plan: the spec reviewer judges
   the done-when and each of your answers on its own, and reads the design as
   asked for
7. **Wraps up**: it proposes the mini plan's design, and the ones you turned
   down, as one decision note, then runs `/ele:wrap`, which offers each inbox
   line to an active plan's inbox or as a plan candidate note. Wrap asks before
   keeping anything, except that with `auto_note_writes` on it keeps notes and
   a handoff unasked

## The mini plan

A Markdown file in the session's scratchpad, never in your repository, so
nothing is left to commit or delete. It has a plan's sections:

- `## Ask`: the task in your words
- `## Requirements`: numbered, the done-when first, then each answer you gave
- `## Decisions`: the design and its files, and each design you turned down
  with why
- `## Inbox`: follow-ups found while building or reviewing, dated with where
  they came from

The checkpoint compares the change against it line by line, and dev re-reads it
if the session compacts. It ends with the session, which is why wrap moves its
inbox lines somewhere that lasts.

Stopping before the checkpoint on your word or a blocked task, it offers a
handoff quoting the task, the done-when and the design, so the next session can
pick the task up after `/clear`. That session has its own scratchpad, so it
confirms your answers again. It also offers the mini plan's inbox lines, as
wrap would.
