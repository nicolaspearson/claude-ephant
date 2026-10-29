# Notes and recall

ele gives a session two kinds of memory. Notes are the short, approved record a
session starts with: decisions, open questions and dated facts. Recall searches
everything earlier sessions said, whether or not anyone kept a note.

## Notes

A note is one to three lines. It holds:

- **decision**: what was decided and why, above all when it rejected the
  obvious alternative
- **question**: something left open, shown at every start until a later note
  answers it
- **fact**: a dated fact about the environment that the code does not hold,
  such as `as of 2026-10-01 the staging database is read-only`

A note never describes the code. The code says what it does, and a note that
restates it is wrong the day the code changes.

[Handoffs and reminders](handoffs-and-reminders.md) are notes too, with their
own rules, and so is the record of an active [plan](plans.md).

### Saving one

Ask for it in your own words: "remember that we chose SQLite over Postgres
because the binary must have no server". The `/ele:remember` skill words the
note, saves it and reports its id.

ele also asks. When a turn ends after work, its Stop hook asks Claude to
propose what is worth keeping, once per stretch of work between checkpoints.
It never asks during a checkpoint, or a `/ele:review` once its reviewers start,
since both end turns mid-run. The `notes_ask` option off stops it asking.
Claude puts each proposed note to you as a Write or Skip question, and writes
the ones you approve. With the `auto_note_writes` option on, it writes them
without asking and reports each id, so a wrong one can be undone.

`/ele:wrap` does the same at the end of a session, together with plan progress,
a [recipe](recipes.md), due reminders and a handoff.

From a terminal, `ele remember` saves one directly:

```sh
ele remember --kind decision "Ship SQLite, not Postgres: the binary must need no server"
ele remember --kind question --global "Should staging deploys need a second approver?"
```

- A note belongs to the working directory's project unless `--project <name>`
  names another; `--global` makes it apply to every project of the owner
- `--everywhere` makes it apply to every session on the machine, whatever its
  owner, owners added later and directories no owner covers included: a note
  about the machine or your tooling, saved once. It is filed under
  `notes/_all/`, takes no `--global`, `--owner` or `--project`, and is never a
  handoff. `ele context`, `ele reminder`, `ele forget` and `ele recall` reach it
  from any owner's directory; `ele reminder` and `ele forget` from a directory no
  owner covers too
- `--supersedes <id>` replaces an earlier note. A decision that supersedes a
  question answers it
- ele refuses a note whose words a live note already says, and names that note

### Changing one

No command edits a note in place. To reword one, save the new wording with
`--supersedes <old id>`: the old one stays searchable, marked superseded.

A note saved by mistake goes with `ele forget <id>`, which moves its file out of
the index into `forgotten/`. `ele forget --restore <id>` brings it back.

Notes are Markdown files under `notes/<owner>/` in the
[data directory](data-and-privacy.md#storage), and can be edited by hand; the
next sync picks up the change, and names any file it cannot read.

### Flags

A flag marks a moment of a session as good or bad, with why, so a later look
at what went well or badly can find it. Say `/ele:flag bad you ran the whole
suite when I asked for one package`, or run `ele flag good|bad "<why>"`; it is
stamped with the session and the time. `--session <id>` flags an earlier
session at its last entry, or at `--at <time>`, an entry's timestamp from
`ele show`. `ele flag list` shows the project's flags with their times, and
`ele show <session> --at <time>` opens the session at one. A flag never shows in the
start block or in `ele context`.

### What a session starts with

At startup, after `/clear` and after a compaction, the session opens with a
block ele prints for its project and owner:

1. Pending handoffs, on startup and `/clear` only
2. Active plans, with the next item
3. Reminders due today or earlier
4. Every open question, then decisions and facts, newest first. Decisions and
   facts are held to about 1,500 tokens, oldest cut first; questions are cut
   only when the whole block grows too long for Claude Code to show
5. A closing line counting what was not shown and how to search

`ele context` prints every live note for the project with no budget, for when
the block cut some.

## Recall

Ask in your own words: "did we already decide how retries work?" The
`/ele:recall` skill searches, then opens the discussions worth reading. From a
terminal:

```sh
ele recall retry backoff
ele recall --since 30d --kind note,prompt migration
```

- Every word must appear. Words are matched as plain text, so punctuation and
  search operators mean nothing special
- Results are grouped by session, best match first: notes, then what you typed,
  then conversation, then tool calls, each with a short snippet
- It searches the current directory's owner. `--project <name>` narrows to one
  project, `--owner <name>` searches another owner, `--all` every owner
- `--kind` takes `note`, `prompt`, `conversation` and `tool`; `--since` a date
  or a number of days back, as `7d`; `--limit` caps the matches, 20 by default

Recall syncs first, so the session you are in is searchable. `--no-sync` skips
that.

### What is searchable

What you typed, Claude's replies, and one line per tool call: the tool and its
main input, such as the command run or the file edited. Tool output, thinking
and compaction summaries are not indexed. Sessions started through the Agent
SDK are archived but not indexed, so automated runs do not flood the results.

### Reading a whole discussion

`ele show <session>` prints a session's entries; a unique prefix of its id is
enough. `--around <text>` centres on the first entry containing the text, and
`--context <n>` sets how many entries show either side.

`ele show --note <id>` opens the session a note came from at the discussion that
produced it. In the block a session starts with, a note from a recorded session
carries that session's id, and the block names this command once after the
notes.
