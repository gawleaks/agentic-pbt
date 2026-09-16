# M0 runtime integration compatibility

**Specification:** `docs/specs/agentic-pbt-specification-v0.2-go.md`
**Probe source:** `testdata/m0-runtime-probes/`
**Status:** In progress

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
| Pi | policy injection before agent run | Live | task-2-report.md: `pi -p "Reply exactly: M0_PI_POLICY_OK"` → model replied exactly that string; log has `session_start`, `before_agent_start`, `agent_end`, `agent_settled` in order | Confirmed only via non-interactive `-p`; no interactive TTY pass has visually confirmed the policy message renders as a labelled (non-user) bubble in the Pi TUI. Still open manual evidence. |
| Pi | extension subprocess execution | Live | task-2-report.md: `m0_probe_exec` tool call → log `process_probe` with `"code":0`, `"stdout":"M0_PROCESS_OK"`, plus `tool_execution_end` | None observed. |
| Pi | end-of-run versus settled behavior | Partial | task-2-report.md: single-turn live run shows `agent_end` then `agent_settled`, no intervening follow-up | Only the no-continuation case has live evidence. Whether `agent_settled` correctly waits for a queued follow-up turn instead of firing on the first `agent_end` requires the continuation probe below and has not been run. |
| Pi | queued follow-up, cancellation, and compaction | Not run | task-3-report.md: `m0-probe-follow-up` command and `m0_probe_wait` tool implemented and exercised only against a mocked `pi` ExtensionAPI (`contract-test.mjs`), no real interactive process | Each requires a live interactive Pi session (submit a prompt, then while idle run `/m0-probe-follow-up`; ask the agent to call `m0_probe_wait` and press Escape; run `/compact` after ≥1 turn). None of the three has been exercised against a live process. Exact commands are in `testdata/m0-runtime-probes/README.md`. |
| Pi | separately launched worker policy loading | Not run | task-3-report.md | Requires starting Pi with both `probe.ts` and Pi's installed `examples/extensions/subagent/` example and delegating a read-only task, then observing whether the worker process auto-loads the parent probe or needs an explicit `-e`. Not attempted in any task. |
| Claude Code | SessionStart and UserPromptSubmit context | Live | task-5-report.md: `claude -p "Read input.txt, then reply exactly M0_CLAUDE_PROMPT_OK."` → log has `SessionStart`, `UserPromptSubmit`, `PostToolUse`, two `Stop` records | None observed for the base event firing; see the resume/compact row for a related caveat. |
| Claude Code | PostToolUse stale marker signal | Live (hook firing only) | task-5-report.md: `PostToolUse` recorded with `M0_CLAUDE_PostToolUse` `additionalContext` in the same run above | The M0 probe does not implement content-staleness detection — that is an M1+ `pbt` concept, not part of this fixture. This row only confirms the `PostToolUse` hook itself fires and can attach `additionalContext`; no stale-marker signal exists yet to observe. |
| Claude Code | Stop bounded continuation | Live, with caveat — **not an unconditional guarantee** | task-5-report.md: 4 independent live runs, each producing exactly 2 `Stop` records (first returns `{"decision":"block","reason":"M0_PROBE_STOP_ONCE"}`, second returns the plain acknowledgement) | The installed Claude Code 2.1.246 build silently discards `hookSpecificOutput` payloads that omit the (undocumented) required `hookEventName` field; the shipped `record-hook.sh` plain acknowledgement is one such payload, and that drop is what keeps the observed run bounded at exactly one continuation. A schema-corrected copy of the hook (tested only in a throwaway `/tmp` copy, never committed) re-triggered `Stop` 9 times before settling. The "exactly 2 Stop records" result is contingent on this runtime build's current hook-validation behavior, not a property `record-hook.sh` enforces on its own. |
| Claude Code | scoped worker hooks | Live | task-5-report.md: `Use m0-worker and report its response.` → log has `SubagentStart`/`SubagentStop`; `subagent_stats` shows `"spawned":1`, `"completed":1`, `"failed":0`; transcript shows the worker itself replied `M0_CLAUDE_WORKER_OK` | None observed. Claude Code has no separate "worker launch command" distinct from the `Task` tool's subagent dispatch. |
| Claude Code | resume, compaction, and cancellation | Partial | task-5-report.md: `--resume <id>` produced a new `SessionStart` on the same session id; `/compact` on a resumed session produced an internal `SubagentStop` then a new `SessionStart`, and the session remained usable afterward; SIGINT sent mid-tool-call produced `"terminal_reason":"aborted_tools"` with no `PostToolUse`/`Stop` record | Three open gaps: (1) the shipped `record-hook.sh` does not read hook stdin, so it cannot itself record the `"source":"resume"`/`"source":"compact"` field — that was confirmed only via a throwaway `/tmp` debug copy of the hook, never the committed script; (2) real interactive Escape-cancel inside the TUI was not run — the SIGINT approximation is a different mechanism (process signal, not a UI keybinding) and may not exercise the same code path; (3) the interactive `/resume` picker UI and an interactive `/compact` slash command inside a live TUI turn were not run — `--resume <id>` and `-p --resume <id> "/compact"` were used instead. |

## Commands and retained evidence

Self-checks re-run in this session (2026-09-16), each deterministic and non-interactive, no network/model access required except where noted:

```
$ ./testdata/m0-runtime-probes/verify-fixture-tools.sh
PASS: create-fixture.sh and assert-events.sh behave as required
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

The live interactive/bounded-`-p` runs behind the "Live"/"Partial" rows above (Pi Step 3/4 policy and process probes; Claude Code prompt lifecycle, bounded Stop, worker, resume, compaction, and SIGINT-cancellation) were executed in Tasks 2 and 5, not re-run here to avoid inventing new results or duplicating live model calls. Their exact commands, exit statuses, and raw JSONL excerpts are retained in `task-2-report.md` and `task-5-report.md` in this plan directory. Per `testdata/m0-runtime-probes/README.md`'s data-retention note, the generated fixtures, logs, and one throwaway `/tmp` debug-plugin copy from those tasks were deleted after their output was captured into the reports; no prompt content, credentials, or other sensitive material is retained or reproduced here beyond the fixed marker strings (`M0_PI_POLICY_OK`, `M0_PROCESS_OK`, `M0_CLAUDE_PROMPT_OK`, `M0_CLAUDE_WORKER_OK`, etc.) the probes themselves are designed to emit.

## M1/M2/M3 consequences

- M1 (Go core, Python testing vertical slice) has no Pi/Claude runtime dependency; nothing in this record blocks it.
- M2 (Pi package) must not assume queued follow-up, cancellation, compaction, or worker-extension policy loading behave as sketched in the spec — none of the four has live evidence. Run Task 3's deferred manual Pi session (see `testdata/m0-runtime-probes/README.md`) before implementing the corresponding M2 behavior.
- M3 (Claude Code plugin) must not treat "exactly one Stop continuation" as a guarantee `record-hook.sh`'s plain acknowledgement provides on its own — it is currently a side effect of this Claude Code build rejecting a malformed `hookSpecificOutput` (missing `hookEventName`). M3's real stop-loop protection should use an explicit, self-enforced bound (e.g., a counter or marker file, as this probe already does for the *first* Stop) rather than relying on upstream schema-validation behavior that could change.
- M3 should design scoped worker hooks around Claude Code's `Task`-tool subagent dispatch — there is no separate "worker launch command" to substitute.
- M3 should plan for stdin-based hook-input capture (e.g., reading the `source` field for resume/compact detection) since the current probe script does not read hook stdin and cannot self-report why a `SessionStart` occurred.
- Per Task 4's deferred note: `claude/verify-plugin.sh`'s `claude plugin validate` check runs unconditionally whenever `claude` is on `PATH` (unlike the Pi live check, which needs an explicit env var). Acceptable for M0 since it is schema-only and offline, but M2/M3 CI should decide deliberately whether plugin-schema validation runs on every CI machine or stays gated.

## Decision

M0 is not complete: five of the ten rows above (Pi continuation/cancellation/compaction/worker-boundary, and the interactive-only parts of Claude Code's Escape-cancel/resume-picker/`/compact`) have no live evidence, only non-interactive approximations or mocked unit tests. Per the exit criteria, M0 stays **In progress** until those manual sessions are run and recorded.

**Decision: proceed with documented limitation.** M1 (Go core), which has no Pi/Claude runtime dependency, may proceed without waiting on the remaining manual evidence. M2 and M3 must not begin implementation that assumes Pi continuation/cancellation/compaction/worker-policy behavior, and must not describe the Claude Stop-hook 2-count as a guaranteed bound, until the outstanding rows in this record carry live evidence.
