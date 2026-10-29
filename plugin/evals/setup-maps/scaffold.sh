#!/usr/bin/env bash
set -euo pipefail

# The skill never offers HOME, so the graders expect the root, HOME/cwd, mapped
[ "$PWD" = "$HOME/cwd" ] || { echo "setup-maps expects the working directory at $HOME/cwd, got $PWD" >&2; exit 1; }

data="$HOME/Library/Application Support/claude-ephant"
mkdir -p "$data"
printf '{"notes_ask": false}\n' > "$data/config.json"
git init -q
