# GitHub

Every call through `gh`, which resolves `{owner}` and `{repo}` from the clone's
remote. A JSON body goes in a file under `<scratch>`, passed with
`--input <file>`, so no body is quoted through the shell.

## Read

- The PR:
  `gh pr view <n> --json state,isDraft,body,url,headRefOid,baseRefOid,headRefName,closingIssuesReferences`.
  `state` is `OPEN`, `CLOSED` or `MERGED`
- The diff: `gh pr diff <n> --color=never`
- The discussions: the review threads, with whether each is resolved, through
  `gh api graphql --paginate -F owner='{owner}' -F repo='{repo}' -F number=<n> -F query=@<file>`,
  the file holding:

  ```graphql
  query($owner: String!, $repo: String!, $number: Int!, $endCursor: String) {
    repository(owner: $owner, name: $repo) {
      pullRequest(number: $number) {
        reviewThreads(first: 100, after: $endCursor) {
          pageInfo { hasNextPage endCursor }
          nodes { isResolved path line originalLine
            comments(first: 100) { nodes { author { login } body } } }
        }
      }
    }
  }
  ```

  `line` is null on an outdated thread; `originalLine` holds its line. And the
  conversation, `gh api --paginate "repos/{owner}/{repo}/issues/<n>/comments"`
- The head checks: `gh pr checks <n> --json name,state,bucket`. It exits
  non-zero when a check failed or is pending; read its output all the same
- A pending review the user holds: the same call as Publish's first step, which
  runs again before the publish question and at publish; read here so the
  summary can say publishing waits on it
- The issues it closes: `closingIssuesReferences`; each whole with
  `gh issue view <number> -R <owner>/<name> --json title,body,comments`, the
  owner and name from that reference's `repository`, which a cross-repository
  issue does not share with the PR
- The archive at a SHA:
  `gh api "repos/{owner}/{repo}/tarball/<sha>" > <scratch>/<name>.tar.gz`
- The open PR for a branch: `gh pr view <branch> --json number,state,headRefOid`,
  a `no pull requests found` error being no PR, the local run; for a commit:
  `gh api "repos/{owner}/{repo}/commits/<sha>/pulls"`. It matches only where the
  PR is open and its `headRefOid`, or `head.sha`, is the SHA reviewed
- A permalink: `gh repo view --json url --jq .url`, then
  `/blob/<full sha>/<path>#L<line>`, or `#L<start>-L<end>` for a range

## Merge state

- The head: `headRefOid` from the PR call again; moved, the files that changed,
  `gh api --paginate "repos/{owner}/{repo}/compare/<reviewed>...<new>" --jq '.files[].filename'`
- Failing checks: `gh pr checks <n> --json name,state,bucket` again, each
  `bucket` `fail`
- Blocking discussions: review threads read again with `isResolved` false,
  which block the merge only where the branch's protection requires
  conversation resolution; name them either way

## Publish

1. A pending review the user already holds blocks a second one:
   `gh api --paginate "repos/{owner}/{repo}/pulls/<n>/reviews" --jq '.[] | select(.state == "PENDING") | .id'`
   printing any id, since only its author sees a pending review. Read before
   the publish question, an id makes it the Check again question; read here,
   at publish, say so and stop, leaving theirs untouched
2. One review holding every added finding:
   `gh api --method POST "repos/{owner}/{repo}/pulls/<n>/reviews" --input <file>`,
   the file:

   ```json
   {"commit_id": "<head SHA read at publish>", "event": "COMMENT",
    "body": "<unanchored findings, or empty>",
    "comments": [{"path": "<path>", "line": 42, "side": "RIGHT",
      "body": "<body>"}]}
   ```

   - `side` is `LEFT` on a removed line, `RIGHT` otherwise
   - A range adds `start_line` and `start_side`
   - An unanchored finding goes in `body`, naming `file:line` and its permalink
   - A `422` saying a pending review already exists is step 1's stop. One
     saying `body` is required: send it once more with `body` listing each
     anchored finding's `file:line`, the only text it adds

   A review is created whole or not at all, so a failure leaves nothing behind
