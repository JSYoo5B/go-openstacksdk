# Octavia health monitors

`monitors.API` wraps Gophercloud v2.15.0 `loadbalancer/v2/monitors` for the `/lbaas/healthmonitors` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /lbaas/healthmonitors` | 201, 202 | Body wrapped in `healthmonitor`; `AdminStateUp` is a pointer so explicit `false` is sent; `HTTPVersion` is a string sent as a JSON number (`"1.1"` becomes `1.1`) |
| `Get` | `GET .../{id}` | 200 | Typed model; `HTTPVersion` is rebuilt from the numeric field with one decimal; extension JSON is not retained |
| `Update` | `PUT .../{id}` | 200, 202 | `Name`, `DomainName`, `HTTPVersion` and `AdminStateUp` are pointers and keep empty/false values; `Delay`, `Timeout`, `MaxRetries`, `MaxRetriesDown`, `URLPath`, `HTTPMethod`, `ExpectedCodes` and `Tags` use omit-empty, so zero values cannot be sent. An empty `UpdateOpts` sends `{"healthmonitor":{}}` |
| `Delete` | `DELETE .../{id}` | 202, 204 | No query options; native 404 stays an HTTP response error |
| `List` | `GET /lbaas/healthmonitors` | 200 | Follows `healthmonitors_links` next links; `ListOpts.AdminStateUp` is a pointer; `Tags` repeat the query key |

Extension inputs use `WithCreateField`, `WithUpdateField` and `WithListQuery`. A field extension that names a core input field fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. Required Create inputs (`Type`) and a non-numeric `HTTPVersion` also fail locally without a request. A zero `Delay`, `Timeout` or `MaxRetries` is not rejected locally and is sent as `0`.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup, duplicate-name errors, ignore-missing removal by default (`Remove` returns nil for a 404) and `provisioning_status` as the wait status. `API.Delete` itself does not ignore a missing monitor.

Acknowledgement of Create, Update or Delete does not imply completion; wait on `provisioning_status` explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
