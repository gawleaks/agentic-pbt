#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
source="$root/integrations/pi/index.ts"

node "$root/integrations/pi/contract-test.mjs"
"$root/testdata/v1-pi/verify-extension.sh"
for forbidden in registerTool registerCommand sendUserMessage agent_end; do
  if grep -Fq "$forbidden" "$source"; then
    printf '%s\n' "FAIL: index.ts contains $forbidden" >&2
    exit 1
  fi
done
printf '%s\n' 'PASS: Pi adapter has no tools, commands, user messages, or agent_end hook'
