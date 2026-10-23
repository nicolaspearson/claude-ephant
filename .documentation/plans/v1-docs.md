---
kind: plan
created: 2026-10-23
---

# v1 docs

## Why

v1 is for users other than the author. The code still indexed a one-off claude-mem export no other user has, and the docs read as the author's build log: dated measurements, a cutover, a build order and the tools removed along the way. A user needs to know what ele does and how to use it, a contributor how to build and release it, and neither needs the history, which git keeps.

## Context

- As of 2026-10-23: outside `.git`, `claude-mem`, `headroom` and `rtk` appeared in 17 files before `drop-claude-mem`; after it, in `.documentation/specs/design.md` (history and cutover sections) and in one `## Context` line of `.documentation/plans/ele-features.md` citing claude-mem's `ContextBudget.ts`
- As of 2026-10-23: `README.md` holds install, update, options and a Contributing section; there is no `docs/`, `CONTRIBUTING.md` or `CLAUDE.md`
- As of 2026-10-23: `.documentation/specs/` holds `design.md`, `stop-check.md` and `checkpoint-and-ticket-audit.md`

## Decisions

- claude-mem indexing is removed, not renamed: no v1 user has the export. A data directory's `archive/claude-mem/` stays on disk untouched; the index version bump drops its rows at the next sync
- User docs are a short README linking to `docs/`, one page per feature plus a command reference: a README holding everything is too long to scan, and a page per feature is what a user lands on from a search
- Specs describe the current design only: behaviour, constraints and the reason for each decision, with no measurements, timelines or removed tools. Git history keeps the old text, so nothing moves to a history file
- `CONTRIBUTING.md` takes the README's Contributing section and the build, release and tap material from the specs; `CLAUDE.md` holds what an agent working in the repository needs and points to `CONTRIBUTING.md` for the rest
- A docs item is done when the page exists, every relative link in it resolves, `task lint-docs` passes, and each command or flag it names is checked against `ele <command> --help`

## Actions

- [x] drop-claude-mem: Remove claude-mem indexing from the index, sync, status, recall, stats and usage, the recall skill and the tests; bump the index version
  - done when `grep -rniE 'claude-mem|headroom|\brtk\b'` outside `.documentation/` finds nothing, `task check` passes, and `ele sync` on an index holding claude-mem rows rebuilds it and `ele stats` shows no claude-mem source
  - asserted in the drop-claude-mem commit: the grep finds nothing outside `.documentation/`; `task check FORCE=1` passes; a copy of the live v10 index, holding 130,175 claude-mem entries, synced by this build against an empty archive was rebuilt ("built by another version of ele"), and `ele stats` then listed no claude-mem source
- [ ] specs-current: Rewrite `design.md`, `stop-check.md` and `checkpoint-and-ticket-audit.md` as the current design: cut the cutover, build order, testing phase, measurements, mining figures and Removed sections, and every claude-mem, headroom and rtk line; move build, release and tap material out for `contributing`
  - done when the three specs hold no dated measurement or timeline, the grep above finds nothing under `.documentation/specs/`, and every behaviour the old `design.md` stated is either kept or contradicted by the code
- [ ] user-guides: Write `docs/` pages: owners, notes and recall, plans, handoffs and reminders, checkpoint and ticket-audit, data and privacy (storage, backup, redaction, no background model), and a command reference
  - blocked: specs-current
  - done when each page meets the docs done-when decision
- [ ] readme-for-users: Rewrite `README.md` for users: what ele is, install, quick start, the skills at a glance, options, update, and links to `docs/` and `CONTRIBUTING.md`
  - blocked: user-guides
  - done when the README meets the docs done-when decision and holds no contributor material
- [ ] contributing: Write `CONTRIBUTING.md`: clone setup with Task, `task install`, `task dev`, the gates, evals, repository layout, specs and plans, conventional commits, the release and tap workflows and the tap token's renewal
  - blocked: specs-current
  - done when it meets the docs done-when decision and each `task` target it names is listed by `task --list`
- [ ] claude-md: Write `CLAUDE.md` for agents working in the repository: layout, the gates to run, where specs and plans live, and a pointer to `CONTRIBUTING.md`
  - blocked: contributing
  - done when it meets the docs done-when decision and a fresh session in the repository loads it
- [ ] Propose each out-of-scope line as a `decision` through `/ele:remember`, run `ele plan done .documentation/plans/v1-docs.md`, and delete this plan

## Inbox

- 2026-10-23, from drop-claude-mem: `.documentation/plans/ele-features.md` cites claude-mem's `ContextBudget.ts` as prior art for a budget figure in its `## Context`; keep it as a citation, reword it, or leave it to that plan

## Out of scope
