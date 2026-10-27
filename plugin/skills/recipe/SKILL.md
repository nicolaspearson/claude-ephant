---
name: recipe
description: Propose a reusable multi-step procedure this session ran as a user skill, and write it once the user approves. Use when the user asks to save a procedure as a recipe or skill; /ele:wrap offers one too; never start one unasked.
argument-hint: "[the procedure to keep]"
---

# Recipe

Turn a procedure the user will run again into a skill a later session lists, so it is not re-derived. Always ask before writing one, whatever `auto_note_writes` says, unless the user handed its exact words, per Ask: a skill loads into every session of every profile, heavier than a note.

## 1. Find

Look over the session, or the procedure the argument names, for one that:

- Has three or more steps, with the commands, paths or checks that make each one work
- Will be run again: the user said so, the session ran it more than once, or it is a recurring chore such as a release, an upgrade or a credential rotation
- Was seen to work in this session, steps that failed left out or fixed
- Is not already a skill, a task target or a documented step of the repository

A procedure one repository's teammates or CI must also run belongs in that repository, as a task target or a doc in a language it already uses: say so instead of proposing a skill. Nothing found: say so in one line and stop.

Done when one procedure is chosen, or the one line says none is worth keeping.

## 2. Draft

A recipe goes to every profile, so each lists it. The targets are the `skills` folder of `$HOME/.claude` and of each `$HOME/.claude-*` directory holding a `projects` folder, the profiles ele archives, one per real folder, as a profile or its `skills` may be a link to another's:

```sh
find "$HOME" -maxdepth 1 \( -name .claude -o -name '.claude-*' \) | while read -r profile; do [ -d "$profile/projects" ] || continue; if [ -d "$profile/skills" ]; then (cd "$profile/skills" && pwd -P); else echo "$(cd "$profile" && pwd -P)/skills"; fi; done | sort -u
```

Pick a short kebab-case name saying what it does, such as `release-cli` or `rotate-staging-token`. Check each target for `<name>`; one there in any target is never overwritten, so pick another name or, where it covers the same procedure, say so and stop.

Draft the whole file:

- Front matter: `name` and a `description` saying what it does and when to use it, the trigger words the user would say
- A one-line purpose, then numbered steps, each with the exact command or check as it ran, placeholders in `<angle brackets>` for what changes per run
- A last line saying what is observable when it is done
- No secrets, tokens, or ids of this session; nothing the steps do not need

Done when the file's exact words are drafted.

## 3. Ask

Ask with AskUserQuestion, loading it through ToolSearch if it is deferred: `question` the target paths and the drafted file, `header` `recipe`, options `Write` and `Skip`. An `Other` that edits the draft is written as edited; any other `Other` is settled with the user first. Only a user who handed the whole file's exact words and asked for it to be saved has approved it: then skip the question. Asking for a recipe, as `/ele:recipe` or "keep this", is not approval of a draft not yet seen.

Done when the recipe is answered `Write` or `Skip`.

## 4. Write

On `Write`, create `<target>/<name>/SKILL.md` in each target with the drafted words, yourself, never printing commands for the user. Report each path, and that a session started after it, in any profile, lists the skill as `/<name>`; a profile added later does not get it.

Done when each path written is reported, with any that failed and why, or the recipe is reported skipped.
