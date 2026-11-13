---
kind: plan
created: 2026-11-13
spec: .documentation/specs/report.md
---

# ele:write-report

## Why

Sessions do research, make changes and spot bugs that someone outside the
session has to hear about: a client, a stakeholder, a teammate. Today each
write-up is asked for by hand, in Slack, WhatsApp, a Markdown or HTML file or a
PDF, and the same corrections are given each time: Slack has no tables, an
outside reader gets a message that stands alone, what is between a client and
their vendor stays out. This plan builds `ele:write-report`, which drafts one
report for one reader in one configured voice, asks for the medium and shapes
the content to what that medium can carry, and checks every claim against the
session's evidence before the user sees it.

## Requirements

1. A skill that reports what a session researched, changed and found (bugs) to
   clients, stakeholders and team members
2. The audience reduced to one, with a single voice, similar to the review
   voice
3. The skill asks the user for the output format (slack, whatsapp, md files,
   html files, pdf's etc.) and shapes the report content accordingly; the
   medium determines the best way to report, e.g. whatsapp can't have a chart
   but a pdf can
4. A name that shows intent: `ele:write-report` or similar
5. Both the client voice and the report voice read like they came from the
   user, not drafted by an agent: no em dashes or other special characters,
   `...` instead of the ellipsis character, `->` instead of the arrow
   character, and so on

## Context

- As of 2026-11-13: `review_voice` is the only `KindPath` option in
  `internal/config` `Options`; `Config.Value` resolves owner, then global, then
  the empty default; `parse` refuses a relative path and `readableFile` a
  missing or non-regular file on `set`, while `Load` checks form only. Tests
  are in `internal/config` `config_test.go` and `cmd/ele` `config_test.go`
- As of 2026-11-13: only `plugin/skills/review/SKILL.md`, section "7. Draft",
  reads `review_voice`, falling back to `reference/review-voice.md` in the
  plugin root; the fallback is prose in the skill, not code
- As of 2026-11-13: skills are found by directory; nothing in Go,
  `plugin.json` or `hooks.json` lists them. `internal/skilllint` `Check`,
  run by `task test`, requires front matter, "Use when" in the description, a
  `SKILL.md` under 500 lines, no entries but `assets`, `references`,
  `reviewers` and `scripts`, and existing `references/` paths
- As of 2026-11-13: `plugin/agents/` holds thin agents, one per skill that
  launches a reviewer (`checkpoint-reviewer`, `plan-reviewer`,
  `review-reader`), plus `explorer` and `architect`
- As of 2026-11-13: `plugin/skills/review/SKILL.md` says, from an observation
  of 2026-10-26, that text before a further tool call in the same turn does not
  reliably reach the user, so review puts its output in AskUserQuestion calls
  or the closing reply; not re-observed
- As of 2026-11-13: CONTRIBUTING.md "Evals" says, from an observation of
  2026-10-29, that AskUserQuestion does not load in the eval sandbox, so an ask
  is graded on the reply; not re-observed
- As of 2026-11-13: `ele checkpoint check [<root>]` exits 0 when every changed
  path is what the last checkpoint recorded, 1 when anything changed since,
  none ran, nothing is changed or over 200 paths changed
- As of 2026-11-13: the checkpoint Stop gate judges a done claim in the final
  text block against the paths the turn's own `Edit` or `Write` calls named in
  the repository; a turn with none is judged on every moved path when a Bash
  command reads as work (`.documentation/specs/checkpoint-and-ticket-audit.md`,
  "In the plugin"). A report reply saying "fixed" in a turn whose Bash printed
  a PDF could be blocked over other sessions' changes; not checked live
- As of 2026-11-13: `internal/checkpoint` `Reviewed` is false when no path is
  changed, so `ele checkpoint check` exits 1 alike for unreviewed changes and
  for none
- As of 2026-11-13: in `cmd/ele` `stopcheck.go`, a turn's AskUserQuestion or
  file write sets `Pending`, and only a skill in `proposalNames` clears it
- As of 2026-11-13: the user's report voice is a client voice file in one of
  their client repositories, its path theirs to give (written 2026-10-26,
  first 80 lines read): say what is true and needed with no session history,
  record an answer as a fact, the assumption inside the question, name the
  default, propose never instruct, figures in full with a date, what a claim
  was checked against, no estimated dates, limits stated plainly; first person
  singular, no em-dashes. Its later lines not checked
- As of 2026-11-13, from past sessions: the user's reports go mostly to Slack
  (ops, platform teams) and WhatsApp (client contacts), drafted and pasted by
  hand; files (HTML status page, Markdown, PDF) second. No report has held a
  chart. On 2026-10-09 the user dropped a PDF because "the html is much
  useful"
- As of 2026-11-13, from vendor docs: Slack's composer takes `*bold*`,
  `_italic_`, `~strike~`, code, quotes and lists, and no headings or tables;
  how pasted Markdown links, headings and tables convert is not checked.
  WhatsApp takes `*bold*`, `_italic_`, `~strike~`, inline code, `-` and `1.`
  lists and `>` quotes, no headings, tables or `[text](url)`; its API caps a
  message at 4,096 characters, the app's own cap not checked
- As of 2026-11-13: pandoc, weasyprint, typst and Google Chrome are installed
  on the user's machine; no LaTeX engine. Chrome's headless `--print-to-pdf`
  on this machine not checked. Typst has printed CJK, Arabic and emoji as empty
  boxes with its default fonts
- As of 2026-11-13: `cmd/ele` `config.go` refuses every config write while
  `CLAUDECODE` (`sessionVariable`) is set, so `report_voice` is set by the user
  in a terminal

## Decisions

- An ele plugin skill, `ele:write-report`: its evidence is ele's notes, plans,
  checkpoint state and recall, and its voice option follows `review_voice`
  - rejected: a user skill under `~/.agents/skills`, because it would reach
    ele's data only through a CLI it does not ship with
- Voice is a `report_voice` path option, set per owner, empty meaning a shipped
  neutral `plugin/reference/report-voice.md` generalised from the user's
  client voice with no person or project named: the review voice is written
  for code review (Conventional Comments labels, "we" for the team's code) and
  does not fit a client
  - rejected: reusing `review_voice`, because its labels and team "we" do not
    fit a reader outside the team
  - rejected: a second section in the review voice file, because one path
    would then hold two voices and a skill would have to find its half
- A report reads as typed by the user: its punctuation is what a keyboard
  types, so no em or en dash, ellipsis character, arrow, curly quote or other
  typographic character, each written as its typed form (`...`, `->`, `-`,
  `"`). The rule is the voice file's, shipped and user's alike, and the
  reviewer checks it on every render, since a model drifts back to them
- One reader: the voice writes for someone who was not in the session and does
  not know the codebase, teammates included; no setting for what the reader
  knows, since the user's corrections (stand alone, define terms first) already
  hold for every reader
- The skill asks, in one AskUserQuestion call, what the invocation has not
  already said: the medium, what the reader must do (decide, approve, act or
  know) and whether the reader is inside the team or outside it. The second
  shapes the opening line; outside turns on the leak check
- One core independent of medium (bottom line; the ask with its owner;
  findings, each with its evidence and an `as of` date; what changed; options
  and recommendation, what won't work included; what is open), rendered
  through one profile per medium under `references/`: length, markup allowed,
  what tables and links become, how it is read. A render may drop or reshape a
  claim, never add one
  - rejected: one template per medium with no shared core, because the same
    facts would be written up once per medium and could disagree
- Media in v1: Slack as pasted into the composer, WhatsApp, Markdown file, HTML
  file, PDF printed from the HTML by headless Chrome, else weasyprint; with
  neither, the HTML is delivered and the reply says so
- The skill never sends. Slack and WhatsApp text is the turn's closing reply in
  a copyable block; a file is written to a path the user confirms, defaulting
  to `~/Downloads/<slug>-<date>.<ext>`
- A reviewer agent, `ele:report-reviewer`, reads the draft before the user
  does: each claim backed by its evidence and dated, every "all" or "none"
  counted; fit to the medium's length and markup; no secret for any reader;
  and for an outside reader, no internal host, file path, ticket id the reader
  cannot open or other client's name. A thin agent of its own, as each
  reviewing skill has, with Glob, Grep, Read and Bash for read-only git
- The reviewer sees no conversation, so the skill writes the evidence down: the
  core as a file in the session's scratch directory, outside every repository
  in scope, each claim with its evidence pointer (a commit, a diff hunk, a
  command's output, a note, the user's words quoted, a fetched source's URL),
  and the diff, log and recall output and each fetched source's text it drew
  on saved beside it; the brief names those files by absolute path, and the
  reviewer checks a research claim against the saved text, never the live page
- The reviewer's findings are fixed in the core, which is reviewed once more;
  an unbacked claim is dropped, never softened; a finding still open after the
  second review is shown to the user beside the draft
- The checkpoint is not run over the report: it reviews a frozen git diff, and
  a report outside the repository is in none. The repositories in scope are the
  working directory's and each one the session's edits or commits name. One
  with uncommitted changes gets `ele checkpoint check`, since the check cannot
  tell no changes from unreviewed ones; where it fails the skill offers
  `ele:checkpoint` first, and declined, writes those changes as "changed, not
  yet reviewed", never done. Committed work is reported by its commit, and no
  change set means nothing to gate
- Evidence by default is this session: the conversation, commits and the
  working-tree diff of the repositories it touched, plan items ticked and
  checkpoint verdicts; a wider scope the user names ("since Monday") is pulled
  through `ele recall` and `ele show`
- No chart in v1: none of the user's reports has held one. Content steps down
  table, where the medium has one, then list, then one sentence with the figure
- The skill writes no ele note, since only the user knows whether a report was
  sent, but joins stop-check's `proposalNames` as wrap and handoff do, so a
  finished report is not followed by the notes ask its own ask and file write
  would raise
- Requirement 2's "similar to the review voice" is the mechanism, a voice file
  set per owner with a shipped default, not the review voice's style; and its
  one audience holds with the inside or outside ask, which only switches the
  leak check
- Requirement 5's "client voice" is the voice client reports are written in,
  the shipped default and the user's own report voice; the original client
  voice file in the user's client repository is not this plan's to change
- The plan follows the 1.0 release, so 1.0 gains no option, skill or agent
  while its audit closes
- The media are a fixed set in the plugin; a new one is a new profile file
- A spec of its own, `.documentation/specs/report.md`, and a docs page,
  `docs/reports.md`, as `ele:review` has

## Actions

- [ ] write-report-spec: Write `.documentation/specs/report.md` from the
  Decisions: the core, the profiles, the voice option, the asks, the
  checkpoint relation, the reviewer, delivery and the evidence scope, each
  with its reason; add the skill to `.documentation/specs/design.md`'s Skills
  row and layout bullet
  - done when the spec holds every Decision above with its reason, and
    `task lint-docs` and `task format` pass
- [ ] write-report-voice-option: Add `report_voice`, `KindPath`, default empty,
  to `internal/config` `Options`, with tests beside `review_voice`'s in both
  `config_test.go` files; add it to design.md's Options bullet,
  `docs/data-and-privacy.md`'s Options table and the README options list
  - done when the tests, run with `CLAUDECODE` unset as `cmd/ele`
    `config_test.go` runs them, show `set report_voice` refusing a relative
    path and a missing file and `get` returning a set file with its owner as
    source; each new test seen red against a mutated `Options` entry; and
    `task check FORCE=1` and `task audit` pass
  - blocked: write-report-spec
- [ ] write-report-voice-shipped: Ask the user for their client voice file's
  path and the client and project names to grep for, writing neither into
  this plan; read the whole file; write `plugin/reference/report-voice.md`, the
  neutral default, from its rules, generalised: no
  person, client or project named, each rule with a Write/Not example as
  `plugin/reference/review-voice.md` has, the typed-punctuation rule among
  them, and the file itself written in typed punctuation
  - done when every client-voice rule is in the file or named in the spec as
    left out with its reason, and a grep for the user's client and project
    names over the file and the spec finds none, and a grep for non-ASCII
    punctuation over the file finds none
  - blocked: write-report-spec
- [ ] write-report-reviewer: Write `plugin/agents/report-reviewer.md` and
  `plugin/skills/write-report/reviewers/report.md`: the claim, medium-fit and
  leak checks over the core and evidence files, reporting each finding with
  the claim quoted and its evidence or its absence; add the agent to
  design.md's Agents row and layout bullet. The skill lint fails a skill
  directory with no `SKILL.md`, so this item also writes its skeleton: front
  matter with `disable-model-invocation: true`, which write-report-skill
  replaces
  - done when, in a `task dev` session, two launches over planted drafts and
    their evidence files report every planted defect and nothing else. The
    inside reader's draft: a claim the saved diff contradicts, an undated
    figure, an "all" never counted, a table in a WhatsApp render, a
    token-shaped secret, an em dash and an ellipsis character. The outside
    reader's: a WhatsApp render over 4,096 characters, an internal hostname, a
    research claim its saved source does not hold. Each also holds a backed,
    dated claim, and the inside reader's an internal hostname, neither
    reported. And `task check FORCE=1` and `task audit` pass
  - blocked: write-report-spec
- [ ] write-report-skill: Write `plugin/skills/write-report/SKILL.md` and its
  profiles `references/{slack,whatsapp,markdown,html,pdf}.md`: checkpoint
  check, evidence, the ask, the core in the voice, the reviewer, the render
  and delivery; the html profile self-contained (CSS inline, no external font
  or local asset) and, where pandoc makes it, run without its `smart`
  extension, which turns `...`, `--` and straight quotes typographic; the pdf
  profile finding Chrome or weasyprint by `command -v` and the macOS
  application path; add `docs/reports.md`, its row in `docs/commands.md`, a
  README bullet, and a `docs/data-and-privacy.md` paragraph on the evidence
  files left in the scratch directory and the default output path; add
  `ele:write-report` to `cmd/ele` `stopcheck.go`'s `proposalNames` with a
  case in `stopcheck_test.go` seen red without it, and to
  `.documentation/specs/stop-check.md`'s proposal skills
  - done when `task check FORCE=1` and `task audit` pass, and in a `task dev`
    session one report on a session holding a change, a research finding with
    its fetched source and a bug found but not fixed is rendered for each of
    the five media, each carrying all three with their evidence and seen by the
    user as fit to paste or send, the files written to
    the path the user confirmed and the PDF opened, with no report turn blocked
    by the checkpoint gate; a grep for em and en dashes, the ellipsis
    character, arrows and curly quotes over each rendered file and the Slack
    and WhatsApp blocks finds none; and a grep for the user's client and
    project names over `docs/reports.md` and the skill's files finds none
  - blocked: write-report-voice-option, write-report-voice-shipped,
    write-report-reviewer
- [ ] write-report-eval: Add `plugin/evals/write-report-whatsapp/`: a
  `scaffold.sh` giving a repository with one committed change and a clean
  tree, so no checkpoint offer arises, a prompt naming medium, action and
  reader so no ask is needed, `allowed_tools` holding `Agent`, `Bash`, `Read`,
  `Skill` and `Write` as `plugin/evals/plan-new-review/case.yaml` lists its
  own, and graders: a `tool_used` grader on the skill, and regexes reading the
  reply's fenced block alone, one for a fact from the scaffold's commit, and
  no `#` heading, no `|` table row,
  no `**`, no em dash, en dash, ellipsis character, arrow or curly quote, at
  most 4,096 characters
  - done when each grader is seen failing on a reply holding its defect or
    lacking its fact, and one `task eval -- ./plugin` run, on the user's
    go-ahead and running every case as the target does, passes this case
  - blocked: write-report-skill
- [ ] write-report-user-voice: Write `~/.claude/conventions/report-voice.md`, a
  copy of the user's client voice generalised beyond its client, leaving the
  original in place, rewritten in typed punctuation with the typed-punctuation
  rule added, and print the `ele config set --owner hyve` and
  `--owner personal` commands for the user to run in a terminal once an `ele`
  knowing `report_voice` is on `PATH`, from a release or `task install`, which
  replaces the binary this session's hooks run
  - done when `ele config get report_voice --owner hyve` and `--owner
    personal` print that path, and a grep over the file for non-ASCII
    punctuation and for the client's and project's names finds none
  - blocked: write-report-voice-option
- [ ] Fold each Decision through `/ele:plan-audit close
  .documentation/plans/write-report.md`, then what this plan built and its
  out-of-scope lines as `Not doing:` into the spec, run `ele plan done
  .documentation/plans/write-report.md`, and delete this plan

## Inbox

- 2026-11-13, from the plan's wrap: write-report-user-voice sets report_voice
  for owners hyve and personal only, but the client voice it generalises is
  used in bitgorilla's horizon sessions; add --owner bitgorilla to the printed
  commands and the done-when

## Out of scope

- Microsoft Teams and email as media: not among the user's media so far
  (ruled 2026-11-13, from the plan's rounds)
- Charts in HTML and PDF: no report so far has held one (ruled 2026-11-13,
  from the plan's rounds)
- Media the user adds from a folder of their own: one profile file in the
  plugin is enough per medium (ruled 2026-11-13, from the plan's rounds)
- Sending a report, or recording that one was sent: the skill drafts, the user
  sends (ruled 2026-11-13, from the plan's rounds)
- A setting for what the reader already knows: one stand-alone voice covers
  every reader (ruled 2026-11-13, from the plan's rounds)
