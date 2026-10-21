#!/usr/bin/env bash
# What the last checkpoint reviewed, for the checkpoint skill to read and write.
#
#   checkpoint-reviewed.sh base [root]     the commit a branch review diffs against
#   checkpoint-reviewed.sh check [root]    exit 0 if the tree is what it reviewed
#   checkpoint-reviewed.sh close [root]    ends a run that stops without a record
#   checkpoint-reviewed.sh freeze root dir [--base commit] [--start] -- path...
#                                           diff for the reviewers, delta since the last record, fingerprint for the re-review check; opens the run; --start resets its clock
#   checkpoint-reviewed.sh hash [root]     fingerprint of HEAD and the uncommitted tree
#   checkpoint-reviewed.sh optout root "glob" ...
#                                           the user exempted these paths for this session
#   checkpoint-reviewed.sh record [root]   after the report, never before; closes the run
set -uo pipefail

# shellcheck source=checkpoint-state.sh
. "$(dirname "${BASH_SOURCE[0]}")/checkpoint-state.sh"

# Small mode: one reviewer. Docs carry no logic for a bugs reviewer to trace.
SMALL_DOC_LINES=100
SMALL_LINES=40

# Dynamic scope: reads $root and $against, appends to $numstat, sets $tracked; freeze's locals.
append_diff() {
  local out=$1 path
  local -a known=() untracked=()
  shift
  for path in "$@"; do
    if git -C "$root" cat-file -e "$against:$path" 2> /dev/null ||
      git -C "$root" ls-files --error-unmatch -- "$path" > /dev/null 2>&1; then
      known+=("$path")
    else
      untracked+=("$path")
    fi
  done
  # One call, so a rename given by both paths pairs instead of reading as a delete and an add.
  if [ ${#known[@]} -gt 0 ]; then
    git -C "$root" diff "$against" -M -- "${known[@]}" >> "$out"
    numstat+=$'\n'$(git -C "$root" diff "$against" -M --numstat -- "${known[@]}")
  fi
  for path in ${untracked[@]+"${untracked[@]}"}; do
    tracked=0
    (cd "$root" && git diff --no-index -- /dev/null "$path") >> "$out"
    numstat+=$'\n'$(cd "$root" && git diff --no-index --numstat -- /dev/null "$path")
  done
}

base() {
  local root=$1 upstream
  upstream=$(git -C "$root" rev-parse -q --verify '@{upstream}' 2> /dev/null) ||
    upstream=$(git -C "$root" rev-parse -q --verify refs/remotes/origin/HEAD 2> /dev/null) || {
    echo "base: $root has no upstream and no origin/HEAD; name the base" >&2
    return 1
  }
  git -C "$root" merge-base HEAD "$upstream" 2> /dev/null || {
    echo "base: HEAD shares no history with $upstream; name the base" >&2
    return 1
  }
}

# Paths arrive as arguments: a shell variable holding a list is one word in zsh,
# and a diff frozen from it is empty or partial with nothing saying so.
freeze() {
  local root=$1 out_dir=$2 against=HEAD diff delta staging hash marker name path numstat lines tracked docs=1 mode
  local baseline current before after field status reviewed=0 changed=0 start=0 summary
  local -a fresh=() renamed=()
  shift 2
  while :; do
    case ${1:-} in
      --base)
        [ $# -ge 2 ] || usage
        against=$2
        shift 2
        ;;
      --start)
        start=1
        shift
        ;;
      *) break ;;
    esac
  done
  [ "${1:-}" = "--" ] && shift
  [ $# -gt 0 ] && [ -d "$out_dir" ] || usage
  marker=$(checkpoint_reviewed_file "$root") || {
    echo "freeze: $root is not a git repository" >&2
    return 1
  }
  if ! git -C "$root" rev-parse -q --verify "$against^{commit}" > /dev/null; then
    echo "freeze: $against is not a commit in $root" >&2
    return 1
  fi
  for path in "$@"; do
    # In the base or the index: a staged rename or deletion has left the working tree.
    if git -C "$root" cat-file -e "$against:$path" 2> /dev/null ||
      git -C "$root" ls-files --error-unmatch -- "$path" > /dev/null 2>&1; then
      if git -C "$root" diff "$against" --quiet -- "$path" 2> /dev/null; then
        echo "freeze: $path has no change against $against" >&2
        return 1
      fi
    elif [ ! -f "$root/$path" ]; then
      echo "freeze: $path is neither tracked nor a file in $root" >&2
      return 1
    fi
    case $path in *.md | *.txt) ;; *) docs=0 ;; esac
  done
  name=$(basename "$(git -C "$root" rev-parse --show-toplevel)")
  diff="$out_dir/checkpoint-$name.diff"
  delta="$out_dir/checkpoint-$name.delta.diff"
  staging="$out_dir/checkpoint-$name.delta"
  hash="$out_dir/checkpoint-$name.hash"
  : > "$diff"
  rm -rf "$delta" "$staging"
  numstat="" tracked=1
  append_diff "$diff" "$@"
  if [ ! -s "$diff" ]; then
    echo "freeze: the diff of $# paths is empty" >&2
    return 1
  fi
  measure
  summary="froze $# paths, $lines lines"
  [ "$against" = HEAD ] || summary+=" against $against"
  # A branch review covers commits, which no record's copies describe.
  if [ "$against" = HEAD ]; then
    numstat="" tracked=1
    # Both sides of a rename go against HEAD, where one call pairs them.
    # -z: otherwise git quotes a non-ASCII, tab or newline path, and the rename never matches.
    while IFS= read -r -d '' status; do
      IFS= read -r -d '' field
      case $status in
        R*)
          renamed+=("$field")
          IFS= read -r -d '' field && renamed+=("$field")
          ;;
      esac
    done < <(git -C "$root" diff HEAD -M --name-status -z -- "$@")
    for path in "$@"; do
      if ! baseline=$(checkpoint_reviewed_bytes "$root" "$path"); then
        fresh+=("$path")
        continue
      fi
      reviewed=1
      if is_renamed "$path"; then
        fresh+=("$path")
        continue
      fi
      current=/dev/null
      [ -f "$root/$path" ] && current=$root/$path
      # cmp calls an empty file equal to /dev/null.
      if [ "$baseline" = "$current" ] ||
        { [ "$baseline" != /dev/null ] && [ "$current" != /dev/null ] && cmp -s "$baseline" "$current"; }; then
        continue
      fi
      changed=$((changed + 1))
      before=/dev/null after=/dev/null
      if [ "$baseline" != /dev/null ]; then
        mkdir -p "$staging/a/$(dirname "$path")" && cp "$baseline" "$staging/a/$path" && before=a/$path
      fi
      if [ "$current" != /dev/null ]; then
        mkdir -p "$staging/b/$(dirname "$path")" && cp "$current" "$staging/b/$path" && after=b/$path
      fi
      (cd "$staging" && git diff --no-index --no-prefix -- "$before" "$after") >> "$delta"
      numstat+=$'\n'$(cd "$staging" && git diff --no-index --numstat -- "$before" "$after")
    done
    if [ "$reviewed" = 1 ]; then
      touch "$delta"
      append_diff "$delta" ${fresh[@]+"${fresh[@]}"}
      changed=$((changed + ${#fresh[@]}))
      measure
      summary+="; $changed changed since the last checkpoint, $lines lines"
    fi
  fi
  checkpoint_write_state "$root" | shasum | cut -d' ' -f1 > "$hash"
  # A re-freeze inside the open run keeps its start unless --start, so duration
  # spans the whole run. Paths overwrite: an abandoned run inside the window
  # still reads open.
  if [ "$start" = 1 ] || ! checkpoint_open "$root" "${CLAUDE_CODE_SESSION_ID:-}"; then
    date +%s > "$marker.started"
  fi
  printf '%s\n' "$@" > "$marker.frozen"
  printf '%s\n' "${CLAUDE_CODE_SESSION_ID:-}" > "$marker.open"
  if [ "$reviewed" = 1 ]; then
    echo "$summary, $mode mode for this root alone: $diff, delta $delta"
  else
    echo "$summary, $mode mode for this root alone: $diff"
  fi
}

# Dynamic scope: reads $renamed, freeze's local.
is_renamed() {
  local field
  for field in ${renamed[@]+"${renamed[@]}"}; do
    [ "$field" = "$1" ] && return 0
  done
  return 1
}

# Sets $lines and $mode from $numstat, $tracked and $docs, all locals of freeze.
measure() {
  local added deleted
  lines=0
  while read -r added deleted _; do
    [ -n "$added" ] || continue
    if [ "$added" = "-" ]; then
      tracked=0
    else
      lines=$((lines + added + deleted))
    fi
  done <<< "$numstat"
  mode=full
  if [ "$tracked" = 1 ] && { [ "$lines" -le "$SMALL_LINES" ] || { [ "$docs" = 1 ] && [ "$lines" -le "$SMALL_DOC_LINES" ]; }; }; then
    mode=small
  fi
}

optout() {
  local root=$1 file glob
  shift
  [ $# -gt 0 ] || usage
  [ -n "${CLAUDE_CODE_SESSION_ID:-}" ] || {
    echo "optout: CLAUDE_CODE_SESSION_ID is unset" >&2
    return 1
  }
  for glob in "$@"; do
    if [[ $glob == /* ]]; then
      echo "optout: '$glob' is absolute; name paths relative to the repository root" >&2
      return 1
    fi
    if [[ /$glob/ == */./* || /$glob/ == */../* || /$glob/ == *//* ]]; then
      echo "optout: '$glob' has a \".\", \"..\" or empty path segment, which no repository path does; write paths as git prints them, e.g. deck/*" >&2
      return 1
    fi
    # shellcheck disable=SC2053 # the glob is the pattern
    if [[ q9x == $glob || Z_0 == $glob || q9x/q9x/q9x.q9x == $glob || Z_0/Z_0/Z_0.Z_0 == $glob ]]; then
      echo "optout: '$glob' is too broad; name the paths" >&2
      return 1
    fi
  done
  file=$(checkpoint_optout_file "$CLAUDE_CODE_SESSION_ID" "$root") || return 1
  printf '%s\n' "$@" >> "$file" && echo "opted out for this session: $*"
}

usage() {
  echo "usage: checkpoint-reviewed.sh base|check|close|hash|record [root]" >&2
  echo "       checkpoint-reviewed.sh freeze <root> <dir> [--base <commit>] [--start] -- <path>..." >&2
  echo "       checkpoint-reviewed.sh optout <root> \"<glob>\" ..." >&2
  exit 2
}

command=${1:-}
case $command in
  freeze | optout) [ $# -ge 3 ] || usage; root=$2 ;;
  *) root=${2:-$(git rev-parse --show-toplevel 2> /dev/null)} || exit 2 ;;
esac
[ -n "$root" ] || exit 2

case $command in
  base) base "$root" ;;
  check)
    if checkpoint_reviewed "$root"; then
      echo "covered: every changed path in $root is what the last checkpoint reviewed"
    else
      echo "not covered: $root has changed since the last checkpoint, or none ran"
      exit 1
    fi
    ;;
  close) checkpoint_close "$root" && echo "closed: $root" ;;
  freeze) shift 2; freeze "$root" "$@" ;;
  hash) checkpoint_write_state "$root" | shasum | cut -d' ' -f1 ;;
  optout) shift 2; optout "$root" "$@" ;;
  record) checkpoint_record_reviewed "$root" && echo "recorded: $root" ;;
  *) usage ;;
esac
