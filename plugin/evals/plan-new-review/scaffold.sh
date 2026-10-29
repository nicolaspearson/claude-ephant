#!/usr/bin/env bash
set -euo pipefail

data="$HOME/Library/Application Support/claude-ephant"
mkdir -p "$data"
printf '{"owners": ["eval"], "paths": {"%s": "eval", "%s": "eval"}, "projects": {"%s": "eval"}}\n' "$PWD" "$(pwd -P)" "$(basename "$PWD")" > "$data/owner-map.json"
printf '{"checkpoint_gate": false, "notes_ask": false}\n' > "$data/config.json"
# No git init: git cannot run in the eval sandbox, so a repository would go unread.

mkdir -p .documentation/specs docs/adr
cat > go.mod <<'GO'
module example.invalid/tally

go 1.24
GO
cat > main.go <<'GO'
package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanWords)
	words := 0
	for scanner.Scan() {
		words++
	}
	fmt.Println(words)
}
GO
cat > .documentation/specs/design.md <<'MD'
# tally

`tally` counts the words on standard input and prints the count.

## Input

Standard input or files named as arguments, read in order. `tally` reads
local input only, as docs/adr/0001-local-input-only.md decides.

## Output

One integer and a newline on standard output. Errors go to standard error
and exit 1.
MD
cat > docs/adr/0001-local-input-only.md <<'MD'
# 1. Local input only

Status: accepted

## Context

tally runs in build pipelines with no network, and on untrusted machines.

## Decision

tally reads standard input and local files only. It never opens a network
connection, fetches a URL or resolves a host name.

## Consequences

Counting a web page means fetching it with another tool and piping it in.
MD
