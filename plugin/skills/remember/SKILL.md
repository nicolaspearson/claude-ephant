---
name: remember
description: Save a decision, an open question or a dated fact so later sessions start with it. Use when the user says "remember this" or "we decided…", or a skill or ele's Stop hook asks for notes.
---

# Remember

In every directory an owner covers, the SessionStart block shows the project's open questions, then its newest decisions and facts, plus the owner's global notes. Write a note only when the user asks or approves.

## What a note holds

- A decision with its reason, an open question, or a dated fact. What the code does stays in the code, where it cannot go stale
- Or a handoff: what `/ele:handoff` says it holds, proposed like any other note
- At most three lines; one is better. The reason inline, not a pointer to a file, ticket or plan
- No secrets. A note is a plain file and is printed into sessions

## Write

```sh
ele remember --kind decision|fact|handoff|question <words>
```

- Owner and project come from the working directory unless `--project` names one; profile, session and time are recorded automatically
- `--project <name>`: the note concerns a project other than the working directory's. Works from any directory; `ele` names the owner or refuses
- `--global`: applies to every project of the owner, not just this one; never on a handoff
- `--supersedes <id>`: replaces an earlier note. A decision that supersedes a question closes the question
- `--owner <name>`: only when the user names the owner, or `ele` refuses for want of one

Quote the words so the shell keeps them as written. Report the id `ele` prints: the user needs it to supersede the note later. `ele` refuses a note that a live note already says, and names that note. Report its id; do not reword to get past it.

To reword a note, save the new wording with `--supersedes <id>`. When the user says a note was saved by mistake, `ele forget <id>` moves it out (`--owner <name>` for another owner's note), and `ele forget --restore <id>` undoes it. `ele` refuses while another note supersedes it: forget that note first, or leave both.

## Before writing

Read the note back as a stranger would, a month from now. If it needs the conversation to make sense, rewrite it.

## Propose

When a skill or ele's Stop hook asks for notes, propose the note it names, or else look over the work for what the next session would otherwise re-derive or re-ask:

- A decision and its reason, above all one that rejected the obvious alternative
- A question the user left open
- A dated fact about the environment that the code does not hold

Nothing worth keeping, say so in one line. Otherwise ask with AskUserQuestion, loading it through ToolSearch if it is deferred, and write each approved note yourself; never print the commands for the user to run. Ask in the same turn, one call, one question per note, at most four, the tool's limit:

- `question`: the note's exact words, and the id it supersedes, if any
- `header`: its kind
- Options: `Write` (save the note), `Skip` (drop it)

Write each note answered `Write`. An `Other` answer that rewords the note is written as reworded; any other `Other` answer is settled with the user before that note is written or skipped. Then report each id, and how many notes were proposed and written.
