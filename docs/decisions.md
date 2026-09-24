# Design decisions

This is the running decision log for agentic-pbt. Add a decision here as soon as
it is agreed; do not add placeholders for undecided items.

## V1 scope

- V1 contains the standalone Go `pbt` CLI, a Python Hypothesis/pytest adapter,
  and a thin Pi extension.
- The CLI remains usable directly by developers and CI. Pi invokes it; Pi does
  not replace it.
- Claude Code integration is out of scope for v1.

## Task and completion workflow

- A task starts explicitly before edits: `pbt begin --task <id>`.
- V1 permits one active task per worktree. Pi resolves the task from that active
  worktree state; concurrent task routing is deferred.
- Either the developer or the implementing agent may run `begin`, but it must
  happen before task edits so `pbt` captures the baseline.
- Pi runs `pbt gate --task <id> --json` only after the agent finishes. It does
  not gate individual edits.
- An unsatisfied gate queues at most three labelled corrective follow-ups,
  including reminders to run verification. Then the task is `blocked` and Pi
  stops automatic continuation.
- `pbt resume --task <id>` is an explicit user action. It preserves the
  baseline and history while granting a new three-follow-up budget.
- Cancellation sets the task to `cancelled`, retains evidence, and suppresses
  automatic follow-ups. Cancellation is never verified completion.

## Repository configuration and verification

- Every target repository commits one `pbt.config.json`.
- V1 supports one Python project root and one configured property-test command.
  It does not select among multiple packages.
- `pbt.config.json` requires a positive `timeoutSeconds` for that command. A
  timeout leaves verification unsatisfied.
- The property-test command is an executable-and-arguments JSON array, never a
  shell string; `pbt` executes it without shell interpolation.
- `pbt verify` runs only that configured property-test command; normal unit and
  integration suites remain project-owned checks.
- `pbt verify` first runs the configured pytest selection with `--collect-only`,
  validates every contract-named pytest node ID, then runs the actual configured
  property-test command.
- Every contract-named pytest node ID must be collected and included in the
  configured command. Exit status zero alone is insufficient.
- `pbt` fingerprints configured relevant files, the configuration, and the
  contract before and after verification; `gate` recomputes freshness. Relevant
  changes make prior passing evidence `stale`.
- `pbt begin` permits a dirty worktree, records pre-existing relevant changes,
  and never changes or claims ownership of them.

## Property contracts and guidance

- A valid property contract is required before `pbt verify` or `pbt gate` can
  pass.
- The implementing agent writes `.pbt/contracts/<task>.json` before substantive
  implementation; the developer can review or correct it.
- Contracts are committed reviewable artifacts. They name the behavior,
  properties, generators, and intended pytest node IDs.
- A contract may state `not_applicable` with a reason for documentation-only or
  otherwise non-behavioral work. It must not waive property testing for a
  behavioral change.
- The distribution includes one shared PBT skill and a focused Hypothesis
  reference. Pi tells the agent to read the skill for the active task.
- The focused reference includes the official Hypothesis documentation link:
  <https://hypothesis.readthedocs.io/en/latest/>. The agent may fetch it with
  its normal tools; v1 `pbt` does not fetch, cache, or treat online
  documentation as verification evidence.

## Pi installation boundary

- The Pi extension resolves `pbt` from the Pi process's `PATH`.
- `PBT_BIN` may supply an absolute executable path when Pi's `PATH` differs from
  the interactive shell.
- If neither resolves to an executable, Pi displays a clear setup error rather
  than treating the task as verified or silently skipping the gate.
- Commands called by Pi accept `--json` and write one versioned result to
  stdout. Diagnostics go to stderr.
- V1 includes `pbt inspect --task <id> --json`, a read-only report of the
  resolved root, property-test command, contract path, and installed
  Hypothesis/pytest versions.
- `pbt gate` exits `0` when satisfied or `not_applicable`, `1` after property
  verification failure, `2` when pending/stale/blocked/incomplete, and `3` for
  invalid configuration or internal/setup errors. JSON gives the precise reason.

## Local operational artifacts

- Task state records separate `verificationState` (`pending`, `running`,
  `passed`, `failed`, `stale`, `not_applicable`) and `workflowState` (`active`,
  `complete`, `blocked`, `cancelled`) so workflow decisions never erase test
  evidence.
- Task state (baseline, retry count, status, and run references) is stored in
  ignored `.pbt/state/<task>.json` files.
- Transient execution evidence is stored in ignored `.pbt/runs/`.
- Only contracts are committed from `.pbt/` in v1.
