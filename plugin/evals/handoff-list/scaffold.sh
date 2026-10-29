#!/usr/bin/env bash
set -euo pipefail

data="$HOME/Library/Application Support/claude-ephant"
mkdir -p "$data/notes/eval"
printf '{"owners": ["eval"], "paths": {"%s": "eval", "%s": "eval"}}\n' "$PWD" "$(pwd -P)" > "$data/owner-map.json"
git init -q
project="$(basename "$PWD")"

handoff() {
	printf -- '---\ndate: %s\nkind: handoff\nowner: eval\nproject: %s\nscope: project\ntime: %s\n%b---\n%s\n' "${2%%T*}" "$project" "$2" "$3" "$4" > "$data/notes/eval/$1.md"
}
# Two pending, so the session's own start leaves them pending; a lone one would be marked shown.
handoff 20261020-aaaaaa 2026-10-20T09:00:00Z 'shown: 2026-10-20T10:00:00Z\n' 'Next: rotate the staging certificate.'
handoff 20261021-bbbbbb 2026-10-21T09:00:00Z '' 'Next: add retries to the upload client.'
handoff 20261022-cccccc 2026-10-22T09:00:00Z '' 'Next: pin pydantic below 2.14.'
