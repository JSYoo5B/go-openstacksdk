# Metadefobjects

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/create_metadef_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.NamespaceScope.Create`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithCreateDescription`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithCreateProperties`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithCreateRequired`

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
- One flat collection POST201 includes required name and only supplied description/properties/required. Description validUTF8 with no length bound; multiline/empty preserved. Properties nil omit/empty{} explicit, keys validUTF8 and values validnonnullUTF8JSONobject without fetched schema/type/title gate. Required nil omit/empty[] explicit, literal UTF8 entries including empty/comma/newline/repeats without trim/dedupe/membership/CSV normalization. No discovery/GET/rollback; server validates property schemas and owns RequiredCSV storage.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreCompletePreflight`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsValidationAndDefaults`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsFixedScopedRoutesAndPayloads`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`, `internal/cmd/sdkgen/glance_metadef_objects_test.go:TestGlanceMetadefObjectsRegistryAndConcreteScope`.
- Create posts metadefs/namespaces/{namespace}/objects with the name and only supplied optional fields, matching Python _create(MetadefObject, namespace_name=..., **attrs).
  Tests: `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsFixedScopedRoutesAndPayloads`.

Python/Go differences:

- Python mutable Resource/descriptor/coercion, arbitrary attrs, adapter/session/version controls and cached return semantics are broader than this typed scoped API.
- Go explicitly bounds literal route input, validates canonical types/UTF8 and original201/200/204 statuses; generic Python Resource response handling differs.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.
- Go validates literal route names and accepts only the documented 201/200/204 statuses; Python accepts any status below 400.

### `python:image/v2/get_metadef_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.NamespaceScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithGetHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithGetHeaders`

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
- One fixed child GET200 has no body/query/fallback. Selected literal parent and child remain route identity even if canonical response names/self/schema differ; optional canonical presence/raw data is passive. Python object_arg is reduced to _get_id before generic _get constructs a resource; mutable descriptor/coercion/session behavior remains broader.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreCompletePreflight`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsFixedScopedRoutesAndPayloads`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsCanonicalModelsAndRawOwnership`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`, `internal/cmd/sdkgen/glance_metadef_objects_test.go:TestGlanceMetadefObjectsRegistryAndConcreteScope`.

Python/Go differences:

- Python mutable Resource/descriptor/coercion, arbitrary attrs, adapter/session/version controls and cached return semantics are broader than this typed scoped API.
- Go explicitly bounds literal route input, validates canonical types/UTF8 and original201/200/204 statuses; generic Python Resource response handling differs.
- Server schema/policy/protection/DB defaults/RequiredCSV and real cloud side effects are not reproduced or proven by local fixtures; no full parity or authenticated cloud claim.
- The pinned get_metadef_object extracts object and namespace Resource IDs and constructs a fresh object with name alternate ID and namespace_name URI seed. Go owns an explicit zero-HTTP InNamespace scope plus literal child Get; NamespaceName() exposes the requested parent and Object.Name reflects only the actual nullable response rather than a synthesized request-name seed. Returned name/self/schema never drive routes.
- Python properties and required descriptors have no declared type and may return arbitrary JSON values. Go provides Properties as an optional object of RawMessages and Required as an optional nonnull string array, with nil/empty distinctions and no coercion; the complete actual document remains independently accessible in Metadata.Body. Names/dates are literal nullable strings, raw id/parent/unknown fields are preserved rather than promoted, and Self is retained whereas Python Resource removes self.
- Concrete header/FullOpts options own snapshots and callbacks, with context/error and captured scope source replacing arbitrary Resource/session/adapter input. The default getter has no body/query, no resolver/parent GET, no IgnoreMissing and no list fallback. Scope/client microversion stays selected; the pinned class inherits maximum microversion None.
- Go enforces safe nonempty UTF-8 namespace/child names up to80runes and actual HTTP200 plus strict nonnull canonical JSON. Python dynamic Resource permits broader input/status/coercion behavior and can retain seeds after JSON ValueError. Existing local tests preserve native rejection and accepted body/read/Close/context/source evidence, without claiming authenticated deployment or other object CRUD/list/bulk APIs.

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

### `python:image/v2/update_metadef_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.NamespaceScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithUpdateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithUpdateName`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithUpdateDescription`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithUpdateProperties`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithUpdateRequired`

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
- One replacement PUT200 uses fixed current child route and mandatory body name=current unless optional rename. Omitted optional description/required/properties reset server values; nil omit versus empty{} / [] explicit. No GET/RMW/cache merge/PATCH/noop/rollback. Pinned Python object_arg reduces via _get_id before generic _update; dirty-only commit may skip empty body/header and does not restore mandatory object name. RequiredCSV/server-schema behavior remains a difference.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreCompletePreflight`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsValidationAndDefaults`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsFixedScopedRoutesAndPayloads`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`, `internal/cmd/sdkgen/glance_metadef_objects_test.go:TestGlanceMetadefObjectsRegistryAndConcreteScope`.
- Update sends PUT metadefs/namespaces/{namespace}/objects/{name} with name and only supplied optional fields, the body a successful Python call with name=... sends.
  Tests: `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsFixedScopedRoutesAndPayloads`.

Python/Go differences:

- Python mutable Resource/descriptor/coercion, arbitrary attrs, adapter/session/version controls and cached return semantics are broader than this typed scoped API.
- Go explicitly bounds literal route input, validates canonical types/UTF8 and original201/200/204 statuses; generic Python Resource response handling differs.
- Python proxy reduces object_arg to _get_id; generic dirty-only commit may no-op and omits dirty identity, whereas Go supplies current name and always sends replacement PUT.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.
- Go validates literal route names and accepts only the documented 201/200/204 statuses; Python accepts any status below 400.
- Python's commit leaves name out of the PUT body unless the caller passes name=..., and Glance rejects an object update without name; Go always sends the current name (WithUpdateName renames).

### `python:image/v2/delete_metadef_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.NamespaceScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithDeleteHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithDeleteIgnoreMissing`

- Zero-HTTP InNamespace validates required literal parent and captures original API/client/provider/type/endpoint/base/microversion for scope lifetime. Parent/child names are nonempty UTF8<=80runes, route-breaking/control/dot syntax rejected, colon/Unicode/space preserved and escaped once. No Ref/find/parent GET or response-driven routing. Each request copies latest ordinary source headers and uses live original auth; scope/source/context rechecked before callbacks, after accepted body and consumed rows.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreCompletePreflight`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsValidationAndDefaults`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`.
- Six concrete option families: FullOpts wholly replaces configuration and last option wins; deep snapshots preserve nil/empty pointer/maps/RawMessage/slices. Per-callback config copied before/after and callbacks once per request/iteration; nil/error callback and owned/conflicting/invalid headers fail before HTTP. Helper reuse and lazy iterator option-slice ownership preserved.
  Tests: `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsCallbackDeepOwnership`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsValidationAndDefaults`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsParallelReusableHelpers`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`.
- Strict original statuses Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/UTF8/JSON/model/context/source failures preserve actual Body/Header/status and original causes in ResponseError with close once/no replay. Actual204 opaque acknowledgement survives handling failures with requested Namespace/childName, nil childName for DeleteAll; native rejected response ownership remains native.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsResponseOwnershipAndNativeHooks`.
- Unchanged shared REST/fixedrequest retains configured prebody retry/reauth/backoff/identical-target redirect policy; guards method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership. Actual statuses cannot be expanded by hook OkCodes. Original read/Close/ctx.Err/customCause plus native reauth ErrOriginal/ErrReauth evidence retained; no accepted-body replay.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsResponseOwnershipAndNativeHooks`.
- One child DELETE204 returns opaque acknowledgement of fixed requested Namespace/Name. Nil IgnoreMissing defaults true; only physically owned clean404 with successful read/Close/context/source returns nil,nil and privately bypasses native404RetryFunc. Explicitfalse preserves native404retry/error; nested transport/callback/reauth/logical404 never quiet. Hidden parent/child404 is not absence proof; no unprotect/enumeration/cleanup. Python delete returns Resource or None rather than actual204ACK.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreCompletePreflight`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsValidationAndDefaults`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsResponseOwnershipAndNativeHooks`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsFixedScopedRoutesAndPayloads`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`, `internal/cmd/sdkgen/glance_metadef_objects_test.go:TestGlanceMetadefObjectsRegistryAndConcreteScope`.
- Delete sends DELETE metadefs/namespaces/{namespace}/objects/{name} and by default treats 404 as success like Python ignore_missing=True.
  Tests: `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsFixedScopedRoutesAndPayloads`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreAcceptedBodiesAndMissingPolicy`.

Python/Go differences:

- Python mutable Resource/descriptor/coercion, arbitrary attrs, adapter/session/version controls and cached return semantics are broader than this typed scoped API.
- Go explicitly bounds literal route input, validates canonical types/UTF8 and original201/200/204 statuses; generic Python Resource response handling differs.
- Python returns Resource/namespace Resource or None; Go returns actual opaque204ACK and uses cause-safe physical404 policy only for individual Delete.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.
- Go validates literal route names and accepts only the documented 201/200/204 statuses; Python accepts any status below 400.
- Go returns the actual 204 acknowledgement where Python returns the Resource or None.

### `python:image/v2/delete_all_metadef_objects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.NamespaceScope.DeleteAll`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithDeleteAllOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithDeleteAllHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects.WithDeleteAllHeaders`

- Zero-HTTP InNamespace validates required literal parent and captures original API/client/provider/type/endpoint/base/microversion for scope lifetime. Parent/child names are nonempty UTF8<=80runes, route-breaking/control/dot syntax rejected, colon/Unicode/space preserved and escaped once. No Ref/find/parent GET or response-driven routing. Each request copies latest ordinary source headers and uses live original auth; scope/source/context rechecked before callbacks, after accepted body and consumed rows.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreCompletePreflight`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsValidationAndDefaults`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`.
- Six concrete option families: FullOpts wholly replaces configuration and last option wins; deep snapshots preserve nil/empty pointer/maps/RawMessage/slices. Per-callback config copied before/after and callbacks once per request/iteration; nil/error callback and owned/conflicting/invalid headers fail before HTTP. Helper reuse and lazy iterator option-slice ownership preserved.
  Tests: `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsCallbackDeepOwnership`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsValidationAndDefaults`, `image/v2/metadefobjects/options_test.go:TestMetadefObjectsOptionsParallelReusableHelpers`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`.
- Strict original statuses Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/UTF8/JSON/model/context/source failures preserve actual Body/Header/status and original causes in ResponseError with close once/no replay. Actual204 opaque acknowledgement survives handling failures with requested Namespace/childName, nil childName for DeleteAll; native rejected response ownership remains native.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsResponseOwnershipAndNativeHooks`.
- Unchanged shared REST/fixedrequest retains configured prebody retry/reauth/backoff/identical-target redirect policy; guards method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership. Actual statuses cannot be expanded by hook OkCodes. Original read/Close/ctx.Err/customCause plus native reauth ErrOriginal/ErrReauth evidence retained; no accepted-body replay.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsResponseOwnershipAndNativeHooks`.
- One collection DELETE204 returns opaque acknowledgement with selected Namespace and nilName; every404strict, no IgnoreMissing helper. No enumerate/perchilddelete/count/wait/unprotect/rollback. Accepted handling failure retains ACK plus error. Python uses MetadefNamespace.delete_all_objects and returns namespace Resource, with policy/protected behavior delegated to server.
  Tests: `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreFixedScopeAndCRUD`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefobjects/core_test.go:TestMetadefObjectsCoreCompletePreflight`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsResponseOwnershipAndNativeHooks`, `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsFixedScopedRoutesAndPayloads`, `connectiontest/connection_image_metadef_objects_test.go:TestConnectionImageMetadefObjectsShareClientAndFixedScope`, `internal/cmd/sdkgen/glance_metadef_objects_test.go:TestGlanceMetadefObjectsRegistryAndConcreteScope`.
- DeleteAll sends one DELETE metadefs/namespaces/{namespace}/objects, the request Python delete_all_objects sends.
  Tests: `image/v2/metadefobjects/contracts_test.go:TestMetadefObjectsFixedScopedRoutesAndPayloads`.

Python/Go differences:

- Python mutable Resource/descriptor/coercion, arbitrary attrs, adapter/session/version controls and cached return semantics are broader than this typed scoped API.
- Go explicitly bounds literal route input, validates canonical types/UTF8 and original201/200/204 statuses; generic Python Resource response handling differs.
- Python returns Resource/namespace Resource or None; Go returns actual opaque204ACK and uses cause-safe physical404 policy only for individual Delete.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.
- Go validates literal route names and accepts only the documented 201/200/204 statuses; Python accepts any status below 400.
- Go returns the actual 204 acknowledgement where Python returns the namespace Resource.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects"
)

type ObjectEvidence struct {
    Created *metadefobjects.Object
    Read *metadefobjects.Object
    Rows []*metadefobjects.Object
    All []*metadefobjects.Object
    Updated *metadefobjects.Object
    Deleted *metadefobjects.Acknowledgement
    Cleared *metadefobjects.Acknowledgement
}

func manageObjects(ctx context.Context, service *image.Service, namespace, name, replacementName string, clearNamespace bool) (out ObjectEvidence, err error) {
    scope, err := service.API.MetadefObjects.InNamespace(ctx, namespace)
    if err != nil { return out, err }
    description := "First line\nSecond line"
    properties := map[string]json.RawMessage{
        "vcpus": json.RawMessage(`{"type":"integer","title":"Virtual CPUs","minimum":1}`),
    }
    required := []string{"vcpus"}
    out.Created, err = scope.Create(ctx, name,
        metadefobjects.WithCreateOpts(metadefobjects.CreateOpts{Description: &description}),
        metadefobjects.WithCreateHeader("X-Request-Source", "example"),
        metadefobjects.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
        metadefobjects.WithCreateDescription(description),
        metadefobjects.WithCreateProperties(properties),
        metadefobjects.WithCreateRequired(required))
    if err != nil { return out, err }

    out.Read, err = scope.Get(ctx, name,
        metadefobjects.WithGetOpts(metadefobjects.GetOpts{}),
        metadefobjects.WithGetHeader("X-Request-Source", "example"),
        metadefobjects.WithGetHeaders(map[string]string{"X-Trace": "get"}))
    if err != nil { return out, err }

    listOptions := []metadefobjects.ListOption{
        metadefobjects.WithListOpts(metadefobjects.ListOpts{}),
        metadefobjects.WithListHeader("X-Request-Source", "example"),
        metadefobjects.WithListHeaders(map[string]string{"X-Trace": "list"}),
        metadefobjects.WithListMaxItems(20),
    }
    for row, listErr := range scope.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = scope.All(ctx, listOptions...)
    if err != nil { return out, err }

    out.Updated, err = scope.Update(ctx, name,
        metadefobjects.WithUpdateOpts(metadefobjects.UpdateOpts{}),
        metadefobjects.WithUpdateHeader("X-Request-Source", "example"),
        metadefobjects.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        metadefobjects.WithUpdateName(replacementName),
        metadefobjects.WithUpdateDescription(description),
        metadefobjects.WithUpdateProperties(properties),
        metadefobjects.WithUpdateRequired(required))
    if err != nil { return out, err }

    out.Deleted, err = scope.Delete(ctx, replacementName,
        metadefobjects.WithDeleteOpts(metadefobjects.DeleteOpts{}),
        metadefobjects.WithDeleteHeader("X-Request-Source", "example"),
        metadefobjects.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
        metadefobjects.WithDeleteIgnoreMissing(false))
    if err != nil || !clearNamespace { return out, err }
    out.Cleared, err = scope.DeleteAll(ctx,
        metadefobjects.WithDeleteAllOpts(metadefobjects.DeleteAllOpts{}),
        metadefobjects.WithDeleteAllHeader("X-Request-Source", "example"),
        metadefobjects.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
    return out, err
}
```

```python
def manage_objects(conn, namespace, name, replacement_name, clear_namespace=False):
    properties = {"vcpus": {"type": "integer", "title": "Virtual CPUs", "minimum": 1}}
    created = conn.image.create_metadef_object(
        namespace, name=name, description="First line\nSecond line",
        properties=properties, required=["vcpus"])
    fetched = conn.image.get_metadef_object(name, namespace)
    rows = list(conn.image.metadef_objects(namespace))
    updated = conn.image.update_metadef_object(
        name, namespace, name=replacement_name,
        description="First line\nSecond line", properties=properties, required=["vcpus"])
    deleted = conn.image.delete_metadef_object(
        replacement_name, namespace, ignore_missing=False)
    cleared = conn.image.delete_all_metadef_objects(namespace) if clear_namespace else None
    return created, fetched, rows, updated, deleted, cleared
```

## References

- [Metadef Object Namespace Record Lists](../../metadef-object-namespace-record-lists.md)
- [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1543-L1692)
- [  Init  ](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/__init__.py#L561-L601)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_objects.py#L174-L221)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)
- [repository.list](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/__init__.py#L625-L632)
- [serializer.index](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_objects.py#L430-L435)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-objects)
- [Contracts Test](contracts_test.go)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Connection Image Metadef Objects Test](../../../connectiontest/connection_image_metadef_objects_test.go)
- [Glance Metadef Objects Test](../../../internal/cmd/sdkgen/glance_metadef_objects_test.go)
