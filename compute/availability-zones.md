# Availability Zones

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_compute/ComputeCloudMixin/list_availability_zone_names`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ListAvailabilityZoneNames`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailabilityZoneNamesOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithUnavailableZones`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailabilityZoneNamesMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailabilityZoneNamesHeader`

- The Cloud name method always consumes the ordinary os-availability-zone inventory. Unavailable defaults false; true includes unavailable rows without requesting the admin detail route. Name values remain passive JSON, including numeric/bool/object/array/null and omitted-as-null, preserving source order and duplicates; a normal empty inventory completes Names and Value as an empty array.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`.
- The state available key is read before the unavailable option is considered. JSON-domain Python truthiness controls ordinary inclusion; missing/null/scalar/array state and a missing available key remain errors even with unavailable true, returning known Inventory without fabricated completed Names or Value.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`.
- A clean actual HTTP 403/500, late actual404, or isolated repeated pagination marker cycle produces the source-compatible completed empty array, discards the collected name prefix, retains Inventory diagnostics, and exposes the original status/body/header/URL or cycle through SuppressedError. Duplicate re-return of the same original retry error does not invent another independent terminal cause.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`.
- Accepted malformed JSON and accepted Close failures with a nested native404 remain visible response errors with actual200 receipt and no accepted replay. A different retry callback cause beside actual HTTP500 retains both causes, source failure beside HTTP500 and cancellation with custom cause beside HTTP403 remain visible, and an option callback returning a native404 does not become a suppressed fetched SDK failure or completed names.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`.
- Concrete typed option/version/header inputs are owned, full replacement and individual helpers apply at most once, and current location is captured before option callbacks once per logical call. Header/version overrides reach the fixed GET while the original selected native client and actual wire location remain unchanged; response Names/Value/Resource/Envelope own independent bytes.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`, `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership`.

Python/Go differences:

- Pinned _compute.py:255-276 catches openstack.exceptions.SDKException, not arbitrary Python failures. Go explicitly classifies a clean actual native HTTP4xx/5xx rejection and an isolated pagination cycle; accepted IO/decode/source/context/URL-transport/local safety and different joined error causes remain visible. SuppressedError and retained Inventory are additive diagnostic results; Source returns only [].
- The Python list[str] annotation does not coerce the untyped zone name descriptor. Names uses []json.RawMessage and Value uses an owned JSON array to preserve the actual JSON values instead of synthesizing strings. Malformed state/key errors use Go errors with a known partial result rather than Python TypeError/KeyError classes.
- Public Source takes only unavailable=False. Go adds concrete header and microversion options; blank selected source performs no version discovery, nil preserves selection, explicit empty suppresses version headers for this call, and no detail/query/body controls are introduced.
- Service supplies an owned snapshot of existing Connection location to declared Resource only; standalone service without that dependency uses null. Wire/Envelope remain the actual row and header/status evidence is additive. This finite name-list mapping does not implement a general mutable Resource/cache/Session surface.
- The shared guarded pager accepts compatible advertised continuation but requires safe fixed collection scope and consistent continuation candidates. Python chooses a precedence channel instead; dictionary links.next is additional Go compatibility. Completing this Cloud row does not close Proxy details=True or native ListDetail.

### `gophercloud:compute/v2/availabilityzones.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.API.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.SinglePageStream`

- Native ordinary List remains lazy and uses the original fixed os-availability-zone GET with selected client/version/token and native AvailabilityZone DTO. One physical page decodes the complete availabilityZoneInfo array before any row is published; embedded next links do not cause another request or a detail lookup.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNativeListTypedSinglePageABI`.
- Actual200 and300 object envelopes preserve ZoneName string, ZoneState.Available bool, Hosts and nested ServiceState booleans plus the native millisecond no-zone timestamp decoder. Missing/null DTO fields retain native zero values, and null or missing plural data yields no rows.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNativeListTypedSinglePageABI`.
- Native success codes remain200/204/300: empty application/json204 exposes EOF, unexpected201 and403 retain original method/URL/body/header/status evidence, and syntax or a wrong known field type fails without publishing an earlier row from the same page.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNativeListTypedSinglePageABI`.

Python/Go differences:

- Pinned v2.15.0 AvailabilityZonePage inherits array-only SinglePageBase.IsEmpty, so native EachPage cannot handle the normal object envelope. The audited SDK generator selects SinglePageStream only for ordinary AvailabilityZones.List; it uses native Pager.AllPages, whose SinglePageBase shortcut bypasses that emptiness check, then the unchanged native extractor. ListDetail keeps its original generated path and is not promoted.
- The SDK exposes context-aware iter.Seq2 rather than returning pagination.Pager. It retains original AllPages single-page semantics and complete native decode, with local cancellation and consumer break ending publication; native DTO fields are separate from the new untyped owned Resource/Wire view.
- Native pagination accepts200/204/300 but does not define a successful empty JSON representation for204. Native PageResultFrom/ExtractInto/null/default behavior and configured prebody provider policies remain native; the SDK does not retrofit owned accepted-response IO evidence or Python descriptor values into this native row.

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
for zone in conn.compute.availability_zones():
    print(zone.name, zone.state, zone.hosts)

print(conn.list_availability_zone_names())
print(conn.list_availability_zone_names(unavailable=True))
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
    cloudName := flag.String("cloud", "dev", "clouds.yaml cloud name")
    unavailable := flag.Bool("unavailable", false, "include unavailable zone names")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloudName, *unavailable); err != nil {
        log.Fatal(err)
    }
}

func printJSON(value any) error {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloudName string, unavailable bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }

    records := make([]map[string]any, 0)
    for record, readErr := range service.ListAvailabilityZones(ctx) {
        if record != nil {
            records = append(records, map[string]any{
                "resource": record.Resource, "wire": record.Wire,
                "envelope": string(record.Envelope),
                "header": record.Header, "status_code": record.StatusCode,
            })
        }
        if readErr != nil {
            return errors.Join(readErr, printJSON(map[string]any{
                "operation": "zones", "partial_records": records,
            }))
        }
    }
    if err := printJSON(map[string]any{"operation": "zones", "records": records}); err != nil {
        return err
    }

    options := make([]compute.AvailabilityZoneNamesOption, 0)
    if unavailable { options = append(options, compute.WithUnavailableZones(true)) }
    result, readErr := service.ListAvailabilityZoneNames(ctx, options...)
    if result != nil {
        suppressed := ""
        if result.SuppressedError != nil { suppressed = result.SuppressedError.Error() }
        printErr := printJSON(map[string]any{
            "operation": "names", "names": result.Names, "value": result.Value,
            "inventory_count": len(result.Inventory), "suppressed_error": suppressed,
        })
        if printErr != nil { return errors.Join(readErr, printErr) }
    }
    return readErr
}
```

## References

- [Python Detail Records](v2/availabilityzones/python-detail-records.md)
- [Policy](https://docs.openstack.org/nova/2024.1/configuration/policy.html)
- [Implementation Plan](../docs/implementation-plan.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Proxy availability_zones](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1854-L1874)
- [AvailabilityZone descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/availability_zone.py#L16-L36)
- [Cloud list_availability_zone_names](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L255-L276)
