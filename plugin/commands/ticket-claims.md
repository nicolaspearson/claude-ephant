---
allowed-tools: Bash(ele ticket claims:*)
description: Report open tickets mistyped as a task or granting themselves execution
disable-model-invocation: true
---

Run the ticket claim check and report what it found:

!`ele ticket claims $ARGUMENTS`

Each finding is a judgement, not a defect. For every one, open the ticket and say
whether the flag holds or is a false positive. A claim made in a ticket's text is
invisible to these label checks and still needs reading.
