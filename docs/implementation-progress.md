# agentic-pbt implementation progress

**Specification:** [`docs/specs/agentic-pbt-specification-v0.2-go.md`](specs/agentic-pbt-specification-v0.2-go.md)  
**Tracker status:** initialized  
**Last assessed:** 2026-09-12

## Status legend

- **Not started** — no implementation evidence exists.
- **In progress** — implementation has started but exit evidence is incomplete.
- **Blocked** — a required dependency or runtime assumption prevents progress.
- **Complete** — all milestone exit criteria have recorded local evidence.

## Current repository assessment

| Item | State | Evidence |
| --- | --- | --- |
| Product implementation | Not started | No `go.mod`, `cmd/`, `internal/`, `integrations/`, `skills/`, `schemas/`, or `testdata/` directories exist. |
| Specification | Available | `docs/specs/agentic-pbt-specification-v0.2-go.md` defines the proposed design. |
| System context diagram | Available | `docs/specs/system-context.md` contains the initial Mermaid C4 context diagram. |
| Version-control metadata | Unavailable here | The current directory is not a Git worktree; commit history and working-tree status cannot be assessed. |

## Milestones

### M0 — Confirm runtime integration points

**State:** Not started

**Goal:** Validate actual Pi and Claude Code lifecycle capabilities before building the full framework.

**Remaining work:**

- [ ] Record installed Pi and Claude Code versions in a compatibility matrix.
- [ ] Create a disposable Pi fixture that proves policy injection, tool/process invocation, end-of-run feedback, cancellation, queued-input behavior, and worker-extension loading.
- [ ] Create a disposable Claude fixture that proves `SessionStart`, `UserPromptSubmit`, `SubagentStart`, `PostToolUse`, `Stop`, and `SubagentStop` behavior.
- [ ] Exercise compaction, cancellation, duplicate callbacks, stop-loop protection, and separate-worktree workers where supported.
- [ ] Record what was locally exercised, what was simulated, and each known runtime limitation.

**Exit evidence:** Versioned compatibility record and reproducible probe results for both runtimes. No full integration is started until the required lifecycle assumptions are confirmed or documented as limitations.

### M1 — Go core with Python testing vertical slice

**State:** Not started

**Goal:** Deliver a standalone Go `pbt` binary that verifies Python Hypothesis/pytest property tests with durable task state and evidence.

**Remaining work:**

- [ ] Create the single Go module, `cmd/pbt` entry point, and only the required initial internal packages.
- [ ] Implement versioned `pbt.config.json` loading and validation for `python-hypothesis-pytest` packages.
- [ ] Implement task begin/resume, canonical repository/worktree identity, pre-edit baselines, and pre-existing-modification recording.
- [ ] Implement reviewable contracts, contract validation, requirement-to-test mappings, and explicit `not_applicable` decisions.
- [ ] Implement content-and-relative-path fingerprints for configured inputs, including untracked, renamed, and deleted files.
- [ ] Implement atomic `.pbt/runs/` evidence writes, `.pbt/contracts/` storage, per-worktree locking, and stale detection before/after runs and at every gate.
- [ ] Implement Hypothesis/pytest detection, machine-readable execution reporting, counterexample/replay capture, and version-provenance documentation resolution.
- [ ] Implement `doctor`, `begin`, `inspect`, `docs`, `verify`, `status`, `gate`, `resume`, and `version --json` with the specified JSON and exit-code semantics.
- [ ] Implement separate verification/workflow states and the persistent, de-duplicated three-attempt repair ledger.
- [ ] Add deterministic core tests and a Python fixture demonstrating implement → fail → repair → pass → edit → stale.
- [ ] Complete and record core-level acceptance scenarios A02–A13 and A19–A20.

**Exit evidence:** A release-style Go binary runs the Python fixture without requiring Go at runtime and has recorded successful evidence for the M1 acceptance scenarios.

### M2 — Pi package

**State:** Not started

**Goal:** Provide a thin Pi extension that uses the shared binary and policy without duplicating verification semantics.

**Remaining work:**

- [ ] Package the canonical property-testing skill and focused references.
- [ ] Implement Pi policy injection, task/status controls, verification tool, state UI, and de-duplicated bounded follow-up.
- [ ] Invoke the `pbt` binary through Pi's supported process facility and translate its JSON only.
- [ ] Verify behaviour when a different skill implements the Python fixture.
- [ ] Test cancellation, queued user input, duplicate events, worker policy loading, and separate-worktree parent verification.
- [ ] Document interactive feedback versus strict outer-wrapper enforcement and the Pi version compatibility result.

**Exit evidence:** Live Pi smoke-test results show policy availability, real property-test evidence, correct bounded correction, and clearly disclosed limitations.

### M3 — Claude Code plugin

**State:** Not started

**Goal:** Provide a thin Claude Code plugin using the same core and shared skill.

**Remaining work:**

- [ ] Package plugin-level hooks and the canonical shared skill.
- [ ] Implement short hook handlers that invoke the CLI and translate its JSON without reimplementing gate logic.
- [ ] Support session restoration, obligation refresh, scoped implementation workers, stale marking, and stop-gate feedback.
- [ ] Preserve existing hooks and honor the installed runtime's stop-loop protection.
- [ ] Verify another skill implementing the fixture and a scoped worker in a separate worktree.
- [ ] Test hook composition, cancellation, compaction/resume, duplicate callbacks, and loop termination in live sessions.
- [ ] Record tested Claude Code versions and unsupported behavior.

**Exit evidence:** Live Claude plugin smoke tests prove the shared CLI semantics, scoped worker behavior, and safe correction-loop termination.

### M4 — fast-check adapter and release candidate

**State:** Not started

**Goal:** Add the TypeScript/JavaScript vertical slice and produce portable macOS/Linux release artifacts.

**Remaining work:**

- [ ] Implement the fast-check/Vitest adapter with detection, documentation provenance, execution preparation, evidence collection, replay guidance, and declared limitations.
- [ ] Add a deterministic fast-check/Vitest fixture and complete A18.
- [ ] Demonstrate the same contract/evidence semantics in both Python and TypeScript fixtures.
- [ ] Add CI that exercises core fixtures independently of live LLM calls; retain live agent smoke tests as release validation.
- [ ] Publish setup instructions and a tested-version compatibility matrix.
- [ ] Build cgo-free release archives for macOS and Linux on amd64 and arm64, including binary, license, and required skill/reference/reporter assets.
- [ ] Publish checksums and `pbt version --json` output.
- [ ] Validate A23–A24 runtime independence and A25–A26 binary/protocol discovery behaviour.

**Exit evidence:** Release archives work in Node-only and Python-only fixture environments, with recorded CI and live-integration validation.

### M5 — Expansion

**State:** Not started

**Goal:** Extend proven shared semantics only after the initial adapters and integrations are stable.

**Remaining work:**

- [ ] Add Clojure/test.check as the next language adapter.
- [ ] Add other agent runtimes through the existing CLI/JSON protocol.
- [ ] Add dependency-aware impact analysis only when conservative all-package verification becomes insufficient.
- [ ] Add independent property review or mutation testing only if observed property-quality failures justify it.
- [ ] Consider Windows support after the macOS/Linux release path is proven.

**Exit evidence:** Each expansion has its own documented compatibility result and does not weaken the shared core's evidence, freshness, or gate semantics.

## Acceptance-scenario coverage

| Group | State | Milestone responsible |
| --- | --- | --- |
| A01 | Not started | M2/M3 |
| A02–A13 | Not started | M1 |
| A14–A15 | Not started | M2/M3 |
| A16–A17 | Not started | M1, then M4 |
| A18 | Not started | M4 |
| A19–A20 | Not started | M1 |
| A21–A22 | Not started | M1, M2, and M3 |
| A23–A26 | Not started | M4 |

## Update rules

When work changes this tracker:

1. Update the affected milestone state and checklist in the same change.
2. Add a dated entry below with commands run, result, runtime versions, and whether the result was live or simulated.
3. Mark a milestone **Complete** only after its stated exit evidence is recorded.
4. Mark a milestone **Blocked** with the specific blocker and the next action required to unblock it.

## Progress log

| Date | Milestone | Change | Verification evidence |
| --- | --- | --- | --- |
| 2026-09-12 | Tracker | Initialized from the v0.2 specification. | Repository inspection found documentation only; no product implementation artifacts. |
