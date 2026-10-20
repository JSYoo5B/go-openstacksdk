# Zun capsules

`capsules.API` wraps Gophercloud v2.15.0 `container/v1/capsules` for the `/capsules` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /capsules` | 202 only | `TemplateOpts` is required; its `Bin` (JSON or YAML) is validated locally and sent unchanged as the string `template` |
| `Get` | `GET /capsules/{id}` | 200, 203 | Returns `any`: `*Capsule` for the legacy shape, `*CapsuleV132` when the body uses the 1.32 shape (object `restart_policy`) |
| `Delete` | `DELETE /capsules/{id}` | 202, 204 | Native 404 stays an HTTP response error; 200 is rejected |
| `List` | `GET /capsules` | 200 | `ListOpts` (`Marker`, `Limit`, `SortKey`, `SortDir`, `AllProjects`); follows the response `next` link |

`Create` returns the same `any` (`*Capsule` or `*CapsuleV132`). `List` always decodes rows as `*CapsuleV132`, so a page in the legacy shape (string `restart_policy`) fails with a decode error and yields no rows; use a client whose microversion returns 1.32-shaped capsules for listing. `AllProjects` is a plain bool: `false` is omitted from the query, not sent as `all_projects=false`.

Extension inputs use `WithCreateField` and `WithListQuery`. A field extension that names `template`, or a nil option, fails with `resource.ErrInvalidOption` before any request; a missing template or an unparsable template (`capsules.ErrInvalidDataFormat`) also fails before any request. There is no update route and no common `Collection`; the package has no `Find` or wait helpers.

Zun selects its microversion through the `OpenStack-API-Version: container <version>` header. The package sends it only when the `ServiceClient` has a `Microversion` (for example `1.32`); it does not choose a version, and the response shape follows what the server returns.

A 202 acknowledgement of Create or Delete does not imply completion: poll `Get` and its `Status` explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
