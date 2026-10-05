# Pi-only v1 design

**Status:** Accepted
**Scope:** Go `pbt` CLI, Python Hypothesis/pytest adapter, and Pi extension only

This design refines the v0.2 proposal for the first shippable slice. It
supersedes its multi-runtime v1 scope: Claude Code, additional languages,
automatic package discovery, release binaries, and worker support are deferred.

## Goals

- Make property verification an explicit, durable part of one Pi implementation
  task at a time.
- Keep verification semantics in a standalone Go CLI usable by Pi, developers,
  and CI.
- Require a reviewable property contract and current pytest/Hypothesis evidence
  before Pi treats an implementation task as complete.
- Preserve a developer's dirty worktree and real test failure evidence.

## Non-goals

- Claude Code integration, Pi workers, multi-package repositories, and custom
  Pi task-control tools.
- Automatic property invention, semantic proof of a property, documentation
  fetching/caching, or running normal unit/integration suites.
- Prebuilt release binaries. V1 is installed with `go install`; projects running
  the resulting executable do not need Go to execute it.

## Architecture

``pbt` is the deep module: it owns configuration validation, active-task state,
contracts, fingerprints, pytest execution, its bundled pytest reporter, evidence,
retry decisions, and the completion gate. Its interface is CLI commands with optional JSON output.

The Pi TypeScript extension is a thin adapter. It does not parse pytest output,
implement freshness, or track retries itself. It:

1. detects an active task for its current worktree;
2. injects concise instructions to read the bundled PBT skill and use `pbt`;
3. calls `pbt gate --json` at `agent_before_settle` for completed outcomes; and
4. renders an actionable setup error if an active task cannot resolve `pbt`.

With no active task, the extension injects nothing, runs no gate, and queues no
follow-up. The extension resolves `pbt` from its process `PATH`, overridden by
`PBT_BIN` when set to an absolute executable path.

## Repository configuration

Each target project commits one `pbt.config.json`:

```json
{
  "schemaVersion": 1,
  "projectRoot": ".",
  "inputs": ["src/**", "tests/**", "pyproject.toml", "uv.lock"],
  "python": ["uv", "run", "python"],
  "command": ["uv", "run", "pytest", "tests/property"],
  "timeoutSeconds": 120
}
```

Paths are relative to the configuration file. `inputs`, the configuration, and
the active contract form the verification fingerprint. `python` runs a small
version query for `inspect`; it reports pytest and Hypothesis versions. `command`
is one executable-and-arguments array, never a shell string. V1 supports exactly
one project root and one property-test command.

## Tasks, contracts, and local artifacts

Before task edits, a developer or implementing agent runs:

```bash
pbt begin --task add-discount-code
```

V1 permits one active task per worktree. `begin` validates the config and
records a baseline, including pre-existing relevant modifications; it never
reverts or claims those modifications.

The implementing agent creates a committed contract at
`.pbt/contracts/add-discount-code.json` before substantive implementation:

```json
{
  "schemaVersion": 1,
  "task": "add-discount-code",
  "applicability": "required",
  "behavior": "Apply a valid percentage discount once.",
  "properties": [
    {
      "id": "discount-total",
      "statement": "A valid percentage reduces the subtotal by that percentage.",
      "generator": "non-negative money amounts and integer percentages 0 through 100",
      "oracle": "independent decimal arithmetic",
      "test": "tests/property/test_discount.py::test_discount_total"
    }
  ]
}
```

Each required property has an ID, behavioral statement, generator description,
independent oracle/invariant/metamorphic relation, and pytest node ID. The
alternate form is:

```json
{
  "schemaVersion": 1,
  "task": "update-readme",
  "applicability": "not_applicable",
  "reason": "Documentation-only change; no behavior changes."
}
```

`not_applicable` is valid only for genuinely non-behavioral work. The Pi policy
instructs agents not to use it to evade verification.

Operational files are local and ignored:

```text
.pbt/state/active.json      active-task pointer for the worktree
.pbt/state/<task>.json      baseline, states, retry ledger, run references
.pbt/runs/<run>/            result JSON, JUnit XML, node-result JSONL, bounded stdout/stderr
```

Only contracts are committed from `.pbt/`.

## CLI interface and states

The v1 CLI commands are:

```text
pbt begin --task <id>
pbt inspect --task <id> --json
pbt verify --task <id> --json
pbt status --task <id> --json
pbt gate --task <id> --json
pbt resume --task <id>
```

Pi-invoked commands emit one versioned JSON result on stdout; diagnostics go to
stderr. `inspect` is read-only and reports the resolved root, command, contract
path, and installed pytest/Hypothesis versions.

Task state separates evidence from workflow:

```text
verificationState: pending | running | passed | failed | stale | not_applicable
workflowState:    active  | complete | blocked | cancelled
```

For example, exhausted repairs produce `failed` plus `blocked`; cancellation
preserves the last evidence state but sets `cancelled`.

`gate` exits 0 for satisfied or justified `not_applicable`, 1 for a property
failure, 2 for pending/stale/blocked/incomplete state, and 3 for invalid
configuration, missing dependencies, or internal/setup errors. JSON supplies a
specific reason code and next action.

## Verification and freshness

`pbt verify` takes one per-worktree lock. A concurrent request returns a
`running` result and cannot overwrite evidence.

For a required contract it:

1. validates config and contract shape;
2. runs the configured pytest selection with `--collect-only`;
3. confirms every contract node ID is collected by that selection;
4. runs the configured command with a temporary `--junitxml` output path and
   the bundled `pbt_pytest_reporter` plugin;
5. requires every contract node ID to report passed, not skipped, in the
   plugin's exact-node-result JSONL;
6. captures bounded output, the JUnit report, and node-result JSONL; and
7. fingerprints the configured inputs, config, and contract before and after the
   run.

A zero process exit alone never satisfies verification. Pytest's JUnit XML does
not reliably preserve node IDs, so the bundled reporter's JSONL is authoritative
for required-node outcomes; JUnit remains human-readable evidence. Missing nodes,
skipped required tests, assertion failures, malformed reports, missing
pytest/Hypothesis, and timeouts cannot pass. Assertion failures, missing nodes, and skipped tests
are repairable `failed` evidence. Setup failures and timeouts are actionable
blockers. Any relevant change after a pass makes evidence `stale`; `gate`
recomputes the fingerprint instead of trusting lifecycle events.

A `not_applicable` contract records `verificationState=not_applicable` without
launching pytest, but still requires a valid active task and contract.

## Pi completion behavior

Before the agent works on an active task, Pi injects the task ID and requires the
agent to read the shared PBT skill, create/update the contract, implement the
feature, and run `pbt inspect`/`pbt verify` as applicable. The skill includes a
focused Hypothesis reference and links to the official documentation:
<https://hypothesis.readthedocs.io/en/latest/>. Fetching that documentation is
an agent action, not a `pbt` feature.

At `agent_before_settle`, Pi asks `pbt gate` for a decision only when the
outcome is `completed`. A satisfied decision requires no action. A new
unsatisfied decision appends a labelled corrective follow-up with the real
failure/evidence location and next action, then returns `continue: true`. The
state ledger de-duplicates repeated callbacks for the same result. At most three
follow-ups, including missing-verification reminders, are issued. Thereafter the
task is `blocked` and Pi does not auto-continue.

For `aborted` or `error` outcomes, Pi returns no continuation. Escape or other
user cancellation therefore retains artifacts and suppresses future automatic
follow-ups. An explicit
`pbt resume --task <id>` restores `active` and grants a fresh three-follow-up
budget without changing the original baseline or deleting evidence.

## Acceptance evidence

Core fixture tests must prove:

- explicit begin, required contract validation, and `not_applicable` handling;
- a failing Hypothesis property, repair to a pass, then stale evidence after a
  relevant edit;
- missing/renamed/skip-required node IDs, timeout, malformed JUnit, and missing
  setup dependencies; and
- dirty-baseline preservation, lock behavior, and retry de-duplication.

Pi fixture tests must prove the installed Pi version's active-task policy
injection, `agent_before_settle` gate invocation, labelled bounded de-duplicated
follow-up, and cancellation behavior. These are workflow guardrails, not a security
boundary: users may disable the extension, while CI can run `pbt verify` and
`pbt gate` directly.
