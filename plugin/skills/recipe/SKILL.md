---
argument-hint: "[the procedure to keep] | audit [skill paths]"
description: Propose a reusable multi-step procedure this session ran as a user skill, and write it once the user approves; with audit, check the user's skills for commands, flags and versions that no longer exist. Use when the user asks to save a procedure as a recipe or skill, or to audit their skills for stale commands; /ele:wrap offers a recipe too; never start one unasked.
name: recipe
---

# Recipe

Turn a procedure the user will run again into a skill a later session lists, so it is not re-derived. Always ask before writing one, whatever `auto_note_writes` says, unless the user handed its exact words, per Ask: a skill loads into every session of every profile, heavier than a note.

With `audit` as the argument, alone or followed by `SKILL.md` paths, read `references/AUDIT.md` and follow it instead of steps 1 to 4. A procedure that starts with the word, such as `audit log rotation`, is a recipe for step 1.

## 1. Find

Look over the session, or the procedure the argument names, for one that:

- Has three or more steps, with the commands, paths or checks that make each one work
- Will be run again: the user said so, the session ran it more than once, or it is a recurring chore such as a release, an upgrade or a credential rotation
- Was seen to work in this session, steps that failed left out or fixed
- Is not already a skill, a task target or a documented step of the repository

A procedure one repository's teammates or CI must also run belongs in that repository, as a task target or a doc in a language it already uses: say so instead of proposing a skill. Nothing found: say so in one line and stop.

Done when one procedure is chosen, or the one line says none is worth keeping.

## 2. Draft

A recipe goes to every profile, so each lists it. The targets are the `skills` folder of the running profile, `${CLAUDE_CONFIG_DIR:-$HOME/.claude}`, and of each other `$HOME/.claude` or `$HOME/.claude-*` directory holding a `projects` folder, one per real folder, as a profile or its `skills` may be a link to another's:

```sh
{ (cd "${CLAUDE_CONFIG_DIR:-$HOME/.claude}" 2>/dev/null && pwd -P); find "$HOME" -maxdepth 1 \( -name .claude -o -name '.claude-*' \) | while read -r profile; do [ -d "$profile/projects" ] && (cd "$profile" && pwd -P); done; } | sort -u | while read -r profile; do if [ -d "$profile/skills" ]; then (cd "$profile/skills" && pwd -P); else echo "$profile/skills"; fi; done | sort -u
```

Run `ele config get recipe_dir`. When it prints a path, that folder holds the recipe once and each target links to it; take it by its real path, `(cd "<path>" && pwd -P)`, as the targets are listed by theirs; a path `cd` cannot enter stops the recipe, reported with the path. When it prints an empty line, or nothing on stdout, as an `ele` older than the option does, each target holds its own copy, and step 4's report says so.

Pick a short kebab-case name saying what it does, such as `release-cli` or `rotate-staging-token`. Never overwrite `<name>`: with the folder set, one in the folder means pick another name; one in a target leaves that target alone, named in the report. Without the folder, one in any target means pick another name. Where a name that means picking another covers the same procedure, say so and stop.

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

On `Write`, run the writes yourself rather than printing them for the user: with the folder set, create `<folder>/<name>/SKILL.md` with the drafted words, then link each target to it with the command below, which skips the folder itself and a target already holding `<name>`, as `ln -s` onto an existing folder links inside it; without it, create `<target>/<name>/SKILL.md` in each target. Report each path written or linked, each target left alone, and that a session started after it, in any profile, lists the skill as `/<name>`; a profile added later does not get it. Without the folder, the report's first line says `recipe_dir` was unset, or unknown to this `ele`, so each profile got its own copy. It then names `ele config set recipe_dir <folder>`, typed in a terminal, as the way to write later recipes once to that folder, and says to update `ele` first when the option was unknown. A user who keeps skills in one place otherwise misses the copy landing outside it.

```sh
for target in <target> ...; do if [ "$target" = "<folder>" ] || [ -e "$target/<name>" ] || [ -L "$target/<name>" ]; then echo "left alone: $target"; else mkdir -p "$target" && ln -s "<folder>/<name>" "$target/<name>" && echo "linked: $target/<name>"; fi; done
```

Done when each path written or linked is reported, with any that failed and why, and, without the folder, the first line saying `recipe_dir` was unset or unknown and how to set it, or the recipe is reported skipped.
