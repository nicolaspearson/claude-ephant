# claude-ephant

Cross-session memory for Claude Code. The `ele` binary archives the transcripts
of every profile under `~/.claude` and `~/.claude-*` and indexes them, so a later
session can search what earlier ones discussed. Decisions, open questions and
dated facts are saved as notes, and each session starts with the ones for its
project.

No model runs in the background: the only model calls are made by skills, inside
the session already running. See [Data and privacy](docs/data-and-privacy.md).

## Skills at a glance

The `ele` plugin adds the skills and hooks that use the binary:

- `/ele:handoff`, `/ele:plan`, `/ele:recall` and `/ele:remember` for memory and
  plans
- `/ele:wrap`, which at a session's end asks what to keep as notes, plan inbox
  items or progress lines, which due reminders to close, and whether to hand off
- `/ele:checkpoint`, a review of the change in subagents before work is reported
  done, with a Stop hook that blocks a done claim made without one
- `/ele:plan-audit`, for stale premises, contradictions, stuck waits and inbox
  triage across the active plans
- `/ele:ticket-audit`, for stale premises and contradictions across a tracker's
  tickets, and `/ele:ticket-claims`, for open GitHub tickets mistyped as a task
- `/ele:checkpoint-stats`, which reports what checkpoints found, and `/ele:mine`,
  which judges past checkpoint runs and review cycles

## Prerequisites

- `gh` or `glab`, for `/ele:ticket-audit`; `/ele:ticket-claims` needs `gh`.
  Log in with `gh auth login` or `glab auth login` first
- `git`
- Go, at the version in `go.mod`, only to install with `go install`
- `jq`, for `/ele:ticket-audit` against GitLab
- `curl`, `tar` and `sha256sum` or `shasum`, for the install script

## Install

1. Install the binary. The plugin's hooks call `ele` from `PATH`, so it must be
   on `PATH` before the plugin is enabled. Releases carry macOS and Linux
   binaries for amd64 and arm64. With Homebrew, on macOS or Linux:

   ```sh
   brew install nicolaspearson/tap/ele
   ```

   Or with the install script, which checks the download against the release's
   `checksums.txt` and puts `ele` in `~/.local/bin`, saying so if that is not
   on `PATH`. `ELE_INSTALL_DIR` picks another directory, `ELE_VERSION` a
   release tag:

   ```sh
   curl -fsSL https://raw.githubusercontent.com/nicolaspearson/claude-ephant/main/install.sh | sh
   ```

   Or with Go, whose bin directory (`go env GOBIN`, else `bin` under the first
   entry of `go env GOPATH`) must be on `PATH`:

   ```sh
   go install github.com/nicolaspearson/claude-ephant/cmd/ele@latest
   ```

2. In Claude Code, add the marketplace and install the plugin:

   ```text
   /plugin marketplace add nicolaspearson/claude-ephant
   /plugin install ele@claude-ephant
   ```

## Quick start

1. Name an owner and the directories whose sessions are theirs, in a terminal.
   A session in no owner's directory starts with a line saying so instead of
   notes; see [Owners](docs/owners.md):

   ```sh
   ele owner add personal
   ele owner add-path ~/code personal
   ele status
   ```

2. Work in a session under that directory. When a decision is made, say
   "remember that…", or approve a note ele proposes after a stretch of work; see
   [Notes](docs/notes-and-recall.md#notes)
3. A later session in that project starts with the note. To find something no
   note kept, ask "did we already…" or run `/ele:recall <words>`; see
   [Recall](docs/notes-and-recall.md#recall)
4. Before `/clear`, run `/ele:handoff`; after it, type `continue` and the new
   session picks up where the last stopped; see
   [Handoffs](docs/handoffs-and-reminders.md#handoffs)

[Plans](docs/plans.md), [Checkpoint and ticket-audit](docs/checkpoint-and-ticket-audit.md)
and [Review](docs/review.md) cover the rest, and the [command reference](docs/commands.md) lists every
command.

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
- **`reminders`**, on: shows due reminders at session start

## Update

An `ele` older than the plugin answers the subcommands it lacks with its usage
text, so update the binary first, the way it was installed: `brew upgrade ele`,
the install script again, or `go install …/cmd/ele@latest`. Then:

```text
/plugin marketplace update claude-ephant
/reload-plugins
```

`ele version` prints a released binary's tag, to check against the
latest release.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
