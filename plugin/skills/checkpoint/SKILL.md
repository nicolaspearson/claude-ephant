---
name: checkpoint
description: Checkpoint work before reporting it done or ready to commit, or when the user asks for a checkpoint or pre-commit review.
---

The session that wrote the code does not review it. It knows what every line was
meant to do, so it reads the diff as confirmation and finds what it intended
rather than what is there. Reviews run in subagents that have seen only the
brief.

Every review runs fresh: no pass is credited from earlier in the session, except
for a tree §1 finds covered or a delta of `0 changed`.

## 1. Scope

Per repository the change touches, first:

```sh
ele checkpoint check <root>
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
  copy differs from the working tree; list these paths for the brief and for §9.

List any rename by both its old and new path.

Nothing uncommitted and nothing untracked: say so and stop, this is not a
checkpoint — unless commits sit unpushed, in which case offer a branch review
(below). A clean tree alone is not that — files outside git, in a symlinked or
unversioned directory, hold work git will never report, reviewed against a saved
copy of what they replaced, or against the file itself where there is none.

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
pre-existing, left, and named in §10. What these catch never reaches a reviewer
as a finding. Then re-read `git status --porcelain` for those paths: a staged
file the formatter rewrote now differs from its staged copy, and joins the
staged-differently list for the brief and §9.

Freeze each repository's diff into the session scratchpad directory, naming every
path on the change list as its own argument, relative to the root. Leave off any
path the formatter returned to HEAD; it leaves the change list too:

```sh
ele checkpoint freeze <root> <scratchpad> [--base <base>] --start -- <path> <path> ...
```

The command writes `checkpoint-<repo>.diff` and the fingerprint
`checkpoint-<repo>.hash` for §8, `<repo>` being the basename of git's root for
it. It refuses a path with no change and prints the line count and the mode this
root alone implies.

**The run.** This freeze opens the run, and the Stop hook treats this session's
checkpoint as in progress while it is open. Only this freeze passes `--start`:
it starts the run's clock, which a run abandoned earlier in the session would
otherwise keep; every later freeze, one after `record` included, keeps the clock
until §12's log. Each freeze keeps the run open for two hours; §11's `record` or
Stopping early's `close` ends it. A run is this session's alone: another
session's checkpoint of the same repository neither ends nor times it.
`record` refuses without an open run of this session: none frozen, closed, a
last freeze over two hours old, or `CLAUDE_CODE_SESSION_ID` unset.

**Delta scope**: when the freeze prints `changed since the last checkpoint`,
read `DELTA.md` beside this file; it can end the checkpoint here.

§1 is done when every repository the change touches is frozen, its freeze naming
every path on the change list, and the staged-differently and rename lists exist
or are stated empty.

## 2. Gather

Work out the gate §4 starts: what CI runs. Read the CI config —
`.gitlab-ci.yml`, `.github/workflows/`, and the task targets they call — and list
its checks: typecheck and build, lint, tests with their coverage thresholds,
generated-code drift. The repository's local gate is often narrower, and as of
2026-10-15 that gap was the commonest way a defect got past a checkpoint. A check
that cannot run here is reported as unverified, never as passing.

With a file under `.documentation/plans/` on the change list, the gate also runs
`(cd <root> && ele plan check <path>)` for each one still on disk. For a change
list of plan files alone, §2's check list is those runs plus any CI check that
reads `.documentation/`, such as a docs lint; §10 names the rest as not run,
not as unverified. It exits 0 on warnings, so read them from the log: one on a
line the change touched is a finding, stage `gate`, source `ele plan check`; any
other, a warning with no line included, is named in §10.

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

If the work came from a ticket, fetch it; from a plan item, read it from the
plan file. A checkpoint that cannot say what was asked cannot say whether it was
delivered.

§2 is done when the CI check list exists and every brief field it gathers has a
value or is stated empty.

## 3. Brief

One brief, shared by every reviewer, every path absolute:

- Each repository root and its frozen diff.
- The scope. Delta: as `DELTA.md` says.
- The change list, untracked and outside-git files marked as such.
- The paths §1 found staged differently from the working tree.
- The convention file paths.
- What was asked — the ticket, or the user's request in their words. A plan
  item or fetched ticket is quoted whole: the item's line and body, or the
  ticket's body and comments. Say which of its requirements this change sets
  out to deliver, where it delivers only some.
- **Observed**: with a quoted item, what was run or measured toward it — a
  real-session run, a measurement — and where its output is. This is evidence,
  not the author's case, and the spec reviewer has no other way to see it.
- **Settled — do not re-raise**: what §2 gathered, each with its reason.
  Settled items are context: a reviewer contradicting one reports it only as a
  question carrying evidence the reason did not have.
- **Report only**: reviewers edit nothing and run no gate, test suite or build.
  The parent runs the gate; its log is at `<path>`, finished once it ends in
  `gate exit`.
- **Errors**: `<absolute path of the plugin root's reference/errors.md>`, read
  when a call fails.
- **Attack surfaces**: what the author is least sure of. State combinations that
  were never enumerated, paths never exercised, what a user sees when the new
  error fires, a trap the codebase documents. Specific beats thorough — "every
  combination of these four fields present or absent, against where each
  resumes" finds what "check edge cases" does not.

The brief carries facts, quotes and doubts; the reviewer builds its own case for
the code.

§3 is done when every field above has a value or is stated empty, every path is
absolute, and Attack surfaces names at least one specific input, state or path.

## 4. Review

**Mode.** The freeze prints the mode each root alone implies. A second
repository or any file outside git makes the change full, whatever each freeze
printed. Small runs one fresh `general-purpose` reviewer that reads the reviewer
files and reports under each of their labels; full runs one reviewer per file.

**Spec.** The spec reviewer runs only when the brief quotes a plan item or
fetched ticket: in full mode as a third reviewer, launched with the other two;
in small mode its file, `reviewers/spec.md`, joins the single reviewer's, which
then also reports under §10's Spec reviewer label.

**Launch.** Full mode: launch them all in one message so they run in parallel,
each as the subagent type below. Each prompt is the brief plus one line: read
the reviewer file below, by absolute path, first and follow it. The reviewer
files are under `reviewers/` beside this file.

| Reviewer              | Subagent type             | Instructions                     |
| --------------------- | ------------------------- | -------------------------------- |
| Bugs                  | `general-purpose`         | `reviewers/bugs.md`              |
| Conventions and prose | `ele:checkpoint-reviewer` | `reviewers/conventions-prose.md` |
| Spec                  | `ele:checkpoint-reviewer` | `reviewers/spec.md`              |

Where `ele:checkpoint-reviewer` is not listed, use `general-purpose`.

Without an Agent tool — when this skill itself runs in a subagent, for instance —
run every review in this file inline, one after another, each from its reviewer
file, and say in the report that the author reviewed its own work.

**Gate.** Start it in the background as the reviewers launch: the checks §2
listed, against the tree the reviewers read. Edits wait for `gate exit`: a fix
or a mutation they read fails them spuriously, or passes them on bytes nobody
shipped. Capture to a file with the exit status as its last line:

```sh
S=<scratchpad>; { <gate command>; echo "gate exit $?"; } > "$S/gate.log" 2>&1
grep -nE 'FAIL|panic|Error|timed out|gate exit' "$S/gate.log"
```

§4 is done when every reviewer the mode and the spec rule call for has reported
— bugs and conventions-prose each with a Verdicts list naming every file on the
change list, spec with a verdict per requirement — and the gate log exists.

## 5. Verify

Before fixing a bugs finding, check it against the code: trace the input it
names to the wrong output, or reproduce it. A prose or conventions finding needs
only the cited line read — as of 2026-10-15 under 3% of them were refuted. A
not-reused finding also needs the existing code it names read. A spec finding is
checked against the item's quoted words and the diff: a missing requirement by a
search of the code for it, or of the brief's Observed field where it needs
something observed; an unasked change by naming the requirement that needed it,
if any.

Give each finding one of §12's verdicts:

- **`deliberate`** — confirmed, and a missing or partial requirement the brief
  says this change does not set out to deliver: left to §7's `next:` line.
- **`fixed`** — confirmed, and fixed below.
- **`open`** — neither confirmed nor refuted: reported as a question. A
  confirmed unasked change, or a finding against a settled item, goes to the
  user and stays `open` until they rule: fixing an unasked change means
  reverting work they may want kept, and a settled item is not overturned on a
  reviewer's word.
- **`refuted`** — listed in the report with the reason.

Reviewers overlap. Merge a finding two of them raised before verifying it, and
log it once.

Keep a running list for §12 of every finding — from a reviewer, the gate, a
mutation or the user — with the fields §12 logs. A finding counts once, at the
first stage that caught it.

Then, in the parent, once the gate has finished, fix the confirmed findings. As
of 2026-10-15 three in four re-review findings were something a fix broke,
mostly comments and ordering, so:

- A rewritten comment or doc passes `reviewers/conventions-prose.md`'s
  per-comment tests, and the same claim is searched for elsewhere in the change.
- An appended entry goes where the ordering puts it.
- The formatter and linter run as §1 runs them.

Then:

- **Every new or changed test goes red once, in this run.** Mutate the code it
  covers, watch it go red for the stated reason, restore. A mutation run before
  the checkpoint does not count. Confirm the mutation landed before reading a
  non-failure — an anchor that did not match leaves the code untouched, and the
  passing test then looks exactly like one that cannot fail.
- **Every scripted edit landed.** Grep for the new string, not the old one.

§5 is done when every finding on the running list has a verdict, every confirmed
one is fixed or routed as above, every new or changed test has gone red in this
run, and every scripted edit's new string is found.

## 6. Gate

Run §4's gate again, captured the same way, once §5's fixes and mutations are
done, unless nothing changed since it started. Say so if skipped.

A red check is pre-existing only once it fails the same way at HEAD: read
`HEAD-EXPORT.md` beside this file for the export and for when such a check
counts as green. Any check that neither passes nor counts as green, and is not
being fixed in this run: see Stopping early.

§6 is done when the latest gate log ends in `gate exit` and every §2 check is
passing, counted green per `HEAD-EXPORT.md`, unverified, or handed to Stopping
early.

## 7. Plan

Per repository the change touches, `ele plan list --project <name>` names its
active plans; if it fails, say so and go on. `(cd <root> && ele plan show <path>)`
gives each one's next item, `<path>` as the listing prints it — `ele plan show`
refuses a path outside the cwd's checkout. Read the item the change worked on
from the plan file itself: usually the next one, never assumed to be. No active
plan, or none the change worked on: say so and go on. A plan in a repository the
change does not touch is out of scope: name it, write nothing.

When the change worked on an active plan's item: read `PLAN.md` beside this
file; it proposes the item's tick or progress line, writes it on approval, and
makes the write a fix §8 re-reviews.

§7 is done when each touched repository's plan state is reported — no active
plan, no item worked on, or the item with a proposed tick or progress line — and
any approved write is in the file, with `ele plan check` and any CI check
reading `.documentation/` run since and no warning on its line left open.

## 8. Re-review the fixes

This step gates the report. Compare each repository's fingerprint with the one §1
saved:

```sh
S=<scratchpad>; ele checkpoint hash <root> | cmp -s - "$S/checkpoint-<repo>.hash"; echo $?
```

`0` in every repository, and no file outside git touched since §1: skip to §9.
Anything else runs this step — a different fingerprint whatever moved it, a
missing saved one, or an edit to a file outside git, which the fingerprint
cannot see.

Freeze again, into a second directory so the first freeze survives; git cannot
diff the fixes alone without a stash or commit. Create the directory first. Name
the change list plus every file a fix created, less any path a fix returned to
HEAD and any untracked file it deleted — the freeze refuses a path with nothing
to diff. Launch one fresh `ele:checkpoint-reviewer`, Bugs included on purpose,
or `general-purpose` where that type is not listed, with both frozen diffs, the
files the fixes touched, the findings they fixed, the brief's Settled, Report
only and Errors fields and, when the spec reviewer ran or §7 wrote, the item
quoted whole from the plan file, with what the change sets out to deliver and
the brief's Observed field. It reads the reviewer files that apply, scoped to
what the fixes changed:

- **Bugs**, if any code changed: what did the fixes break or fail to close?
- **Conventions**, if anything was renamed, moved or reordered, or a fix closed
  a Shape finding.
- **Prose**, if any comment, string or doc changed.
- **Spec**, if a fix closed a spec finding or touched what the item names, or §7
  wrote: does each requirement it reached still hold? For §7's write: does each
  claim the line makes hold against the item's done-when, the diff and the
  Observed field?

One round: anything it finds is verified and then fixed or reported.

§8 is done when every fingerprint matched, or the re-reviewer reported on each
reviewer file that applied and each of its findings has a verdict.

## 9. Commit message

One line, subject only, in the repo's own style. More than one commit where the
change splits cleanly, and say why it splits.

Print the exact commands for the user to run. Per commit: an `add` of that
commit's untracked paths, then a `commit` naming that commit's paths after `--`,
a rename by both its old and new path. A commit given paths takes their
working-tree copies and leaves anything else staged where it is, so the
staged-differently paths from §1 ship as reviewed and work the user staged for
another commit stays out.

§9 is done when every path on the change list appears in exactly one printed
commit: both sides of each rename, and each untracked path also in an `add`.

## 10. Report

Label sections by reviewer — **Bugs reviewer**, **Conventions reviewer**,
**Prose reviewer**, the last two splitting the merged reviewer's report, and
**Spec reviewer** when it ran. Keep the labels: the Stop hook matches the Bugs
and Prose ones when the skill call has scrolled out of its view.

- The mode, small or full, the scope, and how many reviewers §4 launched.
- Per reviewer: findings confirmed and fixed, refuted with reason, left open.
- Which files each reviewer gave a verdict on. A file on the change list with no
  verdict was not reviewed, and is named as such.
- Convention violations, naming the rule and the file it came from.
- Any file marked denser than its peers whose comments were not swept.
- Tests mutated, and whether each failed.
- **Re-review:** what it found, or that the fingerprint had not moved.
- **Plan:** §7's plan state, and any line it wrote, as it stands after §8's
  re-review.
- What was deliberately left, and why, linter findings off the change's lines
  included.
- The gate result, per CI check, naming any that could not run here, any not run
  because the change list is plan files alone, and any red only on failures
  proven at HEAD; `ele plan check` warnings with no line, or off the change's
  lines.

§10 is done when every file on the change list carries a verdict from each
reviewer that ran, or is named unreviewed, and every entry on §5's running list
appears in the report, under its reviewer's label where a reviewer raised it.

## 11. Record

Once every check passes or counts as green under §6, §8 has run or found the
fingerprint unmoved, and every finding is fixed or reported, per repository:

```sh
ele checkpoint record <root>
```

A record taken before the fixes marks unreviewed bytes as reviewed, and the next
checkpoint stops on it.

It refuses without an open run of this session (§1, The run). Run §8's
fingerprint check first, since a new freeze overwrites the saved fingerprint;
then freeze the change list again and record.

§11 is done when `record` prints `recorded: <root>` for every repository.

## 12. Log

Log the run, once per checkpoint, against the repository the change is
mostly in. `/ele:checkpoint-stats` tunes this skill from the log, so a finding
left out is a gap the stats cannot show:

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
- `rereview`: `ran`, `skipped` (§8 found the fingerprint unmoved).
- `scope`: `uncommitted`, `delta` (the freeze wrote a delta), `branch`.
- `severity`: `blocker` (wrong on the main path: wrong output, lost data, a broken
  build), `defect` (wrong only in a reachable edge case), `quality` (correct, but
  misleading a reader, untested, against a convention other than wording or
  order, or a Shape finding), `nit` (wording or order alone, misleading no one).
- `source`: free text; the reviewer (`bugs`, `conventions` or `prose` by the
  merged reviewer's section, or `spec`), the gate check by name, or `user`.
- `stage`: `gate`, `mutation`, `rereview`, `review`, `user`. A finding the user
  raises after this line is written goes in the next checkpoint's record, stage
  `user`, origin `earlier`.
- `verdict`: `deliberate` (confirmed and left on purpose), `fixed`, `open`
  (reported as a question), `refuted`.

`ele checkpoint log` adds `duration`, seconds since the freeze that opened the
run, when a freeze recorded one, and refuses a record the stats would misread,
naming the field.

§12 is done when `ele checkpoint log` prints `logged: <root>` with one finding
per entry on the running list.

## Errors

When a call fails, follow `reference/errors.md` in the plugin root, the
directory holding `skills/`, with the rows below added; stop and report in the
running conversation. A red gate check or mutated test is a result, not an
error here: §5, §6 and Stopping early handle it.

| Error contains | Class | One fix |
| --- | --- | --- |
| `not covered: <root> has changed…` from `check`, exit 1 | Expected answer, not a failure | Go on with §1's status; never rerun |
| `Agent type '…checkpoint-reviewer' not found` | Plugin agent not loaded | Launch `general-purpose` with the same prompt |
| `command not found: <tool>` in a §2 check | The check cannot run here | Report it unverified; no retry |

## Stopping early

A checkpoint that stops without a record — a red check §6 does not count as
green, the user calling it off — ends the run (§1, The run):

```sh
ele checkpoint close <root>
```

When the user says this work needs no checkpoint, exempt its paths for the
session. The command refuses a glob that matches an arbitrary path, such as `*`,
`**` or `*.*`:

```sh
ele checkpoint optout <root> "<glob>" ...
```

Stopping early is done when `close` prints `closed: <root>` per repository.
