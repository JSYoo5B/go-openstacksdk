# Octavia amphorae

`amphorae.API` wraps Gophercloud v2.15.0 `loadbalancer/v2/amphorae` for the administrative `/octavia/amphorae` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Get` | `GET .../{id}` | 200 | Typed model; `CertExpiration`, `CreatedAt` and `UpdatedAt` decode zone-less timestamps; native 404 stays an HTTP response error |
| `List` | `GET /octavia/amphorae` | 200 | Follows `amphorae_links`; typed filters are `LoadbalancerID`, `ImageID`, `Role`, `Status`, `HAPortID`, `VRRPPortID` plus paging/sort keys |
| `Failover` | `PUT .../{id}/failover` | 202 only | No request body |

Extension queries use `WithListQuery`; a nil option fails with `resource.ErrInvalidOption` before any request. There are no create, update or delete operations.

`Find`, `All`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection` with `status` as the wait status. Amphorae have no native delete, so `Remove` is not a supported deletion path.

Failover acknowledgement does not imply completion; wait on `status` explicitly. Cancelling a context stops the local call and does not roll back remote work; a failover is never replayed automatically.
