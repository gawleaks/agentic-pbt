#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root/testdata/v1-python"
go run "$root/cmd/pbt" begin --task discount || true
go run "$root/cmd/pbt" verify --task discount --json
[ "$(go run "$root/cmd/pbt" gate --task discount --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["verificationState"])')" = "passed" ]
printf '# freshness edit\n' >> src/discount.py
if go run "$root/cmd/pbt" gate --task discount --json; then
  echo "expected stale gate failure" >&2
  exit 1
fi
