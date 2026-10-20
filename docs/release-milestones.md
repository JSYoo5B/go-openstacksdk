# Release criteria

A development preview exposes bounded, documented functionality while the full SDK remains incomplete. Version names are plans until tags are published.

## Core user preview

`make smoke` exercises five HTTP-fixture workflows: server/network dependency resolution and partial results, volume creation/attachment/waits, image import acceptance/checksum download, secret metadata/payload reads, and Swift request/checksum upload behavior. It does not claim authenticated cloud acceptance or complete API support.

An installable alpha requires a published module, reproducible generator, valid support references, English usage documentation, independent consumer compilation, passing smoke/full checks, and explicit known limitations. The planned first alpha is `v0.1.0-alpha.1`; this document does not assert that it is published.

## Following milestones

Complete bounded core user contracts before core administration, then following-service user and administrative APIs. Prioritize Octavia/Designate, then Ironic/Introspection, then other services. Keep required inputs, default options, permission branches, versions, results, and errors explicit per operation. [Coverage](implementation-plan.md) records API decisions independently of release readiness.

Authenticated acceptance is a separate release criterion when an authorized deployment is available. Do not label HTTP fixture tests or an alpha publication as deployment acceptance.
