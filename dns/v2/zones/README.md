# Designate zones

`zones.API` wraps Gophercloud v2.15.0 `dns/v2/zones` for the `/zones` routes. Designate bodies are flat JSON (no envelope). Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /zones` | 201, 202 | `TTL` is sent only when greater than 0; empty optional strings, maps and slices are omitted |
| `Get` | `GET /zones/{id}` | 200 | Typed model; unknown response fields are not retained |
| `Update` | `PATCH /zones/{id}` | 200, 202 | `Description` is a pointer, so `""` is sent; `TTL` 0 and empty `Email`/`Masters` are omitted; a zero `UpdateOpts` sends `{}` |
| `Delete` | `DELETE /zones/{id}` | 202 only | Returns the acknowledged zone (typically `PENDING`/`DELETE`); native 404 stays an HTTP response error |
| `List` | `GET /zones` | 200 | Filters `name`, `email`, `description`, `status`, `type`, `ttl`, `sort_key`, `sort_dir`, `limit`, `marker`; follows `links.next` |
| `GetShare`, `ListShares` | `GET /zones/{id}/shares[/{share}]` | 200 | `ListShares` reads `shared_zones` |
| `Unshare` | `DELETE /zones/{id}/shares/{share}` | 204 only | |

`ListOpts.AllProjects` sends `X-Auth-All-Projects: true` only when true (false sends no header); `SudoTenantID` sends `X-Auth-Sudo-Tenant-ID`. `ListSharesOpts.AllProjects` behaves the same. Headers are repeated on every page request.

Extension inputs use `WithCreateField`, `WithUpdateField`, `WithListQuery` and `WithListHeader`. Extension field values `false`, `""` and `nil` (JSON `null`) are sent as given. A field extension that names a core JSON field (for example `name` on Create or `email` on Update), a header extension that collides with `X-Auth-All-Projects` or `X-Auth-Sudo-Tenant-ID`, and nil options fail with `resource.ErrInvalidOption` before any request.

`Share` (`POST /zones/{id}/shares`, 201 only) sends `{"target_project_id": ...}` plus any `WithShareField` extension values. `target_project_id` cannot be overridden through `WithShareField` (`resource.ErrInvalidOption`, no request sent). The pinned native `Share` reflects over its builder argument and would send `{}` for the SDK wrapper, so the generator replaces that call with an audited helper that serializes `ToShareMap()` and keeps the native URL, 201 success code, and result. Other statuses are errors, and an uncertain POST is not replayed.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup through the `name` query, duplicate-name errors, the common `status` filter sent as `status`, and ignore-missing deletion by default (`Remove` returns nil for a 404). `API.Get` and `API.Delete` themselves do not ignore a missing zone.

A 202 from Create, Update or Delete is an acknowledgement; Designate finishes the change asynchronously. Wait on `Status` (and `Action`) explicitly, for example with `WaitFor` or `WaitForDeletion`. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
