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

## 1. Prove Pi policy delivery to the model

Start a fresh Pi session:

```bash
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-pi-policy.XXXXXX")"
testdata/m0-runtime-probes/pi/run.sh "$fixture" "$log"
```

Submit this prompt exactly; do not include the policy text or marker in it:

```text
Quote the exact M0 policy message you received. Reply with only that message.
```

Pass condition: Pi replies exactly:

```text
M0 probe: policy injection reached this Pi turn.
```

Exit Pi, then retain the log evidence:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" session_start before_agent_start \
  && echo "PASS: Pi policy hook fired"
```

Record both the exact reply and the event assertion. The reply proves model
visibility; the log alone only proves the event handler ran.

## 2. Determine whether a Pi worker inherits the probe

Use the Pi worker/subagent launcher you normally use. Start its parent with the M0
probe enabled, then assign the worker one read-only request such as:

```text
Read input.txt and return its exact contents. Do not edit files.
```

Check the worker's process environment or worker probe log:

- **Inherited:** the worker writes M0 events without explicitly receiving
  `-e testdata/m0-runtime-probes/pi/probe.ts`.
- **Not inherited:** it writes no M0 events until the probe is passed explicitly.
- **Unsupported:** your Pi worker launcher cannot launch a child with an extension.

Record exactly one of those three outcomes. Do not call parent evidence proof for
worker evidence.

## 3. Repeat the Pi worker check in a separate worktree

Create a disposable child worktree from the M0 branch:

```bash
parent="$(pwd)"
child="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0-pi-worker.XXXXXX")"
rm -rf "$child"
git worktree add "$child" -b m0-pi-worker-validation HEAD
```

Launch the worker against `$child` and repeat Step 2's read-only task. Record:

- parent worktree path;
- child worktree path;
- whether the probe was inherited or explicitly loaded; and
- that the parent did **not** treat worker evidence as its own integrated result.

Clean up afterward:

```bash
git worktree remove "$child"
git branch -D m0-pi-worker-validation
```

## 4. Capture Claude `stop_hook_active`

The current hook recorder does not retain hook stdin. Make a temporary local copy
of the Claude plugin under `/tmp` and instrument only that copy. Immediately after
`event=...` in its `scripts/record-hook.sh`, add:

```sh
cat > "${M0_PROBE_LOG}.${event}.stdin"
```

Start Claude with the temporary plugin and trigger a Stop. After the session exits:

```bash
rg '"stop_hook_active":(true|false)' "${log}.Stop.stdin"
```

Record whether the field is present and its value on the first and later Stop
calls. Do not commit captured hook input; it can contain session details. If the
field is absent, record `unsupported or absent on tested runtime`.

## 5. Test Claude hook composition

Use a fixture that already has one harmless project or user hook enabled, for
example a hook that appends `existing-hook-ran` to a temporary log. Start Claude
with both that hook and:

```bash
claude --plugin-dir testdata/m0-runtime-probes/claude
```

Submit a read-only prompt. Pass condition: both the existing hook's log and the
M0 log contain their expected event entries. Record whether the two hooks ran in a
stable order, and whether either suppressed the other.

## 6. Test Claude interactive cancellation, resume, and compaction

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

## 7. Run Claude `m0-worker` in a separate worktree

Create a disposable worktree as in Step 3, but use a distinct branch name:

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

## 8. Hand results back for recording

For each step, provide:

- runtime and version;
- exact command or UI action;
- pass, unsupported, or limitation outcome;
- redacted relevant log lines; and
- whether the result changes an M2/M3 assumption.

The agent can then update the compatibility record, run offline checks, and commit
the documented result.
