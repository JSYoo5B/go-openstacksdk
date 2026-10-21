# Bare Metal portgroups

`portgroups.API` is an SDK-owned binding for `/v1/portgroups`. Gophercloud v2.15.0 has no binding for this resource. It is reached with `New(client)` or `baremetal.Service.PortGroups()`. Python openstacksdk Resource semantics are compared in [python-operations-r2-ironic-a.md](../python-operations-r2-ironic-a.md).

| Operation | Route | Success code | Notes |
|---|---|---|---|
| `Create` | `POST /portgroups` | 201 only | Typed `CreateOpts` plus `WithCreateField` extensions |
| `Get` | `GET .../{id}` | 200 only | `WithGetFields` sends `fields=a,b` (wire names) |
| `Update` | `PATCH .../{id}` | 200 only | JSON Patch; empty patch fails before HTTP |
| `Delete` | `DELETE .../{id}` | 204 only | A 404 stays a native error |
| `List` / `ListDetail` | `GET ...` / `GET /portgroups/detail` | 200 only | Follows advertised `next` links; `Fields` is rejected for detailed lists |
| `FindIdentity` | GET by id, then list scan | | name or uuid; default ignore-missing |
| `Find`, `All`, `Remove` | shared `resource.Collection` | | `Remove` by ID ignores 404 unless `resource.WithMissingError()` |

Typed create fields: `Name`, `Address`, `Mode`, `StandalonePortsSupported` (`*bool`), `Properties`, `Extra`, `NodeUUID`, `UUID`. Pointer fields and maps are omitted only when nil, so explicit `false`, `0`, `""` and `{}` are sent. Required string fields are omitted when empty and the server reports them. `WithCreateField` adds root body fields; a typed field or a server-owned field (`created_at`, `updated_at`, `links` and read-only collections) fails with `resource.ErrInvalidOption` before HTTP. `Node` and `Address` list filters.

`UpdateOpts.Operations` is a JSON Patch list built with `Add`, `Replace` and `Remove`. `Add` and `Replace` always send `value` (nil is JSON null); `Remove` sends none. Operations other than add/replace/remove, paths not starting with `/`, and server-owned targets fail before HTTP. `FieldPatches(map)` builds the sorted add list that Python `update_*(**attrs)` sends. `WithUpdateOptions` replaces the list and `WithUpdateOperations` appends.

List filters are `Fields`, `Limit`, `Marker`, `SortKey`, `SortDir` (zero values omitted, negative limit invalid); `WithListQuery` adds extension queries and rejects keys owned by typed filters. With an explicit limit and no advertised next link the iterator asks once more with the last uuid as `marker`, like Python.

Responses keep typed fields plus `Body` (original object, exact numbers), `Header` and `StatusCode`. The client's `Microversion` is sent as `X-OpenStack-Ironic-API-Version` for a baremetal-typed client (1.26 is the highest Python negotiates for this resource); it is never raised automatically. Cancelling a context stops only the local call; accepted mutations are never replayed.
