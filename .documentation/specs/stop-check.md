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
  - A read: every simple command in it, pipelines, chains, loops and `$( )` bodies included, is a listed read (`grep`, `cat`, `ls`, `sed` whose script only prints, deletes, quits or substitutes without `w` or `e`, `find` without `-exec` or `-delete`, `git log`/`status`/`show`/`diff` without `--output` and the like) or ele's own read or bookkeeping (`ele plan show|check|list`, `ele recall`, `ele reminder list|close|reopen`, `ele handoff list|close|reopen`, `ele context`, `ele checkpoint check|hash`, `ele config get`, `ele show`, `ele events`, `ele status`, `ele version`, and any `ele` command whose first flag is `--help`), with no assignment prefix on a command and no `git -c`, `rg --pre` or `sort --compress-program`, and it redirects only to `/dev/null`, a file descriptor or a path under `/tmp/` or `/private/tmp/`
  - A heredoc, and anything it cannot read whole, such as an unclosed quote, is work, except a `$(cat <<'EOF' … EOF)` substitution, the usual way to pass a multi-line note: its quoted delimiter keeps the body literal, so it reads, while an unquoted delimiter or a line after the first delimiter line stays work
  - Read from the call's whole command, since the indexed text collapses whitespace and is cut at 300 characters
  - Neither work nor mid-task: a proposal's own calls, after a proposal or a remember block until the next prompt, so `ele:wrap`'s reads do not re-pend the notes it just proposed
  - The rules are pinned in tests
- An AskUserQuestion call: work for the notes ask but not for mid-task, since a session that only decides is the one most worth notes, and a question changes no files. Not work while a checkpoint or a review runs, as nothing then is. Not work after a remember block or a proposal, until the next prompt: that question asks the proposal's own notes, and counted, skipping every note would ask again. A declined question still counts, so tool results are never parsed
- A subagent's Edit / Write / NotebookEdit calls, read from its transcript in `<session>/subagents/agent-*.jsonl` beside the session's transcript, since Claude Code keeps them out of the parent transcript: work for both the notes ask and mid-task. Not its Bash. Subagent transcripts are read after the session's, so each edit is placed by its line's timestamp. One stamped before the last notes proposal, a proposal skill or an `ele remember` command, is not work: the proposal cleared it, though read after it. It is mid-task outside a checkpoint and not while one runs. Any other edit is work outside a checkpoint, even while a proposal runs, its exemption being for its own calls, and no proposal skill launches an editing subagent. While a checkpoint runs, it is work only if not stamped after the checkpoint started, and then not mid-task, as for the parent: its reviewers are not work, and an edit delegated just before it is. An edit with no timestamp, or one equal to the checkpoint's start or the proposal's, is not placed before either, so is work unless stamped after a running checkpoint's start: a doubtful edit costs one more ask rather than dropping work. With no saved state, existing subagent files are skipped unread, so the first scan of a long session does not read every subagent transcript it ever ran
- Notes proposed, which clears the work before it: the `ele:remember`, `ele:handoff`, `ele:recipe` or `ele:wrap` skill ran, or a command starting `ele remember` did, other than `ele remember --help`, a help lookup and so a read; leading `cd <dir>;` or `cd <dir> &&` aside, its other simple commands still judged as work or read outside a checkpoint or proposal, so a write chained after the note counts and a read marks the session mid-task
- Nothing while a review runs, from the launch of its first `ele:review-reader` agent until its log line, a command starting `ele checkpoint log` (a leading `cd <dir>;` or `cd <dir> &&` allowed), or until a checkpoint starts: the review's turns end while its readers run and when a Raw Markdown answer is printed as plain text, the user's next message resuming it, so its stretch outlives prompts, and a notes ask at any of those stops would interrupt it. It starts at the readers, not at `/ele:review`, since a run refused at its target logs nothing and would leave the stretch open; a run that launches its readers has saved its diff, and logs when it finishes or stops. Work pending before it stays pending for the first stop after it. A review abandoned after its readers launch, its log never run, leaves everything after it uncounted, for notes and mid-task alike, until a checkpoint or `/clear`; that work is never asked about
- Work done since the last run of the `checkpoint` or `ele:checkpoint` skill, as a Skill call or a typed slash command, else since the session started, for mid-task. A question is not mid-task work, and a read is: `/clear` mid-investigation loses what the reads found
- Context size: `transcript.ContextTokens` of the last main-thread response, as `ele context --size` computes it

## Outputs

The two are decided independently; both can go out on one stop.

- Size hint: context ≥ threshold, the `compact_hint_at` option (default 150000; 0 turns the hint off and forgets the band hinted), in a band not yet hinted, band being `(context − threshold) / 100000` rounded down → `systemMessage` only, never a block, since a block costs another turn at full context
  - Mid-task, meaning work done since the last checkpoint: `Context <n>k. At a pause: /compact <focus>`, focus being the title of the lone active plan's next item, else a bare `/compact`. A plan stays active across sessions, so it alone is not mid-task
  - Otherwise: `Context <n>k. Write a handoff (/ele:handoff), then /clear`
- Notes block: off, the `notes_ask` option drops unproposed work at each stop, so turning it on asks only for work done after. On, its default: work done since the last block or proposal, with no block yet in this stretch, no checkpoint or review running and `stop_hook_active` false → block with the remember reason. A stretch runs from one checkpoint to the next: nearly every turn runs work, so without it nearly every turn would block. A checkpoint starts a stretch but keeps unproposed work, so the first stop after it ends, at the next prompt, still asks unless notes were proposed since; a session that never checkpoints is asked once. Not while it runs: its turn can end with reviewers still out
- Neither → exit 0

The remember reason: "Follow /ele:remember's Propose section now: one Write/Skip question per note, at most four per AskUserQuestion call and as many calls as it takes, then write each approved note yourself with ele remember. Never print the command for the user to run. If nothing is worth keeping, say so in one line." It restates the skill's per-call rule because a session can act on the reason without loading the skill. With `auto_note_writes` on, the reason says the notes are written without the question.

## State

Per session, in the data directory: the transcript offset read so far, and each subagent transcript's, with the flags derived from them and the timestamps of the last checkpoint start and notes proposal, whether this stretch has blocked, and the highest band hinted, so each stop reads only new lines and each output fires at most once per stretch or band and never loops. A block suppressed by `stop_hook_active` stays pending. Context below the threshold resets the band, so a session that compacts and regrows is hinted again. Old state is cleaned up on sync.

## Replay

`ele stop-check --replay --since <time> [--until <time>]` runs the hook as built, its `advance`, `subagentEdit` and `rememberBlock`, over archived sessions, so a change to it is judged against real stops rather than reasoned about. It prints one JSON object per stop in the window, `--until` excluded: the session, owner and project, whether the notes ask fired there (`asked`), whether the hook as built would ask (`asks`), with the work kinds that set it pending (`trigger`: `edit`, `bash`, `question`, `subagent-edit`) or why it would not (`whyNot`), and how the stop was found (`stop`).

- Transcripts: each owned main transcript with an indexed line in the window, replayed whole, so a stretch or checkpoint begun before the window carries in, ordered by earliest indexed line. An unowned one is left out: the hook exits before judging it
- Stops, `recorded`: each `stop_hook_summary` line of the main thread. As of 2026-10-27 Claude Code writes one per Stop, after the hook's block feedback. `asked` is a Stop hook block naming `ele:remember` since the previous one
- Stops, `placed`, for a transcript with no `stop_hook_summary` line, such as one older than them: a group of Stop hook blocks under 2 s apart with nothing between, before a prompt that follows output, and at the last line after output. A Stop that blocked nothing and preceded no prompt leaves no trace, so placed stops are approximate
- `stop_hook_active`: a Stop hook blocked since the last prompt. A block's feedback line parses as a prompt and the hook's state reads it as one, but it does not end that, a block group or output
- Subagent edits: the archived subagent transcripts' Edit / Write / NotebookEdit calls, each line counted once across generations, applied at the first stop at or after their stamp, since the hook reads subagent files after the transcript. At a transcript's first judged stop the hook has no saved state and records the files' sizes unread, so edits up to it, unstamped ones included, do not count
- Resumed transcripts copy earlier lines, stops included, with their ids. The hook's state is per session, so a resumed one reads the copied lines in one pass at its first stop and judges nothing at the copied stops: the replay runs the copied lines through the state but judges a stop only once, in the transcript indexed first, which the index's own copy rule makes the original
- `whyNot`, first that holds: `hook-active`, `checkpoint`, `review` running; nothing pending after a proposal since the last stop, `proposal`, else `nothing-pending`; the stretch's ask spent, `stretch`
- Each stop is judged on what the session did. After a stop where the replay and the real hook differ, the session went on as it did, so a later stop can differ only because an earlier one did, such as an ask moved earlier spending the stretch

## Beside the checkpoint gate

- The checkpoint `Stop` hook, `ele checkpoint gate`, also blocks. Neither yields: both may block on one stop, and a remember block suppressed by `stop_hook_active` after a checkpoint block waits, like any pending work, for the first stop after the checkpoint ends
- The remember skill's Propose section is the single definition the hook points at, and never falls back to printing the commands
- The checkpoint skill's Notes step defers to the hook in one line, so nothing is proposed twice
