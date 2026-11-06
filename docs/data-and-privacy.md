# Data and privacy

## No background model

ele runs no model of its own. Archiving, indexing and search are plain code on
your machine, and the only network calls `ele` makes are the ticket commands'
reads through `gh` or `glab`. The only model calls come from skills you or
Claude invoke, such as `/ele:remember` wording a note or `/ele:checkpoint`'s
reviewers, and they run inside the session already open, on its account.

## Storage

Everything lives in one data directory: by default
`~/Library/Application Support/claude-ephant/` on macOS, and on Linux
`$XDG_DATA_HOME/claude-ephant/`, or `~/.local/share/claude-ephant/` when
`XDG_DATA_HOME` is unset. `ELE_HOME` moves it; `ele status` prints where it is.
`ELE_HOME` must be an absolute path or start with `~/`. ele refuses any other
relative value, and every command and hook fails with an error naming it until
you fix it, because hooks run in each project's directory and a relative value
would give each project its own store.

Earlier releases kept Linux data under the macOS path too. The first
`ele` command after upgrading moves that directory to the new one, once, when
the new one does not exist yet, and the next session start tells you. If the
move fails, for example because the two paths are on different filesystems,
ele keeps using the old path and says so at each session start until you move it
by hand or set `ELE_HOME` to it. To run an older `ele` after the move, set
`ELE_HOME` to the new path: an older binary looks only in the old one. Set
`XDG_DATA_HOME` the same way in your shell and in Claude Code, or leave it
unset in both: each looks only where its own value points. A command that
started just before the move can still write to the old path; ele does not
guard that moment, so anything that command wrote stays in the recreated old
directory, unused.

```text
claude-ephant/
  archive/         transcripts, copied verbatim and compressed
  config.json      options
  ele.db           the search index
  forgotten/       notes ele forget moved out
  forgotten.lock   held while ele moves a forgotten note
  last-sync        when the last sync succeeded
  logs/            checkpoints.jsonl and ticket-audits.jsonl
  mining/          checkpoint mining judgments
  move-notice      the data directory and log moves' outcomes, until a session shows them
  notes/           notes, one Markdown file each, by owner
  notes.lock       held while ele rewrites or moves a note
  owner-map.json   owners, path rules and project entries
  plans/           item names of closed plans
  stop-check/      the Stop hook's state per session
  sync-skipped     files and profiles the last sync could not archive or index, until it skips none
  sync.log         output of background syncs; at 1 MiB it moves to sync.log.1, one kept
```

An `archive/claude-mem/` folder, left by earlier releases, is neither read nor
backed up, and you can delete it.

ele never writes its data into a repository. Plans are the exception by design:
they are files you keep in the repository they describe.

The checkpoint log and the ticket-audit ledger live in `logs/`. Releases before
1.0 kept them in `~/.claude/logs/`; the first `ele` command after upgrading
moves each one, once, when the new one does not exist yet, and the next session
start tells you. With `ELE_HOME` set, ele does not move them; each session
start names the old file until you move it to `logs/` yourself. When both the
old and the new file exist, ele merges neither: each session start names both
until you merge the old one into the new one and delete it. `CHECKPOINT_LOG`
skips the checkpoint log's move. A symlinked `logs/ticket-audits.jsonl` stays a
link, and its target gets each record.

The Stop hook reads each session's transcript from where it last stopped, kept
in `stop-check/`. It stops reading after about 10 seconds a stop, a third of
the hook's 30-second timeout, so a long transcript it has no state for is read
over several stops. Until it catches up, it does not ask for notes and does not
read subagent transcripts, which it reads once it has caught up; the context
size hint still shows.

Outside the data directory, and outside `ele backup`: the hook log,
`~/.claude/logs/hooks-<month>.log`, shared with your own hooks, and the copies
of reviewed files in `~/.claude/cache/checkpoint-reviewed/`. A checkpoint run's
state lives in `claude-checkpoint/` under the system temporary directory.

### The archive

Claude Code deletes old transcripts after its `cleanupPeriodDays` setting. ele
copies every profile's transcripts, under `~/.claude`, `~/.claude-*` and
`CLAUDE_CONFIG_DIR`, into `archive/` as they grow, so it keeps what Claude Code
deletes. It syncs in the background when a session starts and ends, and before
each recall.

A profile elsewhere, such as a temporary `CLAUDE_CONFIG_DIR`, is archived by
the syncs that run under it: its own sessions' hooks, or a shell exporting the
variable. It is named after its folder, and `ele status` lists its plugin state
there too. A sync without the variable leaves what was archived in place. One
whose folder name another profile already has, such as `/elsewhere/.claude-work`
beside `~/.claude-work`, is skipped, and `ele status` lists it. A note written while
`CLAUDE_CONFIG_DIR` names a folder whose name another profile has, skipped or
not yet holding sessions, names its profile by the folder's full path, so it
never reads as the other profile's; one written through a symlink to a profile
names that profile.

A `~/.claude-*` that links to another profile, such as `~/.claude-work` linking
to `~/.claude`, is the same profile, archived once under the folder that is not
a link, or under `~/.claude` when it is one of them. Transcripts an earlier
release archived twice under the link's name stay. A folder whose name another
profile already has, `~/.claude-claude` or a bare `~/.claude-` beside
`~/.claude`, is skipped, and `ele status` lists it.

### The index

`ele.db` is disposable. `ele reindex` rebuilds it from `archive/`, `notes/` and
`owner-map.json`, and a new version of ele that indexes differently rebuilds it
at its first sync.

```sh
ele status    # is ele working here: last sync, owner, plugin enabled per profile
ele stats     # what ele holds: entries, notes, sessions, disk use
```

## Secrets

The archive is a verbatim copy, so it holds whatever the transcripts hold. Its
directory is readable only by you (`0700`, files `0600`). The index, `ele.db`
with its `-wal` and `-shm`, holds conversation text too and is `0600` whatever
the data directory's mode, an `ELE_HOME` you made `0755` included. A file ele
cannot change the mode of, such as one another user owns, is left as it is.

Text is redacted before it is indexed or printed, so neither recall nor the
start block shows a secret:

- Known token formats become `[redacted:<kind>]`: Anthropic and OpenAI keys,
  GitHub tokens, GitLab tokens (`glpat-`, `glrt-`, `gldt-`, `glptt-`,
  `glcbt-`, `glft-`, `gloas-`), AWS access key IDs, JWTs and PEM private keys
- The value of any `key=value` or `key: value` whose key looks secret becomes
  `[redacted:assignment]`: JSON keys, escaped JSON inside a quoted command
  (`\"password\":\"…\"`) and hyphenated header names such as `X-Api-Key`
  included. A quoted value runs to its closing quote up to 4 KiB away, across
  lines; one never closed in that span is redacted to the end of its line

`ele mine --digests` writes each unit's digest, an excerpt of its transcript
redacted as above (prompts, earlier asks, subagent briefs, Bash commands, edited
paths), as a `.txt` file in the directory it is given. `/ele:mine` gives it
`<scratchpad>/mine/<analysis>` in the session scratchpad and hands the files to
its classifier subagents. It deletes only the previous run's
`<analysis>-*.out.jsonl` batch files, so the digests stay there, each rerun
overwriting them, until the scratchpad goes.

Redaction is pattern matching, so a secret in an unknown format gets through,
notes included. Keep secrets out of notes.

## Backup

```sh
ele backup <dir>
```

Copies the archive, notes, forgotten notes, the checkpoint log and ticket-audit
ledger, mining data, config, owner map, closed plans' item names and a
consistent copy of the index into `<dir>`, which must be empty or not exist. It
refuses a directory inside the data directory or inside a git checkout. The
index copy is `0600`, as the index is.

## Options

```sh
ele config list                       # each option, its value here and where it comes from
ele config set <key> <value>          # for every project
ele config set --owner <name> <key> <value>
ele config unset [--owner <name>] <key>
```

`set` and `unset` refuse inside a Claude Code session, `!` commands included, so
type them in a terminal. Open sessions read a change at their next hook. The
block a session starts with names any option set away from its default.

| Option             | Default | What it does                                                                                                                |
| ------------------ | ------- | --------------------------------------------------------------------------------------------------------------------------- |
| `auto_note_writes` | `false` | On, notes ele proposes are written without asking, each reported with its id                                                 |
| `checkpoint_gate`  | `true`  | Off, a turn that changed files and claims done is not held for `/ele:checkpoint`                                          |
| `colour`           | `true`  | Off, the messages ele shows you are plain text, for a client that does not render ANSI colour. `NO_COLOR` turns it off too |
| `compact_hint_at`  | `150000` | Context tokens from which the Stop hook suggests `/compact` or a handoff, again every 100k past it; `0` turns the suggestion off |
| `lone_handoff_pending` | `false` | On, a lone handoff stays pending until you pick or drop it, as several do, instead of showing once |
| `notes_ask`        | `true`  | Off, the Stop hook never asks for notes; `/ele:remember` and `/ele:wrap` still propose them |
| `recipe_dir`       | empty   | An absolute or `~/` path to a folder an approved recipe is written to once, each profile's `skills/<name>` linked to it; empty writes a copy into each profile. `set` refuses a path that is not a folder |
| `recipes`          | `true`  | Off, `/ele:wrap` offers no recipe; a typed `/ele:recipe` still proposes one |
| `reminders`        | `true`  | Off, no reminder shows at session start                                                                                     |
| `review_voice`     | empty   | An absolute or `~/` path to your own voice file, which `/ele:review` drafts comments from in place of the shipped one; set per owner with `--owner`. `set` refuses a path naming no readable file, and `unset` returns to the shipped one |

A `config.json` that does not parse reads as all defaults, so the checkpoint gate
stays on; `ele status` says so.
