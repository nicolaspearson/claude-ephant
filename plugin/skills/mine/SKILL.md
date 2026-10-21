---
name: mine
description: Classify the checkpoint runs and review cycles no earlier mining judged, or re-judge named ones, and store the judgments in ele's mining cache.
disable-model-invocation: true
---

# Mine

Each analysis lists the units no pass has judged, past its watermark, unless a re-judge names its units. Subagents classify their digests with a fixed prompt beside this file, and `ele` stores the judgments under that prompt's hash.

| Analysis          | Prompt                        |
| ----------------- | ----------------------------- |
| `checkpoint-runs` | `classify-checkpoint-runs.md` |
| `review-cycles`   | `classify-review-cycles.md`   |

Both analyses unless the user names one. Pass each prompt as an absolute path resolved from this skill's base directory.

**Never edit a prompt to suit a run.** Its bytes are the classifier: a changed prompt judges under a new hash, and re-judging units already judged is a separate request the user scopes by naming its units.

## 1. List

Per analysis, `<scratchpad>` being the session scratchpad:

```sh
mkdir -p <scratchpad>/mine && find <scratchpad>/mine -maxdepth 1 -name '<analysis>-*.out.jsonl' -delete
ele mine --digests <scratchpad>/mine/<analysis> <analysis> > <scratchpad>/mine/<analysis>.jsonl 2> <scratchpad>/mine/<analysis>.err; echo $?
```

To re-judge named units, list only those: write each as `{"session":…,"timestamp":…}` on its own line of `<scratchpad>/mine/<analysis>.units.jsonl` and add `--units <that file>` to the list command; the rest runs unchanged. A unit judged before is stored again only when its prompt or its digest has changed. When `ele` refuses the units file, correct it from the user's list, never by guessing a unit.

Read `<analysis>.err`. On a non-zero exit it holds the error. Otherwise it may warn that the sync failed or the index is incomplete, so the list may be short, and may count the unjudged units at or before the watermark, which this skill leaves alone. Each goes in the report. Each line of `<analysis>.jsonl` is one unit, its `digest` the file to classify. Nothing listed, say so and stop for that analysis.

## 2. Classify

Split each analysis's units into batches of up to five. Launch one `general-purpose` subagent per batch, all in one message so they run in parallel, in the background. Each prompt is:

```
Read <prompt> first and follow it.
Digests:
<digest path>
...
Output file: <scratchpad>/mine/<analysis>-<batch>.out.jsonl
```

## 3. Store

Once every batch has reported, per analysis, join its output files and store them. `awk 1` ends each file's last line, which `cat` would run into the next file's first:

```sh
awk 1 <scratchpad>/mine/<analysis>-*.out.jsonl > <scratchpad>/mine/<analysis>.judged.jsonl
ele mine --judgments <scratchpad>/mine/<analysis>.judged.jsonl --prompt <prompt> --digests <scratchpad>/mine/<analysis> <analysis>
```

When no batch wrote an output file, store nothing and report every unit as unjudged.

`ele` hashes the digest file each batch read. It refuses the whole file when a line names no unit of the analysis, a unit with no digest file, or one unit twice, and stores nothing. Correct that line's `session` or `timestamp` from line 1 of its digest — never the prompt — and store again. A unit a batch left out is not a failure: the watermark stops before it, and the next mine lists it again.

## 4. Report

Per analysis: units listed, judgments stored, the watermark `ele` printed, units left unjudged with the batch that missed them, the count behind the watermark, and the runs each subagent called hard to classify.
