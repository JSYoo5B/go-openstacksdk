# SDK support ledger

Coverage is evaluated against Gophercloud **v2.15.0** and Python openstacksdk **ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe**. Generated functions, test counts, and behavior-contract counts are distinct from completed APIs.

The [operation review JSON](../api/sdk_reviews.json) records public Go APIs, source fingerprints, behavior contracts, test selectors, documentation, differences, and unresolved boundaries. The [coverage tables](implementation-plan.md) are generated from these decisions. `make parity` validates references; `make progress` refreshes counts and `make progress-check` checks freshness.

## Status definitions

| Status | Meaning |
|---|---|
| `supported` | Pinned source behavior is implemented with concrete API, contract tests, and usage documentation. |
| `go_mapping` | An intentional Go API with explicit differences in types, defaults, results, or errors provides the functionality. |
| `unsupported` | Behavior is explicitly excluded, with a concrete reason. It is not supported functionality. |
| `unresolved` | Behavior or evidence remains incomplete, including declarations without reviews. Partial verified contracts can coexist with this status. |

A completed review cannot retain unresolved behavior. Go mappings must explain observable differences rather than use type differences as a reason to avoid required functionality. Removing a method from an implementation target does not make the overall SDK complete.

## Shared unresolved boundaries

- SDK-R1/C1/S1: mutable Resource, adapter/session/cache lifecycles and dynamic descriptors.
- Full unknown/partial/empty/malformed success-response merging and mutable Resource reuse.
- Complete Requests encoding, accepted-status profiles, nonfinite JSON, and arbitrary Python object/iterable identities.
- Fetch/constructor/header/microversion controls on lookup bindings that have not adopted the corresponding owned entry points.
- Authenticated deployment acceptance, policy overrides, and release-specific interoperability.

These shared limitations do not erase narrower completed operation contracts. Read each guide's exact input/result/error profile. [Priority planning](implementation-plan.md), [policy branches](core-policy-branches.md), and [release criteria](release-milestones.md) distinguish implementation targets from verification.

## Completion criteria

A support decision needs concrete public inputs, omission/null/false/zero rules, library-owned serialization, defaults, result ownership, and actual error semantics. Where applicable, verify exact/duplicate names, missing policies, paging, cancellation, deletion, waits, versions, and parent identities. Composite operations need ordered phase evidence, partial-success behavior, and documented retry/cleanup policy.

Keep inherited/descriptor/Resource surfaces visible beyond the directly declared catalog. Source inspection, bounded Python execution, Go HTTP fixtures, example builds, authenticated cloud acceptance, and CI are separate evidence. A source fingerprint validates the comparison identity, not test success or complete SDK equivalence.
