# Manila share transfers

`sharetransfers.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/sharetransfers` for the `/share-transfers` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

The caller owns the service client, including its Manila microversion. When `ServiceClient.Microversion` is set for a `shared-file-system` client the `X-OpenStack-Manila-API-Version` header is sent on every request; the library does not select or negotiate a version. Share transfers require a microversion that supports them on the deployment.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /share-transfers` | 202 only | Body wrapped in `transfer`; `ShareID` is required; an empty `Name` is omitted. The response model carries the one-time `AuthKey` |
| `Accept` | `POST /share-transfers/{id}/accept` | 202 only | Body `{"accept":{"auth_key":...,"clear_access_rules":true}}`; `AuthKey` is required; `ClearAccessRules: false` is omitted, so an explicit `false` cannot be sent |
| `Get` | `GET /share-transfers/{id}` | 200 | Typed model; a null `destination_project_id` decodes to `""` |
| `Delete` | `DELETE /share-transfers/{id}` | 200, 202, 204 | Native 404 stays an HTTP response error |
| `List`, `ListDetail` | `GET /share-transfers`, `GET /share-transfers/detail` | 200 | Typed filters plus `all_tenants`; zero values (including `AllTenants: false`) are omitted; paging advances `offset` by `limit` until an empty page |

Extension inputs use `WithCreateField`, `WithAcceptField`, `WithListQuery` and `WithListDetailQuery`. A field extension that names a core input field (including one whose typed value is omitted, such as `name` or `clear_access_rules`) fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. A query extension with the same key as a typed filter replaces the typed value.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup through `/share-transfers/detail?name=...` with local exact matching, duplicate-name errors, and ignore-missing deletion by default (`Remove` returns nil for a 404). Transfers expose no status field, so a `WithStatus` list filter is rejected before any request. `API.Get` and `API.Delete` themselves do not map a missing transfer to `resource.ErrNotFound`.

Acknowledgement of Create, Accept or Delete does not imply completion; read the transfer (`Accepted`) or the share explicitly. Cancelling a context stops the local call and does not roll back remote work; Create and Accept are never replayed automatically, and the auth key must be handled by the caller as a secret.
