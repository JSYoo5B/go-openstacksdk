# Manila security services

`securityservices.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/securityservices` for the `/security-services` routes. Python openstacksdk Resource and workflow equivalence is tracked separately. The service client's `Microversion` is sent as `X-OpenStack-Manila-API-Version` for the native Manila client types; this package does not negotiate it.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /security-services` | 200 only | `Type` is required; optional string fields are omitted when empty, so an empty string cannot be sent |
| `Get` | `GET /security-services/{id}` | 200 | The model includes the `Password` the server returns |
| `Update` | `PUT /security-services/{id}` | 200 only | `Name` has no omit rule: a nil pointer is sent as JSON `null`. Other pointer fields omit nil and send `""` |
| `Delete` | `DELETE /security-services/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `List` | `GET /security-services/detail` | 200 | Single page; `ListOpts` filters, with false/empty filters omitted |

`UpdateOpts.Name` is the one field where omission is not expressible through typed options: leaving it nil sends `"name": null`. Use `WithUpdateField` only for non-core fields. `UpdateOpts.Type` is a plain string and is omitted when empty.

Extension inputs use `WithCreateField`, `WithUpdateField` and `WithListQuery`. A field extension that names a core input field fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. A missing `Type` fails local validation before any request.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup, duplicate-name errors, and ignore-missing deletion by default. `API.Get` and `API.Delete` do not add `resource.ErrNotFound` or ignore a missing security service.

Request bodies and responses can contain credentials (`password`, `user`); the SDK does not redact them, so avoid logging raw requests. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
