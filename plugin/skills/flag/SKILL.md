---
name: flag
description: Flag a moment of this session as good or bad, with why, so a later study of sessions can find it; or, given list, show the project's flags. Use when the user says "flag this", "flag that as bad", "mark this as good", or asks what has been flagged.
argument-hint: "[good|bad <why> | list]"
---

# Flag

A flag stamps a session at a moment with a verdict and the reason, for a later study of what went well or badly. The start block never shows one. The user asking is the approval: save it without proposing first. With `list` alone as the argument, follow List; otherwise Save.

## Save

The verdict is the argument's first word, `good` or `bad`. With neither, ask the user which with AskUserQuestion; never guess.

The why is the rest of the argument, in the user's words. With none, write one line on what just happened: the action, and what made it good or bad. At most three lines, specific enough for a reader who sees only the transcript around the flag: name the command, file or reply it is about.

```sh
ele flag bad "<why>"
```

It flags the running session, now. `--session <id>` flags an earlier one the user names; a prefix the index holds once is enough. Add `--at <time>` with the timestamp of the entry the flag is about, from `ele show <session> --around <text>`; without it the flag marks that session's last entry. `--project <name>` when the flagged work belongs to a project other than the working directory's.

Report the verdict and the id ele prints. On a refusal, show ele's message and save nothing else in its place.

Done when the reply reports the flag's verdict and id, or ele's refusal.

## List

```sh
ele flag list
```

Show each flag newest first: its verdict, when, its session and why. Say none when there are none. End with `ele show <session> --at <time>` to open the session at one. Change nothing.

Done when the reply shows that command's flags, or says there are none.
