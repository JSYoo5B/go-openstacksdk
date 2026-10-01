# Masakari notifications

Python `conn.instance_ha.notifications`, `get_notification` and
`create_notification` correspond to Go `List`/typed `All`, `Get` and `Create`.

```go
api := notifications.New(client)
notification, err := api.Create(ctx, notifications.CreateOpts{
    Type: "COMPUTE_HOST", Hostname: "compute-01",
    GeneratedTime: time.Now().UTC().Format(time.RFC3339),
}, notifications.WithCreatePayload(map[string]any{
    "event": "STOPPED", "host_status": "UNKNOWN", "cluster_status": "OFFLINE",
}))
if err != nil { return err }
notification, err = api.Wait(ctx, resource.ID(notification.UUID), "finished",
    resource.WithUnlimitedWait(), resource.WithFailureStates("error", "failed"))
```

`UUID` maps to wire `notification_uuid`; `ID` retains the separate raw database
ID. This resource has no name, update or delete endpoint. Name references and
unsupported mutations fail before HTTP. Get/deletion monitoring use stable UUIDs.

Create accepts object payload JSON. `WithCreatePayload` snapshots a Go object
immediately; `WithCreateOptions` snapshots RawMessage input without a float64
round trip. The server validates failure-specific payload semantics and types
(`PROCESS`, `COMPUTE_HOST`, `VM`). Core/read-only fields and SDK headers are
protected from extensions. `Hostname` serializes as `hostname`, not the API
reference's inconsistent `host_name` table label.

`ListOpts` includes sorting, limit/marker, source host UUID, type, status and
generated since. The last maps to `generated-since`. `WithListQuery` adds explicit
server-validated filters but cannot replace concrete fields. Shared options are
available through `Resources.List/All`. Explicit body/HTTP continuation links
retain collection, origin and filters; without a link iteration stops at the page.
An explicit marker may be supplied; automatic marker fallback remains unverified.

Selected numeric 1.N is preserved, with empty selection meaning 1.0. Get returns
workflow details when the server's selected version supports them (>=1.1); it
does not upgrade or perform a separate invented details request. Models preserve
unknown fields, null/omission, large IDs, payload JSON and original timestamp
strings. Workflow and progress-detail items retain their own raw fields; progress
may be a number or string and is not forced to an integer percentage.

Shared waits default to five minutes and two-second polling. Python's status
wait defaults to unlimited; use `WithUnlimitedWait` to request it. Go detects
`error` and `failed`; `WithFailureStates("ERROR")` selects Python's default.
Status comparison is case-insensitive; parent context cancellation and original
HTTP causes are preserved. The generic progress callback reports zero when no
top-level progress field exists, rather than inventing aggregate workflow progress.
Accepted decode errors retain body/header/status in `resource.ResponseError`.

Sources: pinned [Notification](https://opendev.org/openstack/openstacksdk/src/commit/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/instance_ha/v1/notification.py)
and [Masakari API reference](https://docs.openstack.org/api-ref/instance-ha/).
