#!/bin/sh
# Regression check for the review finding on verify-probe.sh: the
# M0_PROBE_PI_LIVE_CHECK=1 EXIT trap must clean up every temp path
# (including contract_dir/contract_log from the earlier trap), not just
# the ones re-listed in its own trap line.
set -eu

dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

scratch="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-cleanup-check.XXXXXX")"
stub_pi_dir="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-cleanup-stub.XXXXXX")"
trap 'rm -rf "$scratch" "$stub_pi_dir"' EXIT

cat > "$stub_pi_dir/pi" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod +x "$stub_pi_dir/pi"

TMPDIR="$scratch" PATH="$stub_pi_dir:$PATH" M0_PROBE_PI_LIVE_CHECK=1 "$dir/verify-probe.sh" >/dev/null 2>&1 || true

leftover="$(find "$scratch" -mindepth 1 2>/dev/null || true)"
if [ -n "$leftover" ]; then
  echo "FAIL: verify-probe.sh leaked temp paths on live-check EXIT trap:" >&2
  printf '%s\n' "$leftover" >&2
  exit 1
fi

echo "PASS: verify-probe.sh live-check EXIT trap cleans up all temp paths"
