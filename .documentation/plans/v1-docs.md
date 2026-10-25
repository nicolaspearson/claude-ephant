---
kind: plan
created: 2026-10-23
---

# v1 docs

## Why

v1 is for users other than the author. The code still indexed a one-off claude-mem export no other user has, and the docs read as the author's build log: dated measurements, a cutover, a build order and the tools removed along the way. A user needs to know what ele does and how to use it, a contributor how to build and release it, and neither needs the history, which git keeps.

## Context

- As of 2026-10-23: outside `.git`, `claude-mem`, `headroom` and `rtk` appeared in 17 files before `drop-claude-mem`; after it, in `.documentation/specs/design.md`, in current-design sections (Owners, Recall, Storage, Rollups, Usage) as well as the history, and in `.documentation/plans/ele-features.md`'s `## Context` citing claude-mem's `ContextBudget.ts`
- As of 2026-10-23: `README.md` holds install, update, options and a Contributing section; there is no `docs/`, `CONTRIBUTING.md` or `CLAUDE.md`
- As of 2026-10-23: `.documentation/specs/` holds `design.md`, `stop-check.md` and `checkpoint-and-ticket-audit.md`

## Decisions

- claude-mem indexing is removed, not renamed: no v1 user has the export. A data directory's `archive/claude-mem/` stays on disk untouched; the index version bump drops its rows at the next sync
- User docs are a short README linking to `docs/`, one page per feature plus a command reference: a README holding everything is too long to scan, and a page per feature is what a user lands on from a search
- Specs describe the current design only: behaviour, constraints and the reason for each decision, with no measurements, timelines or removed tools. Git history keeps the old text, so nothing moves to a history file
- `CONTRIBUTING.md` takes the README's Contributing section and the build, release and tap material from the specs; `CLAUDE.md` holds what an agent working in the repository needs and points to `CONTRIBUTING.md` for the rest
- The build, release and tap material leaving `design.md` seeds a minimal `CONTRIBUTING.md`, moved verbatim, for `contributing` to rewrite: parked anywhere else, it would be lost before that item is built. Decided 2026-10-23
- Specs drop "decided <date>" stamps and every dated figure about ele's own data, but keep a dated `as of` on a fact about Claude Code's behaviour that a decision rests on, since that date tells a reader to re-check it. Decided 2026-10-23
- `specs-current` cuts the mining material too: the run-folder pointers, the Rollups figures and the Stop-hook blocks section. `mining-reports`' items are reworded so that plan's fold adds design, not figures. Decided 2026-10-23
- Open questions in a spec become `question` notes, and build briefs' Done means, move history and author-only references (claude-config, `hook-firings.py`, `~/.agents`, `~/ele-backup`) are cut, keeping the behaviour behind them. Decided 2026-10-23
- The README's Contributing section moves to `CONTRIBUTING.md` verbatim in `readme-for-users`' change, for `contributing` to rewrite: it holds material `CONTRIBUTING.md` lacks, and cut from the README first it would be lost before that item is built. Decided 2026-10-25
- A docs item is done when the page exists, `task lint` (whose rumdl run fails a relative link to a missing file or heading) and `task lint-docs` pass, and each command or flag it names is checked against `ele <command> --help`

## Actions

- [x] drop-claude-mem: Remove claude-mem indexing from the index, sync, status, recall, stats and usage, the recall skill and the tests; bump the index version
  - done when `grep -rniE 'claude-mem|headroom|\brtk\b'` outside `.documentation/` finds nothing, `task check` passes, and `ele sync` on an index holding claude-mem rows rebuilds it and `ele stats` shows no claude-mem source
  - asserted in the drop-claude-mem commit: the grep finds nothing outside `.documentation/`; `task check FORCE=1` passes; a copy of the live v10 index, holding 130,175 claude-mem entries, synced by this build against an empty archive was rebuilt ("built by another version of ele"), and `ele stats` then listed no claude-mem source
- [x] specs-current: Rewrite `design.md`, `stop-check.md` and `checkpoint-and-ticket-audit.md` as the current design: cut the cutover, build order, testing phase, measurements, mining figures and Removed sections, and every claude-mem, headroom and rtk line; move build, release and tap material out for `contributing`
  - done when the three specs hold no dated measurement or timeline, the grep above finds nothing under `.documentation/specs/`, and every behaviour the old `design.md` stated is either kept or contradicted by the code
  - as of 2026-10-23, premise check: the grep outside `.documentation/` finds nothing, so every claude-mem line in `design.md` describes removed code. Its hits are not confined to the history and cutover sections: they are also in Owners, `### claude-mem`, Recall's ranking and export bullets, Storage, Rollups for mining, Usage report, `## claude-mem cutover`, `## Build order` and `### Removed 2026-10-15`. `stop-check.md` and `checkpoint-and-ticket-audit.md` have none
  - as of 2026-10-23: the sections to cut exist in `design.md`: claude-mem cutover, Build order, Testing phase, Phase 2's figures (Rollups for mining, Usage report), and Removed 2026-10-15. `stop-check.md` is still shaped as a build brief (What to build, Decided, Open, Done means); `checkpoint-and-ticket-audit.md` carries its move history (the opening lines, Into ele) and a measurement-only Stop-hook blocks section
  - as of 2026-10-23: the build, release and tap material is `design.md`'s Integration bullets on `task dev`, `task install`, the `release:*` targets, the release workflow and the tap; every target it names is listed by `task --list`, and `.github/workflows/` holds `ci.yml`, `release.yml` and `tap.yml`
  - as of 2026-10-23: other plans lean on these specs. `skill-review` (spec `checkpoint-and-ticket-audit.md`) cites its Errors line, its cost section, the rule that a case only some runs meet sits beside SKILL.md, and the Propose wording in `stop-check.md`. `mining-reports/delete-mining-runs` dropped the specs' pointers to the mining run folders, and that plan's closing action folded into `checkpoint-and-ticket-audit.md`'s Stop-hook blocks section, both reworded in this item's change
  - as of 2026-10-23, before this item's change: the specs also named the author's other repositories and paths: claude-config, `hook-firings.py`, agent-skills, `~/.agents/skills` and `~/ele-backup-2026-10-17/`
  - asserted in the specs-current commit: `grep -rniE 'claude-mem|headroom|\brtk\b' .documentation/specs/` finds nothing; the only dates left in the three specs are `as of` lines on Claude Code behaviour; a fresh comparison of HEAD against the tree, checked against the code, found 8 behaviours lost and the checkpoint's bugs reviewer 1 more (imported judgments carry no digest), all restored; `task check FORCE=1` and `task lint-docs` pass
- [x] user-guides: Write `docs/` pages: owners, notes and recall, plans, handoffs and reminders, checkpoint and ticket-audit, data and privacy (storage, backup, redaction, no background model), and a command reference
  - done when each page meets the docs done-when decision
  - asserted in the user-guides commit: the seven docs/ pages exist; task lint-docs, extended in the same commit to scan docs/ for file:line citations and fail on a relative link whose file is missing (shown red on a planted broken link), passes; a one-off check found every #anchor matches a heading, every --flag in ele's help, and every synopsis in commands.md verbatim in ele --help; three reviewers' claims checked against the code; task check FORCE=1 passes
- [x] markdown-lint: Add rumdl, installed like shellcheck, to `task lint` and `task format`, with heading-anchor checks; reformat the existing `.md` files in their own commit
  - done when `task lint` fails a link to a missing heading or file, and `task format` on the reformatted tree changes nothing
  - asserted in the markdown-lint commit: on a copy of `docs/`, a planted `#handoffsx` failed MD051 and a planted `missing.md` failed MD057, so lint-docs' own relative-link loop was dropped; `.rumdl.toml` turns MD013 off and leaves out `CHANGELOG.md` and `plugin/evals/`; CI installs rumdl 0.3.0 with pipx, and its first run, on the reformat commit's push, found no issues in 38 files
- [x] readme-for-users: Rewrite `README.md` for users: what ele is, install, quick start, the skills at a glance, options, update, and links to `docs/` and `CONTRIBUTING.md`
  - done when the README meets the docs done-when decision and holds no contributor material
  - as of 2026-10-25, premise check: contributor material sits in Contributing (`README.md:116-154`), Prerequisites' "or to work from a clone" (`README.md:30-31`) and Update's `task upgrade` line (`README.md:89`). Contributing holds `task install:local`, `task eval` and its cost, the `gh secret set HOMEBREW_TAP_TOKEN` renewal, the clone build's pseudo-version and the specs pointer, none of them in `CONTRIBUTING.md`
  - as of 2026-10-25: the README's skills list matches `plugin/skills/` (9) plus `plugin/commands/` (`checkpoint-stats`, `ticket-claims`); its four options match the table at `docs/data-and-privacy.md:84`
  - as of 2026-10-25: Options become one line per option linking to `docs/data-and-privacy.md#options`; a Quick start after Install moves owner setup out of Install and walks remember, recall and handoff, each linking to its page
  - asserted in the readme-for-users commit: README.md holds what ele is, Skills at a glance, Prerequisites, Install, Quick start (owner, remember, recall, handoff, each linking its docs/ page), Options as one line each linking docs/data-and-privacy.md#options, Update, and a CONTRIBUTING.md pointer; a grep for task, clone, release: and eval finds nothing; the old Contributing section is byte-identical at CONTRIBUTING.md's Working from a clone; every command named matches ele <command> --help; task lint, task lint-docs, task check FORCE=1 and task audit pass
- [x] contributing: Write `CONTRIBUTING.md`: clone setup with Task, `task install`, `task dev`, the gates, evals, repository layout, specs and plans, conventional commits, the release and tap workflows and the tap token's renewal
  - done when it meets the docs done-when decision and each `task` target it names is listed by `task --list`
  - as of 2026-10-25, premise check: `CONTRIBUTING.md` holds three overlapping parts: Working from a clone (the README's old section, wrapped prose), Build and install and Releases (the specs' material, one-line bullets). Every `task` target they name is in `task --list`; `.github/workflows/` holds `ci.yml`, `release.yml`, `tap.yml`; release-please reads `release-please-config.json` and `.release-please-manifest.json`
  - as of 2026-10-25: nothing in it yet covers clone setup (`task setup`, `setup-tools`, rumdl and shellcheck installed by hand), repository layout, specs and plans, or conventional commits; evals are named but a case's layout is not described: `graders/` plus `prompt.md` (`recall`) or `case.yaml` and `scaffold.sh` (the other five)
  - as of 2026-10-25: sections Setup, Build and run, Checks, Evals, Repository layout, Specs and plans, Commits, Releases; layout lists top-level directories only; plans link `docs/plans.md` for the format; prose wrapped at about 80, the dated `--plugin-dir` fact kept
  - asserted in the contributing commit: CONTRIBUTING.md has Setup, Build and run, Checks, Evals, Repository layout, Specs and plans, Commits and Releases (install script, Homebrew tap, the tap token's renewal), replacing the three overlapping parts; every fact in the old file is kept except the README's install order; each `task` target it names is in `task --list`, and `ele version` is in its help; the claims were checked against Taskfile.yml, the three workflows, release-please-config.json and plugin/evals by three reviewers; task lint, task lint-docs, task check FORCE=1 and task audit pass
- [ ] claude-md: Write `CLAUDE.md` for agents working in the repository: layout, the gates to run, where specs and plans live, and a pointer to `CONTRIBUTING.md`
  - blocked: contributing
  - done when it meets the docs done-when decision and a fresh session in the repository loads it
- [ ] hooklog-comment: `internal/checkpoint/gate.go`'s HookLog doc comment names claude-config ("the format claude-config's hooks write"); say what the format is instead, as `checkpoint-and-ticket-audit.md` now does
  - done when `grep -rn 'claude-config'` outside `.documentation/` finds nothing and `task check` passes
  - 2026-10-23, from specs-current; moved from the inbox 2026-10-25
- [ ] Propose each out-of-scope line as a `decision` through `/ele:remember`, run `ele plan done .documentation/plans/v1-docs.md`, and delete this plan

## Inbox

## Out of scope

- Reword ele-features' `ContextBudget.ts` citation: it cites prior art, depends on nothing, and goes with that plan when it closes (ruled 2026-10-23, from drop-claude-mem)
