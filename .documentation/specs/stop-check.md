# Stop check

## Why

- Without it, notes are proposed unasked only at a checkpoint, so sessions that never reach one (investigations, debugging, questions) never prompt for notes
- Agents print `ele remember …` commands for the user to run instead of asking with AskUserQuestion and writing the note themselves, which the remember skill's Propose section requires
- Nothing stops a session growing: as of 2026-10-19 sessions run with the 1M window, whose auto-compact fires near 967k, and only the user can run `/compact`; no hook output, tool or skill triggers it

## Hook

`ele stop-check` is a `Stop` hook in `plugin/hooks/hooks.json`, run as `ele stop-check || exit 1`.

- Input: stdin JSON with `session_id`, `transcript_path`, `cwd`, `stop_hook_active`
- Exits 0 silently when `cwd` has no owner. `stop_hook_active` true suppresses the block only; a message cannot loop

## What it reads

The transcript, through `internal/transcript`, for:

- Work done since the last block or proposal: Edit / Write / NotebookEdit, or a Bash call other than a read, since a command can change files, and a wrong `/clear` mid-task loses more than a wrong `/compact`
  - A read: every simple command in it, pipelines, chains, loops and `$( )` bodies included, is a listed read (`grep`, `cat`, `ls`, `sed` whose script only prints, deletes, quits or substitutes without `w` or `e`, `find` without `-exec` or `-delete`, `git log`/`status`/`show`/`diff` without `--output` and the like) or ele's own read or bookkeeping (`ele plan show|check|list`, `ele recall`, `ele reminder list|close|reopen`, `ele handoff list|close|reopen`, `ele context`, `ele checkpoint check|hash`, `ele config get`, `ele show`, `ele events`, `ele status`, `ele version`), with no assignment prefix on a command and no `git -c`, `rg --pre` or `sort --compress-program`, and it redirects only to `/dev/null`, a file descriptor or a path under `/tmp/` or `/private/tmp/`
  - A heredoc, and anything it cannot read whole, such as an unclosed quote, is work, except a `$(cat <<'EOF' … EOF)` substitution, the usual way to pass a multi-line note: its quoted delimiter keeps the body literal, so it reads, while an unquoted delimiter or a line after the first delimiter line stays work
  - Read from the call's whole command, since the indexed text collapses whitespace and is cut at 300 characters
  - Neither work nor mid-task: a proposal's own calls, after a proposal or a remember block until the next prompt, so `ele:wrap`'s reads do not re-pend the notes it just proposed
  - The rules are pinned in tests
- An AskUserQuestion call: work for the notes ask but not for mid-task, since a session that only decides is the one most worth notes, and a question changes no files. Not work while a checkpoint runs, as nothing then is. Not work after a remember block or a proposal, until the next prompt: that question asks the proposal's own notes, and counted, skipping every note would ask again. A declined question still counts, so tool results are never parsed
- A subagent's Edit / Write / NotebookEdit calls, read from its transcript in `<session>/subagents/agent-*.jsonl` beside the session's transcript, since Claude Code keeps them out of the parent transcript: work for both the notes ask and mid-task. Not its Bash. While a checkpoint runs, a subagent's edit is work only if made before the checkpoint started, by the line's timestamp, and then not mid-task, as for the parent: its reviewers are not work, and an edit delegated just before it is. With no saved state, existing subagent files are skipped unread, so the first scan of a long session does not read every subagent transcript it ever ran
- Notes proposed, which clears the work before it: the `ele:remember`, `ele:handoff` or `ele:wrap` skill ran, or a command starting `ele remember` did, leading `cd <dir>;` or `cd <dir> &&` aside, its other simple commands still judged as work or read outside a checkpoint or proposal, so a write chained after the note counts and a read marks the session mid-task
- Work done since the last run of the `checkpoint` or `ele:checkpoint` skill, as a Skill call or a typed slash command, else since the session started, for mid-task. A question is not mid-task work, and a read is: `/clear` mid-investigation loses what the reads found
- Context size: `transcript.ContextTokens` of the last main-thread response, as `ele context --size` computes it

## Outputs

The two are decided independently; both can go out on one stop.

- Size hint: context ≥ threshold (default 150000, flag `--hint-at`) in a band not yet hinted, band being `(context − threshold) / 100000` rounded down → `systemMessage` only, never a block, since a block costs another turn at full context
  - Mid-task, meaning work done since the last checkpoint: `Context <n>k. At a pause: /compact <focus>`, focus being the title of the lone active plan's next item, else a bare `/compact`. A plan stays active across sessions, so it alone is not mid-task
  - Otherwise: `Context <n>k. Write a handoff (/ele:handoff), then /clear`
- Notes block: work done since the last block or proposal, with no block yet in this stretch, no checkpoint running and `stop_hook_active` false → block with the remember reason. A stretch runs from one checkpoint to the next: nearly every turn runs work, so without it nearly every turn would block. A checkpoint starts a stretch but keeps unproposed work, so the first stop after it ends, at the next prompt, still asks unless notes were proposed since; a session that never checkpoints is asked once. Not while it runs: its turn can end with reviewers still out
- Neither → exit 0

The remember reason: "Follow /ele:remember's Propose section now: one Write/Skip question per note, at most four per AskUserQuestion call and as many calls as it takes, then write each approved note yourself with ele remember. Never print the command for the user to run. If nothing is worth keeping, say so in one line." With `auto_note_writes` on, the reason says the notes are written without the question.

## State

Per session, in the data directory: the transcript offset read so far, and each subagent transcript's, with the flags derived from them, whether this stretch has blocked, and the highest band hinted, so each stop reads only new lines and each output fires at most once per stretch or band and never loops. A block suppressed by `stop_hook_active` stays pending. Context below the threshold resets the band, so a session that compacts and regrows is hinted again. Old state is cleaned up on sync.

## Beside the checkpoint gate

- The checkpoint `Stop` hook, `ele checkpoint gate`, also blocks. Neither yields: both may block on one stop, and a remember block suppressed by `stop_hook_active` after a checkpoint block waits, like any pending work, for the first stop after the checkpoint ends
- The remember skill's Propose section is the single definition the hook points at, and never falls back to printing the commands
- The checkpoint skill's Notes step defers to the hook in one line, so nothing is proposed twice
