# Octavia load balancers

`loadbalancers.API` wraps Gophercloud v2.15.0 `loadbalancer/v2/loadbalancers` for the `/lbaas/loadbalancers` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /lbaas/loadbalancers` | 201, 202 | Body wrapped in `loadbalancer`; explicit `false` for `AdminStateUp` is sent |
| `Get`, `GetStats`, `GetStatuses` | `GET .../{id}`, `/stats`, `/status` | 200 | Typed model; extension JSON is not retained in the model |
| `Update` | `PUT .../{id}` | 200, 202 | Pointer fields distinguish omitted from empty (`""`, `[]`, `false`) |
| `Delete` | `DELETE .../{id}` | 202, 204 | `DeleteOpts.Cascade` sends `cascade=true`; native 404 stays an HTTP response error |
| `Failover` | `PUT .../{id}/failover` | 202 only | No request body |
| `List` | `GET /lbaas/loadbalancers` | 200 | Follows `loadbalancers_links` next links; `ListOpts` booleans use pointers |

Extension inputs use `WithCreateField`, `WithUpdateField`, `WithListQuery` and `WithDeleteQuery`. A field extension that names a core input field fails with `resource.ErrInvalidOption` before any request; nil options fail the same way.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup, duplicate-name errors, `status` mapped to `provisioning_status`, and ignore-missing deletion by default (`Remove` returns nil for a 404). `API.Delete` itself does not ignore a missing load balancer.

Acknowledgement of Create, Update, Delete or Failover does not imply completion; wait on `provisioning_status` explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
