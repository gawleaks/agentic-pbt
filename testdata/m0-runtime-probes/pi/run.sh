#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture=${1:?usage: run.sh FIXTURE LOG}
log=${2:?usage: run.sh FIXTURE LOG}

: > "$log"
export M0_PROBE_LOG="$log"
cd "$fixture"
exec pi -e "$root/pi/probe.ts"
