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
- `test` includes the skill lint, `internal/skilllint`, over
  `plugin/skills/`: a skill the model can invoke has `Use when` in its
  description, each relative link resolves, each `SKILL.md` stays under 500
  lines, and a skill keeps nothing beside `SKILL.md` but `references/`,
  `scripts/`, `assets/` and `reviewers/`
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

A grader is deterministic wherever the trace or the reply allows: a judge's
verdict can flip between runs on the same reply, and `task eval`'s default
threshold of 1.0 fails a case on one failed grader in one run. Lowering the
threshold hides a real failure along with a flaky one.

- A fact the trace holds, a command run, a file written or a tool called, is a
  `tool_used` grader: the trace records it, so nothing needs judging
- A literal the reply must or must not hold, such as a path or a command name,
  is a `regex` grader on `last_message`: a literal needs no judge
- An `llm` grader judges one claim neither can check, and only a claim the case
  exists to test: each judge grader is another vote that can fail the case. A
  secondary claim, one a deterministic grader already makes unlikely to fail,
  is dropped, each claim of a grader reviewed first: trimming to the main claim
  unread drops checks the case was written for
- AskUserQuestion does not load in the eval sandbox (as of 2026-10-29, claude
  2.1.308), so an ask is graded on the reply, and its behaviour is also checked
  in a real session, the only place the tool is called
- A `max: 0` `tool_used` pattern must not match the command it guards where a
  quoted argument names it, such as `forge` in `grep -E "x|forge"`, or a case
  that only reads the command fails; and it must match the command however
  it is invoked, by path, through `env` or `sh -c`, or after a pipe, or a run
  slips past

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

To publish such a draft, run Release by hand with its tag. The run checks the
tag's commit, refuses a release that is not a draft, then builds, uploads and
publishes it as a push would. A pre-release is never marked latest and never
reaches the Tap job. `gh workflow run` runs the workflow as it is on `main`
unless `--ref` names another branch. To try it without cutting a version, tag
`main` with a throwaway tag, make a draft pre-release on it and run Release:

```sh
git fetch origin && git tag v0.0.0-dispatch origin/main && git push origin v0.0.0-dispatch
gh release create v0.0.0-dispatch --draft --prerelease --verify-tag --title v0.0.0-dispatch --notes "Throwaway"
gh workflow run release.yml -f tag=v0.0.0-dispatch
```

`gh workflow run` only queues the run. Once it finishes, check the release is
published as a pre-release, not latest, and that its tarballs match its sums:

```sh
gh release list --limit 3
dir=$(mktemp -d) && gh release download v0.0.0-dispatch --dir "$dir" && (cd "$dir" && shasum -a 256 -c checksums.txt)
```

Then delete both and check they are gone, the last two commands each expected
to find nothing:

```sh
gh release delete v0.0.0-dispatch --cleanup-tag --yes && git tag -d v0.0.0-dispatch
gh release view v0.0.0-dispatch
git ls-remote --tags origin v0.0.0-dispatch
```

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

### Installing from clean

The [Tap job](#homebrew-tap) checks the tag `install.sh` installs on macOS and
Linux and `brew` on macOS. A person checks the rest: `brew` on Linux, the
plugin's hooks running in a fresh session against each install, and `ele
status` naming the platform's data directory. Run it where ele has never been
installed, never on the `PATH` your own hooks call: an install there replaces
the binary every open session runs. Use a second Mac with its own Homebrew (a
second account shares the first's prefix) and Docker for Linux. Each `version`
line expects the latest release's tag.

On the Mac, run the blocks below in one terminal, since each builds on the
`PATH` the last set. First check nothing is left from an earlier install:

```sh
which -a ele                  # nothing
test ! -e ~/Library/Application\ Support/claude-ephant && echo clean
```

Install both ways:

```sh
curl -fsSL https://raw.githubusercontent.com/nicolaspearson/claude-ephant/main/install.sh | sh
~/.local/bin/ele version
~/.local/bin/ele status       # ~/Library/Application Support/claude-ephant

eval "$(/opt/homebrew/bin/brew shellenv 2>/dev/null || /usr/local/bin/brew shellenv)"
brew install nicolaspearson/tap/ele
"$(brew --prefix)/bin/ele" version
"$(brew --prefix)/bin/ele" status
```

Then a session against each. The hooks run the first `ele` on the `PATH`
`claude` starts with, so each `export` puts the install under test first:

```sh
export PATH="$HOME/.local/bin:$PATH"
command -v claude || curl -fsSL https://claude.ai/install.sh | bash
claude plugin marketplace add nicolaspearson/claude-ephant
claude plugin install ele@claude-ephant

which -a ele                  # ~/.local/bin/ele first
claude                        # /login, then the start shows ele's block; exit
ele status                    # a sync since the session started

export PATH="$(brew --prefix)/bin:$PATH"
which -a ele                  # brew's first
claude                        # the start shows ele's block; exit
ele status
```

On Linux, in Docker; on Apple Silicon this is arm64, and
`--platform linux/amd64` runs the other. `install.sh`, then a session:

```sh
docker run --rm -it ubuntu:24.04 bash
apt-get update && apt-get install -y curl ca-certificates git
curl -fsSL https://raw.githubusercontent.com/nicolaspearson/claude-ephant/main/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
ele version
ele status                    # /root/.local/share/claude-ephant
curl -fsSL https://claude.ai/install.sh | bash
claude plugin marketplace add nicolaspearson/claude-ephant
claude plugin install ele@claude-ephant
claude                        # /login, then the start shows ele's block; exit
ele status
```

`brew`, then a session:

```sh
docker run --rm -it homebrew/brew:latest bash
brew install nicolaspearson/tap/ele
ele version
ele status                    # /home/linuxbrew/.local/share/claude-ephant
curl -fsSL https://claude.ai/install.sh | bash && export PATH="$HOME/.local/bin:$PATH"
claude plugin marketplace add nicolaspearson/claude-ephant
claude plugin install ele@claude-ephant
claude                        # /login, then the start shows ele's block; exit
ele status
```
