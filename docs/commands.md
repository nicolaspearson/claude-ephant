# Command reference

`ele --help` lists every command with its flags; `ele <command> --help` says what
each flag does. A misused command prints its own help and exits 2.

Where a command resolves an owner or project, it takes them from the working
directory. `--owner <name>` and `--project <name>` name others, so the command
works from any directory; see [Owners](owners.md).

## Skills and slash commands

| Command                 | What it does                                                                                     |
| ----------------------- | ------------------------------------------------------------------------------------------------ |
| `/ele:checkpoint`       | Review a change in subagents before it is reported done; see [Checkpoint](checkpoint-and-ticket-audit.md#checkpoint) |
| `/ele:checkpoint-stats` | What checkpoints found and which stage found it, then reviews apart, over the last 30 days or the number given |
| `/ele:handoff`          | Write the handoff the next session opens with; see [Handoffs](handoffs-and-reminders.md#handoffs) |
| `/ele:mine`             | Judge past checkpoint runs and review cycles, for tuning the checkpoint skill                    |
| `/ele:plan`             | Pick up the active plan, or `new <name>` or `convert <spec>`; see [Plans](plans.md)              |
| `/ele:plan-audit`       | Audit active plans against the code and each other                                               |
| `/ele:recall`           | Search earlier sessions; see [Recall](notes-and-recall.md#recall)                                 |
| `/ele:remember`         | Save a decision, question, fact or reminder                                                       |
| `/ele:review`           | Review an MR, PR, branch or commit and publish only its major findings; see [Review](review.md)   |
| `/ele:ticket-audit`     | Audit a tracker's ticket map against the code                                                     |
| `/ele:ticket-claims`    | Flag open GitHub tickets mistyped as a task or granting themselves execution                      |
| `/ele:wrap`             | At a session's end, ask what to keep, which reminders to close and whether to hand off            |

## Notes

```text
ele remember --kind decision|fact|handoff|question|reminder [--done-file <path>] [--due <YYYY-MM-DD>] [--every <days>] [--global] [--owner <name>] [--project <name>] [--session <id>] [--supersedes <id>...] <words>
```

Save a note. `--global` applies it to every project of the owner. `--supersedes`
replaces an earlier note; repeat it for several. `--session` records the session
the note came from, by default the running one. For a reminder, `--due` is the
first day it shows; `--every <days>` makes it a nudge, and `--done-file` ties the
nudge to a file's last change.

```text
ele context [--owner <name>] [--project <name>]
ele context --size
```

Every live note and pending handoff for the project, with no budget. `--size`
prints the running session's context in tokens instead.

```text
ele forget [--owner <name>] [--restore] <id>...
```

Move mistaken notes out of the index; `--restore` brings them back.

```text
ele handoff list [--all] [--owner <name>] [--project <name>]
ele handoff close [--owner <name>] [--project <name>] <id>...
ele handoff reopen [--owner <name>] [--project <name>] <id>...
```

Pending handoffs, newest first; `--all` adds closed and superseded ones. `close`
marks handoffs picked up or dropped; `reopen` shows them again at the next start.

```text
ele reminder list [--all] [--colour] [--owner <name>] [--project <name>]
ele reminder close [--owner <name>] [--project <name>] <id>...
ele reminder reopen [--owner <name>] [--project <name>] <id>...
```

Open reminders, soonest due first; `--all` adds closed and superseded ones, and
`--colour` highlights the list, for `! ele reminder list --colour`. `close` marks
reminders done, or re-arms a nudge; `reopen` opens them again.

## Plans

```text
ele plan start <path>
ele plan show [<path>]
ele plan list [--all] [--owner <name>] [--project <name>]
ele plan check [<path>]
ele plan done [--owner <name>] [--project <name>] <path|id>...
```

`start` makes a plan file active. `show` prints its counts, previous item,
blocked items and next item, the project's active plan by default. `list` lists
active plans; `--all` adds done and superseded ones. `check` lists each plan's
warnings with their lines. `done` closes a plan and keeps its item names, so run
it from the plan's checkout once its last item is done; it does not check.

## History

```text
ele recall [--all | --owner <name>] [--kind <k,k>] [--project <name>] [--since <YYYY-MM-DD|Nd>] [--limit <n>] [--no-sync] <query>
```

Search past sessions and notes; every word must appear. `--kind` takes `note`,
`prompt`, `conversation` and `tool`, comma-separated. `--since` takes a date or a
number of days back, as `7d`. `--limit` caps the matches. `--no-sync` searches
without syncing first.

```text
ele show [--around <text> | --at <time>] [--context <n>] <session>
ele show --note <id> [--context <n>]
```

A session's entries, centred on the first containing `--around`'s text or at or
after `--at`'s time, with `--context` entries either side. `--note` opens the
session a note came from at its discussion.

```text
ele events [--during <skill>] [--kind <kind>] [--name <name>] [--owner <name>] [--project <name>] [--session <id>] [--since <YYYY-MM-DD|Nd>] [--limit <n>] [--no-sync]
```

Skill calls, typed commands, subagent launches, hook blocks, compactions,
injected context, replies, token use and tool errors, one JSON object per line,
the newest 200 by default; `--limit 0` prints all. `--during <skill>` keeps only
events inside a run of that skill.

```text
ele usage [--owner <name>] [--project <name>] [--since <YYYY-MM-DD|Nd>] [--no-sync]
```

Sessions, prompts, tool calls, bytes injected by hooks and the skill list, reply
length and API tokens, main thread and subagents apart.

## Index

```text
ele sync [--background]
ele reindex
ele status
ele stats
ele version
ele backup <dir>
```

`sync` archives new transcripts and updates the index; the plugin's hooks run it.
`reindex` rebuilds the index from the archive and notes. `status` says whether
ele is working here: the data directory, the last sync, the owner here and the
plugin in each profile. `stats` counts what the index holds and its disk use.
`version` prints the release tag. `backup` copies the data directory into a new
or empty directory; see [Data and privacy](data-and-privacy.md#backup).

```text
ele owner add [--move] [--move-to <owner>] [--no-sync] <name>
ele owner add-path [--move] [--move-to <owner>] [--no-sync] <dir> <owner>
ele owner add-project [--move] [--move-to <owner>] [--no-sync] <name> <owner>
ele owner remove-path [--move] [--move-to <owner>] [--no-sync] <dir>
ele owner remove-project [--move] [--move-to <owner>] [--no-sync] <name>
```

Edit the owner map. `--move` refiles the notes of each project that changes owner
under its new owner; `--move-to` names the owner for notes no single owner would
cover. `--no-sync` writes the map without starting a background sync.

## Config

```text
ele config list [--owner <name>]
ele config get [--owner <name>] <key>
ele config set [--owner <name>] <key> <value>
ele config unset [--owner <name>] <key>
```

`list` and `get` show values for the owner here; `set` and `unset` change the
global value unless `--owner` names one, and refuse inside a Claude Code
session. See [Options](data-and-privacy.md#options).

## Used by the skills and hooks

These are called by the plugin and rarely typed. `ele <command> --help` covers
each.

| Command                                         | Called by                                          |
| ----------------------------------------------- | -------------------------------------------------- |
| `ele session-start`                             | SessionStart hook: the block a session opens with  |
| `ele stop-check`                                | Stop hook: the notes ask and the context size hint |
| `ele checkpoint gate`, `ele checkpoint snapshot` | Stop and UserPromptSubmit hooks: the checkpoint gate |
| `ele checkpoint base`, `check`, `close`, `density`, `freeze`, `hash`, `log`, `optout`, `record` | `/ele:checkpoint` |
| `ele checkpoint stats [<days>]`                 | `/ele:checkpoint-stats`                            |
| `ele ticket corpus`, `ele ticket ledger`        | `/ele:ticket-audit` and `/ele:plan-audit`          |
| `ele ticket claims`                             | `/ele:ticket-claims`                               |
| `ele mine`                                      | `/ele:mine`; `ele mine --report` reports what it stored |
| `ele import mining <run>`                       | Nothing: a one-off loader for runs made before `/ele:mine` |
