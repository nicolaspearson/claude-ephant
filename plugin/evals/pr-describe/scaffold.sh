#!/usr/bin/env bash
set -euo pipefail

data="$HOME/Library/Application Support/claude-ephant"
mkdir -p "$data"
printf '{"owners": ["eval"], "paths": {"%s": "eval", "%s": "eval"}}\n' "$PWD" "$(pwd -P)" > "$data/owner-map.json"
printf '{"checkpoint_gate": false, "notes_ask": false}\n' > "$data/config.json"
# A repository of its own: git otherwise roots at the sandbox HOME, whose files read as uncommitted work (as of 2026-10-23).
# The session's /usr/bin/git fails here, Apple's shim unable to write its xcrun cache, while Homebrew's /opt/homebrew/bin/git runs (as of 2026-11-17, claude 2.1.311); a machine without one is not tried.
git init -q -b main
git remote add origin https://github.com/example/upload.git
cat > upload.go <<'EOF'
package upload

const maxRetries = 3
EOF
git add upload.go
git -c user.name=eval -c user.email=eval@example.invalid commit -q -m "upload: client"
# Stands in for a fetched origin, so origin/HEAD names main with nothing pushed.
git update-ref refs/remotes/origin/main HEAD
git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/main

git switch -q -c raise-retries
cat > upload.go <<'EOF'
package upload

// The storage API rate-limits bursts, so seven spaced attempts outlast a limit window.
const maxRetries = 7
EOF
git -c user.name=eval -c user.email=eval@example.invalid commit -q -am "upload: raise retries from 3 to 7"
