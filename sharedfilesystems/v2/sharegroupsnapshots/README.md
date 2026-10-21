# Share group snapshots

SDK-owned Manila binding (absent from Gophercloud v2.15.0). Create it with `New(client)` or through the accessor on the shared-file-system `Service`.

- Microversion floor: 2.55. A numeric selected version below it, or no selected version, fails with `resource.ErrUnsupported` before any request; `latest` is passed through. The SDK never upgrades the selected version.
- Routes: `/share-group-snapshots[/detail]`, `/{id}`, `/{id}/action` (reset_status), `/{id}/members`. `List` uses `/detail` unless `WithListDetails(false)`.
- Every call is one HTTP request. Nothing is retried or replayed, a canceled context sends nothing, and acknowledgement of an asynchronous create/delete does not prove completion.
- Results are typed snapshots with an `Evidence` field (actual JSON object including null and unknown members, response headers, HTTP status).
- Extension JSON/query inputs (`With...Field`, `With...Query`) cannot override typed or server-owned fields; collisions fail before HTTP.
`ListMembers` returns `share_group_snapshot_members` plus the whole response. `ResetStatus` performs no preceding GET.

Python comparison and differences: [python-parity-manila-a-round2.md](../python-parity-manila-a-round2.md).
