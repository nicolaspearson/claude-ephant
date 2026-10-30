# Delta scope

The freeze decides delta scope. Where the last checkpoint froze a path on the
change list and its record still holds those bytes, the freeze also writes
`checkpoint-<repo>.delta.diff`: each such path against the bytes that checkpoint
reviewed, the rest — and both sides of a rename — against HEAD. It says how many
paths changed since, and the mode it prints is the delta's. A branch freeze
writes no delta.

`0 changed`: nothing is new to review; say so, run `close` (`SKILL.md`, Stopping
early), and stop.

The brief (§3) carries the delta diff too, and the rule that findings fall on
lines the delta changes, the full diff being context; a finding outside it is
reported only where the delta's change makes it newly wrong. A spec finding of a
missing requirement falls on no line and is reported whatever the scope.
