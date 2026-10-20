# Designate zone transfer requests

`request.API` wraps Gophercloud v2.15.0 `dns/v2/transfer/request`. Designate bodies are flat JSON (no envelope). Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /zones/{zone}/tasks/transfer_requests` | 201, 202 | Empty `TargetProjectID` and `Description` are omitted; zero options send `{}` |
| `Get` | `GET /zones/tasks/transfer_requests/{id}` | 200 | Typed model including the transfer `Key`; native 404 stays an HTTP response error |
| `Update` | `PATCH /zones/tasks/transfer_requests/{id}` | 200, 202 | Only non-empty fields are sent; an empty string cannot be sent through typed fields |
| `Delete` | `DELETE /zones/tasks/transfer_requests/{id}` | 204 only | Native 404 stays an HTTP response error |
| `List` | `GET /zones/tasks/transfer_requests` | 200 | Filter `status`; follows `links.next` |

Extension inputs use `WithCreateField`, `WithUpdateField` and `WithListQuery`; values `false` and `nil` (JSON `null`) are sent as given. There are no header options. A field extension that names a core JSON field and nil options fail with `resource.ErrInvalidOption` before any request.

`All`, `Find`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`, with ignore-missing deletion by default (`Remove` returns nil for a 404).

The returned `Key` is the secret a receiving project passes to a transfer accept. A 201 or 202 acknowledges the request and does not show that the receiving project has accepted it; read `Status` explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
