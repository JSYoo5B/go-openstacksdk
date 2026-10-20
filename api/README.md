# API reference

The Gophercloud v2.15.0 inventory provides 1,126 public client-taking operations across 194 resource packages. [gophercloud_inventory.json](gophercloud_inventory.json) records generated declarations and transport metadata. `make generate` regenerates facades; `make check` validates contracts and references.

Each leaf package exposes `New(client)` and an API object. Concrete options replace caller-implemented request builders. Operation-specific With functions configure optional inputs, queries, and headers while rejecting protected collisions. Lists use `iter.Seq2`; ordinary operations return values/errors; stream results have explicit ownership.

Generated native bindings are separate from Python Resource/workflow equivalence. SDK-owned APIs can preserve additional metadata, null/omission, raw bodies, headers, statuses, and partial results. Editing a response does not automatically persist it. Consult each service guide and the [review JSON](sdk_reviews.json) for the actual result and error contract.

## Inventories and support

- [Gophercloud operations](gophercloud_inventory.json)
- [Common resource bindings](resource_inventory.json)
- [Python openstacksdk inventory](openstacksdk/README.md)
- [Source-fingerprinted catalog](sdk_support_catalog.json)
- [Operation contracts and decisions](sdk_reviews.json)
- [Support definitions](../docs/sdk-support-ledger.md) and [coverage](../docs/implementation-plan.md)

Read [identity lookup](../docs/finding-identities.md), [listing](../docs/listing.md), [parent scopes](../docs/scoped-resources.md), [waits](../docs/service-waits.md), and [microversions](../docs/microversions.md) before assuming a common policy applies to a specific resource.
