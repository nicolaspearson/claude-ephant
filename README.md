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
- `/ele:ticket-audit`, for stale premises and contradictions across tickets or a
  plan, and `/ele:ticket-claims`, for open GitHub tickets mistyped as a task
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

## Install

1. Install the binary. The plugin's hooks call `ele` from `PATH`, so it must be
   on `PATH` before the plugin is enabled. Releases carry macOS and Linux
   binaries for amd64 and arm64; this puts the latest in `~/.local/bin`, which
   must then be on `PATH`:

   ```sh
   os=$(uname -s | tr '[:upper:]' '[:lower:]')
   arch=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
   curl -fsSL "https://github.com/nicolaspearson/claude-ephant/releases/latest/download/ele_${os}_${arch}.tar.gz" | tar -xz
   mkdir -p ~/.local/bin && mv "ele_${os}_${arch}/ele" ~/.local/bin/ && rm -r "ele_${os}_${arch}"
   ```

   Each release lists the tarballs' SHA-256 sums in `checksums.txt`. With Go
   instead, Go's bin directory (`go env GOBIN`, else `bin` under the first entry
   of `go env GOPATH`) must be on `PATH`:

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
text, so update the binary first, the way it was installed: the download in
Install's first step again, or `go install …/cmd/ele@latest`. Then:

```text
/plugin marketplace update claude-ephant
/reload-plugins
```

## Options

- **Checkpoint gate**, in `/config` under the `ele` plugin: on by default. Other
  open sessions pick up a change after `/reload-plugins`
- `ELE_HOME` moves the data directory; `ele status` shows where it is

## Contributing

Working from a clone needs [Task](https://taskfile.dev). `task install` builds
and installs `ele` from the clone, and `task check` runs every gate but `audit`.
CI runs `task check` and `task audit` on every pull request, and the Release
workflow runs them on every push to `main`. There, release-please keeps a
release PR open with the next version and its changelog, worked out from the
conventional commit messages; merging it tags the release, and the binaries are
attached to it.

The specs are in [`.documentation/specs/`](.documentation/specs/).
