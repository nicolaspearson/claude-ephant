---
kind: plan
created: 2026-11-14
spec: .documentation/specs/stop-check.md
---

# Stop hook

## Why

ele's Stop hook, `ele stop-check`, asks for notes and hints at the context size,
and five questions raised against it since 2026-10-23 in real sessions and
reviews are still open: a first turn that ended over the threshold got no size
hint, a session asked for notes once is never asked again unless it
checkpoints, a subagent's edits before the first stop are never asked about, an
edit a checkpoint cleared reads as mid-task when its stops are read late, and
the hint sends the user to `/ele:handoff` where `/ele:wrap` does more. This plan
probes the two Claude Code behaviours those questions guessed at and records
each in the spec, and fixes the rest without resting a fix on a guess.

## Requirements

1. A probe checks whether a turn's usage lines are on disk when the Stop hook
   reads; the spec says how late the hint can come, or an inconclusive run goes
   to the Inbox
2. A session that never checkpoints again is asked for notes again, 60 minutes
   after the last ask, if work happened since
3. Edits a subagent made before a session's first stop are asked about
4. A subagent edit a checkpoint cleared does not turn the size hint into
   `/compact`, whatever order a finished scan reads it in
5. The size hint names `/ele:wrap`; the Stop hook never runs wrap itself

## Context

- As of 2026-11-14: `cmd/ele` `lastContextTokens` reads the transcript tail in
  a 1 MiB window, growing 4x, for the newest line `transcript.ContextTokens`
  accepts (an assistant line, not a sidechain, with a usage total above 0); with
  none on disk it returns 0 and `sizeHint` gives no hint and no error. Nothing
  in stop-check waits or retries; its only timing is the 10 s `scanBudget`
- As of 2026-11-14: a fresh `stopState` has `Band: noBand`, so a turn over
  `compact_hint_at` hints at its stop; the band is hinted once, so a stop that
  read no usage leaves the next stop to hint. An earlier main-thread assistant
  line of the turn may sit below the threshold where tool results grew it, so a
  missing hint needs every usage line over the threshold to be off disk; not
  checked
- As of 2026-11-14: a real session's first turn, raised 2026-10-28, ended at
  54k with `compact_hint_at` 20000 and no hint, while the same transcript cut at
  that stop and fed to stop-check by hand printed one. That the usage line was
  not yet on disk is not checked; as of 2026-11-14 Claude Code's hooks docs say
  the transcript is written asynchronously and may not yet hold the current
  turn's newest messages when a hook fires
- As of 2026-11-14: `stopState.StretchBlocked` is set by `rememberBlock` when it
  asks, which also clears `Pending`; calls while `Proposing`, until the next
  prompt, are not work, so `Pending` with `StretchBlocked` means work after the
  ask's turn, an AskUserQuestion included; but `subagentEdit` sets `Pending`
  whatever `Proposing` holds, so a subagent edit made before the ask and read
  after it also counts. `StretchBlocked` is cleared by a
  checkpoint event in `advance` and by a fresh state: a replaced transcript, or
  a state file missing, removed by sync after 7 days, or not parsing. Nothing in
  `stopState` holds the ask's time. Replay calls `rememberBlock` at each stop
  (`replayer.stop`). In 2026-10-20..27 one session asked on 2026-10-19 made 44
  edits and 52 subagent edits over about five hours with no further ask; the
  counts are not checked
- As of 2026-11-14: `scanSubagents` records each `agent-*.jsonl` file's size
  unread when `stopState.Subagents` is nil and the parent scan is not behind; a
  behind stop sets an empty map, and the stop that catches up reads every
  subagent file from offset 0 with no deadline
  (`TestRunStopCheckResumesAScanCutByItsBudget`). Replay copies the skip:
  `replayer.stop` applies subagent edits only once `r.judged`, asserted by
  `TestReplayTranscriptAppliesSubagentEditsAtTheFirstStopAfterThem`, and
  `TestScanSubagentsCountsFileChangesOnly` asserts the hook's skip. The spec's
  "What it reads" states the skip without its not-behind condition; its State
  section has it
- As of 2026-11-14: `subagentEdit` sets `MidTask` for an edit stamped before
  `NotesProposed` when `InCheckpoint` is false, and keeps `Pending` for one
  stamped before `CheckpointStarted` inside a checkpoint
  (`TestRunStopCheckKeepsASubagentsEditBeforeACheckpoint`). An edit, a
  checkpoint and a proposal read only after the next prompt has cleared
  `InCheckpoint` set `MidTask`, and `sizeHint` gives `At a pause: /compact`.
  `CheckpointStarted` and `NotesProposed` are transcript timestamp strings, and
  `subagentEdit` compares them with an edit's stamp as strings.
  A stop behind on its budget reaches it with Stop firing, since its catch-up
  scans the parent past the prompt before any subagent file. A subagent line
  not yet on disk at the stop is another order, not checked; the hooks docs say
  the transcript write is asynchronous. As of 2026-11-14 the hooks docs say Stop
  does not run on a user interrupt and an API error fires StopFailure instead,
  so an interrupted or failed turn is read only after the next prompt too; not
  checked on the installed version. The ordering tests read for this plan in
  `stopcheck_test.go` each call a stop before the next prompt. A subagent edit
  stamped after `CheckpointStarted` is read as a reviewer's. Reviewers are
  briefed to edit nothing; `ele:checkpoint-reviewer` has no Edit or Write tool,
  but the Bugs reviewer and small mode run as `general-purpose`, which has
  both, and a subagent launched before the checkpoint can still edit after it
  starts
- As of 2026-11-14: `sizeHint` says `Context <n>k. Write a handoff
  (/ele:handoff), then /clear` when not mid-task, and `At a pause: /compact`,
  with the lone active plan's next item where there is one, when mid-task. The
  only `wrap` in stop-check is `proposalNames`, where `ele:wrap` counts as a
  proposal. The old text is quoted in `sizeHint`, `cmd/ele` `stopcheck_test.go`,
  `internal/highlight` `highlight_test.go`'s stop-check hint case and the spec,
  and paraphrased in `.documentation/specs/design.md`,
  `docs/handoffs-and-reminders.md`, `docs/data-and-privacy.md`, `README.md`,
  `cmd/ele` `help.go`'s stop-check help, `plugin/hooks/hooks.json`'s
  description and `internal/config`'s `compact_hint_at` description, which
  `docs/data-and-privacy.md`'s options table copies; the help and hooks.json
  also say once per stretch between checkpoints. No eval or skill quotes it.
  `highlight.Message` colours any `/x:y` command, `/ele:wrap` included
- As of 2026-11-14: the `hook-output-probe` user skill runs a headless
  `claude -p` session whose hook prints the output under test, one hook per
  event; it captures no transcript at hook time, and `claude -p` has no Esc.
  The spec's Replay section says a transcript holds one `stop_hook_summary`
  per Stop, observed on Claude Code 2.1.311; not re-measured

## Decisions

- A guess about Claude Code's behaviour is probed on the installed version and
  recorded in the spec, and no fix rests on one: the cleared-subagent-edit fix
  covers every order a finished scan reads a subagent edit in, since a stop
  behind on its budget reaches the bug with Stop firing, whatever a user
  interrupt does; a behind stop's own hint is out of scope
- A first turn none of whose usage lines over the threshold is on disk at its
  stop gets its hint one stop late, stated in the spec: the band stays
  unhinted, so the next stop hints
- A notes stretch also ends 60 minutes after its ask when work happened since,
  a fixed figure with no config option: a checkpoint was the only reset a live
  session reaches, and a session that never checkpoints again is otherwise never
  asked again. Work since counts what `Pending` counts, an AskUserQuestion
  included, one rule for both asks, since an answered question is often a
  decision worth a note. The 60 minutes run on the hook's clock behind a package
  variable, as `planClock` is, and replay passes each stop's own timestamp, so
  a live stop and a replayed one judge alike
- Subagent files are read from offset 0 on a session's first stop, as the
  behind path already does once caught up: the unread skip saved a first read
  at the cost of never asking about those edits. The read keeps no deadline on
  any stop with no saved offsets, a stale, unparseable or replaced state
  included, accepting that a stop the timeout kills there saves nothing and the
  next repeats it, as the catch-up read already accepts
- The size hint names `/ele:wrap`, since wrap offers the handoff with the notes
  and reminders a session end needs

## Actions

- [x] probe-first-turn-usage: Probe whether a turn's usage lines are on disk
      when the Stop hook reads the transcript, on the installed Claude Code,
      with a Stop hook whose body records each main-thread assistant usage line
      it can read against those the finished transcript holds, for a turn with
      no tool call and one with tool calls
  - done when, in an interactive session as the report was, the user installs
    the probe hook and drives both turns, and the spec states, as of the Claude
    Code version probed, which usage lines a Stop hook reads and, where none of
    a turn's lines over the threshold is on disk, that its hint comes one stop
    late; a run with no miss is inconclusive, since the write is a race, and
    goes to this plan's Inbox as such
  - asserted in the probe-first-turn-usage commit: in two interactive sessions
    on Claude Code 2.1.311, a Stop hook copying the transcript at each of four
    stops found its turn's usage lines on disk at three and none at the
    fourth, whose copy ended at the turn's `api-request` line; the stop-check
    spec's Context size bullet states it and the one-stop-late hint
- [ ] probe-esc-stop: Find whether Claude Code fires Stop when a turn is
      ended by a user interrupt: archived transcripts mark Esc, Ctrl+C and a
      declined permission alike and can show only that Stop fired, by a
      `stop_hook_summary` after the interrupt; a skip is settled by a Stop hook
      writing a marker, in an interactive session the user drives and
      interrupts with Esc. No item waits on it and no fix rests on it: it
      records, for the spec only, a Claude Code fact beside the subagent rule
  - done when the spec states, as of the Claude Code version seen, whether an
    Esc interrupt fires Stop, and how it was seen
- [ ] cleared-subagent-edit: `subagentEdit` treats a subagent edit stamped
      before `CheckpointStarted` as cleared by that checkpoint when the
      checkpoint is read as over, so it sets no `MidTask`
  - done when stop-check tests with a subagent edit stamped before
    `CheckpointStarted`, read after the next prompt both with no stop between
    and through a stop behind on its budget that catches up after the prompt,
    assert no `MidTask` and the not-mid-task hint; with a proposal after the
    edit there is no ask, and with none `Pending` is kept; the behind test
    writes its over-threshold usage line only after its last behind stop, so
    the band is not spent first; an unstamped edit and one stamped exactly at
    `CheckpointStarted`, read late, still set `MidTask`; and the spec's
    subagent bullet says so
- [ ] subagent-first-read: Read subagent files from offset 0 on a first stop
      that is not behind, ending the unread skip in `scanSubagents` and
      `replayer.stop`
  - done when a stop-check test and a stopreplay test each count a subagent
    edit before the session's first stop;
    `TestScanSubagentsCountsFileChangesOnly` asserts the read; the spec's "What
    it reads", State and Replay sections describe one read rule and State says
    the read keeps no deadline; and the change in asks per session from
    `ele stop-check --replay` over a recent window, before and after, is written
    under this item
- [ ] stretch-time: End a notes stretch 60 minutes after its ask when work
      happened since, so the ask comes again, the ask's time kept in
      `stopState` in the transcript's timestamp form so it compares with an
      edit's stamp, the hook's clock behind a package variable, and replay
      passing each stop's timestamp
  - done when stop-check tests show a second ask after 60 minutes with work
    since the first, an AskUserQuestion alone included; none before 60
    minutes; none after 60 minutes with no work since or with only a subagent
    edit stamped before the ask; and a state file holding `StretchBlocked` with
    no ask time, saved before this change, read as asked at the first stop that
    loads it; a stopreplay test judges an archived stretch by its own
    timestamps; the change in asks per session from
    `ele stop-check --replay` over a recent window, before and after, is
    written under this item; and the spec's stretch rule, State and Replay
    sections, `docs/notes-and-recall.md`, `README.md`, `cmd/ele` `help.go` and
    `plugin/hooks/hooks.json` say so
- [ ] wrap-hint: The size hint says `Wrap up (/ele:wrap), then /clear` when not
      mid-task
  - done when `sizeHint`'s tests and `highlight_test.go`'s stop-check hint case
    assert the new text; the spec, `.documentation/specs/design.md`,
    `docs/handoffs-and-reminders.md`, `docs/data-and-privacy.md`, `README.md`,
    `cmd/ele` `help.go`, `plugin/hooks/hooks.json` and the `compact_hint_at`
    description name `/ele:wrap`; and a grep for `Write a handoff
    (/ele:handoff)` outside `.documentation/plans/` finds none
- [ ] Fold each Decision through
      `/ele:plan-audit close .documentation/plans/stop-hook.md`, then what this
      plan built and its out-of-scope lines as `Not doing:` into the spec, run
      `ele plan done .documentation/plans/stop-hook.md`, and delete this plan

## Inbox

## Out of scope

- The Stop hook running `/ele:wrap` itself: the hint names it and the user
  starts it (ruled 2026-11-14, from this plan's rounds)
- A wait or retry in the Stop hook for a usage line not yet on disk: the next
  stop hints anyway, and a wait slows every stop (ruled 2026-11-14, from this
  plan's rounds)
- A behind stop's size hint judged from partial state, which can give
  `/compact` for an edit whose checkpoint the scan has not reached and spend
  the band: it keeps the hint timely on a long transcript, where the size
  matters most, and a wrong choice between `/compact` and wrap costs little
  (ruled 2026-11-14, from this plan's review loop)
- The checkpoint gate across live sessions: a plan candidate of its own, with
  its own spec, `.documentation/specs/checkpoint-and-ticket-audit.md` (ruled
  2026-11-14, from this plan's rounds)
