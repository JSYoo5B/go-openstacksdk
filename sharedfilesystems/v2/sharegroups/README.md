# Share groups

SDK-owned Manila binding (absent from Gophercloud v2.15.0). Create it with `New(client)` or through the accessor on the shared-file-system `Service`.

- Microversion floor: 2.55 (earlier versions are experimental and need a header this package never sends). A numeric selected version below it, or no selected version, fails with `resource.ErrUnsupported` before any request; `latest` is passed through. The SDK never upgrades the selected version.
- Routes: `/share-groups` (summary rows), `/share-groups/detail`, `/share-groups/{id}`. Create 202/200, Update 200, Delete 200/202/204.
- Every call is one HTTP request. Nothing is retried or replayed, a canceled context sends nothing, and acknowledgement of an asynchronous create/delete does not prove completion.
- Results are typed snapshots with an `Evidence` field (actual JSON object including null and unknown members, response headers, HTTP status).
- Extension JSON/query inputs (`With...Field`, `With...Query`) cannot override typed or server-owned fields; collisions fail before HTTP.
`FindIdentity` mirrors Resource.find (GET first, list fallback after 400/403/404); pass `resource.WithIdentityFindDetails(false)` to search the summary route like Python. `WaitFor`, `WaitForDeletion` and `Remove` come from the Collection.

Python comparison and differences: [python-parity-manila-a-round2.md](../python-parity-manila-a-round2.md).
