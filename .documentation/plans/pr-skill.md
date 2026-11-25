---
kind: plan
created: 2026-11-24
spec: .documentation/specs/pr.md
---

# PR skill

## Why

ele drafts no PR or MR body. Matt Pocock's mattpocock-skills `pr` skill gives
a body shape that reads fast in review (Summary, Evidence, Merge Danger) but
gathers nothing: no diff, no gate, no evidence of its own. `/ele:pr` keeps
that shape and fills it from what ele already holds: the branch diff from
`ele checkpoint base`, the checkpoint gate, plan tick lines and the session's
test runs, checked by a reviewer before it reaches the user.

## Requirements

1. A new `/ele:pr` skill adapted from mattpocock-skills' `pr`
2. `/ele:pr`, model-invocable, for writing a PR or MR body
3. The body is raw Markdown inside a `~~~` fence, so inner code blocks survive
   copying, without the "Generated with Claude Code" footer
4. Draft only: the body saved to a scratch file and the `gh pr create
   --body-file …` or `glab mr create …` command printed for the user; never
   posts or edits a PR or MR
5. Sections Summary (smallest visual that makes the point, guidance the
   skill follows, not a reviewer check), Evidence (before and after, a Gate
   line, a "Left to CI" line for the steps only CI runs) and Merge
   Danger (Door, Blast Radius)
6. Uncommitted work not checkpointed: offer `/ele:checkpoint`; declined,
   Evidence says "changed, not yet reviewed"
7. Voice from `report_voice`, else `reference/report-voice.md`; no new option
8. Every claim in the body is checked against the evidence files by a
   reviewer before the draft is shown
9. Adapted in ele's own words, Matt Pocock's MIT notice carried in
   `plugin/skills/pr/references/CREDITS.md`; the Summary visuals menu
   rewritten, not copied
10. The repository's `GLOSSARY.md` for domain language where one exists
11. Spec `.documentation/specs/pr.md`, `docs/pr.md`, rows in
    `docs/commands.md` and `README.md`, the `design.md` skill lists, and one
    eval case with deterministic graders, written but run only when asked

## Context

- As of 2026-11-24: the source is
  `skills/engineering/pr/SKILL.md` of mattpocock-skills 1.3.1, 170 lines: a
  template with `## Summary`, `## Evidence` (Before/After) and
  `## Merge Danger` (Door, Blast Radius), a menu of Summary visuals
  (pseudocode, call tree, component tree, file tree, Mermaid, diff, whole
  block) and a rule to use `GLOSSARY.md`'s language. It runs no git, reads no
  issue, posts nothing and loads no other file
- As of 2026-11-24: mattpocock-skills is MIT, "Copyright (c) 2026 Matt
  Pocock"; the notice must travel with a substantial copy
- As of 2026-11-24: the source's `CREDITS.md` says its Summary visuals menu
  is copied almost word for word from Dex Horthy's `show-me` skill
  (humanlayer); that skill's licence not checked
- As of 2026-11-24: no skill registry in Go; a skill is found by its
  directory under `plugin/skills/`, and internal/skilllint requires a
  `/ele:<name>` row in `docs/commands.md` and `README.md`, `Use when` in a
  model-invocable skill's description, under 500 lines, and only
  `references/`, `scripts/`, `assets/` and `reviewers/` beside `SKILL.md`
  (internal/skilllint `allowedEntries`, `checkDescription`, `maxSkillLines`,
  `CheckDocs`)
- As of 2026-11-24: `/ele:write-report` is draft only, gathers evidence files
  into a scratch directory, gates uncommitted work with
  `ele checkpoint check`, reads `report_voice`, and reviews through
  `ele:report-reviewer` with `reviewers/report.md`
- As of 2026-11-24: `ele checkpoint base <root>` prints the merge-base with
  the upstream, else `origin/HEAD`; a plan tick line records what was
  asserted, at which boundary and at which commit
- As of 2026-11-24: `ele checkpoint check` exits 1 alike for unreviewed
  changes and for none (cmd/ele help `checkpoint check`), so
  `/ele:write-report` runs it only where `git status --porcelain` lists
  something; no ele command says whether a branch's commits were
  checkpointed, `ele checkpoint stats` only aggregating the log
- As of 2026-11-24: on `main` of this repository `ele checkpoint base` prints
  the merge-base with origin/main, HEAD itself only once main is pushed, so
  a pushed branch with no commits ahead has nothing to describe
- As of 2026-11-24: `gh pr create` takes `--body-file`; `glab mr create`
  (1.131.0) takes `--description-file`
- As of 2026-11-24: `/ele:review`'s references detect the forge from
  `remote.origin.url`; its Publish step is ele's only tracker write, asked
  first
- As of 2026-11-24: `plugin/agents/report-reviewer.md` and
  `.documentation/specs/report.md` describe `ele:report-reviewer` as
  write-report's own agent, launched only by it
- As of 2026-11-24: README's Licence section names only GPL-3.0-or-later
- As of 2026-11-24: the only vendored third-party text is
  internal/ticket's difflib, under `LICENSE-PSF` beside it; the repository
  is GPL-3.0-or-later

## Decisions

- Draft only, printing the create command: every tracker write is the user's
  to authorise, and `/ele:write-report` already never sends
- Reuse `report_voice` over a `pr_voice` option: a PR body is a report to a
  reviewer, and a new option costs config, tests and docs rows
- Reuse `ele:report-reviewer` with a new `reviewers/pr.md` over a new agent:
  its brief already takes a reviewer file, core and evidence
- Attribution in `references/CREDITS.md`: the skill lint allows nothing else
  beside `SKILL.md`, and difflib sets the precedent of the notice beside the
  adapted text
- Rewrite the Summary visuals menu: the `show-me` licence it came from is
  unverified
- No footer, overriding Claude Code's PR attribution line: the user's
  requirement
- Committed work is not gated: the user's rules checkpoint before every
  commit, no ele command says whether a commit was checkpointed, and
  offering a branch checkpoint each run would nag; `/ele:write-report`
  treats committed work the same way
  - rejected: offering a branch-scope `/ele:checkpoint` when the session
    holds no review of the commits, because it fires on nearly every run
    in a session after the work's own

## Actions

- [x] pr-draft: write `plugin/skills/pr/SKILL.md`, model-invocable (no
      `disable-model-invocation`, a `Use when` description), taking an
      optional `<base>` ref, and `plugin/skills/pr/references/CREDITS.md`
      holding Matt Pocock's MIT copyright and permission notice; with the
      spec `.documentation/specs/pr.md`, `docs/pr.md`, the `/ele:pr` rows in
      `docs/commands.md` and `README.md`, `/ele:pr` added to the "Used by"
      rows of `ele checkpoint base` (split from the row it shares with
      `close`, `density` and the rest) and `ele checkpoint check`, README's docs
      list and Licence section (the MIT-licensed skill text), the
      `design.md` skill lists, `report_voice` described as both skills'
      voice (internal/config's option Description and cmd/ele's config test,
      `docs/data-and-privacy.md`, README, `docs/reports.md`,
      `.documentation/specs/report.md`), and `ele:pr` added to cmd/ele
      stopcheck's `draftNames` with test rows like write-report's and the
      spec line saying the Stop hook exempts it. Steps:
      - scope: `<base>`, else `ele checkpoint base`; uncommitted changes
      (`git status --porcelain` lists something) run
      `ele checkpoint check`, exit 1 offering `/ele:checkpoint`, declined
      writing those changes in Evidence as "changed, not yet reviewed";
      committed work is not gated, Evidence calling a commit reviewed
      only where a plan tick line or the session shows a review of it
      - evidence files into a scratch directory: the range's diff and log,
      plan tick lines from the range's diff of `.documentation/plans/`,
      the session's test and gate runs; the Gate line from the gate run
      this session holds, else "Gate: not run in this session"; the "Left
      to CI" line from the steps the CI config (`.github/workflows/`,
      `.gitlab-ci.yml`) runs that the session's gate left out
      - domain language from the repository's `GLOSSARY.md` where one exists
      - voice, draft, deliver: the body fenced by a tilde run longer than
      any inside it, the body file holding the body unfenced, and
      `gh pr create --body-file <file>` or
      `glab mr create --description-file <file>` by the forge of
      `remote.origin.url`
  - done when a real run on this repository with a `<base>` a few commits
    back on `main` prints a `~~~` fenced body with Summary, Evidence
    holding Before, After and a `Gate:` line, and Merge Danger holding Door
    and Blast Radius, no "Generated with Claude Code" line, a body file not
    starting with `~~~` and a `gh pr create --body-file` command, having
    run `ele config get report_voice`, and runs no `gh` or `glab` write;
    `references/CREDITS.md` holds the MIT notice and the Summary visuals
    menu shares no sentence with the source's; a stopcheck test row shows
    an `ele:pr` run exempt, seen red with `ele:pr` removed from
    `draftNames`; and `task format`, `task check FORCE=1` and `task audit`
    pass
  - as of 2026-11-24, premise check: cmd/ele `draftNames` holds
    `ele:recipe` and `ele:write-report`, read in `stopState`; its
    stopcheck_test.go rows "a report keeps earlier work pending" and "a
    typed report keeps earlier work pending" are the two an `ele:pr` pair
    copies; `.documentation/specs/stop-check.md`'s "An `ele:recipe` or
    `ele:write-report` run's own calls are exempt" is the spec line
  - as of 2026-11-24: the "Used by" rows are in `docs/commands.md`'s
    `## Used by the skills and hooks`: `ele checkpoint base` shares a row
    with `close`, `density`, `freeze`, `hash`, `optout` and `record`;
    `ele checkpoint check` is `/ele:checkpoint` and `/ele:wrap`
  - as of 2026-11-24: `report_voice` is also described where the item does
    not name: `.documentation/specs/design.md`'s Options list and
    `plugin/reference/report-voice.md`'s intro; cmd/ele's config test
    asserts no Description wording, so it needs no change unless extended
  - as of 2026-11-24: `/ele:pr` goes in `design.md`'s Skills row and its
    plugin file list, between `plan-audit` and `recall`; README's "run only
    when you type them" list leaves it out, being model-invocable
  - as of 2026-11-24: write-report fences with backticks; this item's tilde
    fence is a deliberate difference. Its source's `agents/openai.yaml` and
    `metadata` front matter are not carried, skilllint `allowedEntries`
    allowing only `SKILL.md`, `assets`, `references`, `reviewers` and
    `scripts`
  - as of 2026-11-24, not checked: the CI configs' contents, the `gh` and
    `glab` flags by a run, `/ele:review`'s forge detection,
    `ele checkpoint stats`, and the `show-me` licence
  - asserted in the pr-draft commit: a run on this repository with base
    HEAD~3 on main printed a `~~~` fenced body (Summary; Evidence with Before,
    After, Gate and Left to CI; Merge Danger with Door and Blast Radius), no
    footer, an unfenced body file and a `gh pr create --body-file` command,
    having run `ele config get report_voice` and no `gh` or `glab` write;
    `CREDITS.md`'s licence byte-identical to the source and no `SKILL.md`
    sentence shared with it; stopcheck's two `ele:pr` rows and the
    `report_voice` config test red with the change reverted, green with it;
    `task format`, `task check FORCE=1` and `task audit` green. With no
    `<base>` the skill measures from `origin/HEAD` before
    `ele checkpoint base`, which prints HEAD on a branch pushed with `-u`; the
    Deliver branch commands and that base were checked by their read-only git
    reads on `add-ele-pr`, not by a full run
- [x] pr-scenarios: run `/ele:pr` through the paths the real run on `main`
      does not reach, fixing the skill where one fails
  - done when a run in a scratch repository whose origin is a GitLab URL
    prints `glab mr create --description-file`; a run with an uncommitted
    change no checkpoint covers offers `/ele:checkpoint`, and declined,
    Evidence reads "changed, not yet reviewed"; a run in a scratch
    repository holding a `GLOSSARY.md` uses a term it defines; a run in a
    scratch repository whose CI config runs a step the session's gate did
    not prints a "Left to CI" line naming it; and, where the skill
    changed, `task format`, `task check FORCE=1` and `task audit` pass
  - as of 2026-11-25, premise check: the skill's Deliver Forge bullet maps
    "a GitLab one" to `glab mr create --description-file` with no host rule,
    so a self-hosted host without `gitlab` in it falls to the body path
    alone; its Checkpoint step offers `/ele:checkpoint` on exit 1 of
    `ele checkpoint check`, which a repository with no checkpoint record
    gives (cmd/ele `checkpointCheck`, internal/checkpoint `Reviewed`), and
    its Draft step reads `GLOSSARY.md` whole; `ci.txt` names "the jobs of
    `.gitlab-ci.yml` that run in a merge request pipeline" without saying
    how `rules:`, `only:` or `workflow:` decide that
  - as of 2026-11-25: the installed plugin has no `skills/pr`, so a run
    needs `claude --plugin-dir <clone>/plugin`, as `task dev` starts; no
    eval scaffold under `plugin/evals/` adds an origin
  - asserted by a real session on the installed 1.4.0 plugin in a scratch
    repository (gitlab.com origin, `GLOSSARY.md`, `.gitlab-ci.yml` with a
    `lint` job the local `check.sh` leaves out, a branch one commit ahead
    and an uncommitted `check.sh`), asked in plain words: it started
    `ele:pr`, offered `/ele:checkpoint` on `ele checkpoint check` exit 1, and
    once declined wrote the change as "changed, not yet reviewed"; the body
    named Ledger, Posting and Balance as `GLOSSARY.md` defines them, Evidence
    held `Left to CI: lint`, and it printed `glab mr create
    --description-file`; no skill change, so no gate
- [x] pr-review: add `plugin/skills/pr/reviewers/pr.md` and the step that
      launches `ele:report-reviewer` with it over the body and the evidence
      files before delivery, an unbacked claim dropped, never softened, a
      finding still open shown to the user beside the draft, and an Errors
      row for the agent not found, as in `/ele:write-report`; a turn ending
      while the review runs says in one line that the draft follows it, the
      draft delivered only once the review returns, and a second `/ele:pr`
      or `/ele:write-report` held until then, as report.md's one report at a
      time; update the pr spec, `docs/pr.md`,
      `plugin/agents/report-reviewer.md`'s description and
      `.documentation/specs/report.md`'s agent lines to name both skills,
      rewriting its "a thin agent of its own, as each reviewing skill has"
      to say the two drafting skills share one, its no-Bash reason holding
      for diff evidence too, read as data; and the Evidence step writes
      `ci.txt` wherever a CI config is present, before Draft writes a
      "Left to CI" line from it (folded from the inbox, 2026-11-25: the
      pr-scenarios run wrote none, so its `Left to CI: lint` had no file)
  - done when a run whose body file has a claim no evidence file holds,
    added by editing the file before the review step, delivers a body
    without that claim or shows it as open, the same run without the edit
    passes, a run in a repository with a CI config leaves `ci.txt` under
    its evidence directory, and
    `task check FORCE=1` and `task audit` pass
  - as of 2026-11-25, premise check: `/ele:write-report`'s step 7 launches
    `ele:report-reviewer` with a brief of reviewer file, reader, core,
    renders and evidence files, fixes, reviews once more, and shows a
    finding still open beside the draft; its Errors table holds the
    `report-reviewer' not found` row; its "one report at a time" is prose in
    the skill's preamble and `.documentation/specs/report.md`, enforced by no
    code (cmd/ele `draftNames` only exempts the notes ask)
  - as of 2026-11-25: `reviewers/report.md`'s Unbacked, Uncounted, Missed
    and Secret checks and its Output carry over; its medium checks, core
    against render, fetched sources and reader-dependent Leak do not;
    `plugin/agents/report-reviewer.md` has Glob, Grep and Read only and a
    description naming write-report alone
  - as of 2026-11-25: no Go code, test or grader names the agent's launching
    skill; `plugin/evals/write-report-whatsapp`'s `launches-reviewer` grader
    matches the agent type only; internal/skilllint `allowedEntries` admits
    `reviewers/` and its `checkLinks` fails a `reviewers/…md` code span with
    no file under the skill's own directory
  - asserted in 79f6417: two real sessions on its plugin (`--plugin-dir`) in
    a scratch repository with a gitlab.com origin, `GLOSSARY.md`, a
    `.gitlab-ci.yml` and an uncommitted `check.sh`, the checkpoint offer
    declined: with "Load-tested at 10,000 Postings a second" appended to
    `body.md` before the review, `ele:report-reviewer` flagged it as unbacked,
    the session dropped it and the second review found nothing; without the
    edit one review found nothing; each left `ci.txt` and `checkpoint.txt`
    under its evidence directory and run files with tree state and later
    edits; `task check FORCE=1` and `task audit` pass. Not exercised: a
    second draft held during a review, the agent-not-found Errors row,
    `reviews.txt`, a repository with no CI config
- [x] pr-questions: a reviewer Question is shown to the user beside the
      draft, never acted on without another review (from the inbox,
      2026-11-25: after a clean first review in the pr-review rerun, the
      session edited `body.md` for a Question and delivered the edit
      unreviewed)
  - done when `plugin/skills/pr/SKILL.md`'s Review step and
    `plugin/skills/pr/reviewers/pr.md` say so, and `task check FORCE=1` and
    `task audit` pass
  - asserted in the pr-questions commit: `SKILL.md`'s Review step shows a
    reviewer Question beside the draft and edits nothing for it, reviews an
    edit made anyway once more, shows what that review finds without fixing
    it, and is done only when the last review has reported and the body has
    not changed since; `reviewers/pr.md` tells the reviewer a Question goes
    to the user, not into the body; checked by review of the text, not by a
    run; `task check FORCE=1` and `task audit` pass
- [x] pr-eval: an eval case under `plugin/evals/` whose `scaffold.sh` makes a
      repository with an origin and a branch commits ahead of it, its prompt
      asking for a PR description in plain words, not the slash command,
      and to take the recommended answer to any ask, since AskUserQuestion
      does not load in the sandbox; CONTRIBUTING.md's sandbox-git note
      extended to the case; deterministic graders: a `~~~` fenced body, the
      three section headings, a `Gate:` line, Door and Blast Radius, no
      "Generated with Claude Code", a `tool_used` Skill call of `ele:pr`, a
      `tool_used` launch of `ele:report-reviewer`, and a `max: 0`
      `tool_used` on `gh pr create` and `glab mr create` however invoked
  - done when each grader is shown red by hand against a reply or trace it
    must reject; the case run with `task eval` and passing if the user asks
    for the run, else the tick line saying it was not run
  - asserted in the pr-eval commit: `plugin/evals/pr-describe`, a scaffold
    with a GitHub `origin/main` by ref and a branch one commit ahead, a
    plain-words prompt taking the recommended answer, and eight
    deterministic graders; each grader shown red by hand in node probes
    against replies and Bash, Skill or Agent inputs it must reject, and green
    on ones it must accept (55 probes) and on a real `/ele:pr` closing reply;
    CONTRIBUTING.md's sandbox-git note names the case; not run with
    `task eval`, since no run was asked for; the scaffold syntax- and
    shellcheck-checked, not run
- [ ] Fold each Decision through `/ele:plan-audit close .documentation/plans/pr-skill.md`, then what this plan built and its out-of-scope lines as `Not doing:` into the spec, run `ele plan done .documentation/plans/pr-skill.md`, and delete this plan

## Inbox

## Out of scope

- A host rule for a self-hosted GitLab whose host lacks `gitlab`: such a
  user still gets the body file's path, and a host test means reading
  `glab`'s config (ruled 2026-11-25, from the pr-scenarios premise check)
- Updating the body of an already open PR or MR: drafting the first body is
  the need shown (ruled 2026-11-24, from `/ele:plan new pr-skill`)
- Commit messages: checkpoint step 9 already prints a subject (ruled
  2026-11-24, from `/ele:plan new pr-skill`)
