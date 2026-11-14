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
and points you at [`/ele:plan new <name>`](plans.md#starting-one).

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
   the one design and goes on. Nothing is edited before this point
5. **Builds** the design you picked, without asking again, and runs what the
   done-when names, or tells you what to run where it names a session of yours
6. **Checkpoints** the change. The checkpoint's spec reviewer checks it against
   your task, the done-when and the design together
7. **Wraps up**: it proposes the design you picked, and the ones you turned
   down, as one decision note, then runs `/ele:wrap`, which asks before keeping
   anything, except that with `auto_note_writes` on it keeps notes and a
   handoff unasked

Stopping before the checkpoint, it offers a handoff quoting the task, the
done-when and the design, so the next session can pick the task up after
`/clear`.
