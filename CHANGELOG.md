# Changelog

## [0.12.0](https://github.com/nicolaspearson/claude-ephant/compare/v0.11.0...v0.12.0) (2026-10-29)


### Features

* **checkpoint:** a test is never changed to pass; bugs.md reports one ([ad4eaf5](https://github.com/nicolaspearson/claude-ephant/commit/ad4eaf561046eef2c47e74c55348a5ce5c95235d))
* **checkpoint:** log and stats take review records ([27813ea](https://github.com/nicolaspearson/claude-ephant/commit/27813ea7d42f3474f23dc754f1d31d5393e8a72f))
* **checkpoint:** port the checkpoint hooks and state scripts to ele checkpoint ([33168d0](https://github.com/nicolaspearson/claude-ephant/commit/33168d0f9cb77aa8f9b1bd7e59051cec53b24f1c))
* **config:** auto_note_writes writes proposed notes without asking; task install builds into the release slot ([a90c3c0](https://github.com/nicolaspearson/claude-ephant/commit/a90c3c0906d4264867b86dfcc66a659b8d3ee144))
* **config:** colour option turns off the start block's and Stop hook's highlighting ([8da515c](https://github.com/nicolaspearson/claude-ephant/commit/8da515c0cc45947af3d5c15349adc68d55ce6eec))
* **config:** notes_ask and compact_hint_at options, the Stop hook's notes ask and size hint switched through config ([ed21d31](https://github.com/nicolaspearson/claude-ephant/commit/ed21d31f98fe4839031a24165190792abd27c735))
* **config:** review_voice, a path option settable per owner ([5e2524f](https://github.com/nicolaspearson/claude-ephant/commit/5e2524f7a0a3a35d9a252e703c0f53db82e91097))
* **flag:** ele flag good|bad and /ele:flag stamp a session moment, listed by ele flag list (session-flags) ([6e001c2](https://github.com/nicolaspearson/claude-ephant/commit/6e001c2f235afe8547b26660ba6e051fe089440b))
* **handoff:** a handoff's first line points at the next plan item or issue URL; wrap skips it only with one active plan and no ticket ([f809502](https://github.com/nicolaspearson/claude-ephant/commit/f809502eef65f0659a7fad848334bdea6775f7af))
* **handoff:** lone_handoff_pending option and /ele:handoff list (handoff-picking) ([d33b749](https://github.com/nicolaspearson/claude-ephant/commit/d33b7490ca1e49522e270945c339eb803c1213c9))
* **handoff:** lone_handoff_pending option and /ele:handoff list (handoff-picking) ([a2a9507](https://github.com/nicolaspearson/claude-ephant/commit/a2a95074f9ed8ffc58d720ab61608e1e908b7882))
* **handoff:** mark guesses, durable paths only, name plan items, take a focus argument ([cc59379](https://github.com/nicolaspearson/claude-ephant/commit/cc59379bbabe5e984145c928d2e8c84ff16ea265))
* **index:** notes-ask hook blocks carry the work before them and whether a note followed ([d568535](https://github.com/nicolaspearson/claude-ephant/commit/d5685352f8a6a7aff22c56d3fe0270a49f1b4057))
* **notes:** an everywhere scope, a note saved once for every owner and every session ([63ff283](https://github.com/nicolaspearson/claude-ephant/commit/63ff283241d82765a9d2a9ec08defce8464a9c39))
* **plan:** /ele:plan new and convert ground their rounds and run a 2-to-3-pass audit and review loop before start (review-loop) ([8ad0016](https://github.com/nicolaspearson/claude-ephant/commit/8ad0016d47ea872c8970fb7dace7ea138cbf7523))
* **plan:** /ele:plan-audit close sorts a closing plan's Decisions into ADR, spec or plan by the three-part test, every closing action naming it (decision-fold) ([59be954](https://github.com/nicolaspearson/claude-ephant/commit/59be95466e8bbac6faec360a646896c236472293))
* **plan:** ele plan check and lint-docs flag ele data in plans; the repository's plans describe what was observed instead ([00d2bee](https://github.com/nicolaspearson/claude-ephant/commit/00d2bee9e366ba9f5e42e8fc517f58114e452c4b))
* **plan:** ele plan check warns on an open unnamed action and on a closing action ticked with others open (unnamed-action-warning) ([8b98642](https://github.com/nicolaspearson/claude-ephant/commit/8b9864234bb71e2612a9103561e301633abf2647))
* **plan:** ele plan start names the plans it could not check for a name clash, and ele plan done stores a closed plan's missing record (start-clash-unchecked); tick new-eval ([fbe9256](https://github.com/nicolaspearson/claude-ephant/commit/fbe92560ded38c8280ed94aec8d71b613e2455c0))
* **plan:** ele:plan-reviewer reviews a draft plan against its requirements, spec, conventions and accepted ADRs (plan-reviewer) ([77cf68d](https://github.com/nicolaspearson/claude-ephant/commit/77cf68daeba0d47d5482fedee8c5800e0f94d0c6))
* **plan:** plan-audit's Drafts mode audits unstarted drafts beside every active plan, reporting severities and asking nothing (drafts-mode) ([3838cd4](https://github.com/nicolaspearson/claude-ephant/commit/3838cd45fa641126d19b79372630f570615f9051))
* **plan:** the ticket corpus keys an Inbox item on its title, naming only an action (ticket-inbox-names) ([a1a2803](https://github.com/nicolaspearson/claude-ephant/commit/a1a280352e7af32709afe0d438919fe37b28d578))
* **recall:** sync indexes a sent heads up prompt as kind heads-up, a turn start like a prompt (sent-heads-ups) ([7e53104](https://github.com/nicolaspearson/claude-ephant/commit/7e5310431bd838b0f71a2d07383bc1896ac11fe7))
* **recipe:** ele:recipe audit checks the user's skills for stale commands, flags and versions ([493f088](https://github.com/nicolaspearson/claude-ephant/commit/493f088c21b384ab80840841bd93aeaccf00c2fe))
* **recipe:** ele:recipe keeps a procedure as a skill in every profile, offered by ele:wrap behind the recipes option ([082ef71](https://github.com/nicolaspearson/claude-ephant/commit/082ef716e08edbbc4c222b8daa2fd3cde5dfd585))
* **recipe:** recipe_dir option writes a recipe once and links each profile (recipe-dir) ([d7cde07](https://github.com/nicolaspearson/claude-ephant/commit/d7cde07859d0124101fadd4bfb78d9ed11da4a90))
* **remember:** Propose asks four Write/Skip questions per call and as many calls as it takes; a remember-many eval proposes five ([5e54c79](https://github.com/nicolaspearson/claude-ephant/commit/5e54c7962096e159d017c9c9ba5bf43f4d6f91d0))
* **reminder:** --every takes minutes, hours or days and --due a time ([4f7a8c9](https://github.com/nicolaspearson/claude-ephant/commit/4f7a8c9a0c34c6bd8099825ee09548cc1a0bc976))
* **reminders:** dated reminders shown at session start from their due date until closed ([fb12e2f](https://github.com/nicolaspearson/claude-ephant/commit/fb12e2faea48710b7d3ff3e9faf67700019dff6c))
* **reminders:** ele reminder list --colour highlights as the start block does; the plain list drops control characters ([c1be2e1](https://github.com/nicolaspearson/claude-ephant/commit/c1be2e1f67ea3b35a412b0c278d9107fba8895aa))
* **reminders:** nudges come back N days after each close, or after a done file changes ([0cc19d4](https://github.com/nicolaspearson/claude-ephant/commit/0cc19d4eae0bd82f1b2bd71c316eed98f9283f55))
* **reminders:** start block's reminders header bold, due date yellow, a nudge's ↻ and interval magenta ([4d4f8d7](https://github.com/nicolaspearson/claude-ephant/commit/4d4f8d789776bd4d72267c1ac96327cd61df46ba))
* **review:** ele:review skill, its reader agent and docs ([6639315](https://github.com/nicolaspearson/claude-ephant/commit/6639315e464e9d47cf45678d7f5a505d1f4b2ade))
* **review:** second list, fixed Raw Markdown shape, merge state and follow-ups from a first real review ([b859135](https://github.com/nicolaspearson/claude-ephant/commit/b8591350b78a64fcaf0b5470b2a71f1261258836))
* **session-start:** keep the start block under Claude Code's hook output cap, cutting oldest questions past it ([50db5c9](https://github.com/nicolaspearson/claude-ephant/commit/50db5c948e7c959e76cd4861ff858d3c9db4b14e))
* **setup:** /ele:setup maps an unowned directory through ele owner, named by the start block (onboarding) ([446ded4](https://github.com/nicolaspearson/claude-ephant/commit/446ded49d3daa80655030d957e3a046f32085ba4))
* **stopcheck:** replay the notes ask over archived stops ([dd54e24](https://github.com/nicolaspearson/claude-ephant/commit/dd54e24630b3c79ab2f0564b9dd3a824cfb9e596))
* **wrap:** add ele:wrap, a session wrap-up that asks before recording ([0d2ff6f](https://github.com/nicolaspearson/claude-ephant/commit/0d2ff6f09cc933449b188b47d8c9d0a011de59a1))


### Bug Fixes

* **backup:** copy plans/ so a restore keeps closed plans' item names (backup-plans) ([ae52352](https://github.com/nicolaspearson/claude-ephant/commit/ae5235296170b04c874ed9bb6d59b929c5072254))
* **checkpoint:** the orphan commit in TestBaseTakesTheUpstreamElseOriginsDefaultBranch sets its own git identity, so CI passes ([63c6b75](https://github.com/nicolaspearson/claude-ephant/commit/63c6b7587f30779dd5138a7a9f80333cdf79ca64))
* **index:** hook-block checkpointed counts a hand-run checkpoint; reply skips a notes-ask answer at the same Stop ([9c1903a](https://github.com/nicolaspearson/claude-ephant/commit/9c1903aba2e1a0d23ed1fadb146b9e0cab7be257))
* **plan:** plan check warns only on ele data ele resolves; ele show finds a prefix by range (plan-check-ids) ([d37da00](https://github.com/nicolaspearson/claude-ephant/commit/d37da00d589864c7bdb2adf2bf0dd31280e62a1c))
* **plan:** stored plans are renamed into place so a backup never copies a torn one (plan-write-atomic) ([42a5829](https://github.com/nicolaspearson/claude-ephant/commit/42a5829f120e3715bfa83f2f09c83a999bc95ea9))
* **recipe:** always write a recipe to the running profile, whatever it holds ([cccc246](https://github.com/nicolaspearson/claude-ephant/commit/cccc246a8c6d2bf7799c9bcf70ee59efbbff8b74))
* **recipe:** audit leaves man-page flags, package runners and delegating scripts out of findings ([6aa7ac9](https://github.com/nicolaspearson/claude-ephant/commit/6aa7ac99f9e9e767c13a18ca5ae19d186207e7a7))
* **recipe:** audit reads man pages in the C locale so flags keep ASCII hyphens (man-hyphens) ([282883d](https://github.com/nicolaspearson/claude-ephant/commit/282883d6d040ba081c7e9b845dc0c7c42e848ba7))
* **recipe:** no-run-step grader catches the command shapes no-run-script does (no-run-step-shapes) ([06b6009](https://github.com/nicolaspearson/claude-ephant/commit/06b60091825414673e278137aa802b9de71c4f81))
* **recipe:** report says when recipe_dir is unset and how to set it ([4cd535b](https://github.com/nicolaspearson/claude-ephant/commit/4cd535b001bf5f0b2370d0adf0f9adb27673cb7c))
* **recipe:** skill runs man in the C locale (man-hyphens) ([700c8ed](https://github.com/nicolaspearson/claude-ephant/commit/700c8edca5360bd8e9b67277298eeaaa0e5070b4))
* **review:** split the walk-through into summary, findings, second list and merge state, each with a done-when ([c6da191](https://github.com/nicolaspearson/claude-ephant/commit/c6da1911fdc4dc18a78ed6472db28ee4fd833745))
* **review:** steps 8 to 12 print by pausing, each print the turn's closing reply ([7199de1](https://github.com/nicolaspearson/claude-ephant/commit/7199de1f50139e03a683c014041c971526418912))
* **review:** the second list joins the last findings call when it has room; reword review-skill's pending-review done-when ([014bd63](https://github.com/nicolaspearson/claude-ephant/commit/014bd63b22e40e35b1c81131fecf6f7bb72107cf))
* **review:** the walk-through shows everything in questions; second-list findings get a table; a pending review is read up front ([dd06277](https://github.com/nicolaspearson/claude-ephant/commit/dd062779191900eb801906e92b5ca4a5e5338db9))
* **review:** with a pending review held, the publish question offers Check again instead of Publish ([8d7c596](https://github.com/nicolaspearson/claude-ephant/commit/8d7c59606ea7f97fa350acc57034b16adcb71af0))
* **stop-check:** a proposal's own calls, cd-prefixed ele remember and read-only Bash stop asking for notes ([e8be496](https://github.com/nicolaspearson/claude-ephant/commit/e8be4964af093a08d84fbc30587bb6464ad5a750))
* **stop-check:** a quoted heredoc in a note's body is a read, a read after a note marks mid-task; TestAdvance sorted ([7f042de](https://github.com/nicolaspearson/claude-ephant/commit/7f042de8e14df4077362c083c78c4ebd346e00e2))
* **stop-check:** a read in a note's substitution marks mid-task; quoted ) and &lt;&lt; inside $( ) no longer end it ([d170ef9](https://github.com/nicolaspearson/claude-ephant/commit/d170ef9451ad27ad5f22d47734a5001bd0035c8d))
* **stop-check:** a recipe run keeps earlier work pending for the notes ask ([055a8c0](https://github.com/nicolaspearson/claude-ephant/commit/055a8c0372bf3fa6b363b3dfda7375e792b90b99))
* **stop-check:** a write chained after ele remember is still work; eval graders read past escaped quotes and newlines ([7897e10](https://github.com/nicolaspearson/claude-ephant/commit/7897e10048a441eeded71acaa586ae13d62eed9c))
* **stop-check:** no notes ask during a review, from its readers' launch to its log line ([775f7ae](https://github.com/nicolaspearson/claude-ephant/commit/775f7ae0f9ec37a2968c2955715943ca358504b6))
* **stopcheck:** a notes proposal clears a subagent's earlier edit ([8427cbb](https://github.com/nicolaspearson/claude-ephant/commit/8427cbbc4a63cd66328ef296c6a57de61082f380))
* **stopcheck:** an ele remember --help lookup is a read, not a note ([7a9e304](https://github.com/nicolaspearson/claude-ephant/commit/7a9e3048e1788a31150d13d3f2780ebbed822b5c))
* **test:** rename shadowed variables in the everywhere-note session-start test ([40080b6](https://github.com/nicolaspearson/claude-ephant/commit/40080b65da00ce9cad5851188130765572017ef2))
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
