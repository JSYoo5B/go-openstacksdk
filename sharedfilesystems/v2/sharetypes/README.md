# Manila share types

`sharetypes.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/sharetypes` for the `/types` routes. Python openstacksdk Resource and workflow equivalence is tracked separately. The service client's `Microversion` is sent as `X-OpenStack-Manila-API-Version` for the native Manila client types (`shared-file-system`, `sharev2`, `share`); this package does not negotiate or validate it.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /types` | 200, 202 | Body wrapped in `share_type`; `IsPublic` is always sent, including `false` |
| `Get`, `GetDefault` | `GET /types/{id}`, `/types/default` | 200 | A legacy `share_type_access:is_public` key, when present, overrides `IsPublic` |
| `Update` | `PUT /types/{id}` | 200 only | `Name`, `Description`, `IsPublic` are pointers: nil omits, `""`/`false` are sent |
| `Delete` | `DELETE /types/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `List` | `GET /types` | 200 | Single page; `ListOpts.IsPublic` is a string, empty omits it |
| `GetExtraSpecs`, `SetExtraSpecs` | `GET`/`POST /types/{id}/extra_specs` | 200 (set: 200, 202) | Set values may be `false`, `""` or `null` |
| `UnsetExtraSpecs` | `DELETE /types/{id}/extra_specs/{key}` | 202, 204 | The key is placed in the path unescaped |
| `ShowAccess` | `GET /types/{id}/share_type_access` | 200 | Returns share type and project IDs |
| `AddAccess`, `RemoveAccess` | `POST /types/{id}/action` | 200, 202 | `addProjectAccess` / `removeProjectAccess`; an empty project is sent as `""` |

Limits inherited from Gophercloud: `CreateOpts.ExtraSpecs.DriverHandlesShareServers` is required, so its zero value `false` fails local validation before any request and cannot be sent through the typed options. `SnapshotSupport` is a pointer, so an explicit `false` is sent when `DriverHandlesShareServers` is true. `Get`, `GetDefault`, `Create` and `Update` decode a `share_type` envelope; a body without it is not a supported response. Unknown response fields are not retained.

Extension inputs use `WithCreateField`, `WithUpdateField`, `WithSetExtraSpecsField`, `WithAddAccessField`, `WithRemoveAccessField` and `WithListQuery`. A field extension that names a core input field fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. For `SetExtraSpecs` an extension field is merged into the `extra_specs` map, so it adds a spec key and collides only with a key already in `ExtraSpecs`.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup, and ignore-missing deletion by default (`Remove` returns nil for a 404). `API.Get` and `API.Delete` do not add `resource.ErrNotFound` or ignore a missing share type.

Share types are administrator-controlled; policy failures surface as HTTP response errors. An acknowledged mutation is not rolled back by cancelling the context, and uncertain mutations are never replayed automatically.

## Python-parity helpers

`UnsetExtraSpecProperty` deletes one extra spec and reports a 404 as `ErrNotFound` unless `resource.WithIgnoreMissing` is given. `CreateWithExtraSpecs` sends the extra specs verbatim, so `driver_handles_share_servers=false`, null and empty values are preserved; the typed `Create` cannot send `false` for that required field. See `../python-parity-a.md`.
