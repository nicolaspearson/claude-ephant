# Pull request bodies

`/ele:pr` drafts the body of a pull or merge request for the branch you are on:
what changed, the evidence that it works, and what a merge risks. It fills each
from what the session holds, the branch's diff, the plan items it ticks and the
tests and gate you ran, and leaves out any claim it has no evidence for. It
opens nothing: you get the body and the command to open the request yourself.

```text
/ele:pr [<base>]
```

Claude starts it when you ask for a PR or MR description; you can also type
it. With no `<base>` it describes the branch since it left your default
branch, `origin/HEAD`.

## A run

1. **Checks the code it will describe.** Uncommitted changes the last
   checkpoint does not cover are offered to `/ele:checkpoint` first. Decline,
   and the body calls them "changed, not yet reviewed". Committed work is taken
   as committed: the body calls a commit reviewed only where a plan tick or
   the session shows the review
2. **Saves the evidence** into the session scratchpad: the branch's diff and
   log, the plan ticks it adds, each test and gate run the session made, what
   the checkpoint check found and any review the session made, and the steps
   your CI config runs
3. **Writes the body** in your report voice, with the names your repository's
   `GLOSSARY.md` uses where it has one:

   | Section      | What it holds                                                                                          |
   | ------------ | ------------------------------------------------------------------------------------------------------ |
   | Summary      | What changed and why, with one small picture of it: a call tree, a file tree, a diagram or a short diff |
   | Evidence     | Before and after, a `Gate:` line for the last gate run and the commit it ran at, and "Left to CI" for steps only CI runs         |
   | Merge Danger | Door, one-way or two-way, and Blast Radius, how far a bad merge reaches                                  |

4. **Reviews** the draft before you see it. The reviewer `/ele:write-report`
   uses, with Read, Grep and Glob only, so a commit message written to steer
   it cannot run anything, checks each claim against the saved evidence: the
   `Gate:` and "Left to CI" lines against the runs and your CI config, any
   commit called reviewed, and that the Summary covers every change. Findings
   are fixed and the draft reviewed once more; an unbacked claim is dropped,
   never softened, and one still open is shown to you beside the draft. The
   review may run in the background: a turn that ends while it runs says so,
   and the draft follows when it returns. One draft runs at a time, shared
   with `/ele:write-report`
5. **Hands it over.** The closing reply holds the body in a `~~~` block to
   copy, and `gh pr create --body-file <file>` for a GitHub remote or
   `glab mr create --description-file <file>` for a GitLab one. Where the
   branch is your default branch or a detached HEAD, has no upstream of its
   own, has unpushed commits or uncommitted changes the body describes, what
   you need first is printed above it in order: `git switch -c`, a reminder to
   commit, then `git push`; it runs none of them

The body carries no "Generated with Claude Code" line.

## Its voice

It writes in the same voice as `/ele:write-report`: the shipped one, or your
own through `report_voice`; see [Options](data-and-privacy.md#options).

## What it leaves behind

The body and its evidence stay in the session scratchpad until it goes. Nothing
is written to ele's data, and no note is kept. Work the session did before the
draft still raises its notes ask.

## Credits

Adapted from the `pr` skill of Matt Pocock's
[mattpocock-skills](https://github.com/mattpocock/skills), under the MIT
licence; the notice is in the skill's `references/CREDITS.md`.
