---
name: handoff
description: Write an ele handoff note that the session after /clear opens with, or, given list, show the project's pending and closed handoffs. Use when the user asks for a handoff note before /clear, rather than a handoff document for another agent, or asks what handoffs there are.
argument-hint: "[list | what the next session is for]"
---

# Handoff

The next fresh start or `/clear` in the project opens with the handoff. The user asking is the approval: write it without proposing first. With `list` alone as the argument, follow List instead; any other argument, even one starting with the word, is the next task for Write.

## List

```sh
ele handoff list --all
```

Print what it prints in two groups, pending then closed, each newest first: the id, when it was written, its text, and for a closed one when it was closed; a superseded one goes under closed with the id that superseded it. Say none when a group is empty. End with `ele handoff close <id>` to drop a pending one and `ele handoff reopen <id>` to show a closed one again at the next start. Change nothing.

Done when the reply shows both groups from that command's output and changed no handoff.

## Write

At most three lines. The first points at the next task: `Next: <plan path> <item name>` for a plan item, `Next: <url>` with the full URL of a GitHub or GitLab issue or merge request, or `Next:` and the task in words when it is neither; point, never restate the item or ticket. The rest: where the work stands, and the files and command to resume from. An argument, if given, is the next task; keep what bears on it. The next session reads the code, `git status` and the active plans' next items itself; spend the lines on what it cannot see there, such as an approach that failed and why.

Write as fact only what this session verified; mark the rest as a guess. Name only paths that survive `/clear` and a reboot: a file in the scratchpad or `/tmp` gets left out, or moved first: into the repo if the project would keep it, under `~/.claude/` if only the agent would run it.

```sh
ele remember --kind handoff "<words>"
```

`--project <name>` when the work belongs to a project other than the working directory's. A decision that outlives the task goes in its own note per `/ele:remember`.

Report the id, then end with: `/clear, then continue`.

Done when the reply reports the handoff's id, or a refusal as `/ele:remember`'s Write does, and ends with `/clear, then continue`.
