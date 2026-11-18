# Close mode

With `close` and one path: the plan's closing action is being worked, and this
mode sends each Decision where it outlives the plan, since the plan's file is
deleted and its reasons with it. It runs alone: none of ticket-audit's sections
and none of `SKILL.md`'s methods. Run it from the plan's checkout, where its
ADRs and spec are found. Sort each line of the plan's `## Decisions` into one
kind:

- **Architectural**: hard to reverse, surprising without context, and the result
  of a real trade-off, all three; failing any part, it is design, since an ADR
  is earned. Where the repository keeps ADRs, found by the ADRs section of
  `reference/plan-practices.md` in the plugin root, draft one ADR a Decision as
  the next number in its series, the topic's in a per-topic series, asked when
  unclear. Follow the format of the series' latest record, or its template where
  it keeps one, the status accepted where the series writes one; its considered
  options are the Decision's `rejected:` lines, and its context, decision and
  every consequence, good and bad, go in the sections that format holds. A
  Decision contradicting an accepted ADR asks whether it supersedes that one,
  its own question. Write the ADR on approval; declined, write none and fold the
  Decision as design. A repository keeping no ADRs folds it as design: it never
  gets one unasked
- **Design**: why what was built behaves as it does, which a later reader would
  otherwise undo. Its reason goes into the spec beside the behaviour it
  explains; where the spec already carries it there, say where and write
  nothing. With no `spec:`, propose it as a `decision` through the
  `ele:remember` skill's Propose section
- **Process**: how the plan was run, its order, sizing or research. It goes with
  the plan; git keeps it

Show the sort, each Decision with its kind, its reason for the kind, and its
fold drafted: the ADR's file name and text, the spec line, the `decision` note,
or where the spec already carries it. Done when the user has approved or changed
each, and each approved fold is written.
