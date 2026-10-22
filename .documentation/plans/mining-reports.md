---
kind: plan
created: 2026-10-21
spec: .documentation/specs/design.md
---

# Mining reports in ele

## Why

ele's mining reports exist only as one-off Python scripts and run data on one machine. Port what is worth reusing to Go features that run over any user's own transcripts and judgments, then delete the scripts and the private run data, so the shared binary and specs point at nothing outside a user's own data.

## Context

- As of 2026-10-22: ele's data directory holds `mining/2026-10-15/` (the 2026-10-15 scripts, prompts, data and reports), `mining/2026-10-16-hook-blocks/` and `mining/2026-10-16-hook-blocks-claude-d7/` (the 2026-10-16 Stop-hook judging run) and `mining/cache/`
- As of 2026-10-22: `mining/cache/checkpoint-runs.jsonl` holds 586 judgments over 517 units, pre-log runs included
- As of 2026-10-22: specs naming the private runs are `design.md`'s Storage and Rollups for mining sections and `checkpoint-and-ticket-audit.md`'s Judging Stop-hook blocks line; the cleanup's grep finds any other

## Decisions

- Scripts and run data stay out of the repository, decided 2026-10-21 by the user: the data holds transcript excerpts, and the scripts are one-off analyses written against its layout, Python in a Go repository
- No per-user baseline, decided 2026-10-21: a user installing now logs every checkpoint from the first, so a mined baseline would duplicate their log. Report figures read `ele mine`'s cache, which already holds this user's pre-log runs
- The 2026-10-15 and 2026-10-16 run data is deleted only once report figures reproduce the 2026-10-15 reports' numbers and the hook-block follow-up has shipped, decided 2026-10-21: until then both are the reference the ports are checked against. `mining/cache/` stays
- `ele import mining` goes in the cleanup, decided 2026-10-21: its only source is the 2026-10-15 folder, whose classifications are already in the cache

## Actions

- [x] report-figures: Report figures from stored mining judgments: findings by source and verdict, re-review kinds, refuted, user-reported and mutations, caught after review and older than the change by category, and the mode split; plus reviewer and gate timing, read from the archived transcripts in each judged unit's window, since no judgment holds a duration; replaces 2026-10-15's `agg2.py`, `agg_inl.py`, `gatetime.py` and `subtools.py`, kept under `mining/2026-10-15/` in ele's data directory
  - captured 2026-10-21, the user's call when the mining scripts were compared before deletion: as of 2026-10-21 `mining.Load` is read only to list pending units, in `cmd/ele/mine.go`, so nothing reports on judgments
  - moved 2026-10-21 from `move-checkpoint-into-ele.md`
  - done when its figures match the 2026-10-15 reports' numbers over the same units, differences named
  - as of 2026-10-21: no judgment carries a duration; 130 of 586 carry a non-null `report_ts`, the time the checkpoint reported; `gatetime.py` timed Bash calls and `subtools.py` reviewer subagents from the transcripts in each run's window
  - as of 2026-10-21: the cache keeps every pass per unit; of 141 subagent-era runs, 60 hold two or more judgments. 2026-10-15's figures took the `classify-sub3.md` pass over `classify-sub.md` and dropped the review cycles a run's window covered
  - as of 2026-10-22: judgment figures built as `ele mine --report`, the last stored judgment per unit; over the 2026-10-15 passes alone the figures `agg2.py` and `agg_inl.py` printed match, except the differences named in design.md's Built: figures line
  - asserted in the report-figures commit: `ele mine --report` over the 2026-10-15 passes alone matches the figures `agg2.py` and `agg_inl.py` printed, except the differences named in design.md's Built: figures line; its timing, run over each run's 2026-10-15 window with the scripts' own patterns by a throwaway check on 2026-10-22, reproduced `gatetime.py` on all 102 runs within 0.01 s and every `subtools.py` row by role and round; over the whole cache it times all 397 judged subagent-era units. `task check FORCE=1` and `task audit` green
- [x] drop-baseline: Remove `ele checkpoint stats --baseline` with `BaselinePath`, `WriteBaseline`, `ErrNoBaseline`, `CHECKPOINT_BASELINE` and their tests, `/ele:checkpoint-stats`' mention of it, and the spec's "Mining baseline" line and "mined baseline" in its Data stays line, once report figures cover what the baseline reported
  - done when each section `ele checkpoint stats --baseline` printed has a report-figures counterpart or is named as dropped under this item, a grep finds none of the named symbols, and `task check FORCE=1` passes
  - asserted in the drop-baseline commit: every section `--baseline` printed has its counterpart in `ele mine --report`'s subagent era, which runs the same `checkpoint.Report` over the same judgments: mode and change size, scope, who found what, severity, re-reviews, caught after review, older than the change, mutations, notes, duration, and the header's count and confidence split. Dropped: the header's date range and unreadable-line count, and the baseline's leaving out the 3 runs the log also holds. `git grep` outside `.documentation/plans/` finds none of the named symbols, `ele checkpoint stats --baseline` now exits 2 (tested, seen red), and `task check FORCE=1` passes
- [ ] block-follow-up: in ele, each Stop-hook "no checkpoint was reported" block carries its `Changed:` paths and the next main-thread reply with its time, and `ele show --at <time>` opens a session at an event's turn, as `checkpoint-and-ticket-audit.md`'s Judging Stop-hook blocks line says; beyond that line, whether the reply disputes the block and whether the checkpoint skill was then invoked, as 2026-10-15's `blocks.py` read them; that line then names ele's reading of the reply in place of `~/.claude/scripts/hook-firings.py`, its disputed figure re-measured by it
  - done when `ele events --kind hook-block` shows each block's `Changed:` paths and the next main-thread reply with its time, `ele show --at <time>` opens a session at that turn, each has a test seen red, and the spec's Judging Stop-hook blocks line names ele's reading with its re-measured figure
  - captured 2026-10-21, the user's call when the mining scripts were compared before deletion: as of 2026-10-21 `ele events --kind hook-block` gives only the block and its reason (`EventHookBlock` in `internal/transcript`), and `~/.claude/scripts/hook-firings.py` reads the reply after a block. The 2026-10-16 judging run's scripts are kept with it under `mining/2026-10-16-hook-blocks/` in ele's data directory
  - moved 2026-10-21 from `move-checkpoint-into-ele.md`
  - reordered 2026-10-22 by a plan audit, the user's call: built before judge-blocks, which reads blocks through it, since one-off scripts are ruled out
- [ ] judge-blocks: Judge the week of Stop-hook "no checkpoint was reported" blocks after claude-config's 9fbce04, 2026-10-15 19:28Z to 2026-10-22 19:28Z, once that week has passed: sort each as hook wrong, agent dodged or unclear, as the 2026-10-16 run did, reading each block through block-follow-up's fields, in its own run folder under `mining/` in ele's data directory; what they show scopes `move-checkpoint-into-ele/port-hooks`
  - done when every block in the week is sorted as hook wrong, agent dodged or unclear in its run folder, with the counts per class, and port-hooks carries a dated line saying what the judging changes for it
  - after: 2026-10-22 19:28Z
  - blocked: block-follow-up
  - as of 2026-10-21: 9fbce04 landed 2026-10-15 19:28Z, per `git -C ~/.claude log -1 9fbce04`
  - as of 2026-10-22: the hook changed inside the week (claude-config 5cd0ce4 and 120ad6a; 3361135 to cab488d on 2026-10-17; the move into ele's plugin on 2026-10-20), so each block is judged against the hook of its time; `ele events --since` takes a date only, so the 19:28Z bounds filter on timestamp; 47 checkpoint blocks after 9fbce04 in the unsynced index at 2026-10-22 09:30Z
- [ ] judge-notes-asks: Judge a week of stop-check notes asks: per Stop hook-block event, the trigger rebuilt from the tool calls before it (Bash, question, subagent edit), notes written after it in that session, asks repeated without new work, and sessions with questions or subagent edits that got no ask
  - moved 2026-10-22 from this plan's inbox by a plan audit; first captured 2026-10-20 from the install of 8596ec2, and moved 2026-10-21 from `skill-lessons.md` when it closed
  - done when each ask in the week is listed with its trigger, the notes written after it, and whether it repeated without new work, in its own run folder under `mining/`, with the sessions that got no ask counted, and question note 20261021-93f75e is answered from it
  - after: 2026-10-27
- [ ] delete-mining-runs: Delete `mining/2026-10-15/`, `mining/2026-10-16-hook-blocks*/`, and judge-blocks' and judge-notes-asks' run folders from ele's data directory, remove `ele import mining`, and drop the specs' pointers to them; verify with `ls` that the folders are gone and with `git grep` that nothing outside `.documentation/plans/` names them
  - done when `ls` shows the folders gone, `ele import mining` is removed with its tests, a grep finds no name of them outside `.documentation/plans/`, and `task check FORCE=1` passes
  - blocked: judge-blocks, judge-notes-asks
  - as of 2026-10-22: `mining/2026-10-17-checkpoint-baseline-405865fa/` deleted by a plan audit, the user's call, nothing reading it since d8cf6ba; judge-notes-asks' folder added the same day, the user's call
- [x] unit-boundaries: Cut mining units where checkpoints actually start and report: anchor `checkpointCommand` in `internal/mining` to `checkpoint</command-name>`, so a typed `/ele:checkpoint-stats` no longer closes a run's window; tell a checkpoint's report from progress text by more than "bugs reviewer" plus a commit or gate over 800 characters (`reportCut`), which misses small-mode and newer reports; and whether two fresh reviewer launches under 5 minutes apart split a cycle, as re-reviews do, counting the runs affected first
  - moved 2026-10-22 from this plan's inbox by a plan audit; first captured from question notes 20261020-ead7f7, 20261016-4c7d8f and 20261016-628f80
  - done when windows anchor on the command tag, the report check catches small-mode and current reports, the split-cycle count is measured and acted on or ruled out, each change has a test seen red, and `ele mine --report` is re-run with the figures it changes named
  - asserted in the unit-boundaries commit: digest tests seen red for the exact checkpoint command and skill names, the record-turn report cut (all three record forms, a named-but-not-run script, a label while a reviewer is out) and the run-split cycle; on real data as of 2026-10-22, runs with no report 46→8 of 187, cycles 100→22 of 166, 2 cycles split, `ele mine --report` re-run with its changed figures in design.md
- [ ] Fold what this plan built, its lasting Decisions (scripts and run data stay out of the repository; no per-user baseline) and its out-of-scope lines as `Not doing:` into the spec, and the hook-block changes into `checkpoint-and-ticket-audit.md`, its Open questions' Judging Stop-hook blocks entry and the line naming this plan's items included, run `ele plan done .documentation/plans/mining-reports.md`, and delete this plan
  - blocked: delete-mining-runs

## Inbox

## Out of scope
