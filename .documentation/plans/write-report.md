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
  reviewing skill has, with Glob, Grep and Read only: it reads fetched web
  text, so nothing a page steers it to can run. The skill saves the `git show`
  of each commit a claim points to beside the evidence, and gives each render's
  character count in the brief
  - rejected: Bash limited in prose to read-only git and `wc -m`, because the
    limit is a request nothing enforces, and an injected page could run any
    command through it
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
  sent, and is exempt in stop-check as `ele:recipe` is: its own ask and file
  write raise no notes ask, and work before it stays pending
  - rejected: joining `proposalNames` as wrap and handoff do, because a
    proposal clears all work before it, so a session's fix or finding would
    never be asked about once a report was drafted
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

- [x] write-report-spec: Write `.documentation/specs/report.md` from the
  Decisions: the core, the profiles, the voice option, the asks, the
  checkpoint relation, the reviewer, delivery and the evidence scope, each
  with its reason; add the skill to `.documentation/specs/design.md`'s Skills
  row
  - done when the spec holds every Decision above with its reason, and
    `task lint-docs` and `task format` pass
  - as of 2026-11-16: premise check. No spec names report.md but this plan,
    and no spec index exists. `.documentation/specs/review.md` is the model: a
    flat spec with no Decisions or Why heading, each reason inline after a
    colon or "since", a `## Voice` section for the option and an
    `## In the plugin` section ending in `Not doing:` bullets, which this plan's
    closing action fills. design.md's Skills row, in `## Names`, is
    alphabetical, so `/ele:write-report` goes after `/ele:wrap`. The layout
    bullet ("The repo is a marketplace, ...") lists only shipped files, so
    `skills/write-report` and `agents/report-reviewer.md` are
    write-report-reviewer's and `reference/report-voice.md`
    write-report-voice-shipped's. `task lint-docs` fails only a `.go:` or
    `.html:` line number and a note-id-shaped string. `task lint` (rumdl
    MD057, MD051) fails a link to a missing file, so the spec does not link
    `docs/reports.md` until it exists. Context lines on `internal/config`
    `Options`, review's "7. Draft", `cmd/ele` `proposalNames` and
    `sessionVariable` re-derived and hold; `sessionVariable` is a deterrent,
    not a lock, so the spec does not call it a boundary
  - as of 2026-11-16: the Decision that the plan follows the 1.0 release is
    left out of the spec, as plan sequencing, not design; 1.0 has shipped
  - asserted in the write-report-spec commit: report.md holds every Decision
    with its reason except the 1.0 sequencing one, left out as plan-only, and
    each rejected option's reason, checked by the spec, bugs and conventions
    reviewers against the plan; design.md's Skills row lists
    `/ele:write-report`; `task lint-docs`, `task format` and
    `task check FORCE=1` pass
- [x] write-report-voice-option: Add `report_voice`, `KindPath`, default empty,
  to `internal/config` `Options`, with tests beside `review_voice`'s in both
  `config_test.go` files; add it to design.md's Options bullet,
  `docs/data-and-privacy.md`'s Options table and the README options list
  - done when the tests, run with `CLAUDECODE` unset as `cmd/ele`
    `config_test.go` runs them, show `set report_voice` refusing a relative
    path and a missing file and `get` returning a set file with its owner as
    source; each new test seen red against a mutated `Options` entry; and
    `task check FORCE=1` and `task audit` pass
  - as of 2026-11-16: premise check. `review_voice` is the last `Options`
    entry and the only `KindPath` one; `report_voice` sorts before it. No
    code keys on the name, and no test ranges over `Options` or compares it to
    the docs, so each behaviour needs a test naming `report_voice`:
    `internal/config` `TestSetRefusesAPathNamingNoReadableFile` covers the
    relative and missing refusals and `TestSetStoresAPathAsTypedAndValueExpandsIt`
    the owner source, both for `review_voice` only; `cmd/ele`
    `TestConfigReviewVoiceRoundTripsGloballyAndPerOwner` sets `CLAUDECODE`
    empty, which the session check reads as unset. The three docs places are
    the only option lists
  - asserted in the write-report-voice-option commit: `internal/config` tests
    show `set report_voice` refusing a relative path and a missing file and
    `Value` returning a set file with owner source; `cmd/ele`
    `TestConfigReportVoiceRoundTripsPerOwner`, with `CLAUDECODE` empty, shows
    the same through `ele config set`, `get` and `list`; each new test went
    red against a renamed `Options` entry, and `KindFolder`, a changed
    description and a non-empty default each turned the tests reading them
    red; `task check FORCE=1` and `task audit` pass
- [x] write-report-voice-shipped: Ask the user for their client voice file's
  path and the client and project names to grep for, writing neither into
  this plan; read the whole file; write `plugin/reference/report-voice.md`, the
  neutral default, from its rules, generalised: no
  person, client or project named, each rule with a Write/Not example as
  `plugin/reference/review-voice.md` has, the typed-punctuation rule among
  them, and the file itself written in typed punctuation; add
  `reference/report-voice.md` to design.md's layout bullet
  - done when every client-voice rule is in the file or named in the spec as
    left out with its reason, and a grep for the user's client and project
    names over the file and the spec finds none, and a grep for non-ASCII
    punctuation over the file finds none
  - as of 2026-11-16: premise check. `plugin/reference/review-voice.md` is the
    model: a `#` title, an intro saying the option replaces the file, then one
    `##` heading per rule with `- Write:` and `- Not:` bullets; its own Not
    example shows em dashes, so this file names the typographic characters in
    words instead. No lint reads `plugin/reference/` but rumdl, and none
    checks non-ASCII, so `LC_ALL=C grep -nP '[^\x00-\x7F]'` is run by hand.
    report.md's `## Voice` named no left-out rule before this item
  - asserted in the write-report-voice-shipped commit:
    `plugin/reference/report-voice.md` carries each rule of the user's client
    voice, generalised with new examples, each with a Write and a Not;
    report.md's `## Voice` names the three project-specific parts and the
    em-dash rule as left out with reasons; a grep for non-ASCII over the file
    and one for the client's, project's and every other proper name in the
    source over the file and the spec find none; checked by the spec, bugs and
    conventions reviewers against the source; `task check FORCE=1` and
    `task audit` pass
- [x] write-report-reviewer: Write `plugin/agents/report-reviewer.md` and
  `plugin/skills/write-report/reviewers/report.md`: the claim, medium-fit and
  leak checks over the core and evidence files, reporting each finding with
  the claim quoted and its evidence or its absence; add the agent to
  design.md's Agents row and layout bullet, and `skills/write-report` to the
  layout bullet. The skill lint fails a skill
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
  - as of 2026-11-17: premise check. Thin agents put `description`, `model`,
    `name` and `tools` in front matter and their limits in prose; none limits
    Bash to read-only git in `tools`, which led to the no-Bash Decision. The
    skill lint passes a `SKILL.md` of front matter alone with
    `disable-model-invocation: true`, and fails a code span naming a missing
    `references/` file. `task dev` is interactive and runs `task install`
    first; CONTRIBUTING.md says it loads the clone's plugin for that session
    only, so a session running the cached plugin has no `ele:report-reviewer`
    (not checked live), and the two launches are run by the user in a
    `task dev` session
  - as of 2026-11-17: the agent (Glob, Grep, Read, sonnet), the reviewer file,
    the skeleton `SKILL.md` and design.md's rows are written; `task check
    FORCE=1` and `task audit` pass; dry runs of an earlier version, with
    `general-purpose` reading the agent and reviewer files, found every planted
    defect, and one extra on the inside draft came from a fixture flaw since
    fixed; not re-run on the current files
  - asserted in the write-report-reviewer commit, over 0fb66b8 and 10e8957.
    `ele:report-reviewer`, launched in `task dev` sessions and limited to
    Glob, Grep and Read, called no other tool. Over the outside fixture, with
    the reviewer file of 0fb66b8, it reported every planted defect and nothing
    else; not re-run after 10e8957. Over the inside fixture, re-run after
    10e8957, it reported every planted defect, the backed claims and both
    internal hosts unreported; it found the em dash and ellipsis by reading,
    not the instructed Grep, and listed the ellipsis line under Unbacked while
    its verdict folded it into typed punctuation. `task check FORCE=1` and
    `task audit` pass on 0fb66b8; after it only Markdown changed, and
    `task lint` and the skill lint pass
- [x] write-report-skill: Write `plugin/skills/write-report/SKILL.md` and its
  profiles `references/{slack,whatsapp,markdown,html,pdf}.md`: checkpoint
  check, evidence, the ask, the core in the voice, the reviewer, the render
  and delivery; the evidence holding the `git show` of each commit, and the
  text of each command output and note, a claim points to, and the reviewer's
  brief giving each render's character count and, for a PDF, the HTML it was
  printed from, since the reviewer runs no command and Grep skips a binary; the
  html
  profile self-contained (CSS inline, no external font or local asset) and,
  where pandoc makes it, run without its `smart` extension, which turns `...`,
  `--` and straight quotes typographic; the pdf profile finding Chrome or
  weasyprint by `command -v` and the macOS application path; replace the
  skeleton's "not built yet" body; add `docs/reports.md`, its row in
  `docs/commands.md`, a README bullet, a `docs/data-and-privacy.md` paragraph
  on the evidence files left in the scratch directory and the default output
  path, and a line in `docs/notes-and-recall.md` that a report raises no notes
  ask while work before it stays pending; exempt
  `ele:write-report` in `cmd/ele` `stopcheck.go` as `recipeSkill` is, setting
  `Proposing` without clearing `Pending`, with a case in `stopcheck_test.go`
  seen red without it, and say so beside `ele:recipe` in
  `.documentation/specs/stop-check.md`
  - done when `task check FORCE=1` and `task audit` pass, and in a `task dev`
    session one report on a session holding a change, a research finding with
    its fetched source and a bug found but not fixed is rendered for each of
    the five media, each carrying all three with their evidence and seen by the
    user as fit to paste or send, the files written to
    the path the user confirmed and the PDF opened, with no report turn blocked
    by the checkpoint gate; a grep for em and en dashes, the ellipsis
    character, arrows and curly quotes over each rendered file and the Slack
    and WhatsApp blocks finds none; a grep for the user's client and project
    names over `docs/reports.md` and the skill's files finds none; and a grep
    for `write-report` finds each docs page this item names, each checked
    against the skill as built
  - as of 2026-11-17: premise check. `cmd/ele` `recipeSkill` is one constant
    whose `advance` case sets only `Proposing`; a Skill call and a typed
    command both record `ele:write-report`, so a second constant beside it
    covers both, tested as `stopcheck_test.go`'s "a recipe keeps earlier work
    pending" is. `plugin/skills/review/SKILL.md` is the model: numbered steps
    each ending "Done when", output in AskUserQuestion calls and the closing
    reply, reviewer files by absolute path under the plugin root, an Errors
    table. The skill lint caps `SKILL.md` under 500 lines and fails a code span
    naming a missing `references/` file. No `ele` command lists the repositories
    a session touched, so the skill names them from the conversation, as
    `/ele:wrap` does. The README holds skill tables, feature bullets, a
    typed-only sentence and a Guides list, and `docs/data-and-privacy.md` a
    Models table, each needing the skill too. pandoc 3.13, weasyprint 70 and
    Chrome are installed
  - as of 2026-11-17: the skill, its five profiles, the Stop-hook exemption
    with two tests seen red, the stop-check spec line and the docs
    (`docs/reports.md`, `docs/commands.md`, README, `docs/data-and-privacy.md`,
    `docs/notes-and-recall.md`) are written and reviewed; `task check FORCE=1`
    and `task audit` pass; a pandoc probe gave one title and nothing external,
    and Chrome printed a PDF
  - asserted in the write-report-skill commit, over 9027d74, aff0296 and
    2e5492c: in one `task dev` session over a scratch repository holding a
    commit, a fetched RFC 9110 and an unfixed off-by-one, `/ele:write-report`
    rendered WhatsApp, Markdown and HTML under the full-coverage fix, and Slack
    and PDF before it, not re-rendered since; each carrying all three with its
    evidence,
    reviewed by `ele:report-reviewer` and seen by the user as fit to send; the
    files at the confirmed path and the PDF opened; no Stop hook block or notes
    ask on any report turn; no typographic character in any render, the Slack
    and WhatsApp blocks included; no client or project name in
    `docs/reports.md` or the skill's files; a grep for `write-report` finds
    each docs page named; `task check FORCE=1` and `task audit` pass, with only
    Markdown changed after
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
- [x] write-report-user-voice: Write `~/.claude/conventions/report-voice.md`, a
  copy of the user's client voice generalised beyond its client, leaving the
  original in place, rewritten in typed punctuation with the typed-punctuation
  rule added, and print the `ele config set --owner hyve`, `--owner personal`
  and `--owner bitgorilla` commands for the user to run in a terminal once an
  `ele` knowing `report_voice` is on `PATH`, from a release or `task install`,
  which replaces the binary this session's hooks run; bitgorilla's horizon
  sessions are where the client voice is used
  - done when `ele config get report_voice` with `--owner hyve`, `--owner
    personal` and `--owner bitgorilla` each print that path, and a grep over
    the file for non-ASCII punctuation and for the client's and project's
    names finds none
  - as of 2026-11-17: premise check. The `ele` on `PATH` is a `task dev`
    build that knows `report_voice`, unset for all three owners;
    `~/.claude/conventions/` held `review-voice.md` and `services.md` and no
    `report-voice.md` before this item
  - asserted in the write-report-user-voice commit: `ele config get
    report_voice` prints `~/.claude/conventions/report-voice.md`, expanded,
    for owners hyve, personal and bitgorilla (owner as source), set by the user
    in a terminal on a build knowing the option; a non-ASCII grep and a grep
    for the source's client, project and every other proper name over the file
    find none; the source has no uncommitted change, its last commit dated
    2026-10-26; the file reviewed by the bugs, conventions and spec reviewers
    against the source
- [ ] Fold each Decision through `/ele:plan-audit close
  .documentation/plans/write-report.md`, then what this plan built and its
  out-of-scope lines as `Not doing:` into the spec, run `ele plan done
  .documentation/plans/write-report.md`, and delete this plan

## Inbox

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
