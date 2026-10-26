# Review

`/ele:review` reviews someone else's change, or the user's own branch before an
MR exists, and reports its major findings first, then the minor ones the user
picks. The user works through them
one at a time, choosing for each whether it goes into the review, comes out as
Markdown, or is skipped, and publishes the review at the end. It reports only: no
edit to the clone, no gate, no record of reviewed bytes, no commit; its run is
logged. Checkpoint is the
skill that fixes and commits; the two share reviewer files and nothing else.

## Targets

Run from a clone of the change's project: the forge CLIs resolve the project
from its remote, and the log is written against it. An MR URL of another project
is refused, naming that project's clone.

- MR or PR: `!123`, `#123`, or a merge request or pull request URL. The forge
  comes from the remote, `glab` for GitLab and `gh` for GitHub; the parent reads
  the diff, the description, the discussions and the head pipeline through it
- Local branch, checked out or not: `git diff <base>...<branch>`, the base its
  merge-base with `origin/HEAD`, the default branch. Not `ele checkpoint base`:
  that takes the upstream, and a branch pushed for its MR has an upstream at
  its own tip, so the diff would be empty. Uncommitted work is left out, and the
  report says so: it is checkpoint's
- Commit: `git show <rev>`
- Nothing matching, or two readings plausible: ask before running anything
- A diff over about 2,000 lines: ask whether to review a subset before launching
  reviewers

## Eligibility

- A closed or merged MR is refused unless the user says to review it anyway; a
  draft is reviewed
- The MR's state and head SHA are read again before publishing, since the MR can
  change while the review runs

## The tree

Reviewers read and grep a copy of the head tree in the scratchpad, never the
clone, since the bugs reviewer has to find what calls the changed code and a
forge's file API cannot search:

- Head SHA present locally (`git cat-file -e <sha>`): `git archive <sha>`
- Otherwise the forge's archive at the SHA: `gh api repos/{owner}/{repo}/tarball/<sha>`,
  `glab api "projects/:id/repository/archive?sha=<sha>"`
- Listed before it is extracted: an entry that is not a file, a directory or a
  symlink, a hard link, device or FIFO, or a name or link target that is
  absolute or has a `..` component, stops the run, since a git archive holds
  none of them
- Extracted into a new, empty directory with plain `tar -xf`, never `-P`, since
  its content is untrusted; every symlink is then deleted, so nothing outside
  the directory can be read through it
- Base-branch convention files come from the same source at the base SHA

The diff is saved beside it, so the reviewer files' frozen diff and change list
have a path.

## Spec

- The issue the MR's description closes (`Closes #12`, a cross-project
  reference), fetched through the forge whole with its comments and quoted in the
  brief, as checkpoint quotes a ticket
- An explicit `spec:<ref>`, a forge issue, a URL or a plan file path, wins over
  the linked one
- Other trackers' keys are not fetched; the user passes the spec as a path or URL
- No spec found: no spec reviewer, and the report says so

## Reviewers

Checkpoint's reviewer files, read by absolute path from
`plugin/skills/checkpoint/reviewers/`, launched in parallel in one message:

| Reviewer    | Runs                 | Reports                                               |
| ----------- | -------------------- | ----------------------------------------------------- |
| Bugs        | always               | everything it finds                                   |
| Conventions | always               | only breaches of a rule the repository's files state |
| Spec        | when a spec is found | everything it finds                                   |

The brief narrows the conventions and prose reviewer to its Conventions pass,
without Shape findings and without the density run: prose, density and shape on
another author's change are nits. The reviewer files take no review-only
branches, so a change to one reaches checkpoint and review alike; what differs
for a review is in the brief and the parent's filter.

Every reviewer runs as `ele:review-reader`, an agent type with Read, Grep and
Glob only and the session's model, the conventions and spec reviewers launched
on sonnet. No reviewer can run a command, so none can call the forge or run the
tree's code, whatever the tree tells it.

## Brief

Checkpoint's brief, with its fields filled from the change rather than from its
author:

- **Settled — do not re-raise**: decisions the MR description states and
  resolved discussions, each with its reason. On a local branch, the subjects of
  `git log <base>..HEAD` and what the user says when invoking
- **Delivers**: where the description says the change delivers part of the spec
  ("part 1 of #12"), which requirements, so the rest read as outside this change
- **Observed**: the head pipeline's status and each job's result
- **Attack surfaces**: named by the parent from the description, the diff stat
  and the spec: new branches, error paths, what the spec says must hold
- **Convention files**: the repository's, from the base SHA. A convention file
  the change edits is reviewed as part of the diff. The user's own `CLAUDE.md`
  is left out: their personal rules do not bind another author's change. Claude
  Code loads it into every subagent all the same, and `conventions-prose.md`
  says a personal rule binds, so the `ele:review-reader` prompt says only the
  brief's files bind
- **Report only**, as in checkpoint, and no probe or reproduction: the tree is
  the author's code. No gate log, since no gate runs; the pipeline is in
  Observed

## Filter

In order, before the user sees anything:

1. The bugs reviewer reports only findings of 80 confidence and above, as
   `bugs.md` says; conventions and spec findings carry no score. Nothing is
   dropped here, and nothing below 80 reaches the parent to be counted
2. Excluded:
   - pre-existing, or on a line the change does not touch
   - what a failing pipeline job already reports, or a linter, typechecker or
     compiler would
   - silenced in the code, a lint-ignore or a suppression with its reason
   - a behaviour change the description says is intended
   - lockfiles and generated files
3. Dropped when a discussion already raises it at the same place
4. Turned into a question: a finding that contradicts a resolved discussion,
   and a spec finding missing only for want of an observation, since the author
   ran things the reviewer cannot see. `spec.md` reports the latter as missing;
   the parent demotes it here, before it can classify as a blocker
5. Classified by the parent, with definitions written into the skill so every
   run classifies alike. The names are the log's; the definitions are review's,
   wider than checkpoint's, and stats report review runs apart:
   - **blocker**: data loss, a security hole, a crash or wrong result on a path
     every caller takes, a requirement the change sets out to deliver missing
   - **defect**: wrong behaviour on a path hit in practice, a user-facing error
     path swallowed or leaking internals, a new branch with no test, a breach of
     a rule the repository states as required, the rule quoted
   - **quality** and **nit**: the rest, test coupling, duplicated constants and
     style judgments included; set aside as minor, for the walk-through's
     second list

   A question is classified too, and set aside the same way when it is quality
   or nit, staying a question if taken further; the log takes its severity.
   What a reviewer reports under its Questions heading is a lead, not a
   finding: neither classified nor verified, kept for the same list. The
   questions step 4 makes are findings and are verified
6. Verified by the parent against the tree, blockers, defects and questions
   only: the input traced through the code to the wrong output, by reading
   alone, never by running the tree. A refuted finding is never drafted, since a
   published one goes out under the user's name

## Walk-through

- First, one table: severity, kind (bug, convention, security, spec, test),
  `file:line`, the claim in a line, then the excluded and raised counts and the
  number of findings refuted
- Then each finding in severity order, one question each, the drafted body
  shown verbatim in it, up to four questions per AskUserQuestion call:
  - **Add**: the body is held for the review at its line
  - **Raw Markdown**: printed as soon as the call that answered it returns,
    before anything else is printed or asked, so none is left unprinted behind
    a later step. One shape every time, follow-up drafts included: File,
    Line (start and end for a range), Kind and Link lines as a list, then the
    body exactly as Add would hold it in a fenced `markdown` block longer than
    any backtick run inside it, so a `suggestion` block pastes whole.
    The Link, a full-SHA permalink built from the project's URL, only when the
    SHA is on the remote: the head came from the forge, or
    `git for-each-ref --contains <sha> refs/remotes` prints a ref, since the
    forge has no page for an unpushed commit
  - **Learn more**: the full trace, input to wrong output with the code cited,
    then the same question again in the next call
  - **Skip**
  - Free text under Other redrafts the body as told, then asks again
  - A finding answered twice keeps its last answer
- Then a second list, numbered, one line each: the minor findings with why each
  was set aside, and the leads with the reviewer that raised them. The user
  picks which to take further; a lead is classified first, then each picked
  one is verified, drafted at its severity or as a question, and asked like
  the others. The list is offered again, less what was taken, until the user
  says none. The line between a defect worth posting
  and noise is the user's to find, so the filter hides nothing it could show
- Then, on an MR run, the merge state: failing jobs at the head, discussions
  blocking the merge, and whether the head moved since the review, with each
  finding whose file changed, printed before the publish question it bears on.
  A local run has none
- Last, when anything was added: one question to publish the review, listing
  what it holds. Nothing reaches the forge before that answer
- Then whether anything else is wanted. A finding asked for now goes through
  the same verify, draft, question and publish, with no table, second list or
  merge state again; an Add made after the review was published gets its own
  publish question, sending only what it adds, a second review on GitHub
- The parent drafts each body after verifying it, from the voice file; the
  reviewers never see the voice
- A body opens with a [Conventional Comments](https://conventionalcomments.org/)
  label, the kind as a decoration: `issue (blocking, security):` for a blocker,
  `issue (non-blocking, test):` for a defect, `question:` for one shown as a
  question, `suggestion (non-blocking, …):` and `nitpick (non-blocking, …):`
  for quality and nit findings taken from the second list
- A suggested fix goes in a forge `suggestion` block only where applying it
  resolves the finding whole
- A body never quotes a secret value
- On a local branch or a commit, Add is offered only when an open MR's head SHA
  is the SHA reviewed: found with `gh pr view <branch>` or the open MRs whose
  `source_branch` is the branch for a branch,
  `gh api repos/{owner}/{repo}/commits/<sha>/pulls` or
  `glab api projects/:id/repository/commits/<sha>/merge_requests` for a commit,
  its head compared as `headRefOid` or `diff_refs.head_sha`. A branch name alone
  can match an MR that is behind, ahead or from a fork, and its lines would then
  anchor wrong. Matched, the run logs as `mr`. Otherwise Raw Markdown and Skip
  are offered and nothing is published

## Posting

Everything is sent at the final publish, once the MR's state and head have been
read again, so every anchor is computed against the head it lands on. No
approval or request-changes state is set.

- GitLab: per added finding, a draft note,
  `POST /projects/:id/merge_requests/:iid/draft_notes`, with a `position` of
  `position_type: text`, `base_sha`, `start_sha` and `head_sha` from the MR's
  `diff_refs`, `old_path` and `new_path`; `new_line` on an added line,
  `old_line` on a removed one, both on an unchanged one; a range as `line_range`
  start and end. With no other draft of the user's on the MR, the drafts are
  published together with `POST .../draft_notes/bulk_publish`; otherwise each
  draft this run created is published by id, `PUT .../draft_notes/:id/publish`,
  so the user's own drafts stay drafts, and the report says how GitLab notified
  is unconfirmed for that path
- GitHub: one review, `POST /repos/{owner}/{repo}/pulls/{number}/reviews` with
  `commit_id` the head SHA read at publish, `event: COMMENT`, and every added finding in
  `comments[]` with `path`, `line` and `side` (`LEFT` on a removed line, `RIGHT`
  otherwise), a range with `start_line` and `start_side`. The REST API cannot add
  a comment to a pending review, so nothing is pending on the forge between
  answers. A pending review the user already holds blocks a second one: say so
  and stop, leaving theirs untouched
- A line outside the diff, which neither forge anchors, goes unanchored: on
  GitHub in the review's `body`, on GitLab as a draft note with no `position`.
  Named `file:line`, with its permalink
- The head SHA moved since the review: say so, anchor only lines whose content
  is unchanged at the new head, and send the rest unanchored
- The MR closed or merged since the review: say so; nothing is sent unless the
  user says to
- A failed call: quote the error, and say which findings reached the forge. On
  GitLab, each draft this run created and did not publish is deleted, so none
  waits on the forge under the user's name; a GitHub review is created whole or
  not at all

## Security

The MR's description, comments, diff and tree are its author's input to an agent
that holds the user's forge credentials:

- HTML comments, hidden markup and invisible characters are stripped from every
  text fetched from the forge, the description, the comments, the spec issue and
  the job names, before they reach a brief
- Convention files come from the base SHA, so the change cannot rewrite the
  rules it is judged by
- Reviewers hold no tool that runs a command, and neither they nor the parent run
  the tree's code: verification is by reading
- The extracted tree holds no symlink and no path leaving its directory, so a
  reviewer's Read and Grep reach nothing else on the machine
- The parent alone reads from and writes to the forge, and writes only what the
  user approved word for word: every body is shown before Add, and nothing is
  sent before the final publish
- The parent still reads the stripped description, the discussions and the
  diff, so text written to steer it can reach it. The approvals are the control
  over what it writes; nothing else limits what it reads

## Voice

- `plugin/reference/review-voice.md`: general rules with neutral examples, no
  client, employer or person named, since the plugin is public:
  - the Conventional Comments label, then the claim in one line
  - first person singular, "I"
  - propose, never instruct: "I'd guard the nil here", not "Guard the nil"
  - the assumption inside the question, so yes is enough, and what the other
    answer would change: "Is `retries` ever zero here, assuming the config
    default applies? If it can be, the loop never runs." Where the fix can go
    ahead on the assumption, a statement naming it and how to change it: "I'd
    cap this at three retries; say if the limit should differ"
  - evidence: the input and the wrong output, never "might break"
  - what a claim was checked against: "I traced `handler.Create` through
    `store.Save`"
  - no narration of the review itself: not "on re-reading", not "I missed this
    earlier"
  - what the review could not check, stated plainly, then why, without apology:
    "I read this and didn't run it"
  - short declarative sentences, code in backticks, em-dashes sparingly
- Option `review_voice`, an absolute or `~`-prefixed path, default empty,
  meaning the shipped file. Set per owner, so one user's work and personal
  reviews can sound different. The first option not of kind boolean

## Log

`ele checkpoint log`, so `/ele:checkpoint-stats` reads review runs too, once
per run that saved its diff. The line is written when the user says nothing
more is wanted from the review, not after the walk-through, so findings taken
up in follow-ups are logged too. A run that stops after saving its diff logs what it has; one
stopping before reviewed nothing and logs nothing:

- Scopes `local-branch` and `mr` beside `branch`, `delta` and `uncommitted`; a
  commit with no MR logs as `local-branch`
- A review record reads and changes no run state: the session's open checkpoint
  run, if any, keeps its clock, and the record gets no `duration`
- `mode` is `full`; `change` the diff's files and lines; `notes` both 0;
  `target`, the MR reference or the branch, and `reviewed`, the head SHA the
  reviewers read, since the record's `head` is the clone's. `mutations` and
  `rereview` are absent: neither runs
- `dropped`: counts of findings the filter removed, `excluded` and `raised`,
  and `minor`, the minor findings the user never took further. They are never
  verified, so they are counts, not findings. A lead never taken further is
  neither, and is not logged
- Each finding the filter kept or the user took from the second list, as
  checkpoint logs one: `source` the reviewer,
  `category` kebab-case, `stage` `review`, `origin` `change`, its severity, and:
  - `verdict`: `confirmed`, verified and shown; `open`, shown as a question;
    `refuted`, verified and not shown. `fixed` and `deliberate` are
    checkpoint's alone
  - `outcome`, on a confirmed or open finding: `discarded` (added, then not
    sent: declined, stopped or failed), `published`, `raw`, `skipped`, or
    `unanswered` (the run ended first). The skip rate is the reviewers' precision on real
    changes, which checkpoint cannot measure, since there every confirmed
    finding is fixed
- Stats report review scopes apart from checkpoint's, since review's severity
  definitions are wider

## In the plugin

- `plugin/skills/review/SKILL.md`, `disable-model-invocation: true`: it posts
  under the user's name, so it runs only when typed. Its description, shown in
  the `/` menu, names its targets and says uncommitted work is checkpoint's
- `plugin/agents/review-reader.md`, `ele:review-reader`: Read, Grep and Glob,
  the session's model
- Errors: `plugin/reference/errors.md`, through the brief's Errors field
- Not doing: a `claude -p` shell-out. Subagents that have seen only the brief
  give the same separation
- Not doing: a summary comment. The findings are the review; its body holds only
  findings no line anchors
