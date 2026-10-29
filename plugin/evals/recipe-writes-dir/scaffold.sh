#!/usr/bin/env bash
set -euo pipefail

data="$HOME/Library/Application Support/claude-ephant"
mkdir -p "$data" "$HOME/agent-skills" "$HOME/.claude-a/projects" "$HOME/.claude-b/projects" "$HOME/.claude-b/skills/renew-dev-cert"
printf '{"owners": ["eval"], "paths": {"%s": "eval", "%s": "eval"}}\n' "$PWD" "$(pwd -P)" > "$data/owner-map.json"
printf '{"notes_ask": false, "recipe_dir": "%s/agent-skills"}\n' "$HOME" > "$data/config.json"
printf -- '---\nname: renew-dev-cert\ndescription: An older skill by the same name.\n---\n' > "$HOME/.claude-b/skills/renew-dev-cert/SKILL.md"
