# Native Admin Infra

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/hypervisors.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.WithListQuery`

- List GETs os-hypervisors/detail with pointer limit, marker, hypervisor_hostname_pattern and with_servers (false still sent) plus extension query, follows hypervisors_links next, decodes mixed-microversion rows, fails the whole page on a row without hypervisor_version, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native pager status errors without operation context, and rejects a nil option before HTTP. The hypervisor decoder turns integer (pre-2.53) or UUID ids into strings, accepts cpu_info as a JSON string (pre-2.28), an object or absent (2.88+), converts scientific-notation versions, and fails when id or hypervisor_version is missing or the envelope is null, while a body without the envelope yields a zero value.
  Tests: `compute/v2/hypervisors/native_contracts_test.go:TestNativeHypervisorRoutesPagingAndDecode`, `compute/v2/hypervisors/native_contracts_test.go:TestNativeHypervisorStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/hypervisors call only; Python Hypervisor resource models and cloud hypervisor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/hypervisors.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.Get`

- Get GETs os-hypervisors/{id} without a query, accepts only 200 and decodes the hypervisor envelope. The hypervisor decoder turns integer (pre-2.53) or UUID ids into strings, accepts cpu_info as a JSON string (pre-2.28), an object or absent (2.88+), converts scientific-notation versions, and fails when id or hypervisor_version is missing or the envelope is null, while a body without the envelope yields a zero value.
  Tests: `compute/v2/hypervisors/native_contracts_test.go:TestNativeHypervisorRoutesPagingAndDecode`, `compute/v2/hypervisors/native_contracts_test.go:TestNativeHypervisorStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/hypervisors call only; Python Hypervisor resource models and cloud hypervisor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/hypervisors.GetExt`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.GetExt`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.WithGetExtOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.WithGetExtQuery`

- GetExt GETs os-hypervisors/{id} with the with_servers pointer query and extension query (none when unset), decodes the 2.53+ servers list or leaves it nil, accepts only 200 and rejects a nil option or empty query key before HTTP. The hypervisor decoder turns integer (pre-2.53) or UUID ids into strings, accepts cpu_info as a JSON string (pre-2.28), an object or absent (2.88+), converts scientific-notation versions, and fails when id or hypervisor_version is missing or the envelope is null, while a body without the envelope yields a zero value.
  Tests: `compute/v2/hypervisors/native_contracts_test.go:TestNativeHypervisorRoutesPagingAndDecode`, `compute/v2/hypervisors/native_contracts_test.go:TestNativeHypervisorStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/hypervisors call only; Python Hypervisor resource models and cloud hypervisor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/hypervisors.GetStatistics`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.GetStatistics`

- GetStatistics GETs os-hypervisors/statistics, decodes the hypervisor_statistics envelope into Statistics and accepts only 200.
  Tests: `compute/v2/hypervisors/native_contracts_test.go:TestNativeHypervisorRoutesPagingAndDecode`, `compute/v2/hypervisors/native_contracts_test.go:TestNativeHypervisorStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/hypervisors call only; Python Hypervisor resource models and cloud hypervisor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/hypervisors.GetUptime`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.GetUptime`

- GetUptime GETs os-hypervisors/{id}/uptime, decodes the hypervisor envelope with an integer or string id into Uptime (a present object without id fails, a missing envelope yields a zero value) and accepts only 200.
  Tests: `compute/v2/hypervisors/native_contracts_test.go:TestNativeHypervisorRoutesPagingAndDecode`, `compute/v2/hypervisors/native_contracts_test.go:TestNativeHypervisorStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/hypervisors call only; Python Hypervisor resource models and cloud hypervisor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/aggregates.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.List`

- List GETs os-aggregates as one SinglePageBase page without following embedded links, decodes integer ids, 2.41 uuid, string metadata and zone-less timestamps, fails the page on a mistyped row, yields nothing for an empty or missing array, reports io.EOF for a bodyless 204 and native pager status errors without operation context.
  Tests: `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateRoutesBodiesAndDecode`, `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/aggregates call only; Python Aggregate resource models, cloud aggregate helpers and image/traits actions are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/aggregates.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithCreateField`

- Create POSTs os-aggregates with {"aggregate":{name, omitempty availability_zone, extensions}}, rejects an empty name, a nil option or an extension colliding with a core key (even an omitted availability_zone) before HTTP, accepts only 200 and returns a nil aggregate without error when the envelope is missing or null.
  Tests: `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateRoutesBodiesAndDecode`, `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/aggregates call only; Python Aggregate resource models, cloud aggregate helpers and image/traits actions are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/aggregates.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.Get`

- Get GETs os-aggregates/{decimal id}, accepts only 200, decodes zone-less created/updated/deleted timestamps (a Z suffix fails, null or empty is zero) and string-only metadata, and returns nil without error for a missing or null envelope.
  Tests: `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateRoutesBodiesAndDecode`, `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/aggregates call only; Python Aggregate resource models, cloud aggregate helpers and image/traits actions are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/aggregates.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateField`

- Update PUTs os-aggregates/{id} with {"aggregate":{omitempty name/availability_zone, extensions}} (an empty UpdateOpts sends an empty object), rejects core-key extensions and nil options before HTTP and accepts only 200.
  Tests: `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateRoutesBodiesAndDecode`, `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/aggregates call only; Python Aggregate resource models, cloud aggregate helpers and image/traits actions are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/aggregates.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.Delete`

- Delete sends DELETE os-aggregates/{id} and accepts only 200, so 202 and 204 are wrapped native unexpected-status errors.
  Tests: `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateRoutesBodiesAndDecode`, `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/aggregates call only; Python Aggregate resource models, cloud aggregate helpers and image/traits actions are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/aggregates.AddHost`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.AddHost`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithAddHostOptions`

- AddHost POSTs os-aggregates/{id}/action with {"add_host":{"host":...}}, rejects an empty host or nil option before HTTP, offers no extension fields, accepts only 200 and decodes the aggregate envelope.
  Tests: `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateRoutesBodiesAndDecode`, `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/aggregates call only; Python Aggregate resource models, cloud aggregate helpers and image/traits actions are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/aggregates.RemoveHost`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.RemoveHost`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithRemoveHostOptions`

- RemoveHost POSTs os-aggregates/{id}/action with {"remove_host":{"host":...}}, rejects an empty host before HTTP, accepts only 200 and decodes the aggregate envelope.
  Tests: `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateRoutesBodiesAndDecode`, `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/aggregates call only; Python Aggregate resource models, cloud aggregate helpers and image/traits actions are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/aggregates.SetMetadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.SetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithSetMetadataOptions`

- SetMetadata POSTs os-aggregates/{id}/action with {"set_metadata":{"metadata":{...}}} sending nil values as null deletions, rejects nil metadata before HTTP, accepts only 200 and decodes the aggregate envelope.
  Tests: `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateRoutesBodiesAndDecode`, `compute/v2/aggregates/native_contracts_test.go:TestNativeAggregateStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/aggregates call only; Python Aggregate resource models, cloud aggregate helpers and image/traits actions are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/services.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithListQuery`

- List GETs os-services with omitempty binary/host and extension query as one SinglePageBase page that ignores services_links, decodes integer or UUID ids as strings, fails the page on a row without id, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native pager status errors without operation context, and rejects a nil option before HTTP.
  Tests: `compute/v2/services/native_contracts_test.go:TestNativeComputeServiceRoutesBodiesAndDecode`, `compute/v2/services/native_contracts_test.go:TestNativeComputeServiceStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/services call only; Python Service resource models and the enable/disable/force-down helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/services.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithUpdateField`

- Update PUTs os-services/{id} with an unwrapped body of omitempty status, disabled_reason and forced_down (false cannot be sent) plus root-level extensions, rejects core-key extensions (including forced_down) and nil options before HTTP, accepts only 200, and decodes the service envelope where a missing envelope is zero but a null or id-less object fails.
  Tests: `compute/v2/services/native_contracts_test.go:TestNativeComputeServiceRoutesBodiesAndDecode`, `compute/v2/services/native_contracts_test.go:TestNativeComputeServiceStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/services call only; Python Service resource models and the enable/disable/force-down helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/services.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.Delete`

- Delete sends DELETE os-services/{id} and accepts only 204, so 200 and 202 are wrapped native unexpected-status errors.
  Tests: `compute/v2/services/native_contracts_test.go:TestNativeComputeServiceRoutesBodiesAndDecode`, `compute/v2/services/native_contracts_test.go:TestNativeComputeServiceStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/services call only; Python Service resource models and the enable/disable/force-down helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/migrations.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/migrations.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/migrations.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/migrations.WithListQuery`

- List GETs os-migrations with pointer filters (a zero limit is still sent), offset-preserving RFC3339 changes-since/changes-before and extension query, reads one SinglePageBase page without following 2.59 migrations_links, decodes int64 ids and zone-less timestamps, fails the page on bad rows, yields nothing for an empty or missing array, reports io.EOF for a bodyless 204 and native pager status errors without operation context, and rejects a nil option before HTTP.
  Tests: `compute/v2/migrations/native_contracts_test.go:TestNativeMigrationListQueryAndDecode`, `compute/v2/migrations/native_contracts_test.go:TestNativeMigrationListStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/migrations call only; Python Migration resource models and server migration helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/diagnostics.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/diagnostics.API.Get`

- Get GETs servers/{id}/diagnostics with default GET OkCodes (200 only), returns the whole body as map[string]any with float64 numbers for both pre-2.48 and 2.48+ shapes, returns a nil map for null and fails on a non-object body.
  Tests: `compute/v2/diagnostics/native_contracts_test.go:TestNativeServerDiagnosticsRouteDecodeAndStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/diagnostics call only; Python ServerDiagnostics resource models are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/availabilityzones.ListDetail`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.API.ListDetail`
- `github.com/JSYoo5B/go-openstacksdk/resource.SinglePageStream`

- ListDetail GETs os-availability-zone/detail once through resource.SinglePageStream (AllPages bypasses SinglePageBase's array-only IsEmpty), decodes the full availabilityZoneInfo object page with nested host/service states and zone-less updated_at before publishing any row, ignores links, yields nothing for a missing or null array, and reports io.EOF for a bodyless 204 and native pager status errors without operation context.
  Tests: `compute/v2/availabilityzones/native_detail_test.go:TestNativeAvailabilityZoneListDetailObjectPage`, `compute/v2/availabilityzones/native_detail_test.go:TestNativeAvailabilityZoneListDetailStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/availabilityzones call only; Python AvailabilityZone detail resource models and cloud zone helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/usage.AllTenants`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.API.AllTenants`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.WithAllTenantsOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.WithAllTenantsQuery`

- AllTenants GETs os-simple-tenant-usage with detailed=1 only when Detailed is true, offset-less wall-clock start/end, omitempty limit/marker and extension query, follows tenant_usages_links next, decodes TenantUsage rows with zone-less timestamps, yields nothing for an empty or missing array, reports io.EOF for a bodyless 204 and native pager status or decode errors without operation context, and rejects a nil option before HTTP.
  Tests: `compute/v2/usage/native_all_tenants_test.go:TestNativeAllTenantsUsageQueryPagingAndDecode`, `compute/v2/usage/native_all_tenants_test.go:TestNativeAllTenantsUsageStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/usage call only; Python Usage resource models and cloud usage helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [Availability Zones](../availability-zones.md)
- [Usage](usage/README.md)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/hypervisors/requests.go)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/hypervisors/results.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/aggregates/requests.go)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/aggregates/results.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/services/requests.go)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/services/results.go)
- [Request](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/migrations/request.go)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/migrations/results.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/diagnostics/requests.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/availabilityzones/requests.go)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/availabilityzones/results.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/usage/requests.go)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/usage/results.go)
