# Files outside git

A file outside git, in a symlinked or unversioned directory, holds work git
never reports, and `ele checkpoint hash` cannot see an edit to it. Where the
change edits one:

- §1: a clean tree is still a checkpoint. The file joins the change list,
  reviewed against a saved copy of what it replaced, or against the file itself
  where there is none
- §3: the brief's change list marks it outside git
- §4: the change is full mode, whatever each freeze printed
- §8: an edit to it since §1 runs the re-review, whatever the fingerprints say

## Capped freeze

A freeze printing `dirty paths past 200, unfingerprinted: <n>` is capped: the
fingerprint hashes 200 dirty paths and names none of the rest, so neither it nor
§8's comparison can see an edit to them. It is handled as a change with a file
outside git:

- §4: the change is full mode
- §8: the re-review always runs, whatever the fingerprints say
