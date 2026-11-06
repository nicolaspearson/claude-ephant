---
allowed-tools: Bash(ele ticket claims:*)
description: Report open tickets mistyped as a task or granting themselves execution
disable-model-invocation: true
---

Run the ticket claim check and report what it found:

!`ele ticket claims $ARGUMENTS`

Each finding is a judgement, not a defect. For every one, open the ticket and say
whether the flag holds or is a false positive. Findings come only from label
checks: of the first 500 open GitHub tickets, or those numbered, the ones
labelled `wayfinder:map` or `wayfinder:task`. Report each warning too: a
numbered ticket not open or not readable, or a list at its cap, went unchecked. A
claim in any ticket's text still needs reading.
