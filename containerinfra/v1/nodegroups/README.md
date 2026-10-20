# Magnum node groups

`nodegroups.API` wraps Gophercloud v2.15.0 `containerinfra/v1/nodegroups` for the `/clusters/{cluster}/nodegroups` routes. `InCluster(ctx, parent)` fixes the parent cluster; an explicit `resource.ID` needs no lookup request, a `resource.Name` is resolved through the cluster list. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST .../nodegroups` | 202 only | `Name` is required. Pointer fields (`NodeCount`, `MaxNodeCount`, `DockerVolumeSize`, `MergeLabels`) send explicit `0` and `false`; `MinNodeCount` is a plain int, so `0` is omitted |
| `Get` | `GET .../nodegroups/{id}` | 200 | Typed `NodeGroup`; `MaxNodeCount` and `DockerVolumeSize` stay pointers so `null` and `0` differ |
| `Update` | `PATCH .../nodegroups/{id}` | 202 only | JSON Patch array (`[]UpdateOpts`); `Value` of `0` and `false` is sent. 200 is rejected. No extension fields |
| `Delete` | `DELETE .../nodegroups/{id}` | 202, 204 | Native 404 stays an HTTP response error; 200 is rejected |
| `List` | `GET .../nodegroups` | 200 | `ListOpts` (`Marker` is numeric, `Limit`, `SortKey`, `SortDir`, `Role`); follows the response `next` link |

Extension inputs use `WithCreateField` and `WithListQuery`. A field extension that names a core input field, or a nil option, fails with `resource.ErrInvalidOption` before any request; missing required inputs and a patch without `Op` or `Path` also fail before any request.

The scope's `Find`, `All`, `Delete`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup, duplicate-name errors and ignore-missing deletion by default (the scope's `Delete` returns nil for a 404). `API.Get` and `API.Delete` keep the native 404 error.

The package sends `OpenStack-API-Version: container-infra <version>` only when the `ServiceClient` has a `Microversion` and adds no version-dependent behavior. A 202 acknowledgement of Create, Update or Delete does not imply completion: wait on `Status` explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
