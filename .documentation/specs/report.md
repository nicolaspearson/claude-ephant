# Report

`/ele:write-report` drafts one report of what a session researched, changed and
found, for one reader outside the session: a client, a stakeholder or a
teammate. It writes in one configured voice, asks for the medium and shapes the
content to what that medium can carry, and has every claim checked against the
session's evidence before the user sees the draft. It drafts only: it never
sends, and it records nothing about whether a report went out.

An ele plugin skill, not a user skill under `~/.agents/skills`: its evidence is
ele's notes, plans, checkpoint state and recall, which a user skill would reach
only through a CLI it does not ship with, and its voice option follows
`review_voice`.

## Evidence

- By default, this session: the conversation, the commits and working-tree diff
  of the repositories in scope, plan items ticked and checkpoint verdicts
- A wider scope the user names ("since Monday") is pulled through `ele recall`
  and `ele show`
- The repositories in scope are the working directory's and each one the
  session's edits or commits name

## Checkpoint

The checkpoint is not run over the report: it reviews a frozen git diff, and a
report written outside the repository is in none. It gates what the report says
of the code instead:

- A repository in scope with uncommitted changes gets `ele checkpoint check`.
  The check exits 1 alike for unreviewed changes and for none, so it runs only
  where something is changed
- Where it fails, the skill offers `/ele:checkpoint` first. Declined, the
  report writes those changes as "changed, not yet reviewed", never as done
- Committed work is reported by its commit, and a repository with no change set
  has nothing to gate

## The ask

One AskUserQuestion call asks what the invocation has not already said:

- the medium
- what the reader must do: decide, approve, act or know. It shapes the opening
  line
- whether the reader is inside the team or outside it. Outside turns on the
  reviewer's leak check; it changes nothing else

One reader, one voice: the voice writes for someone who was not in the session
and does not know the codebase, teammates included. There is no setting for
what the reader already knows, since the corrections a stand-alone report needs
(define a term before using it, no session history) hold for every reader.

## The core

The report is written once, independent of medium, in the voice:

- the bottom line
- the ask, with its owner
- the findings, each with its evidence and an `as of` date
- what changed
- options and a recommendation, what won't work included
- what is open

The core covers the whole span: every change, finding and open item the
evidence holds, whatever the reader must do. The reader's action picks the
bottom line and the ask; the rest gets a line each under the findings, what
changed or what is open, since a reader deciding one question still needs to
know of a bug the same work found. A claim with no evidence stays out all the
same.

The core is a file in the session's scratch directory, outside every repository
in scope, each claim carrying its evidence pointer: a commit, a diff hunk, a
command's output, a note, the user's words quoted, or a fetched source's URL.
The diff, log and recall output and each fetched source's text it drew on are
saved beside it, with the `git show` of each commit and the text of each
command output and note a claim points to. The reviewer sees no conversation
and runs no command, so what it checks against has to be on disk.

A web source is saved as the page's own text, downloaded with `curl` and made
text with pandoc, never the fetch tool's words: the fetch tool is a model
answering about a page, and its quotes can be wrong. The download stays outside
the evidence, so the reviewer reads text, not raw HTML. Where the download
fails, is not text, has no pandoc to make it text, is a sign-in or bot check,
or lacks the claim's passage,
the fetch tool's word-for-word quote is saved instead, marked on its first line
as quoted by a model; the reviewer lists a claim resting on one as a question,
and the closing reply names its source for the user to open before sending.

## Media

Each medium is one profile under the skill's `references/`, rendering the core:
its length, the markup it allows, what tables and links become, and how it is
read. A render may shorten or reshape a claim, never add one, and keeps every
item the core holds, options and the recommendation included, each at least a
line. A WhatsApp render still over its cap goes as a Markdown file instead,
rendered and reviewed afresh. The reviewer reports an item a render dropped,
and a commit or hunk in the saved evidence the core never mentions. One core
with one
profile per medium, not a template per medium, since the same facts written up
once per medium could disagree.

The Slack and WhatsApp markup below is as of 2026-11-13, from vendor docs:

- Slack, as pasted into the composer: `*bold*`, `_italic_`, `~strike~`, code,
  quotes and lists; no headings or tables. How pasted Markdown links, headings
  and tables convert is not checked
- WhatsApp: `*bold*`, `_italic_`, `~strike~`, inline code, `-` and `1.` lists
  and `>` quotes; no headings, tables or `[text](url)`, and at most 4,096
  characters, the cap its API puts on a message; the app's own cap is not
  checked
- Markdown file
- HTML file, self-contained: CSS inline, no external font or local asset. Where
  pandoc makes it, it runs without its `smart` extension, which turns `...`,
  `--` and straight quotes typographic
- PDF, printed from the HTML by headless Chrome, else weasyprint, each found by
  `command -v` and the macOS application path. With neither, the HTML is
  delivered and the reply says so. The reviewer reads a PDF through the HTML
  it was printed from, since its Grep skips a binary file

No chart: no report the skill was designed from held one. Content steps down
from a table, where the medium has one, to a list, to one sentence with the
figure.

The media are a fixed set in the plugin; a new one is a new profile file.

## Voice

- `plugin/reference/report-voice.md`: the shipped default, generalised from a
  client voice the plugin's author writes in, with no person, client or project
  named, since the plugin is public. Each rule carries a Write and a Not example,
  as the review voice does. The client voice it comes from stays where it is,
  unchanged; the voice a report is written in is this file or the user's own
  `report_voice` file. Left out of it: the source's list of the documents it
  governs and who reads them, its name for the client, and its account of the
  project's own decisions file, since all three are one project's; its
  "em-dashes sparingly", since the typed-punctuation rule below bans them. Its
  lowercase `id` rule is folded into one spelling per term
- Option `report_voice`, an absolute or `~`-prefixed path, default empty,
  meaning the shipped file. Set per owner, as `review_voice` is, so work and
  personal reports can sound different. The report voice is like the review
  voice in this mechanism, a file set per owner with a shipped default, not in
  its style. A separate option, not `review_voice`
  reused, since the review voice's Conventional Comments labels and its "we"
  for the team's code do not fit a reader outside the team; and not a second
  section in the review voice file, since one path would then hold two voices
  and the skill would have to find its half
- A report reads as typed by the user: its punctuation is what a keyboard
  types. No em or en dash, ellipsis character, arrow, curly quote or other
  typographic character; each is written in its typed form, `...`, `->`, `-`,
  `"`. The rule is the voice file's, shipped and user's alike, and the reviewer
  checks it on every render, since a model drifts back to those characters

## Reviewer

`ele:report-reviewer` reads the core and its evidence files before the user
sees anything, the brief naming each by absolute path:

- each claim backed by its evidence and dated, and every "all" or "none"
  counted
- a research claim checked against the saved source text, never the live page,
  which may have changed since the session read it
- each render's fit to its medium: length, markup and typed punctuation
- no secret, for any reader
- for an outside reader, no internal host, file path, ticket id the reader
  cannot open, or other client's name

Each finding quotes the claim with its evidence or its absence. Findings are
fixed in the core, which is reviewed once more. An unbacked claim is dropped,
never softened. A finding still open after the second review is shown to the
user beside the draft.

## Delivery

The skill never sends. Slack and WhatsApp text is the turn's closing reply in a
copyable block: as of 2026-10-26, text before a further tool call in the same
turn did not reliably reach the user, an observation `/ele:review` rests on too
and not made again since. A file is written to a path the user
confirms, defaulting to `~/Downloads/<slug>-<date>.<ext>`.

The skill writes no ele note, since only the user knows whether a report was
sent. The Stop hook exempts it as it does `/ele:recipe`: its own
AskUserQuestion and file write raise no notes ask, and work before it stays
pending. Not a proposal skill like `/ele:wrap` and `/ele:handoff`, since a
proposal clears all the work before it, and a fix or finding the report
describes would then never be asked about.

## In the plugin

- `plugin/skills/write-report/SKILL.md` and its profiles under `references/`
- `plugin/agents/report-reviewer.md`, `ele:report-reviewer`: a thin agent of
  its own, as each reviewing skill has, on sonnet, with Glob, Grep and Read
  only. It reads fetched web text, so a page written to steer it could reach
  it; with no tool that runs a command, nothing it is steered to can run, as
  with `ele:review-reader`. The skill saves the `git show` of each commit a
  claim points to beside the evidence, and gives each render's character count
  in the brief. Its instructions are
  `plugin/skills/write-report/reviewers/report.md`
- `plugin/reference/report-voice.md`, the shipped voice
- A docs page of its own, as `/ele:review` has
