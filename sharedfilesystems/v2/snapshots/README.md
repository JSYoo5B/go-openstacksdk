# Manila share snapshots

`snapshots.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/snapshots` for the `/snapshots` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

The caller owns the service client, including its Manila microversion. When `ServiceClient.Microversion` is set for a `shared-file-system` client the `X-OpenStack-Manila-API-Version` header is sent on every request; the library does not select or negotiate a version. Gophercloud has no plain `List` for snapshots, so listing always uses the detail route.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /snapshots` | 200, 201, 202 | Body wrapped in `snapshot`; `ShareID` is required; empty `Name`, `Description`, `DisplayName` and `DisplayDescription` are omitted |
| `Get` | `GET /snapshots/{id}` | 200 | Typed model; unknown response fields are not retained |
| `Update` | `PUT /snapshots/{id}` | 200 only | `DisplayName` and `DisplayDescription` are pointers: omitted, `""` and non-empty values are distinguished; an empty `UpdateOpts` sends `{"snapshot":{}}` |
| `Delete` | `DELETE /snapshots/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `ForceDelete` | `POST /snapshots/{id}/action` | 202 only | Body `{"force_delete":null}` |
| `ResetStatus` | `POST /snapshots/{id}/action` | 202 only | Body `{"reset_status":{"status":...}}`; `Status` is not marked required upstream, so an empty status is sent as `""` and left to the server to reject |
| `ListDetail` | `GET /snapshots/detail` | 200 | Typed filters plus `all_tenants`; zero values (including `AllTenants: false`, `Size: 0`) are omitted; paging advances `offset` by `limit` until an empty page |

Extension inputs use `WithCreateField`, `WithUpdateField`, `WithResetStatusField` and `WithListDetailQuery`. A field extension that names a core input field (including one whose typed value is omitted) fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. A query extension with the same key as a typed filter replaces the typed value. `Get`, `Delete` and `ForceDelete` take no options.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup through `/snapshots/detail?name=...` with local exact matching, duplicate-name errors, `WithStatus` sent as the `status` query and also matched locally, and ignore-missing deletion by default (`Remove` returns nil for a 404). `API.Get` and `API.Delete` themselves do not map a missing snapshot to `resource.ErrNotFound`.

Acknowledgement of Create, Update, Delete, ForceDelete or ResetStatus does not imply completion; wait on `status` explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
