# Checkpoint and ticket-audit

Moved from claude-ephant's design spec to the agent-skills repository on 2026-10-17, and back here with the code on 2026-10-20. Phase 1 of the redesign from the 2026-10-15 history mining shipped on 2026-10-15 and the rest on 2026-10-17. Scripts named below live in `plugin/premigration/` until each is ported to an `ele` subcommand.

## Ticket-audit

- Ledger, `~/.claude/logs/ticket-audits.jsonl`: a record, never a scope. One line per item per audit: its key, its text hash, the methods that reached it, the user's rulings on its findings. Nothing in it skips a method or hides a finding; a later audit re-raises a ruled finding with the ruling attached, labelled text unchanged or changed since
  - Key: a ticket's `repo#number`; a plan item's repository, plan path and checkbox text, normalised with the marker stripped, so a reworded item is a new item with no ruling
  - Hash: a ticket's title, body and comments, since §9 records decisions as notes; a plan item's checkbox line and sub-bullets, marker stripped, so a tick is no change
  - `ele ticket corpus` writes `manifest.json` beside the corpus, key, hash and file per item, and takes `--plan <path>`, one file per item. `ele ticket ledger record` appends at the end of the report, copying the hash from the manifest the audit read, and refuses a key not in it or a method outside §3–§8
  - A ruling on one item's finding goes in the ledger; a decision that would outlive the item is an ele note
  - A ruling that commits the item to an edit not made in this audit is also drafted onto the item, since the builder reads the item, never the ledger: a note on a ticket, a `next:` line in a plan item's body
  - `/ele:ticket-claims` adjudications stay out: "dead" and "still true" are claims about code
- Corpus, `ele ticket corpus`: fetches in parallel; warns on a child holding a literal `Part of #MAP`; names no closed child in its `note:` lines; prints a blocker table that ignores prose after "Blocked by"; compares read-backs without the title line and trailing newline, against the body or with `--note` a comment, never both, so a note quoting a body edit cannot pass for it
- `/ele:ticket-claims` (`ele ticket claims`) checks `execution-map` and `typed-as-task`; `dead-absence` is dropped
- Errors: a table of known errors, each with one fix, and one retry that never repeats the call unchanged, then stop and report; drawn from `ele events --during ticket-audit --kind tool-error`
- Plans: ticket-audit takes an ele plan as its corpus, an item standing for a ticket, for the whole-plan audit: contradictions between items, artefacts with no producer or consumer. Run at a plan's conversion and when asked. The plan format is claude-ephant's design spec, Plans section

## Checkpoint

- Mode is small or full by size; scope is uncommitted, delta or branch
  - Record keeps a copy of each reviewed file in `~/.claude/cache/checkpoint-reviewed/`, never in the repository's `.git`
  - Delta is the freeze's call: whenever a record holds a path on the change list, each path diffs against its reviewed copy, or HEAD where there is none, and the delta's size picks the mode
  - Branch diffs against the merge-base with `@{upstream}`, else `origin/HEAD`, and refuses with neither. Run when asked; offered on a clean tree holding unpushed commits. Writes no delta: no record's copies describe commits
- A run opens with a freeze passing `--start`, or any freeze when none is open, and holds the re-freezes inside it. Each freeze replaces the run's frozen paths, and `--start` resets its clock, so a run abandoned without record or close inside the 120-minute window leaves the next run neither its paths nor its start
- Errors: SKILL.md holds a table of known errors, each with one fix, and one retry that never repeats the call unchanged, then stop and report; the bugs and conventions-prose reviewer files hold the rows reviewers hit. Drawn from `ele events --during checkpoint --kind tool-error`
- Plans: when a change completes the active ele plan's item, checkpoint proposes its tick with an evidence line saying what was asserted and at which commit; otherwise a dated progress line under the item. Written on approval
- Spec reviewer, `reviewers/spec.md` as `ele:checkpoint-reviewer`, only when the brief quotes a plan item or fetched ticket: a third parallel reviewer in full mode, a file added to the single reviewer's list in small mode. It reports what the item asked that is missing or partial, what is done wrong, and what the diff does unasked, each quoting the item's words, in its own section, unranked against the others. The brief says which requirements the change sets out to deliver, and an Observed field names what was run or measured toward the item, since the diff cannot show it. Its findings go through Verify: a missing requirement outside what the change sets out to deliver is logged `deliberate` and left to the plan's `next:` line, a confirmed unasked change goes to the user rather than being reverted, and any confirmed spec finding left unfixed turns the plan tick into a progress line. A missing requirement falls on no line, so delta scope does not drop it. The re-review gets the quoted item, what the change sets out to deliver and the Observed field
- Log fields: `mode`, `scope`, `severity` per finding (blocker, defect, quality, nit), `also_raised_by` per finding, so a merged duplicate still counts its reviewers, and `duration` per run, in seconds from the first freeze to the log line. `/ele:checkpoint-stats` reports each
- Cost, decided 2026-10-16 and measured with ele's token accounting. Reviews stay in subagents, since inline they would grow the parent's context for every later turn, but cheaper ones:
  - Prose and conventions are one reviewer, reporting under both labels so the Stop hook and the stats sources are unchanged
  - An `ele:checkpoint-reviewer` agent type on sonnet, without the Skill tool, runs it and the re-review; the bugs reviewer and the small-mode reviewer stay on the session's model. Replayed on the 04:22Z checkpoint's frozen diff of 2026-10-16, it found both conventions findings and 5 of 7 prose findings, missing a cut-off sentence and a grammar nit, in 74k tokens. An agent type without the Skill tool receives no skill list, tested 2026-10-16
  - Small mode up to 40 lines, 100 for Markdown or text, raised 2026-10-16 from 10 and 30. As of 2026-10-16 the log holds 3 small runs, so the bar is a judgment, not a measurement: re-read it once more small runs are logged
- Mining baseline: `/ele:checkpoint-stats --baseline` (`ele checkpoint stats --baseline`) reports the runs mined from before the log as a second report, never merged into the log's or windowed. The file, `checkpoint-baseline.jsonl` in ele's data directory under `mining/`, is the 300-record backfill less the 3 runs the log also holds; both stay outside git for their excerpts

## Into ele

Decided 2026-10-17: checkpoint and ticket-audit move into ele and are ported to Go. ele already depends on them both ways, and split across repositories that coupling is untested. Started 2026-10-17, decided that day; the Stop-hook and prompt-hook ports wait for the Stop-hook blocks' measurement week. Built under `.documentation/plans/move-checkpoint-into-ele.md`.

- Why: `claude plugin eval` needs a plugin; hooks, scripts, skills and agent type ship and version together; one language. Shareable: the plugin needs no file outside itself and the ele binary, and names its prerequisites
- Prerequisites: in the [README](../../README.md). ele on `PATH` is a hard one, since the plugin's hooks call it
- The checkpoint gate, the Stop hook's block and the prompt hook's snapshot, is the plugin option `checkpoint_gate`, on by default and turned off in `/config`; the hooks read `CLAUDE_PLUGIN_OPTION_CHECKPOINT_GATE`, unset as on. Claude Code cannot turn off one hook of a plugin
- The convention files in the brief are the user's `CLAUDE.md`, under `$CLAUDE_CONFIG_DIR` when set, else `~/.claude`, with what it imports or says to read, and the repository's; the prose reviewer's per-comment tests apply where those hold no comment rules
- Phase 1, move and cut over in one change: skills, reviewer files and the `checkpoint-reviewer` agent into ele's plugin; Bash and Python as they are under `plugin/premigration/`, reached through `${CLAUDE_PLUGIN_ROOT}`; hooks from `settings.json` into the plugin's `hooks/hooks.json`, since both registered fire twice; `/ticket-claims` and `/checkpoint-stats` with their scripts; every reference renamed to `ele:`-qualified names; old copies deleted and their absence verified
- Phase 2, port mechanically: one `ele` subcommand per item, checked against the premigration script's output on the same inputs; each item deletes its script and test and verifies their absence
- `premigration/` breaks the one-language rule on purpose and briefly: kept so a port can be checked against the old code. `task check` runs its Bash suites while it exists; the plan closes only once it is gone
- Stay in claude-config: `/hook-stats`, `hook-firings.py` and the guards' `hook-log.sh`, which cover every hook. ele writes checkpoint's hook-log lines in the same path and format
- Data stays at its paths: run log, audit ledger, reviewed-file cache, mined baseline
- Contract the port keeps: log formats, and the `reviewers/<name>.md` layout ele's mining reads. Mining matches runs by the old bare names as well as the `ele:` ones, since transcripts from before the move keep the old names
- Sweep: before phase 1, search every root for every old path, every old skill and command name, every `ele` call from these files and every reference back from ele. Roots: `~/.claude` and the second profile's `~/.claude-<name>`, `~/.agents`, claude-ephant, each repository's `CLAUDE.md` and `AGENTS.md`, ele's notes. Its dated output is the plan's inventory; the close re-runs it and requires no hit on an old path or bare name outside the kept data paths, the kept log formats and mining's old-name matches
- Done: a real checkpoint through `ele:checkpoint` logs a line and its Stop hook fires from ele; `bootstrap.sh` restores it for both profiles; one eval smoke case runs; `premigration/` gone

## Open questions

Gated on a measurement or on other work, so not plan items. The first is held by an ele decision note in the claude-ephant project; the second waits on the move into ele:

- Judging Stop-hook blocks: as of 2026-10-16, 52 of 89 checkpoint blocks in 7 days were disputed by the next reply, as `~/.claude/scripts/hook-firings.py`'s regex reads it. The read-only sort ran 2026-10-16 over 90 blocks across both profiles: 67 hook wrong, 21 agent dodged, 2 unclear, all before claude-config's 9fbce04; findings in `mining/2026-10-16-hook-blocks/` in ele's data directory. Next, once a week of blocks has accrued after 9fbce04 and scoped by what they show: hook-block events carrying the `Changed:` paths and the next main-thread reply with its time, and `ele show --at <time>` opening a session at an event's turn
- Eval suites with `claude plugin eval`, a plan of their own once the move into ele lands; the move proves the harness with one smoke case. A checkpoint case is a fixture repository with planted defects, a settled decision that must not be re-raised, and a CI check the local gate lacks. Five to ten cases, run when a skill changes
