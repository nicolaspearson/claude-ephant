# Recipes

A recipe is a procedure you will run again, such as a release, an upgrade or a
credential rotation, kept as a skill of your own so a later session follows it
instead of working it out again. A note holds at most three lines, too few for
the steps.

1. After running the procedure, type `/ele:recipe`, naming it if the session ran
   several: `/ele:recipe the release steps`
2. Claude drafts a skill: a name, a description saying when to use it, and the
   steps with the commands as they ran
3. Answer Write or Skip. A recipe is always asked, even with the
   `auto_note_writes` option on, unless you handed Claude the skill's exact
   words and asked for them to be saved

The skill is written to `skills/<name>/SKILL.md` in every profile: the one the
session runs in, and each other `~/.claude` or `~/.claude-*` holding a
`projects` folder. Sessions started after it list it as `/<name>` whichever
profile they run in. A profile added later does not get it. A skill by that name
in any profile is never overwritten. When the `recipe_dir` option below is
unset, Claude's report says so and how to set it.

To keep your skills in one folder, such as a git repository, set the
`recipe_dir` option to it: `ele config set recipe_dir ~/.agents/skills`, typed
in a terminal. The recipe is then written once, to `<folder>/<name>/SKILL.md`,
and each profile's `skills/<name>` is a link to it. A profile already holding a
skill by that name keeps it, and Claude names it. `set` refuses a path that is
not a folder.

`/ele:wrap` offers a recipe too, when the session ran one worth keeping. The
`recipes` option off stops it offering; `/ele:recipe` still works when typed.

A procedure your teammates or CI must also run belongs in the repository, as a
task target or a doc, so Claude says so instead of drafting a skill only you
would have.

## Auditing your skills

`/ele:recipe audit` checks your skills, recipes and any others in a profile's
`skills` folder or the `recipe_dir` folder, for commands, subcommands, flags
and versions that no longer exist on this machine. Name skill files to audit
only those:
`/ele:recipe audit ~/.claude/skills/release-cli/SKILL.md`.

- A command not on your `PATH` is reported, and so is a subcommand or flag that
  neither its `--help`, nor a help it points to, nor its man page names:
  `git log -h` leaves out `--oneline`, which `man git-log` names
- A pinned version older than the installed tool's by a major or minor
  version, such as `go1.21.0` with Go 1.27 installed, is reported as stale
- A package runner, such as `npx`, `bunx`, `pipx run` or `uvx`, or any other
  command that takes a package and runs it, is checked for
  itself only: its package and what follows are listed as not checked, since
  asking the package for its help would fetch and run it
- Nothing but `command -v`, help and version commands and `man` is run, `man`
  in the C locale so a flag's hyphens print as plain `-`. A shell function or
  alias has nothing run, a script is read rather than run (with the files it
  loads, when it defines no commands of its own), and a subcommand's help is
  asked for only when the tool's own help shows it takes one, since a tool
  that ignores an unknown flag would run the step. A compiled tool that
  ignores its flags still runs once

Each finding comes with the line it is on; where the fix is clear, Claude shows
the line before and after and edits the skill only once you approve. Without
paths, plugin skills and claude.ai's synced skills are left out.
