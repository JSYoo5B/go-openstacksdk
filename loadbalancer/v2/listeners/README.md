# Octavia listeners

`listeners.API` wraps Gophercloud v2.15.0 `loadbalancer/v2/listeners` for the `/lbaas/listeners` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /lbaas/listeners` | 201, 202 | Body wrapped in `listener`; pointer fields (`AdminStateUp`, `ConnLimit`, timeouts) send explicit `false` and `0` |
| `Get`, `GetStats` | `GET .../{id}`, `.../{id}/stats` | 200 | Typed model; extension JSON is not retained in the model |
| `Update` | `PUT .../{id}` | 200, 202 | Pointer fields distinguish omitted from empty (`""`, `[]`, `false`, `0`); empty `DefaultPoolID`, `ALPNProtocols` and `TLSVersions` are sent as JSON `null` |
| `Delete` | `DELETE .../{id}` | 202, 204 | No options; native 404 stays an HTTP response error |
| `List` | `GET /lbaas/listeners` | 200 | Follows `listeners_links` next links; `ListOpts` booleans and timeouts use pointers so `false` and `0` are sent |

Extension inputs use `WithCreateField`, `WithUpdateField` and `WithListQuery`. A field extension that names a core input field fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. `Delete`, `Get` and `GetStats` take no extensions.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup, duplicate-name errors, and ignore-missing deletion by default (`Remove` returns nil for a 404). `API.Delete` itself does not ignore a missing listener.

Limitations:

- `CreateOpts` non-pointer fields with `omitempty` (for example `Name`, `HSTSMaxAge`, `HSTSPreload`) cannot send an explicit empty, zero or `false` value on create.
- `WithStatus` is applied locally to `provisioning_status`; it is not sent as a query parameter to the service.
- Nested `DefaultPool` and `L7Policies` create bodies follow the native Gophercloud builders; the SDK adds no validation to them.

Acknowledgement of Create, Update or Delete does not imply completion; wait on `provisioning_status` explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
