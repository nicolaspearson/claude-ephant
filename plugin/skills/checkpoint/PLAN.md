# Plan item

The item is the one `SKILL.md` §7 read from the plan file; § numbers below are
`SKILL.md`'s.

Judge the item against its own text, not the change's intent. Every requirement
it names met, any real-session test among them, no spec finding confirmed under
§5 left unfixed, and every check §2 listed green under §6: the change completes
it. Propose the tick, with one line under it saying what was asserted and at
which boundary, `<name>` being the item's kebab name. The line names the commit
that will carry it, not made yet, so no sha; in branch scope, name the shas of
the commits already holding the work too:

```markdown
- [x] <item>
  - asserted in the <name> commit: <what was observed, at which boundary>
```

The closing action, which deletes the plan, takes no tick line.

Anything short of that: propose a dated progress line under the item instead,
`as of <date>: <what is done>` and `next: <what is left>`. Code that exists but
whose behaviour nobody has observed gets a progress line, not a tick.

Neither line cites ele data: a note id, a session id or prefix, or an `ele show`
of one resolves only in its author's data directory. Say what was observed, not
where; a session's evidence is described, never pointed at.

Write it on approval, then run `ele plan check` and any CI check reading
`.documentation/` over the plan file; a warning or failure on the written line is
a finding. The plan file joins the change list, and the
write counts as a fix for §8, which re-reviews it: a tick line is the claim a
closure rests on, and its author does not review it.
