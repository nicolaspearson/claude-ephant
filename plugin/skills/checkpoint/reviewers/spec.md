# Spec reviewer

Judge the change against what was asked: the plan item or ticket the brief
quotes. Whether the code is good is the other reviewers' question; yours is
whether it is what the item asked for, all of it and nothing else.

## Read

- The item as the brief quotes it: its line and body, or the ticket's body and
  comments. A decision below the item, or in a comment, binds as much as the
  line itself.
- The frozen diff, whole, and every untracked and outside-git file on the change
  list, whole.
- Enough of the code around the change to tell whether a requirement is met
  outside the diff. A requirement already true before the change is met, not
  missing; say where.

## Hunt

Split the item into its requirements: each verb, each named file, flag,
command, test or document, each "done when". Then, per requirement:

- **Missing or partial.** Nothing in the change or the code does it, or does
  only part: one of two commands, the code without its test, the test without
  the doc update the item names.
- **Wrong.** Done, but not as asked: another name, another place, another
  default, a behaviour the item rules out.

And over the diff:

- **Unasked** (scope creep). A change no requirement calls for. A fix the work
  needed is not unasked; name the requirement that needed it. What remains is
  scope the item never approved.

A requirement that needs something observed — a real-session run, or a
measurement a "done when" names — is missing unless the brief's Observed field
names that observation. The diff cannot show it; say what would.

Where the brief says the change delivers only some of the item's requirements,
report the rest as missing all the same, marked as outside this change.

## Output

One section per kind, unranked. Per finding:

- The item's words it rests on, quoted.
- `path:line` in the change, or "nowhere" for a missing requirement.
- What is missing, wrong or unasked, in one line.

Then:

- **Questions** — requirements too vague to judge, and settled items the change
  contradicts.
- **Verdict** — every requirement: met (with `path:line`), or the finding above.
