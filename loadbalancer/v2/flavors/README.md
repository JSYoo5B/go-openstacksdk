# Octavia flavors

`flavors.API` wraps Gophercloud v2.15.0 `loadbalancer/v2/flavors` for the `/lbaas/flavors` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /lbaas/flavors` | 201, 202 | Body wrapped in `flavor`; `Name` and `FlavorProfileId` are required |
| `Get` | `GET .../{id}` | 200 | Typed model; extension JSON is not retained |
| `Update` | `PUT .../{id}` | 200 only | Only non-empty `Name`, `Description` and `Enabled=true` are sent |
| `Delete` | `DELETE .../{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `List` | `GET /lbaas/flavors` | 200 | Follows `flavors_links`; `ListOpts.Enabled` is a pointer so `false` is sent; `Fields` repeat the query key |

Limitation: `CreateOpts.Enabled` and `UpdateOpts.Enabled` are plain bools with omit-empty, so `Enabled: false` is never sent (the server default applies on create, and an update cannot disable a flavor). `WithUpdateField("enabled", false)` is rejected with `resource.ErrInvalidOption` because `enabled` is a core field. Empty `Name` and `Description` strings are likewise omitted.

Extension inputs use `WithCreateField`, `WithUpdateField` and `WithListQuery`. A field extension that names a core input field fails before any request; nil options fail the same way.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection` (exact name lookup, ignore-missing removal by default). `API.Delete` itself does not ignore a missing flavor.

Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
