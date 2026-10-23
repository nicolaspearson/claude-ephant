# Checkpoint and ticket-audit

Moved from claude-ephant's design spec to the agent-skills repository on 2026-10-17, and back here with the code on 2026-10-20. Phase 1 of the redesign from the 2026-10-15 history mining shipped on 2026-10-15 and the rest on 2026-10-17. As of 2026-10-23 every script checkpoint and ticket-audit ran is ported to an `ele checkpoint` or `ele ticket` subcommand.

## Ticket-audit

- Ledger, `~/.claude/logs/ticket-audits.jsonl`: a record, never a scope. One line per item per audit: its key, its text hash, the methods that reached it, the user's rulings on its findings. Nothing in it skips a method or hides a finding; a later audit re-raises a ruled finding with the ruling attached, labelled text unchanged or changed since
  - Key: a ticket's `repo#number`; a plan item's repository, plan path and name, so a reworded item keeps its rulings; an unnamed item's checkbox text, normalised with the marker stripped, so rewording one loses them
  - Hash: a ticket's title, body and comments, since §9 records decisions as notes; a plan item's checkbox line and sub-bullets, marker stripped, so a tick is no change
  - `ele ticket corpus` writes `manifest.json` beside the corpus, key, hash and file per item, and takes `--plan <path>`, repeatable, one file per item, `<plan>-<nn>.md`, refusing two plans of one file name. `ele ticket ledger record` appends at the end of the report, copying the hash from the manifest the audit read, and refuses a key not in it or a method outside §3–§8
  - A ruling on one item's finding goes in the ledger; a decision that would outlive the item is an ele note
  - A ruling that commits the item to an edit not made in this audit is also drafted onto the item, since the builder reads the item, never the ledger: a note on a ticket, a `next:` line in a plan item's body
  - `/ele:ticket-claims` adjudications stay out: "dead" and "still true" are claims about code
- Corpus, `ele ticket corpus`: fetches in parallel; warns on a child holding a literal `Part of #MAP`; names no closed child in its `note:` lines; prints a blocker table that ignores prose after "Blocked by"; compares read-backs without the title line and trailing newline, against the body or with `--note` a comment, never both, so a note quoting a body edit cannot pass for it
- `/ele:ticket-claims` (`ele ticket claims`) checks `execution-map` and `typed-as-task`; `dead-absence` is dropped
- Errors: a table of known errors, each with one fix, and one retry that never repeats the call unchanged, then stop and report; drawn from `ele events --during ticket-audit --kind tool-error`
- Plans: `ele:plan-audit` audits them on ticket-audit's engine, an item standing for a ticket; ticket-audit keeps tracker maps only. The plan audit and the plan format are claude-ephant's design spec, Plans section

## Checkpoint

- Mode is small or full by size; scope is uncommitted, delta or branch
  - Record keeps a copy of each reviewed file in `~/.claude/cache/checkpoint-reviewed/`, never in the repository's `.git`
  - Delta is the freeze's call: whenever a record holds a path on the change list, each path diffs against its reviewed copy, or HEAD where there is none, and the delta's size picks the mode
  - Branch diffs against the merge-base with `@{upstream}`, else `origin/HEAD`, and refuses with neither. Run when asked; offered on a clean tree holding unpushed commits. Writes no delta: no record's copies describe commits
- A run opens with a freeze passing `--start`, or any freeze when none is open, and holds the re-freezes inside it. Each freeze replaces the run's frozen paths, and `--start` resets its clock, so a run abandoned without record or close inside the 120-minute window leaves the next run neither its paths nor its start
- Errors: SKILL.md holds a table of known errors, each with one fix, and one retry that never repeats the call unchanged, then stop and report; the bugs and conventions-prose reviewer files hold the rows reviewers hit. Drawn from `ele events --during checkpoint --kind tool-error`
  - Not doing: a digest of each Bash call's full command, to tell a rerun among commands cut at 300 characters: the tables shipped without it, and as of 2026-10-21, of the 394 failed Bash calls since 2026-10-14 whose next call was known, 1 was rerun unchanged (ruled 2026-10-21, from skill-lessons)
- SKILL.md size, in both skills: no line-count target. SKILL.md keeps what every run needs; a case only some runs meet sits in a file beside it, read when it arises: checkpoint's `BRANCH.md`, `DELTA.md` and `HEAD-EXPORT.md`, ticket-audit's `TRACKERS.md`
- Plans: when a change completes the active ele plan's item, checkpoint proposes its tick with an evidence line saying what was asserted and at which commit; otherwise a dated progress line under the item. Written on approval
- Spec reviewer, `reviewers/spec.md` as `ele:checkpoint-reviewer`, only when the brief quotes a plan item or fetched ticket: a third parallel reviewer in full mode, a file added to the single reviewer's list in small mode. It reports what the item asked that is missing or partial, what is done wrong, and what the diff does unasked, each quoting the item's words, in its own section, unranked against the others. The brief says which requirements the change sets out to deliver, and an Observed field names what was run or measured toward the item, since the diff cannot show it. Its findings go through Verify: a missing requirement outside what the change sets out to deliver is logged `deliberate` and left to the plan's `next:` line, a confirmed unasked change goes to the user rather than being reverted, and any confirmed spec finding left unfixed turns the plan tick into a progress line. A missing requirement falls on no line, so delta scope does not drop it. The re-review gets the quoted item, what the change sets out to deliver and the Observed field
- Log fields: `mode`, `scope`, `severity` per finding (blocker, defect, quality, nit), `also_raised_by` per finding, so a merged duplicate still counts its reviewers, and `duration` per run, in seconds from the first freeze to the log line. `/ele:checkpoint-stats` reports each
- Cost, decided 2026-10-16 and measured with ele's token accounting. Reviews stay in subagents, since inline they would grow the parent's context for every later turn, but cheaper ones:
  - Prose and conventions are one reviewer, reporting under both labels so the Stop hook and the stats sources are unchanged
  - An `ele:checkpoint-reviewer` agent type on sonnet, without the Skill tool, runs it and the re-review; the bugs reviewer and the small-mode reviewer stay on the session's model. Replayed on the 04:22Z checkpoint's frozen diff of 2026-10-16, it found both conventions findings and 5 of 7 prose findings, missing a cut-off sentence and a grammar nit, in 74k tokens. An agent type without the Skill tool receives no skill list, tested 2026-10-16
  - Small mode up to 40 lines, 100 for Markdown or text, raised 2026-10-16 from 10 and 30. As of 2026-10-16 the log holds 3 small runs, so the bar is a judgment, not a measurement: re-read it once more small runs are logged

## Into ele

Decided 2026-10-17: checkpoint and ticket-audit move into ele and are ported to Go. ele already depends on them both ways, and split across repositories that coupling is untested. Done 2026-10-23.

- Why: `claude plugin eval` needs a plugin; hooks, scripts, skills and agent type ship and version together; one language. Shareable: the plugin needs no file outside itself and the ele binary, and names its prerequisites
- Prerequisites: in the [README](../../README.md). ele on `PATH` is a hard one, since the plugin's hooks call it
- The checkpoint gate, the Stop hook's block and the prompt hook's snapshot, is ele's option `checkpoint_gate`, on by default; with it `false` both hooks only clear the session's baseline, so turning it back on starts from the next prompt, and a `config.json` that does not parse leaves the gate on. Claude Code cannot turn off one hook of a plugin
- The convention files in the brief are the user's `CLAUDE.md`, under `$CLAUDE_CONFIG_DIR` when set, else `~/.claude`, with what it imports or says to read, and the repository's; the prose reviewer's per-comment tests apply where those hold no comment rules
- In the plugin: the `checkpoint` and `ticket-audit` skills with their reviewer files, the `checkpoint-reviewer` agent, addressed as `ele:checkpoint-reviewer`, and `/ele:checkpoint-stats` and `/ele:ticket-claims` in `commands/`
- Every script is a grouped subcommand, as `plan`, `owner` and `handoff` are: `ele checkpoint base|check|close|density|freeze|hash|log|optout|record|stats`, `ele checkpoint gate` (Stop) and `ele checkpoint snapshot` (UserPromptSubmit), and `ele ticket claims|corpus|ledger`. Each was run beside the script it replaced on the same inputs before the script was deleted, every difference on purpose listed in its plan item
- A checkpoint run's markers, open, frozen paths and start time, are per session and repository, so two sessions checkpointing one repository neither close nor time each other's runs; the record of reviewed bytes stays per repository, since identical bytes were reviewed whichever session asks. Only `freeze --start` restarts a run's clock, and `log` ends it: before 2026-10-23 a freeze after `record`, or another session's freeze, reset it, and runs logged durations of seconds
- The Stop hook's block keeps its wording, "no checkpoint was reported" word for word since `hook-firings.py` and mining match it, but names `ele:checkpoint` and `ele checkpoint optout`. ele writes its hook-log lines itself, in claude-config's path and format, so the plugin depends on no file of claude-config's
- Stay in claude-config: `/hook-stats`, `hook-firings.py` and the guards' `hook-log.sh`, which cover every hook
- Data stays at its paths: run log, audit ledger, reviewed-file cache
- Contract kept: log formats, and the `reviewers/<name>.md` layout ele's mining reads. Mining matches runs by the old bare names as well as the `ele:` ones, since transcripts from before the move keep the old names; the Stop gate and stop-check accept the bare `checkpoint` name too, for a session resumed from before the move
- `claude plugin eval` runs against the plugin: `task eval` runs the cases in `plugin/evals/`, as of 2026-10-23 recall's and remember's, none yet checkpoint's or ticket-audit's

## Open questions

This one is mining-reports' plan items `block-follow-up` and `judge-blocks`:

- Judging Stop-hook blocks: `ele events --kind hook-block` reads each block's `Changed:` paths, the next main-thread reply with its time, whether a checkpoint then ran and whether the reply disputes the block, and `ele show <session> --at <time>` opens the session at its turn. As of 2026-10-22, by that reading, 56 of 95 checkpoint blocks in the week before claude-config's 9fbce04 (2026-10-08 19:28Z to 2026-10-15 19:28Z) were disputed and 23 followed by a checkpoint; in the week after, to 2026-10-22 11:08Z, 12 of 47 were disputed and 28 followed by a checkpoint. The 2026-10-16 count of 52 of 89 disputed used `hook-firings.py`'s pattern over a window starting before that run's 7 days; that pattern over these replies' first 300 characters disputes 53 of the 95. The read-only sort ran 2026-10-16 over 90 blocks across both profiles: 67 hook wrong, 21 agent dodged, 2 unclear, all before 9fbce04; findings in `mining/2026-10-16-hook-blocks/` in ele's data directory
