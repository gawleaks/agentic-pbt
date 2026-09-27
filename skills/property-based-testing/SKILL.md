---
name: property-based-testing
description: Use when an active implementation task changes behavior and needs a PBT contract, verification evidence, or a response to a failing property test.
---

# Property-Based Testing

For a behavioral change, record a falsifiable contract and verification evidence before calling it complete. A property must constrain behavior; an assertion that repeats the implementation is not evidence.

## Active-task workflow

1. Start the task before edits: `pbt begin --task <id>`.
2. Inspect the resolved project, command, contract path, and installed tools: `pbt inspect --task <id> --json`.
3. Before substantive edits, create the contract at the inspected path. Describe the behavior, generators, independent oracle or relation, and exact pytest node IDs. Review it with the task.
4. Implement the behavior and a property test. Prefer an invariant, reference model, or metamorphic relation over a tautology that duplicates the implementation.
5. Run `pbt verify --task <id> --json`, repair the reported counterexample or setup problem, then run it again.
6. Before completion, run `pbt gate --task <id> --json`. A stale, failed, pending, or blocked result is not completion.

Do not use not_applicable for a behavioral change.

## Choosing a property

| Need | Property shape |
| --- | --- |
| Stable rule | Invariant: output remains in range, ordered, or conserved. |
| Trusted simpler logic | Reference model: compare against an independent calculation. |
| Related valid inputs | Metamorphic relation: transform input and assert the corresponding output change. |

Do not weaken a property merely to make a counterexample pass. First reproduce and classify it: implementation defect, oracle/property defect, invalid generator domain, or environment/setup failure. Narrow a generator only when the contract's real input domain excludes the values; update the contract and test together.

## Reference

For Python test patterns, read `references/python-hypothesis.md`. `pbt` runs the configured test command and records evidence; it does not download or validate online documentation.
