---
name: recipe
description: Propose a reusable multi-step procedure this session ran as a user skill, and write it once the user approves; with audit, check the user's skills for commands, flags and versions that no longer exist. Use when the user asks to save a procedure as a recipe or skill, or to audit their skills for stale commands; /ele:wrap offers a recipe too; never start one unasked.
argument-hint: "[the procedure to keep] | audit [skill paths]"
---

# Recipe

Turn a procedure the user will run again into a skill a later session lists, so it is not re-derived. Always ask before writing one, whatever `auto_note_writes` says, unless the user handed its exact words, per Ask: a skill loads into every session of every profile, heavier than a note.

With `audit` as the argument, run Audit below instead of steps 1 to 4.

## 1. Find

Look over the session, or the procedure the argument names, for one that:

- Has three or more steps, with the commands, paths or checks that make each one work
- Will be run again: the user said so, the session ran it more than once, or it is a recurring chore such as a release, an upgrade or a credential rotation
- Was seen to work in this session, steps that failed left out or fixed
- Is not already a skill, a task target or a documented step of the repository

A procedure one repository's teammates or CI must also run belongs in that repository, as a task target or a doc in a language it already uses: say so instead of proposing a skill. Nothing found: say so in one line and stop.

Done when one procedure is chosen, or the one line says none is worth keeping.

## 2. Draft

A recipe goes to every profile, so each lists it. The targets are the `skills` folder of the running profile, `${CLAUDE_CONFIG_DIR:-$HOME/.claude}`, and of each other `$HOME/.claude` or `$HOME/.claude-*` directory holding a `projects` folder, the profiles ele archives, one per real folder, as a profile or its `skills` may be a link to another's:

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

## Audit

Check the user's skills for commands, subcommands, flags and versions that no longer exist here, so a skill is fixed before a session follows it into a failure. Nothing is run but `command -v`, a help, a version command or `man`, each with none of the skill's arguments and stdin closed. A tool that ignores its flags still runs the step the skill describes, so Check runs nothing for a function or alias and reads a script rather than running it; a compiled tool ignoring its flags is the one case it cannot rule out.

### Which skills

Only when the argument is `audit` alone or followed by `SKILL.md` paths; a procedure that starts with the word, such as `audit log rotation`, is a recipe for step 1. With paths, those files, each by `realpath`. With none, every `<name>/SKILL.md` one level down in the folders Draft's listing prints and the folder `ele config get recipe_dir` prints when set, once per real file, as a profile's skill folder or the file itself may link elsewhere: `{ <Draft's listing command>; ele config get recipe_dir; }`, piped into

```sh
while read -r folder; do [ -d "$folder" ] && find -L "$folder" -mindepth 2 -maxdepth 2 -name SKILL.md; done | while read -r file; do realpath "$file"; done | sort -u
```

Plugin skills are left out: their plugins ship and update them. Done when every skill in scope is listed by its real path.

### Claims

Per skill, every command it tells a session to run, in a code span or block: the command, each subcommand and each flag. And every version it pins for a tool installed here, such as `go 1.23`, `node:20` or `GOTOOLCHAIN=go1.23.0`. Leave out a placeholder in `<angle brackets>`, a version stated as history rather than one to use, and a shell builtin. Done when each skill's claims are listed with their lines.

### Check

- Command: `command -v <cmd>`; a command printing nothing is not found, a finding. One printing anything but an absolute path, a function, alias or builtin, is found and nothing more is run for it: its subcommands, flags and version are not checked. A path to a script the skill ships is checked for the file instead
- Package runner: a command that takes a package and runs it, such as `npx`, `npm exec`, `bunx` or `bun x`, `pnpm dlx`, `yarn dlx`, `pipx run`, and `uvx` or `uv tool run`: after the runner's own `command -v` its package, subcommands and flags are not checked, and nothing is run with the package, a help included: the runner would fetch and run it
- Script: a command whose file starts with `#!` is read, never run; its subcommands and flags are checked against its source. Where the file defines none of them and loads another by a path relative to itself (`require`, `source`, `exec`), as npm's and corepack's do, search the folder holding that loaded file for the name in quotes, leaving out `node_modules`, `vendor`, tests and docs, as a dependency or changelog may quote any name: found, it holds by that search; not found, it is not checked, as the name may be built at run time. A loaded path that is absolute, such as `exec /opt/homebrew/bin/foo`, or built at run time leaves its claims not checked, as its folder may be a whole prefix
- Subcommand and flag: run `<cmd> --help </dev/null`, never with the skill's arguments; where `--help` opens a pager, as git's does, `-h` only once the tool's help names `-h` as help. Output that is not a usage or flag list means the tool ignored the flag: stop running it, and mark its remaining claims not checked. Pass a help flag after a subcommand only when the top-level help names the flag or a `help` command, preferring `<cmd> help <subcommand>`; otherwise the subcommand's flags are not checked. One the help does not name is looked up in a help it points to, such as `go help test`'s `see 'go help build'`, then in the man page, `LC_ALL=C man <cmd>-<subcommand> </dev/null | col -b` for a subcommand and then `LC_ALL=C man <cmd>`, empty output meaning no page, the C locale keeping a flag's hyphens ASCII where a UTF-8 one may render them `‐` or `−`; a short help may list only some flags, as `git log -h` does. Named in any of them, it holds; otherwise, a finding. A command not found leaves its subcommands and flags not checked
- Version: the installed tool's `--version`, or its `version` subcommand only when its help names one. A pin older than the installed version, by major or minor, is stale; a tool not installed leaves its pin not checked

Done when each claim is marked holds, a finding with what was read, or not checked with why.

### Report and fix

Per skill, headed by its real path, absolute as `realpath` prints it, never relative: each finding with its line, the claim, and what the check printed, a command not found being a finding; then the claims that hold, each with where it was named (the help, a help it points to, the man page, or a file the script loads); then the claims not checked and why, a missing command's subcommands and flags among them; or "no findings" with how many claims were checked. For each finding with a clear replacement, such as the flag the help now names or the installed version, propose the exact line before and after, and apply the ones the user approves with the edit tool to the real path. A skill whose real path is in a git repository is left changed in its working tree for the user to commit.

Done when every skill in scope is reported and each proposed fix is applied, declined or open.
