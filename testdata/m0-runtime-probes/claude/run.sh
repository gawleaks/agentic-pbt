#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture=${1:?usage: run.sh FIXTURE}
log=${2:?usage: run.sh FIXTURE LOG}

: > "$log"
export M0_PROBE_LOG="$log"
export M0_PROBE_STOP_MARKER="${log}.stop-once"
rm -f "$M0_PROBE_STOP_MARKER"
cd "$fixture"
exec claude --plugin-dir "$root/claude" --debug hooks
