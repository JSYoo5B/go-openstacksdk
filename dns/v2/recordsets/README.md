# Designate record sets

`recordsets.API` wraps Gophercloud v2.15.0 `dns/v2/recordsets`. Designate bodies are flat JSON (no envelope). Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /zones/{zone}/recordsets` | 201, 202 | Optional empty `Description`, `Records`, `TTL` 0 and `Type` are omitted |
| `Get` | `GET /zones/{zone}/recordsets/{id}` | 200 | Typed model; native 404 stays an HTTP response error |
| `Update` | `PUT /zones/{zone}/recordsets/{id}` | 200, 202 | `Description` and `TTL` are pointers: `""` is sent, `TTL` pointing at 0 or less is sent as JSON `null`, nil is omitted; a zero `UpdateOpts` sends `{}` |
| `Delete` | `DELETE /zones/{zone}/recordsets/{id}` | 202 only | No headers |
| `DeleteWithOpts` | same route | 202 only | `DeleteOpts` headers |
| `ListByZone` | `GET /zones/{zone}/recordsets` | 200 | Follows `links.next` |
| `ListAll` | `GET /recordsets` | 200 | Cross-zone listing; use `ListOpts.ZoneID` to narrow |

List filters are `name`, `type`, `data`, `description`, `status`, `ttl`, `zone_id`, `sort_key`, `sort_dir`, `limit` and `marker`. `AllProjects` sends `X-Auth-All-Projects: true` only when true; `SudoTenantID` sends `X-Auth-Sudo-Tenant-ID`. Create, Update, Delete-with-options and list calls accept these headers; they are sent on every page.

Extension inputs use `WithCreateField`, `WithUpdateField`, `WithListByZoneQuery`, `WithListAllQuery` and the matching `...Header` options. Extension field values `false` and `nil` (JSON `null`) are sent as given. A field extension that names a core JSON field, a header extension that collides with `X-Auth-All-Projects` or `X-Auth-Sudo-Tenant-ID`, and nil options fail with `resource.ErrInvalidOption` before any request. Plain `Delete` has no option parameter.

`InZone(ctx, parent)` fixes the parent zone. An explicit `resource.ID` costs no request; a name is resolved once through the zones list. The returned scope provides the common `Find`, `All`, `Delete` and wait operations. Scope deletion ignores missing record sets by default, while `API.Delete` keeps the native 404 response error.

A 202 from Create, Update or Delete is an acknowledgement; Designate applies the change asynchronously. Wait on `Status` (and `Action`) explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
