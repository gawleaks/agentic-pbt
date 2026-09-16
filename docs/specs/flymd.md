# Agentic Property-Based Testing Harness

Version: 0.2 — Go core and CLI architecture  
Date: 2026-09-12  
Status: proposed design; runtime integrations require local validation  
Working project name: `agentic-pbt`

## 1. Purpose

Build a portable skill and supporting harness that makes property-based testing (PBT) part of feature implementation, even when another skill or delegated agent performs the implementation.

The system must support two responsibilities:

1. Guide agents to write code whose behavior can be exercised through generated inputs and operation sequences.
2. Require agents to implement or update meaningful property tests using the appropriate language library, execute them, and report current verification evidence.

Start with Pi and Claude Code. Keep the shared design reusable for later Codex, Copilot, and OpenCode integrations without assuming those runtimes offer equivalent hooks.

“MUST” denotes a requirement. “SHOULD” allows an explained exception. Commands, schemas, and paths below are proposed project interfaces, not existing agent APIs.

## 2. Scope and limits

### Initial scope

- Local Git repositories on macOS and Linux.
- An independent Go core and CLI, distributed as native `pbt` executables.
- A portable `SKILL.md` and focused language references.
- Python with Hypothesis and pytest as the first complete vertical slice.
- A Pi extension and a Claude Code plugin using the same core.
- TypeScript/JavaScript with fast-check and Vitest as the second vertical slice.
- Explicit project configuration for package roots, test commands, and input files.
- Single active verification writer per worktree, with an integrated parent check after delegation.fLyMd-mAkEr

### Subsequent scope

- Clojure/test.check, other runners and languages.
- Automated package-graph discovery and more precise change impact analysis.
- Additional agent adapters and Windows support.
- Optional independent property review or mutation testing.

### Non-goals

- Proving program correctness or claiming generated tests cover every input.
- Replacing example tests, integration tests, static analysis, or human review.
- Enforcing one application architecture or requiring every function to be pure.
- Replacing an existing implementation skill or orchestrator.
- Building a new property-testing library or a universal shrinking algorithm.
- Installing a mandatory MCP server or requiring a hosted service.
- Treating local evidence as tamper-proof against an agent with filesystem access.

The harness MUST distinguish a workflow guardrail from an external enforcement boundary. A required CI job can independently rerun tests; an interactive agent can still be interrupted or its extension disabled.

## 3. Architecture and ownership

| Component | Owns | Does not own |
| --- | --- | --- |
| Shared skill | Property selection, testable design, generator quality, failure investigation | Agent lifecycle hooks |
| Language adapter | Dependency detection, documentation resolution, runner integration, replay guidance | Business requirements |
| Core and CLI | Configuration, task identity, fingerprints, execution evidence, retry decisions | Inventing assertions or editing production code |
| Pi extension | Policy injection, tools, follow-ups, visible status | Independent verification semantics |
| Claude plugin | Policy injection, hooks, feedback, visible status | A separate testing policy |
| Implementing agent | Contract, implementation, tests, semantic assessment, repairs | Self-declaring a successful execution record |
| CI | Fresh verification in a controlled checkout | Trusting a local “passed” flag |

The core MUST remain independent of Pi and Claude SDK types. Native event payloads are translated by adapters into shared requests. The same inputs MUST produce the same verification and retry decisions in either runtime.

### Execution boundary

`pbt` is an independent local executable. It MUST work without a running agent and without a separately installed Go, Python or Node.js runtime for the core itself. Build release executables without cgo or external shared-library dependencies introduced by the application; verify that property on the supported release platforms. Target projects still supply their own runtimes, test runners and PBT dependencies.

There are three callers of the same core:

| Caller | Invocation |
| --- | --- |
| Implementing agent | Uses its existing shell/terminal tool to execute `pbt` commands |
| Native agent adapter | Launches `pbt` as a subprocess and translates JSON decisions into runtime-specific feedback |
| Developer or CI | Calls the same CLI directly, with no agent dependency |

The Go process launches each configured test runner in the target project's environment. A TypeScript/fast-check project MUST NOT need Python merely to run the harness. Likewise, the core MUST NOT require Node.js when testing a Python project. A selected agent may have its own runtime requirements, independent of the core.

Keep two adapter categories distinct: agent adapters translate lifecycle events; library/runner adapters prepare tests and interpret results. Neither category independently reimplements state, freshness or retry decisions. No daemon, HTTP service, embedded LLM or MCP server is required.

## 4. Repository layout

| Path | Contents |
| --- | --- |
| `go.mod`, `go.sum` | Single Go module and dependency checksums |
| `cmd/pbt/` | Small CLI entry point |
| `internal/core/` | State, contracts, gate and retry decisions |
| `internal/cli/` | Command parsing, exit codes and JSON protocol |
| `internal/config/` | Configuration loading and validation |
| `internal/workspace/` | Task baselines, scope and fingerprints |
| `internal/runner/` | Subprocess execution, cancellation and output capture |
| `internal/evidence/` | Atomic persistence and execution reports |
| `internal/adapters/` | Hypothesis/pytest and fast-check/Vitest runner adapters |
| `internal/docs/` | Documentation resolution and caching |
| `integrations/pi/` | Pi extension and package metadata |
| `integrations/claude/` | Claude plugin manifest and plugin-level hooks |
| `skills/property-based-testing/SKILL.md` | Shared workflow instructions |
| `skills/property-based-testing/references/` | Focused library and advanced-testing guidance |
| `schemas/` | Versioned configuration, contract and evidence schemas |
| `testdata/` | Small repositories exercising real behavior and failure cases |
| `docs/` | Setup, runtime compatibility and operational limitations |

Keep one canonical skill source. Packaging may copy it into each distribution; a build check MUST prevent those copies from diverging. Do not create empty adapters for unsupported languages.

Use one Go module initially; create the internal packages above only as their implementations become necessary. Keep decision functions separate from I/O. Introduce small interfaces at concrete boundaries such as process execution, storage and time rather than a generic plugin framework. The public integration contract is the CLI/JSON protocol, not an importable Go SDK.

## 5. Feature workflow

### 5.1 Establish the task

Create or resume a task identity associated with a canonical repository root and worktree. Capture a baseline before task edits. Record pre-existing modifications without reverting or claiming ownership of them.

Do not create a new task for every message or automatic correction. A feature may span several prompts, compactions, and resumed sessions. Provide explicit begin/resume commands when the task boundary is unclear.

Inspect configured packages, manifests, lockfiles and existing tests. Prefer an existing suitable PBT dependency over installing a second library.

### 5.2 Define the property contract

Before substantial implementation, the agent MUST record a concise contract containing:

- Requirement or behavior identifier.
- Valid input domain, invalid inputs and relevant boundaries.
- Observable property and its preconditions.
- An independent oracle, invariant, reference model or metamorphic relation.
- Generator approach and state-reset needs.
- Intended package and test identifiers; fill final identifiers before verification.
- Reasoned applicability decision for behavior with no meaningful property.

The contract can evolve when requirements are clarified. A contract change invalidates previous verification. A failure alone is not justification to weaken the contract.

Example: a transfer operation may conserve combined balances on success, leave state unchanged on rejection, prevent negative balances when overdrafts are prohibited, and apply an idempotency key once. Only include guarantees actually required by the feature.

### 5.3 Implement testable behavior

The skill MUST instruct agents to:

- Separate business decisions from external effects where this makes testing practical.
- Inject clocks, randomness and external dependencies when behavior depends on them.
- Expose observable inputs, outputs and state transitions.
- Reset fixtures between generated examples and operation sequences.
- Keep database/network integration coverage where isolation would hide important behavior.
- Make the smallest justified refactoring; do not redesign unrelated modules.

Stateful testing is supported conceptually from the start. Each adapter must describe its actual capabilities; a simple explicit model may be used where a native state-machine API is absent.

### 5.4 Write meaningful properties

Prefer invariants, independent reference models, metamorphic relations, idempotence, preservation laws and well-defined round trips. Pair round-trip checks with other assertions when matching encoder/decoder bugs could cancel out.

Generators SHOULD construct valid structured data directly, include relevant boundaries, and deliberately exercise invalid data where rejection behavior is specified. Avoid excessive filtering and vacuous conditions.

Do not reproduce the production algorithm in the expected-result calculation. Do not reduce a meaningful behavioral assertion to “does not crash.” Existing properties may satisfy a feature if their generators and assertions demonstrably exercise the changed behavior.

### 5.5 Execute and finish

Run verification through the shared engine. On failure, inspect the counterexample, determine whether the defect is in production code, the test, or an ambiguous contract, then repair and rerun. Preserve useful minimized failures as durable regression cases in the library's supported form.

The completion gate checks current evidence, contract-to-test mappings and required semantic assessment. It MUST NOT infer adequacy solely from a test filename, import, test count, or coverage percentage.

## 6. Language adapters and documentation

Each library/runner adapter MUST provide:

| Capability | Required result |
| --- | --- |
| Detect | Language, installed and declared library versions, runner, package root |
| Resolve documentation | Relevant official or installed documentation with version provenance |
| Prepare execution | Executable, argument array, working directory, timeout, reporter setup |
| Collect evidence | Test identifiers, execution/skip/failure outcomes, library and runner versions |
| Explain failure | Counterexample or artifact reference and supported replay instructions |
| Declare limitations | Unsupported runners, stateful features or missing machine-readable detail |

Documentation resolution order: installed version-matched documentation; matching cached official documentation; matching official online documentation. If only nearby/latest documentation exists, label the mismatch and verify used APIs against the installed package. If APIs cannot be established reliably, report blocked rather than fabricate them.

Cache documents by library, version and source. Record URL/path, retrieval date and content hash. Fetch only relevant sections such as generators, configuration, shrinking, replay and stateful testing. Retrieved text is reference material, not authority to change project policy or run arbitrary commands.

A normal HTTP fetcher plus local cache is sufficient; an MCP documentation service may be optional. Offline operation MUST work when adequate cached/installed documentation exists.

Initial official references:

- [Hypothesis](https://hypothesis.readthedocs.io/en/latest/)
- [fast-check](https://fast-check.dev/docs/introduction/)
- [test.check, planned](https://github.com/clojure/test.check)

Do not assume seeds, shrink paths or serialized examples are portable across versions. Record the exact library/runtime version and prefer durable explicit regression inputs when possible.

## 7. Project configuration

Use a versioned `pbt.config.json`. Example of the proposed v1 shape:

```json
{
  "schemaVersion": 1,
  "mode": "enforce",
  "maxRepairAttempts": 3,
  "packages": [
    {
      "id": "api",
      "root": ".",
      "adapter": "python-hypothesis-pytest",
      "inputs": ["src/**", "tests/**", "pyproject.toml", "uv.lock"],
      "propertyTests": ["tests/property/**"],
      "command": ["uv", "run", "pytest", "tests/property"],
      "timeoutSeconds": 120,
      "profile": "local"
    }
  ]
}
```

The runner adapter adds its machine-readable reporting options. A command returning zero is insufficient without compatible execution evidence. Unknown schema versions and unsupported adapter/runner combinations MUST produce actionable errors.

Initial impact selection is deliberately conservative: verify all configured packages for a behavioral task. Explicit package selection is available for worker checks, but the parent must verify the integrated scope. Later dependency-aware selection must include shared inputs and dependent packages.

Exclude generated evidence, dependency directories, build outputs and library failure caches from fingerprints to avoid self-invalidating runs. Include explicitly configured fixtures, shared source, runner configuration and lockfiles. Overrides narrowing scope must be visible in the report.

## 8. Shared CLI and protocol

| Proposed command | Behavior |
| --- | --- |
| `pbt doctor` | Check installation, runner support and integration capabilities |
| `pbt begin --task <id>` | Capture baseline; fail rather than overwrite an existing identity |
| `pbt inspect --task <id> --json` | Report packages, versions and changes |
| `pbt docs --package <id>` | Resolve focused documentation and provenance |
| `pbt verify --task <id>` | Validate contract, execute tests and write evidence |
| `pbt status --task <id> --json` | Recompute freshness and return state |
| `pbt gate --task <id> --json` | Return completion/repair/blocked decision |
| `pbt resume --task <id>` | Resume a blocked task with an explicit fresh repair allowance |

Machine-readable output MUST include `schemaVersion`, task/worktree identity, state, reason codes and next action. JSON goes to stdout; diagnostics go to stderr. Invoke subprocesses using executable/argument arrays rather than interpolated shell strings. Resolve relative paths against the declared project root.

Adapters MUST support an explicitly configured absolute executable path, falling back to PATH lookup. Check the binary/protocol version at initialization and fail clearly on incompatibility. Agent adapters must not require `go run`, a Go compiler, or source checkout at runtime. Add `pbt version --json` for binary and supported protocol versions. Long-running execution must stream or retain bounded diagnostics without corrupting the final JSON result. Cancellation and timeouts must clean up child processes, including descendants using platform-appropriate handling.

`verify` and `gate` exit codes: `0` satisfied or explicitly not applicable; `1` a behavioral test failed; `2` pending/stale/blocked/incomplete verification; `3` invalid configuration or internal error. JSON distinguishes the specific outcomes. `status` exits zero for a successfully read state even if that state is failed; consumers inspect its JSON.

A satisfied not-applicable task MUST be reported as `not_applicable`, never as tests passed. A waiver, if later supported, must likewise remain distinct from verification.

## 9. Evidence, freshness and concurrency

Execution evidence MUST be engine-produced and contain:

- Schema version, task ID, run ID, worktree identity and timestamps.
- Package scope, input content fingerprint and contract/configuration hashes.
- Actual executable, arguments, working directory and relevant runtime versions.
- Actual resolved PBT library and runner versions.
- Collected, executed, skipped and failed test identifiers and counts.
- Process exit, timeout/cancellation status and report locations.
- Counterexamples and replay metadata when available.
- Documentation provenance and separately attributed semantic assessment.

Store transient evidence under ignored `.pbt/runs/`; store reviewable contracts under `.pbt/contracts/`. Do not commit logs containing generated sensitive inputs. Evidence is not an authentication mechanism.

Fingerprint file content and relative paths, including tracked, untracked, renamed and deleted relevant files. A Git commit SHA alone is insufficient. Capture fingerprints before and after a run; a mismatch prevents a passed result. Recompute freshness at every gate even if no file-change event fired.

Use atomic evidence writes and a per-worktree verification lock. A crashed run cannot leave a reusable running/pass record. Concurrent mutations make evidence stale. Separate worker worktrees keep separate evidence; after merge, the parent reruns integrated verification.

A resumed task retains its original baseline and retry ledger. A task starting without a known pre-edit baseline must report that limitation and require an explicit scope; it must not silently treat the current tree as evidence of no changes.

## 10. State and completion semantics

Maintain two distinct fields so retry policy never overwrites test history:

| Field | Values |
| --- | --- |
| `verificationState` | `pending`, `running`, `passed`, `failed`, `stale`, `not_applicable` |
| `workflowState` | `active`, `complete`, `blocked`, `cancelled` |

Relevant edits move prior verification to `stale`. A valid stable test run can produce `passed`; an assertion failure produces `failed`. Missing dependencies, unreadable documentation or a timeout block the workflow with an explicit reason and never produce passed verification.

Feature completion requires current passing evidence for every required scope, requirement-to-test mappings, and a recorded semantic assessment with no unresolved obligations. Deterministic checks validate evidence and structure; the implementing agent's assessment of property adequacy is attributed as an assessment, not represented as proof. Independent review remains optional initially.

Documentation-only tasks can finish as `not_applicable` with an explanation. Behavioral changes cannot use that state merely because test setup is inconvenient. Cancelled work may stop immediately without completing or passing.

## 11. Bounded automatic correction

Default: at most three agent repair cycles after the initial unsuccessful verification. This does not limit generated examples, shrinking steps or an explicitly configured library test budget.

Persist the repair ledger per task. De-duplicate lifecycle callbacks and repeated feedback for the same run. Consume an attempt when dispatching a repair instruction, not for every status query. Do not reset the count after edits, compaction, session restart or model change.

Each correction includes the unmet property, actual counterexample/report, current scope and remaining budget. No-progress repetitions or a known external blocker may stop earlier. The agent MUST NOT weaken tests solely to achieve a pass.

At exhaustion, stop automatic continuation, preserve the last verification result, and set `workflowState=blocked`. Present what failed and how to resume. Let the agent stop speaking; do not reinterpret that as successful feature completion. An explicit user request to continue can replenish the repair allowance.

## 12. Pi integration

Package the shared skill with a thin TypeScript extension. The extension invokes the Go `pbt` binary through Pi's supported process-execution facility; it does not import a TypeScript implementation of the core. Use the installed Pi version's documented APIs; record tested versions in a compatibility matrix.

Proposed event mapping:

| Mechanism | Responsibility |
| --- | --- |
| `before_agent_start` | Add concise PBT obligations and durable task status without replacing unrelated instructions |
| Registered tool/command | Expose verification, status and task controls |
| Tool execution events | Mark possible changes; avoid expensive full verification after every edit |
| `agent_end` | Evaluate the gate and request a de-duplicated corrective follow-up when allowed |
| Extension UI | Show engine-derived state and pending verification |

Pi documents `agent_end` as the end of a low-level run; it can be followed by retries or queued messages. Do not implement it as though it were a pre-stop veto. Use a labelled custom extension message to request a new turn, not a message impersonating the user. Coordinate with queued user input, cancellation and retry lifecycle behavior before dispatch.

An extension-level check may run after an assistant's apparent final answer. The UI must clearly distinguish that answer from verified completion. Strict noninteractive consumers must use an outer wrapper that checks the shared gate and returns a nonzero result when unsatisfied.

Third-party Pi worker launchers are separate compatibility targets. Workers must load the policy/package explicitly; do not assume parent extensions are inherited. When injection cannot be guaranteed, report the limitation and retain mandatory parent verification.

Reference: [Pi extension documentation](https://raw.githubusercontent.com/badlogic/pi-mono/main/packages/coding-agent/docs/extensions.md), [Pi skills documentation](https://raw.githubusercontent.com/badlogic/pi-mono/main/packages/coding-agent/docs/skills.md).

## 13. Claude Code integration

Distribute a plugin containing the skill and plugin-level hooks. Do not depend exclusively on hooks inside an optionally invoked skill.

Hook handlers invoke the Go CLI and translate its JSON into the native hook response. An agent-specific Go bridge command, such as `pbt hook claude-stop`, may implement this translation in an isolated integration package within the same module. Such bridge commands must call the shared gate logic rather than duplicate it. The Claude integration MUST NOT add a Python requirement; use executable invocation or a minimal platform shell wrapper where necessary.

| Mechanism | Responsibility |
| --- | --- |
| `SessionStart` | Inject policy and restore task context, including supported compaction/resume cases |
| `UserPromptSubmit` | Refresh current obligations without resetting task identity |
| `SubagentStart` | Inject the shared policy into delegated workers |
| `PostToolUse` | Mark potentially stale evidence |
| `Stop` | Consult the gate and return corrective feedback when allowed |
| `SubagentStop` | Assess obligations assigned to the worker, not unrelated parent work |

Hook handlers should be short: inspect state and request verification through the agent when needed. Long test execution belongs in the shared CLI, not an unbounded stop hook. Preserve existing hooks and honor runtime stop-loop protection, including `stop_hook_active` where supported. Exhausted retries must release continuation while preserving blocked status.

Research/planning workers must not inherit implementation obligations they cannot satisfy. Implementation workers receive a scoped contract; the parent verifies the integrated result. A worker in another worktree does not establish evidence for the parent's current files.

Validate hook payloads and feedback behavior against the installed version. Do not promise that hook coverage includes an arbitrary independently launched process.

Reference: [Claude hooks](https://code.claude.com/docs/en/hooks), [Claude skills](https://code.claude.com/docs/en/skills).

## 14. Applicability and user experience

The injected policy is conditional: behavioral implementation requires assessment and verification; explanations, planning and documentation do not trigger repair loops merely because the repository contains code.

Do not gate individual writes or require the tree to pass while the agent is still implementing. Gate completion, and in strict automation gate the process's success result. Existing skills retain their normal planning and implementation behavior.

Reports must clearly state: affected behavior, property IDs, current execution outcome, counterexample if any, and unresolved obligations. Distinguish passing PBT from overall feature validation; existing project checks remain required where applicable.

Support an advisory mode for installation trials, but default configured implementation tasks to enforcement mode. Unsupported runtime enforcement must be disclosed rather than silently advertised as equivalent.

## 15. Acceptance scenarios

Run deterministic core tests plus real fixture tests. Both native integrations require live smoke tests; simulated hook payloads alone do not establish runtime compatibility.

| ID | Scenario | Expected outcome |
| --- | --- | --- |
| A01 | Another skill implements a Python feature | PBT policy is available, meaningful properties are added or updated, and actual results are recorded |
| A02 | Inject a boundary bug | Generated input exposes it; failure and replay information are retained |
| A03 | Repair the bug | Stable rerun passes and the gate can complete |
| A04 | Edit source after a pass using a shell command | Gate detects stale evidence despite no dedicated edit event |
| A05 | Add an untracked source file or change a lockfile | Relevant fingerprint changes |
| A06 | Modify input during test execution | Run cannot become passed |
| A07 | Test runner returns zero with no properties executed | Verification remains unsatisfied |
| A08 | A required property is skipped | It does not satisfy that obligation |
| A09 | Test timeout, missing dependency or malformed reporter output | No pass; actionable blocked/error reason |
| A10 | Three unsuccessful repair cycles | No fourth automatic repair; blocked status retains failure evidence |
| A11 | Repeated event callback, compaction or restart | No duplicate repair dispatch and no retry-budget reset |
| A12 | Documentation-only task | Explicit not-applicable completion, no pointless tests |
| A13 | Pre-existing dirty worktree | User edits are preserved and baseline ownership is not misrepresented |
| A14 | Implementation worker in a separate worktree | Worker evidence stays scoped; parent verifies after integration |
| A15 | Planning worker returns | No obligation to implement or run feature tests |
| A16 | Cached matching docs available offline | Documentation resolution succeeds without network |
| A17 | Unsupported library version with unverified APIs | Explicit limitation/blocker rather than invented API use |
| A18 | Fast-check/Vitest fixture | Same contract and evidence semantics as the Python fixture |
| A19 | User cancels | Automatic repairs stop and work is not marked verified |
| A20 | CI begins with no local evidence | Fresh execution establishes its own result |
| A21 | Weak tautological property offered | Semantic assessment identifies the gap; test execution alone is not presented as adequate coverage |
| A22 | No-progress repair or external blocker | Stop early with preserved details |
| A23 | Run release binary in a Node.js-only fixture environment without Python or Go | Core inspection, fast-check verification and gate succeed |
| A24 | Run release binary in a Python-only fixture environment without Node.js or Go | Core inspection, Hypothesis verification and gate succeed |
| A25 | Agent adapter has a different PATH from the interactive shell | Configured absolute binary path works; missing binary produces actionable setup feedback |
| A26 | Adapter encounters an incompatible CLI protocol | Clear compatibility error, never a false pass |

Use stable seeded fixtures or explicit boundary generation for harness regression tests so testing the harness does not depend on randomly discovering a rare failure. Production PBT profiles can explore fresh randomness while retaining replay artifacts.

## 16. Delivery milestones

### M0 — confirm runtime integration points

Inspect installed Pi and Claude versions. Build minimal policy-injection and feedback probes in a disposable fixture. Record actual hook/event behavior, compaction, cancellation and worker limitations. Do not implement the full framework before these assumptions are checked.

### M1 — Go core with a Python testing vertical slice

Implement the Go configuration, task identity, contract validation, fingerprints, Hypothesis/pytest reporting, CLI, state and retry ledger. Python is the target fixture language, not the core implementation language. Demonstrate implement → fail → repair → pass → edit → stale in one small repository. Complete A02–A13 and A19–A20 at core level.

### M2 — Pi package

Add policy injection, verification tool, status UI and bounded follow-up. Demonstrate another skill implementing the fixture. Test cancellation and duplicate events. Document the distinction between interactive feedback and strict wrapper enforcement.

### M3 — Claude plugin

Add plugin-level hooks against the same core and fixture. Demonstrate another skill and a scoped implementation worker. Verify hook composition and loop termination with real sessions.

### M4 — fast-check adapter and release candidate

Add fast-check/Vitest and demonstrate both runtimes against both language fixtures. Publish setup instructions and the tested version matrix. CI must exercise core fixtures independently of live LLM calls. Keep live agent smoke tests as an explicit release validation step.

Build release archives for macOS and Linux, on amd64 and arm64. Include the executable, license, and required skill/reference/reporter assets, either embedded or in a documented relocatable bundle. Provide checksums and a binary version. Do not promise a specific download size before measuring the release build. Validate runtime independence with A23–A24 and executable/protocol discovery with A25–A26. Users downloading releases need no Go toolchain; source builds do. Agent integration packages may bundle a matching binary or discover an explicitly installed one, but must document the selected approach and avoid hidden runtime downloads.

### M5 — expansion

Add Clojure/test.check, then additional agents through the existing protocol. Add independent review only if demonstrated property-quality failures justify it.

## 17. Implementation handoff

Begin with M0, then build M1 as a narrow working slice. Avoid speculative abstractions for unsupported languages. Keep native hooks thin and decision logic in the core. Use meaningful behavioral tests for state, freshness and reporting; do not test prompt wording as a substitute for observed behavior.

Before claiming a milestone complete, record what was exercised locally, what was simulated and which runtime versions were used. If a documented integration point is unavailable, update the compatibility record and adapt the integration without weakening shared verification semantics.

This specification authorizes the proposed implementation scope when handed to an implementing agent; it does not imply permission to publish packages, alter unrelated repositories, install global configuration, or change external branch protections.
