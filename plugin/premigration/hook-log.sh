#!/usr/bin/env bash
# Appends to the machine-wide hook log in the format the other hooks use.
# Sourced, never executed.

# Unset by default: the file rotates by month. The suites set it to keep their
# fixtures out of the record.
HOOK_LOG=${HOOK_LOG:-}

hook_class() {
  case "$1" in
    checkpoint) printf quality ;;
    *) printf unclassified ;;
  esac
}

hook_log() { # decision rule [target]
  local stamp file
  stamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  file=${HOOK_LOG:-$HOME/.claude/logs/hooks-${stamp:0:7}.log}
  mkdir -p "$(dirname "$file")" 2> /dev/null || return 0
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$stamp" "$(hook_class "$2")" "$1" "$2" \
    "$(printf '%s' "${3:-}" | tr '\t\n' '  ' | cut -c1-120)" \
    "${PWD##*/}" \
    >> "$file" 2> /dev/null
  return 0
}
