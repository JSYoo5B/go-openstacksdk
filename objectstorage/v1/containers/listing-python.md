# Listing Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/containers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.New`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPaginated`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListPrefix`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListFormat`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListMaxPages`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordListSinglePage`

- The container Collection preserves native Swift marker pagination, prefix/explicit-limit query and typed count/byte fields while applying controls before local exact-name filtering.
  Tests: `api/native_scope_list_controls_test.go:TestNativeScopeListControlsFiveBindingsRawCapBeforeNameAndHostFilters`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsSinglePageSkipsForeignAndCyclicContinuation`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsCrossPageQueryFixedParentAndIteratorReuse`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsZeroDefaultsFollowNativeContinuation`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsKeepWholePageDecodeErrors`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsParentCancellationAndConsumerBreak`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsPreflightNegativeAndTroveQueryPolicy`.
- Lazy bodyless GET uses exact authenticated escaped account root, Accept application/json and no invented format default. Explicit server prefix/format/limit/marker follow Requests JSON-domain null/bool/repeated/number/key iteration, while declared nonserver id/name/count/bytes become descriptor-projected local filters. Unknown queries including end_marker/full_listing are ignored before custom marshaling; paginated duplicates the pinned proxy True and fails before HTTP.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListDefaultProjectionAndIndependentReceipts`, `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListQueryClassificationAndOptionOwnership`, `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListPublicFixtureWireAndSourceCapture`.
- Owned ContainerRecord retains inherited id/name/count/bytes, sixteen Header descriptors and nullable location as 21 declared fields, preserving exact big integers, Source integer/bool conversion (int descriptor preserves bool; ordinary nonnumeric string/container becomes0; nondecimal digit-only string errors) and alias presence vs explicitnull/false/zero/empty. Row Header attrs alone populate descriptors: account response Header/metadata never becomes per-container observations. Wire has all row fields, whole Envelope/Header/status retain actual page, row metadata is independently owned with original types retained in Wire.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListDefaultProjectionAndIndependentReceipts`.
- Shared Python scalar/number/bool equality and recursive _dict_filter preserve truthy-scalar empty predicate, missing nested None and meaningful mapping-get faults. WithRecordListQuery preserves top-level option order and raw nested JSON preserves ordered short-circuit; no swallowed scalar AttributeError is turned into absence.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListLocalEqualityAndDictionaryShortCircuit`, `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListOrderedLocalPredicatesAvoidLaterMappingFault`.
- Account count pagination uses total raw consumed rows before local filters and immutable retained last raw id/name, including Python integer string/Unicode/count bounds. Initial raw limit truthiness controls nonempty short-page fallback; 0/false differ from strings 0/False. Empty array stops before malformed count/Link. Neither ignored full_listing=False nor end_marker disables or truncates pinned named pagination.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListCountPagingUsesRawRowsAndRetainedMarker`, `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListLimitTruthinessAndIgnoredFullListing`.
- Root array and Source singleton object support body pagination key/top next before RFC Link, then count/limit fallback. Valid RFC links keep exact scoped account origin/escaped path and initial filters/limit; omitted query is inherited. Foreign route/filter/limit, malformed/conflicting links and repeated page/marker fail with actual accepted receipt without another physical request.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListAdvertisedContinuationAndAccountGuards`.
- MaxItems counts raw filtered rows and supplies initial limit hint only for falsey limit; cap, single-page and consumer break avoid unused row/continuation work. MaxPages defaults10000 and explicit bounded iteration fails with last physical page proof. Full JSON decode precedes lazy row construction, so malformed whole JSON fails immediately and valid later bad rows are ignored after an already-reached raw cap/break.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListRawCapsAndPageBoundsStopBeforeUnusedRows`.
- Actual accepted100..399 includes transport-exposed199; blank204/malformed/null/scalar/invalidUTF8, list constructor cls/connection/microversion/_synchronized collisions and invalid metadata/typed descriptors preserve exact whole page/status/Header. Source self is not passed to constructor. All400..599 failures, including404, retain native HTTP method/URL/body/status/Header without missing suppression.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListWrongShapesAndConstructorCollisionsKeepPageProof`, `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListRejectedStatusesAreNeverMissingSuppressed`.
- Common REST capture/guards/rejection handling preserves native HTTP plus physical Read/Close causes, custom cancellation and sticky source changes even when restored. Physical faults prevent willing retries and exactly one Close/request is observed across accepted199/200/304 and rejected400/403/404/429/500/503/599. Partial error records have actual receipt and no invented decoded Resource/Wire. When shared REST returns nil Response for a native rejection, the leaf restores receipt solely from actual ErrUnexpectedResponseCode Body/ResponseHeader/Actual and retains every original cause; it never rereads/recloses or admits a rejected page.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListPhysicalReadCloseFailuresAndStickySource`, `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListRejectedStatusesAreNeverMissingSuppressed`.
- Fixed HeaderPolicy enforces authentication/framing/foreign service version absence, selected Swift version or absence, bodyless GET and operation Accept across native retry callbacks. Reserved injection/version removal keeps initial503 and callback cause, one Close/request and ErrInvalidOption. Ordinary retry headers and provider live token retain clean retry; per-operation version and private Type normalization do not mutate selected shared client.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListRetryHeaderOwnershipAndActualReceipt`.
- Concrete option factories and callback outputs snapshot nested raw fields/header maps/pointers; retained configs cannot rewrite wire predicates or projected rows. Source is captured before option callbacks and first consumption; nil input/invalid controls/header/UTF8/cancel fail without forged receipts or HTTP. Independent concurrent reuse and public Gophercloud testhelper actual HTTP fixture cover escaped account path, one account GET per page and source callback rejection.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListQueryClassificationAndOptionOwnership`, `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListPreflightCancellationAndRetainedOptionOwnership`, `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListConcurrentOwnedOptionReuse`, `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListPublicFixtureWireAndSourceCapture`.
- Resource.list operation Accept application/json overrides captured service Accept unless explicit method headers select another value. Conditional If-Match/If-None-Match/If-Modified-Since/If-Unmodified-Since values or absence remain fixed through retry/OmitHeaders; initial503 actual receipt and callback cause survive with one request/Close.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListOperationAcceptAndConditionalRetryOwnership`.
- Shared SDK-C1 exact-decimal JSON equality deliberately avoids CPython float rounding: actual integer9007199254740992 does not match raw filter9007199254740993.0; actual Wire and numeric descriptor remain unchanged.
  Tests: `objectstorage/v1/containers/python_list_contracts_test.go:TestPythonContainerListExactDecimalEqualityPolicy`.

Python/Go differences:

- Go exposes these local controls through the common Collection or fixed-parent scope; existing native typed List option signatures remain unchanged. Zero means unlimited and negative values fail lazily before HTTP.
- The native path does not infer a wire limit from max_items and stops exactly at the raw row cap. Native whole-page extraction or IsEmpty may still reject unused trailing rows; continuation retains the native pager and SDK cycle policy.
- The pinned Swift proxy explicitly forces paginated=True. First-page consumption is an explicit Go scope/Collection extension rather than a proxy keyword default mapping.
- This maps exactly one unresolved pinned named Source operation; already-mapped Gophercloud native List/generated transport is not counted again. Whole mutable Source Resource/cache/dirty-state/session/Adapter/Connection/JMESPath/arbitrary base_path behavior remains global SDK-R1/C1/S1.
- Go returns owned Source-shaped RawResource fields, row Wire and actual whole page receipts. location stays null without Python Connection synthesis. Existing ContainerRecord.Metadata map[string]string independently represents row constructor metadata using shared PythonString; original arbitrary JSON metadata remains in Wire. Pinned BaseResource class-level metadata sharing is intentionally removed.
- Pinned Source fixes paginated=True. end_marker/full_listing are genuinely unknown query fields and stay ignored; dedicated SinglePage and MaxPages(default10000) are explicit Go controls. MaxItems remains raw-cap Source behavior. Typed Limit rejects negative numbers but raw JSON limit preserves Source request/truthiness behavior including negative values and strings.
- Concrete headers/max_items/microversion options model method controls separately from the raw server/local query map. Query factories preserve local predicate call order; caller RecordListOpts.Query Go map has stable sorted remainder order because Go maps have no insertion ordering. Raw nested JSON retains its own dictionary predicate order. JSON-domain/shared Python conversion limits are retained; custom Python objects/runtime repr and full arbitrary session behavior are outside this named profile.
- Go fixes the pinned _get_next_link RFC path reading response.links[next][uri] although ordinary Requests Link parser exposes url; valid RFC links are supported with account origin/escaped-path/filter/limit and cycle/page guards. Version-prefix stripping/arbitrary traversal and ambiguous link/count Header normalization are explicit Go scope/protocol protections. Omitted continuation query persists for reliable Swift account pagination.
- Go operation HeaderPolicy protects initially forbidden auth/framing/foreign-service headers through retries and physical attempts while native ordinary retry headers/live X-Auth-Token/reauth remain available. Selected/per-operation Swift microversion is preserved or absent without invented discovery/version ceiling or client mutation. Actual server unsupported version/ACL/quota errors remain HTTP observations.
- Fixture/race/standalone guide compile are root integration responsibilities. No authenticated Swift deployment, ACL/replication durability or whole-project acceptance is inferred from these local tests.
- Shared exact-decimal JSON equality is an explicit SDK-C1 policy, not complete Python binary64 equality: actual9007199254740992 and raw9007199254740993.0 remain unequal. Conditional HTTP Header values/absence are operation owned through retries; ordinary unrelated headers retain native policy.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prefix", default="assets/")
    parser.add_argument("--page-size", type=int)
    parser.add_argument("--max-items", type=int, default=0)
    args = parser.parse_args()
    query = {"prefix": args.prefix, "max_items": args.max_items}
    if args.page_size is not None:
        query["limit"] = args.page_size
    conn = openstack.connect()
    for container in conn.object_store.containers(**query):
        print(container.name, container.count, container.bytes)


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

func main() {
    prefix := flag.String("prefix", "assets/", "container name prefix")
    pageSize := flag.Int("page-size", 0, "server page size; zero disables limit fallback")
    maxItems := flag.Int("max-items", 0, "raw rows to consume; zero means no row cap")
    flag.Parse()
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { log.Fatal(err) }
    options := []containers.RecordListOption{
        containers.WithRecordListPrefix(*prefix),
        containers.WithRecordListLimit(*pageSize),
        containers.WithRecordListMaxItems(*maxItems),
    }
    for record, err := range swift.Containers.ListRecords(ctx, options...) {
        if err != nil {
            if record != nil {
                log.Printf("actual page status=%d body=%s", record.StatusCode, record.Envelope)
            }
            log.Fatal(err)
        }
        fmt.Printf("name=%s count=%s bytes=%s HTTP=%d\n",
            record.Resource.Body["name"], record.Resource.Body["count"],
            record.Resource.Body["bytes"], record.StatusCode)
    }
}
```

## References

- [Python List Contracts Test](python_list_contracts_test.go)
