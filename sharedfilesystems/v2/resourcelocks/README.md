# Resource locks

SDK-owned Manila binding (absent from Gophercloud v2.15.0). Create it with `New(client)` or through the accessor on the shared-file-system `Service`.

- Microversion floor: 2.81. A numeric selected version below it, or no selected version, fails with `resource.ErrUnsupported` before any request; `latest` is passed through. The SDK never upgrades the selected version.
- Routes: `POST/GET /resource-locks`, `GET/PUT/DELETE /resource-locks/{id}`. Create 200, Update/Get/List 200, Delete 200/202/204.
- Every call is one HTTP request. Nothing is retried or replayed, a canceled context sends nothing, and acknowledgement of an asynchronous create/delete does not prove completion.
- Results are typed snapshots with an `Evidence` field (actual JSON object including null and unknown members, response headers, HTTP status).
- Extension JSON/query inputs (`With...Field`, `With...Query`) cannot override typed or server-owned fields; collisions fail before HTTP.
Collection helpers: `Find`, `FindIdentity`, `All`, `Remove` (ignores a missing lock by default), `WaitForDeletion`. `ListOpts` pointers preserve `limit=0` and `with_count=false`; `count` of a `with_count` response is only in the raw response, not surfaced.

Python comparison and differences: [python-parity-manila-a-round2.md](../python-parity-manila-a-round2.md).
