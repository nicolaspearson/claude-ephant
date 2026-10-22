---
kind: plan
created: 2026-10-22
spec: .documentation/specs/design.md
---

# ele configuration, reminders and recipes

## Why

Give ele a configuration a user can switch and tune from one place, reminders that surface at session start from a date or every N days, and recipes that turn a reusable procedure into a user skill and keep skills from going stale. Configuration comes first so each feature after it is switched through it.

## Context

- As of 2026-10-22: `plugin/.claude-plugin/plugin.json`'s `userConfig` holds one option, `checkpoint_gate` (boolean, default on), read by the checkpoint hooks through `CLAUDE_PLUGIN_OPTION_CHECKPOINT_GATE`
- As of 2026-10-22: note kinds are decision, fact, handoff, plan and question (`Kinds` in `internal/notes`); a dated reminder has been written as a fact note, which shows from the day it is written, not from its date, newest first within the session-start block's budget (`contextBudget` in `cmd/ele`)
- As of 2026-10-22: `~/.claude/hooks/measure-reminder.sh`, a SessionStart hook in claude-config, prints a notice when `~/.claude/cache/hook-stats.last` is missing or over 7 days old; `~/.claude/scripts/hook-firings.py`, which `/hook-stats` runs, touches that file
- As of 2026-10-22: `ele remember` refuses a note over 3 lines, and every live note loads each session, so a multi-step procedure has no home in ele

## Decisions

- One plan for all three, decided 2026-10-22 by the user: two plans are active and the plan audit flags more than three
- Configuration's structure is settled before any option is added, decided 2026-10-22 by the user, so auto note writes, nudges, recipes and reminders each land as an option in one layout
- A reminder shows from its date until closed; a nudge is a recurring reminder that comes back N days after it was last marked done, decided 2026-10-22 by the user
- ele:recipe gets both halves, decided 2026-10-22 by the user: proposing a recipe as a user skill, with the Write/Skip approval notes use, and auditing skills for stale commands and versions
- Auto note writes default to off, decided 2026-10-22 by the user: no user gets notes written without asking unless they turn it on
- A recipe always asks Write/Skip, whatever auto note writes says, decided 2026-10-22 by the user: it writes a skill under the user's skills, heavier than a note

## Actions

- [x] config-structure: Settle with the user, in rounds, what of ele is configurable, where configuration lives (plugin `userConfig`, an ele config file, or both), and how it is read and set from the CLI, the skills and the guide; then build that layout with `checkpoint_gate` moved onto it
  - done when design.md holds a Configuration section naming the layout and how each surface reads and sets it, `checkpoint_gate` reads through it with a test seen red, and `task check FORCE=1` passes
  - asserted in the config-structure commit: design.md's Configuration section names config.json's layout, its resolution order and how hooks, the CLI and `ele help` read and set it, skills reading it as hooks do once one needs to; both checkpoint hooks read `ele config get checkpoint_gate`, and checkpoint-review.test.sh's gate cases went red with the hooks reading a wrong key and with a broken file read as gate off; `task check FORCE=1` green. Not run through the installed plugin: that waits for a release
  - as of 2026-10-22: run through the installed plugin (2724da02de91), with the ele binary printing v0.11.0, in a real session: with `ele config set checkpoint_gate false` a done claim over a new untracked file passed unblocked, and after `ele config unset checkpoint_gate` a second such claim was blocked naming the file and `ele:checkpoint`
- [ ] auto-note-writes: An option, off by default, that when on has ele write proposed notes without the Write/Skip ask
  - done when switching the option changes whether a proposed note is written without asking, each state tested and seen red at each surface config-structure names
- [ ] reminders: Dated reminders: saved through the CLI and a skill with a date, shown in the session-start block from that date until closed, and listed and closed through the CLI
  - done when a reminder dated tomorrow does not show at session start, one dated today does, a closed one does not, and none shows with the option off, each tested and seen red; the CLI lists and closes them, and the skill saves one
- [ ] nudges: Recurring reminders that come back N days after they were last marked done, with `measure-reminder.sh`'s hook-stats notice moved onto one
  - blocked: reminders
  - done when a nudge marked done does not show until N days later and none shows with the option off, each tested and seen red; the hook-stats notice comes from ele, running `/hook-stats` marks it done, and the claude-config changes are handed to the user as commands
  - next: settle whether ele reads `hook-stats.last` or `hook-firings.py` marks the nudge done; either way `measure-reminder.sh` is removed from claude-config
- [ ] recipe-propose: ele:recipe proposes a reusable procedure seen in a session as a user skill, with the same Write/Skip approval as notes whatever auto note writes says, and writes the approved one under the user's skills
  - done when an eval or test seen red has ele:recipe propose a multi-step procedure from a session transcript, an approved recipe is written as a skill a fresh session lists, and with the option off nothing is proposed
- [ ] recipe-audit: Audit the user's skills for commands, flags and versions that no longer exist, as plan-audit does for plans
  - done when the audit, run over a skill that names a removed command or a stale version, reports each one, tested and seen red
- [ ] Fold what this plan built, and its out-of-scope lines as `Not doing:`, into the spec, run `ele plan done .documentation/plans/ele-features.md`, and delete this plan

## Inbox

- 2026-10-21, from the user's config candidates: options beyond those planned, to list with the user once config-structure is done
- 2026-10-21, from the user's config candidates: "Claude heads-up integration", undefined when raised; meaning to be drawn with the user once config-structure is done
- 2026-10-22, from a question raised in a Go upgrade discussion, for recipe-propose: should ele get a scope that spans owners, or one keyed to repo traits such as every Go repo, for multi-step recipes like a Go upgrade? As of 2026-10-22 `--global` covers one owner and notes cap at 3 lines and load every session, so such recipes go in `~/.claude/conventions`; a user skill already spans owners
- 2026-10-22, from the checkpoint review, for reminders: whether reminders count against the session-start block's budget, where a full block drops the oldest notes
- 2026-10-22, from the user: onboarding for new users, a first-run flow that finds a missing owner or unowned directory and sets it up, skill-first (`/ele:setup`) with a scriptable CLI check, no TUI dependency; triage after config-structure

## Out of scope
