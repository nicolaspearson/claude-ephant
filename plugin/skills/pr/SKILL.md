---
argument-hint: "[<base>]"
description: Draft a pull or merge request body for the branch, its Summary, Evidence and Merge Danger filled from the diff, the plan ticks and the session's test and gate runs, in your report voice, each claim checked against that evidence by a reviewer before you see it. Draft only: it prints the create command and never runs it. Use when asked to write, draft or describe a PR or MR body.
name: pr
---

# PR

Draft only: the skill never runs `gh pr create`, `glab mr create` or any
other forge write, and writes no ele note. It prints the body and the command
that would open the request, for the user to run. The body carries no
"Generated with Claude Code" line, whatever an attribution rule elsewhere
asks: the user's own text goes to the reviewer.

One draft at a time, shared with `/ele:write-report`. An earlier `/ele:pr` or
report is in progress while its turn still runs or its own review still runs
in the background; one that stopped, at a checkpoint offer, an error or the
user's word, holds nothing up. A checkpoint this session started over the
changes in scope holds a new draft too, until it has recorded or closed. The
user's word to run a new one anyway overrides either hold. A new `/ele:pr`
held does no work of its own:

- Arriving mid-turn: that turn carries on. After a report written to a file,
  the new one runs in the same turn, from step 1. After an `/ele:pr` body or
  a Slack or WhatsApp block, each of which must close its turn, the line
  under it names the new one for the user to run next
- Arriving in a new turn: one line naming what holds it, and to run it again
  once that is done, is the whole turn

`<root>` is `git rev-parse --show-toplevel` of the working directory.
`<scratch>` is a new directory in the session scratchpad:
`mkdir <scratchpad>/pr-<time>`, `<time>` from `date +%Y%m%d-%H%M%S`, its path
written out in each later command, since Claude Code's Bash tool keeps no
variable between calls. `mkdir` without `-p` fails on an existing directory
rather than reuse its evidence. Then `mkdir <scratch>/evidence`.

Everything shown to the user before the closing reply goes **inside an
AskUserQuestion call**, loaded through ToolSearch if it is deferred. As of
2026-10-26, text meant to print before a further call in the same turn does not
reliably reach the user, while a call's text does.

## 1. Scope

- **Base**: the `<base>` argument; else, where
  `git -C <root> rev-parse --verify --quiet refs/remotes/origin/HEAD` prints a
  commit, `git -C <root> merge-base HEAD origin/HEAD`, the default branch being
  what a request targets; else `ele checkpoint base <root>`, the merge-base
  with the upstream. Not `ele checkpoint base` first: a branch pushed with
  `-u` is its own upstream, so its merge-base is HEAD and every pushed commit
  would drop out
- **Range**: `<base>..HEAD`, plus the working tree's changes where
  `git -C <root> status --porcelain` lists any
- **Nothing to describe**: no commit in the range and a clean tree. Say so in
  one line, naming the base and that a `<base>` argument sets another, and stop

Done when the base is a commit and the range holds a commit or a change.

## 2. Checkpoint

- `git -C <root> status --porcelain` lists nothing: committed work is
  described by its commits, and there is nothing to gate
- It lists changes: `ele checkpoint check <root>`. Exit 0, the changes are
  reviewed. Exit 1, offer `/ele:checkpoint` first, in an AskUserQuestion call.
  Accepted, stop here; the user runs `/ele:pr` again once it is done.
  Declined, every such change is written in Evidence as "changed, not yet
  reviewed", never as done or fixed

The check exits 1 alike for unreviewed changes and for none, so it runs only
where status lists something. It exits 1 too when over 200 paths are changed,
which it names; a checkpoint cannot cover so many, so those changes are marked
not yet reviewed without the offer.

Committed work is not gated. The body calls a commit reviewed only where a
plan tick line in the range or this session's conversation shows a review of
it, saved as step 3 says; otherwise it says nothing of review.

Done when the tree is clean, covered, or its changes are marked not yet
reviewed.

## 3. Evidence

Write each piece the body draws on into `<scratch>/evidence/`, one file each,
named for what it holds:

- `range.diff`: `git -C <root> diff <base>..HEAD`
- `range.log`: `git -C <root> log --format='%h %s%n%b' <base>..HEAD`
- `worktree.diff`, where step 2 found changes: `git -C <root> diff HEAD`, and
  for each path `git -C <root> ls-files --others --exclude-standard` prints,
  `git -C <root> diff --no-index /dev/null <path>`, which exits 1 when it
  prints a diff
- `plans.diff`: `git -C <root> diff <base>..HEAD -- .documentation/plans/`.
  Its added `- [x]` lines and the tick line under each are what the range
  asserts done, at which boundary and at which commit; a deleted plan file is
  a closed plan
- Each test or gate run this session made, one file per run: the command, the
  commit it ran at, whether the working tree held changes then, the paths in
  the range the session edited after it, or none, its exit code and its
  output, as the conversation or its captured log holds it. A run made before
  a later edit to a path in the range is that edit's past, not its evidence
- `checkpoint.txt`, where step 2 found changes: the `ele checkpoint check`
  exit code and line, and the offer's outcome: accepted, declined, or not
  made for over 200 paths
- `reviews.txt`, where this session reviewed a commit in the range: one line
  a commit, its short hash and what showed the review, such as the
  `ele checkpoint record` line, since the reviewer sees no conversation
- `ci.txt`, wherever `.github/workflows/` or `.gitlab-ci.yml` exists: each
  step a pull or merge request's CI runs, one line each with its file and
  job: the workflows under `.github/workflows/` that `pull_request` triggers,
  and the jobs of `.gitlab-ci.yml` that run in a merge request pipeline. A
  release, deploy or scheduled job is not a check on the change. No such job,
  a file saying so. Neither present, no file

Save nothing secret: a token, key or password in an output is cut before it is
written, and named only by its shape.

Done when every claim the body will make has a file under
`<scratch>/evidence/`, and `ci.txt` exists wherever a CI config does.

## 4. Draft

Read, each whole:

- The voice file: `ele config get report_voice`, or, empty,
  `reference/report-voice.md` in the plugin root, the directory holding
  `skills/`
- `<root>/GLOSSARY.md` where one exists. A thing it names is called by its
  name, never a synonym

Write `<scratch>/body.md`, this shape, no preamble above Summary and no
footer:

```markdown
## Summary

<one or two sentences, and the visual that carries the change>

## Evidence

- **Before:** <the failing run, or the old output>
  **After:** <the passing run, or the new output>
- Gate: <command> passed at <sha>
- Left to CI: <step>

## Merge Danger

**Door:** <one-way or two-way>

<what makes it so, where not plain>

**Blast Radius:** <a word or a short phrase>

<who or what a bad merge reaches>
```

Every claim rests on a file under `<scratch>/evidence/`; one with none is left
out, never hedged in. Changes step 2 marked are written "changed, not yet
reviewed".

### Summary

Prose stays short: what changed and why, for a reviewer who knows the codebase
but not this session. Then a visual, the least that shows the change, chosen by
what the change moves:

- A rule or an algorithm: the new rule as a few lines of pseudocode
- Who calls whom: an indented call tree, one call a line
- Where code lives: a file tree cut to the directories touched, a short note
  beside each entry that changed
- Messages between parts: a Mermaid `sequenceDiagram`
- A small step in a shape the reviewer knows: a `diff` of that shape, the call
  tree, file tree or steps, not of the source

  ```diff
   handleUpload
     checkQuota
  -  storeFile
  +  scanFile
  +  storeFile
  ```

- Mostly new code the reviewer will copy or own: the block itself

One visual is the norm and two the most. Each sits under the sentence it
illustrates, and holds only the names that sentence needs. A tree is drawn
with indentation, never box-drawing characters, which the voice's typed
punctuation rules out.

### Evidence

Before and after, from the evidence files. A screenshot beats a run for a
change a user sees, where the session took one. Otherwise a run: the test that
failed before the change and passes after it, named, with the line of output
that shows each.

- **Gate**: the last gate run this session made over the range, as
  `Gate: <command> passed at <sha>`, `<sha>` the HEAD it ran at, followed by
  "with uncommitted changes" where the tree held some when it ran; or failed,
  with the failure. None: `Gate: not run in this session`. Only runs a later
  edit made stale: `Gate: not run since <path> changed`
- **Left to CI**: the steps in `ci.txt` that the gate run left out, which
  only CI will show. No CI config, no gate run, or nothing left out: no line

### Merge Danger

- **Door**: two-way when reverting the merge restores the old behaviour; one-way
  when it cannot, such as a schema migration, deleted or rewritten data, a
  published format, API or file others read, or a release cut from it
- **Blast Radius**: what a bad merge reaches: one command, the hooks every
  session runs, other repositories, every user on upgrade. Name the widest
  reach the diff supports

Done when `<scratch>/body.md` holds the three sections and every claim in it
has its evidence file.

## 5. Review

Launch one `ele:report-reviewer`, its prompt the brief below plus one line:
read the reviewer file, by absolute path, first and follow it. The brief, every
path absolute:

- **Reviewer file**: `reviewers/pr.md` in this skill's directory
- **Body**: `<scratch>/body.md`
- **Evidence files**: each file under `<scratch>/evidence/`
- **Report only**: edit nothing

Then:

- Fix each finding in `<scratch>/body.md`. An unbacked claim is dropped, never
  softened; a secret is cut
- Review once more, with the same brief over the new file
- A finding still open after the second review is shown to the user beside the
  draft, in step 6
- A Question is not a finding: it is shown to the user beside the draft, in
  step 6, and the body is not edited for it. An edit made anyway, after a
  clean review or after the second, is reviewed once more before delivery;
  a finding that review leaves is shown beside the draft, in step 6, and the
  body is not edited again

Claude Code may run the reviewer in the background, its report waking the
session in a later turn. A turn that ends while a review runs ends with one
line: the review is running, the draft follows when it returns, and a second
draft waits until then. The turn the review's report wakes goes on with this
step: its findings fixed, or step 6 once the last review is in.

Done when the last review has reported and the body has not changed since it.

## 6. Deliver

- **Forge**: `git -C <root> config --get remote.origin.url`. A `github.com` URL
  takes `gh pr create --title '<title>' --body-file <scratch>/body.md`; a GitLab
  one, `glab mr create --title '<title>' --description-file <scratch>/body.md`;
  any other, or none, only the body file's path. `<title>` is the change in
  under 70 characters, a `'` in it written `'\''`
- **Branch**: the create command opens a request only from a pushed branch
  that is not the target, holding the commits the body describes. Read, each
  of which prints nothing rather than failing where there is nothing:
  - `<branch>`: `git -C <root> branch --show-current`, empty on a detached HEAD
  - The default branch: where
    `git -C <root> rev-parse --verify --quiet refs/remotes/origin/HEAD` prints
    a commit, `git -C <root> rev-parse --abbrev-ref origin/HEAD` with its
    `origin/` prefix cut, whatever `<base>` step 1 took; otherwise unknown,
    and a line under the block says the target is the forge's default
  - The upstream:
    `git -C <root> for-each-ref --format='%(upstream:short) %(upstream:track)' refs/heads/<branch>`.
    It counts only as `origin/<branch>` and not `[gone]`: empty, another
    branch's, or gone, the branch has no upstream of its own

  Print in the block, above the create command, what the user runs first, in
  this order, running none of it, since git writes are theirs:
  1. `<branch>` empty or the default branch: `git switch -c <new>`, `<new>` a
     few words of the title in lowercase ASCII letters, digits and `-`. It
     carries uncommitted changes with it, so it comes before any commit
  2. Uncommitted changes in the range: a `# commit the changes first:
     /ele:checkpoint prints the commands` line, since a request opened before
     the commit lacks them
  3. A push: after step 1's switch, or with no upstream of its own,
     `git push -u origin <new or branch>`; else, with commits the upstream
     lacks (`git -C <root> rev-list --count '@{upstream}..HEAD'` above 0) or
     uncommitted changes to be committed, `git push`; else none
- **Closing reply**: one line, then the body in a fence of `~` characters
  longer than any `~` run inside it and at least three, so its own code blocks
  survive a copy; then the commands in a `sh` block; then each change marked
  not yet reviewed, each finding step 5 left open, each reviewer Question,
  and any `/ele:pr` or report held mid-turn named for the user to run next.
  The body file holds the body unfenced

The files under `<scratch>` stay until the scratchpad goes; nothing else is
written.

Done when the body and the command are in the closing reply.

## Errors

When a call fails, follow `reference/errors.md` in the plugin root, with these
rows added; stop and report in the running conversation.

| Error contains | Class | One fix |
| --- | --- | --- |
| `report-reviewer' not found` | Plugin agent not loaded | Stop: a reviewer with Bash could be steered by a commit message or a changed file; say the plugin needs reloading |
| `not a git repository` | Not in a repository | Stop: say a PR body needs a branch to describe |
| `name the base` | No base to measure from | Ask for a `<base>` ref in an AskUserQuestion call |
| `unknown revision` from a read of the `<base>` argument | A `<base>` that names no commit | Ask for another in an AskUserQuestion call |
