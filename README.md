# agentic-pbt

`agentic-pbt` is a Pi-only v1 workflow for durable Python
Hypothesis/pytest property-test verification. The Go `pbt` CLI owns task state,
contracts, evidence, freshness, and completion decisions; the Pi extension is a
thin completion guardrail.

## Install from source

From a checkout, install the CLI with Go:

```bash
go install ./cmd/pbt
```

Ensure Go's binary directory is on your shell `PATH`, then load the extension
with its absolute path:

```bash
pi -e /absolute/path/to/agentic-pbt/integrations/pi/index.ts
```

Pi resolves `pbt` from the Pi process `PATH`. If that differs from your shell,
set `PBT_BIN` to the absolute installed executable path before starting Pi:

```bash
export PBT_BIN="$(go env GOPATH)/bin/pbt"
pi -e /absolute/path/to/agentic-pbt/integrations/pi/index.ts
```

`PBT_BIN` must be absolute. The extension displays a setup error instead of
silently treating an active task as verified when it cannot run the CLI.

## Use a task

Commit one `pbt.config.json` for the Python project and commit each reviewable
`.pbt/contracts/<task>.json` contract. Start the task before changing its
behavior, inspect the configured environment, then verify and gate it:

```bash
pbt begin --task <id>
pbt inspect --task <id> --json
pbt verify --task <id> --json
pbt gate --task <id> --json
```

The contract names the behavior, generators, independent oracle or relation,
and exact pytest node IDs. Use `not_applicable` only for genuinely
non-behavioral work. Read
[`skills/property-based-testing/SKILL.md`](skills/property-based-testing/SKILL.md)
for the required contract and Hypothesis workflow.

V1 permits one active task per worktree. It records the pre-edit baseline but
does not claim ownership of existing changes. Local operational data is ignored:

- `.pbt/state/` holds the active-task pointer, baseline, state, and repair ledger.
- `.pbt/runs/` holds result JSON, JUnit XML, exact-node JSONL, and bounded output.
- `.pbt/contracts/` is the committed, reviewable exception.

A passing verification becomes stale after a relevant input, config, or contract
change. `pbt gate` recomputes that freshness rather than trusting a lifecycle
callback.

## Example: add a discount feature with Pi

Suppose `~/work/shop` is an existing Git repository using `uv`, with Python
source under `src/` and property tests under `tests/property/`. After installing
`pbt` as above, prepare the project from its root:

```bash
export PBT_HOME=/absolute/path/to/agentic-pbt # checkout containing integrations/pi and skills/
cd ~/work/shop
uv add --dev pytest hypothesis
mkdir -p skills
cp -R "$PBT_HOME/skills/property-based-testing" skills/
```

Create `pbt.config.json` in `~/work/shop` (the `uv.lock` file comes from `uv`):

```json
{
  "schemaVersion": 1,
  "projectRoot": ".",
  "inputs": ["src/**/*.py", "tests/**/*.py", "pyproject.toml", "uv.lock"],
  "python": ["uv", "run", "python"],
  "command": ["uv", "run", "pytest", "tests/property"],
  "timeoutSeconds": 120
}
```

Commit the config, dependency changes, and copied skill. **Before changing the
discount behavior**, start the task and launch Pi from the *shop* directory:

```bash
git add pbt.config.json pyproject.toml uv.lock skills/property-based-testing
git commit -m "chore(test): configure property verification"
pbt begin --task discount
PBT_BIN="$(go env GOPATH)/bin/pbt" pi -e "$PBT_HOME/integrations/pi/index.ts"
```

Give Pi a concrete request:

> Implement `apply_discount(subtotal, percent)` in `src/discount.py` for
> non-negative integer subtotals and whole-number percentages from 0 to 100.
> Read `skills/property-based-testing/SKILL.md`, inspect the active task, and
> create `.pbt/contracts/discount.json` before implementation. Cover the
> behavior with a Hypothesis property at
> `tests/property/test_discount.py::test_discount_total` using an independent
> oracle or invariant. Run `pbt verify --task discount --json`; fix any failures
> without weakening the property.

Pi can use its existing shell tool to create the contract and test. The
extension injects the active-task instructions and, if Pi tries to finish
without current evidence, queues a bounded repair follow-up. After Pi finishes,
check the result in the project root:

```bash
pbt gate --task discount --json
```

For this behavioral task, the gate must report `verificationState: passed` and
`workflowState: complete`. Review and commit `src/discount.py`, the property
test, and `.pbt/contracts/discount.json`; keep `.pbt/state/` and `.pbt/runs/`
local. Re-run `pbt verify --task discount --json` if you edit relevant files
after the pass.

## Repairs and cancellation

At a completed Pi turn, the extension calls `pbt gate --task <id> --json`.
For a new unsatisfied result, `pbt` can issue up to three labelled repair
follow-ups. Repeated settlement callbacks for the same result do not consume
the budget. After the third follow-up, the task is blocked until explicitly
resumed:

```bash
pbt resume --task <id>
```

Cancellation preserves state and evidence, suppresses automatic follow-ups, and
is not verified completion:

```bash
pbt cancel --task <id> --json
```

## Pi and CI roles

Pi is a workflow guardrail, not a security boundary: users can disable the
extension. CI and developers enforce the same semantics by calling the CLI
directly, for example `pbt verify --task <id> --json` followed by
`pbt gate --task <id> --json`. The extension does not add Pi tools or slash
commands and does not reimplement verification or gate logic.

## Deferred scope

Claude Code integration is deferred. V1 also defers workers, multi-package
selection, other language adapters, online documentation retrieval, release
binaries, and custom Pi task controls.
