# Changelog

## [0.12.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.11.0...v0.12.0) (2026-10-23)


### Features

* **checkpoint:** port the checkpoint hooks and state scripts to ele checkpoint ([33168d0](https://github.com/nicolaspearson/claude-ephant/commit/33168d0f9cb77aa8f9b1bd7e59051cec53b24f1c))
* **config:** auto_note_writes writes proposed notes without asking; task install builds into the release slot ([a90c3c0](https://github.com/nicolaspearson/claude-ephant/commit/a90c3c0906d4264867b86dfcc66a659b8d3ee144))
* **config:** colour option turns off the start block's and Stop hook's highlighting ([8da515c](https://github.com/nicolaspearson/claude-ephant/commit/8da515c0cc45947af3d5c15349adc68d55ce6eec))
* **handoff:** a handoff's first line points at the next plan item or issue URL; wrap skips it only with one active plan and no ticket ([f809502](https://github.com/nicolaspearson/claude-ephant/commit/f809502eef65f0659a7fad848334bdea6775f7af))
* **handoff:** mark guesses, durable paths only, name plan items, take a focus argument ([cc59379](https://github.com/nicolaspearson/claude-ephant/commit/cc59379bbabe5e984145c928d2e8c84ff16ea265))
* **reminders:** dated reminders shown at session start from their due date until closed ([fb12e2f](https://github.com/nicolaspearson/claude-ephant/commit/fb12e2faea48710b7d3ff3e9faf67700019dff6c))
* **reminders:** ele reminder list --colour highlights as the start block does; the plain list drops control characters ([c1be2e1](https://github.com/nicolaspearson/claude-ephant/commit/c1be2e1f67ea3b35a412b0c278d9107fba8895aa))
* **reminders:** nudges come back N days after each close, or after a done file changes ([0cc19d4](https://github.com/nicolaspearson/claude-ephant/commit/0cc19d4eae0bd82f1b2bd71c316eed98f9283f55))
* **reminders:** start block's reminders header bold, due date yellow, a nudge's ↻ and interval magenta ([4d4f8d7](https://github.com/nicolaspearson/claude-ephant/commit/4d4f8d789776bd4d72267c1ac96327cd61df46ba))
* **session-start:** keep the start block under Claude Code's hook output cap, cutting oldest questions past it ([50db5c9](https://github.com/nicolaspearson/claude-ephant/commit/50db5c948e7c959e76cd4861ff858d3c9db4b14e))
* **wrap:** add ele:wrap, a session wrap-up that asks before recording ([0d2ff6f](https://github.com/nicolaspearson/claude-ephant/commit/0d2ff6f09cc933449b188b47d8c9d0a011de59a1))


### Bug Fixes

* **checkpoint:** the orphan commit in TestBaseTakesTheUpstreamElseOriginsDefaultBranch sets its own git identity, so CI passes ([63c6b75](https://github.com/nicolaspearson/claude-ephant/commit/63c6b7587f30779dd5138a7a9f80333cdf79ca64))
* **stop-check:** a proposal's own calls, cd-prefixed ele remember and read-only Bash stop asking for notes ([e8be496](https://github.com/nicolaspearson/claude-ephant/commit/e8be4964af093a08d84fbc30587bb6464ad5a750))
* **stop-check:** a quoted heredoc in a note's body is a read, a read after a note marks mid-task; TestAdvance sorted ([7f042de](https://github.com/nicolaspearson/claude-ephant/commit/7f042de8e14df4077362c083c78c4ebd346e00e2))
* **stop-check:** a read in a note's substitution marks mid-task; quoted ) and &lt;&lt; inside $( ) no longer end it ([d170ef9](https://github.com/nicolaspearson/claude-ephant/commit/d170ef9451ad27ad5f22d47734a5001bd0035c8d))
* **stop-check:** a write chained after ele remember is still work; eval graders read past escaped quotes and newlines ([7897e10](https://github.com/nicolaspearson/claude-ephant/commit/7897e10048a441eeded71acaa586ae13d62eed9c))
* **wrap:** end line names deferred and skipped work; wrap evals get their own repository ([8e7f9ef](https://github.com/nicolaspearson/claude-ephant/commit/8e7f9ef3fcd8094a69b7a0640f4465f19b2657d8))
* **wrap:** propose a handoff unless the start block covers the next step; write it after the rest are answered ([19e931b](https://github.com/nicolaspearson/claude-ephant/commit/19e931bb7bf37b784c1a8bf6ca21b0cbfac3afae))

## [0.11.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.10.0...v0.11.0) (2026-10-22)


### ⚠ BREAKING CHANGES

* **config:** ele config file with per-owner overrides; checkpoint_gate moves off userConfig

### Features

* **config:** ele config file with per-owner overrides; checkpoint_gate moves off userConfig ([bfc4872](https://github.com/nicolaspearson/claude-ephant/commit/bfc487297e9ca903eb2578d6a7a584708a07d9cf))
* **events:** follow each checkpoint block to its paths, reply and checkpoint; add ele show --at ([6edaa43](https://github.com/nicolaspearson/claude-ephant/commit/6edaa4379a32cd3d8679fcf249d4f8bb4e635472))


### Bug Fixes

* **events:** count n't a checkpoint as a dispute ([d3a017c](https://github.com/nicolaspearson/claude-ephant/commit/d3a017c3a2235631088608e8c44de20be4f399a0))
* **mine:** cut units at exact checkpoint names and the turn that records ([bc27b81](https://github.com/nicolaspearson/claude-ephant/commit/bc27b81cea99192bda69c052390331493dd48b0f))
* **mine:** open a checkpoint run on its call when another line shares its time ([c34957a](https://github.com/nicolaspearson/claude-ephant/commit/c34957a4dc38b275c318e15df50380b094b7ee01))
* **mine:** read a compaction summary as no human message ([49e1501](https://github.com/nicolaspearson/claude-ephant/commit/49e1501f533cacf845cb6e741e6478f4d8d09b0e))

## [0.10.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.9.0...v0.10.0) (2026-10-22)


### Features

* **remember:** --supersedes repeats so one note replaces several ([5157085](https://github.com/nicolaspearson/claude-ephant/commit/5157085829866a7fe321dd9731eb8cae98d124a6))


### Bug Fixes

* **plan:** check as-of dates in headings ([09246b6](https://github.com/nicolaspearson/claude-ephant/commit/09246b6bccdb83acfb8c1f15027825406d4aa6df))
* **plan:** resolve a root-relative plan path from a subdirectory ([b869c68](https://github.com/nicolaspearson/claude-ephant/commit/b869c68dc57c9d8cbda8b66ba602ef928c7c03fa))

## [0.9.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.8.0...v0.9.0) (2026-10-22)


### Features

* **eval:** recall smoke case and task eval ([d5a7faa](https://github.com/nicolaspearson/claude-ephant/commit/d5a7faa36836f3902d4494d29e1393fc43191c15))

## [0.8.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.7.0...v0.8.0) (2026-10-22)


### Features

* **mine:** ele mine --report, figures from the last stored judgment per unit ([d4db6d6](https://github.com/nicolaspearson/claude-ephant/commit/d4db6d648e716769adb3d0cad178f940fe99d050))
* **mine:** time each judged unit's window from its transcript; report-figures done ([76fea73](https://github.com/nicolaspearson/claude-ephant/commit/76fea732d54de2e7757e078bbe9e20e8ee063a9d))


### Bug Fixes

* **checkpoint:** tick line names the item, not its position; light-checkpoint ticked ([5da5497](https://github.com/nicolaspearson/claude-ephant/commit/5da5497caefe9813ef7013dec0b4b5ea0beac1dd))

## [0.7.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.6.0...v0.7.0) (2026-10-22)


### Features

* **plan-audit:** ele:plan-audit over every active plan; ticket corpus takes --plan repeated ([c70693d](https://github.com/nicolaspearson/claude-ephant/commit/c70693df9c60a11a093f9051b77f0cb9d6a7a41a))

## [0.6.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.5.0...v0.6.0) (2026-10-21)


### Features

* **checkpoint:** shape check in conventions; plan step before re-review, ele plan check in the gate ([a2d9787](https://github.com/nicolaspearson/claude-ephant/commit/a2d978767131b4d3269091fec753f03984d69a0d))
* **plan:** ele plan check lists stale blockers, dates, claims, inbox items and readiness to close ([b073aba](https://github.com/nicolaspearson/claude-ephant/commit/b073aba813e3cb0d3103a6441c414d5179442cdb))

## [0.5.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.4.0...v0.5.0) (2026-10-21)


### Features

* **plan:** blocked: edges by item name, after: dates, closed-plan store ([630f5a0](https://github.com/nicolaspearson/claude-ephant/commit/630f5a04008afd344d8b89460051b22f5b5e2d56))
* **plan:** items lead with a kebab name; ledger keys on it ([2e04e2a](https://github.com/nicolaspearson/claude-ephant/commit/2e04e2a1caf745b6a265d9936008ea1447df975b))

## [0.4.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.3.0...v0.4.0) (2026-10-21)


### Features

* **events:** --during &lt;skill&gt; keeps a run's events ([29412de](https://github.com/nicolaspearson/claude-ephant/commit/29412def687070473485626e1549dd4510a3465a))
* **events:** keep a tool error's tail; error tables classed from it ([1f21437](https://github.com/nicolaspearson/claude-ephant/commit/1f21437d3a81ffe8639204027e2ccb9732035b98))
* **install:** Homebrew tap and install.sh, checked after each release ([53f8c96](https://github.com/nicolaspearson/claude-ephant/commit/53f8c96990456d13bce6b9d42c095a99ef17f71a))

## [0.3.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.2.0...v0.3.0) (2026-10-21)


### Features

* ele version and task upgrade, checked against the latest release ([da7243f](https://github.com/nicolaspearson/claude-ephant/commit/da7243f42f60bb257e7139b34f8f81a5cfeda743))

## [0.2.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.1.1...v0.2.0) (2026-10-21)


### Features

* **checkpoint:** spec reviewer for plan items and tickets ([ffecf6f](https://github.com/nicolaspearson/claude-ephant/commit/ffecf6f944e8665e950e6e8e48f7b3d49a9ec23f))
* **events:** tool-error events, and index entries by source ([4fafa3b](https://github.com/nicolaspearson/claude-ephant/commit/4fafa3b17999fd99f06269e4e2edabd01e06af2f))

## [0.1.1](https://github.com/nicolaspearson/claude-ephant/compare/v0.1.0...v0.1.1) (2026-10-21)


### Miscellaneous Chores

* release 0.1.1 ([d48fd6f](https://github.com/nicolaspearson/claude-ephant/commit/d48fd6f9335cf91ca2c60bf2d75cba2354b6f535))
