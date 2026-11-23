---
kind: plan
created: 2026-11-23
---

# Homebrew core

## Why

`ele` installs through the third-party tap `nicolaspearson/tap`, which
formulae.brew.sh never lists and `brew search` finds only once tapped. In
homebrew-core it is listed there, installs with a plain `brew install ele`, and
BrewTestBot bumps it on each release. Core takes a self-submitted formula only
past a notability bar the repository does not yet meet, so this plan prepares
what it can now and holds the submission until then.

## Requirements

1. `ele` is listed on formulae.brew.sh and installs with `brew install ele`,
   bumped by BrewTestBot on each release
2. The README licence and the tap's fresh-install path are fixed now; the
   formula is written, checked and submitted only once the notability bar is
   met
3. The maintainer submits it, so it waits for the self-submission bar
4. It is submitted as `ele`, falling back to `claude-ephant` only if reviewers
   object that `ele` is too generic
5. README.md states the licence as GPL-3.0-or-later
6. `ele version` from a core install prints the release tag with no change to
   the code, and the formula's test exercises one ele command beyond `version`
7. The formula is checked by hand once before submission, and the repository
   gains no CI job or file for it
8. Once core accepts it, tap users move to core on `brew upgrade` or by
   README's steps, no doc or task names the tap, and the release still checks
   `install.sh`

## Context

- As of 2026-11-23: Homebrew's Package Acceptance Policy asks a
  self-submission for at least 90 forks, 90 watchers or 225 stars, against 30,
  30 or 75 for anyone else; a repository under 30 days old is normally not
  eligible; a new package needs a working livecheck
- As of 2026-11-23: the repository has 0 stars, 0 forks and 0 watchers, and
  was created 2026-10-15 (`gh api repos/nicolaspearson/claude-ephant`)
- As of 2026-11-23: formulae.brew.sh's API returns 404 for both `ele` and
  `claude-ephant`, and core's `go` is 1.27.4, above `go.mod`'s `go 1.26.8`
- As of 2026-11-23: cmd/ele `buildVersion` prints the `main.version` the
  release build stamps, else a module version that is not `(devel)`, else
  `devel`; a tarball build with no `-X` prints `devel`, read from the code,
  not built
- As of 2026-11-23: the build is pure Go (`modernc.org/sqlite`), with no cgo,
  `go:embed`, `go:generate` or build tags, and no self-update in the binary;
  `syscall.Flock` keeps it to macOS and Linux, which `release.yml` builds
- As of 2026-11-23: `GPL-3.0-or-later` is stated only in
  `plugin/.claude-plugin/plugin.json`; `LICENSE` is the GPL v3 text and no
  source file carries a header
- As of 2026-11-23: the tap is named in README.md's Install, CONTRIBUTING.md's
  Homebrew tap, Installing from clean and Releases sections, Taskfile.yml's
  `TAP_FORMULA` and `release:brew`, `tap.yml`, and `release.yml`'s `tap` job;
  the check installing through `install.sh` lives in `tap.yml`'s `verify` job
- Livecheck finding the version from the repository's GitHub release tags:
  not checked
- `tap_migrations.json` moving an installed tap formula to `homebrew/core` on
  `brew upgrade`, and whether Homebrew's refusal to load an untrusted tap
  (seen 2026-11-23 from `brew info ele`) stops the migration: not checked
- As of 2026-11-23: on this Mac, with the tap tapped but untrusted,
  `brew info ele` exits 1 with "Refusing to load formula nicolaspearson/tap/ele
  from untrusted tap nicolaspearson/tap. Run
  `brew trust --formula nicolaspearson/tap/ele` or
  `brew trust nicolaspearson/tap` to trust it."; `~/.homebrew/trust.json`
  there does not list the formula
- As of 2026-11-23 (Homebrew 7.0.15): `brew install`, `reinstall` and
  `upgrade` record a formula named in full as trusted
  (`Homebrew::Trust.trust_fully_qualified_items!`); a bare `brew upgrade ele`
  or `brew info ele` on an install whose formula was never named in full is
  refused, the upgrade refusal read from the code, not run; README.md's Update
  says `brew upgrade nicolaspearson/tap/ele` for that reason, and that upgrade
  on such an install: not checked

## Decisions

- The maintainer submits the formula, so the submission waits for 90 forks,
  90 watchers or 225 stars
  - rejected: waiting for a third party to submit at 75 stars, because nothing
    in the plan can make that happen
- Submit as `ele`, the name the project ships, and fall back to
  `claude-ephant` only on a reviewer's objection: neither name is in core
- The formula stamps the version through ldflags as the release build does,
  so the code needs no change for a build with no VCS data
- The formula is verified once by hand at submission, in a local tap
  - rejected: a CI job building the source formula each release, because
    core's CI tests the formula on the PR and every bump after it
  - rejected: a `.rb` file in the repository, because the repository has no
    Ruby
- Once core has `ele`, the tap is retired and its users migrated through
  `tap_migrations.json`
  - rejected: keeping the tap alongside core, because a tap formula and a core
    formula of one name cannot be installed side by side, and Homebrew will not
    point core users at an upstream tap

## Actions

- [x] licence-readme: add a Licence section to README.md stating
      GPL-3.0-or-later, as `plugin/.claude-plugin/plugin.json` does
  - done when README.md's Licence section names GPL-3.0-or-later and links
    `LICENSE`, and `task format`, `task check FORCE=1` and `task audit` pass
  - asserted in the homebrew-licence-trust commit: README.md ends with a Licence
    section naming GPL-3.0-or-later and linking `LICENSE`; `task format`,
    `task check FORCE=1` and `task audit` passed over the working tree
- [x] tap-trust: on a machine where the tap was never tapped, run
      `brew install nicolaspearson/tap/ele` from a clean Homebrew, as
      CONTRIBUTING.md's Installing from clean does, and see whether Homebrew
      refuses the untrusted tap; if it does, add the `brew trust` step to
      README.md's Install, CONTRIBUTING.md's Installing from clean and
      `tap.yml`'s Homebrew step
  - done when a fresh install following README.md's Install with Homebrew,
    verbatim, ends with `ele version` printing the latest tag, the command
    and its output recorded under this item; a Tap run the maintainer
    dispatches with the latest tag passes its Homebrew step; and
    `task format`, `task check FORCE=1` and `task audit` pass
  - asserted in the homebrew-licence-trust commit: the v1.3.0 Release run's
    Tap job on a never-tapped macos-latest, standing in for a dispatched run,
    ran `brew install nicolaspearson/tap/ele`, which printed "Trusted formula
    nicolaspearson/tap/ele"; `brew test ele`, asserting `ele version` equals
    `v1.3.0`, the latest tag, and the step's own `ele version` check passed
    silently, so no `brew trust` step is added; `task format`,
    `task check FORCE=1` and `task audit` passed over the working tree
- [ ] core-submit: write the build-from-source formula for `ele` at the latest
      release's tag: `depends_on "go" => :build`, `go build` through
      `std_go_args` with `-s -w -X main.version=v#{version}` on `./cmd/ele`,
      `license "GPL-3.0-or-later"`, a `livecheck` with
      `strategy :github_latest`, since a tag is pushed while its release is
      still a draft, and a `test do` asserting `ele version` prints
      `v#{version}` and running one ele command beyond `version` with `HOME`
      set to `testpath`, asserting its output; verify it in a local tap made with
      `brew tap-new --no-git`, outside the repository, with
      `brew install --build-from-source`, `brew test`,
      `brew audit --new --strict --online` and `brew livecheck`; then the
      maintainer opens the PR to Homebrew/homebrew-core. On a reviewer's
      objection to the name, the formula is renamed `claude-ephant`, still
      installing the `ele` binary
  - blocked: repository notability below the homebrew-core self-submission
    bar; it needs 90 forks, 90 watchers or 225 stars
  - re-measured with
    `gh api repos/nicolaspearson/claude-ephant --jq '{stars: .stargazers_count, forks: .forks_count, watchers: .subscribers_count}'`;
    the blocked line is removed once one figure reaches the bar
  - done when the PR is merged and, with the tap untapped and `<name>` the
    name core accepted, `brew info <name>` shows the formula from
    homebrew/core, `brew install <name>` succeeds and `ele version` prints
    the latest tag, `brew livecheck <name>` printed the release marked
    Latest at submission, and `https://formulae.brew.sh/api/formula/<name>.json`
    returns 200
- [ ] tap-retire: retire the tap now that core has the formula, under the
      name core accepted: first settle the Context line on `tap_migrations.json`
      and untrusted taps with a throwaway local tap, and where migration does
      not reach an untrusted tap, add the uninstall, untap and install steps
      to README.md's Update and the release's CHANGELOG entry; then in
      `nicolaspearson/homebrew-tap`, one commit adding
      `tap_migrations.json` moving it to `homebrew/core` and deleting
      `Formula/ele.rb`; move `tap.yml`'s `install.sh` check into
      `release.yml`, then delete `tap.yml`, `release.yml`'s `tap` job and the
      `HOMEBREW_TAP_TOKEN` secret; make README.md's Install
      `brew install <name>`, Taskfile.yml's `TAP_FORMULA` the core name, with
      `release:brew` run once the BrewTestBot bump has merged rather than at
      release; and rewrite CONTRIBUTING.md's Releases, Repository layout,
      Homebrew tap and Installing from clean sections for core and its
      autobump, Releases' throwaway-draft check no longer running `tap.yml`.
      The tap-repo commit and `gh secret delete HOMEBREW_TAP_TOKEN` are
      printed for the maintainer to run
  - blocked: core-submit; the tap retires only once core holds the formula
  - done when, on a Mac with the tap installed, `brew update` then
    `brew upgrade` moves the formula to `homebrew/core`, or where migration
    does not reach an untrusted tap, README.md's Update steps followed
    verbatim leave it installed from `homebrew/core`, and either way
    `ele version` prints the latest tag; `Formula/ele.rb` is gone from the tap; a Release
    run's `install.sh` check passes from `release.yml`; the first release
    after the merge gets a BrewTestBot bump PR in homebrew-core; no file
    outside CHANGELOG.md and this plan names `nicolaspearson/tap`,
    `homebrew-tap` or `tap.yml`; and `task format`, `task check FORCE=1` and
    `task audit` pass
- [ ] Fold each Decision through
      `/ele:plan-audit close .documentation/plans/homebrew-core.md`, then
      propose each out-of-scope line as a `decision` through `/ele:remember`,
      run `ele plan done .documentation/plans/homebrew-core.md`, and delete
      this plan

## Inbox

- 2026-11-23, from tap-trust's check of Homebrew's trust code: on a Mac whose
  `ele` was installed from the tap but never trusted, run README.md's Update
  `brew upgrade nicolaspearson/tap/ele` and see whether it prints "Trusted
  formula nicolaspearson/tap/ele" and then upgrades, and `brew info ele` then
  loads
  - as of 2026-11-23: on the maintainer's Mac, tapped with `ele` from
    `install.sh` and never brew-installed, the upgrade printed "Trusted formula
    nicolaspearson/tap/ele", then "Error: nicolaspearson/tap/ele not
    installed", and `brew info ele` then loaded; the upgrade of an installed
    untrusted formula needs a Mac with `ele` installed from the tap

## Out of scope

- Promoting the repository to gain stars: the plan only waits for the bar
  (ruled 2026-11-23, from the plan's first round)
