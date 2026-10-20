# Octavia flavor profiles

`flavorprofiles.API` wraps Gophercloud v2.15.0 `loadbalancer/v2/flavorprofiles` for the `/lbaas/flavorprofiles` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /lbaas/flavorprofiles` | 201, 202 | Body wrapped in `flavorprofile`; `Name`, `ProviderName` and `FlavorData` (a JSON string) are required |
| `Get` | `GET .../{id}` | 200 | Typed model; extension JSON is not retained |
| `Update` | `PUT .../{id}` | 200, 202 | Empty strings are omitted, not sent |
| `Delete` | `DELETE .../{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `List` | `GET /lbaas/flavorprofiles` | 200 | Follows `flavorprofiles_links`; `Fields` repeat the query key |

Extension inputs use `WithCreateField`, `WithUpdateField` and `WithListQuery`. A field extension that names a core input field fails with `resource.ErrInvalidOption` before any request; nil options fail the same way.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection` (exact name lookup, ignore-missing removal by default). `API.Delete` itself does not ignore a missing profile.

Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
