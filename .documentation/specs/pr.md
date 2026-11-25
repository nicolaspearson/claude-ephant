# PR

`/ele:pr` drafts the body of a pull or merge request for the branch in the
working directory: a Summary, Evidence and Merge Danger, filled from the
branch's diff, the plan ticks it carries and the test and gate runs the
session made. It drafts only: it prints the body and the command that would
open the request, and never runs a forge write.

Model-invocable, unlike `/ele:write-report`: asking for a PR description in
plain words is the common case, and the skill writes nothing the user has not
seen. An ele plugin skill, not a user skill, because its evidence is ele's:
`ele checkpoint base`, `ele checkpoint check`, plan tick lines and the
`report_voice` option.

## Scope

- The range is `<base>..HEAD`, `<base>` the argument, else the merge-base
  with `origin/HEAD`, else, with no `origin/HEAD`, `ele checkpoint base`; plus
  the working tree's changes where `git status --porcelain` lists any
- The default branch comes before `ele checkpoint base` because a request
  targets it: a branch pushed with `-u` is its own upstream, so
  `ele checkpoint base`'s merge-base is HEAD there and every pushed commit
  would drop out of the body
- A range with no commit and a clean tree has nothing to describe: the skill
  says so, naming the base, and stops

## Checkpoint

- Uncommitted changes get `ele checkpoint check`, run only where status lists
  something, since it exits 1 alike for unreviewed changes and for none. Exit
  1 offers `/ele:checkpoint`; declined, Evidence calls those changes "changed,
  not yet reviewed"
- Committed work is not gated: the user's rules checkpoint before every commit,
  no ele command says whether a commit was checkpointed, and offering a branch
  checkpoint on every run would fire on nearly every run after the work's own.
  `/ele:write-report` treats committed work the same way. The body calls a
  commit reviewed only where a plan tick line in the range or the session
  shows a review of it

## Evidence

Saved into a new directory in the session scratchpad, one file per piece: the
range's diff and log, the working tree's diff with new files, the diff of
`.documentation/plans/` whose added ticks say what the range asserts done, each
test or gate run the session made, and the steps a pull or merge request's CI
runs, release and scheduled jobs left out. A claim
with no file is left out of the body.

- The Gate line names the last gate run over the range at the commit it ran
  at, "with uncommitted changes" where the tree held some, or says "Gate: not run in this session", or, where a later edit made
  every run stale, "Gate: not run since <path> changed"
- The "Left to CI" line names the steps `.github/workflows/` or
  `.gitlab-ci.yml` runs that the session's gate left out, since a green local
  gate is not a green CI. Not "Not run in CI", which reads as steps CI skips.
  No CI config, or no gate run, no line

## Body

- Summary: short prose for a reviewer who knows the codebase but not the
  session, and the least visual that shows the change: pseudocode for a rule,
  a call tree, a file tree, a Mermaid sequence diagram, a `diff` of a shape the
  reviewer knows, or the new block itself. One visual is the norm and two the
  most. Guidance the skill follows, not a check
- Evidence: before and after, a screenshot where the change is seen and the
  session took one, else the test that failed and now passes; then the Gate
  and "Left to CI" lines
- Merge Danger: Door, two-way when a revert restores the old behaviour and
  one-way when it cannot, and Blast Radius, the widest reach the diff supports
- The repository's `GLOSSARY.md`, where one exists, gives the names the body
  uses
- The voice is `report_voice`'s file, else `plugin/reference/report-voice.md`.
  No `pr_voice` option: a PR body is a report to a reviewer, and an option
  costs config, tests and docs rows
- No "Generated with Claude Code" line, overriding Claude Code's PR
  attribution: the body goes to the reviewer as the user's text

## Delivery

- The closing reply holds the body in a fence of `~` longer than any `~` run
  inside it, so its own code blocks survive a copy, and the body file holds it
  unfenced
- The forge comes from `remote.origin.url`: `gh pr create --body-file` for
  GitHub, `glab mr create --description-file` for GitLab, the file's path alone
  for any other. The command is printed, never run: every tracker write is the
  user's to authorise
- The create command opens a request only from a pushed branch other than the
  target, so the skill reads the branch, `origin/HEAD` and the upstream, each
  read printing nothing rather than failing where there is none, and prints
  above it what the user runs first, in order: `git switch -c` on the default
  branch or a detached HEAD, before any commit, since the switch carries
  uncommitted changes and a commit first would land on the default branch; a
  line to commit uncommitted changes through `/ele:checkpoint`, since a
  request opened before the commit lacks them; then `git push -u origin` after
  a switch or where the branch has no upstream of its own, another branch's or
  a gone one included, else `git push` where there is anything to push. It
  runs none: git writes are the user's
- No ele note is written. The Stop hook exempts the run's own calls as it does
  `/ele:write-report`'s: work before it stays pending for the next notes ask

## Attribution

Adapted from the `pr` skill of Matt Pocock's mattpocock-skills 1.3.1, under
MIT. The notice travels in `plugin/skills/pr/references/CREDITS.md`, since the
skill lint allows no other file at the skill's top level, the same way
internal/ticket's difflib carries its PSF licence beside it. The Summary visuals menu is written fresh,
not adapted: the source's `CREDITS.md` credits its menu to Dex Horthy's
`show-me` skill, whose licence was
not checked.

## In the plugin

- `plugin/skills/pr/SKILL.md` and `references/CREDITS.md`
- A docs page of its own, `docs/pr.md`
