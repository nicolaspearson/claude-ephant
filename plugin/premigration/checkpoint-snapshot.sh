#!/usr/bin/env bash
# Records the working tree as a turn begins, so the Stop hook can tell a turn
# that changed files from one that only asked a question over a dirty tree.
set -uo pipefail

# An ele too old for config, or a config.json that does not parse, leaves the gate on.
[ "$(ele config get checkpoint_gate 2> /dev/null)" = false ] && exit 0

# shellcheck source=checkpoint-state.sh
. "$(dirname "${BASH_SOURCE[0]}")/checkpoint-state.sh"

input=$(cat)
session=$(printf '%s' "$input" | jq -r '.session_id // empty' 2>/dev/null)

root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
[ -n "${root:-}" ] || exit 0

state_file=$(checkpoint_state_file "$session" "$root") || exit 0

# The baseline spans a stop-to-stop cycle, not a prompt. A message sent while
# the turn is still running would otherwise re-baseline over edits already made
# and the Stop hook would see a turn that changed nothing. The Stop hook clears
# the file, so an existing one means the turn is still going.
[ -e "$state_file" ] && exit 0

# Renamed into place: a partial baseline reads every missing path as moved.
checkpoint_write_state "$root" > "$state_file.tmp" 2>/dev/null && mv "$state_file.tmp" "$state_file"

exit 0
