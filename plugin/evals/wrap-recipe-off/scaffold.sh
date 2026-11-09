#!/usr/bin/env bash
set -euo pipefail

data="$HOME/Library/Application Support/claude-ephant"
mkdir -p "$data"
printf '{"owners": ["eval"], "paths": {"%s": "eval", "%s": "eval"}}\n' "$PWD" "$(pwd -P)" > "$data/owner-map.json"
# A repository of its own: git otherwise roots at the sandbox HOME, whose files read as uncommitted work (as of 2026-10-23). Whether that git ran in ele's hooks or the session is not checked.
# The session cannot run /usr/bin/git here: Apple's shim cannot write its xcrun cache (as of 2026-10-31, claude 2.1.308).
git init -q
git -c user.name=eval -c user.email=eval@example.invalid commit -q --allow-empty -m init
printf '{"recipes": false}\n' > "$data/config.json"
