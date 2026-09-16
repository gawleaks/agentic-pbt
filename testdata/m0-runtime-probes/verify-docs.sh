#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
compat="$root/docs/compatibility/runtime-integration-m0.md"
progress="$root/docs/implementation-progress.md"

require() {
  grep -Fq "$1" "$2" || { printf 'missing %s in %s\n' "$1" "$2" >&2; exit 1; }
}

require '**Status:** In progress' "$compat"
require '**Not delivered** on 2.1.246' "$compat"
require 'M3 (Claude Code plugin) must include `hookEventName` inside **every** `hookSpecificOutput`' "$compat"
require '## Unproven M0 points' "$compat"
require '**State:** In progress' "$progress"
require 'Pi policy delivery, queued follow-up, cancellation, compaction, and worker-extension loading are implemented but not live-verified.' "$progress"
require 'Claude `additionalContext` delivery is disproven for the current probe on 2.1.246' "$progress"

printf '%s\n' 'PASS: M0 documentation self-check'
