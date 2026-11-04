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
  `readme-guides` is the final sweep, `docs/commands.md` among its pages,
  blocked on the must haves that change behaviour and the should haves that
  edit `docs/`, so the sweep runs once; `cli-docs` ran before triage, and a
  later CLI change updates `docs/commands.md` in its own change (ruled
  2026-10-31)
- Audit findings go to `## Inbox`, dated; an audit fixes nothing itself, and
  `triage-findings` tiers each one before the release
- An audit reads whole files, not a diff: its brief points the reader at
  `plugin/skills/checkpoint/reviewers/bugs.md` and `conventions-prose.md` for
  what counts as a finding, with the area's files standing for the change
  list, and it runs one session per area
- No other plan's item blocks this one: if `mr-review`'s `review-skill` is
  unticked at release, `/ele:review`'s docs mark it experimental
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
- `release-safety`: publish keeps releasing a pre-release draft as a
  pre-release, never latest and never tapped, since neither `install.sh` nor
  brew serves one; only `tap.yml` refuses a pre-release (ruled 2026-11-03)
  - rejected: publish refusing a pre-release, because the throwaway dispatch
    in CONTRIBUTING is a pre-release and neither `install.sh` nor brew
    serves one
- `release-safety`: publish fails before uploading anything when a dispatched
  draft that is not a pre-release sorts below the current latest, naming both
  tags, and when its tag has a pre-release suffix such as `-rc.1`, which
  `sort -V` puts above the bare version; the draft stays a draft (ruled
  2026-11-03)
  - rejected: publishing it as not latest, because a draft below the latest
    is one a later release already superseded, and published, it would list
    an outdated build beside the current one
- `release-safety`: `tap.yml` renders only the current latest release, which
  refuses a pre-release, a draft and an older tag in one check (ruled
  2026-11-03)
  - rejected: separate pre-release and age checks, because the latest release
    is the one tag the formula may name
- `release-safety`: the README gives `go install …/cmd/ele@<tag>` with the
  tag of the release marked Latest on the Releases page, which never shows a
  draft (ruled 2026-11-03)
  - rejected: dropping `go install`, because it is the route needing no
    script or Homebrew
  - rejected: a version release-please bumps in the README, because the
    release PR's merge names the tag while it is still a draft
- `cli-lock`: `ele --version` and `ele -v` alias `ele version`, and
  `ele help <command>` runs `<command> --help`, added before 1.0, since
  `--version` is the first thing typed after an install (ruled 2026-11-02)
  - rejected: adding them in a 1.x minor, because though semver allows it,
    adding them now costs nothing
- `cli-lock`: a date the user types, `YYYY-MM-DD` or `Nd`, is a local calendar
  day, turned into UTC bounds before any comparison; stored stamps stay UTC,
  and only a time written with `Z` is UTC input; a note keeps its local `Date`
  and UTC `Time`, and `newestOfKind` sorts by `Time` alone; `docs/commands.md`
  says so, since `remember --due` and a date-only `after:` are local already
  and the note format is under the 1.0 promise (ruled 2026-11-02)
  - rejected: UTC everywhere, documented, because a `--since` west or east of
    UTC then takes in or drops hours of the day typed
- `cli-lock`: `dto.DecodeAttributes` decodes with `UseNumber`, keeping an
  integer past 2^53, and names the event in its error, since `ele events` is
  an API at 1.0 and an integer past 2^53 would print rounded (ruled
  2026-11-02)
- `cli-lock`: a malformed `after:` or `created`, or an unknown front-matter
  key, is an `ele plan check` warning naming its line, never an unparseable
  plan: a bad `after:` holds its item, a bad `created` and an unknown key are
  ignored; the docs give the UTC `HH:MMZ` form of `after:`, and the spec's
  ledger key is `owner/repo#number`, as the code writes (ruled 2026-11-02)
  - rejected: failing the whole plan, because 1.0 would lock in one typo
    hiding every item
- `first-scan-timeout`: a stop's transcript scan stops at a line boundary
  after 10 seconds, a third of stop-check's 30-second hook timeout, and saves
  its offset, since a killed hook saves nothing (ruled 2026-11-03)
  - rejected: a 64 MB byte budget, because it ignores machine speed, slow cold
    and needlessly slow warm
  - rejected: both bounds, because it has more to explain and test for no case
    either misses
- `first-scan-timeout`: while the scan is behind, the notes block is held
  pending, the size hint still firing, since a proposal not yet read may have
  cleared the work (ruled 2026-11-03)
  - rejected: judging as usual, because it can ask for notes already proposed
  - rejected: suppressing the size hint too, because its count comes from the
    tail and a long session is when it matters
- `first-scan-timeout`: while the scan is behind, subagent transcripts are not
  read; the stop that catches up reads each from its saved offset, from the
  start where it had none, outside the 10-second budget, so edits made during
  the catch-up are asked about (ruled 2026-11-03)
  - rejected: recording them unread on catch-up, because a subagent's edits
    during a catch-up of about 11 stops would never be asked about
  - rejected: reading them while behind, because their edits would be placed
    against a partial parent state
  - accepted risk: the catch-up read is unbounded, so a stop it pushes past
    the 30-second timeout saves nothing and the next repeats it warm; as of
    2026-11-03 the largest local subagent directory, 185 MB in 229 files,
    read from the start in 2.6 s, likely warm
- `block-as-error`: both Stop blocks, the notes ask and the checkpoint gate,
  continue the turn through `hookSpecificOutput.additionalContext` instead of
  `decision: "block"`, since each is the hook working as designed, not an
  error (ruled 2026-11-03)
  - rejected: the notes ask only, because the gate's block is labelled an
    error just the same
- `block-as-error`: both reasons are worded as facts with the skill to
  follow, keeping their rules as statements of how the skill works, since
  the hooks reference warns that imperative system-command text can trip
  prompt-injection defences (ruled 2026-11-03)
  - rejected: keeping the imperative wording, because Claude may then surface
    the text as a suspected injection instead of acting on it
- `block-as-error`: the README states Claude Code 2.1.163 or later, the
  first to read `additionalContext` from a Stop hook; below it both asks are
  expected not to fire, not observed (ruled 2026-11-03)
  - rejected: sending `decision: "block"` beside it for older builds,
    because a newer build may still label the turn an error, unchecked, which
    would defeat the change

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
- [x] skill-lint: A Go lint under `task check`: the description of a skill
  without `disable-model-invocation` contains `Use when`, relative links
  resolve, `SKILL.md` stays under 500 lines, and extra files sit under
  `references/`, `scripts/` or `assets/` or are allowed by name
  - done when each rule fails a planted violation; `checkpoint/SKILL.md` is
    trimmed under 500 lines and each extra file at a skill root is moved, with
    its pointers, or allowed by name, each skill's `reviewers/` allowed by
    name; each description the lint reads holds `Use when`, `checkpoint`'s
    among them; `task check FORCE=1` passes on the tree;
    and CONTRIBUTING names the lint
  - asserted in the skill-lint commit: `internal/skilllint`'s `TestCheck`
    plants a violation per rule, each failing, and a mutation of each rule's
    check turned its case red; `TestPluginSkills` passes over
    `plugin/skills/` under `task check FORCE=1`, which CI runs, with
    `checkpoint/SKILL.md` at 484 lines, the ten root files moved into
    `references/` with their pointers, `reviewers/` allowed by name, and
    checkpoint's, plan-audit's and ticket-audit's descriptions given
    `Use when`; besides Markdown links, the link rule checks code spans naming
    a `.md` file under a skill's own `references/`, `scripts/`, `assets/` or
    `reviewers/`, the form the skills' pointers take
- [x] review-status: `/ele:review`'s docs say whether it is experimental at
  release
  - done when `mr-review`'s `review-skill` is ticked, or `docs/review.md` and
    the README mark it experimental
  - asserted in the commit carrying this tick: `mr-review`'s `review-skill`
    is unticked, and `docs/review.md`'s opening and the README's line linking
    it each say `/ele:review` is experimental
- [ ] readme-guides: Sweep the README and every page under `docs/` against the
  code
  - blocked: help-text, data-dir-edges, bashwork-reads, plan-parser-warnings, ticket-robust
  - done when each page's commands and flags are checked against `--help`,
    each behaviour it states is checked against the code or a run, and
    `task format`, `task lint` and `task lint-docs` pass
- [x] cli-docs: Sweep `docs/commands.md` against every `ele <command> --help`
  - done when every command and flag `ele --help` lists has its row, each
    row's description matches its `--help`, no row names one that is gone, and
    `task lint-docs` passes
  - asserted in the commit carrying this tick: against the help of a clone
    build, all 63 usage lines of `ele --help` appear verbatim in the page's
    synopsis blocks; each command is described, and each flag it takes, the
    intro covering `--owner` and `--project`; no command or flag the page names
    is missing from the help; the descriptions that differed in meaning,
    `version`, `backup`, `status`, `plan check`, `plan done`, `reminder list`
    and `reopen`, `handoff reopen`, `stop-check --replay` and the owner
    commands among them, now follow `--help`; `task lint-docs` passes
- [x] skills-audit: Audit the skills, agents, commands, reference files and
  evals under `plugin/` against their specs, the CLI and each other
  - done when the files read are listed under this item with a dated count
    matching `plugin/`'s, each finding is in the Inbox, and every command or
    flag a skill names exists in `--help`
  - asserted in the commit carrying this tick: the three groups below are
    `find plugin -type f`, 143 files, split with nothing left over; each
    reader reports every file of its group read whole; each finding of the
    five reports, counted against the Inbox, is in an item dated 2026-10-30
    "from skills-audit"; against a clone build's help, each of the 159 `ele`
    mentions under `plugin/` outside `plugin/evals/` names a command that
    exists or is prose, and each flag on the 38 that pass flags is one that
    command's usage lists
  - as of 2026-10-30: 143 files, 4541 lines, all read whole; the list
    below matches `find plugin -type f`. Group A, 14 files, by a bugs and a
    conventions-prose reader: under `plugin/skills/checkpoint/`, `SKILL.md`,
    `references/` `BRANCH.md`, `DELTA.md`, `HEAD-EXPORT.md`, `LOG.md` and
    `PLAN.md`, `reviewers/` `bugs.md`, `conventions-prose.md` and `spec.md`;
    under `plugin/skills/review/`, `SKILL.md` and `references/` `GITHUB.md`
    and `GITLAB.md`; `plugin/agents/` `checkpoint-reviewer.md` and
    `review-reader.md`. Group B, 24 files, by a bugs and a conventions-prose
    reader: the `SKILL.md` of flag, handoff, mine, plan, plan-audit, recall,
    recipe, remember, setup, ticket-audit and wrap; mine's `references/`
    `classify-checkpoint-runs.md` and `classify-review-cycles.md`, plan's
    `references/template.md` and `reviewers/plan.md`, ticket-audit's
    `references/TRACKERS.md`; `plugin/agents/plan-reviewer.md`;
    `plugin/commands/` `checkpoint-stats.md` and `ticket-claims.md`;
    `plugin/reference/` `errors.md`, `plan-practices.md` and
    `review-voice.md`; `plugin/hooks/hooks.json`;
    `plugin/.claude-plugin/plugin.json`. Group C, 105 files, by one reader
    under both reviewer files: every file of the 17 cases under
    `plugin/evals/`, flag-saves, handoff-list, plan-new-review, recall,
    recipe-asks, recipe-audit, recipe-writes, recipe-writes-dir,
    remember-auto-off, remember-auto-on, remember-many, remember-reminder,
    setup-maps, wrap-asks, wrap-deferred, wrap-recipe and wrap-recipe-off
    - 14 items in the Inbox, several bundling a list of like findings
- [x] audit-cmd: Audit `cmd/ele`
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
  - asserted in the audit-cmd tick commit: the list below matches
    `ls cmd/ele`, 25 sources, 23 tests and one testdata file; each finding of
    the four readers' reports, counted against the Inbox, is in an item dated
    2026-10-30 "from audit-cmd", four wrong claims among them fixed by a
    checkpoint, most of their claims
    spot-checked against the code by a checkpoint's readers and the
    ordering, comment and untested-branch items left on the auditors' word
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
- [x] audit-index: Audit `internal/index`, `internal/transcript`,
  `internal/archive`, `internal/store` and `internal/redact`
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
  - asserted in the commit carrying this tick: the list below matches `ls` of
    the five packages, 28 files; each finding of both readers' reports,
    counted against the Inbox, is in an item dated 2026-10-30 "from
    audit-index" or in audit-cmd's dates item; the redaction misses and the
    symlinked-root walk were run, most other claims checked against the code
    by a checkpoint's reader, which led to six fixes to the items
  - as of 2026-10-30: 28 files, 4,339 lines
    - read whole by both readers, bugs and conventions-prose: `index`'s
      events, index, notes, schema, search, settings, stats, transcripts,
      usage and writer `.go`; `transcript`'s events and transcript `.go`;
      `archive.go`, `store.go` and `redact.go`; `index`'s index, notes,
      schema, search, stats and transcripts `_test.go`; `archive_test.go`,
      `store_test.go` and `redact_test.go`
    - read whole by the bugs reader, in part by the other: `index`'s
      `events_test.go` and `usage_test.go`, `transcript`'s `events_test.go`
      and `transcript_test.go`
    - 11 items in the Inbox, several bundling a list of like findings, and
      one added to audit-cmd's dates item; each of both readers' findings,
      counted against the Inbox, is in one
- [x] audit-checkpoint: Audit `internal/checkpoint`
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
  - asserted in the commit carrying this tick: the list below matches `ls
    internal/checkpoint`, 12 files; each finding of both readers' reports,
    counted against the Inbox, is in an item dated 2026-10-30 "from
    audit-checkpoint"; the readers probed the optout globs, the glob timing
    and the space-in-path header, and one traced two 0-second durations to
    the sessions that logged them; a checkpoint's readers checked most other
    claims against the code, ten fixes to the items following
  - as of 2026-10-30: 12 files, 3,385 lines, all read whole by both readers,
    bugs and conventions-prose: density, freeze, gate, glob, log, state and
    stats `.go`, and the density, freeze, log, state and stats `_test.go`;
    `gate` and `glob` have no test file, `cmd/ele`'s gate tests and
    `state_test.go` covering them
    - 15 items in the Inbox, several bundling a list of like findings; each
      of both readers' findings, counted against the Inbox, is in one
- [x] audit-plan: Audit `internal/plan` and `internal/ticket`
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
  - asserted in the commit carrying this tick: the list below matches `ls`
    of the two packages, 11 files; each finding of both readers' reports,
    counted against the Inbox, is in an item dated 2026-10-30 "from
    audit-plan"; the bugs reader probed the parser, `Previous` and difflib
    against Python; a checkpoint's readers checked most other claims against
    the code, and nine fixes to the items followed
  - as of 2026-10-30: 11 files, 3,615 lines, all read whole by both readers,
    bugs and conventions-prose: `plan`'s git, plan and store `.go` and
    `plan_test.go`; `ticket`'s claims, corpus, difflib and ledger `.go` and
    the claims, corpus and ledger `_test.go`. `git.go` and `store.go` are
    covered by `plan_test.go`, `difflib.go` by `corpus_test.go`
    - 17 items in the Inbox, several bundling a list of like findings; each
      of both readers' findings, counted against the Inbox, is in one
- [x] audit-rest: Audit the remaining packages under `internal/`
  - done when the files read are listed under this item with a dated count,
    each finding is in the Inbox, and the five Go audits together name every
    package under `internal/`
  - asserted in the commit carrying this tick: the list below matches `ls`
    of the eight packages, 32 files; the five Go audits name each of the 16
    entries of `ls internal` once; each finding of the four readers'
    reports, counted against the Inbox, is in an item dated 2026-10-30 "from
    audit-rest"; the readers probed the long note line, the blank body, the
    `Profile` and `ID` checks, the `[` home and the digest redaction; a
    checkpoint's readers checked most other claims against the code, and six
    fixes to the items followed
  - as of 2026-10-30: 32 files, 5,728 lines, all read whole by a bugs and a
    conventions-prose reader each: `mining`'s cache, digest, report, take,
    timing, units and watermark `.go` and their `_test.go`; `config`,
    `datadir`, `highlight`, `notes`, `ownermap` and `profile`, each a `.go`
    and its `_test.go`; `dto`'s event, indexstats, miningunit, owner,
    transcript and usage `.go`, which other packages' tests exercise
    - `internal/` holds 16 packages: audit-index named 5, audit-checkpoint 1,
      audit-plan 2 and this audit the other 8
    - 14 items in the Inbox, several bundling a list of like findings; each
      of the four readers' findings, counted against the Inbox, is in one
- [x] audit-tooling: Audit `install.sh`, `plugin/hooks/hooks.json`, the
  workflows under `.github/workflows/`, `Taskfile.yml`, `Taskfile.tools.yml`,
  `.claude-plugin/marketplace.json`, `plugin/.claude-plugin/plugin.json` and
  `release-please-config.json`
  - done when the files read are listed under this item with a dated count
    and each finding is in the Inbox
  - asserted in the commit carrying this tick: the list below names the 10
    files the item lists; each finding of both readers' reports, counted
    against the Inbox, is in an item dated 2026-10-30 "from audit-tooling";
    the bugs reader resolved the action SHAs and probed the dash trap; a
    small-mode checkpoint reader checked most other claims against the
    files, and three fixes to the items followed
  - as of 2026-10-30: 10 files, 828 lines, all read whole by a bugs and a
    conventions-prose reader each: `install.sh`, `plugin/hooks/hooks.json`,
    `ci.yml`, `release.yml` and `tap.yml` under `.github/workflows/`,
    `Taskfile.yml`, `Taskfile.tools.yml`, `.claude-plugin/marketplace.json`,
    `plugin/.claude-plugin/plugin.json` and `release-please-config.json`;
    the five pinned action SHAs match their version comments, resolved with
    `gh api`; `plugin/hooks/` holds no script, and there is no `scripts/`
    - 7 items in the Inbox, several bundling a list of like findings; each
      of both readers' findings, counted against the Inbox, is in one
- [x] triage-findings: Tier every audit finding in the Inbox
  - done when no audit finding is left in the Inbox, each moved to a named
    action under the tier the user rules or to `## Out of scope` on the
    user's ruling, and
    each new must have or should have named on `cut-v1`'s `blocked:` line
  - asserted in the commit carrying this tick: on the user's ruling of
    2026-10-31, the Inbox's 103 items, counted, went 102 into 23 new actions,
    8 must, 9 should and 6 nice, their text moved word for word, and 1 to
    `## Out of scope`; the Inbox is empty; `cut-v1`'s `blocked:` line names
    the 17 new must and should actions, and `readme-guides`' the 8 musts and
    the 5 shoulds that edit `docs/`, each on one line so `ele plan` reads it
    as edges
- [x] interim-release: The user merges the release PR for a 0.x release
  carrying `linux-data-dir` and `cli-freeze`
  - done when that release is the latest with its four tarballs and
    `checksums.txt`, its tag written under this item
  - tag: v0.13.0
  - asserted on GitHub, 2026-11-02: release PR #14 merged as 145b9f3;
    `v0.13.0` is Latest, not draft or pre-release, tagged at 145b9f3, with
    the four tarballs and `checksums.txt`; the compare API puts it ahead of
    abc8047 (linux-data-dir), 5e89a28 and 9e05ba9 (cli-freeze)
- [x] rename-cutover: Move skills and hooks to the names `cli-freeze`
  renamed, then remove the old names
  - done when no file under `plugin/` names an old command or flag or has a
    log record carry `notes`, `plugin/commands/checkpoint-stats.md` and
    `docs/checkpoint-and-ticket-audit.md` take `--since`, `checkpoint stats`
    refuses a bare `<days>`, the
    binary rejects each old name, `docs/commands.md` names only the new ones
    and its `/ele:checkpoint-stats` row drops "or the number given", `log`
    drops a `notes` field before storing, `stats`' notes comment says what the
    log holds, and `task check FORCE=1` passes; or `cli-freeze` renamed
    nothing, said under this item
  - 2026-10-30, from audit-cmd, moved from `cli-lock` on the user's ruling of
    2026-11-02: the bare-number form of `checkpoint stats`, due to go at 1.0,
    is what `plugin/commands/checkpoint-stats.md` passes,
    `docs/commands.md`'s `/ele:checkpoint-stats` row describes ("or the number
    given") and `docs/checkpoint-and-ticket-audit.md` shows as `[days]`;
    removing it needs all three changed with it. The
    two forms also differ: a bare `7` is now minus 7 days, `--since 7d` the UTC
    start of that day
  - 2026-10-30, from audit-checkpoint, moved from `cli-lock` on the user's
    ruling of 2026-11-02: the skill's §12 still sends `notes`, and
    `Append` stores it, where the spec says `log` "accepts and ignores one from
    an older skill"; `stats`' comment "The log no longer records notes" is
    untrue, since `parseRun` ignores what the log holds
  - asserted in the commit carrying this tick: `cli-freeze` renamed only the
    bare `<days>` and dropped only `notes`; `ele checkpoint stats 7` and
    `--since 7d 7` exit 2 with the synopsis, and `--since 7d` windows from 7
    days back (`cmd/ele` checkpoint tests, the refusal seen red with the check
    removed); `checkpoint.Append` drops `notes` from a clone of the record,
    leaving the caller's (`TestAppendDropsNotesFromAnOlderSkill`, seen red with
    the delete removed and with the clone removed), and `stats`' comment says
    only mined runs carry notes; no log example or field
    under `plugin/` carries `notes`, `/ele:checkpoint-stats` runs `--since`
    with an argument and none without, and `docs/commands.md`, its
    `/ele:checkpoint-stats` row no longer saying "or the number given",
    `docs/checkpoint-and-ticket-audit.md` and the spec name only `--since`;
    `task check FORCE=1` and `task audit` pass. `/ele:checkpoint-stats` not
    yet run in a session
- [x] clean-install: Install the interim release from clean through
  `install.sh` and `brew`, on macOS and Linux
  - done when, for the interim release's tag, each install's
    `ele version` prints its tag, the plugin's hooks run in a fresh session, and
    `ele status` names the platform's data directory
  - as of 2026-11-03 (737eb70): v0.13.0 is the release marked Latest, neither
    draft nor pre-release; since its tag `install.sh`, `internal/datadir` and
    `plugin/hooks/hooks.json` are unchanged, so the hooks `main` ships call
    only commands v0.13.0 has; `datadir`'s `resolve` gives
    `~/Library/Application Support/claude-ephant` off Linux and
    `$XDG_DATA_HOME` or `~/.local/share/claude-ephant` on Linux, the paths
    CONTRIBUTING.md's Installing from clean expects, and `ele status` prints
    it on its `Data` line
  - as of 2026-11-03, the user's run on the second Mac: for `install.sh` and
    `brew` each, `ele version` printed v0.13.0, `ele status` named
    `~/Library/Application Support/claude-ephant`, and `which -a ele` put the
    install under test first; nothing showed the hooks ran, since the start
    block reaches only the model, where Installing from clean expected it on
    screen. Installing from clean now has the session quote ele's start
    context and checks `Last sync` after it
  - asserted in the commit carrying this tick, on the user's runs of
    CONTRIBUTING.md's Installing from clean for v0.13.0, as revised in that
    commit, the results confirmed by the user rather than pasted: on the
    second Mac through `install.sh` and `brew`, and in arm64 Docker through
    `install.sh` on `ubuntu:24.04` and `brew` on `homebrew/brew`, each `ele version` printed v0.13.0, each
    `ele status` named the platform's data directory
    (`~/Library/Application Support/claude-ephant`,
    `/root/.local/share/claude-ephant`,
    `/home/linuxbrew/.local/share/claude-ephant`), and in a fresh session
    against each the model quoted ele's start context and `Last sync` fell
    after the session's start. linux/amd64 not run
- [x] redact-promise: Redaction holds to what `docs/data-and-privacy.md`
  promises
  - done when the user rules on the two questions below, multiline quoted values
    and whether `transcript.Parse` redacts every kind; each shape the doc
    promises and each case below the rulings keep is redacted, by a test that
    fails on the old patterns; `internal/index`'s `Version` is bumped so an
    existing index rebuilds; and the doc says what `ele mine --digests` writes,
    where, and that `/ele:mine`'s cleanup leaves the digest files
  - ruled 2026-11-02: a quoted value spans newlines up to its closing quote,
    capped near 4 KiB, and an unclosed one is redacted to its line's end, in
    one pass before any cut; `transcript.Parse` redacts every kind, the
    writer's pass kept as an idempotent re-run, and `[]string` attributes are
    redacted in the same change
  - 2026-10-30, from redact-before-cut's checkpoint: a quoted secret assignment
    whose value spans a newline, `PASSWORD="abc` then `def…"` in a Bash heredoc,
    is not matched on the raw text, since the quoted value excludes newlines;
    `transcript.Truncate`'s whitespace collapse makes it matchable, but a cut
    inside the value leaves no closing quote, so the writer's pass misses it and
    a fragment is indexed. Older than redact-before-cut. Redact again after the
    collapse and before the cut, or let the quoted value span newlines?
  - 2026-10-30, from audit-index: `redact`'s assignment pattern misses what
    `docs/data-and-privacy.md` promises, "any `key=value` or `key: value` whose
    key looks secret, JSON keys included": run against the pattern as of
    2026-10-30, `curl -d "{\"password\":\"hunter2\"}"` and `X-Api-Key: s3cr3t`
    come back unchanged, since the key allows no `\"` and no hyphen. The gitlab pattern matches `glpat-` alone, where the doc says
    "GitLab tokens": `glrt-`, `gldt-`, `glptt-`, `glcbt-`, `glft-` and `gloas-`
    pass. A fix scrubs only what is indexed after it, so it needs
    `internal/index`'s `Version` bumped, as redact-before-cut did, for an
    existing index to rebuild without what the old patterns let through
  - 2026-10-30, from audit-index: event attributes holding a `[]string`, the
    `changed` list `transcript`'s `hookBlockAttributes` takes from hook text and
    `reviewers`, are stored raw: `index`'s `writeEvent` redacts string values
    alone, where `design.md` says "attribute strings are redacted as entries
    are". `transcript.Parse` redacts tool lines but returns prompt and
    conversation text raw for the writer to redact, undocumented, and
    stop-check also reads its `Entry.Text`: should `Parse` redact every kind?
  - 2026-10-30, from audit-rest: `mining`'s `Digest` quotes a checkpoint
    run's other tool calls as `json.Marshal(block.Input)` redacted, so an
    assignment's quotes are `\"` and the value passes, the escaped-quote miss
    of audit-index's redaction item by another path: `API_KEY="s3cr3tvalue123"`
    in an MCP or `WebFetch` input reaches the digest and the classifier
    subagent (probed); no test covers the branch
  - 2026-10-30, from audit-rest: `docs/data-and-privacy.md`'s Secrets says
    text is redacted before it is indexed or printed, but not that
    `ele mine --digests` writes redacted transcript excerpts, prior asks,
    agent prompts, Bash commands and edit paths among them, into the session
    scratchpad for `/ele:mine` to hand to model subagents, nor that the skill's
    cleanup leaves the digest `.txt` files there
  - asserted in the commit carrying this tick: redaction holds to
    `docs/data-and-privacy.md`: escaped JSON keys, hyphenated keys such as
    `X-Api-Key`, `:=` and `==`, a quoted value across lines within 4 KiB
    and an unclosed one to its line's end, per the ruling, a quote escaped
    inside a value, and the six other GitLab prefixes (`internal/redact`,
    14 cases failing on the old patterns); an unquoted value keeps a
    backslash (seen red stopping at it); `transcript.Parse` redacts every
    entry kind, per the ruling (seen red without it), `Entry.Command` left raw
    for stop-check; `writeEvent` redacts `[]string` attributes (seen red);
    `mining`'s digest of another tool's input redacts its escaped quotes and
    `&` (fails on the old patterns); `internal/index`'s `Version` is 14;
    the doc says what `ele mine --digests` writes, where, and that
    `/ele:mine`'s cleanup leaves the digests; `task check FORCE=1` and
    `task audit` pass
- [x] index-private: The index files are private whatever the data directory's
  mode
  - done when `ele.db`, `-wal` and `-shm` are 0600 under an `ELE_HOME` made
    0755, by a test, and `docs/data-and-privacy.md` states their mode
  - 2026-10-30, from audit-index: `store.Open` leaves `ele.db`, `-wal` and
    `-shm` at SQLite's default mode, private only because ele makes the data
    directory 0700; an `ELE_HOME` already 0755 leaves the index, conversation
    text included, world-readable. The docs promise modes for the archive alone
  - asserted in the commit carrying this tick: `store.Open` creates
    `ele.db` 0600 and sets it, `-wal` and `-shm` to 0600 on each open, so in
    a 0755 directory all three are 0600, fresh, over a 0644 database, and
    over an older connection's 0644 WAL files (`internal/store`, seen red
    without the call, with 0644, and with the loop cut to the database); a
    refused chmod is passed over, on the user's ruling of 2026-11-02, untested
    since no test user can be refused; `ele backup`'s index copy is 0600
    (seen red at 0644); `docs/data-and-privacy.md` and `design.md` state the
    modes; `task check FORCE=1` and `task audit` pass
- [x] ticket-clean: `ele ticket corpus --clean` removes only what a corpus
  writes
  - done when a test planting another `.md` in `--out` finds it kept, and the
    help says what `--clean` removes
  - 2026-10-30, from audit-plan: `ele ticket corpus --clean` removes every
    `*.md` in `--out` and `--out/raw`, not only corpus files (`ticket`'s
    `prepare`), so `--out docs --clean` deletes the docs; its help's "empty
    --out first" is untrue too, since other files stay
  - asserted in the commit carrying this tick: `ticket corpus --clean`
    removes only the files `manifest.json` in `--out` lists and the manifest;
    any other `.md` in `--out` or `--out/raw`, a manifest-less old file
    included, refuses the run with or without `--clean` and is kept, since the
    corpus hash covers every `.md` there (`internal/ticket` corpus tests, seen
    red with the refusal removed); a real run then a `--clean` re-run into one
    directory replaces it (seen red with the manifest's `file` key renamed); a
    manifest listing anything but a bare `.md` name is refused; `--clean`'s
    help, `docs/commands.md`, the ticket-audit skill and the spec say so;
    `task check FORCE=1` and `task audit` pass
- [x] remote-secrets: Tracker detection reads the remote's host and never prints
  its credentials
  - done when a remote holding `user:token@` appears in no error, by a test, and
    `detectCLI` and `hasTracker` match the remote's host, not its path, so a
    GitHub remote whose path holds "gitlab" picks `gh`, by a test
  - 2026-10-30, from audit-plan: `ticket`'s `detectCLI` matches "gitlab" then
    "github" anywhere in the lowercased origin URL, so a GitHub repository
    whose path holds "gitlab" picks `glab` and every fetch fails; `hasTracker`
    accepts any remote holding "github.com", path included. Match the host
  - 2026-10-30, from audit-plan: `ticket`'s `detectCLI` and `trackerProject`
    print the raw origin remote in errors, so a remote holding
    `user:token@` puts the token on stderr and into the transcript when it
    names neither tracker with both CLIs installed, or `remoteProject` cannot
    read a project path from it
  - asserted in the commit carrying this tick: `detectCLI` and `hasTracker`
    match the origin's host, so `git@github.com:owner/gitlab-tools.git` picks
    `gh` and a gitlab.com remote whose path holds `github.com` has no GitHub
    tracker, while `github.com-work` ssh aliases still count (`internal/ticket`
    tests, seen red matching the whole remote); errors print a remote cut after
    its last `@`, so `https://user:s3cr3t@…`, `user:s3cr3t@host:path` and a
    credentialed URL `url.Parse` rejects (`%zz`, a `/` in the token) show no
    token (seen red printing the remote whole), nor does a `://` inside the
    token or after it (seen red with the scheme check removed); `task check FORCE=1` and
    `task audit` pass
- [x] notes-robust: Notes survive what `Write` accepts and what a hand edit
  leaves
  - done when a test covers each case below: a body line over 64 KiB fails
    neither its owner nor session start; `Write` refuses what `Parse` rejects
    and an id or profile that escapes the notes folder; a note is written
    through a temp file and a rename; and `rewriteKey` and `Move` lock, so two
    rewrites of one note keep both keys and a moved note sits under one owner;
    the smaller cases, the `*.tmp` sweep, `Move`'s opening `---` check and a
    `Relocate` test, are fixed or ruled out by the user
  - 2026-10-30, from audit-rest: `notes.Parse` reads with a default
    `bufio.Scanner`, so a body line over 64 KiB, which `Write` accepts, fails
    with "token too long" (probed): not `ErrInvalid`, so `ReadOwner` aborts
    the whole owner, and session start, `ele remember`, context, stop-check,
    status and the owner commands fail for it until that file is fixed by
    hand. `ReadOwner`'s doc says it counts the files `Parse` rejects; only
    `ErrInvalid` ones are
  - 2026-10-30, from audit-rest: `notes.Write` accepts what `Parse` then
    rejects or worse. A whitespace-only body is written and read back as
    "empty body", lost (probed), since `problem` tests the untrimmed body;
    `Owner`, `Profile`, `ID` and `Supersedes` entries go unchecked, so a
    `Profile` holding a newline writes an unreadable note and an `ID` of
    `../../esc` writes outside the notes folder (both probed). `Profile` comes
    from the config dir's base name; owner and id are checked by `cmd/ele`
    today, but `Write` is the exported boundary
  - 2026-10-30, from audit-rest: `notes` writes. `Write` creates the file with
    `O_EXCL` and writes in place, so a concurrent read sees half a note and a
    write cut short leaves an invalid file holding its id; `rewriteKey` and
    `Move` are unlocked read-modify-writes, so of two rewrites of one note
    the later rename drops the other's key, and a `MarkShown` that read a note
    before an owner command's `Move` renames it back under its old owner,
    leaving it under both; a crash leaves `*.tmp`
    files nothing sweeps; `Move` never checks the opening `---`;
    `Relocate` has no test
  - asserted in the commit carrying this tick: a body line over 64 KiB parses,
    and its owner's notes read whole (`internal/notes`, seen red without the
    scanner buffer); `Write` refuses a whitespace-only body and an id, owner or
    `supersedes` entry that escapes its folder or breaks the front matter, `.`
    and a padded owner included, and a profile with a control character, each
    seen red with its check removed; `Write` links a temporary file into place,
    leaving no `.tmp` (seen red), which on the user's ruling of 2026-11-02 means
    a data directory without hard links takes no note, documented; `rewriteKey`,
    `Move` and, on that day's ruling, `Relocate` hold the lock beside their
    tree, `notes.lock` (each seen red without it, through a test that a rewrite
    waits for the lock, the two-rewrites outcome asserted by that proxy; the
    lock kept out of the tree, seen red inside it); `Move` refuses a file
    without front matter (seen red); `SweepTemporary` removes temporary note
    files under `notes/` a day old, keeping notes and newer ones, a `[` in the
    path included (seen red on its age, its name filter and a glob), and sync
    calls it, untested there; `Relocate` is tested; no test can tell link from
    a direct write without a seam; `task check FORCE=1` and `task audit` pass
- [x] sync-robust: One bad file stops no sync
  - done when a test covers each case below: a sync with an unreadable, vanished
    or short archive file, or an unreadable note, archives and indexes the rest
    and names that file; a symlinked `projects/` is archived; and `last-sync`
    and the stop-check state are written through a temp file and a rename
  - ruled 2026-11-02: a bad file is skipped and named in `sync.log`, `ele
    status` counts the skipped, and `last-sync` is still written; `sync` exits
    0, and the session-start block says "sync skipped N files; ele status
    lists them" until a sync skips none
  - 2026-10-30, from audit-cmd: `last-sync` is written in place by `syncAll`
    and read without the lock by `writeSessionStart`, so a start racing a sync
    can print "sync stamp is malformed"; `writeStopState` writes the
    stop-check state in place too, where a killed write drops the stretch.
    Write both through a temp file and rename?
  - 2026-10-30, from audit-index: `archive.Sync` walks `source.Directory`
    without resolving it, and `filepath.WalkDir` does not descend a symlinked
    root (probed), so a profile whose `projects/` is a symlink archives nothing
    while sync reports success; `index`'s `noteFiles` resolves its root for
    this reason
  - 2026-10-30, from audit-index: one bad file stops every sync. `archive`'s
    `syncFile` erroring on a transcript removed mid-walk or unreadable aborts
    the walk before `saveManifest`; `index`'s `updateFile` fails "skip N
    indexed bytes: EOF" on an archive file shorter than `indexed_bytes`, as from
    an older backup, or on a corrupt `.gz`; `UpdateNotes` returns on the first
    unreadable note, so `last-sync` is never written. Skip and report instead?
  - asserted in the commit carrying this tick: a sync with an unreadable
    transcript or folder, a corrupt or too-short archive file, an unreadable
    notes folder, or an unreadable or unparseable note skips it, names it as
    `Skipped:` in its output and in `sync-skipped`, archives and indexes the
    rest, writes `last-sync` and exits 0, per the ruling; the start block says
    "ele's sync skipped N files; ele status lists them." and `ele status` lists
    them until a sync skips none (`internal/archive`, `internal/index` and
    `cmd/ele` tests, each seen red with its skip turned back into a failure);
    a cancelled index run stops rather than skipping the rest (seen red); a
    skipped note's indexed copy is dropped (seen red); a symlinked `projects/`
    is archived (seen red walking the unresolved path); `last-sync`,
    `sync-skipped` and the stop-check state are replaced by a rename (seen red
    written in place); a transcript vanishing mid-walk takes the unreadable
    file's path, untested since it cannot be staged; `task check FORCE=1` and
    `task audit` pass
- [x] cli-lock: Rule on each CLI question 1.0 freezes
  - asserted in the cli-lock commit: `--since` on recall, events, usage and
    checkpoint stats and `--replay`'s date bounds read a typed day from local
    midnight, by tests at `parseSince` and `replayTime` in a fixed zone,
    `index`'s `Search` at UTC+4 and UTC-5 bounds and `Events` at UTC+4;
    `newestOfKind` orders by `Time` alone; a bad `after:`, a bad `created` and
    an unknown front-matter key are `ele plan check` warnings naming their
    line, a bad `after:` holding its item, by a `plan.Parse`, `Resolve` and
    `Check` test; docs and specs carry the rulings; `task check FORCE=1` and
    `task audit` pass. Not run: `ele recall --since` against a live index at a
    non-UTC clock; `checkpoint stats`' own test reads the machine's zone, so
    it fails only where the local and UTC dates differ
  - done when each question below has the user's ruling in `## Decisions`; each
    ruling and each defect below is carried into the code and into whichever of
    `--help`, `docs/` and the spec state it; `ele events`' JSON keys are pinned
    by a test asserting literal output; and `dto.DecodeAttributes` keeps
    integers past 2^53 and names its input in its error, or the user rules it
    out
  - ruled 2026-11-02: the four `cli-lock` rulings in `## Decisions`; the
    bare-number and `notes` findings moved to `rename-cutover`
  - as of 2026-11-02: part 1 built and checkpointed: `--version`, `-v` and
    `ele help <command>`; `ele events`' nine keys pinned by literal lines;
    `DecodeAttributes` keeps integers past 2^53 and names its event. next: the
    dates ruling (local typed dates, a note's `Date` and `Time`,
    `newestOfKind`), then the plan front-matter warnings and the docs
    findings (UTC `after:` form, the ledger key)
  - 2026-11-02, premise check for part 2, the code before it: `parseSince`
    returned `Nd` as a UTC date and `YYYY-MM-DD` as typed, which `index`'s
    `Search`, `Events` and `Usage` compared as strings against UTC
    `timestamp`; `checkpoint stats` and stop-check `--replay` parsed a date as
    UTC midnight; `remember` wrote `Date` from local `now` and `Time` as UTC;
    `newestOfKind` sorted by `Date`, then `Time`; `plan.Parse` returned
    `ErrInvalid` on an unknown front-matter key, a bad `created` and a bad
    `after:`; `parseAfter` read a date alone as local midnight; `docs/plans.md`
    and the design spec gave `after: <date>` only; the
    checkpoint-and-ticket-audit spec's ledger key said `repo#number`
  - 2026-10-30, from audit-cmd: `ele help <command>`, `ele --version` and
    `ele -v` exit 2 with "unknown command"; accept them, or record their
    absence as part of the frozen CLI?
  - 2026-10-30, from audit-cmd: dates. `--since` and stop-check `--replay`
    dates are UTC days, `remember --due` local, and `docs/commands.md` states
    neither; a note's `Date` is local while its `Time` is UTC, and
    `newestOfKind` sorts by both. audit-index adds: `index`'s `Search`,
    `Events` and `Usage` compare UTC stamps as strings against the date as
    typed, or `parseSince`'s UTC date for `Nd`, so at UTC+4
    `ele recall --since 2026-10-30` drops the first four local hours of the
    30th, and west of UTC it takes in hours of the 29th
  - 2026-10-30, from audit-plan: docs. `docs/plans.md` and
    `.documentation/specs/design.md` give `after: <date>` alone; the UTC
    `YYYY-MM-DD HH:MMZ` form is in the plan skill only, and a date-only
    `after:` is local midnight. Neither says a malformed `after:`, `created`
    or unknown front-matter key makes the whole plan unparseable, which 1.0
    would lock in. The spec's ledger key says `repo#number`; the code writes
    `owner/repo#number`
  - 2026-10-30, from audit-rest: `ele events` prints `dto.Event` as JSON, an
    API at 1.0, and `TestRunEventsPrintsOneJSONObjectPerLine` decodes the
    output back through the same tags, so renaming `json:"kind"` or dropping
    an `omitempty` keeps it green. `dto.DecodeAttributes` returns the bare
    `json.Unmarshal` error and decodes numbers to `float64`, losing integers
    past 2^53
- [x] release-safety: No release path ships an old or pre-release build to users
  - asserted in the release-safety tick commit, work in 4c9a912, at the
    workflows on `main`: Release run 41569001135 of the non-pre-release draft
    `v0.0.0-old` failed at Find the draft with "v0.0.0-old sorts below the
    latest release, v0.13.0; it stays a draft", leaving it a draft with no
    assets; Tap run 41569006313 of v0.12.0 failed at Refuse all but the latest
    with "v0.12.0 is not the latest release, v0.13.0" before the tap checkout;
    the README's install and Update give `@<tag>`. Not yet seen: Tap called
    from Release right after a real publish, at the next release
  - done when publish refuses a draft, not a pre-release, whose tag sorts
    below the latest, and `tap.yml` refuses any tag but the latest release's,
    each shown by a run the user dispatches, and the README's install and
    Update name no `go install …@latest` a draft's tag can serve
  - 2026-11-03, premise check: `release.yml`'s publish job publishes any
    dispatched draft, a non-pre-release with `--latest` and then tapped, and
    a pre-release with `--prerelease --latest=false` and untapped, with no
    comparison to the current latest; `tap.yml` by hand checks neither
    pre-release nor age; `README.md` gives `go install …/cmd/ele@latest` in
    install and Update; `release-please-config.json` sets `draft` and
    `force-tag-creation`
  - as of 2026-11-03: publish refuses a non-pre-release draft below the
    latest, or with a pre-release suffix, and `tap.yml` refuses all but the
    latest release, both before any upload or tap checkout; the README gives
    `@<tag>`; CONTRIBUTING documents the refusal runs; actionlint passes and
    `sort -V` checked on GNU coreutils 9.4
  - 2026-10-30, from release-dispatch's checkpoint: a hand-published
    non-pre-release draft is marked latest and tapped even when a newer release
    exists, so `install.sh` and brew would move users back to it. Should
    publish refuse a tag that sorts below the current latest?
  - 2026-10-30, from audit-tooling: the README's install and Update give
    `go install …/cmd/ele@latest`, but `release-please-config.json` sets
    `draft` and `force-tag-creation`, so a release's tag is on the remote
    while it is still a draft: `@latest` installs a release left a draft for
    a person, its merge having failed the check, and a GitHub pre-release with
    a plain tag, neither of which `install.sh` or brew serves, and the module
    proxy keeps the version for good. `task release:go` installs `@<tag>` of
    the published release
  - 2026-10-30, from audit-tooling: `tap.yml` by hand refuses neither a
    pre-release nor a tag older than the latest, so either reaches every brew
    user, where Release skips pre-releases; the hand-published-draft item
    above in this action is the same family on the Release side

### Should have

- [x] gate-false-blocks: Re-measure the stop gate's false blocks, then stop a
  done claim matching a commit offered for later or a progress count, and
  another session's writes blocking a turn, whether it wrote nothing or its
  own files are recorded
  - done when each pattern has a test seen red, the re-measured rate is
    written under this item, and
    `.documentation/specs/checkpoint-and-ticket-audit.md` and
    `docs/checkpoint-and-ticket-audit.md` say what the gate now passes
  - as of 2026-11-03 (0acae64), re-measured from `ele events --kind
    hook-block` over 2026-10-23 to 2026-11-03, every owner: 85 checkpoint
    blocks, 57 followed by a checkpoint and 28 not. 53 of the 57 and 22 of the
    28 matched `doneClaim` only on a printed `git commit`. By their replies the
    28 are about 6 turns blocked on another session's files with none of their
    own moved, about 6 whose own files a record already covered while another
    session's were dirty, 13 offering a commit mid-item, 2 progress phrases and
    1 fixture that wanted an optout. 51 of the 57 moved a path this turn's own
    `Edit` or `Write` named; the other 6 moved paths only through Bash, each
    read as work by `commandIsWork`, and one of those also wrote a temporary
    test file it then removed
  - ruled 2026-11-03: a turn that ran `Edit`, `Write` or `NotebookEdit`
    (`MultiEdit` too) on a file in the repository is judged on the moved paths
    those calls named, coverage by the latest record
    included, and `Changed:` lists only them; a turn with none is judged on
    every moved path when a Bash command reads as work, and passes otherwise.
    A commit offered mid-item stays a block, being a ready-to-commit claim;
    one offered for later, "once they pass, you can commit", passes, as does a
    done phrase whose line says what is still left
  - as of 2026-11-03, against the 28: the ruling passes about 9, the 6
    covered-record turns, one that edited files none of which moved, and two
    whose Bash `commandIsWork` reads as no work; three still block, two on
    `ele checkpoint log` and scratchpad writes reading as work and one on an
    edit of its own left unreviewed. The later-commit
    and still-left patterns clear none of the 28: the two progress phrases
    count nothing left on their line. Of the 57, one is lost: it removed a plan
    file through Bash beside a `Write` of the temporary test file, the ruling's
    cost of a Bash write beside an edit in the repository. The others each
    moved an unreviewed edited path or moved paths only through Bash
  - ruled 2026-11-03: no fallback to every moved path for a turn whose edits
    moved nothing beside Bash work; it would restore that one but re-block a
    turn whose edits moved nothing while another session's files did
  - asserted in the commit carrying this tick, at the Stop hook's
    `ele checkpoint gate` boundary in `cmd/ele` tests: a turn's edits in the
    repository decide what it is judged on, coverage by the latest record
    included, and `Changed:` names only those; with none, a Bash command
    `commandIsWork` reads as work brings every moved path in, and a turn whose
    start is past the transcript tail is judged on all; a commit offered for
    later and a progress count pass, a commit offered now and "changes left in
    the working tree" still block. Each rule's test was seen red with that
    rule removed. The re-measure is above; the spec's gate bullets and
    `docs/checkpoint-and-ticket-audit.md` say what passes; `task check
    FORCE=1` and `task audit` pass. Not yet observed in a real session
- [x] gate-data-dir: The checkpoint gate leaves ele's own data directory out
  of a repository's changed paths
  - done when a test, seen red, has the gate pass a turn whose only change is
    under the data directory inside the repository, and
    `.documentation/specs/checkpoint-and-ticket-audit.md` and
    `docs/checkpoint-and-ticket-audit.md` say so
  - as of 2026-11-03: `checkpoint.Gate` takes `MovedPaths` from
    `WriteState`'s `git diff HEAD` and untracked `ls-files`, with no data
    directory filter; `datadir.Path` is `ELE_HOME` when set, which can sit
    inside a checkout, as in the eval sandbox where it was seen; a Bash turn
    then brings every moved path in through `ownPaths`
  - asserted in the gate-data-dir commit: cmd/ele's gate hook test passes a
    Bash turn whose only change is under an `ELE_HOME` inside the repository;
    beside data-directory writes it names main.go and a shared-prefix
    `.ele-other/` path only; with 250 data-directory files past the 200-path
    cap it still blocks on main.go. Each part was seen red: no filter in
    `DirtyPaths`, a prefix match without the slash, the filter after the cap.
    The spec's gate bullets and `docs/checkpoint-and-ticket-audit.md` say so;
    `task check FORCE=1` and `task audit` pass. Not yet observed in a real
    session
- [x] first-scan-timeout: Bound stop-check's first scan of a transcript with
  no usable state, so a large one advances within the Stop hook's timeout
  - done when the timeout is stated under this item, a scan of a transcript
    past 400 MB saves an offset within it, a test shows the offset advancing
    across stops, and `.documentation/specs/stop-check.md` and
    `docs/data-and-privacy.md` say how the scan is bounded
  - as of 2026-11-03: the timeout is 30 seconds, `ele stop-check`'s
    `"timeout": 30` in `plugin/hooks/hooks.json`, and the scan's bound 10
    seconds, `scanBudget` in cmd/ele; `scanTranscript` read to EOF
    with no bound and `writeStopState` runs once, after it, so a scan the
    timeout killed saved no offset; whether Claude Code runs the two Stop
    hooks in parallel not checked, and the bound does not rest on it
  - as of 2026-11-03: a clone build with a scratch `ELE_HOME`, on a synthetic
    456 MB transcript of 2 KB assistant lines, saved offsets of 44, 86, 120
    and 166 MB (10^6 bytes) over four stops of 11.2, 10.1, 10.1 and 10.1 s,
    warm cache; about 4 MB/s, so it would catch up in about 11 stops. On
    2026-10-19 a full parse of a 153 MB transcript took 1 s warm and 9–12 s
    cold, the reason for a time bound over a byte one
  - asserted in the first-scan-timeout commit:
    `TestRunStopCheckResumesAScanCutByItsBudget` (cmd/ele) runs the hook at a
    zero budget and sees the saved offset advance one line a stop, no notes
    block and no subagent read while behind, then a block on a subagent's
    edit once caught up; seen red by passing no deadline, dropping behind
    from the block, reading subagents while behind, dropping the EOF peek,
    allowing zero lines, recording subagents unread on catch-up, and
    dropping the empty-map sentinel saved while behind. A
    clone build saved offsets within a 456 MB transcript at 10.1–11.2 s a stop
    (above). The spec's State and Replay sections and
    `docs/data-and-privacy.md` say how the scan is bounded; `task check
    FORCE=1` and `task audit` pass. Not yet observed in a real session
- [x] block-as-error: Find whether Claude Code can deliver a Stop block reason
  without showing it as an error, and use it if so
  - done when the finding is written under this item, dated with the Claude
    Code version, and `.documentation/specs/stop-check.md` and
    `docs/notes-and-recall.md` say what the user sees
  - as of 2026-11-03, Claude Code 2.1.308: a `{"decision":"block","reason":…}`
    Stop output at exit 0 reaches the model twice, as `Stop hook feedback:`
    and as `Stop hook blocking error from command: "<command>": …`; the
    transcript records a meta user line `Stop hook feedback:\n<reason>`, a
    `hook_blocking_error` attachment, and the reason in `stop_hook_summary`'s
    `hookErrors`
  - as of 2026-11-03: the hooks reference, Stop decision control, documents
    `hookSpecificOutput.additionalContext` with `hookEventName: "Stop"` as
    "non-error feedback": the turn continues under the same loop guards
    (`stop_hook_active`, the 8-continuation cap), the transcript labels it
    `Stop hook feedback` and no hook error notification is shown; added in
    2.1.163. Its "add context" section warns that text framed as an
    imperative system command can trip prompt-injection defences
  - as of 2026-11-03: the index's notes-ask events, `--replay`'s `asked` and
    mining's hook-block counts all key on the `Stop hook feedback:` user
    line (`transcript.hookFeedback`); how `additionalContext` is recorded,
    that line, a `hook_additional_context` attachment, which the parser reads
    as an injection, or `hookAdditionalContext` alone, then not checked;
    the probe below found the attachment
  - as of 2026-11-03, Claude Code 2.1.309: a headless `claude -p` probe in a
    scratch directory, its Stop hook printing `additionalContext` once, kept
    the turn going and the model acted on it; the transcript held a
    `hook_additional_context` attachment (`hookEvent` and `hookName` `Stop`),
    the text in `stop_hook_summary.hookAdditionalContext` with `hookErrors`
    empty, and no `Stop hook feedback:` line; the model got it as
    `<system-reminder>Stop hook additional context: …`. The terminal display
    of an interactive session not yet seen
  - as of 2026-11-03: built, uncommitted: both Stop hooks send
    `additionalContext` with factual reasons; the parser and mining read a
    Stop or SubagentStop `hook_additional_context` attachment as a
    `hook-block` (`transcript.StopContext`), the index rebuilds at version 15,
    and the specs, docs, README floor and `hooks.json` say so; each new test
    seen red. Checkpointed; the clone build is installed
  - as of 2026-11-04, Claude Code 2.1.308, interactive, on the build of
    462f54f: a notes ask showed in the terminal as `Ran 3 stop hooks` and,
    under it, `Stop hook feedback: <reason>`, with no error label; the
    transcript held a Stop `hook_additional_context` attachment and the
    reason in `stop_hook_summary.hookAdditionalContext`, `hookErrors` empty,
    no feedback line and no `hook_blocking_error`; the model got it once, as
    `Stop hook additional context: …`. A gate block and two Stop hooks
    continuing one stop not yet seen
  - asserted in the block-as-error commit and the 2026-11-04 interactive stop
    above: both Stop hooks send `additionalContext`; on 2.1.308 the terminal
    showed the notes ask as `Stop hook feedback` with no error label and the
    model got it once; the parser and mining read the Stop attachment as a
    `hook-block` (tests seen red by removing the Stop case, SubagentStop,
    string content and mining's call), the index rebuilds at 15, the specs
    and docs say what the user sees and the README the 2.1.163 floor;
    `task check FORCE=1` and `task audit` pass. A gate block not yet seen
- [x] config-dir: Honour `CLAUDE_CONFIG_DIR` in profile discovery, archiving a
  profile outside `~/.claude` and `~/.claude-*`
  - done when a test, seen red, archives a session from such a profile,
    `ele status` lists its plugin state, and `design.md` and
    `docs/data-and-privacy.md` say so
  - as of 2026-11-04: `profile.Discover(home)` globs `~/.claude-*` and adds
    `~/.claude`, keeping those holding `projects/`; it reads no
    `CLAUDE_CONFIG_DIR`. Its callers: sync in `cmd/ele` `main.go`,
    `contextSize`, `plugins` (status) and `transcribed` (planset).
    `CLAUDE_CONFIG_DIR` is read only through `profile.Current`, by `flag`,
    `plan` and `remember`, naming a profile by `nameOf` of its directory
  - as of 2026-11-04: the archive keys a file `<profile>/<path within
    projects/>` (`internal/archive`, `Source.Profile`), so a
    `CLAUDE_CONFIG_DIR` profile named by `nameOf` alone, such as
    `/elsewhere/.claude-hyve` beside `~/.claude-hyve`, shares one archive tree
    and manifest keys with another directory; a hyve session sets
    `CLAUDE_CONFIG_DIR=~/.claude-hyve`, already discovered, so the added root
    is deduplicated by real path
  - as of 2026-11-04: only a sync run under that `CLAUDE_CONFIG_DIR`, its own
    sessions' hooks or a shell exporting it, sees the profile; a later sync
    without it archives nothing new from it and keeps what was archived
  - as of 2026-11-04: design, ruled by the user: the added profile is named
    by `nameOf`, as `profile.Current` names it for notes; one whose name a
    discovered profile at another real path holds is skipped as sync skips a
    file, a `Skipped:` line and a `sync-skipped` entry
  - asserted in the config-dir commit: `profile.Discover` adds
    `CLAUDE_CONFIG_DIR` when it holds `projects/`, deduplicated by real path
    and skipped with a `Skipped:` line and `sync-skipped` entry when its folder
    name is taken (tests seen red by mutation: the added root, the name check,
    the `projects/` check, sync's and status's variable, and the real-path
    pass before names); a build of this tree, with a temporary `HOME`,
    `ELE_HOME` and `CLAUDE_CONFIG_DIR`, archived its one transcript under
    `transcripts/tmp.Xy12` and `ele status` showed it enabled; `design.md` and
    `docs/data-and-privacy.md` say so; `task check FORCE=1` and `task audit`
    pass
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
- [ ] skill-fixes: Fix the skill defects that fail a session or skip its review
  - done when each defect below is fixed in its skill or command file, the plan
    and checkpoint skills agree on who proposes a tick, and `task check FORCE=1`
    passes
  - ruled 2026-11-02: checkpoint proposes a tick; `/ele:plan` step 6 says to
    run `ele:checkpoint` once the work is done and itself proposes a dated
    progress line only; checkpoint's §7 gains a branch for an item already
    ticked, re-reviewing its tick line and proposing none
  - 2026-10-30, from skills-audit: checkpoint's §1 builds the change list from
    `git status --porcelain`, which shows a new untracked directory as one
    `?? dir/` line; `ele checkpoint freeze` refuses a directory ("is neither
    tracked nor a file"), and `reference/errors.md` has no row for it, so a
    change adding a package stops at §1 or loses its new files. Fix: use
    `--untracked-files=all` there and for §9's `add` list
  - 2026-10-30, from skills-audit: two skills write an item's tick under
    different gates. `/ele:plan` step 6 proposes the tick once its line "can be
    written", naming no checkpoint; checkpoint's `references/PLAN.md` ticks
    only after §6's gate and an unfixed-spec check, and has the line
    re-reviewed; `docs/plans.md` says checkpoint proposes it. A plan session
    can tick unreviewed work, and checkpoint's §7 has no branch for an item
    already ticked
  - 2026-10-30, from skills-audit: `ticket-audit` §1 and §10, `plan-audit`'s
    ticket-audit sections and `/ele:plan`'s review loop set `AUDIT=$(mktemp
    -d)` or `BASE=$(mktemp -d)` and read `$AUDIT` or `$BASE` in later
    commands, but Claude Code's Bash tool keeps no variable between calls, and
    the assignment prints no path: `ledger rulings --manifest
    "$AUDIT/manifest.json"` reads `/manifest.json` and fails, and the plan
    loop's settled file and `diff -u` lose their directory. Fix: run `mktemp
    -d` alone and write the printed path into later commands as a placeholder
  - 2026-10-30, from skills-audit: `plugin/commands/checkpoint-stats.md`'s
    Duration bullet says a re-review frozen over two hours after the last
    freeze, or a freeze with no session id, restarts the clock; in
    `internal/checkpoint` `openRun` only `--start` or a missing `.started`
    rewrites it, and with no session no clock starts. This adds the command
    file to the item from audit-checkpoint on what `duration` measures, which
    does not name it
  - 2026-10-30, from skills-audit: `/ele:plan`'s review loop step 5 runs `ele
    plan check` on each draft, but `convert` splits drafts by repository and
    the check refuses a path outside the working directory's checkout ("is
    outside the project"), so the step, which does not say to run it from the
    draft's repository, fails on a sibling repository's draft; `plan-audit`
    says to run it from the plan's checkout
  - 2026-10-30, from skills-audit: `ticket-audit`'s `references/TRACKERS.md`
    counts `OPEN` (gh) and `opened` (glab) as open, but its `gh api
    …/dependencies/blocked_by` command prints the REST API's lowercase `open`,
    so a literal match reads an open native blocker as closed and frees the
    ticket it blocks
  - 2026-10-30, from skills-audit: checkpoint's §10 done-when wants a verdict
    on every change-list file "from each reviewer that ran", while
    `reviewers/spec.md` and §4 give spec a verdict per requirement, so a run
    with a spec reviewer cannot meet it literally; `/ele:review` step 5 words
    it right
  - 2026-10-30, from skills-audit: `/ele:review` step 12 says a failed publish
    "leaves nothing half-sent behind, per the forge file", true of GitHub's one
    POST but not of `references/GITLAB.md`, which publishes drafts one by one,
    so a failure part-way leaves those already published live, against the
    step's claim; only the report's "which findings reached the forge" says so
  - 2026-10-30, from skills-audit: `plan-audit`'s ticket-audit §0 recalls
    earlier audits with `<topic>` set to every plan's file name plus `audit`,
    but recall needs every word in one entry, so an audit over several plans
    matches nothing and reports no earlier audit; `/ele:plan` pickup already
    keeps recall to two or three words. Fix: one recall per plan
- [ ] checkpoint-correctness: `ele checkpoint` reports and records what was
  reviewed
  - done when each defect below has a test that fails before its fix, or the
    user rules it out of scope
  - ruled 2026-11-02: a freeze opening a new run after the two-hour window
    restarts the clock; the skill forbids chaining the `--start` freeze with
    `log`; `stats` keeps 0-second and negative durations out of the median and
    counts them on a line of their own; the spec, `log --help` and the skill
    say "the freeze that opened the run"
  - 2026-10-30, from cli-freeze's persisted-files survey: nothing removes a
    checkpoint run's leftovers. The session's `.optout` file under the temp
    dir's `claude-checkpoint/` is never deleted, and a run frozen but never
    recorded or closed leaves its `.started` file in
    `~/.claude/cache/checkpoint-reviewed/` with no sweep
  - 2026-10-30, from audit-cmd: `checkpoint freeze` refuses a missing output
    directory or no paths with the synopsis alone, no reason
    (`checkpointFreeze`'s `rejectCall(flags, nil, …)`); `checkpoint gate` and
    `snapshot` read stdin with no terminal guard, so run by hand they hang,
    where `readHookInput` refuses a terminal
  - 2026-10-30, from audit-checkpoint: `ele checkpoint check` says "covered"
    for bytes nobody reviewed. `Record` stores the hash of every dirty path,
    frozen or not, as its own comment says, and `Reviewed` passes when every
    dirty path matches, so unrelated work dirty at record time is later
    "covered" and the skill stops on it, `checkpointCheck` printing "every
    changed path … is what the last checkpoint reviewed". `Record` also hashes
    the tree at record time without comparing the freeze's fingerprint, so an
    edit between freeze and record is recorded as reviewed; the skill guards
    that by hand
  - 2026-10-30, from audit-checkpoint: `DirtyPaths` sorts and cuts the dirty
    list at `pathCap`, 200, with no sign, so past the 200th path an edit moves
    no fingerprint, `check` stays covered, the skill's re-review is skipped and
    the Stop gate never sees it
  - 2026-10-30, from audit-checkpoint: `ValidateOptout` tests four fixed probe
    strings, so it accepts `*e*`, `*/*/*/*` and `[0-9a-f]*` (each probed),
    where its doc says it refuses a glob matching "nearly every" path and
    `optout --help` one "matching any path"; `*e*` matches nearly every path,
    exempting most of the repository from the gate for the session
  - 2026-10-30, from audit-checkpoint: what the checkpoint `duration` measures.
    It is `log`'s time less `.started`, which a `--start` freeze, or any freeze
    with none present, writes and only `log` and `close` remove. Two of the
    three 0-second runs logged 2026-10-23..30 were traced in their transcripts
    to one chained `freeze --start … && record && log` run after the reviews
    and fixes, the third not traced. One long run, 41,589 s, was traced to
    idle time: `.started` never expires, so a later freeze in the same session
    keeps it, the two-hour window or not. Nothing floors a 0 or clamps a
    negative from clock skew, and `stats` puts both in the median. The spec,
    `log --help` and stats' output say "first freeze", the skill "the freeze
    that opened the run"; the spec says any freeze opens a run when none is
    open but starts the clock only when none is running. Should the clock
    restart after the window, and should the skill forbid chaining the
    `--start` freeze with `log`?
  - 2026-10-30, from audit-checkpoint: `freeze`'s `appendGit` ignores every
    git exit status, its comment saying a diff that finds changes exits 1,
    true of `--no-index` alone; a tracked diff failing with 128 drops from the
    `.diff` while the summary still says "froze N paths"
  - 2026-10-30, from audit-checkpoint: `freeze` path handling. Paths go to git
    without `--literal-pathspecs`, so `pages/[id].tsx` is a glob; a relative
    `../x` is accepted and diffed from outside the root, where no fingerprint
    covers it; a directory is accepted, and `Record`'s frozen paths never match
    the files in it; two repositories sharing a basename overwrite each other's
    `checkpoint-<repo>.*` in one scratchpad; the `.diff` is truncated before
    the "the diff of N paths is empty" refusal and the diff and fingerprint
    steps that can fail, so a freeze stopped there leaves a `.hash` its
    `.diff` no longer matches
  - 2026-10-30, from audit-checkpoint: fingerprint edges in `WriteState` and
    `DirtyPaths`. A dangling symlink hashes as a deleted path; a symlink hashes
    by its target's bytes; an untracked non-regular path hashes empty diff
    output, a constant; a path holding a newline corrupts `State.Bytes` and
    `--stdin-paths`; with no commit yet, the `git diff HEAD` error is
    discarded and staged files are invisible to `hash`, `check` and the gate
  - 2026-10-30, from audit-checkpoint: `Record` writes through fixed
    `.d.tmp`, `.tmp` and `.paths.tmp` names per repository, so two sessions
    recording at once interleave, and swaps its three artefacts by separate
    renames, so a crash between them leaves record and copy disagreeing.
    `Open` rounds the age up, closing a run at about 119 minutes, not 120
- [ ] plan-parser-warnings: `ele plan` parses or warns on every line form it now
  holds or drops in silence
  - done when each line form below is parsed or drawn as a warning by `ele plan
    check`, each by a test, a `blocked:` list wrapped over lines among them;
    `Next` never returns the closing action while another action is open; and
    `Previous` follows a rename and a revert, each by a test
  - ruled 2026-11-02: `Blocked:` and `After:` match in any case, as `done
    when` does; an empty or malformed `blocked:` list, a self-block or cycle,
    a ticked item listing an open blocker, `* [ ]` items and `*`, `+` or
    numbered Inbox lines, and a header merged onto the line before each draw a
    warning naming the line; a fence indented four or more spaces is no
    fence, as CommonMark has it
  - 2026-10-30, from audit-plan: `plan`'s `Next` returns the first open
    action not held, and the closing action carries no `blocked:`, so when
    every other open action is held, by a cycle, prose or an `after:` date
    (probed), Next is the closing action, against `docs/plans.md`'s "Once the
    last action is the only one open"; `writePlanView`'s "every open item is
    blocked" cannot print for a plan in the documented format
  - 2026-10-30, from audit-plan: `plan.Parse` drops an item whose header an
    edit merged onto the previous item's last body line, so the line reads
    `done when x`, then the checkbox and `b: B` (probed): `b` is not counted,
    never Next, an edge to it "does not exist", and `ele plan check` says
    nothing
  - 2026-10-30, from audit-plan: `plan`'s `Previous` misses or misreports
    ticks. `git log -- path` has no `--follow`, so after a `git mv` the earlier
    ticks are gone (probed); a tick later reverted is still Previous while the
    item is Next (probed); a tick committed with a title reword is not a flip;
    any `rev-parse HEAD` failure, a missing git or a corrupt repository
    included, reads as "outside git"
  - 2026-10-30, from audit-plan: plan lines the parser holds or drops with no
    warning, each a question. `Blocked:` and `After:` capitalised are ignored,
    releasing the item, while `done when` matches any case; `blocked: a,`, an
    empty `blocked:` or `blocked: a c` read as prose and hold forever; a
    self-block or a cycle holds with no flag (probed); a ticked item listing
    an open blocker draws none; a fence indented four or more spaces still
    opens a fence, swallowing what follows (probed); `* [ ]` items and `*`,
    `+` or numbered Inbox lines are not counted
- [ ] ticket-robust: `ele ticket` keeps its ledger and lists whole
  - done when each defect below has a test that fails before its fix, or the
    user rules it out of scope
  - 2026-10-30, from audit-plan: `ticket`'s `WriteClaims` lists open issues
    with `--limit 200` and filters by the numbers asked for, so a number that
    is closed, unknown or open beyond the 200 gh returns is dropped unchecked,
    and with no other number flagged it prints "No open ticket is mistyped or
    grants itself execution."
  - 2026-10-30, from audit-plan: a plan item's ticket-audit ledger key is
    `filepath.Base` of the checkout plus plan path and name (`ticket`'s
    `planLocation`), so a worktree or a clone under another folder name loses
    its earlier rulings, and two repositories in same-named folders share
    them; ticket keys take the remote's project path instead, and the spec
    says "a plan action's repository"
  - 2026-10-30, from audit-plan: `ticket`'s `planItems` is a second plan
    parser disagreeing with `plan.Parse`: no fence tracking, so a fenced
    `## Inbox` or `- [ ] x` switches section or adds an item; any column-0
    `-` line under Actions is an action; a `#` heading does not end the
    section. No live plan hits it as of 2026-10-30
  - 2026-10-30, from audit-plan: the ticket-audit ledger. `Record` appends
    with a plain write, so a torn last line from a full disk or a kill makes
    `loadLedger` refuse the file until hand-edited, failing every
    ticket-audit §1, where its doc says it "writes nothing unless every
    record can be recorded"; the ledger grows a line per item per audit, and
    `WriteRulings` prints a re-recorded ruling once per line
  - 2026-10-30, from audit-plan: `ticket`'s `paged` stops at 20 pages, 2,000
    comments, and glab's `search` at 2,000 hits, both silently, where gh's
    path warns at its 500 cap; the comment "20 pages outrun any ticket's
    notes" is undated
- [ ] data-dir-edges: The data directory, owner map and profiles hold up to hand
  edits and symlinks
  - done when each defect below has a test that fails before its fix or the user
    rules it out, and `docs/` and the spec say what stays and what is refused,
    such as a relative `ELE_HOME`
  - ruled 2026-11-02: `sync.log` rotates at 1 MiB to `sync.log.1`, one kept,
    and `status` reads its tail; `backup` and `status` drop `claude-mem/`,
    left on disk and named in the docs as safe to remove; `design.md`'s
    "printed, never parsed" is fixed; `ELE_HOME` expands a leading `~/` and
    any other relative value is refused with an error naming it, every hook
    failing until fixed, the test pinning the relative form changing with it
  - 2026-10-30, from cli-freeze's persisted-files survey: `archive/claude-mem/`
    in the data dir is neither written nor read since its indexing was removed,
    yet `ele backup` copies it and `ele status` counts its size
  - 2026-10-30, from audit-cmd: `sync.log` is opened append-only by
    `startSync` and never cut, 189,635 bytes on the author's machine as of
    2026-10-30, and `ele status` reads it whole. Cap or rotate it?
  - 2026-10-30, from audit-cmd: `design.md` says `sync.log` is "printed, never
    parsed"; `ele status` parses its header stamps to show the log since the
    last success
  - 2026-10-30, from audit-rest: `datadir`'s `resolve` makes a relative
    `ELE_HOME` absolute against the working directory, which a test pins and
    `docs/data-and-privacy.md` does not mention; hooks run in each project's
    directory, so `ELE_HOME=data` gives each project its own store, and a
    literal `~/ele` from a settings `env` block becomes `<cwd>/~/ele`. On
    Linux, a process that resolved the old directory just before `move`
    renames it can recreate it, so `resolve` picks XDG and that process's
    writes are stranded. `moveLog` with both the old and new log present
    leaves the old one with no notice, as the spec has it
  - 2026-10-30, from audit-rest: `ownermap.Read` applies none of the checks
    `ele owner add` does, where `docs/owners.md` says the map "can be edited by
    hand": an owner `../x` becomes a notes folder, a `~/code` rule never
    matches, and `/a/b` and `/a/b/` clean to one key, the
    owner kept depending on map order. `OwnerOfPath` resolves no symlinks, so a
    cwd reached through one misses a rule given the other way. `Save` writes
    `"owners": null` when there are none, and nil path or project maps as
    `null`, through a fixed `.tmp`, unlocked
  - 2026-10-30, from audit-rest: `profile.Discover` follows symlinks and keeps
    no real path, so a `~/.claude-work` linking to `~/.claude` archives and
    indexes every transcript twice, under both profiles, a setup the project's
    recipe skill expects; `~/.claude-claude` and a bare `~/.claude-` both name
    to `claude`, sharing its archive keys, and `Current(".claudefoo")` names
    one `Discover` never lists; a home holding `[` fails its `filepath.Glob`
    (probed)
- [ ] help-text: `--help` matches the code
  - done when each help line below says what the code does, and
    `docs/commands.md` follows it
  - 2026-10-30, from audit-cmd: help text off the code. `backup`'s purpose and
    overview line omit `logs/`, which `backupParts` copies; `reminder list`'s
    omits everywhere reminders, which `remindersHere` lists; `docs/commands.md`'s
    `status` line omits the running sync, the options off their defaults and
    the sync log tail
- [ ] bashwork-reads: Read-only `ele` commands draw no notes ask
  - done when each command below reads as not work, by a test, and
    `stop-check.md`'s "What it reads" lists it
  - 2026-10-30, from audit-cmd: `bashwork`'s read list (`eleNotWork`) leaves
    out `ele checkpoint stats`, `ele config list`, `ele stats` and `ele usage`,
    as does `stop-check.md`'s "What it reads", so `/ele:checkpoint-stats` marks
    the session mid-task and draws a notes ask
- [ ] ci-hardening: CI lints what it ships and pins what it runs
  - done when CI runs actionlint, pins Task and shellcheck, and keys its build
    cache so a changed build saves; `install.sh` cleans up on a signal, accepts
    a version without its `v`, replaces the binary through a rename and limits
    `curl` to https
  - 2026-10-30, from audit-tooling: `install.sh`. Its `trap 'rm -rf "$work"'
    EXIT` does not fire on a signal under dash, the `/bin/sh` of Debian and
    Ubuntu that `curl … | sh` runs, so Ctrl-C mid-download leaves the work
    directory and its partial tarball (probed); `ELE_VERSION=0.12.0` without
    its `v` fails as "could not download", with nothing saying a `v` is
    needed; on Linux, GNU `install -m 0755` unlinks then writes, so a kill
    between them leaves no `ele` on PATH and every hook fails, with 127 or
    with 1 where `|| exit 1` wraps it, where macOS's renames a temp file into
    place; `curl` takes no `--proto '=https'`
  - 2026-10-30, from audit-tooling: no gate lints the workflows: actionlint
    is in neither `task check` nor `ci.yml`, run by hand for release-dispatch
    as its tick records, so a broken
    `if:`, a bad `needs` output or shell in a `run:` block of `release.yml` or
    `tap.yml` first fails during a release; `install.sh` is the only shell
    `lint`'s shellcheck reads
  - 2026-10-30, from audit-tooling: `ci.yml` floats what its own rumdl
    comment says must be pinned, since "the same commit passes today and fails
    tomorrow": `setup-task` installs Task `3.x`, and shellcheck comes from the
    runner image. Its Go build cache key is go.sum and go.mod alone, so after
    the first run for a go.sum every run hits it exactly, never saves, and
    rebuilds whatever changed since that first run
- [ ] eval-fixes: The evals grade what they name
  - done when each grader below fails the runs it should and passes the rest,
    `recipe`'s Audit Command bullet checks a relative path as a file, and the
    scaffolds' sandbox claims agree and carry their date, checked by an eval run
    the user asks for
  - 2026-10-30, from skills-audit: evals. `recipe`'s Audit Command bullet
    stops checking anything `command -v` prints that is not an absolute path,
    which the relative `./tools/forge/bin/forge` is, so `recipe-audit`'s
    `reports-script-holds` fails a run following it; `recipe-audit`'s
    no-edit graders miss `sed -i` and `perl -pi` on a `SKILL.md`; and
    `wrap-recipe`'s `proposes` wants the ask in the final reply, where its
    siblings also accept one through a question tool, as wrap step 3 asks
  - 2026-10-30, from skills-audit: eval scaffolds disagree on git:
    `plan-new-review`'s says git cannot run in the sandbox, while the four
    wrap cases' scaffolds `git init` so git does not root at the sandbox
    HOME, and `setup-maps` relies on it; recipe-audit, recipe-writes-dir and
    plan-new-review also state undated claims about the sandbox's denials.
    Which holds, as of a run?

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
- [ ] redact-extras: Redact the secret shapes the docs do not promise, and stop
  over-matching counts
  - done when each shape below is ruled in or out by the user, those in are
    redacted by a test, and `max_tokens`-style counts survive
  - 2026-10-30, from audit-index: secret shapes `redact` misses that the docs
    do not promise: `Authorization: Bearer <opaque>` and `--api-key s3cr3t`
    (both checked unchanged), `--password hunter2`, `scheme://user:pass@host`, `passphrase` and
    `credential` keys, Slack, Stripe, Google and npm token formats, a PEM block
    cut before its `END` line, a quoted value holding `\"`, whose tail after
    the escape leaks, and any value starting with `[`, which the
    already-redacted skip passes. Which does a 1.0 privacy promise cover?
  - 2026-10-30, from audit-index: `redact` over-matches counts: `"max_tokens":
    4096` becomes `[redacted:assignment]` (checked), as do `input_tokens: 123`
    and prose like `total tokens: 500`, so recall loses numbers
- [ ] mining-robust: One bad mining line or unit stops no listing or report
  - done when each defect below has a test that fails before its fix, or the
    user rules it out
  - 2026-10-30, from audit-rest: `mining`'s `Take` stores each classifier
    line checking only `session` and `timestamp`, and `lastPerUnit` decodes
    every stored result into a typed struct, so one line with
    `"is_checkpoint":"yes"` or a string `change.lines` fails
    `ele mine --report` from then on, as does a `report_ts` that is not
    RFC 3339 once a later cycle in that session is checked against it;
    re-judging under
    the same prompt hashes the same, so `Store` keeps the bad line and only a
    hand edit of the cache repairs it
  - 2026-10-30, from audit-rest: mining file formats. `Load` fails the whole
    cache on a blank line, where `Take` and `Named` skip them; `Store`
    re-marshals every held judgment through `Judgment`, dropping keys it does
    not declare; `loadWatermarks` decodes into `map[string]time.Time`, so a
    `"version": 2` fails every `ele mine`. `design.md` says the cache and its
    watermarks "ignore keys they do not know, so a later format adds a
    `version` key", true of reading the cache alone
  - 2026-10-30, from audit-rest: mining questions. `writeDigests` stops the
    whole listing at the first unit `Digest` cannot read, and that unit stays
    pending, so every later listing stops there; `readTranscript` takes no
    archive lock while `appendMember` appends in place. `Pending` can mine a
    live session's still-open window, and that judgment is final. Inline-era
    review cycles count under "Subagent era" and are timed. `upperQuartile`'s
    p75 can exceed the slowest launch on two values, as its test pins. `Take`
    takes `pending[0]` as the earliest, relying on its callers' sort.
    `subagentPath` joins an unchecked `agentId`. `Judgment`, `Timing`,
    `TimedUnit`, `UnitKey` and `Taken` cross into `cmd/ele` outside `dto`
- [ ] small-cli: Smaller CLI and tooling defects
  - done when each defect below is fixed with a test, or dropped on the user's
    ruling
  - 2026-10-30, from audit-cmd: `ele show --at` compares its argument with
    entry timestamps as strings (`cmd/ele` `show`), while `ele flag --at`
    parses RFC 3339 instants (`indexedMoment`). A flag time to the second skips
    entries stamped with milliseconds in that second, and an offset time lands
    hours off; `TestRunShowCentresOnAMatch` asserts the bare-prefix form
  - 2026-10-30, from audit-cmd: `ele context --size` (`contextSize`) scans the
    whole transcript; stop-check's `lastContextTokens` reads a bounded tail,
    since a transcript can pass 100 MB, and the spec says they compute it alike.
    `contextSize` also errors when the session's transcript glob matches two
    project directories; can a resumed session leave two?
  - 2026-10-30, from audit-cmd: the SessionStart block's handoff section has
    no budget, and the plan section caps only the next item's body
    (`planBodyBudget`), not its blocked list or the multi-plan summary; only
    notes shrink to `blockBudget`'s room, so long pending handoffs can push the
    block past the size the model sees whole. How long do real handoffs run?
  - 2026-10-30, from audit-cmd, for audit-index: `internal/index`'s `GitRoot`
    stops at a worktree's `.git` file, so a worktree's project is its folder's
    name, not the repository's
  - 2026-10-30, from audit-checkpoint: `density`'s `addedLines` keeps the tab
    git ends a `+++ b/` header with when the path holds a space (probed with
    `my file.go`), so the file gets no row, and a quoted header ending `"\t`
    fails `strconv.Unquote`, crediting its lines to the previous file
  - 2026-10-30, from audit-checkpoint: `glob`'s `match` backtracks
    exponentially: `*a*a*a*a*a*a*b` against 60 `a`s took 1.76 s (probed), and
    `OptedOut` runs it on up to 200 paths in the Stop hook
  - 2026-10-30, from audit-tooling: `task precommit` runs `format`, `check`,
    then `lint-fix` (`gofmt -s -w`) and `fix` (`go fix`), so the tree it
    leaves to commit is not the one `check` passed, and `lint`'s `gofmt -l`
    without `-s` never sees those rewrites; CONTRIBUTING documents the order
- [ ] audit-tests: Tests the audits found missing or unable to fail
  - done when each test below exists and has gone red once against a mutation,
    and the order nits beside them are fixed or dropped on the user's ruling
  - 2026-10-30, from audit-cmd: branches no test reaches: `configUnset`'s "has
    no value set", `copyTree`'s non-regular file, `indexedMoment`'s "no
    timestamped entry", `checkPlan`'s parse error, `writeResults`' "Top N
    (--limit)"
  - 2026-10-30, from audit-index: tests. `TestSearchTreatsQueryPunctuationAsWords`
    has no `stop-check` entry, so it proves only no error and no rows.
    Uncovered: a symlinked archive root, redaction of `changed`, hyphenated
    keys, an archive file shorter than `indexed_bytes`, `--since` across a
    local day boundary; `index`'s `disputingReply` and `followNotesAsk`
    heuristics have no test of their patterns alone
  - 2026-10-30, from audit-checkpoint: tests.
    `TestAppendKeepsTheFreezeTimeWhenTheLineIsNotWritten` writes
    `ReviewedFile`'s `.started` while `Append` reads the session's `RunFile`, so
    it cannot fail; `gate`'s `lastTurn`, `finalText`, `spokenText` and patterns
    have no unit test, only `cmd/ele`'s `TestRunCheckpointGate*`. Uncovered:
    over 200 dirty paths, an off-list dirty path after `Record`, a dangling
    symlink, an optout of `*e*`, a path with a space in `density`
  - 2026-10-30, from audit-plan: tests. `plan_test.go`'s "front matter never
    closes" case hits "is not key: value", leaving the real error untested;
    `TestNextIsNoneWhenEveryOpenItemIsBlocked` has no closing action.
    Uncovered: a self-block or cycle, a capitalised `Blocked:`, a merged
    header, an empty `blocked:`, a BOM, tabs or an empty file, which the
    parser handles (probed), `WriteStored`'s cleanup, a renamed or reverted
    tick, difflib's several hunks and empty or identical inputs, a
    fenced heading in `planItems`, a torn ledger line. Out of order:
    `corpus_test.go`'s and `plan_test.go`'s test funcs and the
    `TestCompareMatches…` and `TestDetectCLI…` tables
- [ ] audit-tidy: Order, comments and shape the audits found
  - done when each nit below is fixed or dropped on the user's ruling
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
    as a run-on; a stray blank line opens a loop in `status`; `remember` is
    one function of about 175 lines
  - 2026-10-30, from audit-index: exported with no caller outside its package:
    `store.OpenReadOnly` (tests alone), `archive.LoadManifest`,
    `index.ErrAmbiguousSession`. `index`'s `Reset` still drops
    `claude_mem_tables`, which no schema creates
  - 2026-10-30, from audit-index: small. `transcript`'s exported `Truncate`
    sits among unexported functions; `TestParseEvents`' "older stop hook block"
    rows break the table's order; `index_test.go`'s `rows` names a variable
    `r`; the comment above `toolErrorAttributes` documents `callLine`, an
    attribute key
  - 2026-10-30, from audit-checkpoint: comments off the code. `ValidateOptout`'s
    "nearly every one"; `Freeze`'s "inside an open run", where an expired run's
    clock is kept too; `appendGit`'s exit-status claim; `densityLead`'s
    second half, which describes `minimumAddedComments`; `doneClaim`'s dated
    figures, which do not explain the pattern; `gate`'s "Both passes have to
    appear" paragraph, which narrates; `ReviewedBytes`' doc naming one of six
    false returns
  - 2026-10-30, from audit-checkpoint: order and shape. Out of order:
    `density`'s `skipSuffixes` and `commentStarts`, `ValidateOptout`'s probes,
    rows in `TestMatchGlobMatchesAsBashDoes`, the moved-paths test, two
    `log_test.go` tables, `log_test.go`'s test funcs, `density_test.go`'s
    helpers. `gate`'s `finalText` and `spokenText` walk the same text markers;
    `appendDiff` repeats its diff-then-numstat shape per branch; `stats` names
    `a, b` in a sort closure; every test file's `writeFile` lives in
    `density_test.go`
  - 2026-10-30, from audit-plan: small. `plan.Resolve` panics on a nil
    `lookup` with any plan edge; `WriteClaims` prints "1 open tickets are
    flagged." and its test asserts it; `Corpus` says "no tickets written" for
    a plan with no items, and a plan corpus prints "ticket(s)" and
    `#<plan>-NN`; blank lines in
    front matter are refused; `planLocation`'s parameter `plan` shadows the
    package
  - 2026-10-30, from audit-plan: shape. `WriteClaims` execs through its own
    `run`, bypassing the injected `Commands`, so its test puts a fake `gh` on
    PATH; `ledger`'s `wellFormed` inlines `corpus`' `isWhitespace` and
    `trimWhitespace`; `cleanPath` is `filepath.Clean` without resolving `..`
    and keeping a leading `//`, with no test of its own
  - 2026-10-30, from audit-rest: small. `highlight.Message` strips control
    characters but not bidi overrides or U+2028, and its loops shadow the
    types `rule` and `span`; `config.Options` is an exported mutable slice,
    `Set` on a zero `Config` panics on its nil map, and `Load` names whichever
    bad option map order meets first; `datadir`'s `syscall.Flock` rules out a
    Windows build, which CONTRIBUTING says there is none of
  - 2026-10-30, from audit-rest: order and shape. Out of order: `mining`'s
    `finding` fields `RereviewKind` and `ReraisesUserDecision`, `units.go`'s
    exported functions, `cache.go`'s functions, `report.go`'s `checkpoint`
    method among the types; `notes_test.go`'s `TestParseRejectsInvalidNotes`
    and `TestEveryRoundTripsEachUnit` tables and `highlight_test.go`'s
    `TestMessage`. `Take` names the cache directory `cache` where its
    siblings say `directory`; `take.go` and `units.go`'s `Named` read JSON
    Lines with the same loop and key, which `UnitKey` names; `report.go`'s
    `cut` and `digest.go`'s `clip` truncate alike
- [ ] audit-questions: Answer the audits' open questions
  - done when each question below is answered, then moved to an action or
    dropped, and each nit beside them is fixed or dropped, on the user's ruling
  - 2026-10-30, from audit-cmd: questions on the data commands, each unsettled
    by the code. `backup` fails whole on a symlink, or on a note renamed away
    mid-walk by a concurrent `forget`. `reminder close` prints "due again" for
    a nudge whose done file is unreadable, shown again at once. `forget A B`
    refuses when A supersedes B. Owner commands blame "a sync" for a lock
    `backup` holds, and a `notes.Move` failing partway leaves moved notes with
    no owner-map save. `recall --limit 0` silently means 20. `config set` is an
    unlocked read-modify-write through a fixed `config.json.tmp`. `ticket`
    accepts `--cli` with `--plan` and ignores `ledger rulings --records`
  - 2026-10-30, from audit-cmd: `checkpoint.go`'s `hookInput.StopHookActive`
    takes `true` or `"true"`; `stopcheck`'s hook struct takes a bool alone, so
    a string would fail every Stop. Does Claude Code ever send the string?
  - 2026-10-30, from audit-index: crash and race edges, each unsettled.
    `archive`'s `syncFile` takes `tailHash` after reading the chunk, so a
    rewrite in place between the two records the new hash over old bytes, and
    a rewrite keeping the last 4 KiB is never seen; `appendMember` and
    `saveManifest` rename without syncing the directory; a recall during
    `ele sync --rebuild` reads a partial index with no sign of it
  - 2026-10-30, from audit-checkpoint: questions. Does the Stop hook's
    `session_id` still equal `CLAUDE_CODE_SESSION_ID` after `/clear` or a
    resume, since the gate reads one and freeze, record and optout the other?
    Does `lastTurn` miss a turn whose prompt holds array content, such as an
    image, so an earlier skill call suppresses the gate? `StateFile` sits under
    a shared temp dir another local user could pre-create. `close` prints
    `closed:` with no run open
  - 2026-10-30, from audit-plan: `ticket` copies Python: `isWhitespace` takes
    `\x1c`-`\x1f`, `splitLines` follows `str.splitlines`, the ledger and
    manifest are hand-written to match `json.dumps` byte for byte, errors print
    Python list reprs, and `difflib.go` ports CPython's `SequenceMatcher` and
    `unified_diff` (405 generated cases matched Python 3, probed) with no
    attribution. Is compatibility with the retired scripts still required? If
    so the spec should say `collapseWhitespace` and `itemHash` cannot change
    after 1.0 without flipping every ledger entry to changed; if not, simplify
  - 2026-10-30, from audit-tooling: questions and nits. Does `ele sync
    --background` detach inside Claude Code's SessionEnd window? `test` with
    `FORCE=0` still forces, since `{{if .FORCE}}` reads any value as set;
    `release:check` word-splits `$(which -a ele)`, so a path with a space
    breaks its listing; `bump-minor-pre-major` does nothing from 1.0.0; the
    plugin's description is duplicated verbatim in `marketplace.json` and
    `plugin.json`, which carries no `homepage`, `repository` or `license`
  - 2026-10-30, from skills-audit: checkpoint and review questions and nits. In
    branch scope §9's done-when counts committed-only paths only the uncommitted
    part can commit; §8's fixes reach §11's record with no gate run since §6;
    reviewers are given the gate log's path and never told to read it; §11's
    re-freeze names "the change list", not §8's adjusted one;
    `references/PLAN.md` runs `ele plan check` without the `cd` §2 uses;
    `references/DELTA.md` does not say whether `0 changed` in one of two roots
    stops both; §7's `ele plan list --project` failing on an owner ambiguity
    skips the tick proposal, though `ele plan list` from the root works; this
    adds that skip to the project-owner item; review step 6.2's "a line the
    change does not touch" drops a spec reviewer's missing requirement, which
    step 6.5 calls a blocker; review's `git archive` honours an author's
    `export-ignore`; review never discards `bugs.md`'s Tests to mutate; the two
    Errors tables and `LOG.md`'s `scope` and review's `outcome` lists are unsorted
  - 2026-10-30, from skills-audit: other skills' questions and nits. Skill and
    agent front matter is not alphabetised while `ticket-claims.md` is; `setup`'s
    first two steps and `remember`'s "Before writing" have no done when;
    `.rumdl.toml`'s "the body opens … with no title" is untrue of every skill but
    checkpoint, plan-audit and ticket-audit; `plan-audit`'s "Which plans" and
    plan pickup step 4 list ele plan check's warnings and miss the ele-data and
    inbox-at-close ones; `plan-practices.md` cites two sources its Sources lacks
    and defines a Proposed state no entry uses; no row in `errors.md` itself says
    not to rerun, only checkpoint's added ones; three of its rows describe hooks
    the plugin does not ship, and `setup` keeps its own retry text; `flag` lacks
    the deferred AskUserQuestion clause and does not say `--at` needs
    `--session`; `checkpoint-stats.md` has no `allowed-tools` and passes
    `$ARGUMENTS` unquoted; `setup` offers `--move` and `--move-to` where each
    refusal names one; the mining prompts show a multi-line template, then ask
    for one object per line; should wrap write a plan-candidate note unasked
    under `auto_note_writes`; `handoff`'s "two groups" regroups a list ele prints
    as one
  - 2026-10-30, from skills-audit: eval questions and nits. Grader regexes
    are copied across cases, so a fix lands once per copy; `plan-new-review`'s
    `reads-adr` counts only the Read tool, and can the installed plugin's arm
    read `plan-practices.md` if the sandbox denies `CLAUDE_CONFIG_DIR`?
    `reports-partial-help` wants `man git-log` where the skill prefers `git
    help log`; `reports-findings` needs a Go newer than
    1.21.0 and `reports-stale-flag` any Go, which CONTRIBUTING's Evals section
    does not state; `writes` and `writes-once`
    count Write calls only; `links` needs `.claude-a` literally in `ln -s`;
    `flag-saves`'s `reports` wants a note id with no error alternative; the
    forge fixture's comment describes npm
- [ ] atomic-helpers: One internal package holds the atomic write and the file
  lock, and the notes temp sweep covers `forgotten/`
  - done when `cmd/ele`'s `writeAtomically` and `internal/mining`'s copy, and
    `internal/notes`' lock and `internal/archive`'s, are each one function in
    one package with the old copies deleted, and a test shows a `.tmp` left in
    `forgotten/` by a killed `ele owner` move is swept
  - 2026-11-02, from notes-robust's and sync-robust's checkpoints: three
    copies of one helper pattern; the sweep covers `notes/` only (not checked
    beyond the checkpoint findings)
- [ ] stop-attachments: Count the notes ask when the checkpoint gate and the
  notes ask both continue one stop
  - done when a `hook-output-probe` run with both Stop hooks continuing one
    stop shows whether Claude Code writes one `hook_additional_context`
    attachment per hook or one holding both, and, if one, the index's
    notes-ask events and `--replay`'s `asked` count the ask, by a test seen
    red
  - 2026-11-04, from block-as-error: with one attachment, the event's reason
    is the gate's first line, so both miss the ask

### Release

- [ ] cut-v1: The user lands a `Release-As: 1.0.0` commit and merges the
  release PR
  - blocked: readme-guides, project-owner, skill-fixes, checkpoint-correctness, plan-parser-warnings, ticket-robust, data-dir-edges, help-text, bashwork-reads, ci-hardening, eval-fixes
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
- `bashwork` tracks shell assignments flat, so a subshell's reassignment reads
  as a write elsewhere: contrived, as the audit said (ruled 2026-10-31, from
  audit-cmd)
