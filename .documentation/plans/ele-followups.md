---
kind: plan
created: 2026-10-28
spec: .documentation/specs/design.md
---

# ele follow-ups

## Why

Close the gaps ele-features and recipe-audit left: see shipped features work in real sessions, fix two data bugs, harden the audit, then build the small features raised since.

## Context

- As of 2026-10-28: `backupParts` in `cmd/ele` lists `archive`, `config.json`, `forgotten`, `mining`, `notes` and `owner-map.json`, not `plans`
- As of 2026-10-28: `eleData` in `internal/plan` matches any note-id-shaped string or lowercase UUID; `Index.Session` in `internal/index` finds a session prefix with `LIKE`, which took 572 ms on a 795 MB index against 1-2 ms for `session = ?` and under 1 ms for a range
- As of 2026-10-28: the start block marks a lone pending handoff `shown` and lists several for the session to ask about; the handoff skill has only a Write section
- As of 2026-10-28: a path option must name a readable file; an approved recipe is written into each profile's `skills` folder
- As of 2026-10-28: Claude Code exposes no hook event, setting or SDK message for a heads-up before the user sends it (docs read in part); a sent one arrives as a prompt starting `✦ Heads up ·` and is indexed as a prompt
- As of 2026-10-28: never observed in a real session: a closed `--every` nudge hidden by /clear until its next due, and an `--everywhere` note in a real session's start block

## Decisions

- Items run verify, then fix, then build, with no `blocked:` edges, decided 2026-10-28 by the user: seeing shipped features work comes before adding more
- Evals run only on the user's go-ahead: they cost model calls
- The self-audit is its own plan: it reads flags and sent heads-ups once these items build them, so neither waits on it

## Actions

- [x] real-session-pass: Try each never-observed feature in a real session, and rerun the remember evals
  - done when a `--every 5m` nudge, advance-skips-close, an `--everywhere` note, `notes_ask` and `compact_hint_at`, `colour` off through the CLI and `/ele:recipe audit` over the user's skills are each observed in a real session, what was seen written under this item and a failure added to the inbox; and the evals remember-auto-on, remember-auto-off, remember-reminder and remember-many are green, run on the user's go-ahead
  - as of 2026-10-28: `--every 5m` nudge shown with ↻ in the /clear block; closing it at 10:44:41 re-armed it to 10:50, rounded up to the minute; hidden by /clear before 10:50 and shown after: not seen, test reminder forgotten
  - as of 2026-10-28, premise check: advance skips `reminder close` and `handoff close` in `cmd/ele` bashwork's subcommand table; `--everywhere` is a flag of `ele remember`; `colour`, `compact_hint_at` and `notes_ask` are options in `internal/config`; the recipe skill's audit holds the package-runner, script and man-page rules; the four remember evals are under `plugin/evals`; the installed binary, 4f7a8c9, holds the config options' commit and has the same `cmd/` and `internal/` as HEAD
  - as of 2026-10-28: advance-skips-close seen: a session whose only work was `ele reminder close` at its start ended its turn with no Stop notes ask and no hook-block event
  - as of 2026-10-28: colour off seen: with `ele config set colour false`, run outside a session since `config set` refuses inside one, `ele reminder list --colour` printed 0 escapes against 8 with it on; unset after
  - as of 2026-10-28: an `--everywhere` fact showed in `ele context` for owners personal, hyve and bitgorilla, and in `ele session-start` fed a hook input for a directory no owner covers; an owner added later not tried; the probe note forgotten
  - as of 2026-10-28: the installed plugin is the 4f7a8c9 cache, whose recipe skill predates the package-runner, delegating-script and man-page rules
  - as of 2026-10-28: `/ele:recipe audit` over 10 skills in a `task dev` session: the npx claims were marked not checked and nothing was run with the package; npm's and corepack's node shims were checked against the source they load; git subcommand flags and `env -u` held through man pages read with `col -b`, without `LC_ALL=C`; two real findings fixed in the user's skills
  - as of 2026-10-28: with notes_ask false and compact_hint_at 20000, a new session's edit turn ended with no notes ask and with `Context 55k. At a pause: /compact`; its first turn, at 54k, got no hint, though stop-check fed that transcript cut at the same Stop prints one
  - asserted at the eval boundary through the clone plugin at a0e9fa2: the four remember evals 1.00 each; in real sessions on binaries 4f7a8c9 and, from the recipe audit on, a0e9fa2 (same `cmd/` and `internal/`), every feature but a nudge hiding on /clear until due and an `--everywhere` note in a start block, both carried to the inbox
- [x] backup-plans: `ele backup` copies `plans/`, so a restore keeps closed plans' item names
  - done when a test backs up a data directory holding a closed plan's record, restores it, and a `blocked:` edge into that plan still resolves, seen red without the part; `ele backup`'s help, `docs/` and the spec name plans
  - asserted in the backup-plans commit: TestRunBackupKeepsClosedPlansSoARestoreResolvesTheirEdges closes a plan, deletes its file, backs up, points ELE_HOME at the backup and gets the closed-plan flag on a `blocked:` edge into it; red with `plans` or `notes` left out of `backupParts`; `ele backup`'s help, docs/data-and-privacy.md and the spec's storage tree and backup paragraph name closed plans
- [ ] plan-check-ids: `ele plan check` warns on a note id or session UUID only when ele resolves it, and `ele show` finds a session prefix by range
  - done when a plan holding a repository's own UUID gets no warning and one holding a real session id does, each seen red; with the index unreadable every match warns, as before, so a broken index never silences the check; `ele show <prefix>` resolves through `session >= ? AND session < ?`
- [ ] no-run-step-shapes: recipe-audit's `no-run-step` grader catches the command shapes `no-run-script` does
  - done when its regex matches a second line, an `env`, `timeout N` or `VAR=value` prefix, `sh -c` and a quoted path running `deploy-dev`, and not `cat`, `grep` or `realpath` of it, by hand probe
- [ ] man-hyphens: recipe audit's man-page lookup runs `LC_ALL=C man <page> </dev/null | col -b`, so a page renders ASCII hyphens
  - done when the skill, `docs/` and the spec say so and a run on macOS still finds `--oneline` in `git-log`; Linux not verified, said in the tick
- [ ] handoff-picking: Option `lone_handoff_pending`, boolean, default false, shows a lone handoff as several are; `/ele:handoff list` shows pending and closed handoffs
  - done when a Go test leaves a lone handoff pending, with the ask line, with the option on and marks it shown with it off, each seen red, and a handoff eval has `/ele:handoff list` print pending and closed handoffs from `ele handoff list --all`
- [ ] recipe-dir: Option `recipe_dir` names a folder an approved recipe is written to once, each profile's `skills/<name>` linked to it
  - done when a recipe-writes eval with the option set writes `<recipe_dir>/<name>/SKILL.md` once and links it from each scaffolded profile, a profile already holding the name left alone, seen red; a config test refuses a path that is not a folder; `/ele:recipe audit` lists the folder; unset, recipes are written as now
- [ ] session-flags: `ele flag good|bad "<why>" [--session <id>]` and `/ele:flag` stamp a session moment, listed by `ele flag list`
  - done when a test saves a `flag` note with session and time, lists it with `ele flag list`, and keeps it out of the start block, each seen red, and an eval has `/ele:flag` save one
- [ ] sent-heads-ups: Sync indexes a prompt starting `✦ Heads up ·` as entry kind `heads-up`, which `ele recall --kind` filters
  - done when a sync test indexes one as `heads-up` and not `prompt`, seen red, `ele recall --kind heads-up` finds it, and the spec says an existing index needs `ele reindex` to reclassify old ones
- [ ] onboarding: `/ele:setup` finds a missing owner or an unowned directory through `ele status` and sets it up with `ele owner`
  - done when an eval scaffolding an empty data directory has the skill propose the `ele owner` commands and, on approval, run them and `ele sync`, so a second `ele status` names an owner here, seen red on a skill without the owner step
- [ ] Fold what this plan built, and its out-of-scope lines as `Not doing:`, into the spec, run `ele plan done .documentation/plans/ele-followups.md`, and delete this plan

## Inbox

- 2026-10-28, from real-session-pass: see a closed `--every` nudge hidden by /clear before its next due and shown after, and an `--everywhere` note in a real session's start block under an owner it was not saved from
- 2026-10-28, from real-session-pass: stop-check missed the size hint on a session's first turn: at its Stop the turn ended at 54k with compact_hint_at 20000 and nothing printed, yet the same transcript cut at that Stop and fed by hand gets the hint; guess, not verified: the turn's usage lines were not on disk when the hook read the transcript
- 2026-10-28, from backup-plans' checkpoint: `WriteStored` in `internal/plan` truncates and writes in place, where notes write a temp file and rename, so a backup copying `plans/` during an `ele plan done` can take torn JSON, which `loadPlanSet` then drops and the edge reads "does not exist"; write through a temp file and rename

## Out of scope

- The self-audit, reading ele's records to propose fixes to ele: its own plan, scoped in its own grill (ruled 2026-10-28, from ele-features' close)
- Reading heads-ups before they are sent: Claude Code exposes no hook, setting or SDK message for them (ruled 2026-10-28, from ele-features' close)
- A notes proposal from a sent heads-up: it is a prompt the session answers, and the Stop hook's notes ask covers that turn (ruled 2026-10-28, from sent-heads-ups)
- The `flagged` mining analysis: the self-audit reads flags (ruled 2026-10-28, from session-flags)
