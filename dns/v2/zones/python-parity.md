# Zone helpers beyond the native wrappers

These methods live in hand-written files beside the generated `zones.API`. They cover openstacksdk `dns/v2` zone behavior that Gophercloud v2.15.0 does not expose. All of them take a `context.Context`, never replay an uncertain mutation, and do not imply that local cancellation rolls back remote work.

| Go method | Request | Accepted codes | Notes |
|---|---|---|---|
| `API.DeleteWithShares(ctx, zoneID, deleteShares)` | `DELETE /zones/{id}` | 202 | Sends `X-Designate-Delete-Shares: true` only when `deleteShares` is true; `false` sends the same request as `API.Delete`. Returns the acknowledged zone, not completion. |
| `API.Abandon(ctx, zoneID)` | `POST /zones/{id}/tasks/abandon` | 202, 204 | No request body and no `Content-Type`. |
| `API.Xfr(ctx, zoneID)` | `POST /zones/{id}/tasks/xfr` | 202, 204 | No request body. The acknowledgement does not show that the transfer from the masters finished. |
| `API.Nameservers(ctx, zoneID)` | `GET /zones/{id}/nameservers` | 200 | Returns `[]Nameserver` in response order (`Hostname`, `Priority`); a missing `nameservers` key yields an empty non-nil slice; unknown fields are not retained. |

Python `delete_zone(ignore_missing=True)` corresponds to `API.Remove`, which ignores a missing zone by default; `DeleteWithShares` keeps a native 404 as an error and `resource.IgnoreMissing` converts it to nil when the Python default is wanted.

Python `find_zone` first tries a GET by identifier and falls back to a name list. The Go `Find` takes an explicit `resource.ID` (GET only) or `resource.Name` (list by exact `name`, duplicates fail with `resource.ErrAmbiguous`) and returns `resource.ErrNotFound` by default; pass `resource.WithIgnoreMissing()` for the Python `ignore_missing=True` result of `nil, nil`.

Python `create_zone` silently drops `email` and `ttl` for `type == "SECONDARY"`. `CreateOpts` sends exactly what the caller sets; omit those fields for secondary zones.

Python `wait_for_status` and `wait_for_delete` over zones correspond to `API.WaitFor` and `API.WaitForDeletion`: case-insensitive status comparison, `ERROR` ends the wait with `resource.ErrFailedState`, and deletion completes on a 404. Defaults differ (Go: five-minute timeout, two-second interval); tune them with `resource.WithTimeout` and `resource.WithPollInterval`.

Tests: `api/py_dns_zones_test.go`, `api/py_dns_misc_test.go`.
