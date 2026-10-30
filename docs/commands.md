# Command reference

`ele --help` lists every command with its flags; `ele <command> --help` says
what each flag does, though not always its default. A misused command prints
its own help and exits 2.

Where a command resolves an owner or project, it takes them from the working
directory. `--owner <name>` and `--project <name>` name others, so the command
works from any directory; see [Owners](owners.md).

## Skills and slash commands

| Command                 | What it does                                                                                     |
| ----------------------- | ------------------------------------------------------------------------------------------------ |
| `/ele:checkpoint`       | Review a change in subagents before it is reported done; see [Checkpoint](checkpoint-and-ticket-audit.md#checkpoint) |
| `/ele:checkpoint-stats` | What checkpoints found and which stage found it, then reviews apart, over the last 30 days or the number given |
| `/ele:flag`             | Flag a moment of the session as good or bad with why, or `list` the project's flags; see [Flags](notes-and-recall.md#flags) |
| `/ele:handoff`          | Write the handoff the next session opens with, or `list` pending and closed ones; see [Handoffs](handoffs-and-reminders.md#handoffs) |
| `/ele:mine`             | Judge past checkpoint runs and review cycles, for tuning the checkpoint skill                    |
| `/ele:plan`             | Pick up the active plan, or `new <name>` or `convert <spec>`; see [Plans](plans.md)              |
| `/ele:plan-audit`       | Audit active plans against the code and each other, `drafts` before a start, or `close` a plan's Decisions |
| `/ele:recall`           | Search earlier sessions; see [Recall](notes-and-recall.md#recall)                                 |
| `/ele:recipe`           | Keep a procedure the session ran as a skill of your own, or `audit` your skills for stale commands; see [Recipes](recipes.md) |
| `/ele:remember`         | Save a decision, question, fact or reminder                                                       |
| `/ele:review`           | Review an MR, PR, branch or commit and publish the findings you pick, major ones first; see [Review](review.md) |
| `/ele:setup`            | Map a directory no owner covers to an owner, proposing the `ele owner` commands and running them once approved; see [Owners](owners.md#setting-up) |
| `/ele:ticket-audit`     | Audit a tracker's ticket map against the code                                                     |
| `/ele:ticket-claims`    | Flag open GitHub tickets mistyped as a task or granting themselves execution                      |
| `/ele:wrap`             | At a session's end, ask what to keep, a recipe included, which reminders to close and whether to hand off |

## Notes

```text
ele remember --kind decision|fact|handoff|question|reminder [--done-file <path>] [--due <YYYY-MM-DD[THH:MM]>] [--every <interval>] [--everywhere | --global] [--owner <name>] [--project <name>] [--session <id>] [--supersedes <id>...] <words>
```

Save a note of the `--kind` given. `--global` applies it to every project of
the owner; `--everywhere` to every session, whatever its owner, owners added
later included. `--supersedes` replaces an earlier note; repeat it for several.
`--session` records the session the note came from, by default the running one.
For a reminder, `--due` is the local date, or date and time, it first shows
from, today by default with `--every`; `--every <interval>` makes it a nudge,
hidden that long after each close: `5m`, `2h`, `7d`, or a bare number of days.
`--done-file`, which needs `--every`, ties the nudge to a file's last change.

```text
ele context [--owner <name>] [--project <name>]
ele context --size
```

Every live note and pending handoff for the project, with no budget. `--size`
prints the running session's context in tokens instead.

```text
ele flag bad [--owner <name>] [--project <name>] [--session <id> [--at <time>]] <why>
ele flag good [--owner <name>] [--project <name>] [--session <id> [--at <time>]] <why>
ele flag list [--owner <name>] [--project <name>]
```

Flag the running session at this moment as good or bad, with why. `--session`
flags an earlier one by id, or by a prefix only one indexed session has, at its
last entry, or at `--at <time>`: an RFC 3339 time inside the session, such as
an entry's timestamp from `ele show`. `list` shows the project's flags, newest
first, each with the time `ele show <session> --at <time>` opens it at. The
start block never shows a flag.

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
marks handoffs picked up or dropped; `reopen` makes them pending, so the next
startup or `/clear` shows them.

```text
ele reminder list [--all] [--colour] [--owner <name>] [--project <name>]
ele reminder close [--owner <name>] [--project <name>] <id>...
ele reminder reopen [--owner <name>] [--project <name>] <id>...
```

Open reminders for the project and the owner's global ones, soonest due first;
`--all` adds closed and superseded ones, and `--colour` highlights the list, for
`! ele reminder list --colour`, unless the colour option is off or `NO_COLOR` is
set. `close` marks reminders done, or re-arms a nudge; `reopen` opens them
again, so the start block shows them from their due date.

## Plans

```text
ele plan start <path>
ele plan show [<path>]
ele plan list [--all] [--owner <name>] [--project <name>]
ele plan check [<path>]
ele plan done [--owner <name>] [--project <name>] <path|id>...
```

`start` makes a plan file active, refusing an item name another plan holds and
naming each plan it could not check for one. `show` prints its counts, previous item,
blocked items and next item, the project's active plan by default. `list` lists
active plans; `--all` adds done and superseded ones. `check` lists a plan's
warnings with their lines, every active plan's with no path, and exits 0 on
warnings; `ele plan check --help` lists what it warns on. `done` closes plans
and keeps their item names, so run it from each plan's checkout once its last
item is done; it closes a plan with items still open.
Given a closed plan with no readable record of its names, `done` stores one from
the plan's file or last commit.

## History

```text
ele recall [--all | --owner <name>] [--kind <k,k>] [--project <name>] [--since <YYYY-MM-DD|Nd>] [--limit <n>] [--no-sync] <query>
```

Search past sessions and notes; every word must appear. `--owner` searches that
owner, `--all` every owner, and `--project` only that project, under its own
owner unless `--owner` or `--all` is given. `--kind` takes `note`, `prompt`,
`heads-up`, `conversation` and `tool`, comma-separated. `--since` takes a date
or a number of days back, as `7d`. `--limit` caps the matches, 20 by default.
`--no-sync` searches without syncing first.

```text
ele show [--around <text> | --at <time>] [--context <n>] <session>
ele show --note <id> [--context <n>]
```

A session's entries, centred on the first containing `--around`'s text or at or
after `--at`'s time, with `--context` entries either side, 4 by default.
`--note` opens the session a note came from at its discussion, with 1 entry
either side of each part of it unless `--context` is set, and twice that many
before the note when nothing matches it.

```text
ele events [--during <skill>] [--kind <kind>] [--name <name>] [--owner <name>] [--project <name>] [--session <id>] [--since <YYYY-MM-DD|Nd>] [--limit <n>] [--no-sync]
```

Skill calls, typed commands, subagent launches, hook blocks, compactions,
injected context, replies, token use and tool errors, one JSON object per line,
the newest 200 by default; `--limit 0` prints all. `--kind` keeps one kind:
`agent`, `command`, `compaction`, `hook-block`, `injection`, `reply`, `skill`,
`tokens` or `tool-error`. `--name` keeps one skill, command, subagent type or
hook, an injection's with its matcher as `PostToolUse:Edit`, or `skills` for
the skill list. `--during <skill>` keeps only events inside a run of that skill
or command. `--owner`, `--project`, `--session` and `--since` narrow to one
owner, project, session or window; `--no-sync` reads without syncing first.

```text
ele usage [--owner <name>] [--project <name>] [--since <YYYY-MM-DD|Nd>] [--no-sync]
```

Sessions, prompts, tool calls, bytes injected by hooks and the skill list, reply
length and API tokens, main thread and subagents apart. `--owner`, `--project`
and `--since` narrow it; `--no-sync` reads without syncing first.

```text
ele stop-check --replay --since <time> [--until <time>]
```

Each Stop of the archived main transcripts with a line in the window, one JSON
object per line: whether the notes ask fired (`asked`), whether the hook as
built would ask (`asks`), and the work that set it pending (`trigger`) or why
not (`whyNot`: `hook-active`, `checkpoint`, `review`, `proposal`,
`nothing-pending` or `stretch`). `stop` is `recorded` where Claude Code wrote
the stop down and `placed`, approximately, where an older transcript leaves it
to be inferred. The window is `[--since, --until)`, each a date or RFC 3339
time as `2026-10-20T04:42:00Z`; `--until` defaults to now. Syncs first.

## Index

```text
ele sync [--background]
ele reindex
ele status
ele stats
ele version
ele backup <dir>
```

`sync` archives new transcripts and updates the index; the plugin's hooks run it,
and `--background` starts it detached and returns. `reindex` rebuilds the index
from the archive and notes. `status` says whether ele is working here: the data
directory, the sync state, the owner here, the plugin in each profile and the
sync log. `stats` counts what the index holds and its disk use. `version` prints
the release tag stamped at build, else the version Go stamped, or `devel`.
`backup` copies the archive, notes, forgotten notes, mining runs, config, owner
map, closed plans' item names and the index into a new or empty directory; see
[Data and privacy](data-and-privacy.md#backup).

```text
ele owner add [--move] [--move-to <owner>] [--no-sync] <name>
ele owner add-path [--move] [--move-to <owner>] [--no-sync] <dir> <owner>
ele owner add-project [--move] [--move-to <owner>] [--no-sync] <name> <owner>
ele owner remove-path [--move] [--move-to <owner>] [--no-sync] <dir>
ele owner remove-project [--move] [--move-to <owner>] [--no-sync] <name>
```

Edit the owner map, printing the projects that change owner. `add` lists a new
owner; `add-path` maps a directory and everything under it to an owner;
`add-project` maps a project ahead of any path rule; the `remove-` forms drop a
rule. `--move` refiles the notes of each project that changes owner
under its new owner; `--move-to` names the owner for notes no single owner would
cover. `--no-sync` writes the map without starting a background sync.

## Config

```text
ele config list [--owner <name>]
ele config get [--owner <name>] <key>
ele config set [--owner <name>] <key> <value>
ele config unset [--owner <name>] <key>
```

`list` and `get` show values for the owner here: the owner's override, else the
global value, else the default, and `list` also shows any stored key no option
takes. A `config.json` that does not parse reads as defaults, said on stderr.
`set` and `unset` change the global value unless `--owner` names one, and refuse
inside a Claude Code session; `unset` lets the next level down apply. See
[Options](data-and-privacy.md#options).

## Used by the skills and hooks

These are called by the plugin and rarely typed. `ele <command> --help` covers
each.

| Command                                         | Called by                                          |
| ----------------------------------------------- | -------------------------------------------------- |
| `ele session-start`                             | SessionStart hook: the block a session opens with  |
| `ele stop-check`                                | Stop hook: the notes ask and the context size hint; `--replay` replays it, under [History](#history) |
| `ele checkpoint gate`, `ele checkpoint snapshot` | Stop and UserPromptSubmit hooks: the checkpoint gate |
| `ele checkpoint base`, `check`, `close`, `density`, `freeze`, `hash`, `log`, `optout`, `record` | `/ele:checkpoint` |
| `ele checkpoint stats [--since <YYYY-MM-DD\|Nd>]` | `/ele:checkpoint-stats`                            |
| `ele ticket corpus`, `ele ticket ledger`        | `/ele:ticket-audit` and `/ele:plan-audit`          |
| `ele ticket claims`                             | `/ele:ticket-claims`                               |
| `ele mine`                                      | `/ele:mine`; `ele mine --report` reports what it stored |

```text
ele session-start [--no-sync]
ele stop-check
ele checkpoint base [<root>]
ele checkpoint check [<root>]
ele checkpoint close [<root>]
ele checkpoint density [--ref <ref>] [<root>]
ele checkpoint freeze <root> <dir> [--base <commit>] [--start] -- <path>...
ele checkpoint gate < <stop-hook.json>
ele checkpoint hash [<root>]
ele checkpoint log [<root>] < <record.json>
ele checkpoint optout <root> <glob>...
ele checkpoint record [<root>]
ele checkpoint snapshot < <prompt-hook.json>
ele checkpoint stats [--since <YYYY-MM-DD|Nd>]
ele ticket claims [--quiet] [<number>...]
ele ticket corpus --compare <number> --expect <file> [--note] [--cli glab|gh]
ele ticket corpus --map <number> [--tickets <numbers>] --out <directory> [--clean] [--cli glab|gh]
ele ticket corpus --plan <file> [--plan <file>...] --out <directory> [--clean]
ele ticket corpus --tickets <numbers> --out <directory> [--clean] [--cli glab|gh]
ele ticket ledger record --manifest <file> --records <file> [--ledger <file>]
ele ticket ledger rulings --manifest <file> [--ledger <file>]
ele mine [--digests <dir>] [--units <file>] [--no-sync] checkpoint-runs|review-cycles
ele mine --judgments <file> --prompt <file> --digests <dir> [--no-sync] checkpoint-runs|review-cycles
ele mine --report [--no-sync]
```

`session-start` prints the block a session opens with and starts a background
sync, which `--no-sync` skips. `stop-check` is the Stop hook's notes ask and
context size hint.

Where a `checkpoint` command's `<root>` is optional, it defaults to the working
directory's repository. `base` prints the merge-base a branch checkpoint diffs
against. `check` exits 0 when every changed path is what the last checkpoint
recorded, and 1 when anything changed since, none ran or nothing is changed.
`freeze` writes each path's diff against `HEAD`, or `--base`, into `<dir>` with
a fingerprint, and opens or extends the session's run, `--start` restarting its
clock. `hash` prints the fingerprint of the changed paths, for comparing with a
freeze's. `record` marks what the run froze as reviewed and ends the run;
`close` ends it without a record. `log` appends a checkpoint's or review's
record, read on stdin. `optout` exempts paths matching each glob from the gate
for the session. `density` measures the comment share of the diff since
`--ref`. `gate` and `snapshot` read their hook's input on stdin. `stats`
reports what checkpoints found since `--since`, 30 days back by default, and
takes a bare number of days in its place until 1.0.
`ticket claims` flags open GitHub tickets labelled `wayfinder:map` that grant
themselves execution and `wayfinder:task` tickets naming a source path, only the
numbered ones when given; `--quiet` prints nothing when none is flagged.

`ticket corpus` writes tickets into `--out`, `--clean` emptying it first: a
map's children with `--map`, narrowed by `--tickets`; a list of tickets with
`--tickets`, comma-separated; or each Actions and Inbox item of the `--plan`
files. `--cli` picks `glab` or `gh` over the one the remote names. `--compare`
reads a ticket back after a write against the text in `--expect`, its body or,
with `--note`, a comment.

`ticket ledger record` appends the audit's `--records` for the corpus's
`--manifest`; `rulings` prints earlier rulings on the manifest's items
and whether their text changed since; `--ledger` names the ledger file,
`logs/ticket-audits.jsonl` in the data directory by default.

`mine` lists the unjudged units of an analysis, `checkpoint-runs` or
`review-cycles`, or the units `--units` names, judged or not, writing each
digest into `--digests`. With `--judgments`, a file of classifications, and
`--prompt`, the classifier prompt they were made with, it stores them, reading
the digests back from `--digests`. `--report` reports figures from the last
judgment stored per unit, both analyses together. `--no-sync` reads without
syncing first.
