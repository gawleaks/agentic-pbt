# agentic-pbt implementation progress

**Delivery:** Pi-only v1
**Design:** [`docs/superpowers/specs/2026-09-17-pi-only-v1-design.md`](superpowers/specs/2026-09-17-pi-only-v1-design.md)
**Tracker status:** Complete
**Last assessed:** 2026-09-17

## Current repository assessment

| Item | State | Evidence |
| --- | --- | --- |
| Pi-only v1 CLI | Complete | Go `pbt` implements configuration, contracts, task state, fingerprints, Hypothesis/pytest verification evidence, gates, bounded repairs, resume, and cancellation. |
| Python acceptance fixture | Complete | `testdata/v1-python/verify.sh` proves fail → repair → pass → stale with real Hypothesis/pytest. |
| Shared agent guidance | Complete | `skills/property-based-testing/` supplies the contract and verification workflow plus a focused Hypothesis reference. |
| Pi lifecycle adapter | Complete | `integrations/pi/` injects active-task guidance and delegates the completion gate to the CLI without Pi tools or slash commands. |
| Source workflow | Complete | `README.md` documents source installation, Pi loading, `PBT_BIN`, local evidence, repairs, cancellation, and direct CLI use in CI. |

## Pi-only v1 delivery

Pi-only v1 is complete. It replaces the former M1/M2/M3 delivery ordering with
one delivered scope: the standalone Go CLI, one Python Hypothesis/pytest
vertical slice, the shared skill, and a thin Pi adapter.

- [x] Install from source with `go install ./cmd/pbt`; release binaries are deferred.
- [x] Commit `pbt.config.json` and reviewable `.pbt/contracts/` artifacts.
- [x] Keep task state and verification evidence local under ignored `.pbt/state/` and `.pbt/runs/`.
- [x] Support one active task per worktree and a CLI-owned three-follow-up repair budget.
- [x] Preserve evidence on cancellation and require explicit `pbt resume --task <id>` before another repair budget.
- [x] Keep Pi as a completion guardrail while developers and CI call `pbt verify` and `pbt gate` directly.

Claude Code integration is deferred. Workers, multi-package selection, other
language adapters, online documentation retrieval, custom Pi controls, and
release binaries are also deferred.

## Archived M0 compatibility evidence

M0 records raw runtime observations from before the Pi-only v1 delivery. They
remain in [`docs/compatibility/runtime-integration-m0.md`](compatibility/runtime-integration-m0.md)
for provenance, including the Claude fixture observations. They are archived
compatibility evidence, not Pi-only v1 delivery blockers; no Claude session is
required for the v1 verification suite.

### M0 — Confirm runtime integration points (archived)

**State:** In progress

The historical fixture has live evidence for policy delivery, hook firing, process invocation, queued follow-up, cancellation, and compaction. Claude `additionalContext` delivery is live-verified with valid `hookEventName` output.
Its unresolved interactive observations do not alter the delivered Pi-only v1
scope or block its use.

## Verification suite

The complete local suite is:

```bash
asdf exec gofmt -w cmd internal
asdf exec go vet ./...
asdf exec go test ./...
skills/property-based-testing/verify-skill.sh
integrations/pi/verify-extension.sh
testdata/v1-python/verify.sh
testdata/m0-runtime-probes/verify-docs.sh
testdata/m0-runtime-probes/verify-fixture-tools.sh
PATH=/usr/bin:/bin testdata/m0-runtime-probes/claude/verify-plugin.sh
git diff --check
```

The final Claude command validates only the offline fixture/plugin contract; it
does not launch a Claude session.

## Progress log

| Date | Scope | Change | Verification evidence |
| --- | --- | --- | --- |
| 2026-09-17 | Pi-only v1 | Delivered the Go CLI, Python fixture, shared PBT skill, and thin Pi lifecycle adapter. | Go unit tests, skill check, Pi adapter checks, and the real Python acceptance fixture. |
| 2026-09-17 | Documentation | Documented source installation and operational workflow; archived M0 observations as non-blocking evidence. | Full local suite above, including offline Claude fixture validation. |
