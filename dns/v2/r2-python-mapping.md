# Designate additions and cloud helpers

New SDK-owned packages: [blacklists](blacklists/README.md), [tlds](tlds/README.md), [zoneexports](zoneexports/README.md), [zoneimports](zoneimports/README.md), [ptr/floatingips](ptr/floatingips/README.md), [limits](limits/README.md), [servicestatuses](servicestatuses/README.md); accessors live in `service_sdk_owned.go`. Shared transport rules sit in an internal package (flat bodies, explicit success codes, links.next paging).

## Scope headers

The pinned Python sends `x-auth-sudo-project-id` and `x-auth-all-projects`; the Designate API reference documents the same names. The new SDK-owned packages and the quota singleton send `X-Auth-Sudo-Project-ID`. The generated Gophercloud-based helpers (`recordsets`, list helpers) send Gophercloud's `X-Auth-Sudo-Tenant-ID` instead; the two spellings are not interchangeable for a given server unless it accepts both, so use the SDK-owned packages when the documented name matters. An explicit `false` all-projects value is sent as `false`; Python sends the header only when truthy.

## Quotas

`quotas.API.Current(ctx, ...)` is Python `quotas()`: `GET /quotas` returning the single flat quota object, with optional `WithCurrentSudoProject` and `WithCurrentAllProjects`. Python `get_quota`, `update_quota` and `delete_quota` map to `InProject(...).Get/Update/Reset`; see [quotas](quotas/README.md).

## Zone shares and recordsets

`zones.API.ListSharesFiltered` adds the `target_project_id` filter and scope headers. `zones.API.FindShare` is `find_zone_share` (GET, then share list matching the ID). `RecordSets.InZone(...).FindIdentity` is `find_recordset`. Waits for record sets, exports, imports and PTR records use the common Collection `Wait`/`WaitDeleted`.

## Cloud helpers

`Service` and `Connection` methods `ListDNSZones`, `GetDNSZone`, `SearchDNSZones`, `CreateDNSZone`, `UpdateDNSZone`, `DeleteDNSZone`, `ListDNSRecordSets`, `GetDNSRecordSet`, `SearchDNSRecordSets`, `CreateDNSRecordSet`, `UpdateDNSRecordSet`, `DeleteDNSRecordSet` take a zone name or ID. Differences from Python:

- Zone and record set lookups use `FindIdentity` (GET by identifier, then name list) and return `nil, nil` when absent (`Get*`) or `false, nil` (`Delete*`); update helpers and helpers needing a zone return an error wrapping `resource.ErrNotFound`. Passing a Zone object to skip the lookup is replaced by passing its ID.
- `CreateDNSZone` sends explicit JSON null for an unset email and description, upper-cases and validates the type, and drops email and ttl for SECONDARY zones. A 201 or 202 only acknowledges creation. Failures keep their cause and name the zone.
- `GetDNSZone` offers no filters: the pinned `find_zone` raises TypeError for any.
- Search helpers list raw wire rows and apply the SDK's `_filter_list` implementation (glob name/ID, mapping filters, JMESPath for record sets). `SearchDNSZones` also sends scalar filter values as query parameters like Python (booleans as `True`/`False`); non-scalar values and string filters are rejected before HTTP. Python filters Resource views; Go filters wire fields.
- `UpdateDNSZone` and `UpdateDNSRecordSet` use the native update options, so omitted fields are not sent. `CreateDNSRecordSet` omits an empty description and zero ttl as Python's truthiness does.
- `DeleteDNSRecordSet` is strict after a successful lookup (Python `ignore_missing=False`); `DeleteDNSZone` ignores a concurrent 404.
