#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root/testdata/v1-python"

write_correct() {
  cat > src/discount.py <<'PY'
def apply_discount(subtotal: int, percent: int) -> int:
    return subtotal * (100 - percent) // 100
PY
}

assert_state() {
  printf '%s\n' "$1" | python3 -c '
import json
import sys

actual = json.load(sys.stdin)["verificationState"]
expected = sys.argv[1]
if actual != expected:
    sys.exit(f"verificationState mismatch: actual={actual!r}, expected={expected!r}")
' "$2"
}

tmpdir=$(mktemp -d)
cleanup() {
  write_correct
  rm -rf "$tmpdir"
}
trap cleanup 0 HUP INT TERM

rm -rf .pbt/state .pbt/runs
asdf exec go build -o "$tmpdir/pbt" "$root/cmd/pbt"
pbt="$tmpdir/pbt"

cat > src/discount.py <<'PY'
def apply_discount(subtotal: int, percent: int) -> int:
    return subtotal - percent
PY

"$pbt" begin --task discount
if failed=$("$pbt" verify --task discount --json); then
  echo "expected failed verification" >&2
  exit 1
else
  status=$?
fi
[ "$status" -eq 1 ]
assert_state "$failed" failed

write_correct
if passed=$("$pbt" verify --task discount --json); then
  status=$?
else
  echo "expected passed verification" >&2
  exit 1
fi
[ "$status" -eq 0 ]
assert_state "$passed" passed

if gated=$("$pbt" gate --task discount --json); then
  status=$?
else
  echo "expected passed gate" >&2
  exit 1
fi
[ "$status" -eq 0 ]
assert_state "$gated" passed

printf '# freshness edit\n' >> src/discount.py
if stale=$("$pbt" gate --task discount --json); then
  echo "expected stale gate failure" >&2
  exit 1
else
  status=$?
fi
[ "$status" -eq 2 ]
assert_state "$stale" stale
