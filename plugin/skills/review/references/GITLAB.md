# GitLab

Every call through `glab api`, which resolves `:id` from the clone's remote. A
JSON body goes in a file under `<scratch>`, passed with
`--input <file> -H 'Content-Type: application/json'`, so no body is quoted
through the shell.

## Read

- The MR: `glab api "projects/:id/merge_requests/<iid>"`. `state` is `opened`,
  `closed` or `merged`; `draft`; `description`; `diff_refs` holds `base_sha`,
  `start_sha` and `head_sha`
- The diff: `glab mr diff <iid> --raw`
- The discussions: `glab api --paginate "projects/:id/merge_requests/<iid>/discussions"`.
  A discussion is resolved when every note with `resolvable` true has
  `resolved` true; a note's `position` gives its path and line
- The head pipeline: the first of
  `glab api "projects/:id/merge_requests/<iid>/pipelines"` whose `sha` is the
  head SHA, none being a notice; then its jobs,
  `glab api --paginate "projects/:id/pipelines/<pipeline id>/jobs"`, each with
  `name` and `status`
- The issues it closes: `glab api "projects/:id/merge_requests/<iid>/closes_issues"`;
  each issue whole with `glab api "projects/<project id>/issues/<issue iid>"`
  and its `/notes`
- The archive at a SHA:
  `glab api "projects/:id/repository/archive.tar.gz?sha=<sha>" > <scratch>/<name>.tar.gz`
- The open MR for a branch: `glab api "projects/:id/merge_requests?state=opened&source_branch=<branch>"`,
  an empty list being no MR; for a commit:
  `glab api "projects/:id/repository/commits/<sha>/merge_requests"`. It matches
  only where the MR's own `diff_refs.head_sha`, read with the first call above,
  is the SHA reviewed
- A permalink: the project's `web_url`, from `glab api "projects/:id"`, then
  `/-/blob/<full sha>/<path>#L<line>`, or `#L<start>-<end>` for a range

## Merge state

- The head: `diff_refs.head_sha` from the MR call again; moved, the files that
  changed, `git -C <root> diff --name-only <reviewed>..<new>` once the new head
  is fetched, else `glab api "projects/:id/repository/compare?from=<reviewed>&to=<new>"`'s
  `diffs[].new_path`
- Failing jobs: the head pipeline's jobs, read again for the head now, each
  `status` `failed` and not `allow_failure`
- Unresolved discussions: from the discussions read again, each with a note
  whose `resolvable` is true and `resolved` false. They block the merge where
  the project's `only_allow_merge_if_all_discussions_are_resolved`, from
  `glab api "projects/:id"`, is true; name them either way

## Publish

1. The user's drafts already on the MR:
   `glab api "projects/:id/merge_requests/<iid>/draft_notes"`. Remember whether
   there were any
2. Per added finding, one draft note:
   `glab api --method POST "projects/:id/merge_requests/<iid>/draft_notes" --input <file> -H 'Content-Type: application/json'`,
   the file holding `note`, the body, and `position`:

   ```json
   {"note": "<body>",
    "position": {"position_type": "text",
      "base_sha": "<diff_refs.base_sha>", "start_sha": "<diff_refs.start_sha>",
      "head_sha": "<diff_refs.head_sha>",
      "old_path": "<path before>", "new_path": "<path after>",
      "new_line": 42}}
   ```

   - `new_line` on an added line, `old_line` on a removed one, both on an
     unchanged one
   - A range adds `line_range`, `start` and `end` each with `type` (`new` or
     `old`), its line, and `line_code`: the SHA-1 of `new_path`
     (`printf %s <path> | shasum`), then `_<old>_<new>`, the old and new line
     counters at that row of the diff
   - Unanchored: no `position`, the body naming `file:line` and its permalink

   Keep each response's `id`
3. No other draft of the user's in step 1: publish all at once,
   `glab api --method POST "projects/:id/merge_requests/<iid>/draft_notes/bulk_publish"`.
   Otherwise each created draft by id,
   `glab api --method PUT "projects/:id/merge_requests/<iid>/draft_notes/<id>/publish"`,
   so the user's own drafts stay drafts, and say it is unconfirmed how GitLab
   notifies the author of drafts published one by one
4. A failure after any draft was created: delete each draft this run created and
   did not publish,
   `glab api --method DELETE "projects/:id/merge_requests/<iid>/draft_notes/<id>"`,
   so none waits on the forge under the user's name
