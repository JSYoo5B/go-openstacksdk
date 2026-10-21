# Manila share replicas

`replicas.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/replicas` for the `/share-replicas` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

The caller owns the service client, including its Manila microversion. When `ServiceClient.Microversion` is set for a `shared-file-system` client the `X-OpenStack-Manila-API-Version` header is sent on every request; the library does not select or negotiate a version.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /share-replicas` | 202 only | Body wrapped in `share_replica`; `ShareID` is required; empty optional fields and an empty `SchedulerHints` map are omitted |
| `Get` | `GET .../{id}` | 200 | Typed model; unknown response fields are not retained |
| `List`, `ListDetail` | `GET /share-replicas`, `GET /share-replicas/detail` | 200 | `ListOpts` filters `share_id`; zero `Limit`/`Offset`/`Marker` are omitted; paging advances `offset` by `limit` until an empty page |
| `Delete` | `DELETE .../{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `ForceDelete` | `POST .../{id}/action` | 202 only | Body `{"force_delete":null}` |
| `Resync` | `POST .../{id}/action` | 202 only | Body `{"resync":null}` |
| `Promote` | `POST .../{id}/action` | 202 only | Body `{"promote":{...}}`; `QuiesceWaitTime` 0 is omitted, so an explicit zero cannot be sent |
| `ResetStatus` | `POST .../{id}/action` | 202 only | Body `{"reset_status":{"status":...}}`; `Status` is required |
| `ResetState` | `POST .../{id}/action` | 202 only | Body `{"reset_replica_state":{"replica_state":...}}`; `State` is required |
| `ListExportLocations`, `GetExportLocation` | `GET .../{id}/export-locations[/{location}]` | 200 | Typed `ExportLocation` models |

Extension inputs use `WithCreateField`, `WithPromoteField`, `WithResetStatusField`, `WithResetStateField`, `WithListQuery` and `WithListDetailQuery`. A field extension that names a core input field (including one whose typed value is omitted) fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. A query extension with the same key as a typed filter replaces the typed value. `ForceDelete`, `Resync`, `Delete`, `Get` and the export-location reads take no options.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`. Replicas have no name, so lookups are by ID. `All` reads `/share-replicas/detail`; a `WithStatus` filter is applied locally and is not sent as a `status` query parameter. `Remove` ignores a 404 by default; `API.Delete`, `API.Get` and the other native calls keep a 404 as `gophercloud.ErrUnexpectedResponseCode`.

Acknowledgement of Create, Delete, ForceDelete, Resync, Promote, ResetStatus or ResetState does not imply completion; wait on `status` or `replica_state` explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.

## Python-parity helpers

`ForceRemove` posts `{"force_delete":null}` for one replica ID. A 404 is ignored by default and reported as `ErrNotFound` with `resource.WithMissingError`. See `../python-parity-a.md`.
