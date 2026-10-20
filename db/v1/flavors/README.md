# Trove flavors

`flavors.API` wraps Gophercloud v2.15.0 `db/v1/flavors` for the read-only `/flavors` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `List` | `GET /flavors` | 200 (204 is an empty list) | Follows `flavors_links`; no filter parameters exist |
| `Get` | `GET /flavors/{id}` | 200 | Typed `Flavor`; native 404 stays an HTTP response error |

`Flavor.ID` is an integer and `Flavor.StrID` carries the string identifier; `RAM` of `0` is a real value. `Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` are generated from the common `resource.Collection`: `Find` with an ID reference calls `Get`, with a name reference matches an exact name over the full list (`resource.ErrAmbiguous` for duplicates, `resource.ErrNotFound` when missing), and `resource.WithQuery` on `All` returns `resource.ErrUnsupported` before any request. The pinned API has no flavor delete, so `Remove` and `WaitForDeletion` are not covered by a behavior contract.

Models are Gophercloud's; response fields they do not declare are not retained. Cancelling a context stops the local call. Reads are never replayed automatically.
