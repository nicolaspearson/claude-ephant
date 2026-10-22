#!/usr/bin/env bash
#
# Regression suite for checkpoint-review.sh. Run after any edit to it:
#   bash plugin/premigration/checkpoint-review.test.sh
#
# Builds a throwaway git repository under TMPDIR — the hook reads a real
# working tree and cannot be tested without one. Nothing outside TMPDIR.
#
# The PASS cases under "report phrasings" are transcribed from reports the
# previous detector blocked. It required the literal "quality review" and the
# literal "prose audit"; every report below says neither and ran every pass.

set -uo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

REVIEW=$HERE/checkpoint-review.sh
RECORD=$HERE/checkpoint-reviewed.sh
SNAPSHOT=$HERE/checkpoint-snapshot.sh
SESSION=checkpoint-test-session

# Decisions provoked here are fixtures. They must not reach the real log,
# which is the measurement instrument.
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
export HOOK_LOG=$WORK/hooks.log
export TMPDIR="$WORK/state"
export CHECKPOINT_REVIEWED_DIR="$WORK/reviewed"
mkdir -p "$TMPDIR"
# Inherited from a Claude Code session, it holds runs open across cases.
unset CLAUDE_CODE_SESSION_ID

# The hooks read the gate through ele; an installed one may predate config.
export ELE_HOME=$WORK/ele
mkdir -p "$ELE_HOME"
(cd "$HERE/../.." && go build -o "$WORK/bin/ele" ./cmd/ele) || exit 1
export PATH=$WORK/bin:$PATH
gate_off() { CLAUDECODE='' ele config set checkpoint_gate false > /dev/null; }
gate_on() { rm -f "$ELE_HOME/config.json"; }

REPO=$WORK/repo
git init -q "$REPO"
git -C "$REPO" config user.email test@example.com
git -C "$REPO" config user.name Test
printf 'package main\n' > "$REPO/main.go"
git -C "$REPO" add -A
git -C "$REPO" commit -qm initial

DONE='Ready to commit:
```sh
git commit -m "x"
```'
FAILED=0
LAST_OUT=

snapshot() {
  printf '{"session_id":"%s"}' "$SESSION" | (cd "$REPO" && bash "$SNAPSHOT") > /dev/null
}

# A turn is a user prompt followed by what the assistant said or called.
transcript() {
  local file=$WORK/transcript.jsonl
  : > "$file"
  local arg
  for arg in "$@"; do
    case $arg in
      PROMPT:*)
        jq -nc --arg t "${arg#PROMPT:}" \
          '{type:"user",isMeta:false,message:{role:"user",content:$t}}' >> "$file"
        ;;
      SKILL:*)
        jq -nc --arg s "${arg#SKILL:}" \
          '{type:"assistant",message:{role:"assistant",content:[{type:"tool_use",name:"Skill",input:{skill:$s}}]}}' >> "$file"
        ;;
      *)
        jq -nc --arg t "$arg" \
          '{type:"assistant",message:{role:"assistant",content:[{type:"text",text:$t}]}}' >> "$file"
        ;;
    esac
  done
  printf '%s' "$file"
}

probe() { # want label [transcript] [stop_hook_active] [cwd]
  local want=$1 label=$2 tp=${3:-} stop=${4:-false} cwd=${5:-$REPO} got
  LAST_OUT=$(printf '{"session_id":"%s","transcript_path":"%s","stop_hook_active":%s}' \
    "$SESSION" "$tp" "$stop" | (cd "$cwd" && bash "$REVIEW") 2>&1)
  case "$LAST_OUT" in
    *'"decision":"block"'*) got=BLOCK ;;
    *) got=PASS ;;
  esac
  if [ "$got" = "$want" ]; then printf '  ok   %-5s %s\n' "$got" "$label"
  else
    printf '  FAIL  want=%-5s got=%-5s %s\n' "$want" "$got" "$label"
    FAILED=1
  fi
}

names() { # the block has to say what moved
  if printf '%s' "$LAST_OUT" | grep -qF "$1"; then printf '  ok   NAMES %s\n' "$1"
  else
    printf '  FAIL  reason does not name %s\n' "$1"
    FAILED=1
  fi
}

dirty() { printf 'package main // %s\n' "$RANDOM$RANDOM" > "$REPO/main.go"; }
clean() { git -C "$REPO" reset -q && git -C "$REPO" checkout -q -- . && git -C "$REPO" clean -qfd; }

echo "### turns that are not checkpoints ###"
clean
snapshot
probe PASS "clean tree"
clean
snapshot
dirty
probe PASS "stop hook already fired" "" true
clean
dirty
rm -f "$TMPDIR"/claude-checkpoint/* 2> /dev/null
probe PASS "no baseline: dirt predates the session"
clean
dirty
snapshot
probe PASS "dirty before the turn, unchanged during it"
probe PASS "outside a repository" "" false "$WORK"

echo "### a turn that changed files ###"
clean
snapshot
dirty
probe PASS "tracked edit, work not claimed done"
clean
snapshot
dirty
probe BLOCK "tracked edit, claimed done, nothing reported" \
  "$(transcript PROMPT:go "$DONE")"
clean
snapshot
dirty
gate_off
probe PASS "gate option false" \
  "$(transcript PROMPT:go "$DONE")"
gate_on
clean
snapshot
dirty
printf '{"checkpoint_gate": fals' > "$ELE_HOME/config.json"
probe BLOCK "config.json that does not parse keeps the gate on" \
  "$(transcript PROMPT:go "$DONE")"
gate_on
clean
rm -f "$TMPDIR"/claude-checkpoint/* 2> /dev/null
gate_off
snapshot
gate_on
dirty
probe PASS "gate option off: snapshot writes no baseline" \
  "$(transcript PROMPT:go "$DONE")"
clean
snapshot
dirty
probe BLOCK "tracked edit, quality named but no prose pass" \
  "$(transcript PROMPT:go '# Quality review of the diff. Ready to commit.')"
clean
snapshot
dirty
probe BLOCK "tracked edit, prose named but no quality pass" \
  "$(transcript PROMPT:go '**Prose — ran now, two trims.** Ready to commit.')"

echo "### report phrasings the old detector blocked ###"
clean
snapshot
dirty
probe PASS "quality review + comment audit" \
  "$(transcript PROMPT:go '**Quality review — partial credit.** **Comment audit — stale.** Ready to commit.')"
clean
snapshot
dirty
probe PASS "em-dash pass labels" \
  "$(transcript PROMPT:go '**Quality — ran now.** **Prose — ran now, two trims.** Ready to commit.')"
clean
snapshot
dirty
probe PASS "this turn was the pass" \
  "$(transcript PROMPT:go '**Quality — this turn was the pass.** **Prose — the important one here.** Ready to commit.')"
clean
snapshot
dirty
probe PASS "the phrases the old detector wanted" \
  "$(transcript PROMPT:go 'A quality review and a separate comment and prose audit. Ready to commit.')"

clean
snapshot
dirty
probe BLOCK "the prompt asked for the passes, the reply ran none" \
  "$(transcript 'PROMPT:do a quality review and a prose audit' 'Edited the file. Ready to commit.')"

clean
snapshot
dirty
probe BLOCK "a multi-line prompt asked for the passes, the reply ran none" \
  "$(transcript 'PROMPT:please
do a quality review and a prose audit' 'Edited the file. Ready to commit.')"

echo "### the skill itself ###"
clean
snapshot
dirty
probe PASS "checkpoint skill invoked this turn" \
  "$(transcript PROMPT:go SKILL:checkpoint 'Done. Ready to commit.')"
clean
snapshot
dirty
probe PASS "ele:checkpoint skill invoked this turn" \
  "$(transcript PROMPT:go SKILL:ele:checkpoint 'Done. Ready to commit.')"
clean
snapshot
dirty
probe PASS "user typed /ele:checkpoint as a prompt" \
  "$(transcript 'PROMPT:/ele:checkpoint' "$DONE")"
clean
snapshot
dirty
probe BLOCK "checkpoint skill invoked in an earlier turn" \
  "$(transcript PROMPT:earlier SKILL:checkpoint '**Quality — ran.** **Prose — ran.**' PROMPT:now 'Changed one more thing. Ready to commit.')"

echo "### turns that hand back ###"
clean
snapshot
dirty
probe PASS "edit, ends on a question" \
  "$(transcript PROMPT:go 'Added the seed. Want me to add the second variant too?')"
clean
snapshot
dirty
probe PASS "claimed done early, ended mid-task" \
  "$(transcript PROMPT:go 'Ready to commit.' SKILL:none 'Found one more case; fixing it next turn.')"

echo "### the user's own git actions ###"
clean
dirty
snapshot
git -C "$REPO" add main.go
probe PASS "user staged a dirty file" "$(transcript PROMPT:go "$DONE")"
clean
dirty
snapshot
git -C "$REPO" commit -qam wip
probe PASS "user committed a dirty file" "$(transcript PROMPT:go "$DONE")"
clean
printf 'new\n' > "$REPO/new.go"
snapshot
git -C "$REPO" add new.go
probe PASS "user staged an untracked file" "$(transcript PROMPT:go "$DONE")"
git -C "$REPO" rm -qf --cached new.go
clean
dirty
snapshot
git -C "$REPO" checkout -q -- main.go
probe BLOCK "agent reverted a dirty file" "$(transcript PROMPT:go "$DONE")"
clean
printf 'one\n' > "$REPO/notes.txt"
snapshot
printf 'two\n' > "$REPO/notes.txt"
probe BLOCK "untracked file content changed" "$(transcript PROMPT:go "$DONE")"

echo "### the checkpoint's own report labels ###"
clean
snapshot
dirty
probe PASS "bugs and prose reviewer labels" \
  "$(transcript PROMPT:go '**Bugs reviewer:** two fixed. **Prose reviewer:** one comment cut. Ready to commit.')"
clean
snapshot
dirty
probe BLOCK "bug-fix prose that only looks like labels" \
  "$(transcript PROMPT:go 'Fixed the off-by-one bug — loop stops at len. Trimmed a stale comment — it named the old function. Ready to commit.')"
clean
snapshot
dirty
probe PASS "user typed /checkpoint" \
  "$(transcript 'PROMPT:<command-message>checkpoint</command-message>
<command-name>/checkpoint</command-name>' "$DONE")"
clean
snapshot
dirty
probe PASS "user typed /ele:checkpoint" \
  "$(transcript 'PROMPT:<command-message>ele:checkpoint</command-message>
<command-name>/ele:checkpoint</command-name>' "$DONE")"
clean
snapshot
dirty
probe BLOCK "diff hunk before the done claim" \
  "$(transcript PROMPT:go 'Here:
```diff
@@ -1 +1 @@
-old
+new
```
Ready to commit.')"

echo "### the block says what moved ###"
clean
snapshot
dirty
probe BLOCK "tracked change" "$(transcript PROMPT:go "$DONE")"
names "main.go"
clean
snapshot
printf 'coverage\n' > "$REPO/coverage.out"
probe BLOCK "untracked file appeared" "$(transcript PROMPT:go "$DONE")"
names "coverage.out"
names "Invoke the \`ele:checkpoint\` skill."
names "'$HERE/checkpoint-reviewed.sh' optout '$(git -C "$REPO" rev-parse --show-toplevel)' \\\"<glob>\\\" ..."

echo "### a state the last checkpoint reviewed ###"
hashed() { bash "$RECORD" hash "$REPO"; }
reviewed() { bash "$RECORD" "$1" "$REPO" > /dev/null; }
recorded() { # path... — one run of the skill: freeze, then record
  CLAUDE_CODE_SESSION_ID=$SESSION bash "$RECORD" freeze "$REPO" "$WORK" -- "$@" > /dev/null &&
    CLAUDE_CODE_SESSION_ID=$SESSION bash "$RECORD" record "$REPO" > /dev/null
}
same() { # want label a b
  local got=DIFF
  [ "$3" = "$4" ] && got=SAME
  if [ "$got" = "$1" ]; then printf '  ok   %-5s %s\n' "$got" "$2"
  else
    printf '  FAIL  want=%-5s got=%-5s %s\n' "$1" "$got" "$2"
    FAILED=1
  fi
}
covered() { # want label
  local got=NO
  reviewed check && got=YES
  if [ "$got" = "$1" ]; then printf '  ok   %-5s %s\n' "$got" "$2"
  else
    printf '  FAIL  want=%-5s got=%-5s %s\n' "$1" "$got" "$2"
    FAILED=1
  fi
}
clean
snapshot
dirty
recorded main.go
probe PASS "reviewed state, skill call out of view" "$(transcript PROMPT:go "$DONE")"
covered YES "check after record"
dirty
covered NO "edited after the record"
snapshot
dirty
probe BLOCK "edited after the record, claimed done" "$(transcript PROMPT:go "$DONE")"
clean
dirty
printf 'other\n' > "$REPO/other.go"
recorded main.go other.go
git -C "$REPO" add other.go
git -C "$REPO" commit -qm partial
covered YES "user committed part of the reviewed work"
clean
dirty
before=$(hashed)
git -C "$REPO" add main.go
same SAME "hash survives staging" "$before" "$(hashed)"
dirty
same DIFF "hash moves on an edit" "$before" "$(hashed)"
clean
covered NO "clean tree"
rm -rf "$CHECKPOINT_REVIEWED_DIR"
dirty
covered NO "no checkpoint ever recorded"

echo "### freeze ###"
froze() { # want label [--base commit] [path...]
  local want=$1 label=$2 got=OK
  local -a options=()
  shift 2
  if [ "${1:-}" = --base ]; then
    options=(--base "$2")
    shift 2
  fi
  LAST_OUT=$(bash "$RECORD" freeze "$REPO" "$WORK" ${options[@]+"${options[@]}"} -- "$@" 2>&1) || got=REFUSED
  if [ "$got" = "$want" ]; then printf '  ok   %-7s %s\n' "$got" "$label"
  else
    printf '  FAIL  want=%-7s got=%-7s %s: %s\n' "$want" "$got" "$label" "$LAST_OUT"
    FAILED=1
  fi
}
says() { # the output has to carry this
  if printf '%s' "$LAST_OUT" | grep -qF "$1"; then printf '  ok   SAYS  %s\n' "$1"
  else
    printf '  FAIL  output does not say %s: %s\n' "$1" "$LAST_OUT"
    FAILED=1
  fi
}
clean
dirty
printf 'new\n' > "$REPO/new file.go"
froze OK "tracked and untracked paths, one with a space" main.go "new file.go"
says "froze 2 paths, 3 lines, full mode"
if grep -qF '+++ b/new file.go' "$WORK/checkpoint-repo.diff" && grep -qF '+++ b/main.go' "$WORK/checkpoint-repo.diff"; then
  echo "  ok   DIFF  holds both paths"
else
  echo "  FAIL  diff lacks a path"
  FAILED=1
fi
same SAME "hash file is the fingerprint" "$(cat "$WORK/checkpoint-repo.hash")" "$(hashed)"
froze OK "one tracked edit" main.go
says "small mode"
froze REFUSED "unchanged path" other.go
froze REFUSED "unchanged path beside a changed one" main.go other.go
froze REFUSED "missing path" gone.go
froze REFUSED "no paths"
clean
git -C "$REPO" mv other.go renamed.go
froze OK "staged rename, both paths" other.go renamed.go
if grep -qF 'rename from other.go' "$WORK/checkpoint-repo.diff"; then
  echo "  ok   DIFF  pairs the rename"
else
  echo "  FAIL  diff does not pair the rename"
  FAILED=1
fi
clean
git -C "$REPO" rm -q other.go
froze OK "staged deletion" other.go
says "1 lines"
clean
mkdir -p "$WORK/plain"
printf 'x\n' > "$WORK/plain/f.md"
if (cd "$WORK" && bash "$RECORD" freeze "$WORK/plain" "$WORK" -- f.md > /dev/null 2>&1); then
  echo "  FAIL  froze a directory outside git"
  FAILED=1
elif [ -e "$WORK/.open" ]; then
  echo "  FAIL  refusal left a marker in the working directory"
  FAILED=1
else
  echo "  ok   REFUSED a directory outside git, nothing written"
fi

lines() { # path count: a committed empty file given count new lines
  : > "$REPO/$1"
  git -C "$REPO" add "$1"
  git -C "$REPO" commit -qm "$1" > /dev/null 2>&1
  seq "$2" > "$REPO/$1"
}
clean
lines code.go 40
froze OK "40 lines of code" code.go
says "small mode"
clean
lines code.go 41
froze OK "41 lines of code" code.go
says "full mode"
clean
lines notes.md 100
froze OK "100 lines of Markdown" notes.md
says "small mode"
clean
lines notes.md 101
froze OK "101 lines of Markdown" notes.md
says "full mode"
clean
lines notes.txt 100
froze OK "100 lines of text" notes.txt
says "small mode"
clean
lines notes.md 39
dirty
froze OK "docs and code past the code limit" notes.md main.go
says "full mode"
clean
printf '\0\1' > "$REPO/blob.bin"
git -C "$REPO" add blob.bin
git -C "$REPO" commit -qm blob
printf '\0\2' > "$REPO/blob.bin"
froze OK "one tracked binary" blob.bin
says "full mode"

echo "### delta since the last record ###"
export CLAUDE_CODE_SESSION_ID=$SESSION
DELTA=$WORK/checkpoint-repo.delta.diff
holds() { # want label file string
  local got=NO
  grep -qF -- "$4" "$3" 2> /dev/null && got=YES
  if [ "$got" = "$1" ]; then printf '  ok   %-5s %s\n' "$got" "$2"
  else
    printf '  FAIL  want=%-5s got=%-5s %s\n' "$1" "$got" "$2"
    FAILED=1
  fi
}
rm -rf "$CHECKPOINT_REVIEWED_DIR"
clean
printf 'package main\n// one\n' > "$REPO/main.go"
froze OK "no record yet" main.go
[ -e "$DELTA" ] && { echo "  FAIL  a delta with no record"; FAILED=1; } || echo "  ok   no delta without a record"
reviewed record
printf 'package main\n// one\n// two\n' > "$REPO/main.go"
froze OK "edited after the record" main.go
says "; 1 changed since the last checkpoint, 1 lines, small mode"
holds YES "delta holds the new line" "$DELTA" "+// two"
holds NO "delta leaves out the reviewed line" "$DELTA" "+// one"
holds YES "delta names the path as git does" "$DELTA" "+++ b/main.go"
holds YES "the full diff still holds the reviewed line" "$WORK/checkpoint-repo.diff" "+// one"
printf 'package main\n// one\n' > "$REPO/main.go"
printf 'new\n' > "$REPO/new.go"
froze OK "reviewed path unchanged, an unreviewed one added" main.go new.go
says "1 changed since the last checkpoint, 1 lines, full mode"
holds NO "delta leaves out the unchanged path" "$DELTA" "main.go"
holds YES "delta diffs the unreviewed path against HEAD" "$DELTA" "+++ b/new.go"
froze OK "nothing changed since the record" main.go
says "0 changed since the last checkpoint, 0 lines"
[ -s "$DELTA" ] && { echo "  FAIL  an empty delta has content"; FAILED=1; } || echo "  ok   an empty delta is an empty file"
rm "$REPO/main.go"
froze OK "reviewed path deleted since" main.go
holds YES "delta deletes from the reviewed copy" "$DELTA" "+++ /dev/null"
holds YES "delta removes the reviewed line" "$DELTA" "-// one"
printf 'package main\n// one\n// two\n' > "$REPO/main.go"
printf 'tampered\n' > "$(. "$HERE/checkpoint-state.sh" && checkpoint_reviewed_file "$REPO").d/main.go"
froze OK "a copy that no longer matches the record" main.go
[ -e "$DELTA" ] && { echo "  FAIL  a delta from a copy the record does not vouch for"; FAILED=1; } || echo "  ok   a copy that no longer matches is no record"
clean
printf 'package main\n// gone\n' > "$REPO/main.go"
git -C "$REPO" rm -q --cached other.go
rm "$REPO/other.go"
froze OK "a deletion, for the record" other.go
reviewed record
git -C "$REPO" checkout -q HEAD -- other.go
printf 'back\n' > "$REPO/other.go"
froze OK "a path reviewed as deleted, back again" other.go
holds YES "delta adds it from nothing" "$DELTA" "+back"
holds NO "delta does not diff it against HEAD" "$DELTA" "-other"
: > "$REPO/other.go"
froze OK "a path reviewed as deleted, back empty" other.go
says "1 changed since the last checkpoint, 0 lines"
clean
reviewed close
printf 'package main\n// one\n' > "$REPO/main.go"
printf 'off the list\n' > "$REPO/other.go"
froze OK "one path frozen, another dirty" main.go
reviewed record
[ -e "$(. "$HERE/checkpoint-state.sh" && checkpoint_reviewed_file "$REPO").d/other.go" ] && { echo "  FAIL  record copied a path no run froze"; FAILED=1; } || echo "  ok   record copies only what the run froze"
froze OK "the path dirty but never frozen" other.go
[ -e "$DELTA" ] && { echo "  FAIL  a delta from bytes no run froze"; FAILED=1; } || echo "  ok   a path off the frozen list is no record"
clean
reviewed close
printf 'package main\n// one\n' > "$REPO/main.go"
rm "$REPO/other.go"
froze OK "one path frozen, another deleted" main.go
reviewed record
froze OK "the deletion never frozen" other.go
[ -e "$DELTA" ] && { echo "  FAIL  a delta from a deletion no run froze"; FAILED=1; } || echo "  ok   a deletion off the frozen list is no record"
clean
reviewed close
printf 'package main\n// one\n' > "$REPO/main.go"
printf 'refrozen\n' > "$REPO/other.go"
bash "$RECORD" freeze "$REPO" "$WORK" -- main.go > /dev/null
bash "$RECORD" freeze "$REPO" "$WORK" -- other.go > /dev/null
reviewed record
printf 'package main\n// one\n// two\n' > "$REPO/main.go"
froze OK "a path only an earlier freeze in the window named" main.go
[ -e "$DELTA" ] && { echo "  FAIL  record vouched for a path the last freeze left off"; FAILED=1; } || echo "  ok   a second freeze replaces the first's paths"
clean
printf 'a\nb\nc\nd\ne\n' > "$REPO/rename.go"
git -C "$REPO" add rename.go
git -C "$REPO" commit -qm rename
printf 'a\nb\nc\nd\ne\nf\n' > "$REPO/rename.go"
froze OK "a path to rename, for the record" rename.go
reviewed record
git -C "$REPO" mv rename.go moved.go
printf 'a\nb\nc\nd\ne\nf\ng\n' > "$REPO/moved.go"
froze OK "the reviewed path renamed and edited" rename.go moved.go
holds YES "delta pairs the rename" "$DELTA" "rename from rename.go"
says "2 changed since the last checkpoint, 2 lines"
clean
git -C "$REPO" config core.quotePath true
printf 'a\nb\nc\nd\ne\n' > "$REPO/ünï.go"
git -C "$REPO" add ünï.go
git -C "$REPO" commit -qm quoted
printf 'a\nb\nc\nd\ne\nf\n' > "$REPO/ünï.go"
froze OK "a path git quotes, for the record" ünï.go
reviewed record
git -C "$REPO" mv ünï.go möved.go
printf 'a\nb\nc\nd\ne\nf\ng\n' > "$REPO/möved.go"
froze OK "the quoted path renamed and edited" ünï.go möved.go
holds YES "delta pairs the quoted rename" "$DELTA" "rename from"
says "2 changed since the last checkpoint, 2 lines"
clean
printf 'package main\n// abandoned\n' > "$REPO/main.go"
froze OK "a run abandoned before its record" main.go
touch -t 202001010000 "$CHECKPOINT_REVIEWED_DIR"/*.open
if LAST_OUT=$(bash "$RECORD" record "$REPO" 2>&1); then
  echo "  FAIL  record vouched for an abandoned run"
  FAILED=1
else
  echo "  ok   REFUSED a record of an abandoned run"
fi
says "freeze again, then record"
covered NO "the abandoned run's tree"
CLAUDE_CODE_SESSION_ID=another-session bash "$RECORD" freeze "$REPO" "$WORK" -- main.go > /dev/null
if LAST_OUT=$(bash "$RECORD" record "$REPO" 2>&1); then
  echo "  FAIL  record vouched for another session's run"
  FAILED=1
else
  echo "  ok   REFUSED a record of another session's run"
fi
covered NO "another session's tree"
same SAME "the refused record leaves the other run open" another-session "$(cat "$(. "$HERE/checkpoint-state.sh" && checkpoint_reviewed_file "$REPO").open")"

echo "### branch ###"
rm -rf "$CHECKPOINT_REVIEWED_DIR"
clean
branch=$(git -C "$REPO" branch --show-current)
if bash "$RECORD" base "$REPO" > /dev/null 2>&1; then
  echo "  FAIL  a base with no upstream and no origin"
  FAILED=1
else
  echo "  ok   REFUSED a base with no upstream and no origin"
fi
pushed=$(git -C "$REPO" rev-parse HEAD)
git -C "$REPO" update-ref refs/remotes/origin/main "$pushed"
git -C "$REPO" symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/main
printf 'package main\n' > "$REPO/branch.go"
git -C "$REPO" add branch.go
git -C "$REPO" commit -qm branch
same SAME "base falls back to origin's default branch" "$pushed" "$(bash "$RECORD" base "$REPO")"
git -C "$REPO" commit -q --allow-empty -m ahead
git -C "$REPO" branch -q upstream
git -C "$REPO" config "branch.$branch.remote" .
git -C "$REPO" config "branch.$branch.merge" refs/heads/upstream
same SAME "base prefers the upstream" "$(git -C "$REPO" rev-parse HEAD)" "$(bash "$RECORD" base "$REPO")"
git -C "$REPO" config --unset "branch.$branch.remote"
git -C "$REPO" config --unset "branch.$branch.merge"
orphan=$(git -C "$REPO" commit-tree -m orphan "$(git -C "$REPO" hash-object -w -t tree /dev/null)")
git -C "$REPO" update-ref refs/remotes/origin/main "$orphan"
if LAST_OUT=$(bash "$RECORD" base "$REPO" 2>&1); then
  echo "  FAIL  a base from unrelated history: $LAST_OUT"
  FAILED=1
else
  echo "  ok   REFUSED a base from unrelated history"
fi
says "shares no history"
git -C "$REPO" update-ref refs/remotes/origin/main "$pushed"
froze OK "a committed path against the base" --base "$pushed" branch.go
says "froze 1 paths, 1 lines against $pushed, small mode"
holds YES "the diff holds the commit" "$WORK/checkpoint-repo.diff" "+++ b/branch.go"
froze REFUSED "a path the branch did not change" --base "$pushed" main.go
froze REFUSED "a base that is not a commit" --base nowhere branch.go
printf 'package main\n// again\n' > "$REPO/branch.go"
reviewed record
printf 'package main\n// again and again\n' > "$REPO/branch.go"
froze OK "a branch review with a record" --base "$pushed" branch.go
[ -e "$DELTA" ] && { echo "  FAIL  a branch review wrote a delta"; FAILED=1; } || echo "  ok   a branch review writes no delta"
rm -rf "$CHECKPOINT_REVIEWED_DIR"

echo "### a checkpoint still running ###"
export CLAUDE_CODE_SESSION_ID=$SESSION
clean
snapshot
dirty
froze OK "freeze opens the run" main.go
probe PASS "frozen, not recorded, claimed done" "$(transcript PROMPT:go "$DONE")"
started="$(. "$HERE/checkpoint-state.sh" && checkpoint_reviewed_file "$REPO").started"
echo 1 > "$started"
froze OK "freeze for the re-review" main.go
[ "$(cat "$started")" = 1 ] && echo "  ok   the re-review's freeze keeps the run's start" || { echo "  FAIL  the re-review's freeze reset the start"; FAILED=1; }
bash "$RECORD" freeze "$REPO" "$WORK" --base HEAD --start -- main.go > /dev/null
[ "$(cat "$started")" -gt 1 ] && echo "  ok   --start opens a new run inside the window" || { echo "  FAIL  a new run inside the window kept the abandoned start"; FAILED=1; }
echo 1 > "$started"
touch -t 202001010000 "$CHECKPOINT_REVIEWED_DIR"/*.open
snapshot
dirty
probe BLOCK "open marker abandoned hours ago" "$(transcript PROMPT:go "$DONE")"
froze OK "freeze again" main.go
[ "$(cat "$started")" -gt 1 ] && echo "  ok   a freeze after an abandoned run starts a new one" || { echo "  FAIL  a new run kept the abandoned start"; FAILED=1; }
reviewed record
snapshot
dirty
probe BLOCK "record closed the run" "$(transcript PROMPT:go "$DONE")"
froze OK "freeze, then stop without a record" main.go
reviewed close
snapshot
dirty
probe BLOCK "close ended the run" "$(transcript PROMPT:go "$DONE")"
CLAUDE_CODE_SESSION_ID=another-session bash "$RECORD" freeze "$REPO" "$WORK" -- main.go > /dev/null
snapshot
dirty
probe BLOCK "another session's run is open" "$(transcript PROMPT:go "$DONE")"
reviewed close

echo "### opted out ###"
clean
mkdir -p "$REPO/deck"
snapshot
printf 'slide\n' > "$REPO/deck/one.md"
CLAUDE_CODE_SESSION_ID=$SESSION bash "$RECORD" optout "$REPO" 'deck/*' > /dev/null
probe PASS "every moved path exempted" "$(transcript PROMPT:go "$DONE")"
snapshot
printf 'slide 2\n' > "$REPO/deck/one.md"
dirty
probe BLOCK "a moved path outside the exemption" "$(transcript PROMPT:go "$DONE")"
for glob in '*' '**' '**/*' '*.*' '*/*' '?*' '[!q]*'; do
  if bash "$RECORD" optout "$REPO" "$glob" > /dev/null 2>&1; then
    echo "  FAIL  optout accepted $glob, which matches every path"
    FAILED=1
  else
    echo "  ok   REFUSED optout of $glob"
  fi
done
if bash "$RECORD" optout "$REPO" "$REPO/deck/*" > /dev/null 2>&1; then
  echo "  FAIL  optout accepted an absolute glob, which matches no path"
  FAILED=1
else
  echo "  ok   REFUSED absolute optout glob"
fi
for glob in '' '../deck/*' './deck/*' 'deck/' 'deck/../x' 'deck/./x' 'deck//x'; do
  if bash "$RECORD" optout "$REPO" "$glob" > /dev/null 2>&1; then
    echo "  FAIL  optout accepted '$glob', which matches no path"
    FAILED=1
  else
    echo "  ok   REFUSED optout of '$glob'"
  fi
done
for glob in '*.md' '.github/*'; do
  if bash "$RECORD" optout "$REPO" "$glob" > /dev/null 2>&1; then
    echo "  ok   ACCEPT optout of '$glob'"
  else
    echo "  FAIL  optout refused '$glob'"
    FAILED=1
  fi
done
if (unset CLAUDE_CODE_SESSION_ID; bash "$RECORD" optout "$REPO" 'deck/*') > /dev/null 2>&1; then
  echo "  FAIL  optout without a session"
  FAILED=1
else
  echo "  ok   REFUSED optout without a session"
fi

echo
[ "$FAILED" = 0 ] && echo "ALL PASS" || echo "SOME FAILED"
exit "$FAILED"
