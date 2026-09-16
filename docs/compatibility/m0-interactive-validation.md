# M0 interactive validation runbook

Run these checks yourself from the `m0-runtime-probes` worktree. They use real
Pi/Claude sessions and may consume the credentials authenticated in those tools.
Do not run them until you have verified the intended account with `pi --version`
and `claude --version`.

Store logs under `/tmp`; do not commit them. Update
`docs/compatibility/runtime-integration-m0.md` only with observed results.

## 1. Prepare a Pi session

```bash
cd /path/to/agentic-pbt/.worktrees/m0-runtime-probes
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-pi.XXXXXX")"
testdata/m0-runtime-probes/pi/run.sh "$fixture" "$log"
```

In Pi:

1. Submit any short prompt. Confirm the labelled `agentic-pbt-m0-policy` message
   is visible in the transcript. This proves TUI injection, but not necessarily
   model delivery.
2. Submit `/m0-probe-follow-up` while Pi is idle. Confirm exactly one follow-up
   turn occurs and it replies `M0_PI_FOLLOW_UP_OK`.
3. Ask the agent to call `m0_probe_wait`, then press Escape while it runs.
   Confirm control returns and no follow-up was injected.
4. In a separate session, send one prompt and run `/compact`. Confirm the session
   remains usable afterward.

After exiting Pi, inspect the evidence:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" \
  session_start before_agent_start follow_up_queued cancellation_probe session_compact agent_settled
rg '"event":"cancellation_probe".*("aborted":true|"killed":true)' "$log"
```

Record the command, Pi version, result, and a redacted log excerpt in the M0
compatibility record. If any event is missing, record it as a limitation.

## 2. Check Pi worker loading

Start Pi with both the M0 probe and Pi's installed subagent example. Delegate one
read-only task to its worker. Record whether the worker log contains the M0 probe
events without explicitly passing `-e .../pi/probe.ts` to the worker process.

Repeat in a linked Git worktree. Parent evidence must remain scoped to the parent
worktree; worker evidence must not be treated as integrated parent evidence.

## 3. Fix the Claude hook probe before testing context delivery

The current probe logs events but drops every `additionalContext` payload on Claude
Code 2.1.246 because `hookSpecificOutput.hookEventName` is missing.

In `testdata/m0-runtime-probes/claude/scripts/record-hook.sh`, keep the first
`Stop` block response unchanged. Replace the final plain acknowledgement with:

```sh
if [ "$event" = "Stop" ]; then
  printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"Stop"}}'
else
  printf '%s\n' "{\"hookSpecificOutput\":{\"hookEventName\":\"${event}\",\"additionalContext\":\"M0_CLAUDE_${event}\"}}"
fi
```

This makes non-Stop context payloads valid while keeping later Stop events from
injecting more feedback. Validate before a live session:

```bash
claude plugin validate testdata/m0-runtime-probes/claude
```

## 4. Validate Claude context delivery and stop handling

```bash
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-claude.XXXXXX")"
M0_PROBE_LOG="$log" \
M0_PROBE_STOP_MARKER="${log}.stop-once" \
claude --plugin-dir testdata/m0-runtime-probes/claude
```

In the interactive session:

1. Submit a prompt that uses `Read` on `input.txt`. Confirm the injected
   `M0_CLAUDE_SessionStart`, `M0_CLAUDE_UserPromptSubmit`, and
   `M0_CLAUDE_PostToolUse` context is visible to Claude.
2. Let the agent stop. Confirm the first Stop asks for one continuation and the
   second Stop ends normally. Check whether `stop_hook_active` is present in the
   hook input before relying on it for loop protection.
3. Run a scoped `m0-worker` task, then repeat with a worker in a separate Git
   worktree.
4. Test `/resume`, `/compact`, and Escape cancellation in the interactive TUI.
5. Repeat one run with an existing user/project hook enabled to verify plugin-hook
   composition rather than replacement.

After each run:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" \
  SessionStart UserPromptSubmit PostToolUse Stop
```

For worker runs, also require:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" SubagentStart SubagentStop
```

## 5. Record results

For each capability, mark it `Live`, `Partial`, `Not run`, or `Not delivered` in
`docs/compatibility/runtime-integration-m0.md`. Include only redacted event lines,
the command, runtime version, and the observed limitation. Never call the current
Claude Stop behavior an unconditional guarantee.
