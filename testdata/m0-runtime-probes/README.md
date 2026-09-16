# M0 runtime probes

Run probes only in a temporary Git fixture created by `./create-fixture.sh`.
Set `M0_PROBE_LOG` to an absolute JSONL path before launching either runtime.
Each record has `timestamp`, `runtime`, `event`, and `cwd`; logs may contain
prompt text or paths, so keep them outside the repository and delete them when
the compatibility record has captured the result.

A passing script assertion proves that an event reached the probe. It does not
replace the manual observation noted by the matching command below.

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

Not yet implemented. Tasks 4-5 of this plan will add:

- Task 4: the Claude runtime launch command, analogous to Task 2.
- Task 5: the manual observation notes for the Claude probe run.

This README will be updated with the exact commands and observations once
those tasks land.
