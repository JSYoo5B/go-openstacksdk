# Trove datastores

`datastores.API` wraps Gophercloud v2.15.0 `db/v1/datastores` for the read-only `/datastores` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `List` | `GET /datastores` | 200 (204 is an empty list) | Single page: `datastores_links` is ignored; no query parameters exist |
| `Get` | `GET /datastores/{id}` | 200 | Typed `Datastore` with embedded `Versions`; native 404 stays an HTTP response error |
| `ListVersions` | `GET /datastores/{id}/versions` | 200 (204 is an empty list) | Single page |
| `GetVersion` | `GET /datastores/{id}/versions/{version}` | 200 | Typed `Version`; native 404 stays an HTTP response error |

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` are generated from the common `resource.Collection`. `Find` matches an exact name over the full list (`resource.ErrAmbiguous` for duplicates, `resource.ErrNotFound` when missing). `resource.WithQuery` on `All` returns `resource.ErrUnsupported` before any request. The pinned API has no datastore delete, so `Remove` and `WaitForDeletion` have no remote effect to rely on and are not covered by a behavior contract.

Models are Gophercloud's; response fields they do not declare are not retained. Cancelling a context stops the local call. Reads are the only operations in this package, and none is replayed automatically.
