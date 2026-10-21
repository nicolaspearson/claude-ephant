---
kind: plan
created: 2026-10-21
---

# ele configuration

## Why

Settle what of ele a user can switch or tune, how that configuration is laid out, and how it is set and read from the CLI, the skills and the guide. To be drawn with the user at pickup, from the inbox below.

## Context

- As of 2026-10-21: the plugin's `userConfig` in `plugin/.claude-plugin/plugin.json` holds one option, `checkpoint_gate` (boolean, default on), read by the checkpoint hooks

## Decisions

## Actions

- [ ] draw-items: Draw the plan's why and items from its inbox with the user, in rounds
- [ ] Propose each out-of-scope line as a `decision` through `/ele:remember`, run `ele plan done .documentation/plans/config.md`, and delete this plan

## Inbox

- 2026-10-21, from the user: auto note writes, switched on and off through config
- 2026-10-21, from the user: other config candidates
- 2026-10-21, from the user: config structure
- 2026-10-21, from the user: config interaction across the CLI, skills and the guide
- 2026-10-21, from the user: Claude heads-up integration

## Out of scope
