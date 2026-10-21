# Share instances

SDK-owned Manila binding (absent from Gophercloud v2.15.0). Create it with `New(client)` or through the accessor on the shared-file-system `Service`.

- Microversion floor: 2.3. A numeric selected version below it, or no selected version, fails with `resource.ErrUnsupported` before any request; `latest` is passed through. The SDK never upgrades the selected version.
- Routes: `GET /share_instances`, `GET /share_instances/{id}`, `POST /share_instances/{id}/action` for `reset_status` and `force_delete`. Actions accept 200/202/204.
- Every call is one HTTP request. Nothing is retried or replayed, a canceled context sends nothing, and acknowledgement of an asynchronous create/delete does not prove completion.
- Results are typed snapshots with an `Evidence` field (actual JSON object including null and unknown members, response headers, HTTP status).
- Extension JSON/query inputs (`With...Field`, `With...Query`) cannot override typed or server-owned fields; collisions fail before HTTP.
`Remove` force-deletes and ignores a missing instance by default; it is administrator-only on the service.

Python comparison and differences: [python-parity-manila-a-round2.md](../python-parity-manila-a-round2.md).
