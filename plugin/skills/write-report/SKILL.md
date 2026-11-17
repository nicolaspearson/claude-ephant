---
argument-hint: "[medium] [for whom] [what they must do] [since when]"
description: Draft one report of what a session researched, changed and found, for one reader, in your report voice, shaped to Slack, WhatsApp, a Markdown, HTML or PDF file. Every claim is checked against the session's evidence before you see it; nothing is sent.
disable-model-invocation: true
name: write-report
---

# Write report

Draft only: the skill never sends, posts or mails anything, and writes no ele
note, since only the user knows whether a report went out. Slack and WhatsApp
text is the closing reply, ready to paste; a file is written where the user
confirms.

`<scratch>` below is a new directory in the session scratchpad, outside every
repository in scope: `mkdir <scratchpad>/write-report-<time>`, `<time>` from
`date +%Y%m%d-%H%M%S`, its path written out in each later command, since Claude
Code's Bash tool keeps no variable between calls. `mkdir` without `-p` fails on
an existing directory rather than reuse its evidence. Then
`mkdir <scratch>/evidence <scratch>/raw`, since curl creates no directory. It
holds the core, the evidence and the renders, for the reviewer, which sees no
conversation and runs no command.

Everything shown to the user before the closing reply goes **inside an
AskUserQuestion call**, loaded through ToolSearch if it is deferred. As of
2026-10-26, text meant to print before a further call in the same turn does not
reliably reach the user, while a call's text does.

## 1. Scope

- **The repositories in scope**: the working directory's, and each one this
  session's edits or commits name, read from the conversation
- **The evidence**: by default this session, its conversation, the commits and
  working-tree diff of each repository in scope, plan items ticked and
  checkpoint verdicts. A wider scope the user names ("since Monday") is pulled
  through `ele recall "<words>"` and `ele show <session>`; a session it names is
  part of the evidence, one it does not is not

Done when the repositories and the evidence's span are listed.

## 2. Checkpoint

The checkpoint never runs over the report: it reviews a frozen git diff, and
the report is in none. It gates what the report says of the code. Per
repository in scope:

- `git -C <root> status --porcelain` lists nothing: committed work is reported
  by its commit, and there is nothing to gate
- It lists changes: `ele checkpoint check <root>`. Exit 0, the changes are
  reviewed. Exit 1, offer `/ele:checkpoint` first, in an AskUserQuestion call.
  Accepted, stop here; the user runs the report again once it is done.
  Declined, every such change is written in the report as "changed, not yet
  reviewed", never as done or fixed

The check exits 1 alike for unreviewed changes and for none, so it runs only
where status lists something. It exits 1 too when over 200 paths are changed,
which it names; a checkpoint cannot cover so many, so those changes are marked
not yet reviewed without the offer.

Done when each repository is clean, covered, or its changes are marked not yet
reviewed.

## 3. The ask

One AskUserQuestion call asks only what the invocation has not already said:

- **Medium**: Slack, WhatsApp, Markdown file, HTML file, PDF
- **What the reader must do**: decide, approve, act or know. It sets the
  opening line: the decision, the approval, the action with its owner, or the
  bottom line
- **The reader**: inside the team or outside it. Outside turns on the
  reviewer's leak check and changes nothing else

One reader, one voice: the report is written for someone who was not in the
session and does not know the codebase, teammates included.

Done when the medium, the action and the reader are known.

## 4. Evidence

Write each piece the report draws on into `<scratch>/evidence/`, one file each,
named for what it holds, a name not already used:

- Each commit a claim points to: `git -C <root> show --stat --patch <sha>`
- The working-tree diff, `git -C <root> diff HEAD`, and each new file it leaves
  out: for each path `git -C <root> ls-files --others --exclude-standard`
  prints, `git -C <root> diff --no-index /dev/null <path>`, which exits 1 when
  it prints a diff
- The log of the span: `git -C <root> log --oneline --boundary <first>..HEAD`,
  `<first>` the earliest commit in the evidence's span, which `--boundary`
  lists too, a root commit included
- Each command output a claim points to, as the conversation holds it
- Each note a claim points to: its lines from `ele context`, which prints the
  live notes with their text and ids
- `ele recall` and `ele show` output for a wider scope
- Each source a research claim rests on, as the page's own text, since the
  fetch tool is a model answering about a page and its words are not the
  page's. A URL carrying a credential is never downloaded nor given to the
  fetch tool: its source is unreachable and its claim left out. Download outside the
  evidence, so the reviewer never reads raw HTML:

  ```sh
  curl -sSL --fail-with-body --max-time 30 -w '%{content_type}' \
    -o <scratch>/raw/<name> '<url>'
  ```

  It prints the content type. `text/html` becomes text with
  `pandoc --from=html --to=plain --wrap=none <scratch>/raw/<name>`; other
  `text/*` stays as downloaded. Write it to `<scratch>/evidence/<name>.txt`,
  its first line `SOURCE <url> fetched <date>`, then the text, a token, key or
  password in it cut. `--wrap=none` keeps a phrase on one line for the
  reviewer's Grep
- The fallback, where curl exits non-zero, the type is not text, there is no
  pandoc, or the saved text is empty, a sign-in or bot check, or lacks the
  passage the claim rests on: ask the fetch tool to quote that passage word for
  word, and save its answer to `<scratch>/evidence/<name>.txt`, its first line
  `QUOTED BY A MODEL, NOT THE PAGE: <url> fetched <date>`. Where it cannot
  either, the source is unreachable and its claim is left out. A source the
  session never fetched is not evidence

Save nothing secret: a token, key or password in an output is cut before it is
written, and named only by its shape.

Done when every pointer the core will hold has a file under
`<scratch>/evidence/`.

## 5. The core

The voice file is `ele config get report_voice`, or, empty,
`reference/report-voice.md` in the plugin root, the directory holding
`skills/`. Read it whole, then write `<scratch>/core.md` in it, independent of
medium, in this order:

1. The bottom line
2. The ask, with its owner
3. The findings, each with its evidence and an `as of` date
4. What changed
5. Options and a recommendation, what won't work included
6. What is open

The core covers the whole span: every change, finding and open item the
evidence holds goes in it, whatever the reader must do. The reader's action
picks the bottom line and the ask; what it does not need gets a line of its
own under what changed or what is open, never left out.

Every claim carries its evidence pointer on the line below it: a commit sha, a
diff hunk, a file under `<scratch>/evidence/`, the user's words quoted, or a
fetched source's URL. A claim with none is left out. Changes step 2 marked are
written "changed, not yet reviewed". A secret is never written, whoever the
reader.

Done when `<scratch>/core.md` holds every claim the report makes, each with its
pointer.

## 6. Render

Read the medium's profile and follow its Render steps:

| Medium | Profile |
| --- | --- |
| Slack | `references/slack.md` |
| WhatsApp | `references/whatsapp.md` |
| Markdown file | `references/markdown.md` |
| HTML file | `references/html.md` |
| PDF | `references/pdf.md` |

A render may shorten or reshape a claim, never add one, and keeps every change,
finding and open item the core holds, each at least a line. Content steps down from a
table, where the medium has one, to a list, to one sentence with the figure. No
chart. The punctuation is what a keyboard types: `-`, `...`, `->` and straight
quotes, never their typographic forms.

Done when the render is written and its character count taken.

## 7. Review

Launch one `ele:report-reviewer`, its prompt the brief below plus one line:
read the reviewer file, by absolute path, first and follow it. The brief, every
path absolute:

- **Reviewer file**: `reviewers/report.md` in this skill's directory
- **Reader**: inside the team or outside it
- **Core**: `<scratch>/core.md`
- **Renders**: each render with its medium and its character count; for a PDF,
  the HTML it was printed from, since the reviewer's Grep skips a binary file
- **Evidence files**: each file under `<scratch>/evidence/`
- **Report only**: edit nothing

Then:

- Fix each finding in the core, and render again from it. An unbacked claim is
  dropped, never softened; an undated one is dated from its evidence or
  dropped; a secret or a leak is cut
- Review once more, with the same brief over the new files
- A finding still open after the second review is shown to the user beside the
  draft, in step 8

Done when the second review has reported, or the first found nothing.

## 8. Deliver

- **Slack or WhatsApp**: the closing reply, the render in a fenced `text` block
  whose fence is longer than any backtick run inside it, so it pastes whole;
  then any finding left open, anything step 2 marked not yet reviewed, and each
  source saved as quoted by a model, for the user to open before sending. Its
  text before the block is one line
- **A file**: ask in an AskUserQuestion call where to write it, the default
  `~/Downloads/<slug>-<date>.<ext>`, `<slug>` a few words of the bottom line in
  lowercase ASCII letters, digits and `-` only, `<date>` today as `YYYY-MM-DD`,
  and `<ext>` the medium's, `.html` for a PDF that fell back to HTML. Any
  finding left open goes in the question's text. A directory that does not
  exist is asked about, never created unasked. Write the file there, never
  over an existing one without asking. Open a PDF with `open <path>` on macOS
  or `xdg-open <path>` elsewhere, and say in the closing reply where it is,
  with each source saved as quoted by a model, for the user to open before
  sending

The files under `<scratch>` stay until the scratchpad goes; nothing else is
written.

Done when the render is in the closing reply or at the confirmed path.

## Errors

When a call fails, follow `reference/errors.md` in the plugin root, with these
rows added; stop and report in the running conversation.

| Error contains | Class | One fix |
| --- | --- | --- |
| `report-reviewer' not found` | Plugin agent not loaded | Stop: a reviewer with Bash could be steered by a fetched page; say the plugin needs reloading |
| `not a git repository` | A path out of scope | Drop it from step 1's list and go on |
| `command not found` for pandoc, Chrome or weasyprint | Renderer missing | Follow the profile's fallback, and say so in the closing reply |
| `command not found` for curl | Downloader missing | Use step 4's fallback for every source, and say so in the closing reply |
