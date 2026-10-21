# Octavia availability zone profiles

`availabilityzoneprofiles.API` is an SDK-owned binding for `/lbaas/availabilityzoneprofiles` (Octavia API 2.14 and later). Gophercloud v2.15.0 has no binding for these routes. It is reached through `loadbalancer/v2.Service.AvailabilityZoneProfiles()`. Every operation requires administrative privileges.

Profiles are identified by UUID (`id`). `AvailabilityZoneData` is a JSON document encoded as a **string** and is sent and returned verbatim; it is never parsed.

| Operation | Route | Success | Notes |
|---|---|---|---|
| `Create` | `POST .../availabilityzoneprofiles` | 201 | Body wrapped in `availability_zone_profile`. `Name`, `ProviderName` and `AvailabilityZoneData` are all required. |
| `Get` | `GET .../{id}` | 200 | ID validated as one unescaped path segment. |
| `Update` | `PUT .../{id}` | 200 | `Name`, `ProviderName`, `AvailabilityZoneData` are optional pointers: nil is omitted, an empty string is sent. An update with no field and no extension is rejected locally. |
| `Delete` | `DELETE .../{id}` | 204 | A missing profile is `resource.ErrNotFound`; a profile used by an availability zone returns the server's 409 once. |
| `List` / `All` | `GET .../availabilityzoneprofiles` | 200 | Filters `ID`, `Name`, `ProviderName`, `AvailabilityZoneData`, `Fields`, `Limit`, `Marker`, `PageReverse`; follows `availability_zone_profiles_links` and carries filters onto continuation pages. |
| `Find` / `FindIdentity` / `Remove` | | | Shared `resource.Collection` policies. `FindIdentity` mirrors Python `find_availability_zone_profile` (GET by ID first, then exact-name list); duplicate names fail. `Remove` ignores a missing profile unless `resource.WithMissingError()` is given. |

Returned profiles keep the declared view separate from `Body` (raw wire fields), `Header` and `StatusCode`. Extensions use `WithCreateField`, `WithUpdateField` and `WithListQuery`; a name that collides with a core input fails before any request. Mutations are sent once and never replayed, and context cancellation does not roll back remote work.

Python comparison: [python-parity.md](python-parity.md).
