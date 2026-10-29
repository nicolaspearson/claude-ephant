---
kind: plan
created: 2026-10-29
---

# v1 release

## Why

Before cutting 1.0.0, settle what a 1.0 locks in: the data directory, the CLI
surface and the privacy of the index, then bring the docs up to date and audit
the skills and the code, so the first stable release promises only what it can
keep.

## Requirements

1. The README and every `docs/` page match the 1.0 binary's behaviour
2. `docs/commands.md` matches every `ele --help`
3. Audit the skills in this repository
4. Audit the entire repository: the Go code, the plugin with its evals,
   `install.sh`, `hooks.json`, the workflows, the Taskfiles, the plugin
   manifests and `release-please-config.json`
5. List what must, should or could land before v1, the user ruling each into
   a tier; work every must have, and every should have the user keeps

## Context

- As of 2026-10-29: the latest release is v0.11.0, and release PR #12 for
  0.12.0 is open
- As of 2026-10-29: `internal/datadir`'s `Path` returns
  `~/Library/Application Support/claude-ephant` on every OS, `ELE_HOME`
  overriding it; `.documentation/specs/design.md`'s default data dir row and
  `docs/data-and-privacy.md` say the same
- As of 2026-10-29: `internal/transcript`'s `Truncate` cuts text at
  `inputLimit` (300 runes) before `internal/index`'s writer redacts it: tool
  input, an agent description, a hook block reason and a reply among its
  callers; a tool error's text is redacted before its cut
- As of 2026-10-29: `release.yml` runs on push to `main` only, and its publish
  job runs only when the release's commit is the one the run checked; no Task
  target or CI job runs actionlint
- As of 2026-10-29: `release-please-config.json` sets `bump-minor-pre-major`;
  a `Release-As: 1.0.0` commit footer, or `release-as` under `packages["."]`,
  makes the release PR cut 1.0.0
- As of 2026-10-29: the README says releases carry macOS and Linux binaries
  for amd64 and arm64, and no page under `docs/` states a compatibility promise
- As of 2026-10-29: the checkpoint log's `notes` counts are required by
  `ele checkpoint log` and read by `ele checkpoint stats`; not checked whether
  any logged run carries a non-zero count
- As of 2026-10-29: `internal/index`'s `Version` marks the index, which is
  rebuilt from the archive when its version differs
- As of 2026-10-29: `plugin/skills/checkpoint/SKILL.md` is 506 lines; the
  plugin holds 13 skills, 3 agents, 2 commands and 3 reference files
- As of 2026-10-29: the README and `docs/` were last rewritten for users in a
  plan closed 2026-10-25
- Not checked: the stop-gate false-block rates, the first-scan timing and the
  recall fallback's `--owner` failure, each taken from an earlier session's
  report; their items re-measure first

## Decisions

- Must haves gate the release; each should have is ticked or moved to
  `## Out of scope` by the user before it, which removes its edge from
  `cut-v1`; nice to haves never gate it
- Each item that changes behaviour updates the spec and `docs/` page
  describing it in the same change, as the repository's `CLAUDE.md` requires;
  `readme-guides` and `cli-docs` are a final sweep, blocked on the must haves
  that change behaviour, so the sweep runs once
- Audit findings go to `## Inbox`, dated; an audit fixes nothing itself, and
  `triage-findings` tiers each one before the release
- An audit reads whole files, not a diff: its brief points the reader at
  `plugin/skills/checkpoint/reviewers/bugs.md` and `conventions-prose.md` for
  what counts as a finding, with the area's files standing for the change
  list, and it runs one session per area
- No other plan's item blocks this one: if `mr-review`'s `review-skill` is
  unticked at release, `/ele:review`'s docs mark it experimental;
  `grader-alternation` and the eval-graders plan do not gate the release
- On Linux the data directory is `$XDG_DATA_HOME/claude-ephant`, else
  `~/.local/share/claude-ephant`; on first use, with it absent and the old path
  present, ele renames the old one once and says so. macOS is unchanged;
  `ELE_HOME` still wins
  - rejected: copying the old directory, because two stores would diverge
  - rejected: XDG on macOS too, because it moves every current user's data
- A broken 1.0.0 is fixed forward as 1.0.1, never retagged
- A manual Release trigger taking a tag recovers a stuck draft
  - rejected: a documented by-hand recovery only, because a stuck 1.0 draft
    would need tarballs and sums attached by hand
- The 1.0 promise: semver covers CLI commands and flags, the data directory and
  note format, which later versions migrate forward themselves, and the
  behaviour of the `ele` commands the hooks call; `hooks.json` and skill
  wording are not covered, since the plugin ships from `main`, so `ele` must be
  at least as new as the plugin. It lives in the README with the supported
  platforms
- The 0.12.0 release PR merges now; an interim 0.x release, cut once
  `linux-data-dir` and `cli-freeze` land, is what `clean-install` tests
- A command or flag `cli-freeze` renames keeps its old name working through
  the interim release; skills and hooks move to the new name after that
  release ships, and the old names are removed before `cut-v1`, since the
  plugin reaches users from `main` ahead of the binary
- The skill lint lands before the skills audit, so the audit's mechanical
  findings are the lint's and stay fixed
- Release workflow runs and installs from clean are the user's: the user
  triggers a dispatch, and a clean install runs in a container, VM or separate
  account, never on the PATH this machine's hooks call

## Actions

### Must have

- [ ] promise: State the 1.0 compatibility promise and supported platforms in
  the README, as the Decisions word it
  - done when the README states what semver covers and what it does not, that
    `ele` must be at least as new as the plugin, and names macOS and Linux on
    amd64 and arm64 with no Windows build; and `task format`, `task lint` and
    `task lint-docs` pass
- [ ] linux-data-dir: Move Linux's data directory to XDG, as the Decisions
  word it
  - done when `internal/datadir` tests, each seen red, cover: Linux with and
    without `XDG_DATA_HOME`; the one-time rename with its message; no rename
    when the new path exists; two first uses at once leaving one store; a
    failed rename, such as across filesystems, leaving the old path in use with
    a message; macOS unchanged and `ELE_HOME` winning. A Linux run in CI or a
    container moves a seeded old directory and `ele status` names the new one.
    `design.md`'s data dir row and `docs/data-and-privacy.md` say so, the
    latter with a line on downgrading past the move
- [ ] cli-freeze: Read every `ele <command> --help` for names, flags and
  output that 1.0 would lock in; settle the checkpoint log's `notes` field;
  check each file ele persists carries a version a later binary migrates from
  - done when the commands read are listed under this item with a count, dated
    and matching `ele --help`'s; each inconsistency found is fixed or listed as
    kept with its reason; the `notes` field is dropped or kept with its reason;
    each file ele persists in its data directory is listed with its version
    marker, added where missing, or why it needs none; the spec and `docs/`
    page of each change say so; and `task check FORCE=1` passes
- [ ] redact-before-cut: Redact every text `transcript.Truncate` cuts before
  the cut, as a tool error already is
  - done when a test per caller whose text reaches the index, each seen red,
    indexes a secret straddling the 300-rune cut and finds no fragment of it;
    `internal/index`'s `Version` is bumped, so an index built before the fix
    is rebuilt on the first sync with no fragment left; `design.md` says
    redaction runs before the cut
- [ ] release-dispatch: Add a `workflow_dispatch` trigger to `release.yml`
  taking a tag, which checks that commit, builds, uploads and publishes its
  draft
  - done when actionlint, run by hand, passes; CONTRIBUTING's Releases section
    describes it; a pre-release is never marked latest and never reaches the
    Tap job; a run the user triggers on a throwaway pre-release draft
    publishes it with tarballs whose sums match its `checksums.txt`; and the
    user deletes that release and its tag, their absence checked
- [ ] skill-lint: A Go lint under `task check`: the description of a skill
  without `disable-model-invocation` contains `Use when`, relative links
  resolve, `SKILL.md` stays under 500 lines, and extra files sit under
  `references/`, `scripts/` or `assets/` or are allowed by name
  - done when each rule fails a planted violation; `checkpoint/SKILL.md` is
    trimmed under 500 lines and each extra file at a skill root is moved, with
    its pointers, or allowed by name, each skill's `reviewers/` allowed by
    name; each description the lint reads holds `Use when`, `checkpoint`'s
    among them; `task check FORCE=1` passes on the tree;
    and CONTRIBUTING names the lint
- [ ] review-status: `/ele:review`'s docs say whether it is experimental at
  release
  - done when `mr-review`'s `review-skill` is ticked, or `docs/review.md` and
    the README mark it experimental
- [ ] readme-guides: Sweep the README and every page under `docs/` against the
  code
  - blocked: promise, linux-data-dir, cli-freeze, redact-before-cut,
    review-status
  - done when each page's commands and flags are checked against `--help`,
    each behaviour it states is checked against the code or a run, and
    `task format`, `task lint` and `task lint-docs` pass
- [ ] cli-docs: Sweep `docs/commands.md` against every `ele <command> --help`
  - blocked: cli-freeze, linux-data-dir
  - done when every command and flag `ele --help` lists has its row, each
    row's description matches its `--help`, no row names one that is gone, and
    `task lint-docs` passes
- [ ] skills-audit: Audit the skills, agents, commands, reference files and
  evals under `plugin/` against their specs, the CLI and each other
  - blocked: skill-lint, cli-freeze
  - done when the files read are listed under this item with a dated count
    matching `plugin/`'s, each finding is in the Inbox, and every command or
    flag a skill names exists in `--help`
- [ ] audit-cmd: Audit `cmd/ele`
  - blocked: cli-freeze, linux-data-dir, redact-before-cut
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
- [ ] audit-index: Audit `internal/index`, `internal/transcript`,
  `internal/archive`, `internal/store` and `internal/redact`
  - blocked: redact-before-cut
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
- [ ] audit-checkpoint: Audit `internal/checkpoint`
  - blocked: cli-freeze
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
- [ ] audit-plan: Audit `internal/plan` and `internal/ticket`
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
- [ ] audit-rest: Audit the remaining packages under `internal/`
  - blocked: linux-data-dir
  - done when the files read are listed under this item with a dated count,
    each finding is in the Inbox, and the five Go audits together name every
    package under `internal/`
- [ ] audit-tooling: Audit `install.sh`, `plugin/hooks/hooks.json`, the
  workflows under `.github/workflows/`, `Taskfile.yml`, `Taskfile.tools.yml`,
  `.claude-plugin/marketplace.json`, `plugin/.claude-plugin/plugin.json` and
  `release-please-config.json`
  - blocked: release-dispatch
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
- [ ] triage-findings: Tier every audit finding in the Inbox
  - blocked: skills-audit, audit-cmd, audit-index, audit-checkpoint,
    audit-plan, audit-rest, audit-tooling
  - done when no audit finding is left in the Inbox, each moved to a named
    action under the tier the user rules or to `## Out of scope` on the
    user's ruling, and
    each new must have or should have named on `cut-v1`'s `blocked:` line
- [ ] interim-release: The user merges the release PR for a 0.x release
  carrying `linux-data-dir` and `cli-freeze`
  - blocked: linux-data-dir, cli-freeze, release-dispatch
  - done when that release is the latest with its four tarballs and
    `checksums.txt`, its tag written under this item
- [ ] rename-cutover: Move skills and hooks to the names `cli-freeze`
  renamed, then remove the old names
  - blocked: interim-release
  - done when no file under `plugin/` names an old command or flag, the
    binary rejects each old name, `docs/commands.md` names only the new ones,
    and `task check FORCE=1` passes; or `cli-freeze` renamed nothing, said
    under this item
- [ ] clean-install: Install the interim release from clean through
  `install.sh` and `brew`, on macOS and Linux
  - blocked: interim-release
  - done when, for the interim release's tag, each install's
    `ele version` prints its tag, the plugin's hooks run in a fresh session, and
    `ele status` names the platform's data directory

### Should have

- [ ] gate-false-blocks: Re-measure the stop gate's false blocks, then stop a
  done claim matching a commit offered for later or a progress count, and
  writes from outside the session blocking a turn that wrote nothing
  - done when each pattern has a test seen red, the re-measured rate is
    written under this item, and
    `.documentation/specs/checkpoint-and-ticket-audit.md` and
    `docs/checkpoint-and-ticket-audit.md` say what the gate now passes
- [ ] gate-data-dir: The checkpoint gate leaves ele's own data directory out
  of a repository's changed paths
  - done when a test, seen red, has the gate pass a turn whose only change is
    under the data directory inside the repository, and
    `.documentation/specs/checkpoint-and-ticket-audit.md` and
    `docs/checkpoint-and-ticket-audit.md` say so
- [ ] first-scan-timeout: Bound stop-check's first scan of a transcript with
  no usable state, so a large one advances within the Stop hook's timeout
  - done when the timeout is stated under this item, a scan of a transcript
    past 400 MB saves an offset within it, a test shows the offset advancing
    across stops, and `.documentation/specs/stop-check.md` and
    `docs/data-and-privacy.md` say how the scan is bounded
- [ ] block-as-error: Find whether Claude Code can deliver a Stop block reason
  without showing it as an error, and use it if so
  - done when the finding is written under this item, dated with the Claude
    Code version, and `.documentation/specs/stop-check.md` and
    `docs/notes-and-recall.md` say what the user sees
- [ ] config-dir: Honour `CLAUDE_CONFIG_DIR` in profile discovery, archiving a
  profile outside `~/.claude` and `~/.claude-*`
  - done when a test, seen red, archives a session from such a profile,
    `ele status` lists its plugin state, and `design.md` and
    `docs/data-and-privacy.md` say so
- [ ] project-owner: `--project` works on a project with entries under
  several owners, resolving the working directory's owner where it holds the
  project
  - as of 2026-10-29: `ele plan list --project claude-ephant` fails with "has
    entries under owners hyve, personal; pass --owner <name>" while plain
    `ele plan list` works; `ele recall --project` fails the same way;
    `context` and `handoff list` resolve the owner alike, read from code, not
    run. plan-audit, ticket-audit, the recall skill's Project context and
    checkpoint's recall fallback and plan listing are among the skills giving
    the `--project` form
  - done when a test per command taking `--project`, each seen red, resolves
    the working directory's owner for a project it holds and still refuses
    one it does not; a plan audit here gets past its first call; and the
    recall skill's Project context, `design.md` and `docs/commands.md` say so

### Nice to have

- [ ] show-paging: `ele show` pages through a whole session
  - done when a flag prints entries from a given one, or all, and
    `design.md` and `docs/commands.md` name it
- [ ] scratch-log: Checkpoint runs over a repository under a temp or
  scratchpad path stay out of the stats
  - done when a test, seen red, leaves such a run out of
    `ele checkpoint stats`, and `.documentation/specs/checkpoint-and-ticket-audit.md` and
    `docs/checkpoint-and-ticket-audit.md` say so
- [ ] rumdl-pin: `task lint` and `task format` refuse a rumdl version other
  than CI's pin
  - done when a mismatched version fails the precondition with the pinned one
    named, and CONTRIBUTING says so
- [ ] sdk-cli: Index `sdk-cli` sessions while `sdk-ts` stays out
  - done when a test, seen red, indexes an `sdk-cli` session and skips an
    `sdk-ts` one, and `design.md`'s SDK entrypoint line and
    `docs/data-and-privacy.md` say so
- [ ] build-version: Record the running ele version in each hook event
  - done when an indexed hook event carries the version, `ele events` prints
    it, and `design.md`'s `ele events` section and `docs/commands.md` say so
- [ ] checkpoint-evals: Eval suites for checkpoint and ticket-audit, five to
  ten fixture cases with planted defects
  - done when, run through `task eval` when the user asks, the suites pass,
    each case seen failing on a skill with its check removed

### Release

- [ ] cut-v1: The user lands a `Release-As: 1.0.0` commit and merges the
  release PR
  - blocked: promise, linux-data-dir, cli-freeze, redact-before-cut,
    release-dispatch, skill-lint, review-status, readme-guides, cli-docs,
    skills-audit, audit-cmd, audit-index, audit-checkpoint, audit-plan,
    audit-rest, audit-tooling, triage-findings, interim-release,
    rename-cutover, clean-install, gate-false-blocks, gate-data-dir,
    first-scan-timeout, block-as-error, config-dir, project-owner
  - done when v1.0.0 is the latest release with its four tarballs and
    `checksums.txt`, the Tap job pushed its formula, and v1.0.0 installs from
    clean through `install.sh` and `brew`
- [ ] Fold each Decision through `/ele:plan-audit close
  .documentation/plans/v1-release.md`, propose each out-of-scope line as a
  `decision` through `/ele:remember`, run `ele plan done
  .documentation/plans/v1-release.md`, and delete this plan

## Inbox

## Out of scope

- Self-audit of ele's own records: a feature, not a release gate (ruled
  2026-10-29, from the v1 list)
- Context compression of large tool output: undecided until `ele usage` shows
  the share (ruled 2026-10-29, from the v1 list)
- Global rollups in the SessionStart block: content undecided (ruled
  2026-10-29, from the v1 list)
- Batch plan pickup: a workflow change, not a release gate (ruled 2026-10-29,
  from the v1 list)
- A review fix step handing picked findings to checkpoint: crosses a settled
  split between review and checkpoint (ruled 2026-10-29, from the v1 list)
