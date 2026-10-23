#!/usr/bin/env bash
set -euo pipefail

data="$HOME/Library/Application Support/claude-ephant"
mkdir -p "$data"
printf '{"owners": ["eval"], "paths": {"%s": "eval", "%s": "eval"}}\n' "$PWD" "$(pwd -P)" > "$data/owner-map.json"
# Gate off: git roots the sandbox at HOME, so ele's own state files read as changed and the gate replaces the graded reply.
printf '{"checkpoint_gate": false}\n' > "$data/config.json"
