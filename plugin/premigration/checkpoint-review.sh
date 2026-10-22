#!/usr/bin/env bash
# Refuses to finish a turn that changed files and says the work is done without
# running the checkpoint skill or reporting its reviews.
#
# Written because the convention alone did not hold. The failure mode is
# specific: finding something during the build feels like having reviewed, so
# the deliberate passes get skipped.
set -uo pipefail

# An ele too old for config, or a config.json that does not parse, leaves the gate on.
[ "$(ele config get checkpoint_gate 2> /dev/null)" = false ] && exit 0

# shellcheck source=checkpoint-state.sh
. "$(dirname "${BASH_SOURCE[0]}")/checkpoint-state.sh"
# shellcheck source=hook-log.sh
. "$(dirname "${BASH_SOURCE[0]}")/hook-log.sh"

# Only a turn claiming done is a checkpoint, and only its last text block counts:
# a done claim followed by more work is not one. As of 2026-10-07, replayed over
# the prior 7 days' 1085 blocks: 21 of the 384 the next reply disputed would
# still block, and 495 of the other 701.
DONE_RE='git commit|ready (to|for) (commit|merge|review|ship)|(all|work is|that.s|now) done|is complete'

# A pass label, not a fixed sentence. Demanding the literal "quality review"
# and "prose audit" blocked turns that had just run the checkpoint.
PROSE_RE='(prose|comments?)[[:space:]]*(audit|pass|review|sweep|—|-|:)'
QUALITY_RE='quality[[:space:]]*(review|pass|—|-|:)|bugs[[:space:]]+review'

# Enough to hold a turn; the whole transcript is megabytes.
TRANSCRIPT_TAIL=400

input=$(cat)

field() { printf '%s' "$input" | jq -r "$1" 2>/dev/null; }

# Only meaningful in a git repository.
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
[ -n "${root:-}" ] || exit 0

state_file=$(checkpoint_state_file "$(field '.session_id // empty')" "$root") || exit 0

# However this hook exits, the turn is over — so the next prompt takes a fresh
# baseline. Clearing it here is also what lets the snapshot hook tell a new turn
# from a message sent partway through one.
trap 'rm -f "$state_file"' EXIT

# One interruption per turn. Continuing from this hook always proceeds, so a
# refusal can never loop.
[ "$(field '.stop_hook_active // false')" = "true" ] && exit 0

# Only this turn's edits count. Without that, every question asked between an
# edit and a commit gets interrupted.
#
# A missing baseline means this turn's start was never seen. Blocking on dirt
# that predates the session is the false positive this guard exists to avoid.
[ -r "$state_file" ] || exit 0

moved=$(checkpoint_moved_paths "$state_file" "$root")
[ -n "$moved" ] || exit 0

# Bytes the last checkpoint reviewed need no second one, however the turn ended.
checkpoint_reviewed "$root" && exit 0

checkpoint_open "$root" "$(field '.session_id // empty')" && exit 0

checkpoint_opted_out "$(field '.session_id // empty')" "$root" "$moved" && exit 0

# Scoped to this turn: an earlier checkpoint says nothing about edits made since.
turn=""
transcript=$(field '.transcript_path // empty')
if [ -n "$transcript" ] && [ -r "$transcript" ]; then
  turn=$(tail -n "$TRANSCRIPT_TAIL" "$transcript" 2>/dev/null |
    jq -r 'if .type=="user" and (.isMeta != true) and ((.message.content|type)=="string")
           then "\u001eTURN " + .message.content
           elif .type=="assistant"
           then ([.message.content[]?
                  | if .type=="text" then "\u001eTEXT\n" + .text
                    elif .type=="tool_use" then "\u001eTOOL " + (.name // "") + " " + ((.input.skill // "")|tostring)
                    else empty end] | join("\n"))
           else empty end' 2>/dev/null |
    awk '/^\036TURN/ { buf = "" } { buf = buf $0 "\n" } END { printf "%s", buf }')
fi

final=$(printf '%s' "$turn" | awk '/^\036TEXT$/ { on = 1; buf = ""; next } /^\036/ { on = 0; next } on { buf = buf $0 "\n" } END { printf "%s", buf }')
printf '%s' "$final" | grep -qiE "$DONE_RE" || exit 0

# The only signal that survives a report written tersely.
if printf '%s' "$turn" | grep -qiE $'\036TOOL Skill (ele:)?checkpoint|command-name>/?(ele:)?checkpoint|^\036TURN */(ele:)?checkpoint'; then
  exit 0
fi

# A quality and a prose pass both have to appear: naming one and skipping the
# other is the habit this exists to catch. What the prompt asked for does not
# count — only what was said back.
said=$(printf '%s' "$turn" | awk '/^\036TEXT$/ { on = 1; next } /^\036/ { on = 0; next } on')
if printf '%s' "$said" | grep -qiE "$QUALITY_RE" &&
  printf '%s' "$said" | grep -qiE "$PROSE_RE"; then
  exit 0
fi

hook_log block checkpoint "$(basename "$root")"

optout="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/checkpoint-reviewed.sh"
jq -nc --arg changed "$(printf '%s\n' "$moved" | head -n 20 | paste -sd' ' -)" --arg root "$root" --arg optout "$optout" '{decision: "block", reason: (
  "Files changed this turn, the reply says the work is done, and no checkpoint was reported.\n\nChanged: " + $changed +
  "\n\nInvoke the `ele:checkpoint` skill.\n\nIf the work is not done — mid-task, waiting on the user, or an investigation — say it is not a checkpoint, what remains, and stop.\n\nIf the user said this work needs no checkpoint, exempt its paths for this session: `" + ($optout | @sh) + " optout " + ($root | @sh) + " \"<glob>\" ...`"
)}'
