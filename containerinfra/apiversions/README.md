# Magnum API versions

`apiversions.API` wraps Gophercloud v2.15.0 `containerinfra/apiversions`. Requests go to the service root derived from the client endpoint (the version segment is removed), not below the versioned endpoint. Python openstacksdk equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `List` | `GET /` | 200 | Single page; typed `APIVersion` (`ID`, `Status`, `Version` from `max_version`, `MinVersion`) |
| `Get` | `GET /{version}/` | 200 | Expects exactly one entry in `versions`: none returns `ErrVersionNotFound`, several return `ErrMultipleVersionsFound` |

There are no extension options. Native 404 responses stay HTTP response errors; `Find` with a `resource.ID` (the common `resource.Collection`) maps them to `resource.ErrNotFound`. API versions have no delete route, so the generated `Remove`, `WaitFor` and `WaitForDeletion` helpers are not meaningful for this resource.

Version discovery does not select a microversion by itself; the package sends `OpenStack-API-Version: container-infra <version>` only when the `ServiceClient` has a `Microversion`. Cancelling a context stops the local call.
