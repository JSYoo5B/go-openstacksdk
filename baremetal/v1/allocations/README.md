# Bare Metal allocations

`allocations.API` wraps Gophercloud v2.15.0 `baremetal/v1/allocations` for the `/v1/allocations` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /allocations` | 201, 202 | Flat JSON body; `ResourceClass` is required, empty optional fields are omitted |
| `Get` | `GET /allocations/{id}` | 200 | Typed model; JSON `null` strings decode to `""` and null timestamps to the zero time |
| `Delete` | `DELETE /allocations/{id}` | 202, 204 | Native 404 stays a `gophercloud.ErrUnexpectedResponseCode` |
| `List` | `GET /allocations` | 200 | Follows `allocations_links` next links |

`ListOpts` filters are `Node`, `ResourceClass`, `State`, `Fields` (comma-separated), `Limit`, `Marker`, `SortKey` and `SortDir`. Zero values are omitted, so an empty string cannot be sent as a filter.

Extension inputs use `WithCreateField` and `WithListQuery`. A field extension that names a core create field fails with `resource.ErrInvalidOption` before any request, even when the core field is unset; nil options and empty query keys fail the same way. A missing `ResourceClass` is rejected before HTTP by the native builder.

`CreateOpts` uses `omitempty` for every optional field, so an explicitly empty `Name`, `CandidateNodes`, `Traits` or `Extra` is not sent. There is no update operation in the pinned Gophercloud package.

Allocation creation is asynchronous. A 201 or 202 response only acknowledges the request: a new allocation is normally returned with `State` `allocating` and no `NodeUUID`. Poll `Get` until `State` is `active` or `error` (`LastError` carries the failure); this package does not provide an allocation waiter. Deleting an allocation is not a completion signal either.

`Find`, `All` and `Remove` use the common `resource.Collection` (ID lookup, exact name lookup, duplicate-name errors). `Remove` with an ID reference sends `DELETE` directly and ignores a 404 by default; `resource.WithMissingError()` turns it into `resource.ErrNotFound`. `API.Delete` itself keeps the native 404 error. `All` supports `resource.WithPaginated(false)` and `resource.WithMaxItems`.

The client's `Microversion` is sent as `X-OpenStack-Ironic-API-Version` when the service client type is `baremetal`; the header is absent when no microversion is configured. The library does not raise the version automatically, and the server decides whether the selected version is accepted.

Cancelling a context stops the local call and does not roll back remote work. A pre-canceled context makes no request, a deadline stays `context.DeadlineExceeded`, and mutations are never replayed automatically.
