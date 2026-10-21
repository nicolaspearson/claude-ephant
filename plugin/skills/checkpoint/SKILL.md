---
name: checkpoint
description: Checkpoint work before reporting it done or ready to commit, or when the user asks for a checkpoint or pre-commit review.
---

The session that wrote the code does not review it. It knows what every line was
meant to do, so it reads the diff as confirmation and finds what it intended
rather than what is there. Reviews run in subagents that have seen only the
brief.

Every review runs fresh. No pass is credited from earlier in the session; the one
exception is a tree whose bytes the last checkpoint already reviewed.

## 1. Scope

`<premigration>` is `premigration/` at the plugin root, two levels above this
skill's base directory, here and in the files beside this one.

Per repository the change touches, first:

```sh
<premigration>/checkpoint-reviewed.sh check <root>
```

Covered in every repository: the last checkpoint reviewed these exact bytes, and
staging or committing part of them since does not change that. Say so, repeat its
commit commands if its report is in view, and stop. Run anyway only when the user
asks for a fresh review — the record does not see a changed convention file or
ticket — or for a branch review, which the record never covers.

Otherwise:

```sh
git -C <root> status --porcelain
git -C <root> diff HEAD --stat
```

Read the two status columns:

- **`U` in either, or `AA` / `DD`** — unmerged paths, from a merge, rebase,
  cherry-pick, revert or conflicted stash. Stop and say so; this is not a
  checkpoint.
- **First column `M`, `A`, `R`, `C` or `T`, second `M`, `D` or `T`** — the staged
  copy differs from the working tree. Reviewers read the working tree and a plain
  commit ships the staged copy, so list these paths for the brief and for §8.

List any rename by both its old and new path: §8's commit needs both.

Nothing uncommitted and nothing untracked: say so and stop, this is not a
checkpoint — unless commits sit unpushed, in which case offer a branch review
(below). A clean tree alone is not that — a symlinked or unversioned directory
holds work git will never report, reviewed against a saved copy of what it
replaced, or against the file itself where there is none.

**Branch scope**, when the user asks for a review of the branch or accepts the
offer: read `BRANCH.md` beside this file first; it changes what the sections
below mean.

Build the change list: every tracked file in the diff, every untracked file from
`git status --porcelain`, every changed file outside git. `git diff HEAD` does not
show untracked files, and a review given only the diff reviews none of them — the
files a change adds usually carry its logic. Uncommitted work unrelated to this
change stays off the list.

Run the repository's formatter over the paths on the change list, so work off
the list keeps its bytes. Run its linter over the packages (or directories)
holding them — a Go linter given single files cannot see their siblings and
reports names it cannot resolve — with no fix flag, since a fixing run over a
package rewrites siblings off the list (golangci-lint: `--new-from-rev=HEAD`).
Fix by hand what it reports on lines the change touched; anything else is
pre-existing, left, and named in §9. What these catch never reaches a reviewer
as a finding. Then re-read `git status --porcelain` for those paths: a staged
file the formatter rewrote now differs from its staged copy, and joins the
staged-differently list for the brief and §8.

Freeze each repository's diff into the session scratchpad directory, naming every
path on the change list as its own argument, relative to the root. Leave off any
path the formatter returned to HEAD; it leaves the change list too:

```sh
<premigration>/checkpoint-reviewed.sh freeze <root> <scratchpad> [--base <base>] --start -- <path> <path> ...
```

The script writes `checkpoint-<repo>.diff` and the fingerprint
`checkpoint-<repo>.hash` for §7, `<repo>` being the basename of git's root for
it. It refuses a path with no change and prints the line count and the mode this
root alone implies. It also tells the Stop hook this session's checkpoint is
running, until §11 records it, `close` ends it (Stopping early, below), or two
hours pass. Only this freeze passes `--start`: it starts the run's clock, which a
run abandoned earlier in the session would otherwise keep. §7's freeze continues
the run without it.

**Delta scope** is the freeze's call, not yours. Where the last checkpoint froze
a path on the change list and its record still holds those bytes, the freeze
also writes `checkpoint-<repo>.delta.diff`: each such path against the bytes
that checkpoint reviewed, the rest — and both sides of a rename — against HEAD.
It says how many paths changed since, and the mode it prints is the delta's.
`0 changed`: nothing is new to review; say so, run `close`, and stop. A branch
freeze writes no delta. No delta file: the scope is uncommitted.

§1 is done when every repository the change touches is frozen.

## 2. Gather

Work out the gate §4 starts: what CI runs. Read the CI config —
`.gitlab-ci.yml`, `.github/workflows/`, and the task targets they call — and list
its checks: typecheck and build, lint, tests with their coverage thresholds,
generated-code drift. The repository's local gate is often narrower, and as of
2026-10-15 that gap was the commonest way a defect got past a checkpoint. A check
that cannot run here is reported as unverified, never as passing.

Gather what is settled, for the brief:

- The project's live notes — decisions, dated facts and open questions — and
  earlier sessions on the area the change touches, per the `ele:recall` skill's
  Project context section, `<name>` being the repository's basename and
  `<topic>` the area. Where that skill is not listed, run
  `ele recall --project <name> "<topic>"`, then `ele context --project <name>`.
  A failure there goes in the brief.
- Rulings the user made this session.
- Findings an earlier round left on purpose, with the reason.
- Failures already known to happen at HEAD, each proved in a HEAD export
  (`HEAD-EXPORT.md` beside this file), not assumed.

Gather the paths of every set of conventions in force over the changed files —
paths only, the reviewer reads them:

- The user's `CLAUDE.md`, under `$CLAUDE_CONFIG_DIR` when set, else `~/.claude`,
  with every file it imports or says to read for the changed code.
- The repo root `CLAUDE.md` / `AGENTS.md`, plus any in a directory containing a
  changed file.
- Whatever those point at: `PATTERNS.md`, `CONTRIBUTING.md`, an architecture doc,
  an ADR index, a style guide.
- Linter, formatter and `.editorconfig` settings, so no reviewer duplicates what
  the gate enforces.

If the work came from a ticket, fetch it. A checkpoint that cannot say what was
asked cannot say whether it was delivered.

§2 is done when the CI check list exists and every brief field it gathers has a
value or is stated empty.

## 3. Brief

One brief, shared by every reviewer, every path absolute:

- Each repository root and its frozen diff.
- The scope. Delta: the delta diff too, and the rule that findings fall on
  lines the delta changes, the full diff being context; a finding outside it
  is reported only where the delta's change makes it newly wrong.
- The change list, untracked and outside-git files marked as such.
- The paths §1 found staged differently from the working tree.
- The convention file paths.
- What was asked — the ticket, or the user's request in their words.
- **Settled — do not re-raise**: what §2 gathered, each with its reason. A
  reviewer contradicting one reports it only as a question carrying evidence the
  reason did not have.
- The path §4's gate log is written to.
- **Attack surfaces**: what the author is least sure of. State combinations that
  were never enumerated, paths never exercised, what a user sees when the new
  error fires, a trap the codebase documents. Specific beats thorough — "every
  combination of these four fields present or absent, against where each
  resumes" finds what "check edge cases" does not.

Leave out the author's case for why the code is right.

## 4. Review

**Mode.** The freeze prints the mode each root alone implies. A second
repository or any file outside git makes the change full, whatever each freeze
printed. Small runs one fresh `general-purpose` reviewer that reads both
reviewer files and reports under all three labels; full runs two.

Start the gate in the background as the reviewers launch: the checks §2 listed,
against the tree the reviewers read. Nothing edits that tree until the checks
finish: a fix or a mutation they read fails them spuriously, or passes them on
bytes nobody shipped. Capture to a file with the exit status as its last line —
a log without `gate exit` is still running, not a pass:

```sh
S=<scratchpad>; { <gate command>; echo "gate exit $?"; } > "$S/gate.log" 2>&1
grep -nE 'FAIL|panic|Error|timed out|gate exit' "$S/gate.log"
```

Full mode: launch both in one message so they run in parallel, each as the
subagent type below. Each prompt is the brief plus one line: read the reviewer
file below first and follow it.

The reviewer files sit beside this file, under `reviewers/`. Pass them as
absolute paths — a subagent does not load this skill and cannot resolve them.

| Reviewer              | Subagent type             | Instructions                     |
| --------------------- | ------------------------- | -------------------------------- |
| Bugs                  | `general-purpose`         | `reviewers/bugs.md`              |
| Conventions and prose | `ele:checkpoint-reviewer` | `reviewers/conventions-prose.md` |

`ele:checkpoint-reviewer` runs a cheaper model. Where that type is not listed, use
`general-purpose`.

Without an Agent tool — when this skill itself runs in a subagent, for instance —
run every review in this file inline, one after another, each from its reviewer
file, and say in the report that the author reviewed its own work.

## 5. Verify

A finding is a claim, and a reviewer can be wrong. Before fixing a bugs finding,
check it against the code: trace the input it names to the wrong output, or
reproduce it. A prose or conventions finding needs only the cited line read — as
of 2026-10-15 under 3% of them were refuted. Confirmed, it is fixed below.
Refuted, list it with the reason. Neither, it goes in the report as a question.
A finding against a settled item is not applied on the reviewer's word: it goes
to the user.

Reviewers overlap. Merge a finding two of them raised before verifying it, and
log it once.

Keep a running list for §12: every finding, from a reviewer, the gate, a
mutation or the user, with the stage that caught it, its source and any other
source that raised it, a kebab-case category, its severity, the verdict, whether
the change, a fix in this run or older code introduced it, and a one-line
summary. §12 defines each field. A finding counts once, at the first stage that
caught it.

Then, in the parent, once the gate has finished, fix the confirmed findings. As
of 2026-10-15 three in four re-review findings were something a fix broke,
mostly comments and ordering, so:

- A rewritten comment or doc passes `reviewers/conventions-prose.md`'s
  per-comment tests, and the same claim is searched for elsewhere in the change.
- An appended entry goes where the ordering puts it.
- The formatter and linter run as §1 runs them.

Then:

- **Every new or changed test fails once, in this run.** Mutate the code it
  covers, watch it fail for the stated reason, restore. A mutation run before
  the checkpoint does not count. Confirm the mutation landed before reading a
  non-failure — an anchor that did not match leaves the code untouched, and the
  passing test then looks exactly like one that cannot fail.
- **Every scripted edit landed.** Grep for the new string, not the old one.

## 6. Gate

Run §4's gate again, captured the same way, once §5's fixes and mutations are
done, unless nothing changed since it started. Say so if skipped.

A red check is pre-existing only once it fails the same way at HEAD: read
`HEAD-EXPORT.md` beside this file for the export and for when such a check
counts as green. Any check that neither passes nor counts as green, and is not
being fixed in this run: see Stopping early.

## 7. Re-review the fixes

This step gates the report. Compare each repository's fingerprint with the one §1
saved:

```sh
S=<scratchpad>; <premigration>/checkpoint-reviewed.sh hash <root> | cmp -s - "$S/checkpoint-<repo>.hash"; echo $?
```

`0` in every repository, and no file outside git touched since §1: skip to §8.
Anything else runs this step — a different fingerprint whatever moved it, a
missing saved one, or an edit to a file outside git, which the fingerprint
cannot see.

Freeze again, into a second directory so the first freeze survives; git cannot
diff the fixes alone without a stash or commit. Create the directory first. Name
the change list plus every file a fix created, less any path a fix returned to
HEAD and any untracked file it deleted — the freeze refuses a path with nothing
to diff. Launch one fresh `ele:checkpoint-reviewer`, or `general-purpose` where that
type is not listed, with both frozen diffs, the files the fixes touched and the
findings they fixed. It reads the reviewer files that apply, scoped to what the
fixes changed:

- **Bugs**, if any code changed: what did the fixes break or fail to close?
- **Conventions**, if anything was renamed, moved or reordered.
- **Prose**, if any comment, string or doc changed.

One round. Anything it finds is verified and fixed, or reported — not a third
review.

## 8. Commit message

One line, subject only, in the repo's own style. More than one commit where the
change splits cleanly, and say why it splits.

Print the exact commands for the user to run. Per commit: an `add` of that
commit's untracked paths, then a `commit` naming that commit's paths after `--`,
a rename by both its old and new path. A commit given paths takes their
working-tree copies and leaves anything else staged where it is, so the
staged-differently paths from §1 ship as reviewed and work the user staged for
another commit stays out.

## 9. Report

Label sections by reviewer — **Bugs reviewer**, **Conventions reviewer**,
**Prose reviewer**, the last two splitting the merged reviewer's report. Keep the
labels: the Stop hook matches them when the skill call has scrolled out of its
view.

- The mode, small or full, the scope, and how many reviewers §4 launched.
- Per reviewer: findings confirmed and fixed, refuted with reason, left open.
- Which files each reviewer gave a verdict on. A file on the change list with no
  verdict was not reviewed, and is named as such.
- Convention violations, naming the rule and the file it came from.
- Any file marked denser than its peers whose comments were not swept.
- Tests mutated, and whether each failed.
- **Re-review:** what it found, or that the fingerprint had not moved.
- What was deliberately left, and why, linter findings off the change's lines
  included.
- The gate result, per CI check, naming any that could not run here and any red
  only on failures proven at HEAD.

"Done, tests pass" has skipped the checkpoint.

## 10. Plan

Per repository the change touches, `ele plan list --project <name>` names its
active plans; if it fails, say so and go on. `(cd <root> && ele plan show <path>)`
gives each one's next item, `<path>` as the listing prints it — `ele plan show`
refuses a path outside the cwd's checkout. Read the item the change worked on
from the plan file itself: usually the next one, never assumed to be. No active
plan, or none the change worked on: say so and go on. A plan in a repository the
change does not touch is out of scope: name it, write nothing.

Judge the item against its own text, not the change's intent. Every requirement
it names met, any real-session test among them, and every check §2 listed green
under §6: the change completes it. Propose the tick, with one line under it
saying what was asserted and at which boundary, `N` being the item's position
under Actions:

```markdown
- [x] <item>
  - asserted in the item <N> commit: <what was observed, at which boundary>
```

Anything short of that: propose a dated progress line under the item instead,
`as of <date>: <what is done>` and `next: <what is left>`. Code that exists but
whose behaviour nobody has observed gets a progress line, not a tick.

Write it on approval, before §11, so the record covers it. The plan file joins
the change's commit: reprint §8's commands with its path.

## 11. Record

Once every check passes or counts as green under §6, §7 has run or found
the fingerprint unmoved apart from §10's approved plan write, and every finding
is fixed or reported, per repository:

```sh
<premigration>/checkpoint-reviewed.sh record <root>
```

A record taken before the fixes marks unreviewed bytes as reviewed, and the next
checkpoint stops on it.

It refuses without an open run of this session: none frozen, closed, another
session's, a last freeze over two hours old, or `CLAUDE_CODE_SESSION_ID` unset.
Run §7's fingerprint check first, since a new freeze overwrites the saved
fingerprint; then freeze the change list again and record.

## 12. Log

Log the run, once per checkpoint, against the repository the change is
mostly in. `/ele:checkpoint-stats` tunes this skill from the log, so a finding left
out is a gap the stats cannot show:

```sh
ele checkpoint log <root> <<'EOF'
{"change": {"files": 4, "lines": 120},
 "findings": [{"also_raised_by": ["bugs"], "category": "stale-comment", "origin": "fix",
   "severity": "quality", "source": "prose", "stage": "rereview",
   "summary": "Comment still names the old flag.", "verdict": "fixed"}],
 "mode": "full",
 "mutations": {"failed": 6, "inconclusive": 1, "run": 7},
 "notes": {"approved": 0, "proposed": 0},
 "rereview": "ran",
 "scope": "uncommitted"}
EOF
```

- `also_raised_by`: optional; the other sources that raised a finding merged into
  this one, so the severity table still credits each.
- `change`: files on the change list, and lines added plus removed.
- `mode`: `full`, `small`.
- `mutations`: `run`; `failed` for the reason stated; `inconclusive`, a mutation
  that did not compile or did not land. One that landed and did not bite is a
  finding, stage `mutation`, not a count.
- `notes`: both 0; ele's Stop hook proposes notes outside the checkpoint.
- `origin`: `change`, `earlier` (code older than the change), `fix` (introduced
  by a fix in this run).
- `rereview`: `ran`, `skipped` (§7 found the fingerprint unmoved).
- `scope`: `uncommitted`, `delta` (the freeze wrote a delta), `branch`.
- `severity`: `blocker` (wrong on the main path: wrong output, lost data, a broken
  build), `defect` (wrong only in a reachable edge case), `quality` (correct, but
  misleading a reader, untested, or against a convention other than wording or
  order), `nit` (wording or order alone, misleading no one).
- `source`: free text; the reviewer (`bugs`, or `conventions` or `prose` by the
  merged reviewer's section), the gate check by name, or `user`.
- `stage`: `gate`, `mutation`, `rereview`, `review`, `user`. A finding the user
  raises after this line is written goes in the next checkpoint's record, stage
  `user`, origin `earlier`.
- `verdict`: `deliberate` (confirmed and left on purpose), `fixed`, `open`
  (reported as a question), `refuted`.

`ele checkpoint log` adds `duration`, seconds since the freeze that opened the run, when a
freeze recorded one, and refuses a record the stats would misread, naming the field.

## Stopping early

A checkpoint that stops without a record — a red check §6 does not count as
green, the user calling it off — ends the run, so the Stop hook stops treating it
as in progress:

```sh
<premigration>/checkpoint-reviewed.sh close <root>
```

When the user says this work needs no checkpoint, exempt its paths for the
session. The script refuses a glob that matches an arbitrary path, such as `*`,
`**` or `*.*`:

```sh
<premigration>/checkpoint-reviewed.sh optout <root> "<glob>" ...
```
