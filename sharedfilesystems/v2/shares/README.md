# Manila shares

`shares.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/shares` for the `/shares` routes of the Shared File System (Manila) API. Python openstacksdk Resource and workflow equivalence is tracked separately.

The client must be a Shared File System service client. When `ServiceClient.Microversion` is set, Gophercloud sends `X-Openstack-Manila-Api-Version`; when it is empty no header is sent. `shares.API` does not validate the selected version locally, so an operation that needs a newer version (for example `Revert` 2.27, `ListExportLocations` 2.9, `ListAccessRights` 2.7) reaches the service and fails there.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /shares` | 200, 201 | Body wrapped in `share`; `IsPublic` is a pointer so explicit `false` is sent; `Size` is always sent |
| `Get` | `GET /shares/{id}` | 200 | Typed model; unknown response fields are not retained |
| `Update` | `PUT /shares/{id}` | 200 | `DisplayName`, `DisplayDescription`, `IsPublic` are pointers; empty values are sent, unset values are omitted |
| `Delete` | `DELETE /shares/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `ListDetail` | `GET /shares/detail` | 200 | Offset paging only when `ListOpts.Limit` is set |
| `Extend`, `Shrink` | `POST /shares/{id}/action` | 202 | Bodies `{"extend":{"new_size":N}}` and `{"shrink":{"new_size":N}}`; `new_size` is sent even when 0 |
| `Revert` | `POST .../action` | 202 | `{"revert":{"snapshot_id":...}}` |
| `ResetStatus` | `POST .../action` | 202 | `{"reset_status":{"status":...}}` |
| `ForceDelete`, `Unmanage` | `POST .../action` | 202 | Bodies `{"force_delete":null}` and `{"unmanage":null}` |
| `GrantAccess` | `POST .../action` | 200 | `{"allow_access":{"access_type","access_to","access_level"}}`; an empty `AccessLevel` is sent as `""` |
| `RevokeAccess` | `POST .../action` | 200, 202 | `{"deny_access":{"access_id":...}}` |
| `ListAccessRights` | `POST .../action` | 200 | Body `{"access_list":null}`; an empty or null list decodes to no rows |
| `GetMetadata`, `GetMetadatum` | `GET .../metadata`, `.../metadata/{key}` | 200 | Returns the `metadata` or `meta` map |
| `SetMetadata`, `UpdateMetadata` | `POST .../metadata` | 200 | Both send the same POST; see below |
| `DeleteMetadatum` | `DELETE .../metadata/{key}` | 200 only | 202 and 204 are rejected |
| `ListExportLocations`, `GetExportLocation` | `GET .../export_locations[/{id}]` | 200 | `IsAdminOnly` and `Preferred` decode explicit `false` |

Extension inputs are body fields: `WithCreateField`, `WithUpdateField`, `WithExtendField`, `WithShrinkField`, `WithRevertField`, `WithResetStatusField`, `WithGrantAccessField`, `WithRevokeAccessField`. A field that names a core input field, and any nil option, fails with `resource.ErrInvalidOption` before any request. `WithListDetailQuery` appends a query parameter and replaces a typed filter that uses the same key rather than rejecting the collision.

Known limitations:

- `UpdateMetadata` uses POST in Gophercloud v2.15.0, identical on the wire to `SetMetadata`. The replace-all semantics of the Manila metadata update route are not verified. Extension field options on the metadata operations are merged into the `metadata` object rather than the request root and should not be used.
- `GrantAccess`, `RevokeAccess` and `ListAccessRights` use the legacy share actions. Access rules with locks, metadata or the 2.45+ endpoints are covered by `shareaccessrules`.
- `ListDetail` offset paging stops when a page is empty or no `Limit` is set.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup after an ID miss, duplicate-name errors, `status` passed as a query filter, `WithMaxItems` as a local cap, and ignore-missing deletion by default (`Remove` returns nil for a 404; `WithMissingError` reports it). `API.Delete` itself does not ignore a missing share.

Acknowledgement of Create, Extend, Shrink, Revert, ResetStatus, ForceDelete, Unmanage, Delete or access changes does not imply completion; wait on `status` (or the access rule `state`) explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically, including after a 5xx response or a deadline.
