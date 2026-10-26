# Review

`/ele:review` reviews someone else's merge request or pull request, or your own
branch before you open one, with the same reviewers `/ele:checkpoint` uses. It
shows only the major findings, and you choose for each whether it goes into the
review, comes out as Markdown to post yourself, or is skipped. It changes
nothing in your clone and commits nothing; uncommitted work is checkpoint's.

```text
/ele:review <!123 | #123 | MR or PR URL | branch | commit> [spec:<issue, URL or path>]
```

Run it from a clone of the change's project. It needs `glab` for GitLab or `gh`
for GitHub, logged in. It runs only when you type it, since it posts under your
name.

## A run

1. **Reads** the change: the diff, the description and discussions, the head
   pipeline, and the issue the description closes, or the spec you pass. HTML
   comments, hidden markup and invisible characters are stripped first. A closed
   or merged MR is refused unless you say to review it anyway
2. **Reviews** a copy of the head tree, extracted into the session scratchpad,
   never your clone. A bugs reviewer and a conventions reviewer run in parallel,
   and a spec reviewer when there is a spec. They can read and search the copy
   and nothing else: none can run a command, so none runs the change's code
3. **Filters** what they find: anything not on the change's lines, already
   raised in a discussion, reported by a failing job, or minor is dropped and
   counted. What is left is classified as a blocker, a defect or a question, and
   each is traced through the code before you see it
4. **Walks through** the findings: a table first, then one question each, the
   comment shown word for word. **Add** holds it for the review, **Raw
   Markdown** prints it with its file and line, and a permalink when the commit
   is on the remote, **Learn more**
   shows the trace, and **Skip** drops it. Typing your own answer redrafts it
5. **Publishes** once, at the end, after one last question listing what the
   review holds. On GitHub it is one review; on GitLab, draft notes published
   together, or one by one when you hold drafts of your own there, which stay
   drafts. It sets no approval: do that on the forge

On your own branch, or a commit, Add is offered only when an open MR's head is
the exact commit reviewed; otherwise you get Raw Markdown and Skip.

## Comments in your voice

Each comment opens with a [Conventional Comments](https://conventionalcomments.org/)
label, `issue (blocking, bug):` for a blocker, `issue (non-blocking, test):` for
a defect, `question:` for a question, and is worded from a voice file. The
shipped one proposes rather than instructs, gives the input and the wrong output
rather than "might break", and says what it could not check. Point
`review_voice` at your own, per owner if your work and personal reviews should
sound different; see [Options](data-and-privacy.md#options).

## What reviews found

Each run is logged with the checkpoints, and `/ele:checkpoint-stats` reports
reviews in their own section: how often you skipped each reviewer's findings,
and how many the filter dropped.
