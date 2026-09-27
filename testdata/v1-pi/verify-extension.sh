#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
pbt="$root/testdata/v1-pi/pbt"

[ -x "$pbt" ]
"$pbt" status --json | node -e 'let data=""; process.stdin.on("data", chunk => data += chunk).on("end", () => { const result = JSON.parse(data); if (result.schemaVersion !== 1 || result.task !== "repair") process.exit(1); })'
if output=$("$pbt" gate --task repair --json); then
  printf '%s\n' 'FAIL: failed gate exited zero' >&2
  exit 1
fi
printf '%s' "$output" | node -e 'let data=""; process.stdin.on("data", chunk => data += chunk).on("end", () => { const result = JSON.parse(data); if (result.followUp !== "Run pbt verify") process.exit(1); })'
printf '%s\n' 'PASS: v1 Pi pbt fixture'
