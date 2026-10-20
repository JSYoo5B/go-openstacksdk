# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/regions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.RegionRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.WithRecordListJMESPath`

- All with resource.WithQuery forwards parent_region_id, limit and marker to GET regions as Python regions(**query) does for mapped parameters and follows links.next.
  Tests: `identity/v3/regions/python_parity_test.go:TestPythonRegionListQueryAndPaging`.
- Region inherits Resource directly with raw nullable description/links/parent_region_id/id/name and no declared Headers. Parent region is server query; name is local Body, unknown kwargs are discarded before marshal, and independent declared/Wire/Envelope/header/status channels retain actual proof.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsRawDefaultsLocalNameAndIndependentEvidence`.
- Source root dictionary links ignored with root-key precedence, null is an iterable error with proof, explicit short limit pages continue by last raw ID until empty and local-filtered last row still supplies cursor; raw max_items and break stop lookahead.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsSourceDictionaryLinksAndPresence`, `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsFilteredMarkerRawCapAndBreak`.
- Loaded ordered local subset/equality follows CPython float rounding/underflow and bool/integer equality; earlier mismatch avoids later bad shape and reached shape failure has actual proof.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsOrderedLoadedLocalEquality`.
- Opaque unconsumed generator JMESPath preserves HTTP0 projection/type/literal errors/results and generator/nested identity without fake JSON or receipt; falsey/nonstring remains lazy.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsProxyExpressionsRemainOpaqueUntilConsumed`.
- Typed/formal controls and one-pass conflicting attrs preserve restored parent query/header/version, original expression priority, unknown discard and HTTP0 collisions. Options/source/cancel before and after callbacks are sticky with owned maps/raw JSON.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsProxyFormalRestorationAndBoundCollision`, `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsOwnedOptionsStickySourceAndCancellation`.
- Explicit location clones configured current project, ignores unknown wire project/zone and normalizes no-descriptor zone null. New profile opts Source valid100..399; raw bool limit is intentionally decimal-Go-only HTTP0 invalid rather than wire1/0.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsExplicitLocationStatusAndBooleanLimitBoundary`.
- 28 physical accepted/rejected Read/Close/restored Source/custom cancel lanes keep actual proof, Close once, native401 independently inspectable ErrOriginal/ErrReauth and zero retry/backoff/reauth. Clean401/503/429/498 retains live token and owned route/header/query.
  Tests: `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsPhysicalFaultsKeepActualProofAndStopRetry`, `identity/v3/regions/python_records_test.go:TestPythonRegionRecordsCleanNativeRetryUsesLiveAuthAndCapturedHeaders`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Library-owned concrete functional options, raw nullable Source view and separate actual receipt preserve existing native APIs. No user builder interface or new transport framework.
- Go caps/limit require nonnegative decimal integers. Source raw limit True/False is sent as True/False and truthy True activates fallback; Go rejects bool before HTTP and does not substitute1/0. Safe fixed origin/path/query/header/cursor/cycle bounds are explicit.
- Source rootlink JSON precedence/dictionary-ignore/null-error is preserved. RFC HTTP Link uses explicit Go interoperability for ordinary Requests next[url] versus Source next[uri] bug.
- Unconsumed Proxy generator is opaque and no fake JSON/HTTP association is created. Go iter.Seq2 reuse reissues captured requests instead of sharing Python one-shot state.
- Standalone default location is null. Explicit cloned recorded facts are a Go convenience; Region has no zone descriptor and normalizes zone null, while GroupUser reuses existing User explicit-zone convenience. Unknown response scope fields are not trusted.
- Already selected Keystone v3 native API required. Mutable Resource object/group subclass routing, automatic Connection location/cache, generic synchronized lifecycle and session remain SDK-R1/C1/S1.
- Generic SDK-R1/C1/S1 response cache, arbitrary synchronized Resource/object lifecycle and automatic Connection/session bridge are not completed. These two finite mappings have supported delta0; root Go/race/main gates must pass before catalog promotion.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack


def main():
    conn = openstack.connect()
    for row in conn.identity.regions(
        parent_region_id="Root", description="operations", limit=100,
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.Regions.ListRecords(ctx,
        regions.WithRecordListQuery("parent_region_id", "Root"),
        regions.WithRecordListQuery("description", "operations"),
        regions.WithRecordListLimit(100),
        regions.WithRecordListMaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("%s\n", data)
    }
}
```

## References

- [Region](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/region.py)
