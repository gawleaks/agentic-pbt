#!/bin/sh
set -eu

event=${1:?missing hook event name}
: "${M0_PROBE_LOG:?M0_PROBE_LOG must be set}"
: "${M0_PROBE_STOP_MARKER:?M0_PROBE_STOP_MARKER must be set}"

printf '{"timestamp":"%s","runtime":"claude","event":"%s","cwd":"%s","hookEventName":"%s"}\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$event" "$PWD" "${CLAUDE_HOOK_EVENT_NAME:-}" >> "$M0_PROBE_LOG"

if [ "$event" = "Stop" ] && [ ! -e "$M0_PROBE_STOP_MARKER" ]; then
  : > "$M0_PROBE_STOP_MARKER"
  printf '%s\n' '{"decision":"block","reason":"M0_PROBE_STOP_ONCE"}'
  exit 0
fi

printf '%s\n' "{\"hookSpecificOutput\":{\"additionalContext\":\"M0_CLAUDE_${event}\"}}"
