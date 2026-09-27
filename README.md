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
