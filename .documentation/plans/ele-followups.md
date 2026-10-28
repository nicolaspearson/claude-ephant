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
- As of 2026-10-28: never observed in a real session: a `--every 5m` nudge, advance-skips-close, an `--everywhere` note, `notes_ask` and `compact_hint_at`, `colour` off through the CLI, and `/ele:recipe audit` over the user's skills with the man-page, package-runner and delegating-script rules; the remember evals were not rerun after the skill's `--every` lines

## Decisions

- Items run verify, then fix, then build, with no `blocked:` edges, decided 2026-10-28 by the user: seeing shipped features work comes before adding more
- Evals run only on the user's go-ahead: they cost model calls
- The self-audit is its own plan: it reads flags and sent heads-ups once these items build them, so neither waits on it

## Actions

- [ ] real-session-pass: Try each never-observed feature in a real session, and rerun the remember evals
  - done when a `--every 5m` nudge, advance-skips-close, an `--everywhere` note, `notes_ask` and `compact_hint_at`, `colour` off through the CLI and `/ele:recipe audit` over the user's skills are each observed in a real session, what was seen written under this item and a failure added to the inbox; and the evals remember-auto-on, remember-auto-off, remember-reminder and remember-many are green, run on the user's go-ahead
- [ ] backup-plans: `ele backup` copies `plans/`, so a restore keeps closed plans' item names
  - done when a test backs up a data directory holding a closed plan's record, restores it, and a `blocked:` edge into that plan still resolves, seen red without the part; `ele backup`'s help, `docs/` and the spec name plans
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

## Out of scope

- The self-audit, reading ele's records to propose fixes to ele: its own plan, scoped in its own grill (ruled 2026-10-28, from ele-features' close)
- Reading heads-ups before they are sent: Claude Code exposes no hook, setting or SDK message for them (ruled 2026-10-28, from ele-features' close)
- A notes proposal from a sent heads-up: it is a prompt the session answers, and the Stop hook's notes ask covers that turn (ruled 2026-10-28, from sent-heads-ups)
- The `flagged` mining analysis: the self-audit reads flags (ruled 2026-10-28, from session-flags)
