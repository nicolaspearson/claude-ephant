#!/bin/sh
# ELE_VERSION: a release tag, its v optional, default the latest. ELE_INSTALL_DIR: default ~/.local/bin.
set -eu

repository=nicolaspearson/claude-ephant

fail() {
	echo "install.sh: $*" >&2
	exit 1
}

# All in a function called on the last line, so a download cut short runs nothing.
main() {
	command -v curl >/dev/null || fail "needs curl"
	command -v tar >/dev/null || fail "needs tar"

	os=$(uname -s | tr '[:upper:]' '[:lower:]')
	case $os in
	darwin | linux) ;;
	*) fail "no release for $os" ;;
	esac
	case $(uname -m) in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*) fail "no release for $(uname -m)" ;;
	esac
	name="ele_${os}_${arch}"

	if [ -n "${ELE_VERSION:-}" ]; then
		case $ELE_VERSION in
		v*) version=$ELE_VERSION ;;
		*) version=v$ELE_VERSION ;;
		esac
		base="https://github.com/$repository/releases/download/$version"
	else
		base="https://github.com/$repository/releases/latest/download"
	fi
	directory=${ELE_INSTALL_DIR:-$HOME/.local/bin}

	work=$(mktemp -d)
	trap 'rm -rf "$work"' EXIT
	# dash runs no EXIT trap on a signal unless the signal's trap exits.
	trap 'exit 129' HUP
	trap 'exit 130' INT
	trap 'exit 143' TERM
	curl --proto '=https' -fsSL -o "$work/$name.tar.gz" "$base/$name.tar.gz" || fail "could not download $base/$name.tar.gz"
	curl --proto '=https' -fsSL -o "$work/checksums.txt" "$base/checksums.txt" || fail "could not download $base/checksums.txt"

	expected=$(awk -v file="$name.tar.gz" '$2 == file { print $1 }' "$work/checksums.txt")
	if command -v sha256sum >/dev/null; then
		actual=$(sha256sum "$work/$name.tar.gz" | cut -d' ' -f1)
	elif command -v shasum >/dev/null; then
		actual=$(shasum -a 256 "$work/$name.tar.gz" | cut -d' ' -f1)
	else
		fail "checking the download needs sha256sum or shasum"
	fi
	if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
		fail "$name.tar.gz does not match checksums.txt"
	fi

	tar -xzf "$work/$name.tar.gz" -C "$work"
	mkdir -p "$directory"
	# Same-directory rename: atomic, so the old ele survives a kill.
	install -m 0755 "$work/$name/ele" "$directory/.ele.new"
	mv -f "$directory/.ele.new" "$directory/ele"

	if installed=$("$directory/ele" version 2>/dev/null); then
		echo "Installed ele $installed at $directory/ele"
	else
		echo "Installed $directory/ele"
	fi
	case ":$PATH:" in
	*":$directory:"*) ;;
	*) echo "$directory is not on PATH; add it in your shell's profile: export PATH=\"$directory:\$PATH\"" ;;
	esac
}

main
