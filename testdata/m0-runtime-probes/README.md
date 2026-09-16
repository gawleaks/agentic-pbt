# M0 runtime probes

Run probes only in a temporary Git fixture created by `./create-fixture.sh`.
Set `M0_PROBE_LOG` to an absolute JSONL path before launching either runtime.
Each record has `timestamp`, `runtime`, `event`, and `cwd`; logs may contain
prompt text or paths, so keep them outside the repository and delete them when
the compatibility record has captured the result.

A passing script assertion proves that an event reached the probe. It does not
replace the manual observation noted by the matching command below.

The M0 go/no-go record built from these probes is
[`docs/compatibility/runtime-integration-m0.md`](../../docs/compatibility/runtime-integration-m0.md).

## Launch commands and manual observations

### Pi

```bash
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-pi.XXXXXX")"
testdata/m0-runtime-probes/pi/run.sh "$fixture" "$log"
```

`testdata/m0-runtime-probes/pi/verify-probe.sh` covers everything that is
mechanically checkable without an interactive session: `run.sh`'s argument
handling, and (via `pi/contract-test.mjs`, a mocked `pi` ExtensionAPI with
no live process) the `m0-probe-follow-up` command and `m0_probe_wait` tool
logic — one-time follow-up dedup, `sendMessage(..., { deliverAs: "followUp",
triggerTurn: true })`, and `cancellation_probe` recording of
`killed`/`aborted`. Set `M0_PROBE_PI_LIVE_CHECK=1` to additionally run a
bounded (20s), non-interactive `pi -e probe.ts --no-session -p "..."` smoke
check that confirms the extension still loads and records `session_start`
and `before_agent_start` under a real `pi` process.

**Manual-only observations.** A non-interactive `pi -p` run cannot prove
continuation, cancellation, compaction, or subagent-worker behavior — those
require a live interactive (or RPC) session that stays open across turns,
which this fixture intentionally does not launch or wait for. The following
must be run by hand and the results recorded for Task 6:

```bash
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-pi.XXXXXX")"
testdata/m0-runtime-probes/pi/run.sh "$fixture" "$log"
```

- Continuation: submit one prompt, then run `/m0-probe-follow-up` while Pi
  is idle. Expect exactly one labelled follow-up turn, reply
  `M0_PI_FOLLOW_UP_OK`, and `agent_settled` only after that follow-up turn
  (not treating the first `agent_end` as a final-stop veto):
  `testdata/m0-runtime-probes/assert-events.sh "$log" follow_up_queued agent_end agent_settled`
- Cancellation: ask the agent to call `m0_probe_wait`, then press Escape
  while it runs. Expect control returned without the probe injecting a
  follow-up, and `cancellation_probe` with `aborted: true` or `killed: true`.
- Compaction: in a separate short session, run `/compact` after at least one
  turn. Expect `session_compact` and a usable session afterward.
  `testdata/m0-runtime-probes/assert-events.sh "$log" cancellation_probe session_compact`
- Worker boundary: start Pi with both `probe.ts` and Pi's installed
  `examples/extensions/subagent/` example, delegate one read-only task to
  its worker, and record whether the worker process loads the parent probe
  automatically or requires an explicit `-e "$M0_PROBE_ROOT/pi/probe.ts"`.

If cancellation or compaction is unavailable in the installed Pi mode,
record the exact command, output, and limitation rather than claiming
support.

### Claude

```bash
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-claude.XXXXXX")"
testdata/m0-runtime-probes/claude/run.sh "$fixture" "$log"
```

`testdata/m0-runtime-probes/claude/verify-plugin.sh` covers everything that
is mechanically checkable without an interactive session: file/executable
presence, `record-hook.sh`'s env/arg validation, JSONL record shape, and
the bounded one-Stop marker (the first `Stop` event appends a
`{"decision":"block","reason":"M0_PROBE_STOP_ONCE"}` continuation request
and creates `${M0_PROBE_LOG}.stop-once`; every subsequent `Stop` in the
same run is still recorded but returns only the plain
`M0_CLAUDE_Stop` acknowledgement), and `run.sh`'s argument handling, log
truncation, stop-marker reset, and cwd/env handoff to
`claude --plugin-dir`. When a `claude` binary is on `PATH` it also runs
`claude plugin validate` against the plugin directory (schema-only, no
live session) — validated against the installed Claude Code 2.1.246 with
the manifest ([`claude/.claude-plugin/plugin.json`](claude/.claude-plugin/plugin.json))
and hook configuration ([`claude/hooks/hooks.json`](claude/hooks/hooks.json)),
unchanged from the brief.

**Manual-only observations.** A non-interactive validation pass cannot
prove that hooks actually fire during a live session, that the
`SubagentStart`/`SubagentStop` pair reaches the `m0-worker` agent, or that
the bounded `Stop` marker is observed inside the real Claude Code TUI.
Task 5 exercised this with bounded, non-interactive `claude -p` calls
against the plugin and fixture above (`run.sh` itself `exec`s an
interactive session and has no prompt argument, so it was not used
directly for these calls). Full commands and raw log excerpts are in
`.superpowers/sdd/2026-09-12-m0-runtime-integration-probes/task-5-report.md`.

- Prompt lifecycle: a `Read input.txt` prompt recorded `SessionStart`,
  `UserPromptSubmit`, and `PostToolUse` with `M0_CLAUDE_<event>`
  `additionalContext` in every case.
- Bounded Stop: the first `Stop` returned
  `{"decision":"block","reason":"M0_PROBE_STOP_ONCE"}` and created the
  marker; the run continued for exactly one more turn, then the second
  `Stop` ended it normally. Reproduced across 4 independent live runs
  with exactly 2 `Stop` records each — **not an unconditional guarantee**;
  see the schema caveat below.
- Worker boundary: `Use m0-worker and report its response.` produced
  `SubagentStart`/`SubagentStop`, and the transcript shows the worker
  itself replying `M0_CLAUDE_WORKER_OK` before the parent relayed it.
- Resume: `claude -p --resume <id> "..."` re-emitted `SessionStart` whose
  hook payload carries `"source":"resume"`, keeping the same session id.
- Compaction: `/compact` on a resumed session with prior turns fired a
  `SubagentStop` (the internal summarization subagent) followed by a new
  `SessionStart` with `"source":"compact"`; a subsequent read-only prompt
  on the same session id completed normally.
- Cancellation (approximation): SIGINT sent to a `claude -p` process mid
  `Bash` call produced `"terminal_reason":"aborted_tools"` and no
  `PostToolUse`/`Stop` record — this is a process-signal approximation of
  Escape-cancel, not proof of the interactive TUI's own key handling.

**Manual limitations — recorded, not fabricated.** Pressing Escape inside
the interactive TUI, the `/resume` picker UI, and running `/compact`
inside a live interactive session were not exercised; the evidence above
substitutes `-p`/`--resume`/SIGINT calls that approximate but do not
prove those exact interactive code paths. Separately, the installed
Claude Code build silently discards `Stop`/`SubagentStop`
`hookSpecificOutput` that omits the (undocumented-here) required
`hookEventName` field — the shipped `record-hook.sh` plain acknowledgement
omits it, and the probe's "exactly one continuation" bound depends on
that drop; a schema-corrected copy of the hook was observed to keep
re-triggering `Stop` (bounded, self-terminating, but no longer exactly
one continuation). See the task-5 report for the reproduction.

| Runtime | Action | Required log events | Manual observation |
| --- | --- | --- | --- |
| Pi | normal prompt | `session_start`, `before_agent_start`, `agent_end`, `agent_settled` | labelled policy message appears |
| Pi | `m0_probe_exec` | `process_probe`, `tool_execution_end` | output is `M0_PROCESS_OK` |
| Claude | read prompt | `SessionStart`, `UserPromptSubmit`, `PostToolUse` | injected context is present |
| Claude | worker task | `SubagentStart`, `SubagentStop` | worker returns `M0_CLAUDE_WORKER_OK` |
| Claude | first stop | two `Stop` events | exactly one continuation occurs |
