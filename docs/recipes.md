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
in any profile is never overwritten.

`/ele:wrap` offers a recipe too, when the session ran one worth keeping. The
`recipes` option off stops it offering; `/ele:recipe` still works when typed.

A procedure your teammates or CI must also run belongs in the repository, as a
task target or a doc, so Claude says so instead of drafting a skill only you
would have.
