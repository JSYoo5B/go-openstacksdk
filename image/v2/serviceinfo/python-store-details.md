# Python Store Details

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/stores`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.New`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.API.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.API.ListStores`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.API.AllStores`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithListStoresOptions`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithListStoresDetails`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithListStoresMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithListStoresPaginated`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithListStoresQuery`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithListStoresHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.NewWithDependencies`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.API.ListStoreRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.API.AllStoreRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithStoreRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithStoreRecordListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithStoreRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithStoreRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithStoreRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithStoreRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithStoreRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithStoreRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithStoreRecordListFilters`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithStoreRecordListDetails`

- ListStores/AllStores default to basic GET info/stores with actual200 only. Local Details true chooses info/stores/detail and never becomes query; server403/404 and feature-unavailable remain errors with no basic fallback. Store embeds Metadata with actual per-page Header/status and raw canonical/unknown fields.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoDefaultRoutesAndEvidence`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreDiscoveryRoutesAndRawModels`.
- Store stores envelope is mandatory nonnull array and rows must be objects. Canonical id/description missing/null become empty strings, nonnull require strings. Canonical default/read-only are optional flags accepting JSON bool or lexical true/false strings; is_default/case variants are raw unknown only. Properties missing/null remain nil else raw object; optional type is string and weight exactint64, without coercion. Store IDs are passive discovery data, never derived request routes.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoCanonicalModelsAndPresence`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreCanonicalAndNullableDecoding`.
- Concrete defaults Details=false,Limit=0,Marker empty,MaxItems=0,Paginated=nil mean basic route, omitted wire limit/marker, uncapped allpages. Negative controls, malformed limit/marker, invalidUTF8, nil/cause-returning callbacks, body/argument/microversion and reserved query controls reject before HTTP. Wire query helper is a Go extension and does not implement Python Body filters or unknown-query discard; ordinary headers use protected shared validation.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoCompletePreflight`, `image/v2/serviceinfo/options_test.go:TestServiceInfoOptionsRejectUnsupportedInputsBeforeHTTP`.
- List creation performs no HTTP and copies callback slice. Each iteration applies callbacks once to owned independent query/header/pointers and source snapshots. Whole typed replacement snapshots Paginated at creation/application; later leaf options replace settings in order. Retained callback handles cannot change prepared request or subsequent callbacks; standard options and iterator reuse are concurrent-safe.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoPreparedOptionsAndLazyParallelReuse`, `image/v2/serviceinfo/options_test.go:TestServiceInfoOptionsSnapshotReplacementAndRetainedHandles`, `image/v2/serviceinfo/options_test.go:TestServiceInfoOptionsConcurrentReusableSnapshots`.
- Paginated=false stops at firstpage and MaxItems is a local raw-row cap, stops before unused row decode or next-link interpretation, and never generates a wire limit hint. Only advertised Link/body-next/HTTPLink continues, bounded to fixed origin/reverse-prefix/selectedpath/initialnonpaginationquery and cycle checks; empty page stops. Missing link ends even a nonempty response. Explicit caller limit/marker remain wire preferences but never authorize automatic marker fallback, because actual Glance discovery ignores those queries and returns the same full store set.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoPaginationCapsAndRawMarkers`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCorePagingUsesAdvertisedLinksAndStopsAtControls`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreServerIgnoresPaginationQueries`.
- ListStores may yield valid prior rows then terminal row/page error. AllStores discards prior rows and returns nil on terminalerror; successful empty collection returns nonnil empty slice. Cap and firstpage control may stop before otherwise unused malformed rows/continuations without inventing deployment pagination support.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoPaginationCapsAndRawMarkers`, `image/v2/serviceinfo/contracts_test.go:TestServiceInfoDefaultRoutesAndEvidence`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCorePagingUsesAdvertisedLinksAndStopsAtControls`.
- Context/image client/provider/base/source headers validate before options, after callbacks and before each request/page. Capture origin/reverse prefix/provider/header policy, retain the original provider live token and reject source/provider/target replacement. Fixedrequest retains configured same-scope redirect callbacks and rejects method/origin/path/query changes before transport.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoCompletePreflight`, `image/v2/serviceinfo/contracts_test.go:TestServiceInfoProviderSourceAndRetryBoundaries`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreCapturesSourceAndGuardsLaterPages`.
- Accepted actual200 owns whole raw body/Header/StatusCode and closes the body once. Read/Close/JSON/full-root UTF8/model/context failures preserve ResponseError evidence and independent original/custom causes; accepted body failures do not replay. Native rejected errors and native ErrUnableToReauthenticate.ErrOriginal/ErrReauth field ownership remain unchanged. Shared DoJSON rejects RetryFunc ownership-field mutation and rechecks captured200 even when native OkCodes expands.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoAcceptedFailuresAndNativeCompatibility`, `image/v2/serviceinfo/contracts_test.go:TestServiceInfoProviderSourceAndRetryBoundaries`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreBodyAndModelFailuresKeepEvidence`.
- Exact canonical keys alone populate typed fields. Full raw root is valid UTF8 flat JSON object; unknown and case-variant fields remain raw without overriding canonical fields. Raw model Body/Header/status are independently owned, preserving unknown large numbers as json.RawMessage.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoCanonicalModelsAndPresence`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreCanonicalAndNullableDecoding`, `internal/rest/response_validation_test.go:TestResponseValidationChecksWholeBodyBeforeGetAndListDecoders`, `internal/rest/response_validation_test.go:TestResponseValidationRunsPerPageAndPreservesLazyBreak`.
- Generated ImageV2.ServiceInfo and upper Image.API share the configured authenticated client/prefix/liveauth with Connection. Store list_only and ImportInfo service_info inventory are SDK-owned records; original native ImageImport.Get route/signature/result remains unchanged. Generator rejects native Get/canonical model drift without inventing an upstream operation.
  Tests: `connectiontest/connection_image_serviceinfo_test.go:TestConnectionImageServiceInfoSharesClientPrefixAndLiveAuthentication`, `internal/cmd/sdkgen/glance_serviceinfo_test.go:TestGlanceServiceInfoRegistryAndNativeCompatibility`, `internal/cmd/sdkgen/glance_serviceinfo_test.go:TestGlanceServiceInfoRejectsNativeGetAndCanonicalModelDrift`.
- Owned basic StoreRecord adds the six-field declared view. Body id/name/description are nullable untyped values, is_default uses nullable Python truthiness including nonempty false string, properties is a dict descriptor and name never synthesizes absent id. Ordered aliases/duplicate keys and raw precision are preserved beside independent Wire/Envelope/Header/status.
  Tests: `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsProjectionAndOwnedReceipts`, `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsDescriptorsAliasesAndLocations`.
- Basic /info/stores forwards limit/marker and applies canonical id/name/description/is_default/properties filters locally after descriptor conversion. Unknown and remote default filter keys are ignored before encoding; recursive dictionary subset and exact JSON semantic comparisons reuse the shared selector.
  Tests: `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsCanonicalFiltersAndUnknownQueryPolicy`.
- Nil limit is omitted, explicit zero is preserved, positive raw cap can supply a limit hint and raw rows are capped before filtering. Shared guarded pagination supports collection links, Source-falsey next, exact /v2/info/stores alias, initial-limit marker fallback and partial All. Repeated full discovery pages stop at a cycle error rather than looping indefinitely.
  Tests: `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsSourcePagingAndRawCaps`.
- Lazy/reusable options capture maps/pointers/bytes and Source before callbacks/current location. Without recognized Body attributes explicit row computed location survives; any recognized Body recomputes current location. Captured client/target/outer guard faults stay sticky through read/Close and yields.
  Tests: `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsLazyRepeatedCapturedOptions`, `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsDescriptorsAliasesAndLocations`, `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsBodyFailuresAndStickySourceGuards`.
- Actual 200..399 pages require whole UTF-8 JSON and mandatory stores envelopes with object/array rows. Consumed constructor binding collisions, malformed rows and accepted IO/shape/continuation faults remain errors with full original page evidence; unused rows are skipped by early break/cap.
  Tests: `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsEnvelopeStatusesAndConstructorFailures`, `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsBodyFailuresAndStickySourceGuards`.
- Both Connection entries retain their reviewed client/location behavior. Owned default basic discovery remains /info/stores, and the new opt-in detail selector is separate from legacy typed strict200 detail schema; no local role gate is introduced.
  Tests: `connectiontest/connection_image_serviceinfo_records_test.go:TestConnectionImageServiceInfoRecordsShareLocationAndNativeClient`, `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsPreflightAndLegacyDetailCompatibility`.
- Owned Details=true selects /info/stores/detail without a details query, reuses exactly the six-field Store declaration, preserves extra detailed fields only in Wire, and applies canonical filters after descriptor conversion.
  Tests: `image/v2/serviceinfo/python_store_details_test.go:TestPythonStoreRecordsDetailsSelectsFixedRouteAndSchema`.
- Details is owned in reusable options and can be reset to false; guarded detail paging retains partial rows/error receipts, caps raw rows before filters, surfaces server403 and context errors, while default basic and typed detail contracts remain intact.
  Tests: `image/v2/serviceinfo/python_store_details_test.go:TestPythonStoreRecordsDetailsBasicSelectionAndReusableOptions`, `image/v2/serviceinfo/python_store_details_test.go:TestPythonStoreRecordsDetailsPagingPartialAndCaps`, `image/v2/serviceinfo/python_store_details_test.go:TestPythonStoreRecordsDetailsHasNoLocalRoleGateAndOwnsFailures`, `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsPreflightAndLegacyDetailCompatibility`.

Python/Go differences:

- Legacy typed ListStores/AllStores retain their strict200/schema/partial-result profile. Owned ListStoreRecords/AllStoreRecords now support both fixed basic and detail paths with the same six-field Source declaration and independent Wire receipts.
- This extends the existing owned pager rather than claiming full Python parity: 200..399 acceptance, UTF8/safe fixed continuation routes, dictionary next handling, cycle checks and owned JSON comparisons remain reviewed Go bounds; Source100..199/malformed links/runtime behavior can differ.
- Arbitrary Proxy base_path/conflicting/JMESPath/session/cache/Requests behavior remains outside this concrete option profile and is tracked with SDK-R1/C1/S1 and whole-SDK objectives. No authenticated deployment/controller/permission or release verification is claimed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack


def main():
    conn = openstack.connect()
    for store in conn.image.stores(details=True, is_default=True, paginated=False):
        print(store.id, store.properties)


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "log"
    "os"

    image "github.com/JSYoo5B/go-openstacksdk/image/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    endpoint := os.Getenv("OS_IMAGE_ENDPOINT")
    if endpoint == "" { log.Fatal("OS_IMAGE_ENDPOINT is required") }
    provider := &gophercloud.ProviderClient{TokenID: os.Getenv("OS_TOKEN")}
    client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: gophercloud.NormalizeURL(endpoint), Type: "image"}
    service := image.New(client)
    for row, err := range service.ServiceInfo.ListStoreRecords(context.Background(),
        serviceinfo.WithStoreRecordListDetails(true),
        serviceinfo.WithStoreRecordListPaginated(false),
        serviceinfo.WithStoreRecordListFilter("is_default", true),
    ) {
        if err != nil { log.Fatal(err) }
        log.Printf("id=%s properties=%s status=%d", row.Resource.Body["id"], row.Resource.Body["properties"], row.StatusCode)
    }
}
```
