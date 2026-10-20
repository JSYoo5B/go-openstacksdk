# Trove configuration groups

`configurations.API` wraps Gophercloud v2.15.0 `db/v1/configurations` for `/configurations` and the datastore parameter routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /configurations` | 200 only | Body wrapped in `configuration`; `Name` and `Values` are required before any request. `Values` entries such as `false` and `0` are sent as given; an empty `Description` and a nil `Datastore` are omitted |
| `Get` | `GET /configurations/{id}` | 200 | Typed `Config`; native 404 stays an HTTP response error |
| `Update` | `PATCH /configurations/{id}` | 200, 202, 204 | Partial body: empty `Name`, nil `Values` and nil `Description` are omitted; a non-nil `Description` pointer to `""` is sent |
| `Replace` | `PUT /configurations/{id}` | 201, 202 | Same body builder as `Update`; a zero `UpdateOpts` sends an empty `configuration` object |
| `Delete` | `DELETE /configurations/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `List` | `GET /configurations` | 200 (204 is an empty list) | Single page: `configurations_links` is ignored |
| `ListInstances` | `GET /configurations/{id}/instances` | 200 | Follows `instances_links`; yields `instances.Instance` values |
| `ListDatastoreParams`, `GetDatastoreParam` | `GET /datastores/{datastore}/versions/{version}/parameters[/{name}]` | 200 | `Param.Min`/`Max` are floats; zero cannot be told apart from absent |
| `ListGlobalParams`, `GetGlobalParam` | `GET /datastores/versions/{version}/parameters[/{name}]` | 200 | Same `Param` model |

Extension inputs use `WithCreateField`, `WithUpdateField` and `WithReplaceField`, merged into the `configuration` object. A field extension that names an input already set fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. Gophercloud has no extension point for list or delete queries, so none is offered.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup over the full list, duplicate-name errors (`resource.ErrAmbiguous`), `resource.ErrNotFound` for a missing reference, and ignore-missing deletion by default (`Remove` returns nil for a 404 unless `resource.WithMissingError()` is passed). Configurations have no status attribute, so `resource.WithStatus` and `resource.WithQuery` on `All` return `resource.ErrUnsupported` before any request. `API.Delete` itself does not ignore a missing configuration.

Acknowledgement of a mutation is not completion, and attaching a configuration to an instance is a separate instance operation. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
