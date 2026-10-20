# Designate zone transfer accepts

`accept.API` wraps Gophercloud v2.15.0 `dns/v2/transfer/accept`. Designate bodies are flat JSON (no envelope). Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /zones/tasks/transfer_accepts` | 201, 202 | `Key` and `ZoneTransferRequestID` are required; missing values fail before HTTP |
| `Get` | `GET /zones/tasks/transfer_accepts/{id}` | 200 | Typed model; native 404 stays an HTTP response error |
| `List` | `GET /zones/tasks/transfer_accepts` | 200 | Filter `status`; follows `links.next` |

There is no Update or Delete for transfer accepts. Extension inputs use `WithCreateField` and `WithListQuery`; values `false` and `nil` (JSON `null`) are sent as given. There are no header options. A field extension that names a core JSON field and nil options fail with `resource.ErrInvalidOption` before any request.

`All` and `Find` use the common `resource.Collection`.

A 201 or 202 acknowledges the accept task and does not prove that the zone has changed owner; read `Status` explicitly. Cancelling a context stops the local call and does not roll back remote work; a failed or uncertain accept is never replayed automatically.
