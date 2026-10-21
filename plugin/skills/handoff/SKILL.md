---
name: handoff
description: Write an ele handoff note that the session after /clear opens with. Use when the user asks for a handoff note before /clear, rather than a handoff document for another agent.
---

# Handoff

The next fresh start or `/clear` in the project opens with the handoff. The user asking is the approval: write it without proposing first.

## Write

At most three lines: the next task, where the work stands, and the files and command to resume from. The next session reads the code and `git status` itself; spend the lines on what it cannot see there, such as an approach that failed and why.

```sh
ele remember --kind handoff "<words>"
```

`--project <name>` when the work belongs to a project other than the working directory's. A decision that outlives the task goes in its own note per `/ele:remember`.

Report the id, then end with: `/clear, then continue`.
