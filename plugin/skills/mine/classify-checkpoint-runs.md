# Classify checkpoint runs

You are mining past runs of a code-review skill called "checkpoint". Each run is one digest file; the paths and your output file are in the message that launched you.

Judge each run by what its digest shows; the skill's text differs from run to run. In outline: the parent freezes the diff, launches fresh reviewer subagents, or a single combined reviewer in small mode, verifies each finding (confirmed and fixed, refuted with a reason, or left open as a question), mutates new tests, runs the gate, re-reviews the fixes if the tree's fingerprint moved, prints commit commands and records the run. "Covered" means `checkpoint-reviewed.sh check` or `ele checkpoint check` found the tree already reviewed and the run stopped early.

Digest layout:

- Line 1: JSON metadata. `session` and `timestamp` key the run; also owner, project, era, durationSeconds, agents, agentDescriptions, agentSeconds, bashCommands, gateCommands, edits, hookBlocks, triggeredByHook, covered, end and nextHuman
- `=== PRIOR USER ASK`: the user's messages before the run
- `=== RUN EVENTS`: the parent session from the invocation, one line each: `HUMAN`, `TEXT` (the parent's text), `AGENT` (a subagent launch), `BASH`, other tool calls by name, and `HOOK-BLOCK`, a Stop hook interrupting the parent. It ends at the parent's final report, the next checkpoint invocation, the next review that is not this run's, or four hours on. Reviewers ran in the background, so the user often typed while the run was going
- `=== AGENT <description> at <time>`: one per subagent launched, with its prompt, its tool calls and its final report
- `=== FINAL TEXTS (full)`: the last two long parent messages cut off in RUN EVENTS, each up to 30,000 characters. Every cut message ends in `[TRUNCATED, see FINAL TEXTS]`, but an earlier one has no section here: what it said past the cut is not in the digest
- `=== NEXT HUMAN`: the user's messages after the window

Rules: read-only. Write only your output file. Never copy a token or credential. No git write commands, no kubectl, helm or aws. No helper scripts on disk.

For each digest, read it whole and write ONE JSON object per line to your output file:

```
{"session": "<line 1 session, exactly>", "timestamp": "<line 1 timestamp, exactly>",
 "is_checkpoint": true,            // false if the run aborted, was declared not a checkpoint, or had nothing to review
 "covered_shortcircuit": false,
 "mode": "full|small|none",
 "change": {"files": 4, "lines": 120},     // null where not visible
 "reviewers": [ {"role": "bugs|conventions|prose|conventions-prose|spec|combined", "round": 1, "raised": 3, "reran_gate_or_tests": false, "notes": "optional"} ],
 "findings": [ {"source": "bugs|conventions|prose|spec|combined|gate|mutation|user|parent",
                "stage": "review|rereview|gate|mutation|user",
                "category": "kebab-case", "summary": "one line",
                "verdict": "fixed|refuted|open|deliberate",
                "origin": "change|fix|earlier",
                "also_raised_by": ["prose"],            // other sources in the same round that flagged the same issue; merge duplicates into ONE finding
                "rereview_kind": "broken-by-fix|missed-first-round|restated-first-round|n/a",
                "reraises_user_decision": false,        // true if it re-raises something the user had already decided in the session (cite in summary)
                "refute_reason": "if refuted, why, short"} ],
 "mutations": {"run": 0, "failed": 0, "inconclusive": 0},
 "rereview": "ran|skipped|not-done",        // skipped = fingerprint unmoved; not-done = should have run but did not
 "gate": {"runs": 2, "failed_after_review": false, "what_failed": ""},
 "time_sinks": "where the wall time went, short: e.g. 'bugs reviewer 9 min re-ran go test ./...; parent waited; 3 gate runs'",
 "hook_blocks_in_run": 0, "hook_block_justified": "n/a|yes|no (why)",
 "repeated_work": "re-raised decisions, re-review restating round 1, re-run on unchanged code, etc. or ''",
 "after": {"user_reported_problem": false, "what": "", "category": "", "should_have_been_caught_by": "bugs|conventions|prose|gate|mutation|nobody|n/a",
           "user_disputed_a_finding_or_fix": false, "user_said_checkpoint_unwanted": false},
 "notes": {"proposed": 0, "approved": 0},
 "memory_would_help": "a concrete case where a saved note (a past decision, a known flaky test, a convention ruling) would have avoided work or a wrong finding, else ''",
 "report_ts": "timestamp of the parent's final report, or null if it never reported",
 "confidence": "high|medium|low",   // how reliably the digest lets you fill findings and verdicts
 "evidence": "short pointers: timestamps or quoted phrases (<=12 words each)"}
```

Guidance:

- Copy `session` and `timestamp` from line 1 byte for byte. A changed digit, or a dropped trailing zero, names no run and the whole batch is refused
- Count a finding once. A reviewer's "Questions" or "Pre-existing" lines are not findings unless the parent acted on them or reported them; a reviewer's "none" verdicts are zero findings
- The verdict comes from what the parent did or said, not from the reviewer. Fixed: fixed. Explained why the reviewer was wrong: refuted, with the reason. Reported to the user as a question, or left: open. Confirmed but kept on purpose: deliberate
- For re-review findings, rereview_kind: broken-by-fix (the fix introduced it), missed-first-round (in the original change, missed by round 1), restated-first-round (already raised in round 1)
- "after": judge from the HUMAN lines after the parent's final report and from NEXT HUMAN. Flag user_reported_problem only when the user reports a bug, a wrong fix, a convention breach, a CI or test failure or another defect in the reviewed work; a new feature request is not that. A digest cut off before a human reply leaves it false. Ignore unrelated later work in the window
- If the window holds several checkpoint cycles from one invocation, classify the first and list the later cycles' start times in "evidence"
- If the digest is truncated or ambiguous, lower confidence rather than guess. Never invent findings

When done, reply with only: the output path, the number of lines written, and one line per run that was hard to classify.
