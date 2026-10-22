# claude-ephant

Cross-session memory for Claude Code. The `ele` binary archives the transcripts
of every profile under `~/.claude` and `~/.claude-*` and indexes them, so a later
session can search what earlier ones discussed. Decisions, open questions and
dated facts are saved as notes, and each session starts with the ones for its
project. The `ele` plugin adds the skills and hooks that use it:

- `/ele:handoff`, `/ele:plan`, `/ele:recall` and `/ele:remember` for memory and
  plans
- `/ele:checkpoint`, a review of the change in subagents before work is reported
  done, with a Stop hook that blocks a done claim made without one
- `/ele:plan-audit`, for stale premises, contradictions, stuck waits and inbox
  triage across the active plans
- `/ele:ticket-audit`, for stale premises and contradictions across a tracker's
  tickets, and `/ele:ticket-claims`, for open GitHub tickets mistyped as a task
- `/ele:checkpoint-stats`, which reports what checkpoints found, and `/ele:mine`,
  which judges past checkpoint runs and review cycles

No model runs in the background: the only model calls are made by skills, inside
the session already running.

## Prerequisites

- `gh` or `glab`, for `/ele:ticket-audit`; `/ele:ticket-claims` needs `gh`.
  Log in with `gh auth login` or `glab auth login` first
- `git`
- Go, at the version in `go.mod`, only to install with `go install` or to work
  from a clone
- `jq` and `shasum`, for the checkpoint hooks
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

3. Name an owner and the directories whose sessions are theirs. A session in no
   owner's directory gets no notes at start:

   ```sh
   ele owner add <name>
   ele owner add-path <dir> <name>
   ele status
   ```

## Update

An `ele` older than the plugin answers the subcommands it lacks with its usage
text, so update the binary first, the way it was installed: `brew upgrade ele`,
the install script again, or `go install …/cmd/ele@latest`. Then:

```text
/plugin marketplace update claude-ephant
/reload-plugins
```

`ele version` prints a released binary's tag, to check against the
latest release. From a clone, `task upgrade` does all of this; see Contributing.

## Options

- **Checkpoint gate**, in `/config` under the `ele` plugin: on by default. Other
  open sessions pick up a change after `/reload-plugins`
- `ELE_HOME` moves the data directory; `ele status` shows where it is

## Contributing

Working from a clone needs [Task](https://taskfile.dev). `task install` builds
and installs `ele` from the clone, and `task check` runs every gate but `audit`;
its lint needs `shellcheck`. CI runs `task check` and `task audit` on every
pull request, and the Release workflow runs them on every push to `main`.
There, release-please keeps a release PR open with the next version and its
changelog, worked out from the conventional commit messages; merging it tags
the release, the binaries are attached to it, and the Tap workflow updates the
formula in `nicolaspearson/homebrew-tap`, then installs the release with
`install.sh` on macOS and Linux and with `brew` on macOS. Run Tap by hand with
a tag, v0.3.0 or later, to re-render the formula for that release. It pushes
with the `HOMEBREW_TAP_TOKEN` secret, a fine-grained token with write access to
that repository's contents alone. It expires 2027-10-21: renew it with the same
scope and run `gh secret set HOMEBREW_TAP_TOKEN` again.

`task upgrade` does the whole update from a clone, with Go, `gh` and `claude`:
it installs the latest release's binary through `install.sh` to `~/.local/bin`
(`ELE_INSTALL_DIR` overrides), then fails unless that copy is the `ele` on
`PATH` and `ele version` prints the release's tag, and names any copy left
behind it on `PATH`. It then updates the marketplace and the plugin; sessions
already open load it after `/reload-plugins`.

`task eval` runs the installed plugin's eval cases with `claude plugin eval`,
and `task eval -- ./plugin` the clone's cases, in `plugin/evals/`.
Each run is a real session on your account and costs model calls, so no gate
runs it.

The specs are in [`.documentation/specs/`](.documentation/specs/).
