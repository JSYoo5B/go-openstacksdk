# Python Detail Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/availability_zones`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.API.ListDetailRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ListAvailabilityZones`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.WithAvailabilityZoneListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.WithAvailabilityZoneMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.WithAvailabilityZoneHeader`

- The details=False branch is available through ListRecords and Service.ListAvailabilityZones at the ordinary fixed path. id/name/state/hosts are passive raw descriptors with missing or null represented by JSON null; zoneName/name and zoneState/state aliases follow original dictionary order, unknown fields remain Wire-only, and a name never fabricates inherited id.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership`, `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`, `compute/v2/availabilityzones/python_detail_records_test.go:TestPythonZoneDetailRecordsLeaveOrdinaryRoute`.
- A logical iteration owns options lazily, captures source and Service binding before callbacks, snapshots current location once before the callbacks, and keeps location confined to independent Resource views. Actual row Envelope/Wire and header/status remain independent. Selected version is retained, explicit empty is versionless, and source mutation stops later callbacks before HTTP.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership`, `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`.
- Owned ordinary rows accept the Source-facing 200..399 success range through the shared bodyless guarded reader, preserving reverse endpoint prefixes and actual metadata. Advertised linked pages retain duplicate records, an empty raw page stops before its next link, consumer break stops continuation, and late HTTP403 is visible with its original receipt after prior rows.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership`.
- The actual collection implementation composes the existing optional page reader, singleton-object representation and decode-no-content policies, plus an opt-in first advertised server limit that remains fixed without enabling initial-limit marker fallback. This shared behavior is separate from native SinglePageBase DTO listing.
  Tests: `internal/rest/list_test.go:TestListOptionalReaderAndRepresentationsKeepDefaultPolicy`, `internal/rest/list_test.go:TestListFirstServerLimitIsOptInAndFixedAcrossContinuations`, `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership`.
- The details=True branch is ListDetailRecords: it sends a bodyless, queryless GET os-availability-zone/detail with the client microversion in X-OpenStack-Nova-API-Version and OpenStack-API-Version, no version headers for a blank client or an explicit empty WithAvailabilityZoneMicroversion, and caller headers from WithAvailabilityZoneHeader.
  Tests: `compute/v2/availabilityzones/python_detail_records_test.go:TestPythonZoneDetailRecordsUseDetailRouteAndKeepHosts`.
- Detail rows use the same AvailabilityZoneRecord projection as ListRecords: zoneName/zoneState become name/state, hosts with per-service active/available/updated_at state stay raw JSON, id and location are null, unknown vendor fields remain only in Wire, and Envelope/Header/StatusCode keep the actual row and page receipt.
  Tests: `compute/v2/availabilityzones/python_detail_records_test.go:TestPythonZoneDetailRecordsUseDetailRouteAndKeepHosts`.
- ListDetailRecords follows availabilityZoneInfo_links continuations on the detail path and rejects a continuation that leaves os-availability-zone/detail before HTTP, accepts 200..399 including a singleton object page, and reports 403/404 rejections as resource.OperationError{ListDetailRecords, availabilityzones} wrapping the native status error and body.
  Tests: `compute/v2/availabilityzones/python_detail_records_test.go:TestPythonZoneDetailRecordsFollowDetailLinks`, `compute/v2/availabilityzones/python_detail_records_test.go:TestPythonZoneDetailRecordsStatuses`.

Python/Go differences:

- Python selects the branch with availability_zones(details=False|True); Go exposes ListRecords for details=False and ListDetailRecords for details=True.
- The pinned public signature is availability_zones(details=False), with no public **query. Go offers only concrete headers/microversion options for both branches; it does not invent reachable Source body filters, limit/marker/max-items arguments, arbitrary base_path, or supplied Resource parameters.
- AvailabilityZone declares id/name/state/hosts without typed conversion, with default None and no alternate ID. Resource is the declared owned view; Wire is actual JSON including unknown/vendor fields, Envelope is the actual row, and physical Header/StatusCode are additive Go evidence. Python returns hosts as a dict while Go keeps raw JSON. Service maps existing Connection current_location while standalone leaf location is null.
- Blank source version causes no discovery because the Source class has no _max_microversion. Explicit per-call empty is a Go versionless override; the original native client and live authentication remain unchanged under the existing guarded read policy.
- Go requires valid UTF8 object/page representations, supports one object in the plural envelope, and decodes an accepted204 representation rather than treating an empty204 as a successful empty list. Shared reader errors remain visible; Python JSON/shape exception class spelling is not reproduced.
- Python selects body links, then top-level next, then HTTP Link according to source precedence. Go validates all advertised candidates consistently, restricts origin/prefix/route/query changes (detail continuations must stay on os-availability-zone/detail), and supports dictionary links.next as extra compatibility. An advertised first positive limit may be adopted but does not create the absent initial limit or enable marker fallback.
- Owned records do not provide generic mutable Resource/session/cache behavior; the generated native ListDetail remains a separate typed Gophercloud listing.

## References

- [`availability_zones(details=True)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [`AvailabilityZoneDetail`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/availability_zone.py)
