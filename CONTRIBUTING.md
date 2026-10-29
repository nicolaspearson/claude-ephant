# Contributing

## Setup

Working from a clone needs Go, at the version in `go.mod`, and
[Task](https://taskfile.dev). Every task needs Go, for the Taskfile's global
variables.

```sh
task setup
```

downloads the modules and installs the pinned Go tools the gates call
(`task setup-tools` alone reinstalls them). The gates check that each tool was
built with the Go in use, since a tool built by an older Go reports nothing
under a newer one; after a Go upgrade, run `task setup-tools` again.
`shellcheck` and `rumdl` are installed by hand, with Homebrew or your package
manager; CI pins rumdl 0.3.0.

`task` with no target lists them all.

## Build and run

The plugin's hooks call `ele` from `PATH`, so the `ele` there must be at least
as new as the plugin.

- `task install` builds the clone into `~/.local/bin` (`ELE_INSTALL_DIR`
  overrides), the directory `install.sh` writes, and fails unless `PATH` finds
  that copy first. With the clone build in another directory, a release there
  shadowed it, so a feature could not be tried in sessions or evals before
  publishing. Whichever of `task install` and `task upgrade` ran last is the
  `ele` every session's hooks run; `ele version` prints a tag for a release and
  a pseudo-version for a clone build
- `task dev` runs `task install`, then starts `claude --plugin-dir ./plugin`;
  flags after `--` go to `claude`. As of 2026-10-22 that session loads the
  clone's plugin as `ele@inline` in place of the installed
  `ele@claude-ephant`, for that session only, its SessionStart, Stop and
  UserPromptSubmit hooks each running once; SessionEnd was not observed
- `task install:local` runs `task install`, then updates the installed plugin
  to the last commit pushed to `main`
- `task upgrade` puts the release back: it runs `task release:script`, then
  updates the marketplace and the plugin, with `gh` and `claude`. Sessions
  already open load the plugin after `/reload-plugins`
- `task release:brew`, `task release:go` and `task release:script` each install
  the latest release one way: Homebrew, `go install …@<tag>`, or `install.sh`.
  Each fails unless that copy is first on `PATH` and `ele version` prints the
  release's tag, and names every copy behind it
- `task run -- <args>` runs `ele` from source; `task build` builds it into
  `dist/`

## Checks

- `task check` runs `lint`, `lint-docs`, `unused`, `tidy-check` and `test`.
  `task test FORCE=1` and `task check FORCE=1` bypass the test cache
- `task audit` runs `govulncheck`. It needs the network, so `check` leaves it
  out
- `task format` formats Go and Markdown; `task precommit` formats, runs
  `check`, then rewrites with `gofmt -s` and `go fix`
- `lint` runs `gofmt`, `go vet`, `shadow`, `staticcheck`, `shellcheck` on
  `install.sh` and `rumdl`, which also fails a relative link to a missing file
  or heading. `lint-docs` fails a `.go` or `.html` line-number citation, or a
  note id, under `docs/` or `.documentation/`: cite a package and an
  identifier, and describe what was observed in place of a note id
- CI runs `task check FORCE=1` and `task audit` on every pull request, every
  push to `main` and weekly

## Evals

`task eval` runs the installed plugin's eval cases with `claude plugin eval`,
and `task eval -- ./plugin` the clone's. Each case is a directory under
`plugin/evals/` with `graders/` that score the session, and either a
`prompt.md` whose front matter sets its runs, turns and tools, or a `case.yaml`
that sets those and the prompt and can name a `scaffold.sh` that prepares the
sandbox first. `task eval` grants Bash and Write, which a case gets only when
its own tools list them. Each run is a real session on your account and costs
model calls, so no gate runs it.

## Repository layout

| Path                 | Holds                                                                                                                             |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `cmd/ele/`           | the `ele` command and its subcommands                                                                                             |
| `internal/`          | the packages behind it                                                                                                            |
| `plugin/`            | the `ele` plugin: `skills/`, `commands/`, `agents/`, `hooks/hooks.json`, `evals/`, and `reference/` for files several skills read |
| `.claude-plugin/`    | the marketplace that serves `plugin/`                                                                                             |
| `docs/`              | user guides, linked from the README                                                                                               |
| `.documentation/`    | `specs/` and `plans/`                                                                                                             |
| `.github/workflows/` | `ci.yml`, `release.yml` and `tap.yml`                                                                                             |
| `install.sh`         | the install script                                                                                                                |

## Specs and plans

The specs under [`.documentation/specs/`](.documentation/specs/) describe the
current design: behaviour, constraints and the reason for each decision. They
hold no history; git keeps it.

The plans under `.documentation/plans/` are ele plans, worked one item a
session with `/ele:plan`; [Plans](docs/plans.md) describes the format.

## Commits

Commit subjects are [Conventional Commits](https://www.conventionalcommits.org),
`<type>(<scope>): <subject>`, with types such as `feat`, `fix`, `docs`,
`build`, `ci`, `refactor` and `style`. release-please reads them for the next
version and the changelog: below 1.0, a `fix` bumps the patch, and a `feat` or
a breaking change the minor. A breaking change puts `!` after the type and
scope, right before the colon: `feat(config)!: …`.

## Releases

Every push to `main` runs the check, then release-please opens or updates a
release PR with the next version and its changelog. The check runs there first
because a release PR opened with `GITHUB_TOKEN` triggers no checks; the
repository must allow Actions to create pull requests.

Merging the release PR creates a draft release with its tag. When the
release's commit is the one the run checked, one job cross-compiles macOS and
Linux for amd64 and arm64 with `CGO_ENABLED=0`, uploads unversioned tarballs
and a `checksums.txt`, and publishes the release as latest, so
`releases/latest/download/<name>` is a stable URL. A release of any other
commit stays a draft for a person. There is no GoReleaser and no Windows build.

The plugin has no version: the marketplace serves `plugin/` from `main`, so a
plugin change reaches users at their next marketplace update after the push,
and the binary it needs only at the next release.

### Install script

`install.sh` is POSIX `sh` needing `curl`, `tar` and `sha256sum` or `shasum`,
not `gh`. It downloads the latest release, or `ELE_VERSION`'s, checks it
against `checksums.txt`, and installs into `~/.local/bin` (`ELE_INSTALL_DIR`
overrides), warning rather than failing when that is not on `PATH`.

### Homebrew tap

The formula is a binary one: each release tarball with its sum from
`checksums.txt`, and a `test do` running `ele version`, so it needs no Go and
covers Linuxbrew. The Tap workflow, `tap.yml`, renders it and pushes it to
`nicolaspearson/homebrew-tap`, then installs the release through `install.sh`
on macOS and Linux and through `brew` on macOS, failing unless `ele version`
prints the tag. Release calls it after publishing. Run it by hand with a tag,
v0.3.0 or later, to re-render the formula for that release.

It pushes with the `HOMEBREW_TAP_TOKEN` secret, a fine-grained token with write
access to that repository's contents alone. It expires 2027-10-21; after that
the Tap job fails at the tap checkout, once the release is already published.
To renew it, create a token with the same scope and run:

```sh
gh secret set HOMEBREW_TAP_TOKEN
```

If a release's Tap job already failed, run Tap by hand with that release's tag.
