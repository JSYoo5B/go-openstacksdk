# Share network subnets

SDK-owned Manila binding (absent from Gophercloud v2.15.0). Create it with `New(client)` or through the accessor on the shared-file-system `Service`.

- Microversion floor: 2.51 (metadata routes: 2.78). A numeric selected version below it, or no selected version, fails with `resource.ErrUnsupported` before any request; `latest` is passed through. The SDK never upgrades the selected version.
- Routes: `/share-networks/{network}/subnets[/{id}]` and `.../{id}/metadata[/{key}]`; `InNetwork(ctx, networkID)` fixes the parent without a request and Create uses the envelope key `share-network-subnet`.
- Every call is one HTTP request. Nothing is retried or replayed, a canceled context sends nothing, and acknowledgement of an asynchronous create/delete does not prove completion.
- Results are typed snapshots with an `Evidence` field (actual JSON object including null and unknown members, response headers, HTTP status).
- Extension JSON/query inputs (`With...Field`, `With...Query`) cannot override typed or server-owned fields; collisions fail before HTTP.
Metadata keys are path-escaped; `DeleteMetadata` deduplicates keys in first-seen order and stops at the first failure.

Python comparison and differences: [python-parity-manila-a-round2.md](../python-parity-manila-a-round2.md).
