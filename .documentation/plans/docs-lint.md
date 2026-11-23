---
kind: plan
created: 2026-11-22
---

# Docs lint

## Why

A CLI or plugin change that skips the docs is caught today only by a later
hand diff. The skill lint fails a skill with no row in the two skill tables,
but not a slash command, not a row left behind by a rename, and nothing
compares `docs/commands.md` with ele's help. This plan makes that drift fail
`task check`.

## Requirements

1. A slash command under `plugin/commands/` with no `/ele:<name>` row in
   `docs/commands.md`'s Skills and slash commands or `README.md`'s Skills at a
   glance fails `task check`, as a skill does today
2. A row in either section naming no present skill or command fails
   `task check`
3. Every synopsis line ele's help prints, for `ele --help` and each
   `ele <command> --help`, is a whole line of `docs/commands.md`, and every
   `ele` line in that file's code fences is one of them; flag descriptions are
   not checked
4. Requirement 2 also covers the `/ele:` names in `docs/commands.md`'s Used by
   the skills and hooks table
5. A group or subcommand added to ele's help is run by the help test with no
   other edit

## Context

- As of 2026-11-22: `internal/skilllint` `CheckDocs` builds the expected cells
  from the directories under `plugin/skills/` only, and looks each up in the
  first-column cells `sectionRows` returns; nothing reads `plugin/commands/`,
  and nothing diffs the rows the other way. `sectionRows` also returns the
  header and separator cells. It runs only from `TestPluginSkillsDocumented`,
  under `task test`
- As of 2026-11-22: `plugin/commands/` holds `checkpoint-stats.md` and
  `ticket-claims.md`, flat files with no `name:` field. Both sections hold 17
  rows, exactly the 15 skills and 2 commands, so the new row checks land green
- As of 2026-11-22: the Used by the skills and hooks table in
  `docs/commands.md` has 7 rows naming `/ele:` skills after the first cell, one
  or two to a cell, as in `/ele:checkpoint` and `/ele:wrap`. Its
  `checkpoint stats` row escapes `|` as `\|` in the first cell, which
  `sectionRows`' split on `|` breaks into an extra cell
- As of 2026-11-22: cmd/ele's `usage` const holds every synopsis line ele's
  help prints, 64 of them, each a whole line of `docs/commands.md`, and every
  `ele` line in its code fences is a `usage` line. Doc order differs from
  `usage` order, so both checks are by set, not by position
- As of 2026-11-22: of 30 flag descriptions sampled from cmd/ele, one appears
  in `docs/commands.md`; the docs paraphrase them
- As of 2026-11-22: cmd/ele `TestRunAnswersHelpForEveryCommand` runs `--help`
  over a hand-written list of group subcommands plus every key of `commands`
  not in a hand-written list of groups. `ticket ledger` has a third level,
  `record` and `rulings`, and the test runs it as `ticket ledger`
- As of 2026-11-22: each `ele <command> --help` prints only `usage` lines,
  chosen by cmd/ele `synopses`, so `usage` holds every synopsis help prints
- As of 2026-11-22: CONTRIBUTING.md's Checks section and CLAUDE.md's Docs
  section describe the row rule for `plugin/skills/` only; no spec describes
  the skill lint

## Decisions

- Two items, one a package: the row checks in `internal/skilllint`, the help
  check in cmd/ele, which alone can read `usage` since `internal/skilllint`
  cannot import package main. Each updates the prose describing its own check
  in the same change
- The help check compares synopsis lines only, which carry every flag name:
  the docs paraphrase flag descriptions, so a description check would fail on
  wording, not drift
- A reverse row check keeps only first cells of the form `/ele:<name>`, so
  header and separator cells, and prose under the tables, are never read as
  rows. In the Used by the skills and hooks table it reads every `/ele:<name>`
  code span in a row instead, since a cell names one or two and an escaped
  `\|` shifts the columns

## Actions

- [x] skill-rows: make `internal/skilllint` `CheckDocs` require a row for each
      `plugin/commands/*.md` as for each skill, and fail a row naming no
      present skill or command in either section and in `docs/commands.md`'s
      Used by the skills and hooks table; update CONTRIBUTING.md's Checks
      section and CLAUDE.md's Docs bullet to match
  - done when `TestCheckDocs` has cases for a missing command row, a stale
    row in each section and a stale name second in a two-name cell of the
    hooks table, each gone red once by mutating the check,
    `TestPluginSkillsDocumented` passes on the repository, CONTRIBUTING.md's
    Checks section and CLAUDE.md's Docs bullet name commands and stale rows,
    and `task format`, `task check FORCE=1` and `task audit` pass
  - as of 2026-11-23: premises re-derived, all hold; `TestCheckDocs` is a
    table over a one-skill temp tree, so the new command path changes every
    `CheckDocs` call site
  - asserted in the skill-rows commit: `TestCheckDocs` cases for a missing
    command row, a stale row in each section and a stale name second in a
    two-name hooks cell each went red under a mutation of `CheckDocs`;
    `TestPluginSkillsDocumented` passed on the repository; CONTRIBUTING.md's
    Checks and CLAUDE.md's Docs bullet name commands and stale rows;
    `task format`, `task check FORCE=1` and `task audit` exited 0
- [x] help-docs: add a cmd/ele test that every `usage` line is a whole line of
      `docs/commands.md` and every `ele` synopsis line in its code fences is a
      `usage` line, and derive `TestRunAnswersHelpForEveryCommand`'s groups
      and subcommands from `usage`; update CONTRIBUTING.md's Checks section
  - done when the new test fails on a `usage` line dropped from the docs and
    on a stray `ele` line added to them, each gone red once, a subcommand
    and a group added to `usage` alone are each run by the help test, as are
    `ticket ledger record` and `ticket ledger rulings`, CONTRIBUTING.md's
    Checks section names the help check, and `task format`,
    `task check FORCE=1` and `task audit` pass
  - as of 2026-11-23: premises re-derived. `usage` has 64 synopsis lines and
    `docs/commands.md`'s `text` fences the same 64, no duplicates, so the
    hooks section no longer repeats a synopsis. Groups dispatch through local
    maps in their own files, so `usage` is the only uniform source; a usage
    line's path is its leading lowercase words. `ele ticket ledger record
    --help` prints both ledger synopses under the name `ticket ledger`, and
    `purposes` has no key for either, so the help test's assertions take the
    longest command name with a `purposes` entry, not the full path
  - asserted in the help-docs commit: `TestCommandsDocHoldsEveryUsageLine`
    went red on `ele version` dropped from `docs/commands.md` and on a stray
    fenced `ele bogus`; `TestRunAnswersHelpForEveryCommand` ran and failed on
    `ele plan zzz` and `ele zzz thing` added to `usage` alone, and went red on
    `ticket ledger record` asserted under its full path; a probe of the
    derived paths listed both `ticket ledger record` and `rulings`;
    CONTRIBUTING.md's Checks names the help check; `task format`,
    `task check FORCE=1` and `task audit` exited 0
- [ ] Fold each Decision through
      `/ele:plan-audit close .documentation/plans/docs-lint.md`, then propose
      each out-of-scope line as a `decision` through `/ele:remember`, run
      `ele plan done .documentation/plans/docs-lint.md`, and delete this plan

## Inbox

## Out of scope

- Checking flag descriptions against the docs: the docs paraphrase them, so
  the check would test wording, not drift (ruled 2026-11-22, from
  `/ele:plan new docs-lint`)
- The skill lint's front-matter checks for `plugin/commands/*.md`: a separate
  gap, with no docs row involved (ruled 2026-11-22, from
  `/ele:plan new docs-lint`)
