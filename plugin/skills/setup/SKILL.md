---
name: setup
description: Set up ele after installing it, or in a directory no owner covers, by proposing the ele owner commands that map it and running them once the user approves. Use when the user asks to set up or configure ele, when the start block says no owner covers the directory, or when ele status names no owner here.
---

# Setup

Every session ele indexes and every note it keeps belongs to an owner: the person or organisation whose work it is. A directory no owner covers gets no notes, and recall there needs `--owner` or `--all`. Setup maps the working directory to an owner through `ele owner`, which writes `owner-map.json`. The user approves each mapping: never write one unasked.

## Find what is missing

```sh
ele status
```

Read the `Here` line:

- An owner and project, as `personal · claude-ephant`: say this directory is set up, under that owner, and stop
- A project or path rule naming an owner the map does not list: the fix is the `ele owner add <name>` it prints
- No owner covers the directory: propose a mapping, below

Read `owner-map.json` in the `Data` directory for the owners already listed; a missing file lists none.

## Propose

With AskUserQuestion, loading it through ToolSearch if it is deferred, ask in one call:

- Which owner: each listed owner, and a new one. A new owner's name is letters, digits, `-` and `_`, starting with a letter or digit; take it from the user's `Other`, never invent one
- What to map, absolute paths written out, each option with the command it runs:
  - The directory above the root: `ele owner add-path <parent> <owner>`, covering every project under it. Offer it first, unless it is the home directory or `/`
  - The root alone: `ele owner add-path <root> <owner>`
  - This project by name, wherever it is checked out: `ele owner add-project <project> <owner>`, the project being the one `ele status` named

Build the paths from the directory the `Here` line names, never from `git rev-parse`: ele matches rules against that path as the shell reached it, and git prints it with symlinks resolved. The root is the nearest directory at or above it holding `.git`, or that directory itself outside a repository. Where `pwd -P` prints another path, each path rule is proposed for both forms, as two commands. In a worktree, whose `.git` is a file, the directory above the root holds only worktrees: offer the directory above the main checkout instead, the checkout being the directory above `git rev-parse --path-format=absolute --git-common-dir`.

An unlisted owner the map already names needs only the owner question: confirm the `ele owner add` it printed.

## Run

On approval, run the commands in order: `ele owner add <owner>` first for a new owner, then the mapping. Pass `--no-sync` to every one but the last: each write starts a background sync, whose lock refuses the next write.

When one refuses with `a sync holds the archive lock` or `no sync has completed`, run `ele sync` with the Bash tool's timeout at its maximum: it waits for a running sync, and a first sync over every transcript can take minutes. Then run the refused command once more. When the timeout cuts `ele sync` off, say a sync is still running and to run `/ele:setup` again once `ele status` shows none.

When the last command prints `nothing written` after an earlier one took `--no-sync`, run `ele sync --background`, so the earlier write is still synced.

A refusal over notes a project has under another owner names `--move` and `--move-to <owner>`: show it and ask which, never pick one.

Then run `ele status` again.

Report what each command printed, and the `Here` line now naming the owner and project. Name what the mapping covers: every project under a path rule, or the one project. Say the background sync files the existing sessions under the owner, and that a session started after it shows this project's notes.

Done when `ele status` names an owner here, or the reply shows the refusal that stopped it.
