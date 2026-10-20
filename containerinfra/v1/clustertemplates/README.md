# Magnum cluster templates

`clustertemplates.API` wraps Gophercloud v2.15.0 `containerinfra/v1/clustertemplates` for the `/clustertemplates` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /clustertemplates` | 201 only | `COE` and `ImageID` are required. Pointer fields (`Public`, `Hidden`, `FloatingIPEnabled`, `TLSDisabled`, `DockerVolumeSize`, ...) send explicit `false` and `0`; omitted pointers and empty strings are not sent |
| `Get` | `GET /clustertemplates/{id}` | 200 | Typed `ClusterTemplate`; unknown response fields are not retained |
| `Update` | `PATCH /clustertemplates/{id}` | 200, 202 | JSON Patch array (`[]UpdateOpts`); `Value` of `false` and `""` is sent, `RemoveOp` omits the value. Returns the updated template. No extension fields |
| `Delete` | `DELETE /clustertemplates/{id}` | 202, 204 | Native 404 stays an HTTP response error; 200 is rejected |
| `List` | `GET /clustertemplates` | 200 | `ListOpts` (`Marker`, `Limit`, `SortKey`, `SortDir`); follows the response `next` link |

Extension inputs use `WithCreateField` and `WithListQuery`. A field extension that names a core input field, or a nil option, fails with `resource.ErrInvalidOption` before any request; missing required inputs and a patch without `Op` or `Path` also fail before any request.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup, duplicate-name errors and ignore-missing deletion by default. `API.Get` and `API.Delete` keep the native 404 error; `Collection.Get` maps it to `resource.ErrNotFound`.

The package sends `OpenStack-API-Version: container-infra <version>` only when the `ServiceClient` has a `Microversion` and adds no version-dependent behavior. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
