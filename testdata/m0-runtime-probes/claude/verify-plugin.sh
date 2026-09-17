#!/bin/sh
# TDD check for the Claude Code plugin hook probe.
# RED before the plugin files exist; GREEN after.
#
# Verifies the parts that are mechanically checkable without an
# interactive Claude session: file/executable presence, record-hook.sh's
# env/arg validation and JSONL output (including the bounded one-Stop
# marker), and run.sh's argument validation, log truncation, marker reset,
# and cwd/env handoff. Also runs `claude plugin validate` against the
# plugin directory (schema-only, no live session) when a `claude` binary
# is available on PATH.
set -eu

dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

[ -f "$dir/.claude-plugin/plugin.json" ] || { echo "FAIL: missing .claude-plugin/plugin.json" >&2; exit 1; }
[ -f "$dir/hooks/hooks.json" ] || { echo "FAIL: missing hooks/hooks.json" >&2; exit 1; }
[ -f "$dir/agents/m0-worker.md" ] || { echo "FAIL: missing agents/m0-worker.md" >&2; exit 1; }
[ -x "$dir/scripts/record-hook.sh" ] || { echo "FAIL: scripts/record-hook.sh missing or not executable" >&2; exit 1; }
[ -x "$dir/run.sh" ] || { echo "FAIL: run.sh missing or not executable" >&2; exit 1; }

log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-claude-log.XXXXXX")"
marker="${log}.stop-once"
rm -f "$marker"
trap 'rm -rf "$log" "$marker"' EXIT

if "$dir/scripts/record-hook.sh" SessionStart 2>/dev/null; then
  echo "FAIL: record-hook.sh should require M0_PROBE_LOG/M0_PROBE_STOP_MARKER" >&2
  exit 1
fi

if M0_PROBE_LOG="$log" M0_PROBE_STOP_MARKER="$marker" "$dir/scripts/record-hook.sh" 2>/dev/null; then
  echo "FAIL: record-hook.sh should require an event argument" >&2
  exit 1
fi

out="$(M0_PROBE_LOG="$log" M0_PROBE_STOP_MARKER="$marker" "$dir/scripts/record-hook.sh" SessionStart)"
grep -Fq '"event":"SessionStart"' "$log" || { echo "FAIL: SessionStart not recorded" >&2; exit 1; }
grep -Fq '"runtime":"claude"' "$log" || { echo "FAIL: runtime field missing" >&2; exit 1; }
case "$out" in
    *'"hookEventName":"SessionStart"'*'"additionalContext":"M0_CLAUDE_SessionStart"'*) ;;
    *) echo "FAIL: unexpected SessionStart stdout: $out" >&2; exit 1 ;;
esac

[ ! -e "$marker" ] || { echo "FAIL: marker should not exist before first Stop" >&2; exit 1; }

out="$(M0_PROBE_LOG="$log" M0_PROBE_STOP_MARKER="$marker" "$dir/scripts/record-hook.sh" Stop)"
[ -e "$marker" ] || { echo "FAIL: first Stop must create the marker" >&2; exit 1; }
case "$out" in
  *'"decision":"block"'*'"reason":"M0_PROBE_STOP_ONCE"'*) ;;
  *) echo "FAIL: first Stop did not request continuation: $out" >&2; exit 1 ;;
esac
grep -Fq '"event":"Stop"' "$log" || { echo "FAIL: first Stop not recorded" >&2; exit 1; }

stop_count_before="$(grep -Fc '"event":"Stop"' "$log")"
out="$(M0_PROBE_LOG="$log" M0_PROBE_STOP_MARKER="$marker" "$dir/scripts/record-hook.sh" Stop)"
stop_count_after="$(grep -Fc '"event":"Stop"' "$log")"
[ "$stop_count_after" -eq $((stop_count_before + 1)) ] || { echo "FAIL: second Stop should still be recorded" >&2; exit 1; }
case "$out" in
  *'"decision":"block"'*) echo "FAIL: second Stop must not request another block: $out" >&2; exit 1 ;;
esac
case "$out" in
    *'"hookEventName":"Stop"'*) ;;
    *) echo "FAIL: second Stop missing hookEventName: $out" >&2; exit 1 ;;
esac
case "$out" in
    *'"additionalContext"'*) echo "FAIL: second Stop must not inject additionalContext: $out" >&2; exit 1 ;;
esac

out="$(M0_PROBE_LOG="$log" M0_PROBE_STOP_MARKER="$marker" \
    "$dir/scripts/record-hook.sh" SubagentStop)"

case "$out" in
    *'"hookEventName":"SubagentStop"'*) ;;
    *) echo "FAIL: SubagentStop missing hookEventName: $out" >&2; exit 1 ;;
esac
case "$out" in
    *'"additionalContext"'*) echo "FAIL: SubagentStop must not inject additionalContext: $out" >&2; exit 1 ;;
esac

echo "PASS: record-hook.sh contract (env/arg validation, JSONL shape, bounded one-Stop marker)"

stub_dir="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-claude-stub.XXXXXX")"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-claude-fixture.XXXXXX")"
fixture="$(CDPATH= cd -- "$fixture" && pwd)"
run_log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-claude-run-log.XXXXXX")"
validate_out="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-claude-validate.XXXXXX")"
trap 'rm -rf "$log" "$marker" "$stub_dir" "$fixture" "$run_log" "${run_log}.stop-once" "$run_log.stub" "$validate_out"' EXIT

cat > "$stub_dir/claude" <<'EOF'
#!/bin/sh
printf 'cwd=%s\n' "$PWD" > "$M0_PROBE_LOG.stub"
printf 'log=%s\n' "$M0_PROBE_LOG" >> "$M0_PROBE_LOG.stub"
printf 'marker=%s\n' "$M0_PROBE_STOP_MARKER" >> "$M0_PROBE_LOG.stub"
printf 'args=%s\n' "$*" >> "$M0_PROBE_LOG.stub"
EOF
chmod +x "$stub_dir/claude"

if "$dir/run.sh" 2>/dev/null; then
  echo "FAIL: run.sh should error with no arguments" >&2
  exit 1
fi

if "$dir/run.sh" "$fixture" 2>/dev/null; then
  echo "FAIL: run.sh should error with only a fixture argument" >&2
  exit 1
fi

printf 'pre-existing line that must be truncated\n' > "$run_log"
: > "${run_log}.stop-once"
PATH="$stub_dir:$PATH" "$dir/run.sh" "$fixture" "$run_log"

[ ! -s "$run_log" ] || { echo "FAIL: run.sh must truncate the log before launch" >&2; exit 1; }
[ ! -e "${run_log}.stop-once" ] || { echo "FAIL: run.sh must reset the stop marker" >&2; exit 1; }
grep -Fq "cwd=$fixture" "$run_log.stub" || { echo "FAIL: run.sh did not cd into the fixture" >&2; exit 1; }
grep -Fq "log=$run_log" "$run_log.stub" || { echo "FAIL: run.sh did not export M0_PROBE_LOG" >&2; exit 1; }
grep -Fq "marker=${run_log}.stop-once" "$run_log.stub" || { echo "FAIL: run.sh did not export M0_PROBE_STOP_MARKER" >&2; exit 1; }
grep -Fq "args=--plugin-dir $dir --debug hooks" "$run_log.stub" || { echo "FAIL: run.sh did not launch claude with the expected args" >&2; exit 1; }

echo "PASS: run.sh contract (usage, truncation, marker reset, cwd, env, launch args)"

if command -v claude >/dev/null 2>&1; then
  if claude plugin validate "$dir" >"$validate_out" 2>&1; then
    echo "PASS: claude plugin validate"
  else
    echo "FAIL: claude plugin validate"
    cat "$validate_out" >&2
    exit 1
  fi
else
  echo "SKIP: claude plugin validate (claude binary not found on PATH)"
fi
