---
name: recall
description: Search what was discussed in earlier Claude Code sessions. Use when the user asks "did we already…", or when a past decision, command or fix would save re-deriving it.
---

# Recall

Search past sessions with `ele`: notes, the transcripts of every profile and the archived claude-mem history. Every word must appear.

## Search

```sh
ele recall <words>
```

Scoped to the current directory's owner, or with `--project <name>` to that project's owner. Widen or narrow only when the user asks or the scope is clearly wrong:

- `--all` or `--owner <name>`: another owner's history. Content then crosses into this session, so say which owner it came from
- `--kind note,prompt,conversation,tool,summary,observation`
- `--since <YYYY-MM-DD|7d>`, `--limit <n>`

## Project context

What a project already knows: its live notes and earlier sessions on a topic. Recall runs first: it syncs, and `ele context` does not.

```sh
ele recall --project <name> "<topic>"
ele context --project <name>
```

A project with no single owner fails both: say so rather than choosing an owner.

## Expand

```sh
ele show <session> --around <word>
```

A unique prefix of the session id is enough. `--context <n>` sets how many entries either side.

For the discussion behind a note, open its session at the parts that share the most of the note's rarer words, up to three, or just before the note when nothing matches; no search word needed:

```sh
ele show --note <id>
```

## Using what comes back

- Every result is dated. It says what was true then, not what the code is now: re-check any claim about the code before relying on it
- Cite the session id and date when a recalled decision shapes the answer
- A note result shows its id just before its text; `ele remember --supersedes <id>` replaces it. One marked `(superseded by <id>)` was replaced: search the topic again with `--kind note` and rely on the newer note
- Secrets are redacted in the index; `[redacted:<kind>]` is not the real value
- Nothing found means no entry indexed in the searched scope and filters contains all those words; when another sync is running, the latest session may not be indexed yet. Try fewer or different words, or ask whether to search other owners, before concluding it was never discussed
