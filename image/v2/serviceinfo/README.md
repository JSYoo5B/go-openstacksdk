# Serviceinfo

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
  Tests: `connection_image_serviceinfo_test.go:TestConnectionImageServiceInfoSharesClientPrefixAndLiveAuthentication`, `internal/cmd/sdkgen/glance_serviceinfo_test.go:TestGlanceServiceInfoRegistryAndNativeCompatibility`, `internal/cmd/sdkgen/glance_serviceinfo_test.go:TestGlanceServiceInfoRejectsNativeGetAndCanonicalModelDrift`.
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
  Tests: `connection_image_serviceinfo_records_test.go:TestConnectionImageServiceInfoRecordsShareLocationAndNativeClient`, `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsPreflightAndLegacyDetailCompatibility`.
- Owned Details=true selects /info/stores/detail without a details query, reuses exactly the six-field Store declaration, preserves extra detailed fields only in Wire, and applies canonical filters after descriptor conversion.
  Tests: `image/v2/serviceinfo/python_store_details_test.go:TestPythonStoreRecordsDetailsSelectsFixedRouteAndSchema`.
- Details is owned in reusable options and can be reset to false; guarded detail paging retains partial rows/error receipts, caps raw rows before filters, surfaces server403 and context errors, while default basic and typed detail contracts remain intact.
  Tests: `image/v2/serviceinfo/python_store_details_test.go:TestPythonStoreRecordsDetailsBasicSelectionAndReusableOptions`, `image/v2/serviceinfo/python_store_details_test.go:TestPythonStoreRecordsDetailsPagingPartialAndCaps`, `image/v2/serviceinfo/python_store_details_test.go:TestPythonStoreRecordsDetailsHasNoLocalRoleGateAndOwnsFailures`, `image/v2/serviceinfo/store_records_test.go:TestStoreRecordsPreflightAndLegacyDetailCompatibility`.

Python/Go differences:

- Legacy typed ListStores/AllStores retain their strict200/schema/partial-result profile. Owned ListStoreRecords/AllStoreRecords now support both fixed basic and detail paths with the same six-field Source declaration and independent Wire receipts.
- This extends the existing owned pager rather than claiming full Python parity: 200..399 acceptance, UTF8/safe fixed continuation routes, dictionary next handling, cycle checks and owned JSON comparisons remain reviewed Go bounds; Source100..199/malformed links/runtime behavior can differ.
- Arbitrary Proxy base_path/conflicting/JMESPath/session/cache/Requests behavior remains outside this concrete option profile and is tracked with SDK-R1/C1/S1 and whole-SDK objectives. No authenticated deployment/controller/permission or release verification is claimed.

### `python:image/v2/get_import_info`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.New`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.API.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.API.GetImportInfo`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithGetImportInfoHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.NewWithDependencies`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.API.GetImportInfoRecord`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithImportRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithImportRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.WithImportRecordHeaders`

- GetImportInfo performs ID-free singleton GET info/import with actual200 only and no list/cache/capability gate. Flat root metadata carries actual Header/status/raw unknown fields. Original native ImageImport.Get still performs the same discovery wire route and retains its existing typed callable/model.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoDefaultRoutesAndEvidence`, `image/v2/serviceinfo/contracts_test.go:TestServiceInfoAcceptedFailuresAndNativeCompatibility`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreDiscoveryRoutesAndRawModels`.
- Canonical import-methods missing/null stays nil; nonnull is object. Nested description/type missing/null become empty strings else require strings. Value missing/null stays nil, emptyarray stays nonnil empty, and nonnullarray requires string entries without enum filtering. Nested raw Body retains unknown future constraints/exact numbers; alternate envelopes are never unwrapped.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoCanonicalModelsAndPresence`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreCanonicalAndNullableDecoding`.
- GetImportInfo concrete options support ordinary validated headers only. Nil callbacks, callback causes, fields/query/arguments/microversion and protected/malformed headers fail before HTTP. Each callback receives owned configuration; retained handles cannot mutate the final prepared header policy.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoCompletePreflight`, `image/v2/serviceinfo/contracts_test.go:TestServiceInfoPreparedOptionsAndLazyParallelReuse`, `image/v2/serviceinfo/options_test.go:TestServiceInfoOptionsRejectUnsupportedInputsBeforeHTTP`.
- Context/image client/provider/base/source headers validate before options, after callbacks and before each request/page. Capture origin/reverse prefix/provider/header policy, retain the original provider live token and reject source/provider/target replacement. Fixedrequest retains configured same-scope redirect callbacks and rejects method/origin/path/query changes before transport.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoCompletePreflight`, `image/v2/serviceinfo/contracts_test.go:TestServiceInfoProviderSourceAndRetryBoundaries`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreCapturesSourceAndGuardsLaterPages`.
- Accepted actual200 owns whole raw body/Header/StatusCode and closes the body once. Read/Close/JSON/full-root UTF8/model/context failures preserve ResponseError evidence and independent original/custom causes; accepted body failures do not replay. Native rejected errors and native ErrUnableToReauthenticate.ErrOriginal/ErrReauth field ownership remain unchanged. Shared DoJSON rejects RetryFunc ownership-field mutation and rechecks captured200 even when native OkCodes expands.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoAcceptedFailuresAndNativeCompatibility`, `image/v2/serviceinfo/contracts_test.go:TestServiceInfoProviderSourceAndRetryBoundaries`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreBodyAndModelFailuresKeepEvidence`.
- Exact canonical keys alone populate typed fields. Full raw root is valid UTF8 flat JSON object; unknown and case-variant fields remain raw without overriding canonical fields. Raw model Body/Header/status are independently owned, preserving unknown large numbers as json.RawMessage.
  Tests: `image/v2/serviceinfo/contracts_test.go:TestServiceInfoCanonicalModelsAndPresence`, `image/v2/serviceinfo/core_test.go:TestServiceInfoCoreCanonicalAndNullableDecoding`, `internal/rest/response_validation_test.go:TestResponseValidationChecksWholeBodyBeforeGetAndListDecoders`, `internal/rest/response_validation_test.go:TestResponseValidationRunsPerPageAndPreservesLazyBreak`.
- Generated ImageV2.ServiceInfo and upper Image.API share the configured authenticated client/prefix/liveauth with Connection. Store list_only and ImportInfo service_info inventory are SDK-owned records; original native ImageImport.Get route/signature/result remains unchanged. Generator rejects native Get/canonical model drift without inventing an upstream operation.
  Tests: `connection_image_serviceinfo_test.go:TestConnectionImageServiceInfoSharesClientPrefixAndLiveAuthentication`, `internal/cmd/sdkgen/glance_serviceinfo_test.go:TestGlanceServiceInfoRegistryAndNativeCompatibility`, `internal/cmd/sdkgen/glance_serviceinfo_test.go:TestGlanceServiceInfoRejectsNativeGetAndCanonicalModelDrift`.
- Owned ImportRecord adds the declared four-field view id/name/import_methods/location. Inherited id/name are nullable untyped JSON with no alternate ID; the dict descriptor preserves arbitrary nested objects and raw precision, maps nonnull nonobjects to {}, and retains null defaults. Ordered canonical/remote aliases and duplicate first-position/last-value handling match the pinned collector.
  Tests: `image/v2/serviceinfo/import_record_test.go:TestImportInfoRecordDictionaryAliasesAndDefaults`.
- ID-free query-free GET /info/import accepts actual 200..399. Empty or invalid JSON syntax yields bare declared defaults and Wire nil; valid parsed nonobjects and invalid whole UTF-8 fail with actual proof. Unknown/self/response computed location remain passive Wire and do not enter Body projection.
  Tests: `image/v2/serviceinfo/import_record_test.go:TestImportInfoRecordAcceptedToleranceAndRejectedNative`, `image/v2/serviceinfo/import_record_test.go:TestImportInfoRecordProjectionAndOwnedReceipts`.
- Resource, Wire, Envelope, Header and status are independently owned. Location is captured before option callbacks, ignores response computed location and remains fixed; header helper inputs and callback configurations snapshot independently with whole-options replacement and last-wins merges.
  Tests: `image/v2/serviceinfo/import_record_test.go:TestImportInfoRecordProjectionAndOwnedReceipts`, `image/v2/serviceinfo/import_record_test.go:TestImportInfoRecordLocationAndConcreteOptionSnapshots`.
- Original API/client/provider/endpoint/raw ResourceBase/microversion/type and outer operation guards stay sticky through options and read/Close. Accepted body faults retain actual receipt and original causes without replay; rejected native errors preserve native policy and do not fabricate a successful record.
  Tests: `image/v2/serviceinfo/import_record_test.go:TestImportInfoRecordBodyFailuresAndStickySourceGuards`, `image/v2/serviceinfo/import_record_test.go:TestImportInfoRecordAcceptedToleranceAndRejectedNative`.
- Invalid context/source/header/option input fails before HTTP and Source is validated before callbacks. Legacy strict200 GetImportInfo and the native ImageImport callable/model stay unchanged beside the new owned dictionary projection.
  Tests: `image/v2/serviceinfo/import_record_test.go:TestImportInfoRecordPreflightAndLegacyABI`, `internal/cmd/sdkgen/glance_serviceinfo_test.go:TestGlanceServiceInfoRegistryAndNativeCompatibility`.
- Both Connection Image and ImageV2 supply the same native service client, live provider authentication and captured current location to ServiceInfo. Owned import and basic stores preserve separate response proof channels across calls and pages without an extra discovery gate.
  Tests: `connection_image_serviceinfo_records_test.go:TestConnectionImageServiceInfoRecordsShareLocationAndNativeClient`.

Python/Go differences:

- The existing seven strict typed GetImportInfo contracts above remain unchanged; GetImportInfoRecord provides the pinned dictionary/default/fetch translation without replacing native ImageImport.Get.
- Go returns an independent read-only ImportRecord and concrete header helpers rather than mutable Python cached Resource identity. These helpers and explicit CloudLocation injection are Go extensions; no ID/Name/Resource/query/body is required by the pinned no-argument singleton proxy.
- Actual 200..399, whole UTF-8, protected headers, exact JSON precision and fixed-route sticky guards are Go bounds; Python JSON parser/nonfinite and binary float corner behavior may differ. Physical Close faults are errors and native rejected IO policy remains unchanged.
- Full Resource/session/cache/discovery lifecycle remains a separate whole-SDK objective. Source inspection and local HTTP tests do not establish authenticated cloud availability or deployment permission.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
)

func discoverStores(ctx context.Context, conn *openstack.Connection) ([]*serviceinfo.Store, error) {
    service, err := conn.ImageV2(ctx)
    if err != nil {
        return nil, err
    }
    return service.ServiceInfo.AllStores(ctx,
        serviceinfo.WithListStoresDetails(true),
        serviceinfo.WithListStoresMaxItems(20),
    )
}
```

```python
for store in conn.image.stores(details=True, max_items=20):
    print(store.id, store.description, store.is_default)
```

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
)

func discoverImportMethods(ctx context.Context, client *gophercloud.ServiceClient) (*serviceinfo.ImportInfo, error) {
    return image.New(client).API.ServiceInfo.GetImportInfo(ctx)
}
```

```python
info = conn.image.get_import_info()
methods = info.import_methods
print(methods)
```

## References

- [Records](records.md)
- [Python Store Details](python-store-details.md)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#list-stores)
- [detail API](https://docs.openstack.org/api-ref/image/v2/index.html#list-stores-detail)
- [basic controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L53-L93)
- [detail controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L165-L195)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#import-methods-and-values-discovery)
- [Proxy.stores/get_import_info](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2273-L2295)
- [Store/Import](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L22-L44)
- [Contracts Test](contracts_test.go)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Connection Image Serviceinfo Test](../../../connection_image_serviceinfo_test.go)
- [Glance Serviceinfo Test](../../../internal/cmd/sdkgen/glance_serviceinfo_test.go)
- [Response Validation Test](../../../internal/rest/response_validation_test.go)
- [Usage](usage.md)
