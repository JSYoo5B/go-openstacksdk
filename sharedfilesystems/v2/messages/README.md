# Manila user messages

`messages.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/messages` for the `/messages` routes. Messages require a Manila microversion that supports them (2.37 or newer); this package does not negotiate it, and the service client's `Microversion` is sent as `X-OpenStack-Manila-API-Version` for the native Manila client types. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Get` | `GET /messages/{id}` | 200 | Timestamps use the Manila no-zone format |
| `List` | `GET /messages` | 200 | Single page; `ListOpts` filters (`ID`, `ActionID`, `DetailID`, `MessageLevel`, `RequestID`, `ResourceID`, `ResourceType`, `SortKey`, `SortDir`, `Limit`); empty/zero filters are omitted; `next` links are not followed |
| `Delete` | `DELETE /messages/{id}` | 202, 204 | Native 404 stays an HTTP response error |

Extension inputs use `WithListQuery`; nil options fail with `resource.ErrInvalidOption` before any request.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`, including ignore-missing deletion by default (`Remove` returns nil for a 404). `API.Get` and `API.Delete` do not add `resource.ErrNotFound` or ignore a missing message.

`Limit` is only sent as a query value; the SDK does not continue past the first page. Cancelling a context stops the local call and does not roll back a delete; deletes are never replayed automatically.
