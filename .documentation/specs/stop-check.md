# Stop check

## Why

As of 2026-10-17, ele's hooks (`plugin/hooks/hooks.json`) only print the SessionStart block and sync. Proposing notes unasked happens only when the checkpoint skill's Notes step runs, which is only at a checkpoint and only if the agent follows it. In practice:

- sessions that never reach a checkpoint (investigations, debugging, questions) never prompt for notes
- agents often print `ele remember …` commands for the user to run instead of asking with AskUserQuestion and writing the note themselves, which the remember skill's Propose section requires. It happened twice in one session on 2026-10-17
- as of 2026-10-19, ele events since 2026-10-10, counting subagent responses toward 150k: 8 of the 87 sessions that passed 150k ran `ele:handoff`, 1 of 22 in claude-ephant and 7 of 65 elsewhere. `ele:remember` ran in 10 of 48 claude-ephant sessions and 8 of 86 elsewhere
- as of 2026-10-19, sessions run with the 1M window, whose auto-compact fires near 967k, so nothing stops a session growing far past 150k. Only the user can run `/compact`: no hook output, tool or skill triggers it

## What to build

1. `ele stop-check`, a new subcommand in `cmd/ele`, registered as a `Stop` hook in `plugin/hooks/hooks.json`
   - Before writing it, confirm the current Claude Code Stop hook contract from the docs (stdin fields; the `{"decision":"block","reason":…}` output that makes the agent continue; `systemMessage` without a block, which ends the turn). Do not rely on memory
   - Input: stdin JSON with `session_id`, `transcript_path`, `cwd`, `stop_hook_active`
   - Exit 0 silently when `cwd` has no owner (`internal/ownermap`). `stop_hook_active` true suppresses the block only; a message cannot loop
2. Read the transcript with `internal/transcript` to find:
   - work done since the last block or proposal (step 5): Edit / Write / NotebookEdit or any Bash call, decided 2026-10-19: a command can change files, and a wrong `/clear` mid-task loses more than a wrong `/compact`. The rule is pinned in a test
   - an AskUserQuestion call, work for the notes ask but not for mid-task, decided 2026-10-20: a session that only decides is the one most worth notes, and a question changes no files. Not work while a checkpoint runs, as nothing then is. Not work after a remember block or a proposal, until the next prompt: that question asks the proposal's own notes, and counted, skipping every note would ask again. A declined question still counts, so tool results are never parsed
   - a subagent's Edit / Write / NotebookEdit calls, read from its transcript in `<session>/subagents/agent-*.jsonl` beside the session's transcript, since Claude Code keeps them out of the parent transcript: work for both the notes ask and mid-task, decided 2026-10-20. Not its Bash: as of 2026-10-20, in one session an Explore agent made 21 Bash calls and its reviewers 4 to 17 each, all reads. While a checkpoint runs, a subagent's edit is work only if made before the checkpoint started, by the line's timestamp, and then not mid-task, as for the parent: its reviewers are not work, and an edit delegated just before it is. With no saved state, existing subagent files are skipped unread: as of 2026-10-20, on the 153 MB session with 181 MB of subagent files and a warm cache, reading them took the first scan from 0.7s to between 1.6s and 2.7s; skipped, it stayed at 0.7s
   - notes proposed, which clears the work before it: the `ele:remember` or `ele:handoff` skill ran, or a command starting `ele remember` did
   - work done since the last run of the `checkpoint` or `ele:checkpoint` skill, as a Skill call or a typed slash command, else since the session started, for mid-task in step 3; a question is not mid-task work
   - context size: `transcript.ContextTokens` of the last main-thread response, as `ele context --size` computes it
3. Decide the two outputs independently; both can go out on one stop:
   - size hint: context ≥ threshold (default 150000, flag `--hint-at`) in a band not yet hinted, band being `(context − threshold) / 100000` rounded down → `systemMessage` only, never a block, since a block costs another turn at full context
     - mid-task, meaning work done since the last checkpoint (step 2): `Context <n>k. At a pause: /compact <focus>`, focus being the title of the lone active plan's next item, else a bare `/compact`. A plan stays active across sessions, so it alone is not mid-task
     - otherwise: `Context <n>k. Write a handoff (/ele:handoff), then /clear`
   - work done since the last block or proposal, with no block yet in this stretch, no checkpoint running and `stop_hook_active` false → block, reason = remember instruction. A stretch runs from one checkpoint to the next, decided 2026-10-19: any Bash call is work, so without it nearly every turn would block. A checkpoint starts a stretch but keeps unproposed work, so the first stop after it ends, at the next prompt, still asks unless notes were proposed since; a session that never checkpoints is asked once. Not while it runs: its turn can end with reviewers still out
   - neither → exit 0
4. Reason text, short and imperative:
   - remember: "Follow /ele:remember's Propose section now: one AskUserQuestion call, one Write/Skip question per note, then write each approved note yourself with ele remember. Never print the command for the user to run. If nothing is worth keeping, say so in one line."
5. Per-session state in the data directory (`internal/datadir`): the transcript offset read so far, and each subagent transcript's, with the flags derived from them, whether this stretch has blocked, and the highest band hinted, so each stop reads only new lines and each output fires at most once per stretch or band and never loops. A block suppressed by `stop_hook_active` stays pending. Context below the threshold resets the band, so a session that compacts and regrows is hinted again. Clean up old state on sync
6. Update `plugin/skills/remember` so the Propose section is the single definition the hook points at

## Decided

2026-10-19:

- The checkpoint `Stop` hook, `plugin/premigration/checkpoint-review.sh` since 2026-10-20, also blocks. Neither yields: both may block on one stop, and a remember block suppressed by `stop_hook_active` after a checkpoint block waits, like any pending work, for the first stop after the checkpoint ends
- The remember skill's Propose section no longer falls back to printing the commands
- The design spec's Session handoff section names the size hint as what suggests the handoff

## Open

- How does Claude Code combine two Stop hooks' block reasons on one stop? As of 2026-10-19 the hooks docs say only that matching hooks run in parallel. If only one reason reaches the agent, the remember block's stretch is spent unasked. Observe in a real session

## In the checkpoint skill

The checkpoint skill's Notes step defers to the hook in one line, so nothing is proposed twice. Applied 2026-10-19 on the user's approval of the diff.

## Done means

- Table tests for the decision function: every combination of {work done / not} × {proposed / not} × {context under / at / over the threshold} × {band hinted / not} × {mid-task with one plan / with several / with none / between tasks} × {last checkpoint a Skill call / a typed command / none} × {`stop_hook_active`} × {owner / no owner}, asserting the block and the message separately: block vs none, which message, and the `/compact` focus
- A test that a block is not repeated without new work after it
- A test that the size hint is not repeated within a band across stops, `stop_hook_active` stops included, and is shown again after a compaction drops the context below the threshold and it regrows
- Each new test fails once against a mutated decision function before it passes
- An end-to-end run of `ele stop-check` against a recorded transcript fixture, asserting the JSON it prints
- `task` gate green, run with the cache forced; say whether it was forced
- Changes left in the working tree, with commit commands printed
