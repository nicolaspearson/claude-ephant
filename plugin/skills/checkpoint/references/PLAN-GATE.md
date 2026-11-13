# Plan files on the change list

§ numbers below are `SKILL.md`'s.

The gate also runs `(cd <root> && ele plan check <path>)` for each plan file
still on disk. For a change list of plan files alone, §2's check list is those
runs plus any CI check that reads `.documentation/`, such as a docs lint; §10
names the rest as not run, not as unverified. It exits 0 on warnings, so read
them from the log: one on a line the change touched is a finding, stage `gate`,
source `ele plan check`; any other, a warning with no line included, is named in
§10.
