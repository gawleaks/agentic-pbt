#!/bin/sh
set -eu

log=${1:?usage: assert-events.sh LOG EVENT...}
shift
[ -f "$log" ] || { printf 'missing probe log: %s\n' "$log" >&2; exit 1; }

for event in "$@"; do
  if ! grep -Fq "\"event\":\"$event\"" "$log"; then
    printf 'missing event %s in %s\n' "$event" "$log" >&2
    exit 1
  fi
done
