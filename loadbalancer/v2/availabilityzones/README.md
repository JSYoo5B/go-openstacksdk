# Octavia availability zones

`availabilityzones.API` is an SDK-owned binding for `/lbaas/availabilityzones` (Octavia API 2.14 and later). Gophercloud v2.15.0 has no binding for these routes. The package is reached through `loadbalancer/v2.Service.AvailabilityZones()`. A client whose base already ends in `lbaas` is respected without double insertion.

Availability zones are identified by **name**; there is no UUID. Create, update and delete need administrative privileges (the server answers 403 otherwise).

| Operation | Route | Success | Notes |
|---|---|---|---|
| `Create` | `POST .../availabilityzones` | 201 | Body wrapped in `availability_zone`. `Name` and `AvailabilityZoneProfileID` are required; nil `Description`/`Enabled` are omitted, an empty description and `Enabled=false` are sent. |
| `Get` | `GET .../availabilityzones/{name}` | 200 | Name is validated as one unescaped path segment. |
| `Update` | `PUT .../availabilityzones/{name}` | 200 | Only `Description` and `Enabled` are mutable. An update with no field and no extension is rejected locally (`resource.ErrInvalidOption`). |
| `Delete` | `DELETE .../availabilityzones/{name}` | 204 | A missing zone is `resource.ErrNotFound`; a zone in use returns the server's 409 once. |
| `List` / `All` | `GET .../availabilityzones` | 200 | Filters `Name`, `Description`, `AvailabilityZoneProfileID`, `Enabled`, `Fields`, `Limit`, `Marker`, `PageReverse`; follows `availability_zones_links` and carries filters onto continuation pages. |
| `Find` / `FindIdentity` / `Remove` | | | Shared `resource.Collection` policies. `FindIdentity` mirrors Python `find_availability_zone`: GET by name first, then an exact-name list (`?name=`); duplicate list matches fail. `Find` with `resource.ID` uses GET, with `resource.Name` uses the list filter. `Remove` ignores a missing zone unless `resource.WithMissingError()` is given. |

Every returned `AvailabilityZone` keeps the declared view (`Name`, `Description`, `Enabled`, `AvailabilityZoneProfileID`) separate from `Body` (raw wire fields, so null and absent values stay distinguishable), `Header` and `StatusCode`.

Extensions use `WithCreateField`, `WithUpdateField` and `WithListQuery`. A field or query extension that names a core input fails before any request. Success status codes are explicit: any other 2xx is an error. Mutations are sent once and never replayed; cancelling a context stops the local call and does not roll back remote work.

Python comparison: [python-parity.md](python-parity.md).
