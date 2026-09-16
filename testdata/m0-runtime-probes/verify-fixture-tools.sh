#!/bin/sh
# TDD check for create-fixture.sh and assert-events.sh.
# RED before the two scripts exist; GREEN after.
set -eu

dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

fixture="$("$dir/create-fixture.sh")"
trap 'rm -rf "$fixture"' EXIT

[ -d "$fixture/.git" ] || { echo "FAIL: fixture is not a git repo: $fixture" >&2; exit 1; }

case "$fixture" in
  /*) ;;
  *) echo "FAIL: fixture path is not absolute: $fixture" >&2; exit 1 ;;
esac

log_line="$(git -C "$fixture" log --oneline -1)"
case "$log_line" in
  *"chore: initialize M0 probe fixture"*) ;;
  *) echo "FAIL: unexpected git log: $log_line" >&2; exit 1 ;;
esac

log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-log.XXXXXX")"
trap 'rm -rf "$fixture" "$log"' EXIT
printf '%s\n' '{"event":"fixture_created"}' > "$log"

if ! "$dir/assert-events.sh" "$log" fixture_created; then
  echo "FAIL: assert-events.sh should have found fixture_created" >&2
  exit 1
fi

if "$dir/assert-events.sh" "$log" absent_event; then
  echo "FAIL: assert-events.sh should have failed for absent_event" >&2
  exit 1
fi

echo "PASS: create-fixture.sh and assert-events.sh behave as required"
