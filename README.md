# claude-ephant

Memory and a review workflow for Claude Code, kept on your own machine.

Every Claude Code session starts from nothing, and Claude Code deletes old
transcripts. `ele` archives them, indexes them for search, and opens each
session with the decisions, open questions and handoffs of its project. The
`ele` plugin adds the skills that plan work across sessions and review it
before it is called done.

## Why ele

### It remembers, so you never re-explain

- **Every session starts briefed.** Each one opens with its project's due
  reminders, open questions and newest decisions, so Claude knows what was
  settled before you type a word
- **Everything you discuss, searchable.** Ask "did we already decide how
  retries work?" and `/ele:recall` searches every session of the work: what
  you typed, Claude's replies and the commands it ran. Then it opens the
  discussion that settled it, even months after Claude Code's
  `cleanupPeriodDays` deleted the transcript, since ele copies each one as it
  grows
- **Every account, one memory.** Run Claude Code under several accounts or
  profiles (`~/.claude`, `~/.claude-work`, `CLAUDE_CONFIG_DIR`) and ele
  archives them all into one store. A note, handoff or plan made under one is
  there when you open the same project under another, and recall searches
  them together
- **`/clear` without losing your place.** A handoff carries where the work
  stands into the next session: type `continue` and it picks up. With several
  streams of work, none is dropped
- **Reminders that find you.** "Remind me on Friday" shows at every session
  start from Friday until you close it; a nudge comes back on its interval
- **It watches your context.** By default, past 150k tokens it suggests
  `/compact` or a handoff, before a long context costs you on every turn
- **Clients and your own work kept apart.** A client's sessions never surface in
  your personal recall, and their notes never reach your sessions
- **Procedures become skills.** `/ele:recipe` keeps a procedure a session
  worked out as a skill of your own, and audits your skills for commands that
  no longer exist

### It plans and reviews, so done means done

- **A fresh-eyes review before work is called done.** `/ele:checkpoint` hands
  the change to subagents that never saw it written, for bugs, conventions and
  prose, and the spec when there is a ticket or plan item. It works out what
  your CI runs and runs it, makes every new test fail once to prove it can,
  and re-reviews its own fixes. A Stop hook holds a "done" claimed over
  changed files without one
- **Plans that survive `/clear`.** `/ele:plan` works a plan one item a
  session, ele keeping your place, and `/ele:plan-audit` finds the premises the
  code has since moved past
- **Small tasks, agreed and checked.** `/ele:dev` agrees what done looks like,
  explores the code in subagents before asking you anything, weighs designs
  from separate architects where there is a real choice, and ends in a
  checkpoint
- **Merge requests reviewed in your voice.** `/ele:review`, experimental, runs
  the same reviewers over someone else's MR or PR, major findings first, and
  posts only what you pick
- **Tickets checked against the code.** `/ele:ticket-audit` finds the tickets
  whose premises the code no longer holds, before you pick one up

### It is yours

- **Your data stays yours.** The archive, index and notes live in one
  directory on your machine, the archive and index readable only by you. ele
  has no server, no account and no telemetry, and sends nothing to its
  maintainers. What a skill reads into your session reaches Anthropic, as
  anything in a Claude Code session does; see
  [What leaves your machine](docs/data-and-privacy.md#what-leaves-your-machine)
- **No background model.** Archiving, indexing and search are plain code. A
  model runs only when a skill does, inside the session you already have open
- **Secrets redacted.** Known token formats, bearer tokens, passwords in URLs
  and secret-looking values are redacted before anything is indexed or shown;
  see [Secrets](docs/data-and-privacy.md#secrets)
- **You can help improve it, if you choose.** `/ele:mine` and
  `/ele:checkpoint-stats` measure, on your machine, how well the reviewers do.
  The reports are mostly counts; you read one, cut what you would rather keep,
  and decide whether to share it; see [Improving ele](docs/improving-ele.md)

### In the author's own use

As of 2026-11-14, on the author's own work across two Claude Code accounts:

- 790 sessions and 2,224 notes since 2026-09-09, kept apart under three owners
- 1,635 checkpoints in the month since the checkpoint log began on 2026-10-15,
  fixing 9,795 findings before the work was called done, with 823 more checked
  and refuted
- Re-reviews of those fixes caught 1,831 problems a fix had introduced, and 261
  the first review missed
- 6,518 mutations of code under new tests, 6,129 of which made the test fail
  as it should

## Skills at a glance

### Memory

| Skill           | What it does                                                                         | Guide                                                     |
| --------------- | ------------------------------------------------------------------------------------ | --------------------------------------------------------- |
| `/ele:flag`     | Mark a moment as good or bad, with why, for later study                              | [Flags](docs/notes-and-recall.md#flags)                   |
| `/ele:handoff`  | Write the note the session after `/clear` opens with                                 | [Handoffs](docs/handoffs-and-reminders.md#handoffs)       |
| `/ele:recall`   | Search what earlier sessions discussed                                               | [Recall](docs/notes-and-recall.md#recall)                 |
| `/ele:remember` | Save a decision, open question, dated fact or reminder                               | [Notes](docs/notes-and-recall.md#saving-one)              |
| `/ele:setup`    | Map a directory to an owner, after installing or in a new repository                 | [Owners](docs/owners.md#setting-up)                       |
| `/ele:wrap`     | At a session's end, ask what to keep, which reminders to close, whether to hand off  | [Wrapping up](docs/handoffs-and-reminders.md#wrapping-up) |

### Planning and building

| Skill             | What it does                                                                     | Guide                                          |
| ----------------- | -------------------------------------------------------------------------------- | ---------------------------------------------- |
| `/ele:dev`        | Take a task too small for a plan from an agreed done-when to a checkpoint        | [Dev](docs/dev.md)                             |
| `/ele:plan`       | Work a plan one item a session, or write a new one                               | [Plans](docs/plans.md)                         |
| `/ele:plan-audit` | Find stale premises, contradictions and stuck waits across plans                 | [Auditing plans](docs/plans.md#auditing-plans) |
| `/ele:recipe`     | Keep a procedure you will run again as a skill of your own, or audit your skills | [Recipes](docs/recipes.md)                     |

### Review

| Skill                | What it does                                                                       | Guide                                                              |
| -------------------- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `/ele:checkpoint`    | Review a change in subagents before it is reported done                            | [Checkpoint](docs/checkpoint-and-ticket-audit.md#checkpoint)       |
| `/ele:review`        | Review an MR, PR, branch or commit and publish the findings you pick; experimental | [Review](docs/review.md)                                           |
| `/ele:ticket-audit`  | Check a tracker's tickets against the code                                         | [Ticket-audit](docs/checkpoint-and-ticket-audit.md#ticket-audit)   |
| `/ele:ticket-claims` | Find open GitHub tickets mistyped as a task                                        | [Ticket claims](docs/checkpoint-and-ticket-audit.md#ticket-claims) |

### Improving ele

| Skill                   | What it does                                           | Guide                                                                                |
| ----------------------- | ------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| `/ele:checkpoint-stats` | Report what checkpoints found and which stage found it | [What checkpoints found](docs/checkpoint-and-ticket-audit.md#what-checkpoints-found) |
| `/ele:mine`             | Judge past checkpoint runs and review cycles           | [Mining](docs/improving-ele.md#mining)                                               |

Claude starts most of these when what you say calls for one. `/ele:plan`,
`/ele:dev`, `/ele:review`, `/ele:ticket-claims`, `/ele:checkpoint-stats` and
`/ele:mine` run only when you type them.

## Prerequisites

- Claude Code 2.1.163 or later, the first to read the context the Stop hooks
  keep a turn going with
- `gh` or `glab`, for `/ele:ticket-audit` and `/ele:review`;
  `/ele:ticket-claims` needs `gh`.
  Log in with `gh auth login` or `glab auth login` first
- `git`
- Go, at the version in `go.mod`, only to install with `go install`
- `jq`, for `/ele:ticket-audit` against GitLab
- `curl`, `tar` and `sha256sum` or `shasum`, for the install script

## Install

1. Install the binary. The plugin's hooks call `ele` from `PATH`, so it must be
   on `PATH` before the plugin is enabled. Releases carry binaries for the
   platforms under [Compatibility](#compatibility). With Homebrew, on macOS or
   Linux:

   ```sh
   brew install nicolaspearson/tap/ele
   ```

   Or with the install script, which checks the download against the release's
   `checksums.txt` and puts `ele` in `~/.local/bin`, saying so if that is not
   on `PATH`. `ELE_INSTALL_DIR` picks another directory, `ELE_VERSION` a
   release tag, its `v` optional:

   ```sh
   curl -fsSL https://raw.githubusercontent.com/nicolaspearson/claude-ephant/main/install.sh | sh
   ```

   Or with Go, whose bin directory (`go env GOBIN`, else `bin` under the first
   entry of `go env GOPATH`) must be on `PATH`, `<tag>` being the tag of the
   release marked Latest on the
   [Releases page](https://github.com/nicolaspearson/claude-ephant/releases):

   ```sh
   go install github.com/nicolaspearson/claude-ephant/cmd/ele@<tag>
   ```

   Not `@latest`: a release's tag is pushed while it is still a draft, so
   `@latest` can install a draft.

2. In Claude Code, add the marketplace and install the plugin:

   ```text
   /plugin marketplace add nicolaspearson/claude-ephant
   /plugin install ele@claude-ephant
   ```

## Quick start

1. Name an owner and the directories whose sessions are theirs. A session in no
   owner's directory starts with a line saying so, and only the notes made
   with `--everywhere`; run `/ele:setup` there, which proposes the `ele owner`
   commands that map it and runs them once you approve. See
   [Owners](docs/owners.md)
2. Work in a session under that directory. When a decision is made, say
   "remember that…", or approve a note ele proposes after a stretch of work; see
   [Notes](docs/notes-and-recall.md#notes)
3. A later session in that project starts with the note. To find something no
   note kept, ask "did we already…" or run `/ele:recall <words>`; see
   [Recall](docs/notes-and-recall.md#recall)
4. Before `/clear`, run `/ele:handoff`; after it, type `continue` and the new
   session picks up where the last stopped; see
   [Handoffs](docs/handoffs-and-reminders.md#handoffs)

## Guides

- [Owners](docs/owners.md): which sessions and notes belong to which project
- [Notes and recall](docs/notes-and-recall.md): saving, changing and finding
  what was decided
- [Handoffs and reminders](docs/handoffs-and-reminders.md): carrying work
  across `/clear`, and dated reminders
- [Plans](docs/plans.md): work that spans sessions
- [Dev](docs/dev.md): one task from done-when to checkpoint
- [Recipes](docs/recipes.md): procedures kept as your own skills
- [Checkpoint and ticket-audit](docs/checkpoint-and-ticket-audit.md): review
  before done, and tickets checked against the code
- [Review](docs/review.md): merge requests and pull requests, experimental
- [Data and privacy](docs/data-and-privacy.md): what is stored, where, what is
  redacted, what leaves your machine, and which models the subagents run on
- [Improving ele](docs/improving-ele.md): measuring the skills on your own data,
  and sharing it if you choose
- [Command reference](docs/commands.md): every command and flag

## Options

`ele config list` shows each option, its value here and where that value comes
from; `ele config set <key> <value>` sets one, typed in a terminal. See
[Options](docs/data-and-privacy.md#options) for `--owner` overrides and what
each does.

- **`auto_note_writes`**, off: writes the notes ele proposes without asking
- **`checkpoint_gate`**, on: stops a turn that changed files and claims done
  without a checkpoint, once per turn, to ask for `/ele:checkpoint`
- **`colour`**, on: highlights the messages ele shows you; `NO_COLOR` turns it
  off
- **`compact_hint_at`**, 150000: context tokens from which ele suggests
  `/compact` or a handoff; 0 turns it off
- **`lone_handoff_pending`**, off: keeps a lone handoff pending until you pick
  or drop it, instead of showing it once
- **`notes_ask`**, on: asks for notes once per stretch of work, and again an
  hour after asking if work has happened since
- **`recipe_dir`**, empty: a folder an approved recipe is written to once,
  linked from each profile
- **`recipes`**, on: `/ele:wrap` offers a procedure the session ran as a skill
- **`reminders`**, on: shows due reminders at session start
- **`report_voice`**, empty: your own voice file for `/ele:write-report`'s
  reports
- **`review_voice`**, empty: your own voice file for `/ele:review`'s comments

## Update

An `ele` older than the plugin answers the subcommands it lacks with its usage
text, so update the binary first, the way it was installed: `brew upgrade ele`,
the install script again, or `go install …/cmd/ele@<tag>` with the latest
release's tag. Then:

```text
/plugin marketplace update claude-ephant
/reload-plugins
```

`ele version` prints a released binary's tag, to check against the
latest release.

## Compatibility

`ele` runs on macOS and Linux, on amd64 and arm64. There is no Windows build.

From 1.0.0, `ele` follows semantic versioning. Within a major version, a
release does not break:

- its commands and flags
- the data directory and the note format; a later version migrates them
  forward itself
- the behaviour of the `ele` commands the plugin's hooks call

The hook log under `~/.claude/logs`, shared with your own hooks, and
`/ele:checkpoint`'s review state, a cache, live outside the data directory and
outside this promise.

The plugin is not covered: it installs from `main`, so `hooks.json` and the
skills' wording change with any commit. Keep `ele` at least as new as the
plugin; see [Update](#update).

## Contributing

Code: see [CONTRIBUTING.md](CONTRIBUTING.md). Without writing any, a report of
how the checkpoint reviewers did on your own work helps tune them for
everyone; [Improving ele](docs/improving-ele.md) says how to make one, what it
holds, and how to share it.
