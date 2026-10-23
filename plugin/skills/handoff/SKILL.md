---
name: handoff
description: Write an ele handoff note that the session after /clear opens with. Use when the user asks for a handoff note before /clear, rather than a handoff document for another agent.
argument-hint: "[what the next session is for]"
---

# Handoff

The next fresh start or `/clear` in the project opens with the handoff. The user asking is the approval: write it without proposing first.

## Write

At most three lines. The first points at the next task: `Next: <plan path> <item name>` for a plan item, `Next: <url>` with the full URL of a GitHub or GitLab issue or merge request, or `Next:` and the task in words when it is neither; point, never restate the item or ticket. The rest: where the work stands, and the files and command to resume from. An argument, if given, is the next task; keep what bears on it. The next session reads the code, `git status` and the active plans' next items itself; spend the lines on what it cannot see there, such as an approach that failed and why.

Write as fact only what this session verified; mark the rest as a guess. Name only paths that survive `/clear` and a reboot: a file in the scratchpad or `/tmp` gets left out, or moved first: into the repo if the project would keep it, under `~/.claude/` if only the agent would run it.

```sh
ele remember --kind handoff "<words>"
```

`--project <name>` when the work belongs to a project other than the working directory's. A decision that outlives the task goes in its own note per `/ele:remember`.

Report the id, then end with: `/clear, then continue`.
