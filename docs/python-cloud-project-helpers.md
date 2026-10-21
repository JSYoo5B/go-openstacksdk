# Project-scoped cloud helpers

These Connection helpers cover the Python cloud-layer methods that first resolve a
project, role, domain, group or aggregate and then call a service operation. Every
helper takes a `context.Context`, sends each mutation at most once and never replays
it after an uncertain failure. A cancelled context stops the sequence before the next
request; it does not undo a request that was already sent.

Lookups use the Keystone v3 client of the Connection (`Connection.IdentityV3`). The
Python `ensure_service_version(identity, '3')` check has no separate Go step because
Go has only a v3 Identity binding.

## Project quotas

`FindProjectComputeQuotas`, `GetComputeQuotas`, `SetComputeQuotas`,
`DeleteComputeQuotas` and the block storage counterparts (`FindProjectBlockStorageQuotas`,
`GetBlockStorageQuotas`, `SetBlockStorageQuotas`, `DeleteBlockStorageQuotas`) accept
an unqualified project name or ID. Each call performs one strict lookup first: `GET
/projects/{value}` followed, after a 404, by a name-filtered list. A missing project
returns `resource.ErrNotFound` and no quota request is sent. The explicit
`ProjectQuotas` and `BlockStorageProjectQuotas` scopes remain the way to bind a known
ID with no lookup or to reuse one resolved project across several calls.

- `SetComputeQuotas` appends `WithUpdateForce(true)` after the caller's options, as the
  Python helper overwrites `force` with `True`. The concrete `UpdateOpts` still decides
  which limits are sent, so zero and `-1` values are preserved.
- `SetBlockStorageQuotas` adds no force flag.
- The Go helpers return the quota model (`Set`/`Get`) or the reset response (`Delete`);
  Python returns `None` for set and delete.
- Python accepts arbitrary keyword quota names; Go uses the typed `UpdateOpts` plus
  `WithUpdateField` for extension keys.
- Block storage in Go is Cinder v3 only; Python's helper also works against v2.

## `GetComputeLimits` and `GetComputeUsage`

`GetComputeLimits(ctx, "")` reads the limits of the authenticated project with no
Identity request and no tenant query. A non-empty value is resolved strictly and sent
as `tenant_id`. Go returns the whole limits resource; Python returns only the
`absolute` part of it.

`GetComputeUsage` resolves the project strictly and reads one usage object. Start and
end are optional UTC values (`ComputeUsageRange`); `ParseComputeUsageDate` accepts a
date, a date-time with `T` or a space, fractional seconds and an optional `Z` or numeric
offset (converted to UTC) and rejects everything else. It is a bounded subset of the
ISO 8601 grammar the Python parser accepts. Pagination links are not followed, which
matches the single Python fetch.

## Role grants

`GrantRole` and `RevokeRole` take a role name or ID and a `RoleAssignmentTarget`
(user or group, project/domain/system scope, inherited flag). The lookup order is
domain, role, user, group, project; the domain ID filters the user, group and project
lookups but never the role lookup. All lookups are strict. An existing assignment
makes `GrantRole` return `(false, nil)` without writing; a missing assignment makes
`RevokeRole` return `(false, nil)` without writing; otherwise one PUT or DELETE is sent
and `(true, nil)` is returned.

Documented differences from Python:

- Only HTTP 404 on the HEAD check means "assignment absent". Python's raw session call
  returns false for any non-204 status, so 403 or 500 are also read as absent there; Go
  returns those as errors instead of continuing to a write.
- A failed PUT or DELETE is returned as an error. Python ignores the status of the write
  and returns `True`.
- A successful write is only an acknowledgement; no wait for propagation is done and
  Python's `wait`/`timeout` arguments are also unused there.
- System scope always uses the `/system` route; the Python `system` string is not sent.
- Empty strings mean "not given" in Go.

## `UpdateDomain` and `UpdateGroup`

`UpdateDomain` takes a `resource.Ref`: an ID updates without a lookup (Python
`domain_id`), a name finds the domain by ID-or-name first (Python `name_or_id`) and
returns `resource.ErrNotFound` when absent. An empty `Name` and nil `Description` or
`Enabled` are omitted; `Enabled=false` and an empty non-nil `Description` are sent
(Python drops empty strings). When nothing is supplied no PATCH is sent and a domain
carrying only the ID is returned, as in Python where an empty commit sends nothing.

`UpdateGroup` finds the group strictly (extra lookup queries such as `domain_id`
go through `resource.WithIdentityFindQuery`) and sends only attributes that differ from
the found group; with no difference no PATCH is sent and a copy of the found group is
returned.

## Host aggregates

`UpdateAggregate` finds the aggregate strictly and sends one PUT with only the supplied
attributes (including a null availability zone through `request.Null`). `DeleteAggregate`
returns `(false, nil)` for a missing aggregate. A digit-only identity is deleted with no
lookup and a 404 on the DELETE is also `false`. Python's `str.isdigit` also accepts some
non-ASCII digits that Go sends through the name lookup. The generated Delete binding
accepts only HTTP 200 from Nova, while Python accepts any status below 400.
