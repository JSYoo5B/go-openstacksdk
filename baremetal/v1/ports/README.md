# Bare Metal ports

`ports.API` wraps Gophercloud v2.15.0 `baremetal/v1/ports` for the `/v1/ports` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /ports` | 201, 202 | Flat JSON body; `PXEEnabled` and `IsSmartNIC` are `*bool`, so explicit `false` is sent |
| `Get` | `GET /ports/{id}` | 200 | Typed model; null `portgroup_uuid` decodes to `""` |
| `Update` | `PATCH /ports/{id}` | 200 only | JSON Patch array; 202 and 204 are rejected as unexpected status |
| `Delete` | `DELETE /ports/{id}` | 202, 204 | Native 404 stays a `gophercloud.ErrUnexpectedResponseCode` |
| `List` | `GET /ports` | 200 | Follows `ports_links` next links; `Fields` is allowed |
| `ListDetail` | `GET /ports/detail` | 200 | Same filters; `Fields` fails before HTTP |

`ListOpts` filters are `Node`, `NodeUUID`, `PortGroup`, `Address`, `Fields` (comma-separated), `Limit`, `Marker`, `SortKey` and `SortDir`. Zero values are omitted.

`UpdateOpts` is a slice of patches. `UpdateOperation` always serializes `op`, `path` and `value`: an explicit `false`, `""` or empty map is sent as is, and a `remove` operation without a value sends `"value": null`. `WithUpdateOptions` replaces the initial patch list rather than appending to it.

Extension inputs use `WithCreateField`, `WithListQuery` and `WithListDetailQuery`. A field extension that names a core create field fails with `resource.ErrInvalidOption` before any request, even when the core field is unset; nil options and empty query keys fail the same way. `CreateOpts` uses `omitempty` for strings and maps, so an empty `PhysicalNetwork` or `Extra` is not sent on create; use a JSON Patch update to set empty values.

`Find`, `All` and `Remove` use the common `resource.Collection`. Collection lookups use the detail route (`/ports/detail`), so they cannot use field selection. Ports declare no status attribute, so `resource.WithStatus` returns `resource.ErrUnsupported` without HTTP. `Remove` with an ID reference sends `DELETE` directly and ignores a 404 by default; `resource.WithMissingError()` turns it into `resource.ErrNotFound`. `API.Delete` and `API.Get` keep the native 404 error.

The client's `Microversion` is sent as `X-OpenStack-Ironic-API-Version` when the service client type is `baremetal`; the header is absent when no microversion is configured. Some fields depend on the Ironic API version; the library does not raise the version automatically, and the server decides what the selected version accepts.

Cancelling a context stops the local call and does not roll back remote work. A pre-canceled context makes no request, a deadline stays `context.DeadlineExceeded`, and mutations are never replayed automatically.
