# M0 Runtime Integration Probes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish recorded, live evidence for the Pi and Claude Code lifecycle assumptions that later `agentic-pbt` integrations require.

**Architecture:** Keep M0 deliberately disposable: a small fixture creator plus two thin, runtime-native probes. Each probe writes newline-delimited JSON events to a caller-provided log; it contains no `pbt` logic, state machine, or test-runner integration. A compatibility record is the only durable result and documents observed behavior and limitations by runtime version.

**Tech Stack:** POSIX `sh`, Git, Node.js 24.16.0, Pi 0.81.1 extension API, Claude Code 2.1.246 plugin hooks, Markdown.

**Spec:** `docs/specs/agentic-pbt-specification-v0.2-go.md` (sections 12, 13, 15, and M0 in section 16)

## Global Constraints

- Validate the installed Pi 0.81.1 and Claude Code 2.1.246 runtimes; do not infer compatibility from documentation or simulated hook payloads.
- Use a disposable local Git fixture for all live runs. Do not create global configuration or alter external branch protections.
- Do not start the Go core, `pbt` CLI, schemas, language adapters, or production Pi/Claude integrations in M0.
- Keep probes runtime-native and thin: Pi uses its documented extension API; Claude uses plugin-level `hooks/hooks.json`; neither probe contains shared verification decisions.
- Keep hook handlers fast. They only append one event and emit bounded probe context; they never run test suites or use a repair loop.
- Record what was executed live, what was manually observed, what was simulated, and the exact runtime versions. A simulated payload alone is never M0 completion evidence.
- Do not add an npm package, Python dependency, Go toolchain, daemon, MCP server, or hosted service. `go` is not installed in the current environment and is not needed for M0.
- The existing project directory is not a Git worktree. Commit steps apply only after this project is placed in a Git repository; do not fabricate a commit.

---

## Planned file structure

| File | Responsibility |
| --- | --- |
| `testdata/m0-runtime-probes/create-fixture.sh` | Creates an isolated, committed temporary Git repository and emits its path. |
| `testdata/m0-runtime-probes/assert-events.sh` | Fails unless a JSONL probe log contains each required event name. |
| `testdata/m0-runtime-probes/README.md` | Exact commands for Pi and Claude live probes, manual observations, and cleanup. |
| `testdata/m0-runtime-probes/pi/probe.ts` | Pi extension that records lifecycle events, injects a labelled policy message, and exposes one subprocess probe tool. |
| `testdata/m0-runtime-probes/pi/run.sh` | Starts Pi with the probe and fresh event log in the disposable fixture. |
| `testdata/m0-runtime-probes/claude/.claude-plugin/plugin.json` | Minimal Claude plugin manifest for the hook probe. |
| `testdata/m0-runtime-probes/claude/hooks/hooks.json` | Plugin-level hook registration for the M0 lifecycle events. |
| `testdata/m0-runtime-probes/claude/scripts/record-hook.sh` | Reads a hook invocation, appends an event record, and returns bounded hook feedback; `Stop` continues exactly once. |
| `testdata/m0-runtime-probes/claude/agents/m0-worker.md` | Minimal worker definition used only to trigger `SubagentStart` and `SubagentStop`. |
| `testdata/m0-runtime-probes/claude/run.sh` | Starts Claude Code with the local probe plugin and a fresh event log. |
| `docs/compatibility/runtime-integration-m0.md` | Versioned compatibility matrix, observations, commands, logs, limitations, and M0 decision. |
| `docs/implementation-progress.md` | M0 checklist/state and dated evidence entry. |

## Task 1: Create the disposable-fixture and log assertion tools

**Files:**
- Create: `testdata/m0-runtime-probes/create-fixture.sh`
- Create: `testdata/m0-runtime-probes/assert-events.sh`
- Create: `testdata/m0-runtime-probes/README.md`

**Interfaces:**
- Produces: `create-fixture.sh` writes one absolute fixture path to stdout and exits nonzero if Git setup fails.
- Produces: `assert-events.sh LOG EVENT...` exits zero only when every requested event string occurs in the JSONL log.
- Consumes: `M0_PROBE_ROOT` is the absolute path of `testdata/m0-runtime-probes`; launchers resolve all relative probe paths from it.

- [ ] **Step 1: Create the fixture creator**

Create `testdata/m0-runtime-probes/create-fixture.sh` with no dependencies beyond POSIX `sh` and Git:

```sh
#!/bin/sh
set -eu

fixture="$(mktemp -d "${TMPDIR:-/tmp}/agentic-pbt-m0.XXXXXX")"
trap 'rm -rf "$fixture"' INT TERM HUP

git -C "$fixture" init -q
git -C "$fixture" config user.email "m0-probe@example.invalid"
git -C "$fixture" config user.name "agentic-pbt M0 probe"
printf '%s\n' '# M0 disposable fixture' > "$fixture/README.md"
printf '%s\n' 'probe input' > "$fixture/input.txt"
git -C "$fixture" add README.md input.txt
git -C "$fixture" commit -qm 'chore: initialize M0 probe fixture'

printf '%s\n' "$fixture"
trap - INT TERM HUP
```

- [ ] **Step 2: Create the event assertion helper**

Create `testdata/m0-runtime-probes/assert-events.sh`:

```sh
#!/bin/sh
set -eu

log=${1:?usage: assert-events.sh LOG EVENT...}
shift
[ -f "$log" ] || { printf 'missing probe log: %s\n' "$log" >&2; exit 1; }

for event in "$@"; do
  if ! grep -Fq "\"event\":\"$event\"" "$log"; then
    printf 'missing event %s in %s\n' "$event" "$log" >&2
    exit 1
  fi
done
```

- [ ] **Step 3: Add the M0 runbook**

Create `testdata/m0-runtime-probes/README.md` with these rules:

```markdown
# M0 runtime probes

Run probes only in a temporary Git fixture created by `./create-fixture.sh`.
Set `M0_PROBE_LOG` to an absolute JSONL path before launching either runtime.
Each record has `timestamp`, `runtime`, `event`, and `cwd`; logs may contain
prompt text or paths, so keep them outside the repository and delete them when
the compatibility record has captured the result.

A passing script assertion proves that an event reached the probe. It does not
replace the manual observation noted by the matching command below.
```

Then document the exact Pi and Claude launch commands added by Tasks 2 and 4, plus the manual observations in Tasks 3 and 5.

- [ ] **Step 4: Verify fixture creation and assertions**

Run:

```bash
chmod +x testdata/m0-runtime-probes/create-fixture.sh testdata/m0-runtime-probes/assert-events.sh
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
git -C "$fixture" log --oneline -1
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-log.XXXXXX")"
printf '%s\n' '{"event":"fixture_created"}' > "$log"
testdata/m0-runtime-probes/assert-events.sh "$log" fixture_created
! testdata/m0-runtime-probes/assert-events.sh "$log" absent_event
rm -rf "$fixture" "$log"
```

Expected: Git shows `chore: initialize M0 probe fixture`; the positive assertion succeeds and the negated missing-event assertion succeeds.

- [ ] **Step 5: Commit when Git becomes available**

```bash
git add testdata/m0-runtime-probes/create-fixture.sh testdata/m0-runtime-probes/assert-events.sh testdata/m0-runtime-probes/README.md
git commit -m "test: add M0 runtime probe fixture"
```

Do not run this step in the current non-Git directory.

## Task 2: Add the Pi lifecycle and subprocess probe

**Files:**
- Create: `testdata/m0-runtime-probes/pi/probe.ts`
- Create: `testdata/m0-runtime-probes/pi/run.sh`
- Modify: `testdata/m0-runtime-probes/README.md`

**Interfaces:**
- Consumes: `M0_PROBE_LOG`, an absolute JSONL file path; absence is a startup error.
- Produces: events named `session_start`, `before_agent_start`, `tool_execution_end`, `agent_end`, `agent_settled`, `process_probe`, and `session_compact`.
- Produces: Pi custom message type `agentic-pbt-m0-policy`; it is a labelled extension message, never a user-impersonating message.

- [ ] **Step 1: Write the Pi probe extension**

Create `testdata/m0-runtime-probes/pi/probe.ts`. Use `appendFileSync` so an aborted process still leaves prior event records, `pi.exec()` for the subprocess proof, and `pi.sendMessage()` for the policy injection:

```ts
import { appendFileSync } from "node:fs";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

const logPath = process.env.M0_PROBE_LOG;
if (!logPath) throw new Error("M0_PROBE_LOG must be an absolute event-log path");

function record(event: string, cwd: string, details: Record<string, unknown> = {}) {
  appendFileSync(logPath!, `${JSON.stringify({
    timestamp: new Date().toISOString(), runtime: "pi", event, cwd, ...details,
  })}\n`);
}

export default function (pi: ExtensionAPI) {
  pi.on("session_start", (event, ctx) => record("session_start", ctx.cwd, { reason: event.reason }));
  pi.on("before_agent_start", (_event, ctx) => {
    record("before_agent_start", ctx.cwd);
    return {
      message: {
        customType: "agentic-pbt-m0-policy",
        content: "M0 probe: policy injection reached this Pi turn.",
        display: true,
      },
    };
  });
  pi.on("tool_execution_end", (event, ctx) =>
    record("tool_execution_end", ctx.cwd, { tool: event.toolName, isError: event.isError }));
  pi.on("agent_end", (_event, ctx) => record("agent_end", ctx.cwd));
  pi.on("agent_settled", (_event, ctx) => record("agent_settled", ctx.cwd, { idle: ctx.isIdle() }));
  pi.on("session_compact", (event, ctx) => record("session_compact", ctx.cwd, { reason: event.reason }));

  pi.registerTool({
    name: "m0_probe_exec",
    label: "M0 Probe Exec",
    description: "Runs a harmless child process for the M0 Pi integration probe.",
    parameters: Type.Object({}),
    async execute(_id, _params, signal, _update, ctx) {
      const result = await pi.exec(process.execPath, ["-e", "process.stdout.write('M0_PROCESS_OK')"], { signal });
      record("process_probe", ctx.cwd, { code: result.code, killed: result.killed, stdout: result.stdout });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
```

- [ ] **Step 2: Add the Pi launcher**

Create `testdata/m0-runtime-probes/pi/run.sh`:

```sh
#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture=${1:?usage: run.sh FIXTURE}
log=${2:?usage: run.sh FIXTURE LOG}

: > "$log"
export M0_PROBE_LOG="$log"
cd "$fixture"
exec pi -e "$root/pi/probe.ts"
```

Make it executable and append this exact launch sequence to the README:

```bash
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-pi.XXXXXX")"
testdata/m0-runtime-probes/pi/run.sh "$fixture" "$log"
```

- [ ] **Step 3: Verify that Pi loads the extension**

Run the launcher. In the Pi session, submit `Reply exactly: M0_PI_POLICY_OK`.

Expected manual evidence:

- the transcript displays the labelled `agentic-pbt-m0-policy` message;
- the model reply is produced after the injected policy;
- `$log` contains `session_start`, `before_agent_start`, `agent_end`, and `agent_settled`.

After Pi is idle, run:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" session_start before_agent_start agent_end agent_settled
```

- [ ] **Step 4: Verify process invocation and tool lifecycle**

In the same session, instruct the agent: `Call m0_probe_exec once and report its exact output.`

Expected manual evidence: Pi displays the custom tool execution and the agent reports `M0_PROCESS_OK`.

After it settles, run:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" process_probe tool_execution_end
rg '"event":"process_probe".*"code":0.*M0_PROCESS_OK' "$log"
```

- [ ] **Step 5: Commit when Git becomes available**

```bash
git add testdata/m0-runtime-probes/pi testdata/m0-runtime-probes/README.md
git commit -m "test: add Pi M0 lifecycle probe"
```

## Task 3: Exercise Pi continuation, cancellation, compaction, and worker boundaries

**Files:**
- Modify: `testdata/m0-runtime-probes/pi/probe.ts`
- Modify: `testdata/m0-runtime-probes/README.md`

**Interfaces:**
- Produces: `follow_up_queued` and `cancellation_probe` event records.
- Consumes: Pi's documented `agent_end`, `agent_settled`, `sendMessage(..., { deliverAs: "followUp" })`, and `ctx.signal` behavior.
- Does not produce: a stop veto, automatic repair budget, or production worker integration.

- [ ] **Step 1: Extend the probe with an explicit one-time follow-up command**

Add this command inside the extension factory in `probe.ts`:

```ts
let followUpSent = false;

pi.registerCommand("m0-probe-follow-up", {
  description: "Queue one labelled M0 follow-up after the current agent run.",
  handler: async (_args, ctx) => {
    if (followUpSent) return;
    followUpSent = true;
    record("follow_up_queued", ctx.cwd);
    pi.sendMessage({
      customType: "agentic-pbt-m0-follow-up",
      content: "M0 probe follow-up: reply exactly M0_PI_FOLLOW_UP_OK.",
      display: true,
    }, { deliverAs: "followUp", triggerTurn: true });
  },
});
```

The command intentionally uses a labelled custom extension message rather than `sendUserMessage()`.

- [ ] **Step 2: Add an abortable process command**

Add a second custom tool that waits long enough to cancel:

```ts
pi.registerTool({
  name: "m0_probe_wait",
  label: "M0 Probe Wait",
  description: "Waits for ten seconds so the M0 probe can observe cancellation.",
  parameters: Type.Object({}),
  async execute(_id, _params, signal, _update, ctx) {
    const result = await pi.exec(process.execPath, ["-e", "setTimeout(() => {}, 10000)"], { signal });
    record("cancellation_probe", ctx.cwd, { code: result.code, killed: result.killed, aborted: Boolean(signal?.aborted) });
    return { content: [{ type: "text", text: result.killed ? "M0_WAIT_CANCELLED" : "M0_WAIT_COMPLETED" }], details: {} };
  },
});
```

- [ ] **Step 3: Run the one-time continuation probe**

In a new Pi run, submit one normal prompt, then execute `/m0-probe-follow-up` while Pi is idle.

Expected manual evidence: Pi starts exactly one labelled follow-up turn, returns `M0_PI_FOLLOW_UP_OK`, and eventually emits `agent_settled`. It must not treat `agent_end` as a final-stop veto.

Run:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" follow_up_queued agent_end agent_settled
```

Retain the log path, the number of `agent_end` records, and whether `agent_settled` occurs only after the follow-up for Task 6's compatibility record.

- [ ] **Step 4: Run cancellation and compaction probes**

Ask the agent to call `m0_probe_wait`, then press Escape while it runs. In a separate short session run `/compact` after at least one turn.

Expected manual evidence:

- cancellation returns control without a follow-up being injected by the probe;
- the log contains `cancellation_probe` with `aborted: true` or `killed: true`;
- compaction produces `session_compact` and a subsequent session state remains usable.

Run:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" cancellation_probe session_compact
```

If cancellation or compaction is unavailable in the installed Pi mode, record the exact command, output, and limitation rather than claiming support.

- [ ] **Step 5: Check a separate-process worker boundary**

Use Pi's installed `examples/extensions/subagent/` only as a compatibility target. Start Pi with both `probe.ts` and the example subagent extension, then delegate one read-only task to its worker.

Record whether the worker process loads the parent probe automatically. If it does not, start a worker explicitly with `-e "$M0_PROBE_ROOT/pi/probe.ts"` and record that explicit loading is required. Do not modify or package the third-party example.

- [ ] **Step 6: Commit when Git becomes available**

```bash
git add testdata/m0-runtime-probes/pi/probe.ts testdata/m0-runtime-probes/README.md
git commit -m "test: complete Pi M0 probe"
```

## Task 4: Add the Claude Code plugin hook probe

**Files:**
- Create: `testdata/m0-runtime-probes/claude/.claude-plugin/plugin.json`
- Create: `testdata/m0-runtime-probes/claude/hooks/hooks.json`
- Create: `testdata/m0-runtime-probes/claude/scripts/record-hook.sh`
- Create: `testdata/m0-runtime-probes/claude/agents/m0-worker.md`
- Create: `testdata/m0-runtime-probes/claude/run.sh`
- Modify: `testdata/m0-runtime-probes/README.md`

**Interfaces:**
- Consumes: `M0_PROBE_LOG` and `M0_PROBE_STOP_MARKER`, both absolute paths supplied by `run.sh`.
- Produces: `SessionStart`, `UserPromptSubmit`, `PostToolUse`, `SubagentStart`, `SubagentStop`, and `Stop` JSONL event records.
- Produces: a single `Stop` continuation request; subsequent Stop events return no corrective output.

- [ ] **Step 1: Create the plugin manifest and worker**

Create `testdata/m0-runtime-probes/claude/.claude-plugin/plugin.json`:

```json
{
  "name": "agentic-pbt-m0-probe",
  "version": "0.0.0",
  "description": "Disposable runtime hook probe for agentic-pbt M0"
}
```

Create `testdata/m0-runtime-probes/claude/agents/m0-worker.md`:

```markdown
---
name: m0-worker
description: Returns a fixed response for the agentic-pbt M0 hook probe.
tools: Read
---

Reply exactly `M0_CLAUDE_WORKER_OK`. Do not modify files.
```

- [ ] **Step 2: Create the hook recorder**

Create `testdata/m0-runtime-probes/claude/scripts/record-hook.sh`:

```sh
#!/bin/sh
set -eu

event=${1:?missing hook event name}
: "${M0_PROBE_LOG:?M0_PROBE_LOG must be set}"
: "${M0_PROBE_STOP_MARKER:?M0_PROBE_STOP_MARKER must be set}"

printf '{"timestamp":"%s","runtime":"claude","event":"%s","cwd":"%s","hookEventName":"%s"}\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$event" "$PWD" "${CLAUDE_HOOK_EVENT_NAME:-}" >> "$M0_PROBE_LOG"

if [ "$event" = "Stop" ] && [ ! -e "$M0_PROBE_STOP_MARKER" ]; then
  : > "$M0_PROBE_STOP_MARKER"
  printf '%s\n' '{"decision":"block","reason":"M0_PROBE_STOP_ONCE"}'
  exit 0
fi

printf '%s\n' "{\"hookSpecificOutput\":{\"additionalContext\":\"M0_CLAUDE_${event}\"}}"
```

Make it executable. The marker prevents an unbounded stop loop while still proving that a Stop hook can request exactly one additional turn.

- [ ] **Step 3: Register the plugin hooks**

Create `testdata/m0-runtime-probes/claude/hooks/hooks.json`:

```json
{
  "description": "M0 lifecycle probe hooks",
  "hooks": {
    "SessionStart": [{ "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/scripts/record-hook.sh SessionStart", "timeout": 5 }] }],
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/scripts/record-hook.sh UserPromptSubmit", "timeout": 5 }] }],
    "PostToolUse": [{ "matcher": "Read|Write|Edit|Bash", "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/scripts/record-hook.sh PostToolUse", "timeout": 5 }] }],
    "SubagentStart": [{ "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/scripts/record-hook.sh SubagentStart", "timeout": 5 }] }],
    "SubagentStop": [{ "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/scripts/record-hook.sh SubagentStop", "timeout": 5 }] }],
    "Stop": [{ "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/scripts/record-hook.sh Stop", "timeout": 5 }] }]
  }
}
```

- [ ] **Step 4: Create the Claude launcher**

Create `testdata/m0-runtime-probes/claude/run.sh`:

```sh
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
```

Add the corresponding fixture/log/launcher command to the README.

- [ ] **Step 5: Validate the plugin before live execution**

Run:

```bash
chmod +x testdata/m0-runtime-probes/claude/scripts/record-hook.sh testdata/m0-runtime-probes/claude/run.sh
claude plugin validate testdata/m0-runtime-probes/claude
```

Expected: validation succeeds. If the installed 2.1.246 schema rejects any field, change only that probe manifest/hook configuration to the validator's accepted form, then record the exact accepted schema in the compatibility document.

- [ ] **Step 6: Commit when Git becomes available**

```bash
git add testdata/m0-runtime-probes/claude testdata/m0-runtime-probes/README.md
git commit -m "test: add Claude M0 hook probe"
```

## Task 5: Run live Claude Code lifecycle probes

**Files:**
- Modify: `testdata/m0-runtime-probes/README.md`

**Interfaces:**
- Consumes: the Task 4 plugin and the temporary Git fixture.
- Produces: live log evidence for all required hook events, one bounded Stop continuation, and documented results for compaction/cancellation and workers.

- [ ] **Step 1: Verify session policy injection and prompt lifecycle**

Launch Claude with `run.sh`. Submit: `Read input.txt, then reply exactly M0_CLAUDE_PROMPT_OK.`

Expected manual evidence: the session sees `M0_CLAUDE_SessionStart` and `M0_CLAUDE_UserPromptSubmit` context, performs a `Read`, and returns the requested response. The event log includes `SessionStart`, `UserPromptSubmit`, and `PostToolUse`.

Run:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" SessionStart UserPromptSubmit PostToolUse
```

- [ ] **Step 2: Verify exactly one Stop continuation**

Allow the first response to finish without further input.

Expected manual evidence: `M0_PROBE_STOP_ONCE` causes one continuation, then the second Stop completes normally because the marker file exists. Retain the debug-hook lines and event count for Task 6's compatibility record.

Run:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" Stop
[ "$(grep -Fc '"event":"Stop"' "$log")" -ge 2 ]
```

The test is a failure if the agent loops indefinitely or if no post-block completion occurs.

- [ ] **Step 3: Verify the scoped worker lifecycle**

In the parent session, explicitly delegate `Use m0-worker and report its response.`

Expected manual evidence: the worker responds `M0_CLAUDE_WORKER_OK`; the parent log includes `SubagentStart` and `SubagentStop`; the worker receives only the intended fixed task rather than an implementation obligation.

Run:

```bash
testdata/m0-runtime-probes/assert-events.sh "$log" SubagentStart SubagentStop
```

If the installed client uses a different command for plugin-provided agents, use that documented command, preserve the exact prompt in the compatibility record, and do not claim the events fired unless the log proves them.

- [ ] **Step 4: Verify resume, compaction, and cancellation behavior**

Run three independent short sessions:

1. Resume the first session and verify a new `SessionStart` record is written with the runtime's resume source.
2. Trigger `/compact`, then submit a short read-only prompt and verify the session remains usable.
3. Start a tool-using prompt and cancel it through the interactive client.

For each run, capture the command/session action, hook debug output, log events, and whether a Stop correction was attempted. A missing event is a documented runtime limitation, not a reason to alter shared semantics.

- [ ] **Step 5: Update the probe runbook**

Add a table to `testdata/m0-runtime-probes/README.md` that maps each live command to expected event names and the manual UI observation:

```markdown
| Runtime | Action | Required log events | Manual observation |
| --- | --- | --- | --- |
| Pi | normal prompt | `session_start`, `before_agent_start`, `agent_end`, `agent_settled` | labelled policy message appears |
| Pi | `m0_probe_exec` | `process_probe`, `tool_execution_end` | output is `M0_PROCESS_OK` |
| Claude | read prompt | `SessionStart`, `UserPromptSubmit`, `PostToolUse` | injected context is present |
| Claude | worker task | `SubagentStart`, `SubagentStop` | worker returns `M0_CLAUDE_WORKER_OK` |
| Claude | first stop | two `Stop` events | exactly one continuation occurs |
```

- [ ] **Step 6: Commit when Git becomes available**

```bash
git add testdata/m0-runtime-probes/README.md
git commit -m "test: complete Claude M0 probe"
```

## Task 6: Publish compatibility evidence and close the milestone

**Files:**
- Create: `docs/compatibility/runtime-integration-m0.md`
- Modify: `docs/implementation-progress.md`

**Interfaces:**
- Consumes: retained command transcripts, JSONL logs, runtime version output, and manual observations from Tasks 2–5.
- Produces: the sole M0 go/no-go record for M1 and later integration work.

- [ ] **Step 1: Create the compatibility record skeleton**

Create `docs/compatibility/runtime-integration-m0.md` with this exact initial structure:

```markdown
# M0 runtime integration compatibility

**Specification:** `docs/specs/agentic-pbt-specification-v0.2-go.md`  
**Probe source:** `testdata/m0-runtime-probes/`  
**Status:** In progress

## Environment

| Item | Value |
| --- | --- |
| Operating system | macOS; record `sw_vers -productVersion` at execution time |
| Node.js | 24.16.0 |
| Pi | 0.81.1 |
| Claude Code | 2.1.246 |
| Go | Not installed; not required for M0 |

## Results

| Runtime | Capability | Live result | Evidence | Limitation / decision |
| --- | --- | --- | --- | --- |
| Pi | policy injection before agent run | Not run | — | — |
| Pi | extension subprocess execution | Not run | — | — |
| Pi | end-of-run versus settled behavior | Not run | — | — |
| Pi | queued follow-up, cancellation, and compaction | Not run | — | — |
| Pi | separately launched worker policy loading | Not run | — | — |
| Claude Code | SessionStart and UserPromptSubmit context | Not run | — | — |
| Claude Code | PostToolUse stale marker signal | Not run | — | — |
| Claude Code | Stop bounded continuation | Not run | — | — |
| Claude Code | scoped worker hooks | Not run | — | — |
| Claude Code | resume, compaction, and cancellation | Not run | — | — |

## Commands and retained evidence

List each live command, its exit status, the JSONL log path or redacted excerpt, and the hook debug-log path. Do not commit generated prompt content or sensitive inputs.

## M1/M2/M3 consequences

List only observed constraints that change later implementation. Examples include the approved Pi event for final gate evaluation, required explicit worker extension loading, or a Claude hook behavior that requires a short handler.

## Decision

M0 is complete only when every row above has a live result or an explicit, reproducible runtime limitation. The decision is either `proceed`, `proceed with documented limitation`, or `blocked`.
```

- [ ] **Step 2: Record actual versions and command outputs**

Run and paste the complete outputs or concise excerpts with commands into the record:

```bash
sw_vers -productVersion
node --version
pi --version
claude --version
claude plugin validate testdata/m0-runtime-probes/claude
```

- [ ] **Step 3: Update the progress tracker truthfully**

In `docs/implementation-progress.md`, change M0 to `In progress` when Task 1 begins. After all live runs, replace each M0 checklist item with its observed result, add the compatibility-record path, and append a dated progress-log row with runtime versions and the `proceed`/`blocked` decision.

Do not mark M0 complete merely because the probes compile or because simulated inputs succeeded.

- [ ] **Step 4: Review M0 against the specification**

Verify the record covers these mandatory M0 outcomes:

```text
Pi: policy injection, feedback/follow-up, process invocation, cancellation,
queued input, compaction, and worker-policy limitation.

Claude Code: plugin-level hooks, SessionStart, UserPromptSubmit, PostToolUse,
Stop, SubagentStart, SubagentStop, compaction/resume, cancellation, hook
composition/loop termination, and worker scope.
```

Expected: every item points to live evidence or an explicit limitation; nothing claims that the interactive hook is a tamper-proof enforcement boundary.

- [ ] **Step 5: Commit when Git becomes available**

```bash
git add docs/compatibility/runtime-integration-m0.md docs/implementation-progress.md
git commit -m "docs: record M0 runtime compatibility"
```

## Plan self-review

- **Spec coverage:** Tasks 1–3 cover Pi M0 requirements; Tasks 4–5 cover Claude M0 requirements; Task 6 records versioned live evidence and updates the milestone state. No Go core, testing adapter, or release work is pulled forward from M1/M4.
- **Scope:** Pi and Claude probes share one M0 exit gate and one compatibility record, so keeping them in this plan avoids duplicate fixture and evidence tooling. Their implementations remain isolated by runtime.
- **Dependency check:** The plan needs only installed Pi, Claude Code, Node.js, Git, and POSIX shell. It does not depend on Go, Python, Node package installation, an MCP server, or a hosted service.
- **Known initial constraint:** The current directory is not a Git repository. Preserve the commit commands for when the repository is initialized; do not invent commit evidence.
