---
kind: plan
created: 2026-10-22
spec: .documentation/specs/design.md
---

# ele configuration, reminders and recipes

## Why

Give ele three user-facing features it lacks: a configuration a user can switch and tune from one place, reminders that surface at session start from a date or every N days, and recipes that turn a reusable procedure into a user skill and keep skills from going stale. Configuration comes first so the other two can be switched through it.

## Context

- As of 2026-10-22: `plugin/.claude-plugin/plugin.json`'s `userConfig` holds one option, `checkpoint_gate` (boolean, default on), read by the checkpoint hooks through `CLAUDE_PLUGIN_OPTION_CHECKPOINT_GATE`
- As of 2026-10-22: note kinds are decision, fact, handoff, plan and question (`Kinds` in `internal/notes`); a dated reminder has been written as a fact note, which shows at every session start from the day it is written, not from its date
- As of 2026-10-22: `~/.claude/hooks/measure-reminder.sh`, a SessionStart hook in claude-config, prints a notice when `~/.claude/cache/hook-stats.last` is missing or over 7 days old
- As of 2026-10-22: notes hold at most 3 lines and show every session, so a multi-step procedure has no home in ele

## Decisions

- One plan for all three, decided 2026-10-22 by the user: two plans are active and the plan audit flags more than three
- Configuration's structure is settled before any option is added, decided 2026-10-22 by the user, so reminders, nudges and auto note writes each land as an option in one layout
- A reminder shows from its date until closed; a nudge is a recurring reminder that comes back N days after it was last marked done, decided 2026-10-22 by the user
- ele:recipe gets both halves, decided 2026-10-22 by the user: proposing a recipe as a user skill, with the Write/Skip approval notes use, and auditing skills for stale commands and versions

## Actions

- [ ] config-structure: Settle with the user, in rounds, what of ele is configurable, where configuration lives (plugin `userConfig`, an ele config file, or both), and how it is read and set from the CLI, the skills and the guide; then build that layout with `checkpoint_gate` moved onto it
  - done when design.md holds a Configuration section naming the layout and how each surface reads and sets it, `checkpoint_gate` reads through it with a test seen red, and `task check FORCE=1` passes
- [ ] auto-note-writes: An option that, when on, has ele write proposed notes without the Write/Skip ask
  - blocked: config-structure
  - done when switching the option changes whether a proposed note is written without asking, each state tested and seen red
- [ ] reminders: Dated reminders: saved through the CLI and a skill with a date, shown in the session-start block from that date until closed, and listed and closed through the CLI
  - blocked: config-structure
  - done when a reminder dated tomorrow does not show at session start, one dated today does, a closed one does not, each tested and seen red, and the CLI lists and closes them
- [ ] nudges: Recurring reminders that come back N days after they were last marked done, with `measure-reminder.sh`'s hook-stats notice moved onto one
  - blocked: reminders
  - done when a nudge marked done does not show until N days later, tested and seen red, and the hook-stats notice comes from ele, with removing `measure-reminder.sh` from claude-config handed to the user as commands
- [ ] recipe-propose: ele:recipe proposes a reusable procedure seen in a session as a user skill, with the same Write/Skip approval as notes, and writes the approved one under the user's skills
  - blocked: config-structure
  - done when a session that ran a multi-step procedure can have ele:recipe propose it, and an approved recipe is a skill Claude Code lists
- [ ] recipe-audit: Audit the user's skills for commands, flags and versions that no longer exist, as plan-audit does for plans
  - blocked: recipe-propose
  - done when the audit, run over a skill that names a removed command or a stale version, reports each one
- [ ] Fold what this plan built, and its out-of-scope lines as `Not doing:`, into the spec, run `ele plan done .documentation/plans/ele-features.md`, and delete this plan

## Inbox

- 2026-10-21, from the user, among the config candidates: other config candidates
- 2026-10-21, from the user, among the config candidates: Claude heads-up integration; meaning to be drawn with the user once config-structure is done
- 2026-10-22, from a question raised in a Go upgrade discussion, for recipe-propose: should ele get a scope that spans owners, or one keyed to repo traits such as every Go repo, for multi-step recipes like a Go upgrade? As of 2026-10-22 `--global` covers one owner and notes cap at 3 lines and load every session, so such recipes go in `~/.claude/conventions`; a user skill already spans owners

## Out of scope
