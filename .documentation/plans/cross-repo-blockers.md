---
kind: plan
created: 2026-11-12
spec: .documentation/specs/design.md
---

# Cross-repository blockers

## Why

A plan item that waits on work in another repository can only say so in
prose today: ele reads plans of the current project alone, so the wait holds
the item until someone notices the other item is done and removes the line by
hand. This plan lets a `blocked:` line name an item of a plan in another
repository, and has ele find that plan's state and free the item once the
other item is ticked.

## Requirements

1. A plan in one repository can wait on an item in another repository's plan,
   and ele frees the item once that item is ticked
2. A blocker ticked while its plan is still active counts, not only once that
   plan is closed with `ele plan done`
3. Only plans this machine's ele data directory started are reachable
4. A plan under one owner can be blocked on a plan under another owner
5. An unreachable plan (checkout moved, until a session in its new place has
   synced, or deleted; plan never started) holds the item, and `ele plan show`
   and `ele plan check` flag it with a reason

## Context

- As of 2026-11-12: `internal/plan` `edges` reads a `blocked:` head before its
  `;` as names matching `edgeName`, `item` or `plan/item` in lowercase kebab;
  one entry failing turns the line into prose, and a comma in an unparsed head
  is a malformed-list warning. `continues` refuses to join a wrapped line
  holding a colon
- As of 2026-11-12: `plan.Plan.Resolve` sends an edge naming a plan to a
  `Lookup` callback answering `BlockerMissing`, `BlockerNoPlan`,
  `BlockerAbandoned`, `BlockerDone` or `BlockerOpen`; only edges naming no plan
  enter the cycle check
- As of 2026-11-12: `cmd/ele` `loadPlanSet` and `planSet.lookup` read the
  current project's active plans from their files under this checkout's root
  and its closed plans from `plan.ReadStored` records in `<datadir>/plans`,
  matching a plan by file name without `.md`. The session-start and stop hooks
  call `loadPlanSet` too
- As of 2026-11-12: a plan note stores owner, project and the plan's path
  relative to its git root, no checkout path or remote; a closed plan's
  stored record holds its item names, ticks and path, no project or owner
- As of 2026-11-12: a project is `filepath.Base` of its git root
  (`internal/index` `Projects.Resolve`); notes sit under
  `<datadir>/notes/<owner>/`, and `projectNotes` reads one owner's folder
- As of 2026-11-12: the index records each synced session's working directory
  as the session gave it, which may sit below its git root, with its project
  in the `directories` table (`internal/index` `entryWriter.recordDirectory`);
  `cmd/ele` `indexedDirectories` reads it, returning nothing with no index and
  refusing one with no completed sync at its version
- As of 2026-11-12: `.documentation/specs/design.md`'s Plans section,
  `docs/plans.md` and `plugin/skills/plan/SKILL.md` say a plan in another
  repository cannot block and that its wait is written as prose with no comma
  before its `;`
- As of 2026-11-13: no active plan holds a same-project `<plan>/<item>` edge
  since v1-release closed; the new form must keep resolving one, which
  `internal/plan`'s `TestResolveFreesABlockedItemOnceItsBlockerIsTicked`
  carries, not a live plan line
- Not checked: whether the test fixtures can run a sync over a second
  checkout's transcript, or need to seed the `directories` table through
  `internal/index`

## Decisions

- A blocker in another repository is `<project>/<plan>/<item>`, `<project>`
  being that checkout's git root folder name, the identity ele keys notes,
  recall and `plan list --project` on. The project segment takes a folder
  name's characters, `[A-Za-z0-9._-]`, while the plan and item segments stay
  lowercase kebab; a project whose folder name holds any other character
  cannot be named, and its wait stays prose. The plan segment stays, since
  `<project>/<item>` cannot be told from `<plan>/<item>`. ele reads the plan
  notes of every owner for that project; finds an active plan's checkout among
  the index's session directories, each taken to its git root and kept only
  where it still exists, the owner map gives it the plan note's owner and a
  file sits at the plan note's path, reading the plan file there so a tick
  counts once saved; and reads a closed plan from its stored record. The other
  checkout's working tree is the truth: a tick on a branch not checked out
  there does not count. Nothing new is stored, and plans started before the
  change resolve once a session in their checkout has synced. A blocker it
  cannot resolve holds the item under a new unreachable state, its flag naming
  the reason: no completed sync, no listed checkout still there, checked out
  in two places, plan file missing or unreadable, a plan of that name started
  under two owners, or no plan of that name started
  - rejected: naming the repository by its normalised git remote, with a
    per-plan record of root, ticks and blockers refreshed on each read,
    because its lines run long, a repository with no remote or a renamed one
    cannot be named, plans closed before the change stay unreachable, hooks and
    read commands write to the data directory and run extra git calls, and it
    would give ele a second project identity beside the folder name its notes
    use
- A three-part blocker naming the current project resolves as
  `<plan>/<item>` does, so writing the full form for one's own project works
- A `blocked:` line naming another project is the ask that lets ele cross
  owners under the spec's "nothing crosses owners unless asked": it reads that
  project's plan notes and plan file whatever their owner, and prints only the
  blocker's resolved state and flag, never another owner's notes
- The session-start and stop hooks resolve blockers in other repositories, as
  `plan show` and `check` do, so the blocked counts agree across them; the
  index is opened only when a plan holds such a blocker. The start block then
  carries another owner's blocker state, the one exception to "the
  SessionStart block never crosses owners"
- The same-project `BlockerNoPlan` hint names the three-part form in place of
  writing the wait as prose

## Actions

- [ ] foreign-blockers: `blocked:` names an item of another repository's plan
      as `<project>/<plan>/<item>`, resolved, held and flagged as the
      Decisions say, in `plan show`, `plan check` and both hooks; the Plans,
      Owners and Principles sections of `.documentation/specs/design.md`,
      `docs/plans.md`, and the `plan` and `plan-audit` skills describe it in
      the same change. Where a fixture cannot sync a second checkout's
      transcript, it seeds the `directories` table through `internal/index`
  - done when, in a fixture of two repositories under two owners, `ele plan
    show` in one holds an item blocked on an open item of the other, frees it
    once that item is ticked in the other checkout while its plan is active,
    and keeps it free after `ele plan done` there and that plan file's
    deletion; `ele plan show` and `ele plan check` both flag, each naming its
    reason, a blocker whose checkout is removed and one naming a plan whose
    file is in the other checkout but never started, and one naming a plan
    started under two owners; the session-start block's blocked count matches
    `show`, and the stop hook's next item skips the held item; a three-part
    blocker naming its own project resolves as `<plan>/<item>`; the
    same-project `BlockerNoPlan` flag names the three-part form; and
    `.documentation/specs/design.md`, `docs/plans.md`,
    `plugin/skills/plan/SKILL.md` and `plugin/skills/plan-audit/SKILL.md`
    describe it
- [ ] Fold each Decision through
      `/ele:plan-audit close .documentation/plans/cross-repo-blockers.md`,
      then what this plan built and its out-of-scope lines as `Not doing:`
      into the spec, run
      `ele plan done .documentation/plans/cross-repo-blockers.md`, and delete
      this plan

## Inbox

- 2026-11-13, from the v1-release close checkpoint's re-review: `internal/plan`'s
  resolve tests inject the lookup, so no test covers the real lookup scoping a
  same-project `<plan>/<item>` edge, which `foreign-blockers` changes

## Out of scope

- Detecting cycles that run across repositories: `ele plan check` would load
  every repository's plans on each check; it keeps its in-repository cycle
  check (ruled 2026-11-12, from the plan's question rounds)
- Converting waits already written as prose in other repositories' plans: each
  is converted when its plan is next picked up (ruled 2026-11-12, from the
  plan's question rounds)
