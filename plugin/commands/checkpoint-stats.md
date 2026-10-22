---
description: What checkpoints found and which stage found it, to tune the ele:checkpoint skill
disable-model-invocation: true
---

Run the measurement and read it back:

```bash
ele checkpoint stats $ARGUMENTS
```

Argument is a number of days; the default is 30. The checkpoints from before
the log starts are not in it: `ele mine --report` reports them, from the
judgments `/ele:mine` stored.

Report what each section says about the skill, not the code:

- **Who found what.** A reviewer whose findings are often refuted has guidance
  that misleads it; name it and the categories it gets wrong. A reviewer that
  finds little across many full-mode rows may not earn its cost. This table
  credits only a finding's source; read a reviewer's yield from the severity
  table, which credits merged duplicates too.
- **By source and severity.** A reviewer whose confirmed findings are mostly
  nits costs a full pass for wording; one that finds the blockers earns it.
  `shared` counts findings another source also raised: a reviewer whose findings
  are nearly all shared adds little the others did not find. Records logged
  before severity existed show as unrated.
- **Re-reviews.** Counted only where the round ran. Findings broken by a fix are
  its reason to exist; findings missed by the first review say the first pass
  has a gap. Few of either over many rounds is the case for dropping or
  narrowing it.
- **Caught after review.** Gate catches are what reviewers leave to tools, which
  is fine for formatting and lint and a gap for anything else. Mutation and user
  catches are what the whole review missed.
- **Older than the change.** Recurring categories here are what earlier work let
  through, and what the skill does not look for yet.
- **Mutations.** A high inconclusive share means mutations are chosen badly:
  they do not compile or their anchors miss. Weak tests show as mutation-stage
  findings instead.
- **Duration.** From the first freeze to the log line, so it includes the
  reviewers, the fixes, the re-review, and any wait for the user's reply before
  the log line is written. A re-review frozen more than two hours after the
  previous freeze, or any freeze with no session id, restarts the clock there. Full mode's median beside small's is the
  cost to weigh against what the extra reviewers find. Runs logged before
  duration existed, or with no freeze behind them, carry none; say how many.

A finding the user raised after a report is logged with the next checkpoint, so
it carries that run's mode. Say how many checkpoints the window holds. Under ten, call every pattern a
guess. A window reaching back before the date the log starts, which the report
prints, is short by that much.
