# Blazar hosts

`hosts.API` wraps Gophercloud v2.15.0 `reservation/v1/hosts` for the `/os-hosts` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /os-hosts` | 201 only | Flat body: `name` plus every `ExtraCapabilities` entry at the top level; explicit `false`, `0`, `""` and `nil` (JSON null) are sent |
| `Get` | `GET /os-hosts/{id}` | 200 | Typed `Host`; unrecognized response keys populate `ExtraCapabilities`, `UpdatedAt` is nil for null |
| `Update` | `PUT /os-hosts/{id}` | 200 only | Flat body of the `ExtraCapabilities` entries; an empty map fails with `gophercloud.ErrMissingInput` before any request |
| `Delete` | `DELETE /os-hosts/{id}` | 204 only | 200 and 202 are rejected; native 404 stays an HTTP response error |
| `List` | `GET /os-hosts` | 200 | `ListOpts` has no fields; the response is one page and any `next` value is ignored |

Extension inputs use `WithCreateField`, `WithUpdateField` and `WithListQuery`. A field extension that names a core field (`name`) or a capability already in `ExtraCapabilities`, and nil options, fail with `resource.ErrInvalidOption` before any request. A capability named like a core field in `ExtraCapabilities` itself fails in the native builder, also before any request.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: ID lookup, ignore-missing deletion by default (`Remove` returns nil for a 404, other errors are kept) and `Collection.Get` mapping 404 to `resource.ErrNotFound`. `All` filters `status` locally instead of sending it on the wire. `API.Get` and `API.Delete` keep the native 404 error.

Blazar requires no microversion header from this package. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
