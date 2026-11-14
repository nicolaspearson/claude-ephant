---
argument-hint: "<task>"
description: Take one development task too small for a plan from intake to checkpoint in this session; agree its done-when, explore the code in subagents, settle questions, weigh designs from separate architects where there is a real choice, build, then checkpoint and wrap.
disable-model-invocation: true
name: dev
---

# Dev

A task one session finishes, with one done-when and no plan file. The user typing `/ele:dev` is the ask for every step below, the wrap included. Three approvals come before the build: the done-when, the answers to the questions, and the design where there is a real choice. After the last, the build runs unasked. Edit and Write nothing before step 5: what is built before the design is chosen is the design chosen for the user.

Every question to the user goes through AskUserQuestion with your recommended answer as its first option, its label ending `(Recommended)`, and each option's description giving the evidence for it: a file and identifier the explorers cited, what a recall found, or a constraint in the convention files. A recommendation the evidence does not settle says so, and says what would settle it. A recommendation in the question text alone leaves the options unranked to the user.

## 1. Intake

Restate the task in a line or two, from the argument or, with none, from the conversation; with neither, ask for it. A task that as stated needs more than one done-when, or more than one session, is a plan: say which, tell the user to run `/ele:plan new <name>`, and stop. The done-when waits for step 3, where the explorers' findings can ground it. Done when the task is restated, or stopped at the size check

## 2. Explore

Launch 2 or 3 `ele:explorer` agents in one message, each on a different angle, such as the code the task changes, an existing feature to follow, and the tests and gate over it. Read no code yourself first, since a session that reads it launches no explorer, and so runs no recall. Brief each by absolute path: `reference/explorer.md` in the plugin root, the directory holding `skills/`, the file to follow; the user's `CLAUDE.md`, under `$CLAUDE_CONFIG_DIR` when set, else `~/.claude`, and the repository's, the convention files; and give it the task in the user's words, and its question and angle, one angle always being what would show the task done: the tests, commands and pages that cover the behaviour. Where `ele:explorer` is not listed, launch `general-purpose` with the same brief. Then read the files they name to read. Done when every explorer has reported and those files are read

## 3. Questions

The first round proposes the done-when, what will be observable once the task is done, such as a command's output, a test passing, or a real-session run, grounded in what the explorers found covers the behaviour. It names the outcome, never the mechanism, so it holds whichever design step 4 picks. Then ask in rounds what the findings leave open: scope, edge cases, error handling, compatibility. Never ask a question whose answers are alternative designs, such as a flag against a new subcommand: note it for step 4, where the architects design each side, since a design picked here leaves step 4 nothing to weigh. A round asks every question whose prerequisites are settled, numbered, each recommended as the section above says; a question that hangs on one still open waits for a later round. A question the explorers' findings cannot ground gets another explorer on it before it is asked. Facts are the explorers' to find: ask the user for decisions only. A done-when the findings show would need to be two, or an answer that makes the task need more than one done-when or session, stops dev as step 1 does, proposing no handoff. Done when the user has confirmed a done-when, no question is open, and the user has confirmed the answers

## 4. Design

A real choice is two or more designs that each meet the done-when and differ in what the change costs or leaves behind: its files, its interfaces, its behaviour beyond the done-when. A design question step 3 noted is one. Say whether there is one and why, naming the designs weighed. With none, state the one design in a few lines, the files it changes and how, and go on; the checkpoint quotes it. With one, launch two `ele:architect` agents in one message, each under a different constraint pulling that choice its own way, such as the smallest change against the shape easiest to extend. Brief each by absolute path: `reference/architect.md` in the plugin root, the file to follow; the convention files step 2 names; and give it the task in the user's words, the done-when, the answers, the explorers' findings the choice rests on, and its constraint. Where `ele:architect` is not listed, launch `general-purpose` with the same brief. Ask the user to pick, each design an answer with its Costs, recommended as the section above says. A choice failing the test gets no architects, since each is a full design run, on opus for `ele:architect`. Done when whether there is a real choice is said, and the user has picked a design or the one design is stated

## 5. Build

Build the chosen design, following the convention files. Run what the done-when names, a command, a test or a measurement, and keep its output for the checkpoint's Observed field; a real-session run the done-when names is the user's, so say what to run. Done when the design is built and what the done-when names has been run, or handed to the user

## 6. Checkpoint

Run `ele:checkpoint`. Its brief quotes, as what was asked, the task in the user's words, the confirmed done-when and the chosen design together, since the done-when alone would leave every file the design touches reading as unasked; its Observed field names step 5's output. Done when the checkpoint has recorded the change, or stopped early and said why

## 7. Wrap

Where step 4 weighed designs, propose the chosen one and each rejected, with its reason from that design's Costs, as one `decision` note through `/ele:remember`'s Propose section. Then run `ele:wrap`. Done when the decision note is proposed, where there is one, and the wrap has run

## Stopping mid-task

A session stopping before step 6, by the user's word or because the task is blocked, proposes `/ele:handoff`, quoting the task, the done-when and the chosen design, since no plan file carries the place across `/clear`. A size stop, in step 1 or 3, proposes none: `/ele:plan new` launches its own explorers before its first round, so the plan re-derives what dev's explorers found. Done when the handoff is written, the user declines it, or the stop was a size stop
