# Data and privacy

## No background model

ele runs no model of its own. Archiving, indexing and search are plain code on
your machine, and the only network calls `ele` makes are the ticket commands'
reads through `gh` or `glab`. The only model calls come from skills you or
Claude invoke, such as `/ele:remember` wording a note or `/ele:checkpoint`'s
reviewers, and they run inside the session already open, on its account.

## Storage

Everything lives in one data directory, `~/Library/Application Support/claude-ephant/`
by default. `ELE_HOME` moves it; `ele status` prints where it is.

```text
claude-ephant/
  archive/         transcripts, copied verbatim and compressed
  config.json      options
  ele.db           the search index
  forgotten/       notes ele forget moved out
  last-sync        when the last sync succeeded
  mining/          checkpoint mining judgments
  notes/           notes, one Markdown file each, by owner
  owner-map.json   owners, path rules and project entries
  plans/           item names of closed plans
  stop-check/      the Stop hook's state per session
  sync.log         output of background syncs
```

ele never writes its data into a repository. Plans are the exception by design:
they are files you keep in the repository they describe.

The checkpoint and ticket-audit skills keep their logs under `~/.claude/`:
`logs/checkpoints.jsonl`, `logs/ticket-audits.jsonl`, `logs/hooks-<month>.log`,
and the copies of reviewed files in `cache/checkpoint-reviewed/`. A checkpoint
run's state lives in `claude-checkpoint/` under the system temporary directory.

### The archive

Claude Code deletes old transcripts after its `cleanupPeriodDays` setting. ele
copies every profile's transcripts, under `~/.claude` and `~/.claude-*`, into
`archive/` as they grow, so it keeps what Claude Code deletes. It syncs in the
background when a session starts and ends, and before each recall.

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
directory is readable only by you (`0700`, files `0600`).

Text is redacted before it is indexed or printed, so neither recall nor the
start block shows a secret:

- Known token formats become `[redacted:<kind>]`: Anthropic and OpenAI keys,
  GitHub and GitLab tokens, AWS access key IDs, JWTs and PEM private keys
- The value of any `key=value` or `key: value` whose key looks secret, JSON keys
  included, becomes `[redacted:assignment]`

Redaction is pattern matching, so a secret in an unknown format gets through,
notes included. Keep secrets out of notes.

## Backup

```sh
ele backup <dir>
```

Copies the archive, notes, forgotten notes, mining data, config, owner map and a
consistent copy of the index into `<dir>`, which must be empty or not exist. It
refuses a directory inside the data directory or inside a git checkout. It does
not copy `plans/`, so a restored backup has lost closed plans' item names.

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
| `checkpoint_gate`  | `true`  | Off, a turn that changed files and claims done is not blocked for `/ele:checkpoint`                                          |
| `colour`           | `true`  | Off, the messages ele shows you are plain text, for a client that does not render ANSI colour. `NO_COLOR` turns it off too |
| `reminders`        | `true`  | Off, no reminder shows at session start                                                                                     |
| `review_voice`     | empty   | An absolute or `~/` path to your own voice file, which `/ele:review` drafts comments from in place of the shipped one; set per owner with `--owner`. `set` refuses a path naming no readable file, and `unset` returns to the shipped one |

A `config.json` that does not parse reads as all defaults, so the checkpoint gate
stays on; `ele status` says so.
