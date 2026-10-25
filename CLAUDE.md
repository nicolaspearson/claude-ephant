# claude-ephant

`ele`, a Go binary, and the `ele` Claude Code plugin under `plugin/`.
[CONTRIBUTING.md](CONTRIBUTING.md) has the
[layout](CONTRIBUTING.md#repository-layout), the
[task targets](CONTRIBUTING.md#build-and-run) and the
[release flow](CONTRIBUTING.md#releases).

## Before calling work done

- `task check FORCE=1` and `task audit`: together, what CI runs
- `task format` after editing Markdown

## The ele this session runs

The plugin's hooks call `ele` from `PATH`, this session's hooks included.
`task install` replaces that binary with the clone build, so a broken build
breaks the running session's hooks; `task upgrade` puts the release back.

## Docs

- `README.md` and `docs/` are for users; build and release material goes in
  `CONTRIBUTING.md`
- The specs live in `.documentation/specs/`. They and `docs/` describe the
  current design: behaviour, constraints and the reason for each decision.
  History stays in git. A dated `as of` belongs only on a fact about Claude
  Code's behaviour that a decision rests on
- A change to behaviour updates the spec and `docs/` page that describe it in
  the same change, each command or flag checked against `ele <command> --help`
- Cite a package and an identifier, never a `.go` or `.html` line number:
  `task lint-docs` fails one under `docs/` or `.documentation/`

## Plans

`.documentation/plans/` holds ele plans, worked with `/ele:plan`;
[docs/plans.md](docs/plans.md) has the format. A plan never cites ele data, a
note or session id, which resolves only in its author's data directory: delete
provenance, describe what was observed as evidence, and move a dependency into
the plan.

## Evals

`task eval` runs real sessions on the user's account and costs model calls:
run it only when asked.
