# Contributing

## Working from a clone

Working from a clone needs [Task](https://taskfile.dev). `task install` builds
`ele` from the clone into `~/.local/bin` (`ELE_INSTALL_DIR` overrides), the
directory the install script and `task upgrade` use, so a clone build replaces
the release for every session's hooks until `task upgrade` puts the release
back; `ele version` prints a tag for a release and a pseudo-version for a clone
build. `task dev` runs `task install`, then starts Claude Code with
`--plugin-dir ./plugin`, which as of 2026-10-22 loads the clone's plugin in
place of the installed `ele@claude-ephant` for that session only; flags after
`--` go to `claude`. `task install:local` runs `task install`, then updates
the installed plugin to the last commit pushed to `main`. To test a release
install route, `task release:brew`, `task release:go` and
`task release:script` each install the latest release that way, then fail
unless that copy is the `ele` on `PATH` and prints the release's tag, listing
every copy on `PATH`.
`task check` runs every gate but `audit`; its lint needs `shellcheck` and
`rumdl`. CI runs `task check` and `task audit` on every pull request, and the
Release workflow runs them on every push to `main`.
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
it runs `task release:script`, then updates the marketplace and the plugin;
sessions already open load it after `/reload-plugins`.

`task eval` runs the installed plugin's eval cases with `claude plugin eval`,
and `task eval -- ./plugin` the clone's cases, in `plugin/evals/`.
Each run is a real session on your account and costs model calls, so no gate
runs it.

The specs are in [`.documentation/specs/`](.documentation/specs/).

## Build and install

- `task dev` installs the clone's `ele` and starts `claude --plugin-dir ./plugin`; as of 2026-10-22 that session loads the clone's plugin as `ele@inline` in place of the installed `ele@claude-ephant`, its SessionStart, Stop and UserPromptSubmit hooks each running once; SessionEnd not observed
- `ele` on `PATH` must be at least as new as the plugin: `task install` before enabling the plugin
- `task install` builds the clone into `${ELE_INSTALL_DIR:-~/.local/bin}`, the slot `install.sh` writes: hooks call `ele` from `PATH`, and with the clone build in another directory a release there shadowed it, so a feature could not be tried in sessions or evals before publishing. Whichever of `task install` and `task upgrade` ran last is the `ele` every session runs. `task check` runs every gate but `audit`, which needs the network; `task precommit` formats first
- `task release:brew`, `task release:go` and `task release:script` install the latest release through Homebrew, `go install …@<tag>` or `install.sh`, which verifies its download against `checksums.txt`, each failing unless that copy is first on `PATH` and `ele version` prints the release's tag, and naming every copy behind it. `task upgrade` runs `release:script`, then updates the marketplace and plugin through `claude plugin`. Like every task it needs Go, for the Taskfile's global variables

## Releases

- Every push to main runs CI's check, then release-please opens or updates a release PR from the conventional commits. Its merge creates a draft release with its tag, and when that release's commit is the one the run checked, one job cross-compiles macOS and Linux for amd64 and arm64 with `CGO_ENABLED=0`, uploads unversioned tarballs and a `checksums.txt`, and publishes the release as latest, so `releases/latest/download/<name>` is a stable URL. The check runs first because a release PR opened with `GITHUB_TOKEN` triggers no checks; the repository must allow Actions to create pull requests. No GoReleaser, so no tool to pin, and no Windows build
- Install: the README installs with `brew install nicolaspearson/tap/ele` first, `install.sh` second and `go install …@latest` third. The formula is a binary one, each release tarball with its sum from `checksums.txt` and a `test do` running `ele version`, so it needs no Go and covers Linuxbrew. `.github/workflows/tap.yml` renders it and pushes it to `nicolaspearson/homebrew-tap`, called by Release after publishing or by hand with a tag, then installs that tag through `install.sh` on macOS and Linux and through `brew` on macOS, failing unless `ele version` prints the tag. It pushes with `HOMEBREW_TAP_TOKEN`, a fine-grained token on the tap's contents alone, valid as of 2026-10-21 for one year, to about 2027-10-21; after that the Tap job fails at the tap checkout, once the release is already published. `install.sh` is POSIX `sh` needing `curl`, `tar` and `sha256sum` or `shasum`, not `gh`: the latest tag or `ELE_VERSION`'s, checked against `checksums.txt`, into `${ELE_INSTALL_DIR:-~/.local/bin}`, warning rather than failing when that is not on `PATH`; `task lint` runs `shellcheck` on it
