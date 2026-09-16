#!/bin/sh
set -eu

fixture="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0.XXXXXX")"
trap 'rm -rf "$fixture"' INT TERM HUP

git -C "$fixture" init -q
git -C "$fixture" config user.email "m0-probe@example.invalid"
git -C "$fixture" config user.name "agentic-pbt M0 probe"
printf '%s\n' '# M0 disposable fixture' > "$fixture/README.md"
printf '%s\n' 'probe input' > "$fixture/input.txt"
git -C "$fixture" add README.md input.txt
git -C "$fixture" commit -qm 'chore: initialize M0 probe fixture'

printf '%s\n' "$fixture"
trap - INT TERM HUP
