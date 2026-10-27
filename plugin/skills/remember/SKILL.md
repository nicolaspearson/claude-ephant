---
name: remember
description: Save a decision, an open question, a dated fact or a reminder for a later date so later sessions start with it. Use when the user says "remember this", "we decided…" or "remind me on…", or a skill or ele's Stop hook asks for notes.
---

# Remember

In every directory an owner covers, the SessionStart block shows the project's due reminders and open questions, then its newest decisions and facts, plus the owner's global notes and the everywhere notes, which also show where no owner covers. Write a note only when the user asks or approves, or when `auto_note_writes` is on and the note is proposed per Propose below.

## What a note holds

- A decision with its reason, an open question, or a dated fact. What the code does stays in the code, where it cannot go stale
- Or a reminder: what to come back to and when. It shows from its due date at every start until `ele reminder close <id>`; one that recurs, such as "every week", is a nudge, which closing re-arms
- Or a handoff: what `/ele:handoff` says it holds, proposed like any other note
- At most three lines; one is better. The reason inline, not a pointer to a file, ticket or plan; a handoff's first line is the exception, per `/ele:handoff`
- No secrets. A note is a plain file and is printed into sessions

## Write

```sh
ele remember --kind decision|fact|handoff|question <words>
ele remember --kind reminder --due YYYY-MM-DD <words>
ele remember --kind reminder --every <days> [--due YYYY-MM-DD] <words>
```

- `--due`: the local date a reminder first shows; work out a spoken date such as "next Tuesday" from `date +%F`, and give the date in the report
- `--every <days>`: a nudge, shown again that many days after each close; with no `--due` it shows today
- Owner and project come from the working directory unless `--project` names one; profile, session and time are recorded automatically
- `--project <name>`: the note concerns a project other than the working directory's. Works from any directory; `ele` names the owner or refuses
- `--global`: applies to every project of the owner, not just this one; never on a handoff
- `--everywhere`: applies to every session on the machine, whatever its owner, owners added later included: a note true in all of the user's work, such as one about the machine or their tooling. Never with `--global`, `--owner` or `--project`, nor on a handoff
- `--supersedes <id>`: replaces an earlier note; repeat it so one note replaces several. A decision that supersedes a question closes the question
- `--owner <name>`: only when the user names the owner, or `ele` refuses for want of one

Quote the words so the shell keeps them as written. Report the id `ele` prints: the user needs it to supersede the note later. `ele` refuses a note that a live note already says, and names that note. Report its id; do not reword to get past it.

To reword a note, save the new wording with `--supersedes <id>`. When the user says a note was saved by mistake, `ele forget <id>` moves it out (`--owner <name>` for another owner's note), and `ele forget --restore <id>` undoes it. `ele` refuses while another note supersedes it: forget that note first, or leave both.

Done when each note written has the id `ele` printed reported, each reminder with its due date, and each refused one the id of the live note `ele` named, or `ele`'s reason where it names none.

## Before writing

Read the note back as a stranger would, a month from now. If it needs the conversation to make sense, rewrite it.

## Propose

When a skill or ele's Stop hook asks for notes, propose the note it names, or else look over the work for what the next session would otherwise re-derive or re-ask:

- A decision and its reason, above all one that rejected the obvious alternative
- A question the user left open
- A dated fact about the environment that the code does not hold

A note true in every owner's work, such as one about the machine or the user's tooling, takes `--everywhere`; one true in every project of this owner, `--global`.

Nothing worth keeping, say so in one line. Otherwise run `ele config get auto_note_writes`. When it prints `true`, write each note yourself without asking, then report each id with the note's words, so a wrong one can be undone with `ele forget <id>`. Otherwise ask with AskUserQuestion, loading it through ToolSearch if it is deferred, and write each approved note yourself; never print the commands for the user to run. Ask in the same turn, one question per note: at most four per call, the tool's limit, and as many calls as it takes, so no note is dropped. Each question holds:

- `question`: the note's exact words, and the ids it supersedes, if any
- `header`: its kind
- Options: `Write` (save the note), `Skip` (drop it)

Write each note answered `Write`. An `Other` answer that rewords the note is written as reworded; any other `Other` answer is settled with the user before that note is written or skipped. Then report each id, and how many notes were proposed and written.

Done when the one line says nothing is worth keeping, or every proposed note is written, skipped or reported refused per Write, and the report gives each written note's id, with its words when written unasked, and how many were proposed and written when asked.
