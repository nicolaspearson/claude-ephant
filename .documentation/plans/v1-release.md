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

- [x] promise: State the 1.0 compatibility promise and supported platforms in
  the README, as the Decisions word it
  - done when the README states what semver covers and what it does not, that
    `ele` must be at least as new as the plugin, and names macOS and Linux on
    amd64 and arm64 with no Windows build; and `task format`, `task lint` and
    `task lint-docs` pass
  - as of 2026-10-29: `release.yml`'s build step targets darwin and linux on
    amd64 and arm64 only, and `install.sh` fails on any other OS or machine;
    the README's Update section already says to update the binary before the
    plugin
  - asserted in the promise commit: README's Compatibility section names macOS
    and Linux on amd64 and arm64 with no Windows build (matching
    `release.yml`'s build targets and `install.sh`), what semver covers from
    1.0.0 and that the plugin is not covered, and that `ele` must be at least
    as new as the plugin; `task format`, `task lint` and `task lint-docs` exit 0
- [x] linux-data-dir: Move Linux's data directory to XDG, as the Decisions
  word it
  - done when `internal/datadir` tests, each seen red, cover: Linux with and
    without `XDG_DATA_HOME`; the one-time rename with its message; no rename
    when the new path exists; two first uses at once leaving one store; a
    failed rename, such as across filesystems, leaving the old path in use with
    a message; macOS unchanged and `ELE_HOME` winning. A Linux run in CI or a
    container moves a seeded old directory and `ele status` names the new one.
    `design.md`'s data dir row and `docs/data-and-privacy.md` say so, the
    latter with a line on downgrading past the move
  - as of 2026-10-29: `datadir.Path` has 23 callers in `cmd/ele`, every
    command reached through `run` in `main.go`; the hooks call `sync`,
    `session-start`, `checkpoint gate`, `stop-check` and `checkpoint snapshot`;
    the archive lock is `archive/transcripts/.lock` in the data dir, with
    `archive.TryLock`
  - as of 2026-10-29: design: `Path` stays free of side effects, on Linux the
    XDG path if present, else the old one if present, else XDG; `Migrate`,
    called in `run` before each subcommand, renames under a lock on the new
    path's parent, skipping while a sync holds the old archive lock, and
    writes its outcome to `move-notice`, which `session-start` shows and
    removes, since the move can land in a hook whose stderr no one sees
  - asserted in the linux-data-dir commit: `internal/datadir` and `cmd/ele`
    tests, each seen red under its own mutation, cover Linux with, without and
    with a relative `XDG_DATA_HOME`, the one-time rename and its notice, no
    rename when the new path exists, 32 concurrent first uses leaving one
    store, a skip while a sync holds the archive lock, an EXDEV rename keeping
    the old path with a notice, macOS unchanged, `ELE_HOME` winning, and
    `session-start` showing the notice once; a `debian:stable-slim` container
    run moved a seeded old directory and `ele status` named
    `/root/.local/share/claude-ephant`; `design.md` and
    `docs/data-and-privacy.md` say so, with the downgrade line;
    `task check FORCE=1` and `task audit` pass
- [x] cli-freeze: Read every `ele <command> --help` for names, flags and
  output that 1.0 would lock in; settle the checkpoint log's `notes` field;
  check each file ele persists carries a version a later binary migrates from
  - done when the commands read are listed under this item with a count, dated
    and matching `ele --help`'s; each inconsistency found is fixed or listed as
    kept with its reason; the `notes` field is dropped or kept with its reason;
    each file ele persists in its data directory is listed with its version
    marker, added where missing, or why it needs none; the spec and `docs/`
    page of each change say so; and `task check FORCE=1` passes
  - as of 2026-10-30: read 55 commands' `--help`, the 63 synopsis lines of
    `ele --help` (`ticket ledger` counted as `record` and `rulings`):
    `backup`; `checkpoint` `base`, `check`, `close`, `density`, `freeze`,
    `gate`, `hash`, `log`, `optout`, `record`, `snapshot`, `stats`; `config`
    `get`, `list`, `set`, `unset`; `context`; `events`; `flag` `bad`, `good`,
    `list`; `forget`; `handoff` `close`, `list`, `reopen`; `mine`; `owner`
    `add`, `add-path`, `add-project`, `remove-path`, `remove-project`; `plan`
    `check`, `done`, `list`, `show`, `start`; `recall`; `reindex`; `remember`;
    `reminder` `close`, `list`, `reopen`; `session-start`; `show`; `stats`;
    `status`; `stop-check`; `sync`; `ticket` `claims`, `corpus`, `ledger
    record`, `ledger rulings`; `usage`; `version`. Fixed: a
    flag placeholder holding `]` was cut in its help row (`remember --due`),
    and one ending `...` now keeps it (`--supersedes <id>...`);
    the nine checkpoint subcommands the skill and hooks call (`base`, `check`,
    `close`, `freeze`, `gate`, `hash`, `optout`, `record`, `snapshot`) had no
    purpose and sat outside the overview, now in it, since semver covers them;
    `checkpoint stats` takes `--since <YYYY-MM-DD|Nd>` as `events`, `usage`
    and `recall` do, a bare `<days>` kept until `rename-cutover`. Kept:
    `stop-check --replay --since` takes a date or RFC 3339 time, not `Nd`,
    since a replay window needs times and the command is for development
  - as of 2026-10-30: `notes` dropped: every run logged since 2026-10-20 has
    it 0, and the skill sends 0, the Stop hook proposing notes instead; `log`
    accepts and ignores it, `checkpoint stats` stops reading it, and
    `rename-cutover` removes it from the checkpoint and review skills.
    `ele mine --report` keeps its Notes line from mined judgments, whose
    sessions before 2026-10-20 hold non-zero counts: the baseline for the Stop
    hook's notes ask
  - as of 2026-10-30: persisted files and their markers, in `design.md`'s
    Storage section: only `ele.db` has one (`internal/index`'s `Version`); the
    JSON files ignore unknown keys, so a later format adds `version` and reads
    its absence as 1.0; notes reject an unknown key, which only a downgrade
    meets; `last-sync`, `sync.log`, `move-notice` and stop-check state are
    rewritten or reset. Outside the data dir: `~/.claude/logs`'
    `checkpoints.jsonl` and `ticket-audits.jsonl`, moved by `logs-into-data-dir`;
    the hook log, shared with the user's own hooks, and the review state in
    `~/.claude/cache/checkpoint-reviewed/` and the temp dir, a cache, stay
    outside and uncovered
  - asserted in the cli-freeze commit: all 55 commands' `--help` read from a
    clone build and listed, matching `ele --help`'s 63 synopsis lines;
    `help_test`, seen red under mutation, asserts every command, the nine
    checkpoint subcommands included, prints a purpose and each flag's
    placeholder whole as its synopsis has it; `checkpoint log` accepts a record
    with no `notes` and stats reads one, each seen red; `checkpoint stats
    --since` windows by date, seen red; `design.md`'s Storage section gives
    each persisted file's version marker or why it needs none;
    `task check FORCE=1` and `task audit` pass
- [x] logs-into-data-dir: Move `checkpoints.jsonl` and `ticket-audits.jsonl`
  from `~/.claude/logs` into the data directory, so the promise, `ele backup`,
  `ELE_HOME` and the Linux move cover them
  - done when tests, each seen red, cover: a first use moving each log once,
    with a notice as the data dir move gives; no move when the new file
    exists; `CHECKPOINT_LOG` and `--ledger` still winning; and `ele backup`
    copying both. The README's promise names the hook log and the review state
    as outside it; `design.md`, `docs/data-and-privacy.md` and
    `.documentation/specs/checkpoint-and-ticket-audit.md` say where each lives;
    and `task check FORCE=1` passes
  - asserted in the logs-into-data-dir commit: a first `ele` command moves each
    of `~/.claude/logs`' `checkpoints.jsonl` and `ticket-audits.jsonl` into the
    data dir's `logs/`, once, by hard link, with a `move-notice` line per log
    that a repeated failure leaves once; an existing new file is left alone;
    `ELE_HOME` skips both moves, leaving a notice that names the old log, and
    `CHECKPOINT_LOG` skips the checkpoint log's;
    `CHECKPOINT_LOG` and `--ledger` still win; `ele backup` copies `logs/`;
    each seen red under mutation in `internal/datadir`, `internal/checkpoint`,
    `internal/ticket` and `cmd/ele` tests. A clone build run against the real
    environment moved both logs and printed both notices, then they were moved back by hand. README's promise names
    the hook log and review state as outside; `design.md`,
    `docs/data-and-privacy.md` and the audit spec say where each lives;
    `task check FORCE=1` and `task audit` pass
- [x] redact-before-cut: Redact every text `transcript.Truncate` cuts before
  the cut, as a tool error already is
  - done when a test per caller whose text reaches the index, each seen red,
    indexes a secret straddling the 300-rune cut and finds no fragment of it;
    `internal/index`'s `Version` is bumped, so an index built before the fix
    is rebuilt on the first sync with no fragment left; `design.md` says
    redaction runs before the cut
  - asserted in the redact-before-cut commit: `transcript.Truncate` redacts
    before it collapses and cuts; `internal/index`'s
    TestUpdateKeepsNoFragmentOfASecretTheCutSplits indexes a GitHub token
    straddling the 300-rune cut through each caller whose cut text reaches the
    index (a tool line by primary input and by its JSON fallback, an agent
    call's description, a Stop hook block's reason) and finds no `ghp_`
    fragment, each case red with the redaction removed and red again with the
    limit raised past the token; a hook block's reply and a tool error were
    redacted before their cut already; `Version` 13, and `cmd/ele`'s
    TestRunSyncRebuildsAnIndexThatCutBeforeRedacting leaves 3 fragments of a
    version-12 index at 12 and none at 13; `design.md` says redaction runs
    before each cut of indexed text; `task check FORCE=1` and `task audit` pass
- [x] release-dispatch: Add a `workflow_dispatch` trigger to `release.yml`
  taking a tag, which checks that commit, builds, uploads and publishes its
  draft
  - done when actionlint, run by hand, passes; CONTRIBUTING's Releases section
    describes it; a pre-release is never marked latest and never reaches the
    Tap job; a run the user triggers on a throwaway pre-release draft
    publishes it with tarballs whose sums match its `checksums.txt`; and the
    user deletes that release and its tag, their absence checked
  - asserted in the release-dispatch tick commit, work in 4d8c42f: run
    41010566211, dispatched on the user's go-ahead on a throwaway draft
    pre-release, checked out and checked the tag's commit, published it as a
    pre-release with v0.12.0 still latest and Tap skipped, its four tarballs
    matching `checksums.txt`; actionlint, run by hand by the user, passes;
    CONTRIBUTING's Releases section describes the dispatch and the throwaway
    run; on the same go-ahead the release and tag were deleted, and both
    checked absent
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
  - blocked: promise, linux-data-dir, cli-freeze, logs-into-data-dir,
    redact-before-cut, review-status
  - done when each page's commands and flags are checked against `--help`,
    each behaviour it states is checked against the code or a run, and
    `task format`, `task lint` and `task lint-docs` pass
- [ ] cli-docs: Sweep `docs/commands.md` against every `ele <command> --help`
  - done when every command and flag `ele --help` lists has its row, each
    row's description matches its `--help`, no row names one that is gone, and
    `task lint-docs` passes
- [ ] skills-audit: Audit the skills, agents, commands, reference files and
  evals under `plugin/` against their specs, the CLI and each other
  - blocked: skill-lint
  - done when the files read are listed under this item with a dated count
    matching `plugin/`'s, each finding is in the Inbox, and every command or
    flag a skill names exists in `--help`
- [ ] audit-cmd: Audit `cmd/ele`
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
  - as of 2026-10-30: 49 files, all of `cmd/ele`, each command name below
    standing for its `.go` and `_test.go`
    - read whole by a bugs and a conventions-prose reader each: all 25
      sources
    - read whole by one bugs reader: the tests of backup, config, flag,
      forget, help, main, owner, plan, recall, remember, reminder, status,
      ticket, usage and version; handoff and planset have none
    - read in part: the tests of bashwork, checkpoint, context, events, mine,
      sessionstart, stopcheck and stopreplay, and `testdata/stop-check.jsonl`,
      its head read and grepped for personal data
    - 21 items in the Inbox, one of them for audit-index, several bundling
      a list of like findings
- [ ] audit-index: Audit `internal/index`, `internal/transcript`,
  `internal/archive`, `internal/store` and `internal/redact`
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
- [ ] audit-checkpoint: Audit `internal/checkpoint`
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
- [ ] audit-plan: Audit `internal/plan` and `internal/ticket`
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
- [ ] audit-rest: Audit the remaining packages under `internal/`
  - done when the files read are listed under this item with a dated count,
    each finding is in the Inbox, and the five Go audits together name every
    package under `internal/`
- [ ] audit-tooling: Audit `install.sh`, `plugin/hooks/hooks.json`, the
  workflows under `.github/workflows/`, `Taskfile.yml`, `Taskfile.tools.yml`,
  `.claude-plugin/marketplace.json`, `plugin/.claude-plugin/plugin.json` and
  `release-please-config.json`
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
  - done when that release is the latest with its four tarballs and
    `checksums.txt`, its tag written under this item
- [ ] rename-cutover: Move skills and hooks to the names `cli-freeze`
  renamed, then remove the old names
  - blocked: interim-release
  - done when no file under `plugin/` names an old command or flag or has a
    log record carry `notes`, `plugin/commands/checkpoint-stats.md` and
    `docs/checkpoint-and-ticket-audit.md` take `--since`, `checkpoint stats`
    refuses a bare `<days>`, the
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
  - blocked: promise, linux-data-dir, cli-freeze, logs-into-data-dir,
    redact-before-cut, release-dispatch, skill-lint, review-status,
    readme-guides, cli-docs, skills-audit, audit-cmd, audit-index,
    audit-checkpoint, audit-plan, audit-rest, audit-tooling, triage-findings,
    interim-release,
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

- 2026-10-30, from cli-freeze's persisted-files survey: nothing removes a
  checkpoint run's leftovers. The session's `.optout` file under the temp
  dir's `claude-checkpoint/` is never deleted, and a run frozen but never
  recorded or closed leaves its `.started` file in
  `~/.claude/cache/checkpoint-reviewed/` with no sweep
- 2026-10-30, from cli-freeze's persisted-files survey: `archive/claude-mem/`
  in the data dir is neither written nor read since its indexing was removed,
  yet `ele backup` copies it and `ele status` counts its size
- 2026-10-30, from redact-before-cut's checkpoint: a quoted secret assignment
  whose value spans a newline, `PASSWORD="abc` then `def…"` in a Bash heredoc,
  is not matched on the raw text, since the quoted value excludes newlines;
  `transcript.Truncate`'s whitespace collapse makes it matchable, but a cut
  inside the value leaves no closing quote, so the writer's pass misses it and
  a fragment is indexed. Older than redact-before-cut. Redact again after the
  collapse and before the cut, or let the quoted value span newlines?
- 2026-10-30, from release-dispatch's checkpoint: a hand-published
  non-pre-release draft is marked latest and tapped even when a newer release
  exists, so `install.sh` and brew would move users back to it. Should
  publish refuse a tag that sorts below the current latest?
- 2026-10-30, from audit-cmd: `ele show --at` compares its argument with
  entry timestamps as strings (`cmd/ele` `show`), while `ele flag --at`
  parses RFC 3339 instants (`indexedMoment`). A flag time to the second skips
  entries stamped with milliseconds in that second, and an offset time lands
  hours off; `TestRunShowCentresOnAMatch` asserts the bare-prefix form
- 2026-10-30, from audit-cmd: `bashwork`'s read list (`eleNotWork`) leaves
  out `ele checkpoint stats`, `ele config list`, `ele stats` and `ele usage`,
  as does `stop-check.md`'s "What it reads", so `/ele:checkpoint-stats` marks
  the session mid-task and draws a notes ask
- 2026-10-30, from audit-cmd: `sync.log` is opened append-only by
  `startSync` and never cut, 189,635 bytes on the author's machine as of
  2026-10-30, and `ele status` reads it whole. Cap or rotate it?
- 2026-10-30, from audit-cmd: `last-sync` is written in place by `syncAll`
  and read without the lock by `writeSessionStart`, so a start racing a sync
  can print "sync stamp is malformed"; `writeStopState` writes the
  stop-check state in place too, where a killed write drops the stretch.
  Write both through a temp file and rename?
- 2026-10-30, from audit-cmd: `ele help <command>`, `ele --version` and
  `ele -v` exit 2 with "unknown command"; accept them, or record their
  absence as part of the frozen CLI?
- 2026-10-30, from audit-cmd: help text off the code. `backup`'s purpose and
  overview line omit `logs/`, which `backupParts` copies; `reminder list`'s
  omits everywhere reminders, which `remindersHere` lists; `docs/commands.md`'s
  `status` line omits the running sync, the options off their defaults and
  the sync log tail
- 2026-10-30, from audit-cmd: `checkpoint freeze` refuses a missing output
  directory or no paths with the synopsis alone, no reason
  (`checkpointFreeze`'s `rejectCall(flags, nil, …)`); `checkpoint gate` and
  `snapshot` read stdin with no terminal guard, so run by hand they hang,
  where `readHookInput` refuses a terminal
- 2026-10-30, from audit-cmd: `ele context --size` (`contextSize`) scans the
  whole transcript; stop-check's `lastContextTokens` reads a bounded tail,
  since a transcript can pass 100 MB, and the spec says they compute it alike
- 2026-10-30, from audit-cmd: the SessionStart block's handoff section has
  no budget, and the plan section caps only the next item's body
  (`planBodyBudget`), not its blocked list or the multi-plan summary; only
  notes shrink to `blockBudget`'s room, so long pending handoffs can push the
  block past the size the model sees whole. How long do real handoffs run?
- 2026-10-30, from audit-cmd: the bare-number form of `checkpoint stats`,
  due to go at 1.0, is what `plugin/commands/checkpoint-stats.md` passes,
  `docs/commands.md`'s `/ele:checkpoint-stats` row describes ("or the number
  given") and `docs/checkpoint-and-ticket-audit.md` shows as `[days]`;
  removing it needs all three changed with it. The
  two forms also differ: a bare `7` is now minus 7 days, `--since 7d` the UTC
  start of that day
- 2026-10-30, from audit-cmd: `design.md` says `sync.log` is "printed, never
  parsed"; `ele status` parses its header stamps to show the log since the
  last success
- 2026-10-30, from audit-cmd: dates. `--since` and stop-check `--replay`
  dates are UTC days, `remember --due` local, and `docs/commands.md` states
  neither; a note's `Date` is local while its `Time` is UTC, and
  `newestOfKind` sorts by both
- 2026-10-30, from audit-cmd: questions on the data commands, each unsettled
  by the code. `backup` fails whole on a symlink, or on a note renamed away
  mid-walk by a concurrent `forget`. `reminder close` prints "due again" for
  a nudge whose done file is unreadable, shown again at once. `forget A B`
  refuses when A supersedes B. Owner commands blame "a sync" for a lock
  `backup` holds, and a `notes.Move` failing partway leaves moved notes with
  no owner-map save. `recall --limit 0` silently means 20. `config set` is an
  unlocked read-modify-write through a fixed `config.json.tmp`. `ticket`
  accepts `--cli` with `--plan` and ignores `ledger rulings --records`
- 2026-10-30, from audit-cmd: `bashwork`'s `shellScan.expand` tracks
  assignments flat, ignoring subshell scope and `${f#…}` operators, so
  `f=src/x.go; (f=/tmp/y); echo > $f` reads as a write to /tmp. Contrived
- 2026-10-30, from audit-cmd: `checkpoint.go`'s `hookInput.StopHookActive`
  takes `true` or `"true"`; `stopcheck`'s hook struct takes a bool alone, so
  a string would fail every Stop. Does Claude Code ever send the string?
- 2026-10-30, from audit-cmd: duplicated helpers. `remember`'s `~`
  expansion repeats `ticket`'s `expandHome`; the one-line collapse
  `strings.Join(strings.Fields(redact.Redact(…)), " ")` recurs in flag,
  handoff, reminder, remember, recall, sessionstart and `transcript`; six
  note-state filters in plan, handoff and reminder share one shape;
  `checkPlan` and `showPlan` repeat the plan-set load; recall repeats
  `syncedIndex`'s sync-then-warn. Shared helpers (`repeatedFlag`,
  `unownedError`, `projectNotes`, `supersededBy`, `newestOfKind`) sit in
  their first consumer's file
- 2026-10-30, from audit-cmd: ordering. Out of alphabetical order:
  `noteMove` and `ownerChange` in owner; `copyFile` and `copyTree` in backup;
  handoff's and plan's function runs; `unownedError` in remember; `show`'s
  flag declarations; `subagentEdit`, `subagentWorks`, `logsCheckpoint`,
  `writesNote`, `takeJudgments`, `digestName`, `noteSource`; remember's and
  plan's test tables and funcs. Tests in another command's file: two in
  `config_test.go`, one each in `flag_test.go` and `recall_test.go`, four in
  `reminder_test.go`
- 2026-10-30, from audit-cmd: comments. `withNextDue`'s says it replaces
  each nudge's Due, but only done-file nudges change, in place.
  `stopSummaryType`'s claim about Claude Code is undated. recall's "a quarter
  of the best" measurement and "None: nothing precedes it." are unclear.
  plan's "ele prints plan paths from the checkout root." sits above the wrong
  line
- 2026-10-30, from audit-cmd: small code. `status` alone uses `sort.Slice`;
  plan's `slices.IndexFunc` result `at` is unused; `remember`'s
  `"ele: --%v"` reads right only while `notes.ParseEvery`'s message starts
  `every`; `handoff`'s refusal builds a format string at run time and reads
  as a run-on; a stray blank line opens a loop in `status`
- 2026-10-30, from audit-cmd: branches no test reaches: `configUnset`'s "has
  no value set", `copyTree`'s non-regular file, `indexedMoment`'s "no
  timestamped entry", `checkPlan`'s parse error, `writeResults`' "Top N
  (--limit)"
- 2026-10-30, from audit-cmd, for audit-index: `internal/index`'s `GitRoot`
  stops at a worktree's `.git` file, so a worktree's project is its folder's
  name, not the repository's

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
