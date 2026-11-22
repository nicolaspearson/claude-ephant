# Improving ele with your data

ele sends nothing back to its maintainers: no telemetry, no usage figures, no
crash reports. What it knows about how well its skills work comes from what
you choose to measure on your own machine and, if you choose, share. No report
on this page reaches the maintainers unless you share it yourself. What a
skill reads into your session, a mining digest included, reaches Anthropic, as
anything in a Claude Code session does; see
[What leaves your machine](data-and-privacy.md#what-leaves-your-machine).

## What you can measure

- **Flags.** `/ele:flag good|bad <why>` marks a moment of a session, so a later
  look at what went well or badly can find it; see
  [Flags](notes-and-recall.md#flags)
- **Checkpoint stats.** `/ele:checkpoint-stats [YYYY-MM-DD|Nd]` reads the
  checkpoint log and reports which reviewer found what, how much was refuted,
  what the re-review and the checks caught, and how long runs took; see
  [What checkpoints found](checkpoint-and-ticket-audit.md#what-checkpoints-found)
- **Mining.** `/ele:mine` judges past checkpoint runs and review cycles,
  including those from before the checkpoint log started, so the figures
  cover more than the log does; see [Mining](#mining)

## Mining

`/ele:mine` is typed, never started by Claude. It costs model calls on your
account, so run it when you want the figures, not routinely.

1. `ele mine` lists the checkpoint runs and review cycles no earlier mining
   judged, past where the last run stopped. Units of a session active in the
   last hour are held back, since that session can still grow
2. For each one, it writes a digest, an excerpt of the transcript
   [redacted](data-and-privacy.md#secrets) like everything ele indexes, into
   the session scratchpad, where it stays until the scratchpad goes
3. Subagents of your session classify the digests in batches of up to five,
   each with a fixed prompt shipped with the plugin. Like anything a session
   reads, the digests go to Anthropic on your account, as the session's own
   model calls; see
   [What leaves your machine](data-and-privacy.md#what-leaves-your-machine)
4. `ele` stores the judgments in `mining/` in the
   [data directory](data-and-privacy.md#storage), keyed by the prompt's hash,
   so a later prompt judges beside the old one rather than over it

Then read the figures:

```sh
ele mine --report
```

It reports from the last judgment of each unit: how many runs re-reviewed,
how reviews were started, which reviewer found what and at what severity,
and what the checks and the user caught after review.

To judge again units already judged, name them to `/ele:mine`; it never
re-judges on its own.

## What a report holds

`ele checkpoint stats` prints counts by reviewer, stage, verdict, severity and
category, and durations. Three things in it come from your work:

- A category is a free-form kebab-case label Claude gave each finding while
  running the checkpoint, not one from a fixed list, so it can carry a term
  from your project
- A check is named as the checkpoint recorded it, so a gate such as
  `go test ./app/flows/payments/...` carries its command and package path
- With nothing logged in the window, it prints the log's absolute path, your
  home directory included

`ele mine --report` prints the same kinds of counts, and one section more. Its
categories are free-form labels too, written by the classifier from each
digest. Under each `User reported a problem after …` line, it lists every such
checkpoint with the first 8 characters of its session id, its date, and up to
160 characters of the classifier's summary of what you reported. That summary
is written from your conversation, and can name a client, a screen, a file or
a person.

Read a report before you share it. Delete the lines under `User reported a
problem`, keeping the counts on the line itself, and cut anything else you
would rather keep.

## Sharing a report

Sharing is up to you, and ele has no way to do it for you: the binary's only
network calls are the ticket commands' reads through `gh` or `glab`.

To help tune the shipped reviewers and checkpoint stages:

1. Run `/ele:mine`, or skip it and use the checkpoint log alone
2. Save the reports in a terminal, the first only if you ran `/ele:mine`:

   ```sh
   ele mine --report > mine-report.txt
   ele checkpoint stats --since 30d > checkpoint-stats.txt
   ```

3. Read each report you saved, and cut what
   [What a report holds](#what-a-report-holds) names
4. Open a
   [mining report issue](https://github.com/nicolaspearson/claude-ephant/issues/new?template=mining-report.yml)
   and paste each into its field. An issue is public: anyone can read what you
   paste

A reviewer that is often refuted, one whose findings are nearly all raised by
another, or a check that keeps catching what review missed, is exactly what
the figures from many machines can show and one machine cannot.
