#!/usr/bin/env bash
# Working-tree fingerprint shared by the checkpoint hooks and the checkpoint
# skill's record. Sourced, never executed.
#
# One line per dirty path, hashed by on-disk content rather than index or diff,
# so the user staging or committing between turns moves nothing.

# Every path is hashed, so the count bounds the hook's runtime.
CHECKPOINT_PATH_CAP=200

checkpoint_close() {
  local file
  file=$(checkpoint_reviewed_file "$1") || return 1
  rm -f "$file.open" "$file.frozen"
}

checkpoint_dirty_paths() {
  local root=$1
  {
    git -C "$root" diff HEAD --name-only --no-renames -z 2>/dev/null
    git -C "$root" ls-files --others --exclude-standard -z 2>/dev/null
  } | tr '\0' '\n' | sort -u | head -n "$CHECKPOINT_PATH_CAP"
}

# Paths whose content moved between the baseline and now. A path that went
# clean counts only while HEAD stayed put: clean under a new HEAD is a commit.
checkpoint_moved_paths() {
  local baseline=$1 root=$2
  awk -F'\t' '
    FNR == 1 { head[FILENAME == ARGV[1] ? "was" : "now"] = $0; next }
    FILENAME == ARGV[1] { was[$2] = $1; next }
    { now[$2] = $1; if (!($2 in was) || was[$2] != $1) print $2 }
    END { if (head["was"] == head["now"]) for (p in was) if (!(p in now)) print p }
  ' "$baseline" <(checkpoint_write_state "$root") | sort -u
}

# Frozen by this session and not yet recorded or closed. Background reviewers end
# turns mid-run, and those turns fall outside the Stop hook's view of the skill
# call. A marker older than CHECKPOINT_OPEN_MINUTES is an abandoned run.
checkpoint_open() {
  local root=$1 session=$2 file
  [ -n "$session" ] || return 1
  file=$(checkpoint_reviewed_file "$root") || return 1
  [ -n "$(find "$file.open" -mmin "-${CHECKPOINT_OPEN_MINUTES:-120}" 2>/dev/null)" ] &&
    [ "$(cat "$file.open")" = "$session" ]
}

# Every moved path matches a glob the user exempted this session.
checkpoint_opted_out() {
  local session=$1 root=$2 moved=$3 file path glob matched
  file=$(checkpoint_optout_file "$session" "$root") || return 1
  [ -r "$file" ] || return 1
  while IFS= read -r path; do
    [ -n "$path" ] || continue
    matched=1
    while IFS= read -r glob; do
      # shellcheck disable=SC2053 # the glob is the pattern
      [[ $path == $glob ]] && { matched=0; break; }
    done < "$file"
    [ "$matched" = 0 ] || return 1
  done <<< "$moved"
}

checkpoint_optout_file() {
  local root file
  root=$(git -C "$2" rev-parse --show-toplevel 2>/dev/null) || return 1
  file=$(checkpoint_state_file "$1" "$root") || return 1
  printf '%s.optout' "$file"
}

# Copies only what the run froze: a dirty path off the change list is in the
# record for `check`, but was never reviewed, so no delta may skip it.
checkpoint_record_reviewed() {
  local root=$1 file hash path
  file=$(checkpoint_reviewed_file "$root") || return 1
  # No live run of this session: nothing frozen here for this record to vouch for.
  checkpoint_open "$root" "${CLAUDE_CODE_SESSION_ID:-}" || {
    echo "record: no open run of this session in $root (none, expired, another session's, or CLAUDE_CODE_SESSION_ID unset); freeze again, then record" >&2
    return 1
  }
  checkpoint_write_state "$root" > "$file.tmp" || return 1
  rm -rf "$file.d.tmp" && mkdir -p "$file.d.tmp" || return 1
  sed 1d "$file.tmp" | while IFS=$'\t' read -r hash path; do
    [ "$hash" != - ] && [ -f "$root/$path" ] || continue
    grep -qxF -- "$path" "$file.frozen" 2>/dev/null || continue
    mkdir -p "$file.d.tmp/$(dirname "$path")" && cp "$root/$path" "$file.d.tmp/$path"
  done
  { cat "$file.frozen" 2>/dev/null || :; } > "$file.paths.tmp"
  rm -rf "$file.d" && mv "$file.d.tmp" "$file.d" && mv "$file.paths.tmp" "$file.paths" &&
    mv "$file.tmp" "$file" && checkpoint_close "$root"
}

# Covered while every dirty path still holds the bytes the last checkpoint
# reviewed. HEAD is ignored: committing part of reviewed work leaves the rest
# covered.
checkpoint_reviewed() {
  local root=$1 file now
  file=$(checkpoint_reviewed_file "$root") || return 1
  [ -r "$file" ] || return 1
  now=$(checkpoint_write_state "$root" | sed 1d)
  [ -n "$now" ] || return 1
  ! printf '%s\n' "$now" | grep -qvxF -f <(sed 1d "$file")
}

# The bytes the last checkpoint reviewed at a path: its copy, or /dev/null where
# it reviewed a deletion. Fails where that run did not freeze the path, the
# record holds neither, or the copy no longer hashes to what the record says.
checkpoint_reviewed_bytes() {
  local root=$1 path=$2 file hash
  file=$(checkpoint_reviewed_file "$root") || return 1
  grep -qxF -- "$path" "$file.paths" 2>/dev/null || return 1
  hash=$(sed 1d "$file" 2>/dev/null | P=$path awk -F'\t' '$2 == ENVIRON["P"] { print $1; exit }')
  case $hash in
    '') return 1 ;;
    -) echo /dev/null ;;
    *)
      [ -f "$file.d/$path" ] &&
        [ "$(git -C "$root" hash-object --path="$path" -- "$file.d/$path" 2>/dev/null)" = "$hash" ] &&
        printf '%s\n' "$file.d/$path"
      ;;
  esac
}

# Latest checkpoint per repository, not per session: identical bytes were
# reviewed whichever session asks.
checkpoint_reviewed_file() {
  local root dir
  # Keyed by git's own root: a symlinked or /private path names the same repo.
  root=$(git -C "$1" rev-parse --show-toplevel 2>/dev/null) || return 1
  dir="${CHECKPOINT_REVIEWED_DIR:-$HOME/.claude/cache/checkpoint-reviewed}"
  mkdir -p "$dir" 2>/dev/null || return 1
  printf '%s/%s' "$dir" "$(printf '%s' "$root" | shasum | cut -d' ' -f1)"
}

# One baseline per session per repository — a session may touch several.
checkpoint_state_file() {
  local session=$1 root=$2 dir key
  [ -n "$session" ] && [ -n "$root" ] || return 1
  dir="${TMPDIR:-/tmp}/claude-checkpoint"
  mkdir -p "$dir" 2>/dev/null || return 1
  key=$(printf '%s\n%s' "$session" "$root" | shasum | cut -d' ' -f1)
  printf '%s/%s' "$dir" "$key"
}

# Line 1 HEAD, then `hash<TAB>path`. A deleted path hashes as `-`.
checkpoint_write_state() {
  local root=$1 paths files path
  git -C "$root" rev-parse -q --verify HEAD 2>/dev/null || echo none
  paths=$(checkpoint_dirty_paths "$root")
  [ -n "$paths" ] || return 0
  files=$(printf '%s\n' "$paths" | while IFS= read -r path; do
    [ -f "$root/$path" ] && printf '%s\n' "$path"
  done)
  if [ -n "$files" ]; then
    paste <(printf '%s\n' "$files" | git -C "$root" hash-object --stdin-paths 2>/dev/null) \
      <(printf '%s\n' "$files")
  fi
  printf '%s\n' "$paths" | while IFS= read -r path; do
    if [ -f "$root/$path" ]; then
      continue
    elif [ -e "$root/$path" ]; then
      printf '%s\t%s\n' "$(git -C "$root" diff HEAD -- "$path" 2>/dev/null | git hash-object --stdin)" "$path"
    else
      printf -- '-\t%s\n' "$path"
    fi
  done
}
