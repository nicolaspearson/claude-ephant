# Log fields

The values each field of `ele checkpoint log`'s record takes.

- `also_raised_by`: optional; the other sources that raised a finding merged into
  this one, so the severity table still credits each.
- `change`: files on the change list, and lines added plus removed.
- `mode`: `full`, `small`.
- `mutations`: `run`; `failed` for the reason stated; `inconclusive`, a mutation
  that did not compile or did not land. One that landed and did not bite is a
  finding, stage `mutation`, not a count.
- `notes`: both 0; ele's Stop hook proposes notes outside the checkpoint.
- `origin`: `change`, `earlier` (code older than the change), `fix` (introduced
  by a fix in this run).
- `rereview`: `ran`, `skipped` (§8 found the fingerprint unmoved).
- `scope`: `uncommitted`, `delta` (the freeze wrote a delta), `branch`.
- `severity`: `blocker` (wrong on the main path: wrong output, lost data, a broken
  build), `defect` (wrong only in a reachable edge case), `quality` (correct, but
  misleading a reader, untested, against a convention other than wording or
  order, or a Shape finding), `nit` (wording or order alone, misleading no one).
- `source`: free text; the reviewer (`bugs`, `conventions` or `prose` by the
  merged reviewer's section, or `spec`), the gate check by name, or `user`.
- `stage`: `gate`, `mutation`, `rereview`, `review`, `user`. A finding the user
  raises after this line is written goes in the next checkpoint's record, stage
  `user`, origin `earlier`.
- `verdict`: `deliberate` (confirmed and left on purpose), `fixed`, `open`
  (reported as a question), `refuted`.
