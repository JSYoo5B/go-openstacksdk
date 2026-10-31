# Metadef Object Namespace Record Lists

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/metadef_namespaces`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListVisibility`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListSortKey`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListSortDir`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListResourceTypes`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListSinglePage`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.List`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.All`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.NewWithDependencies`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListFilters`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListVisibility`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListResourceTypes`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListSortKey`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListSortDir`

- Literal namespace identity, fixed source capture and strict preflight: name is nonempty UTF8<=80runes; colon/Unicode/spaces remain, route-breaking/control/dot syntax rejected before callbacks/HTTP. No Find, Ref/name resolver, generic Resources or response-driven routing. Fixed original provider/prefix/header/microversion with live original auth; source checks after callbacks/body and before consumed rows/pages.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCompletePreflight`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Canonical Namespace is a validUTF8 nonnull object with exact-case optional string fields namespace/display_name/description/visibility/owner/self/schema and protected bool. Missing/null nil, empty/false present; literal nullable dates are unparsed. Nested definitions/links/unknown shapes stay independently owned raw JSON with exact large-number precision; no ID/Name/Links synthesis. Wrong types are atomic model failures with actual whole response proof.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`.
- Concrete FullOpts replaces all configuration; helper values, maps, optional pointers and option slice are copied, per-callback config independently copied before/after and callback once per request/iteration. Nil/error callbacks and protected headers fail before HTTP; helper reuse/parallel iterations retain snapshots.
  Tests: `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsParallelReusableHelpers`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Strict original Create201/Get-List-Update200/Delete204 accepted statuses. Accepted read/Close/UTF8/JSON/model/context/source failure preserves original bytes/header/status and read/Close/ctx.Err/customCause in owned ResponseError, without accepted-body replay. Delete actual204 preserves nonnil opaque acknowledgement plus error; close once. Native rejected response ownership is retained.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`.
- Shared unchanged REST/fixedrequest preserves configured native prebody retry/reauth/backoff and identical-target redirect policy, blocks changed method/origin/path/query and owned KeepResponseBody/JSONResponse/RawBody/serialized JSON. Original accepted status enforced even if native hook expands OkCodes; unchanged serialized replacement may remain, nilbody differsJSONnull. Native reauth ErrOriginal/ErrReauth fields remain native.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`.
- Reusable lazy List/All prepares per iteration, only six typed query keys. Limit nilomit/0wire/negativeinvalid; emptyMarker omitted, literalCSV resource_types unchanged, enums validated, server defaults/clamp delegated. MaxItems is local, SinglePage skipsnext only; no limit hint, arbitrary query, Python local Body filter or invented marker.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCompletePreflight`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAdvertisedPaginationAndCaps`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsParallelReusableHelpers`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesAdvertisedPagingAndControls`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Required nonnull namespaces array; rows decoded/check source only when consumed. Cap/break skip unused malformed rows and next; empty success stops; All nonnil empty and discards partials on errors. Only canonical advertised body next is followed after consumption. Exact /v2/metadefs/namespaces rootrelative maps to captured reverse-prefix collection while original Body stays unchanged. Sameorigin/path and initial key/value query retained, singleton bounded marker advances; changed target/query/escaped alias/dot/authority or repeated URL/marker rejected before HTTP. No first/schema/self/links/header/fallback.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAdvertisedPaginationAndCaps`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesAdvertisedPagingAndControls`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`, `connectiontest/connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Owned namespace projection includes12 Body descriptors and location (13 fields): inherited id/name; canonical is_protected aliases protected in original member order with JSON truthiness/null; tags missing[]/null/scalar list; associations list-of-dict converts nonobject elements to{}. Present id includingnull authoritative else namespace. Declared Body recomputes current location; no declared Body retains supplied raw location.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsDeclaredProjectionAndOwnedChannels`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsDescriptorNullDefaultAndAliasOrder`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCurrentLocationAndStickyGuards`.
- Public namespace **query maps six server keys plus canonical declared Body local filters; visibility query takes precedence. Remote protected, unknown/tag query attrs are ignored; reserved heterogeneous/target/session/version controls rejected explicitly. Options own snapshots, full replacement/last wins, nil/errors/header/query conflicts fail before HTTP.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsSourceQueriesAndNormalizedLocalFilters`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCapsPartialsAndIteratorOwnership`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCompletePreflightAndFaultReceipts`.
- Namespace Limit pointer distinguishes nil omission and explicit numericzero wire0. Zero disables marker fallback but follows explicit continuations; positive MaxItems hint replaces falseyzero or absentlimit. Positive requested limits persist; consumed raw-row cap precedes filters. Shared paging zero permission is opt-in and other services retain positive-only validation.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsExplicitZeroLimitAndBindingCollisions`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCapsPartialsAndIteratorOwnership`, `internal/rest/list_zero_limit_test.go:TestListZeroLimitRequiresServiceOptIn`, `internal/rest/list_zero_limit_test.go:TestListZeroLimitRetainsExplicitLinksAndFalseyCapHint`.
- Actual200..399 requires whole UTF8 JSON, nonnull namespaces object envelope, array or single object; consumed rows nonnull objects. Constructor connection/microversion/_synchronized presence fails with original receipt/partial rows; self/other unknown fields remain passive actual Wire.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsActualStatusesAndMandatoryRepresentation`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsExplicitZeroLimitAndBindingCollisions`.
- Generic Body/HTTP Link continuation, short-page fallback using actual consumed id/namespace, empty stop, filters and cursor inheritance, cycle/drift/fixed path safety and exact relative /v2 collection aliases preserve pagination while AllRecords keeps partial rows.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsGenericPaginationAndRawMarkers`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCapsPartialsAndIteratorOwnership`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsExplicitZeroLimitAndBindingCollisions`, `internal/rest/list_versioned_path_test.go:TestListVersionedPathAliasRequiresExactRelativeCollection`.
- Accepted read/Close/UTF8/JSON/context/source errors preserve actual response and causes; source/outer invariants are sticky through preparation and body handling. Shared read boundary check precedes Close restoration, body closes once and accepted handling errors are never replayed.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCurrentLocationAndStickyGuards`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCompletePreflightAndFaultReceipts`, `internal/rest/guarded_response_test.go:TestDoJSONGuardedObservesReadFailureBeforeCloseRestoresSource`.
- Connection Image/ImageV2 share the original provider/client and current location; actual non200 pages, descriptor projection, remote Wire values, snapshot headers and live token survive two pages. Older typed strict200 API remains separate and unchanged.
  Tests: `connectiontest/connection_image_metadef_record_lists_test.go:TestConnectionImageMetadefRecordListsShareLocationAndPaging`, `connectiontest/connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.

Python/Go differences:

- This named mapping uses owned ListRecords/AllRecords; legacy typed CRUD/List/All keeps its original strict statuses and bounded projection. It does not assert all inherited mutable Resource/dirty/cache/session surfaces are implemented.
- Go fixes session/target/version/auth ownership, validates UTF8 and safe literal routes, retains independent receipts and raw Wire, and guards origin/path/query/cycles/conflicting links. Exact relative versioned collection links are allowed; arbitrary Python target or query replacements are excluded.
- Selected semantic query values use Go scalar/scalar-array encoding; object/nested-array query forms are rejected. Local comparison keeps bool and number distinct with exact decimal equality, unlike some Python equality/float/exception behavior.
- Dictionary links:{next:URL} is an explicit Go extension; Source dictionary iteration expects rel/href member objects. Namespace integer limit model does not preserve the dynamic Source difference between numeric0 and quoted0. Deprecated JMESPath applies to the Source generator before collection and may return heterogeneous non-list values; typed Go listing rejects that branch.
- Default server policy places these reader operations in core user; deployment visibility/ownership/policy and real cloud state are handled by the server and not proven by local fixtures.

### `python:image/v2/metadef_objects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.NamespaceScope.List`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.NamespaceScope.All`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.NamespaceScope.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.NamespaceScope.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.NewWithDependencies`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListFilters`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListVisibility`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListResourceTypes`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListSortKey`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithRecordListSortDir`

- Zero-HTTP InNamespace validates required literal parent and captures original API/client/provider/type/endpoint/base/microversion for scope lifetime. Parent/child names are nonempty UTF8<=80runes, route-breaking/control/dot syntax rejected, colon/Unicode/space preserved and escaped once. No Ref/find/parent GET or response-driven routing. Each request copies latest ordinary source headers and uses live original auth; scope/source/context rechecked before callbacks, after accepted body and consumed rows.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreCompletePreflight`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsValidationAndDefaults`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`.
- Canonical Object requires whole validUTF8 nonnullJSONobject. Exact-case optional nullable name/description/self/schema/literal dates remain pointers; no ID/namespace/name/links seed or time parsing. Properties root optionalobject with passive anyJSON inner RawMessages; required optionalstrictstringarray without null elements. Missing/null nil, empty map/array nonnil; atomic type errors retain actual response proof, raw unknown values and numeric precision independently owned.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsCanonicalModelsAndRawOwnership`.
- Six concrete option families: FullOpts wholly replaces configuration and last option wins; deep snapshots preserve nil/empty pointer/maps/RawMessage/slices. Per-callback config copied before/after and callbacks once per request/iteration; nil/error callback and owned/conflicting/invalid headers fail before HTTP. Helper reuse and lazy iterator option-slice ownership preserved.
  Tests: `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsCallbackDeepOwnership`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsValidationAndDefaults`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsParallelReusableHelpers`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`.
- Strict original statuses Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/UTF8/JSON/model/context/source failures preserve actual Body/Header/status and original causes in ResponseError with close once/no replay. Actual204 opaque acknowledgement survives handling failures with requested Namespace/childName, nil childName for DeleteAll; native rejected response ownership remains native.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsResponseOwnershipAndNativeHooks`.
- Unchanged shared REST/fixedrequest retains configured prebody retry/reauth/backoff/identical-target redirect policy; guards method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership. Actual statuses cannot be expanded by hook OkCodes. Original read/Close/ctx.Err/customCause plus native reauth ErrOriginal/ErrReauth evidence retained; no accepted-body replay.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsResponseOwnershipAndNativeHooks`.
- Lazy finite one bodyless/queryless GET200 validates whole root and requirednonnullobjectsarray then only consumed rows. Local MaxItems0unlimited/negativeinvalid; cap/callerbreak skip unused bad rows. No query limits/markers/sort/filter or next/schema/self/headerLink continuation/hints/fallback despite copied namespace pagination prose in API reference. All success empty nonnil, error nil without partial rows; iterator yields already-consumed rows.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFiniteConsumptionAndOwnership`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsFiniteListAndLocalControls`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsCallbackDeepOwnership`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsParallelReusableHelpers`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`, `internal/cmd/sdkgen/glance_metadef_objects_test.go:TestGlanceMetadefObjectsRegistryAndConcreteScope`.
- Owned namespace-only default listing follows pinned Proxy.metadef_objects: bodyless fixed GET, seven untyped Body fields plus fixed namespace/current location view9. Present id including null wins; absent id falls back name; unknown/self/date/raw values remain passive in independent Wire/Envelope. Three Source constructor duplicate keys connection/microversion/_synchronized reject with actual page receipt and prior rows.
  Tests: `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsFixedRoutesProjectionAndReceipts`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsIdentityPresenceAndLocation`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsConstructorBindingFailuresKeepPartialReceipt`.
- Owned pages accept actual200..399 with whole UTF8 mandatory objects JSON and singleton object envelope. Generic links/next/HTTPLink, last consumed raw marker, short-page fallback only for positive initial limit/hint, empty stop and exact relative /v2 collection alias behind reverse prefix are supported. Go fixes origin/path/query and rejects cycles/conflicting or changed targets before the next request.
  Tests: `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsStatusEnvelopeAndEmptyPolicy`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsContinuationRepresentations`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsRejectedContinuationKeepsPartialReceipt`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsMarkerFallbackUsesWireIdentity`, `internal/rest/list_versioned_path_test.go:TestListVersionedPathAliasRequiresExactRelativeCollection`.
- Concrete option factories snapshot maps/pointers/slices/selected JSON; callback configuration copied before/after, nil/error/options/reserved headers fail pre-HTTP, each lazy iteration resnapshots its source. Query/Body filters and controls are explicit Go extensions to the public Python namespace-only signature; semantic selected filters and query ownership are retained.
  Tests: `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsLazyRepeatAndOptionSnapshots`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsSemanticFilters`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsCompletePreflight`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsFullOptionsAndRetainedCallbacksOwnStorage`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsQueryExtensionsAndFixedContinuation`.
- Raw consumed-row cap precedes local filters; early break/cap skips unconsumed rows/next; AllRecords preserves successful partial rows. Accepted read/Close/UTF8/JSON/row/cancel/context failures retain actual Body/Header/status and original causes without accepted replay.
  Tests: `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsRawCapAndEarlyStop`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsAcceptedReadCloseAndCancelCauses`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsRejectedContinuationKeepsPartialReceipt`.
- SDK source, namespace and outer guard snapshots remain sticky through options/native retries/physical attempts/read/Close/yield. Read-time source failures are observed before Close can restore state; native target/body/status ownership and live provider token behavior remain bounded.
  Tests: `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsPreflightOrderAndLocationFailures`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsNativeRetryAndOwnership`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsSourceAndOuterGuardAcrossPhysicalBoundaries`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsObservedGuardFailureSurvivesRestoration`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsNamespaceGuardAndRowProvenance`, `internal/rest/guarded_response_test.go:TestDoJSONGuardedObservesReadFailureBeforeCloseRestoresSource`.
- Connection Image and ImageV2 use the same shared native client/provider and owned location dependencies; location captured before callbacks, nullable identity and fixed URI view survive actual multi-page receipts. Existing typed finite strict200 List/All remains separate.
  Tests: `connectiontest/connection_image_metadef_record_lists_test.go:TestConnectionImageMetadefRecordListsShareLocationAndPaging`, `image/v2/metadefobjects/list_records_test.go:TestMetadefObjectRecordsLegacyFiniteListRemainsSeparate`.

Python/Go differences:

- This named mapping uses owned ListRecords/AllRecords; legacy typed CRUD/List/All keeps its original strict statuses and bounded projection. It does not assert all inherited mutable Resource/dirty/cache/session surfaces are implemented.
- Go fixes session/target/version/auth ownership, validates UTF8 and safe literal routes, retains independent receipts and raw Wire, and guards origin/path/query/cycles/conflicting links. Exact relative versioned collection links are allowed; arbitrary Python target or query replacements are excluded.
- Selected semantic query values use Go scalar/scalar-array encoding; object/nested-array query forms are rejected. Local comparison keeps bool and number distinct with exact decimal equality, unlike some Python equality/float/exception behavior.
- Dictionary links:{next:URL} is an explicit Go extension; Source dictionary iteration expects rel/href member objects. Namespace integer limit model does not preserve the dynamic Source difference between numeric0 and quoted0. Deprecated JMESPath applies to the Source generator before collection and may return heterogeneous non-list values; typed Go listing rejects that branch.
- Default server policy places these reader operations in core user; deployment visibility/ownership/policy and real cloud state are handled by the server and not proven by local fixtures.
- Pinned public metadef_objects(namespace) accepts no query kwargs: all caller query/filter/header/cap/pagination options are explicit Go extensions; default no-option calls retain the public Source responsibility.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"

for value in conn.image.metadef_namespaces(
    visibility="private", owner="project-id", is_protected=False,
    limit=10, max_items=20, paginated=True
):
    print(value.to_dict())

for value in conn.image.metadef_objects(namespace):
    print(value.to_dict())
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

    "github.com/JSYoo5B/go-openstacksdk"
    namespaces "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
    objects "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal object parent")
    namespaceFilter := flag.String("namespace-filter", "", "optional namespace Body filter")
    owner := flag.String("owner", "", "optional namespace owner Body filter")
    objectName := flag.String("object-name", "", "optional object name Body filter")
    visibility := flag.String("visibility", "", "optional namespace server query")
    limit := flag.Int("limit", 0, "explicit server page limit")
    maxItems := flag.Int("max-items", 0, "maximum consumed rows per list")
    paginated := flag.Bool("paginated", true, "follow list continuations")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    err := run(ctx, *cloud, *namespace, *namespaceFilter, *owner, *objectName,
        *visibility, *limit, *maxItems, *paginated)
    if err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            if printErr := printJSON(map[string]any{
                "error_status": response.StatusCode, "error_header": response.Header,
                "error_body": response.Body,
            }); printErr != nil { log.Print(printErr) }
        }
        log.Fatal(err)
    }
}

func printJSON(value any) error {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, namespaceFilter, owner,
    objectName, visibility string, limit, maxItems int, paginated bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.ImageV2(ctx)
    if err != nil { return err }
    namespaceOptions := []namespaces.RecordListOption{
        namespaces.WithRecordListOpts(namespaces.RecordListOpts{MaxItems: maxItems}),
        namespaces.WithRecordListHeader("X-Request-Source", "metadef-record-lists-example"),
        namespaces.WithRecordListPaginated(paginated),
    }
    if limit > 0 {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListLimit(limit))
    }
    if visibility != "" {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListVisibility(visibility))
    }
    if namespaceFilter != "" {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListFilter("namespace", namespaceFilter))
    }
    if owner != "" {
        namespaceOptions = append(namespaceOptions, namespaces.WithRecordListFilter("owner", owner))
    }
    for record, readErr := range service.MetadefNamespaces.ListRecords(ctx, namespaceOptions...) {
        if readErr != nil { return readErr }
        if err := printJSON(record); err != nil { return err }
    }
    scope, err := service.MetadefObjects.InNamespace(ctx, namespace)
    if err != nil { return err }
    objectOptions := []objects.RecordListOption{
        objects.WithRecordListOpts(objects.RecordListOpts{Limit: limit, MaxItems: maxItems}),
        objects.WithRecordListHeader("X-Request-Source", "metadef-record-lists-example"),
        objects.WithRecordListPaginated(paginated),
    }
    if objectName != "" {
        objectOptions = append(objectOptions, objects.WithRecordListFilter("name", objectName))
    }
    records, readErr := scope.AllRecords(ctx, objectOptions...)
    for _, record := range records {
        if err := printJSON(record); err != nil { return errors.Join(readErr, err) }
    }
    return readErr
}
```

## References

- [Metadefobjects](v2/metadefobjects/README.md)
- [namespace](v2/metadefnamespaces/README.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1449-L1613)
- [Metadef Object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_object.py)
- [namespace class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py)
- [inherited list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2436)
- [jmespath.py1.0.1 interpreter](https://github.com/jmespath/jmespath.py/blob/1.0.1/jmespath/visitor.py)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
