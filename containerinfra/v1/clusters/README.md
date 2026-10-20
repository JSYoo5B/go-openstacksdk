# Magnum clusters

`clusters.API` wraps Gophercloud v2.15.0 `containerinfra/v1/clusters` for the `/clusters` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /clusters` | 202 | Returns the new cluster UUID, not the cluster. `CreateTimeout` and `Name` are always serialized by the native builder, so omitted values go out as `"create_timeout": null` and `"name": ""`; explicit `0` and `false` pointers are sent as given |
| `Get` | `GET /clusters/{id}` | 200 | Typed `Cluster`; unknown response fields are not retained |
| `Update` | `PATCH /clusters/{id}` | 200, 202 | JSON Patch array (`[]UpdateOpts`); `Value` of `0`, `false` and `""` is sent, `RemoveOp` omits the value. Returns the UUID. No extension fields |
| `Resize` | `POST /clusters/{id}/actions/resize` | 200, 202 | `NodeCount` is a required pointer, so `0` is sent; empty `NodesToRemove` and `NodeGroup` are omitted. Returns the UUID |
| `Upgrade` | `POST /clusters/{id}/actions/upgrade` | 202 only | `MaxBatchSize` defaults to `1` when nil; an explicit `0` is sent. Returns the UUID |
| `Delete` | `DELETE /clusters/{id}` | 202, 204 | Native 404 stays an HTTP response error; 200 is rejected |
| `List`, `ListDetail` | `GET /clusters`, `GET /clusters/detail` | 200 | `ListOpts` (`Marker`, `Limit`, `SortKey`, `SortDir`); follows the response `next` link |

Extension inputs use `WithCreateField`, `WithResizeField`, `WithUpgradeField`, `WithListQuery` and `WithListDetailQuery`. A field extension that names a core input field, or a nil option, fails with `resource.ErrInvalidOption` before any request. Missing required inputs (`ClusterTemplateID`, `NodeCount`, `ClusterTemplate`, patch `Op`/`Path`) also fail before any request. An extension query replaces a typed query value of the same name.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup, duplicate-name errors, ignore-missing deletion by default, and `Collection.Get` mapping 404 to `resource.ErrNotFound`. Collection listing uses the `/clusters/detail` route and filters `status` locally instead of sending it on the wire. `API.Get` and `API.Delete` themselves keep the native 404 error.

Magnum selects its microversion through the `OpenStack-API-Version: container-infra <version>` header. The package sends it only when the `ServiceClient` has a `Microversion`; it does not choose a version and no version-dependent field is added or removed.

A 202 acknowledgement from Create, Update, Resize, Upgrade or Delete does not imply completion: wait on `Status` (for example `CREATE_COMPLETE`, `UPDATE_COMPLETE`) explicitly. Cancelling a context stops the local call and does not roll back remote work; a failed or timed-out mutation is never replayed automatically.
