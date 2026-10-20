# Designate TSIG keys

`tsigkeys.API` wraps Gophercloud v2.15.0 `dns/v2/tsigkeys` for the `/tsigkeys` routes. Designate bodies are flat JSON (no envelope). Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /tsigkeys` | 201 only | `Name`, `Algorithm`, `Secret`, `Scope` and `ResourceID` are all required; missing values fail before HTTP |
| `Get` | `GET /tsigkeys/{id}` | 200 | Typed model, including `Secret`; native 404 stays an HTTP response error |
| `Update` | `PATCH /tsigkeys/{id}` | 200 only | Only non-empty fields are sent; a zero `UpdateOpts` sends `{}`. An empty string cannot be sent through typed fields |
| `Delete` | `DELETE /tsigkeys/{id}` | 204 only | Native 404 stays an HTTP response error |
| `List` | `GET /tsigkeys` | 200 | Filters `name`, `algorithm`, `scope`, `limit`, `marker`; follows `links.next` |

Extension inputs use `WithCreateField`, `WithUpdateField` and `WithListQuery`; values `false` and `nil` (JSON `null`) are sent as given. There are no header options. A field extension that names a core JSON field and nil options fail with `resource.ErrInvalidOption` before any request.

`Find`, `All` and `Remove` use the common `resource.Collection`: exact name lookup through the `name` query, duplicate-name errors, and ignore-missing deletion by default (`Remove` returns nil for a 404). TSIG keys have no status, so the common status list filter fails with `resource.ErrUnsupported` before HTTP.

Responses include the key `Secret`; callers should treat returned values as sensitive. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
