# Contributing

## Setup

Working from a clone needs Go, at the version in `go.mod`, and
[Task](https://taskfile.dev). Every task needs Go, for the Taskfile's global
variables.

```sh
task setup
```

downloads the modules and installs the pinned Go tools the gates call
(`task setup-tools` alone reinstalls them). The gates check that each tool
loading Go packages was built with the Go in use, since one built by an older
Go reports nothing under a newer one; after a Go upgrade, run
`task setup-tools` again.

`task setup-rumdl` installs rumdl at the version `Taskfile.tools.yml` pins,
with pipx, as CI does; it needs pipx. Homebrew installs only the latest, so
remove a Homebrew rumdl, which would shadow pipx's on `PATH`. `task format` and
`task lint` refuse any other version, naming the pinned one, since another
rumdl's rules can fail a commit CI passes. `shellcheck` is installed by hand,
with Homebrew or your package manager; CI pins shellcheck 0.11.0 and Task
3.54.0.

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
- `task format` formats Go, with `gofmt -s`, and Markdown; `task precommit`
  rewrites with `go fix`, formats, then runs `check`, so `check` passes the
  tree left to commit
- `lint` runs `gofmt -s`, `go vet`, `shadow`, `staticcheck`, `shellcheck` on
  `install.sh`, `actionlint` on the workflows, their `run:` blocks through
  shellcheck, and `rumdl`, which also fails a relative link to a missing file
  or heading. `lint-docs` fails a `.go` or `.html` line-number citation, or a
  note id, under `docs/` or `.documentation/`: cite a package and an
  identifier, and describe what was observed in place of a note id
- `test` includes the skill lint, `internal/skilllint`, over
  `plugin/skills/`: a skill the model can invoke has `Use when` in its
  description, each relative link resolves, each `SKILL.md` stays under 500
  lines, and a skill keeps nothing beside `SKILL.md` but `references/`,
  `scripts/`, `assets/` and `reviewers/`; it also holds `plugin.json`'s
  description equal to `marketplace.json`'s entry for `ele`
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
model calls, so no gate runs it. The `recipe-audit` case needs Go on `PATH`:
its `reports-stale-flag` grader wants the audit to report `go build`'s missing
`-fastlink`, found from Go's help, and its `reports-stale-pin` grader wants the
scaffold's `go1.21.0` reported as older than the installed Go, which must be
1.22 or newer.

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
Linux for amd64 and arm64 with `CGO_ENABLED=0`, uploads unversioned tarballs,
each holding `ele`, `LICENSE` and the `LICENSE-PSF` that `internal/ticket`'s
port of difflib carries, and a `checksums.txt`, and publishes the release as
latest, so `releases/latest/download/<name>` is a stable URL. A release of any
other commit stays a draft for a person. There is no GoReleaser and no Windows
build.

To publish such a draft, run Release by hand with its tag. The run checks the
tag's commit and refuses a release that is not a draft. It refuses a draft
that is not a pre-release when its tag sorts below the latest release, since
publishing it would move `install.sh` and brew users back, and publishing it
as not latest would list an outdated build beside the current one; and when
its tag has a pre-release suffix such as `-rc.1`, which `sort -V` places above
the bare version. Then it builds, uploads and publishes it as a push would. A
pre-release is still published, since the throwaway dispatch below is one, but
never marked latest and never reaches the Tap job. `gh workflow
run` runs the workflow as it is on `main` unless `--ref` names another branch.
To try it without cutting a version, tag `main` with a throwaway tag, make a
draft pre-release on it and run Release:

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

Run this only once the refusing workflows are on `main`, or pass `--ref` with
a pushed branch that holds them: a Release without the check publishes
`v0.0.0-old` as the latest and taps it, and a Tap without it moves brew back
to an older release. To see the refusals, make the throwaway draft without
`--prerelease`, so the age check applies to it, its tag sorting below the
latest release, and run Release with it; then run Tap with a published release
that is not the latest:

```sh
git fetch origin && git tag v0.0.0-old origin/main && git push origin v0.0.0-old
gh release create v0.0.0-old --draft --verify-tag --title v0.0.0-old --notes "Throwaway"
gh workflow run release.yml -f tag=v0.0.0-old
gh workflow run tap.yml -f tag="$(gh release list --exclude-drafts --exclude-pre-releases --json tagName,isLatest --jq 'map(select(.isLatest | not))[0].tagName')"
```

Once both finish, Release's Build and publish job fails at Find the draft
with `v0.0.0-old sorts below the latest release`, the release still a draft,
and Tap's Update the formula job fails at Refuse all but the latest, before it
checks out the tap. Then delete the throwaway as above, with `v0.0.0-old`.
Should Release publish it instead, it is the latest and Tap pushes it to brew:
delete it at once, then run Tap with the release that is latest again.

The plugin has no version: the marketplace serves `plugin/` from `main`, so a
plugin change reaches users at their next marketplace update after the push,
and the binary it needs only at the next release.

### Install script

`install.sh` is POSIX `sh` needing `curl`, `tar` and `sha256sum` or `shasum`,
not `gh`. It downloads the latest release, or `ELE_VERSION`'s, its `v`
optional, over https only, checks it against `checksums.txt`, and installs into
`~/.local/bin` (`ELE_INSTALL_DIR` overrides), warning rather than failing when
that is not on `PATH`. The binary is renamed into place, so a kill mid-install
leaves the old one and at most a stray `.ele.new` beside it, and the download
is removed on an interrupt as on exit.

### Homebrew tap

The formula is a binary one: each release tarball with its sum from
`checksums.txt`, and a `test do` running `ele version`, so it needs no Go and
covers Linuxbrew. The Tap workflow, `tap.yml`, renders it and pushes it to
`nicolaspearson/homebrew-tap`, then installs the release through `install.sh`
on macOS and Linux and through `brew` on macOS, failing unless `ele version`
prints the tag. Release calls it after publishing. Run it by hand with the
latest release's tag to re-render the formula; it refuses any other tag, so a
pre-release, a draft or an older release never reaches brew users.

It pushes with the `HOMEBREW_TAP_TOKEN` secret, a fine-grained token with write
access to that repository's contents alone. It expires 2027-10-21; after that
the Tap job fails at the tap checkout, once the release is already published.
To renew it, create a token with the same scope and run:

```sh
gh secret set HOMEBREW_TAP_TOKEN
```

If a release's Tap job already failed, run Tap by hand with that release's tag,
while it is still the latest.

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
`claude` starts with, so each `export` puts the install under test first. The
start block reaches only the model, so nothing on screen shows the hooks ran:
in each session, ask Claude to quote ele's session-start context verbatim. In a
directory no owner covers, it holds the line saying so, which names
`/ele:setup`, after a line saying no sync is recorded yet in the first session on
a new data directory; after setup, a new session's holds the notes header. The
SessionStart and SessionEnd hooks each start a sync, so `ele status`'s `Last
sync` after the session is later than its start; only the quote shows
SessionStart ran:

```sh
export PATH="$HOME/.local/bin:$PATH"
command -v claude || curl -fsSL https://claude.ai/install.sh | bash
claude plugin marketplace add nicolaspearson/claude-ephant
claude plugin install ele@claude-ephant

which -a ele                  # ~/.local/bin/ele first
claude                        # /login, ask for ele's start context; exit
ele status                    # Last sync after the session started

export PATH="$(brew --prefix)/bin:$PATH"
which -a ele                  # brew's first
claude                        # ask for ele's start context; exit
ele status                    # Last sync after the session started
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
claude                        # /login, ask for ele's start context; exit
ele status                    # Last sync after the session started
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
claude                        # /login, ask for ele's start context; exit
ele status                    # Last sync after the session started
```
