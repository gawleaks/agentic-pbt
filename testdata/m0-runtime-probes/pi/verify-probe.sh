#!/bin/sh
# TDD check for the Pi probe extension and launcher.
# RED before pi/probe.ts and pi/run.sh exist; GREEN after.
#
# Verifies the parts that are mechanically checkable without a live model
# call: file presence, run.sh's argument validation, log truncation, and
# cwd/env handoff to `pi -e`; and, via contract-test.mjs, the follow-up
# command and cancellation tool logic added in Task 3 (mocked pi API, no
# interactive/live pi process). Set M0_PROBE_PI_LIVE_CHECK=1 to also
# exercise probe.ts against a real `pi` process (needs a configured model).
set -eu

dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
root="$(CDPATH= cd -- "$dir/.." && pwd)"

[ -f "$dir/probe.ts" ] || { echo "FAIL: missing pi/probe.ts" >&2; exit 1; }
[ -x "$dir/run.sh" ] || { echo "FAIL: pi/run.sh missing or not executable" >&2; exit 1; }

stub_dir="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-pi-stub.XXXXXX")"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-pi-fixture.XXXXXX")"
fixture="$(CDPATH= cd -- "$fixture" && pwd)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-pi-log.XXXXXX")"
contract_dir="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-pi-contract.XXXXXX")"
contract_log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-pi-contract-log.XXXXXX")"
trap 'rm -rf "$stub_dir" "$fixture" "$log" "$log.stub" "$contract_dir" "$contract_log"' EXIT

cat > "$stub_dir/pi" <<'EOF'
#!/bin/sh
printf 'cwd=%s\n' "$PWD" > "$M0_PROBE_LOG.stub"
printf 'log=%s\n' "$M0_PROBE_LOG" >> "$M0_PROBE_LOG.stub"
printf 'args=%s\n' "$*" >> "$M0_PROBE_LOG.stub"
EOF
chmod +x "$stub_dir/pi"

if "$dir/run.sh" 2>/dev/null; then
  echo "FAIL: run.sh should error with no arguments" >&2
  exit 1
fi

if "$dir/run.sh" "$fixture" 2>/dev/null; then
  echo "FAIL: run.sh should error with only a fixture argument" >&2
  exit 1
fi

printf 'pre-existing line that must be truncated\n' > "$log"
PATH="$stub_dir:$PATH" "$dir/run.sh" "$fixture" "$log"

[ ! -s "$log" ] || { echo "FAIL: run.sh must truncate the log before launch" >&2; exit 1; }
grep -Fq "cwd=$fixture" "$log.stub" || { echo "FAIL: run.sh did not cd into the fixture" >&2; exit 1; }
grep -Fq "log=$log" "$log.stub" || { echo "FAIL: run.sh did not export M0_PROBE_LOG" >&2; exit 1; }
grep -Fq "args=-e $root/pi/probe.ts" "$log.stub" || { echo "FAIL: run.sh did not launch probe.ts" >&2; exit 1; }

echo "PASS: run.sh contract (usage, truncation, cwd, env, launch args)"

mkdir -p "$contract_dir/node_modules/typebox"
cat > "$contract_dir/node_modules/typebox/package.json" <<'EOF'
{"name":"typebox","version":"0.0.0","type":"module","main":"index.mjs"}
EOF
cat > "$contract_dir/node_modules/typebox/index.mjs" <<'EOF'
export const Type = { Object: (schema) => schema };
EOF
cp "$dir/probe.ts" "$contract_dir/probe.ts"
: > "$contract_log"
M0_PROBE_LOG="$contract_log" M0_PROBE_TS_PATH="$contract_dir/probe.ts" node "$dir/contract-test.mjs"

if [ -n "${M0_PROBE_PI_LIVE_CHECK:-}" ]; then
  live_fixture="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-pi-live.XXXXXX")"
  live_log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-pi-live-log.XXXXXX")"
  trap 'rm -rf "$stub_dir" "$fixture" "$log" "$log.stub" "$contract_dir" "$contract_log" "$live_fixture" "$live_log"' EXIT
  git -C "$live_fixture" init -q
  M0_PROBE_LOG="$live_log" timeout 20 pi -e "$dir/probe.ts" --no-session -p "Reply exactly: PROBE_CHECK" >/dev/null 2>&1 || true
  grep -Fq '"event":"session_start"' "$live_log" || { echo "FAIL: probe.ts did not record session_start" >&2; exit 1; }
  grep -Fq '"event":"before_agent_start"' "$live_log" || { echo "FAIL: probe.ts did not record before_agent_start" >&2; exit 1; }
  echo "PASS: probe.ts records session_start and before_agent_start via a live pi run"
else
  echo "SKIP: live pi run (set M0_PROBE_PI_LIVE_CHECK=1 to enable)"
fi
