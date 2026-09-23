# M0 runtime integration compatibility

**Specification:** `docs/specs/agentic-pbt-specification-v0.2-go.md`
**Probe source:** `testdata/m0-runtime-probes/`
**Status:** In progress

> **Evidence retention.** The per-task reports in
> `.superpowers/sdd/2026-09-12-m0-runtime-integration-probes/` are **not tracked in
> Git** (`.superpowers/` is in `.gitignore`) and the probe fixtures and JSONL logs
> they were taken from were deleted after capture. Every claim below therefore
> inlines its own command and log excerpt in [Inlined evidence](#inlined-evidence);
> the task-report citations are provenance pointers only, not the durable record.

## Environment

| Item | Value |
| --- | --- |
| Operating system | macOS 26.6.2 (`sw_vers -productVersion`, recorded 2026-09-16) |
| Node.js | v24.16.0 |
| Pi | 0.81.1 |
| Claude Code | 2.1.246 (Claude Code) |
| Go | Not installed; not required for M0 |

## Results

| Runtime | Capability | Live result | Evidence | Limitation / decision |
| --- | --- | --- | --- | --- |
| Pi | `before_agent_start` policy hook fires, returns a message, and reaches the model | Live | [E1](#e1-pi-lifecycle-events) records hook firing; [E1a](#e1a-pi-policy-delivery) records the interactive prompt and exact reply | The initial request to quote hidden policy text was correctly refused, so it was not evidence. The current harmless conditional instruction is discriminating: the prompt `M0 policy delivery check` does not contain `M0_PI_POLICY_DELIVERED`, but the model replied with it exactly. The labelled non-user bubble still has no separately recorded TUI observation. |
| Pi | extension subprocess execution | Live | [E2](#e2-pi-subprocess-probe): log has `process_probe` with `"code":0`, `"stdout":"M0_PROCESS_OK"`, plus `tool_execution_end` | Discriminating: `M0_PROCESS_OK` is written by the child process and captured by `pi.exec()` inside the extension, so the log line cannot be produced by the model alone. The model's own echo of the string is *not* the evidence. |
| Pi | end-of-run versus settled behavior | Partial | [E1](#e1-pi-lifecycle-events): single-turn live run shows `agent_end` then `agent_settled`, no intervening follow-up | Only the no-continuation case has live evidence. Whether `agent_settled` correctly waits for a queued follow-up turn instead of firing on the first `agent_end` requires the continuation probe below and has not been run. |
| Pi | queued follow-up, cancellation, and compaction | Live | Interactive Pi validation on 2026-09-16: `follow_up_queued`/`agent_settled` assertion passed; `cancellation_probe` recorded `killed:true`, `aborted:true`; after the compaction-lifecycle probe update, `session_before_compact` and `session_compact` assertion passed. | The run proves these three behaviors in the M0 fixture only. Worker loading, separate-worktree behavior, and duplicate-event handling remain unproven. |
| Pi | separately launched worker policy loading | Unsupported in current setup | Local inspection found no Pi extensions or agent definitions. Pi's bundled subagent example hard-codes `model: claude-sonnet-4-5`, which requires unavailable/unapproved Anthropic authentication. | No permitted worker launcher/model is configured, so no child process can be launched for this probe. Revisit only after a permitted Pi worker model is configured. |
| Pi | worker launched in a separate worktree | Unsupported in current setup | Same worker-launcher limitation as the preceding row. | A separate-worktree worker test cannot run until a permitted worker launcher/model exists. Parent verification is therefore not assumed. |
| Claude Code | `SessionStart` / `UserPromptSubmit` hooks fire | Live | [E3](#e3-claude-prompt-lifecycle): log has `SessionStart`, `UserPromptSubmit`, `PostToolUse`, and two `Stop` records; the 2026-09-17 interactive run also confirmed context delivery. | The original `M0_CLAUDE_PROMPT_OK` response was non-discriminating. Context delivery became live evidence only after the probe emitted required `hookSpecificOutput.hookEventName`. |
| Claude Code | `PostToolUse` hook fires | Live — hook firing only | [E3](#e3-claude-prompt-lifecycle): `PostToolUse` recorded in the same run | The M0 probe does not implement content-staleness detection — that is an M1+ `pbt` concept, not part of this fixture. This row confirms only that the `PostToolUse` hook itself fires and is recorded; no stale-marker signal exists yet to observe. |
| Claude Code | injected `additionalContext` reaches the session | Live | Interactive run on 2026-09-17: Claude listed `M0_CLAUDE_SessionStart`, `M0_CLAUDE_UserPromptSubmit`, and `M0_CLAUDE_PostToolUse` verbatim after reading `input.txt`; the associated log recorded `SessionStart`, `UserPromptSubmit`, and `PostToolUse`. | Claude Code 2.1.246 requires `hookEventName` inside every `hookSpecificOutput`. The probe now emits the current event name; later `Stop` and `SubagentStop` events omit `additionalContext` to avoid triggering another turn. |
| Claude Code | Stop bounded continuation | Live | The 2026-09-17 interactive run recorded two `Stop` events, with the first returning `{"decision":"block","reason":"M0_PROBE_STOP_ONCE"}` and the second ending normally. | The corrected probe enforces the bound with its marker: later `Stop`/`SubagentStop` responses emit only `hookEventName`, not `additionalContext`. `stop_hook_active` remains untested. |
| Claude Code | `stop_hook_active` reentrancy flag | Not run | None | `record-hook.sh` does not read hook stdin, so the runtime-supplied `stop_hook_active` field was never observed, and no run tested whether honoring it terminates a continuation loop. Required before M3 can rely on it for stop-loop protection. |
| Claude Code | composition with pre-existing user/project hooks | Not run | None | All probe runs used `--plugin-dir` in isolation; the one run that touched the operator's own global hooks used `--setting-sources project` to *exclude* them. Whether plugin hooks compose with (rather than replace or conflict with) existing user/project hooks for the same event is untested. |
| Claude Code | scoped worker hooks | Live | [E6](#e6-claude-scoped-worker): log has `SubagentStart`/`SubagentStop`; `subagent_stats` shows `"spawned":1`, `"completed":1`, `"failed":0`; transcript shows the worker itself replied `M0_CLAUDE_WORKER_OK` | Claude Code has no separate "worker launch command" distinct from the `Task` tool's subagent dispatch. The worker's reply string is fixed in the agent definition, so it is discriminating for *which* agent answered, not for context injection. |
| Claude Code | scoped worker in a separate worktree | Not run | None | Spec M3 requires a scoped worker to run against a separate worktree. Every probe run used one fixture directory as both parent and worker cwd; no cross-worktree run was attempted. |
| Claude Code | resume, compaction, and cancellation | Partial | [E7](#e7-claude-resume-compaction-cancellation): `--resume <id>` produced a new `SessionStart` on the same session id; `/compact` produced an internal `SubagentStop` then a new `SessionStart`, session still usable; SIGINT mid-tool-call produced `"terminal_reason":"aborted_tools"` with no `PostToolUse`/`Stop` | Three open gaps: (1) `record-hook.sh` does not read hook stdin, so it cannot record the `"source":"resume"`/`"source":"compact"` field — that was seen only via a throwaway `/tmp` debug copy of the hook, never the committed script; (2) interactive Escape-cancel inside the TUI was not run — the SIGINT approximation is a different mechanism (process signal, not a UI keybinding); (3) the interactive `/resume` picker UI and an interactive `/compact` inside a live TUI turn were not run — `--resume <id>` and `-p --resume <id> "/compact"` were used instead. |

## Inlined evidence

Marker strings (`M0_PI_POLICY_DELIVERED`, `M0_PROCESS_OK`, `M0_CLAUDE_PROMPT_OK`,
`M0_CLAUDE_WORKER_OK`, `M0_PROBE_STOP_ONCE`) are fixed constants the probes emit or
the prompts dictate; no prompt content, credentials, session ids, or file contents
beyond these are reproduced here.

Common setup for every Claude excerpt below:

```bash
root="$(pwd)/testdata/m0-runtime-probes/claude"
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-claude.XXXXXX")"
: > "$log"; rm -f "${log}.stop-once"; cd "$fixture"
```

### E1 Pi lifecycle events

```bash
$ pi -e testdata/m0-runtime-probes/pi/probe.ts --no-session -p "Reply exactly: M0_PI_POLICY_OK"
$ testdata/m0-runtime-probes/assert-events.sh "$log" \
    session_start before_agent_start agent_end agent_settled   # PASS
```

Log events in order: `session_start`, `before_agent_start`, `agent_end`,
`agent_settled`. The historical model reply `M0_PI_POLICY_OK` was dictated by the
prompt, so it is non-discriminating.

### E1a Pi policy delivery

In a fresh interactive Pi session using the current probe, the operator submitted:

```text
M0 policy delivery check
```

Pi replied exactly:

```text
M0_PI_POLICY_DELIVERED
```

The reply token occurs only in the injected conditional policy instruction, not in
the user prompt. This is live evidence that the policy reached the model.

### E2 Pi subprocess probe

```bash
$ pi -e testdata/m0-runtime-probes/pi/probe.ts --no-session \
    -p "Call m0_probe_exec once and report its exact output."
$ testdata/m0-runtime-probes/assert-events.sh "$log" process_probe tool_execution_end  # PASS
$ rg '"event":"process_probe".*"code":0.*M0_PROCESS_OK' "$log"                         # match
```

### E3 Claude prompt lifecycle

```bash
$ M0_PROBE_LOG="$log" M0_PROBE_STOP_MARKER="${log}.stop-once" timeout 90 claude -p \
    "Read input.txt, then reply exactly M0_CLAUDE_PROMPT_OK." \
    --plugin-dir "$root" --permission-mode bypassPermissions \
    --output-format json --no-session-persistence
```

```
{"timestamp":"2026-09-16T09:30:26Z","event":"SessionStart", ...}
{"timestamp":"2026-09-16T09:30:27Z","event":"UserPromptSubmit", ...}
{"timestamp":"2026-09-16T09:30:31Z","event":"PostToolUse", ...}
{"timestamp":"2026-09-16T09:30:32Z","event":"Stop", ...}
{"timestamp":"2026-09-16T09:30:35Z","event":"Stop", ...}
```

```bash
$ testdata/m0-runtime-probes/assert-events.sh "$log" SessionStart UserPromptSubmit PostToolUse  # PASS
```

### E4 Claude bounded Stop

Same call as E3, re-run with `--output-format stream-json --include-hook-events`:
`Stop` #1 returned `{"decision":"block","reason":"M0_PROBE_STOP_ONCE"}`, the agent
produced one more turn, `Stop` #2 returned the plain acknowledgement and did not
block. Exactly 2 `Stop` records in each of 4 independent live runs.

```bash
$ grep -Fc '"event":"Stop"' "$log"   # 2
$ [ -e "${log}.stop-once" ]          # marker present
```

### E5 Claude hookSpecificOutput schema rejection

Observed on stderr under `--include-hook-events` (non-fatal):

```
Hook JSON output validation failed — hookSpecificOutput is missing required field "hookEventName"
```

This was emitted for every `hookSpecificOutput` payload in the original probe — i.e.
`SessionStart`, `UserPromptSubmit`, `PostToolUse`, `SubagentStart`, `SubagentStop`,
and each `Stop` after the first. The probe was then corrected to include the current
event as `hookSpecificOutput.hookEventName`; on 2026-09-17, an interactive run
confirmed context delivery for `SessionStart`, `UserPromptSubmit`, and `PostToolUse`.
Later `Stop`/`SubagentStop` responses now contain only `hookEventName`, avoiding the
previous repeated-feedback behavior.

### E6 Claude scoped worker

```bash
$ M0_PROBE_LOG="$log" M0_PROBE_STOP_MARKER="${log}.stop-once" timeout 90 claude -p \
    "Use m0-worker and report its response." \
    --plugin-dir "$root" --permission-mode bypassPermissions --output-format json
```

`subagent_stats`: `"spawned":1`, `"by_type":{"agentic-pbt-m0-probe:m0-worker":1}`,
`"completed":1`, `"failed":0`. Log order: `SessionStart`, `UserPromptSubmit`,
`SubagentStart`, `SubagentStop`, `Stop`, `Stop`. Transcript (`--forward-subagent-text`)
shows the subagent turn itself replying `M0_CLAUDE_WORKER_OK`, then the parent
relaying it.

```bash
$ testdata/m0-runtime-probes/assert-events.sh "$log" SubagentStart SubagentStop  # PASS
```

### E7 Claude resume, compaction, cancellation

```bash
# resume: a persisted session, then
$ claude -p --resume <session-id> "Reply exactly M0_CLAUDE_RESUME_FOLLOWUP_OK." \
    --plugin-dir "$root" --permission-mode bypassPermissions \
    --output-format stream-json --include-hook-events --verbose
# → new SessionStart recorded; system/init confirms the session id is unchanged.

# compaction: on that same persisted session
$ claude -p --resume <session-id> "/compact" --plugin-dir "$root" \
    --permission-mode bypassPermissions --output-format json
# → SessionStart, then SubagentStop (internal summarization subagent), then a new
#   SessionStart. A follow-up prompt on the same id completed normally
#   (SessionStart, UserPromptSubmit, PostToolUse, 2x Stop).
# On a fresh session: "result":"Not enough messages to compact."

# cancellation (approximation only — process signal, not the TUI keybinding)
$ claude -p "Run: sleep 20 && echo done. ..." ... & cpid=$!; sleep 5; kill -INT "$cpid"
# → "is_error":true, "terminal_reason":"aborted_tools", "stop_reason":"tool_use";
#   log had only SessionStart and UserPromptSubmit — no PostToolUse, no Stop.
```

The `"source":"resume"` / `"source":"compact"` values were visible only through a
throwaway `/tmp` debug copy of `record-hook.sh` that `cat`s hook stdin; the shipped
script never reads stdin and cannot report them.

## Commands and retained evidence

Self-checks re-run in this session (2026-09-16), each deterministic and non-interactive, no network/model access required except where noted:

```
$ ./testdata/m0-runtime-probes/verify-fixture-tools.sh
PASS: create-fixture.sh and assert-events.sh behave as required
exit=0

$ ./testdata/m0-runtime-probes/verify-docs.sh
PASS: M0 documentation self-check
exit=0

$ ./testdata/m0-runtime-probes/pi/verify-probe.sh
PASS: run.sh contract (usage, truncation, cwd, env, launch args)
PASS: probe.ts follow-up and cancellation contract checks
SKIP: live pi run (set M0_PROBE_PI_LIVE_CHECK=1 to enable)
exit=0

$ ./testdata/m0-runtime-probes/pi/test-cleanup-trap.sh
PASS: verify-probe.sh live-check EXIT trap cleans up all temp paths
exit=0

$ ./testdata/m0-runtime-probes/claude/verify-plugin.sh
PASS: record-hook.sh contract (env/arg validation, JSONL shape, bounded one-Stop marker)
PASS: run.sh contract (usage, truncation, marker reset, cwd, env, launch args)
PASS: claude plugin validate
exit=0
```

Version commands, re-run in this session:

```
$ sw_vers -productVersion
26.6.2

$ node --version
v24.16.0

$ pi --version
0.81.1

$ claude --version
2.1.246 (Claude Code)

$ claude plugin validate testdata/m0-runtime-probes/claude
⚠ Found 1 warning: author: No author information provided.
✔ Validation passed with warnings
```

The live runs behind the "Live"/"Partial" rows were executed in Tasks 2 and 5 and
are not re-run here, to avoid inventing new results or duplicating live model calls.
Their commands and log excerpts are inlined above; the generated fixtures, logs, and
the one throwaway `/tmp` debug-plugin copy were deleted after capture per the
data-retention note in `testdata/m0-runtime-probes/README.md`.

## Unproven M0 points

Nothing in this list has live evidence. M2/M3 must not assume any of it.

1. Pi policy-message **delivery** to the model (only the handler firing is proven).
2. Pi labelled-policy-bubble rendering in the interactive TUI.
3. Pi queued follow-up (`/m0-probe-follow-up`) and `agent_settled` waiting for it.
4. Pi cancellation (`m0_probe_wait` + Escape).
5. Pi compaction (`/compact`).
6. Pi separately launched worker policy loading.
7. Pi worker in a separate worktree.
8. Claude `stop_hook_active` reentrancy flag.
9. Claude hook composition with pre-existing user/project hooks.
10. Claude interactive Escape-cancel inside the TUI.
11. Claude interactive `/resume` picker UI.
12. Claude interactive `/compact` inside a live TUI turn.
13. Claude scoped worker in a separate worktree.
14. Claude resume/compact `source` field via the *shipped* hook (seen only through a
    throwaway `/tmp` debug copy).

## M1/M2/M3 consequences

- M1 (Go core, Python testing vertical slice) has no Pi/Claude runtime dependency; nothing in this record blocks it.
- M2 (Pi package) must not assume policy-message delivery, queued follow-up, cancellation, compaction, worker-extension policy loading, or separate-worktree workers behave as sketched in the spec — none has live evidence. Run the deferred manual Pi session (see `testdata/m0-runtime-probes/README.md`) first. Any future policy-injection probe must use a marker carried **in the injected text itself**, not one the prompt dictates, or the result will not discriminate.
- M3 (Claude Code plugin) must include `hookEventName` inside **every** `hookSpecificOutput` object it emits, for every hook event; on Claude Code 2.1.246 the runtime silently discards any payload that omits it. M0 now proves context delivery for `SessionStart`, `UserPromptSubmit`, and `PostToolUse` when that field is present.
- M3 should use an explicit self-enforced Stop bound (the M0 probe uses a marker file for the first `Stop` and emits no later `additionalContext`) and should read `stop_hook_active` from hook stdin, which is itself untested here.
- M3 should design scoped worker hooks around Claude Code's `Task`-tool subagent dispatch — there is no separate "worker launch command" to substitute — and must still prove the separate-worktree and hook-composition cases, both unprobed.
- M3 should plan for stdin-based hook-input capture (the `source` field for resume/compact detection, and `stop_hook_active`) since the current probe script does not read hook stdin.
- Per Task 4's deferred note: `claude/verify-plugin.sh`'s `claude plugin validate` check runs unconditionally whenever `claude` is on `PATH` (unlike the Pi live check, which needs an explicit env var). Acceptable for M0 since it is schema-only and offline, but M2/M3 CI should decide deliberately whether plugin-schema validation runs on every CI machine or stays gated.

## Decision

M0 is not complete. Of the 15 capability rows above, 7 are Live, 3 are Partial or
hook-firing-only, and 5 are Not run. The [Unproven M0 points](#unproven-m0-points)
list enumerates 14 open items. Per the exit criteria, M0 stays **In progress** until those manual sessions
are run and recorded.

**Decision: proceed with documented limitation.** M1 (Go core), which has no Pi/Claude
runtime dependency, may proceed without waiting on the remaining manual evidence. M2
and M3 must not begin implementation that assumes any item in the unproven list, and
must not describe the Claude Stop-hook 2-count as a guaranteed bound, until those rows
carry live evidence.
