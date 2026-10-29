# Owners

Every session ele indexes and every note it keeps belongs to an owner: the
person or organisation whose work it is. Owners keep one body of work out of
another's. A client's sessions never show up in your personal project's recall,
and their notes never reach your sessions.

An owner is not a Claude Code profile. One owner's repositories can run under
several profiles (`~/.claude`, `~/.claude-work`), and ele records the profile
on each session as a fact, but isolation follows the owner.

## Setting up

In a session, run `/ele:setup` after installing, or in a repository no owner
covers yet; the start block's line saying no owner covers the directory names
it. It reads `ele status`, asks which owner, a listed one or a new one, and what
to map: the directory above the repository, so its sibling projects come too,
the repository alone, or the project by name wherever it is checked out. Once
you approve, it runs the `ele owner` commands and shows `ele status` naming the
owner.

By hand, name an owner, then map the directories whose sessions are theirs:

```sh
ele owner add personal
ele owner add-path ~/code personal
ele owner add work
ele owner add-path ~/code/acme work
ele status
```

`ele status` shows the owner and project of the directory you run it in. Where
no owner covers it, it says so and prints the `ele owner` command that would map
it.

## How a session gets its owner

A session's project is the basename of the git root above its working
directory, or the directory itself outside a repository. Its owner is decided
in this order:

1. A project entry, set with `ele owner add-project <name> <owner>`, wins
2. Otherwise the longest path rule containing the working directory decides, so
   `~/code/acme` beats `~/code`

A session matching neither has no owner. It is still archived and indexed, but
default recall leaves it out, and its sessions start with a line saying no owner
covers the directory and naming `/ele:setup`, instead of notes.

## Changing the map

| Command                                      | What it does                                      |
| -------------------------------------------- | ------------------------------------------------- |
| `ele owner add <name>`                       | List a new owner                                  |
| `ele owner add-path <dir> <owner>`           | Map a directory and everything under it           |
| `ele owner add-project <name> <owner>`       | Map one project, ahead of any path rule           |
| `ele owner remove-path <dir>`                | Drop a path rule                                  |
| `ele owner remove-project <name>`            | Drop a project entry                              |

Each prints the projects that change owner. It refuses a relative path, an
unlisted owner or a rule that is not there. It also refuses while a sync is
running, and before the first sync has finished: run it again once the sync is
done.

Notes belong to an owner too. When a change moves a project to another owner,
the command refuses while that project has notes under the old one, and says
how many:

- `--move` refiles them under the project's new owner
- `--move-to <owner>` refiles them under a listed owner you name, for a project
  no single owner covers after the change

After writing the map, the command starts a background sync, which rebuilds the
index so every session is filed under its new owner. `--no-sync` skips it; the
next sync rebuilds instead.

The map is `owner-map.json` in the data directory and can be edited by hand.
ele notices a changed map at the next sync and rebuilds the same way.

## Crossing owners

Nothing crosses owners unless you ask:

- `/ele:recall` and `ele recall` search the current directory's owner.
  `--owner <name>` searches another, `--all` every owner
- `--project <name>` on `recall`, `remember`, `context` and the list commands
  works on that project under its own owner, from any directory
- The block a session starts with never shows another owner's notes
- `ele config set --owner <name>` overrides an option for one owner's projects;
  see [Data and privacy](data-and-privacy.md#options)
