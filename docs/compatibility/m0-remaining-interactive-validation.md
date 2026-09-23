# Remaining M0 interactive validation

Run these checks one at a time from the M0 worktree. They use the credentials
currently authenticated in Pi or Claude Code; verify the intended account before
starting. Keep each log under `/tmp`, do not commit it, and record only redacted
results in `docs/compatibility/runtime-integration-m0.md`.

```bash
cd /path/to/agentic-pbt/.worktrees/m0-runtime-probes
```

M0 remains **in progress** until these checks are recorded as passed, unsupported,
or limited by an observed runtime constraint.

## Pi status

Policy delivery is complete: the interactive prompt `M0 policy delivery check`
produced the exact policy-directed response `M0_PI_POLICY_DELIVERED`.

Pi worker inheritance and separate-worktree validation are unsupported in the
current setup. There is no installed Pi worker launcher or agent definition; Pi's
bundled worker example is fixed to an Anthropic model, for which use is not
approved. Do not install it or use credentials to satisfy this M0 item. Revisit
only after a permitted worker model is configured.

## 1. Capture Claude `stop_hook_active`

Use a disposable plugin copy; hook stdin can contain session details and must not
be committed:

```bash
cd /path/to/agentic-pbt/.worktrees/m0-runtime-probes
plugin="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-claude-plugin.XXXXXX")"
cp -R testdata/m0-runtime-probes/claude/. "$plugin"
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-claude-stop-active.XXXXXX")"
```

In `$plugin/scripts/record-hook.sh`, immediately after the two `M0_PROBE_...`
environment checks, add the following. It preserves a separate stdin file for
each hook process:

```sh
stdin_file="${M0_PROBE_LOG}.${event}.$$.stdin"
cat > "$stdin_file"
```

Then add this block immediately **before** the existing first-Stop block. On a
re-entrant Stop, it records that the flag was honored and allows Claude to stop:

```sh
if [ "$event" = "Stop" ] && grep -Fq '"stop_hook_active":true' "$stdin_file"; then
  printf '{"timestamp":"%s","runtime":"claude","event":"stop_hook_active_allow"}\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$M0_PROBE_LOG"
  printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"Stop"}}'
  exit 0
fi
```

Launch an interactive session with only the temporary plugin:

```bash
: > "$log"
M0_PROBE_LOG="$log" \
M0_PROBE_STOP_MARKER="${log}.stop-once" \
claude --plugin-dir "$plugin" --debug hooks
```

Submit a short read-only prompt and let Claude finish. The first `Stop` should
block once; the re-entrant `Stop` should be allowed by the new guard. After exit:

```bash
rg -n '"stop_hook_active"' "${log}".Stop.*.stdin || true
rg -n '"event":"stop_hook_active_allow"' "$log" || true
```

Pass condition: a captured `Stop` input contains `"stop_hook_active":true` and
the log contains `stop_hook_active_allow`. If the field is absent or never true,
record that observed limitation instead. Remove the disposable files afterward:

```bash
rm -rf "$plugin" "$fixture" "$log" "${log}.stop-once" "${log}".*.stdin
```

## 2. Test Claude hook composition

Use a fixture that already has one harmless project or user hook enabled, for
example a hook that appends `existing-hook-ran` to a temporary log. Start Claude
with both that hook and:

```bash
claude --plugin-dir testdata/m0-runtime-probes/claude
```

Submit a read-only prompt. Pass condition: both the existing hook's log and the
M0 log contain their expected event entries. Record whether the two hooks ran in a
stable order, and whether either suppressed the other.

## 3. Test Claude interactive cancellation, resume, and compaction

Start the normal M0 Claude session:

```bash
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-claude-interactive.XXXXXX")"
testdata/m0-runtime-probes/claude/run.sh "$fixture" "$log"
```

Run these in separate sessions so each result is unambiguous:

1. **Escape cancellation:** ask Claude to start a long Bash operation, then press
   Escape. Record whether `PostToolUse` or `Stop` fires after cancellation.
2. **Resume picker:** use `/resume`, choose the just-created session, submit a
   read-only prompt, and confirm `SessionStart` is logged again.
3. **Compaction:** create enough conversation history to compact, run `/compact`,
   then submit a read-only prompt. Confirm the session remains usable and record
   the event sequence.

For each session, save its own `$log` path before launching and run:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" SessionStart
```

Add further required events only when they actually occurred. Missing events are
results to record, not failures to hide.

## 4. Run Claude `m0-worker` in a separate worktree

Create a disposable worktree with a distinct branch name:

```bash
child="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-claude-worker.XXXXXX")"
rm -rf "$child"
git worktree add "$child" -b m0-claude-worker-validation HEAD
```

Start the parent Claude session with the M0 plugin. Assign `m0-worker` a read-only
task whose working directory is `$child`. Confirm:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" SubagentStart SubagentStop
```

Record the parent and worker paths and state explicitly that worker evidence is
scoped to the worker; it does not verify the parent's integrated files. Remove the
child worktree and branch afterward.

## 5. Hand results back for recording

For each step, provide:

- runtime and version;
- exact command or UI action;
- pass, unsupported, or limitation outcome;
- redacted relevant log lines; and
- whether the result changes an M2/M3 assumption.

The agent can then update the compatibility record, run offline checks, and commit
the documented result.
