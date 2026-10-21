---
kind: plan
created: 2026-10-21
spec: .documentation/specs/design.md
---

# Mining reports in ele

## Why

ele's mining reports exist only as one-off Python scripts and run data on one machine. Port what is worth reusing to Go features that run over any user's own transcripts and judgments, then delete the scripts and the private run data, so the shared binary and specs point at nothing outside a user's own data.

## Context

- As of 2026-10-21: ele's data directory holds `mining/2026-10-15/` (the 2026-10-15 scripts, prompts, data and reports), `mining/2026-10-16-hook-blocks/` and `mining/2026-10-16-hook-blocks-claude-d7/` (the 2026-10-16 Stop-hook judging run), `mining/2026-10-17-checkpoint-baseline-405865fa/` (the baseline) and `mining/cache/`
- As of 2026-10-21: `BaselinePath` in `internal/checkpoint` hardcodes the baseline folder as `--baseline`'s default path; for any other user the file is missing and `--baseline` returns `ErrNoBaseline`
- As of 2026-10-21: `mining/cache/checkpoint-runs.jsonl` holds 586 judged runs, pre-log ones included; `checkpoint-baseline.jsonl` holds 297 records in the checkpoint-log format
- As of 2026-10-21: specs naming the private runs or the baseline are `design.md`'s Storage and Rollups for mining sections and `checkpoint-and-ticket-audit.md`'s Mining baseline and Judging Stop-hook blocks lines; the cleanup's grep finds any other

## Decisions

- Scripts and run data stay out of the repository, decided 2026-10-21 by the user: the data holds transcript excerpts, and the scripts are one-off analyses written against its layout, Python in a Go repository
- No per-user baseline, decided 2026-10-21: a user installing now logs every checkpoint from the first, so a mined baseline would duplicate their log. Report figures read `ele mine`'s cache, which already holds this user's pre-log runs
- The private run data is deleted only once report figures reproduce the 2026-10-15 reports' numbers and the hook-block follow-up has shipped, decided 2026-10-21: until then both are the reference the ports are checked against. `mining/cache/` stays
- `ele import mining` goes in the cleanup, decided 2026-10-21: its only source is the 2026-10-15 folder, whose classifications are already in the cache

## Actions

- [ ] Report figures from stored mining judgments: findings by source and verdict, re-review kinds, refuted, user-reported and mutations, plus reviewer and gate timing; replaces 2026-10-15's `agg2.py`, `agg_inl.py`, `gatetime.py` and `subtools.py`, kept under `mining/2026-10-15/` in ele's data directory
  - captured 2026-10-21, the user's call when the mining scripts were compared before deletion: as of 2026-10-21 `mining.Load` is read only to list pending units, in `cmd/ele/mine.go`, so nothing reports on judgments
  - moved 2026-10-21 from `move-checkpoint-into-ele.md`
  - done when its figures match the 2026-10-15 reports' numbers over the same units, differences named
- [ ] Remove `ele checkpoint stats --baseline`, its default path in `internal/checkpoint`'s `BaselinePath`, `/ele:checkpoint-stats`' mention of it and the spec's "Mining baseline" line, once report figures cover what the baseline reported
- [ ] Hook-block follow-up in ele: each Stop-hook "no checkpoint was reported" block carries its `Changed:` paths and the next main-thread reply with its time, and `ele show --at <time>` opens a session at an event's turn, as `checkpoint-and-ticket-audit.md`'s Judging Stop-hook blocks line says; beyond that line, whether the reply disputes the block and whether the checkpoint skill was then invoked, as 2026-10-15's `blocks.py` read them
  - blocked: waits until the week of Stop-hook blocks after 9fbce04 is judged, from about 2026-10-22, since that line scopes it by what they show
  - captured 2026-10-21, the user's call when the mining scripts were compared before deletion: as of 2026-10-21 `ele events --kind hook-block` gives only the block and its reason (`EventHookBlock` in `internal/transcript`), and `~/.claude/scripts/hook-firings.py` reads the reply after a block. The 2026-10-16 judging run's scripts are kept with it under `mining/2026-10-16-hook-blocks/` in ele's data directory
  - moved 2026-10-21 from `move-checkpoint-into-ele.md`
- [ ] Delete `mining/2026-10-15/`, `mining/2026-10-16-hook-blocks*/` and `mining/2026-10-17-checkpoint-baseline-*/` from ele's data directory, remove `ele import mining`, and drop the specs' pointers to them; verify with `ls` that the folders are gone and with a grep of the repository that nothing names them
- [ ] Fold what this plan built, and its out-of-scope lines as `Not doing:`, into the spec, and the hook-block and baseline changes into `checkpoint-and-ticket-audit.md`, delete this plan, and run `ele plan done .documentation/plans/mining-reports.md`

## Inbox

## Out of scope
