# Metadef Resource Types Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/metadef_resource_types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.API.All`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.NewWithDependencies`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListFilters`

- Global List/All capture source per lazy iteration before option callbacks, not API lifetime. Original client/provider/type/endpoint/ServiceURL/microversion stays fixed during iteration, latest valid ordinary headers and live original auth retained; source/context checks before callbacks/request/accepted body/each consumedrow. No namespace discovery, global create/delete/fetch, or identity retargeting.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreSourceLifetimeAndNativePolicies`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreCompletePreflight`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesFiniteIterationAndLocalCaps`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_resource_types_test.go:TestConnectionImageMetadefResourceTypesShareClientAndScopes`.
- ResourceType canonical exact optional nullable Name/CreatedAt/UpdatedAt are literal string pointers; strict whole UTF8 nonnullobject/canonical nonnullstring types atomic. Unknown raw JSON including protected, big numbers and null stays independently owned in Body; no typed Protected/ID/parent/Self/Links/date parsing or request-seeded identity.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesCanonicalModelsAndRawOwnership`.
- List concrete options use FullOpts whole replacement/lastwins and independent helper/header/map/pointer/option-slice/config snapshots before/after callbacks. Callback once per request or lazy iteration; nil/error callback/owned invalid headers fails preHTTP. Shared ListOption is reusable across the two real list operations, without assigning Create/Delete helper contracts to a list review.
  Tests: `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsCallbacksAndRequestOwnership`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsValidationAndDefaults`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesFiniteIterationAndLocalCaps`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesOwnedPreparationAndSource`.
- One lazy bodyless/queryless GET200, exact requirednonnull named array and wholeUTF8JSON checked before yielding, only consumed strict rows decoded. Preserve wire order/repeats; MaxItems0unlimited/negativeinvalid, cap/break skips unused malformed rows and extra HTTP. All nonnil empty on success/nil no partial on error. No query/limit/marker/sort/clientfilter/link-follow/fallback/cache even when response includes passive next metadata.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFiniteConsumptionAndRootEvidence`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesFiniteIterationAndLocalCaps`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFixedRoutesAndPassiveModels`, `internal/cmd/sdkgen/glance_metadef_resource_types_test.go:TestGlanceMetadefResourceTypesRegistryAndConcreteScope`.
- Original200 list only decoded, whole actualBody/Header/status and accepted read/Close/JSON/UTF8/canonical/contextErr/customCause/source errors preserved via ResponseError, onceClose/no replay; row/header/raw proof independently owned.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFiniteConsumptionAndRootEvidence`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesResponseOwnershipAndNativeHooks`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesCanonicalModelsAndRawOwnership`.
- Configured native prebody retry/reauth/backoff and same-target redirects preserved; exact method/origin/path/query/serialized body/owned response fields and originalstatus gates. Native MoreHeaders mutation remains advanced provider policy; no ordinary-header hook rejection promise. Preserve read/Close/transport/ctx.Err/customCause and native ErrOriginal/ErrReauth fields without inventing Unwrap.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreSourceLifetimeAndNativePolicies`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesResponseOwnershipAndNativeHooks`.
- Declared untyped nullable fields, explicit id versus alternate name, collector-dependent computed location and fixed namespace URI are projected independently of complete Wire/page/headers/status. Connection Image and ImageV2 share client and captured current location; URI parent is an explicit additional Go view field, not default Python to_dict serialization.
  Tests: `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsFixedRoutesProjectionAndReceipts`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsIdentityPresenceAndLocation`, `connectiontest/connection_image_metadef_records_test.go:TestConnectionImageMetadefRecordsShareClientAndCurrentLocation`.
- Concrete options own callbacks/header/filter/pointer/slice snapshots and full replacement. Only limit/marker are server queries; declared Body filters are local, unknown captured encoding errors are ignored, path/session/version controls and typed/semantic collisions fail before HTTP. Raw MaxItems cap and wire limit hint precede filters and unconsumed rows.
  Tests: `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsLazyRepeatAndOptionSnapshots`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsSemanticFilters`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsCompletePreflight`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsPreflightOrderAndLocationFailures`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsRawCapAndEarlyStop`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsFullOptionsAndRetainedCallbacksOwnStorage`.
- Shared pager accepts body links/plural links/next/HTTP Link and singleton rows. Explicit-limit marker fallback follows short nonempty pages from retained wire identity, while cap/single-page/empty-page/early-break avoid unused continuation work. Same-origin/collection/filters, conflicting links and marker/cycle rules are explicit guarded Go choices; partial AllRecords survive later failure.
  Tests: `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsMarkerFallbackUsesWireIdentity`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsContinuationRepresentations`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsRejectedContinuationKeepsPartialReceipt`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsRawCapAndEarlyStop`.
- Actual200..399 requires whole-page UTF8 and mandatory plural JSON representation; empty204 and consumed malformed rows fail with actual page receipt. Accepted read/Close/cancellation causes and rejected native status evidence are preserved; no accepted-body replay or invented response.
  Tests: `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsStatusEnvelopeAndEmptyPolicy`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsAcceptedReadCloseAndCancelCauses`.
- Native retries retain the logical query/source/status policy. Sticky observed source and outer guards cover callbacks, physical roundtrip/read/Close/retry and post-yield boundaries even if a changed source is later restored; existing finite typed List/All stay strict200 and query-free.
  Tests: `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsNativeRetryAndOwnership`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsSourceAndOuterGuardAcrossPhysicalBoundaries`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsObservedGuardFailureSurvivesRestoration`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsLegacyFiniteListRemainsSeparate`.

Python/Go differences:

- Named list declarations map to owned records, semantic query/body filtering and paging; Python mutable Resource lifecycle/cache/adapter overrides remain separate common SDK goals. Go uses fixed literal namespace and a selected service version, guarded headers and a copied declared view with namespace_name URI provenance beyond default Python to_dict.
- Go deliberately requires whole-page UTF8/object/plural JSON, one positive query limit and a nonempty string marker; local JSON filtering distinguishes bool from number and compares exact decimal numbers. Unknown row fields remain complete only in Wire, separately from declared Resource.
- Shared Go paging restricts origin/collection/filter changes, conflicting continuation links and repeated markers, and accepts links.next dictionaries as an additional representation. Source prioritizes links and permits broader URL/query changes including /v-prefix handling. The actual pinned Glance controller lists finite rows and ignores limit/marker; callers can choose Paginated=false.
- No live authenticated cloud, Python execution, server policy/DB/collation/transaction effects or dynamic session/cache lifecycle equivalence is claimed. Existing typed strict200 finite APIs and create/delete contracts are preserved; record lists accept actual200..399 with mandatory representation.

### `python:image/v2/create_metadef_resource_type_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.NamespaceScope.Create`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithCreatePrefix`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithCreatePropertiesTarget`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.NamespaceScope.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordCreateAttributes`

- InNamespace is zeroHTTP; required safe literal namespace, original API/client/provider/type/endpoint/ServiceURL/microversion captured for scope lifetime without retained context or parentGET/lookup/Ref. Selected child is nonempty UTF8<=80runes with route-breaking/control/dot rejection and exact-once path escape, no trim/UUID/nameFind. Latest valid ordinary source headers copied per request and original provider auth remains live, guards before callbacks/request/accepted body/each consumedrow.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFixedRoutesAndPassiveModels`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreCompletePreflight`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreSourceLifetimeAndNativePolicies`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_resource_types_test.go:TestConnectionImageMetadefResourceTypesShareClientAndScopes`.
- Association adds exact optional nullable Prefix/PropertiesTarget to Name and literal date strings. Strict whole UTF8 nonnull object/wrong nonnull canonical types fail atomically, no partial mutation. Body/Header/pointers independently owned; unknown precision/raw protected/self/links/parent remain passive, no request identity seed or returned-name routing.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesCanonicalModelsAndRawOwnership`.
- Create concrete option family uses FullOpts whole replacement/lastwins and independent helper/header/map/pointer/option-slice/config snapshots before/after callbacks. Callback once/request; nil/error callback/owned invalid headers fails preHTTP, no runtime builder or auth/transport callback bridge.
  Tests: `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsCallbacksAndRequestOwnership`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsValidationAndDefaults`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsParallelReusableHelpers`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesOwnedPreparationAndSource`.
- One flat POST201 body always selected name; Prefix/PropertiesTarget pointers nilomitted/empty supplied with validUTF8<=80literal JSON strings including controls, no colon rule/trim. Server schema/WSME readonly/unknown/bounds and DB effects not emulated. No parent/global lookup/precreate/RMW/cache/update/wait/rollback; actual server may create absent global type as side effect. Passive result independently decoded, no guarantee of DB/cloud completion.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFixedRoutesAndPassiveModels`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreCompletePreflight`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesFixedRoutesAndPayloads`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesOwnedPreparationAndSource`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsValidationAndDefaults`, `internal/cmd/sdkgen/glance_metadef_resource_types_test.go:TestGlanceMetadefResourceTypesRegistryAndConcreteScope`.
- Original201 Create only decoded, actual wholeBody/Header/status and accepted read/Close/JSON/UTF8/canonical/contextErr/customCause/source errors preserved via ResponseError, onceClose/no replay; no synthetic input identity.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesResponseOwnershipAndNativeHooks`.
- Configured native prebody retry/reauth/backoff and same-target redirects preserved; exact method/origin/path/query/serialized body/owned response fields and originalstatus gates. Native MoreHeaders mutation remains advanced provider policy; no ordinary-header hook rejection promise. Preserve read/Close/transport/ctx.Err/customCause and native ErrOriginal/ErrReauth fields without inventing Unwrap.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreSourceLifetimeAndNativePolicies`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesResponseOwnershipAndNativeHooks`.
- Owned CreateRecord constructs a fresh Source-style dirty Body from exactly id/name/created_at/updated_at/prefix/properties_target, all untyped raw JSON including null/complex/large values. No SDK-required name/id; empty attributes POST {}. Only explicitly supplied six Body keys are transmitted, no missing default synthesis or child-path validation of passive create values, no parent/global lookup/precreate/cache/RMW/rollback or extra endpoint.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordCreateSixRawFieldsAndEmptyBody`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationCompletePreflight`, `connectiontest/connection_image_metadef_association_mutations_test.go:TestConnectionImageMetadefAssociationMutationsShareLocationAndIdentity`.
- New concrete RecordCreateOpts and named Header(s)/Attribute(s)/Opts capture independent maps/slices/raw JSON and callback configs, whole replacement plus individual last-wins and bulk semantic-set replacement. Only resource.WithFilter/WithFilters carriers accepted; unknown attributes and their captured encoding errors discarded before encoding. Known selected errors, nil/nonsemantic options, owned Accept/auth/header conflicts and 17 fixed route/client controls reject preHTTP; callback once and retained/caller changes cannot alter captured input.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordCreateFactoriesSelectionAndLocationSnapshot`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationCompletePreflight`.
- Owned created Resource eagerly materializes six nullable raw Body descriptors plus fixed namespace_name and current location, eight fields versus Source default to_dict seven excluding URI. Present id including null authoritative, absent id falls back to name. Present recognized response fields overlay explicit seed, missing values preserve seed/default null; self/unknown/response namespace/location remain passive only in actual Wire. Location captured before options, direct New null, supplied dependency cloned; both Connection Image and ImageV2 inject the same current location, Source fixed-URI collector ignores response location.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordCreateSixRawFieldsAndEmptyBody`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordCreateActualStatusesSeedOverlayAndOwnedChannels`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordCreateFactoriesSelectionAndLocationSnapshot`, `connectiontest/connection_image_metadef_association_mutations_test.go:TestConnectionImageMetadefAssociationMutationsShareLocationAndIdentity`.
- Owned CreateRecord accepts only actual 200..399; complete response UTF8 required. Empty/opaque/invalid JSON syntax succeeds with seed/default Resource and Wire nil, while valid parsed nonobject fails with actual ResponseError. Valid object yields independently owned declared Resource, actual Wire and whole Envelope/Header/status; no request seed or defaults injected into Wire and no returned field changes parent or route.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordCreateActualStatusesSeedOverlayAndOwnedChannels`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordCreateToleratedDecodeAndStrictWholeBody`, `connectiontest/connection_image_metadef_association_mutations_test.go:TestConnectionImageMetadefAssociationMutationsShareLocationAndIdentity`.
- Owned mutation guard checks source/scope/outer Connection/context before location/options and after every callback, physical/read/Close boundary; observed failure is sticky even if later restored. Create location failure blocks options/HTTP. Callback source/namespace changes and cancellation causes reject without request or preserve actual accepted receipt; no retained context or caller binding mutation.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationStickySourceAndOuterPreflight`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationAcceptedFaultReceiptsAndFrozenTargets`.
- Owned CreateRecord keeps original accepted status/body/header and joined read/Close/contextErr/customCause/source evidence on failure, onceClose and no accepted replay. Native configured prebody retry/reauth ownership remains bounded by serialized raw body, method/target and original 200..399 status policy; equivalent retry body allowed, changed body/expanded status rejected with native proof. Existing strict201 typed Create and raw CRUD behavior retained.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationAcceptedFaultReceiptsAndFrozenTargets`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationNativeRetryOwnershipAndLegacyPolicies`.

Python/Go differences:

- Owned CreateRecord provides fixed immutable declared Resource/Wire/receipt views and an eight-field Resource including namespace_name; Source returns a mutable association whose default to_dict has seven fields excluding URI. General Resource cache/commit/session/dirty reuse parity is outside this named create contract.
- Source resource_type/namespace_name/connection/_synchronized constructor bindings can collide, base_path is a legitimate dynamic route override, and __conflicting_attrs/microversion have constructor behavior. Go explicitly rejects all 17 reserved route/client controls and keeps its captured client/parent/session/version/header ownership; these controls are not all Python keyword collisions.
- Parent is a safe nonempty UTF8<=80rune literal with route/control/dot rejection and exact-once escaping, without Python Resource/Ref/name lookup. This local route policy does not constrain passive create Body id/name, which remain raw JSON; server schema and readonly/date constraints are not emulated.
- Whole UTF8 and complete parsed object boundaries are explicit Go choices. Both new owned creation and Source tolerate empty or invalid JSON syntax; malformed valid nonobjects fail. Existing raw Create remains strict201 and typed, while owned CreateRecord uses actual200..399.
- The SDK has no admin-only branch, but pinned Glance server policy57f7dd9e assigns both association writes to metadef_admin -> role:admin. Classified as core admin; server overrides and authenticated permission/protection/DB effects remain unexecuted. See docs/glance-policy-priorities.md. No full Python Resource or server-policy parity is claimed.

### `python:image/v2/delete_metadef_resource_type_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.NamespaceScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithDeleteHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.NamespaceScope.DeleteRecord`

- InNamespace is zeroHTTP; required safe literal namespace, original API/client/provider/type/endpoint/ServiceURL/microversion captured for scope lifetime without retained context or parentGET/lookup/Ref. Selected child is nonempty UTF8<=80runes with route-breaking/control/dot rejection and exact-once path escape, no trim/UUID/nameFind. Latest valid ordinary source headers copied per request and original provider auth remains live, guards before callbacks/request/accepted body/each consumedrow.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFixedRoutesAndPassiveModels`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreCompletePreflight`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreSourceLifetimeAndNativePolicies`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_resource_types_test.go:TestConnectionImageMetadefResourceTypesShareClientAndScopes`.
- Delete concrete option family uses FullOpts whole replacement/lastwins and independent helper/header/map/pointer/option-slice/config snapshots before/after callbacks. Callback once/request; nil/error callback/owned invalid headers fails preHTTP, no runtime builder or auth/transport callback bridge.
  Tests: `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsCallbacksAndRequestOwnership`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsValidationAndDefaults`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsParallelReusableHelpers`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesOwnedPreparationAndSource`.
- One fixed childDELETE original204; IgnoreMissing nildefaulttrue privately accepts404 and bypasses native404 RetryFunc only there. Cleanphysically owned404 after completeRead/Close/context/source checks returns nil,nil; handlingfailure returns nil,error with wholeactual404 ResponseError. Strictfalse retains native404 ownership/hooks. Only204 ACK+error handlingretention. Hidden parent404 not absenceproof, no preflightGET/unprotect/count/globalcleanup; association removed only, globaltype remains.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFixedRoutesAndPassiveModels`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreCompletePreflight`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesFixedRoutesAndPayloads`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesResponseOwnershipAndNativeHooks`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsValidationAndDefaults`.
- Onlyactual204 Delete returns opaque Acknowledgement requestedNamespace/Name with independent wholeBody/Header/status, ACK retained with accepted204handlingerror. Owned404 never yieldsACK: cleanignore nil,nil or handlingfailure nil,error with actual404ResponseError. OnceClose/no replay and read/Close/contextErr/customCause/source evidence retained.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesResponseOwnershipAndNativeHooks`.
- Configured native prebody retry/reauth/backoff and same-target redirects preserved; exact method/origin/path/query/serialized body/owned response fields and originalstatus gates. Native MoreHeaders mutation remains advanced provider policy; no ordinary-header hook rejection promise. Preserve read/Close/transport/ctx.Err/customCause and native ErrOriginal/ErrReauth fields without inventing Unwrap.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreSourceLifetimeAndNativePolicies`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesResponseOwnershipAndNativeHooks`.
- Owned DeleteRecord chooses ID string or a private pre-option RawResource clone, never both; reads present id including null before fallback name, discards all other input Body/namespace/location/receipt, never mutates caller Resource. Selected safe literal UTF8<=80rune identity and captured parent determine one query/body-free child DELETE with no parent/child/global lookup, follow-self, unprotect or cleanup. Delete does not call CloudLocation.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordDeleteOpaqueStatusesAndFixedIdentitySnapshot`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationCompletePreflight`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationStickySourceAndOuterPreflight`, `connectiontest/connection_image_metadef_association_mutations_test.go:TestConnectionImageMetadefAssociationMutationsShareLocationAndIdentity`.
- Owned DeleteRecord reuses existing concrete DeleteOpts/Header(s)/IgnoreMissing with full replacement, independently copied header/ignore pointer/config/options before and after callbacks and once/request execution. Literal identity validates before callbacks; missing/both/null/nonstring/unsafe input, nil options and owned header conflicts reject preHTTP. Retained callback pointers and source header changes cannot retarget identity or captured policy.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordDeleteOpaqueStatusesAndFixedIdentitySnapshot`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordDeletePhysicalMissingAndOptionOwnership`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationCompletePreflight`.
- Owned DeleteRecord accepts actual200..399 as opaque arbitrary bytes without JSON or UTF8 decoding and returns independently owned Acknowledgement selected Namespace/Name plus whole actual Body/Header/status. Read/Close/cancel/source/outer handling faults still return frozen ACK together with actual ResponseError and original causes, onceClose/no accepted replay; response fields never select another target.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordDeleteOpaqueStatusesAndFixedIdentitySnapshot`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationAcceptedFaultReceiptsAndFrozenTargets`, `connectiontest/connection_image_metadef_association_mutations_test.go:TestConnectionImageMetadefAssociationMutationsShareLocationAndIdentity`.
- Owned DeleteRecord IgnoreMissing nil/defaulttrue or explicittrue privately handles directly received physical404 without native404 retry and retains actual404 ACK, including ACK+ResponseError on handling failure. This evidence means handled missing, not successful deletion or proven absence. Explicitfalse preserves native404 Body/Header/status/retry handling without ACK; transport-claimed404 is never converted to physical missing. Source public proxy returns None and old raw Delete clean404 remains nil,nil.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordDeletePhysicalMissingAndOptionOwnership`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationNativeRetryOwnershipAndLegacyPolicies`, `connectiontest/connection_image_metadef_association_mutations_test.go:TestConnectionImageMetadefAssociationMutationsShareLocationAndIdentity`.
- Owned DeleteRecord source/scope/context/outer guard is sticky before/after options and physical/read/Close, even after restored observed state. Native configured prebody retry/reauth retains exact body-free method/target and original accepted status policy, forbids injected request body or expanded status acceptance, and preserves native/read/Close/context/custom causes. Existing raw strict204-only Delete ACK and clean404 semantics remain separate and unchanged.
  Tests: `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationStickySourceAndOuterPreflight`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationAcceptedFaultReceiptsAndFrozenTargets`, `image/v2/metadefresourcetypes/record_mutations_test.go:TestMetadefResourceTypeRecordMutationNativeRetryOwnershipAndLegacyPolicies`.

Python/Go differences:

- Source association delete passes the supplied Resource into Proxy._get_resource, which can reuse and mutate its namespace URI; unlike property deletion it does not first normalize the child to ID. Go privately snapshots only identity before options, never mutates input, and keeps a safe nonempty UTF8<=80rune fixed literal parent/child without Resource/Ref/name lookup.
- Source public delete returns None. Go owned DeleteRecord retains opaque actual200..399 acknowledgements and default physical404 receipts, including ACK with accepted handling failure. Old strict204 raw Delete still returns no ACK for clean handled404. Actual404 handling is bounded evidence, not server deletion success or absence proof.
- Defaulttrue directly owned physical404 bypasses native404 retry as explicit Go missing policy; explicitfalse keeps native404 response/retry errors. Transport claims and general Python NotFound hierarchy are not treated as proof of received physical404.
- Captured route/client/version/header ownership, local UTF8/literal route validation and immutable identity are explicit Go choices. General mutable Resource URI/session/cache/request-control behavior is outside this named deletion contract.
- The SDK has no admin-only branch, but pinned Glance server policy57f7dd9e assigns both association writes to metadef_admin -> role:admin. Classified as core admin; server overrides and authenticated permission/protection/DB effects remain unexecuted. See docs/glance-policy-priorities.md. No full Python Resource or server-policy parity is claimed.

### `python:image/v2/metadef_resource_type_associations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.NamespaceScope.List`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.NamespaceScope.All`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.NamespaceScope.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.NamespaceScope.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.NewWithDependencies`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes.WithRecordListFilters`

- InNamespace is zeroHTTP; required safe literal namespace, original API/client/provider/type/endpoint/ServiceURL/microversion captured for scope lifetime without retained context or parentGET/lookup/Ref.. Latest valid ordinary headers copied per request, original provider liveauth and fixed source/context guarded before callbacks/request/accepted body/each consumedrow; no child ID lookup or synthetic returned parent.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFixedRoutesAndPassiveModels`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreCompletePreflight`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreSourceLifetimeAndNativePolicies`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_resource_types_test.go:TestConnectionImageMetadefResourceTypesShareClientAndScopes`.
- Association adds exact optional nullable Prefix/PropertiesTarget to Name and literal date strings. Strict whole UTF8 nonnull object/wrong nonnull canonical types fail atomically, no partial mutation. Body/Header/pointers independently owned; unknown precision/raw protected/self/links/parent remain passive, no request identity seed or returned-name routing.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesCanonicalModelsAndRawOwnership`.
- List concrete options use FullOpts whole replacement/lastwins and independent helper/header/map/pointer/option-slice/config snapshots before/after callbacks. Callback once per request or lazy iteration; nil/error callback/owned invalid headers fails preHTTP. Shared ListOption is reusable across the two real list operations, without assigning Create/Delete helper contracts to a list review.
  Tests: `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsCallbacksAndRequestOwnership`, `image/v2/metadefresourcetypes/options_test.go:TestMetadefResourceTypesOptionsValidationAndDefaults`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesFiniteIterationAndLocalCaps`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesOwnedPreparationAndSource`.
- One lazy bodyless/queryless GET200, exact requirednonnull named array and wholeUTF8JSON checked before yielding, only consumed strict rows decoded. Preserve wire order/repeats; MaxItems0unlimited/negativeinvalid, cap/break skips unused malformed rows and extra HTTP. All nonnil empty on success/nil no partial on error. No query/limit/marker/sort/clientfilter/link-follow/fallback/cache even when response includes passive next metadata.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFiniteConsumptionAndRootEvidence`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesFiniteIterationAndLocalCaps`, `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFixedRoutesAndPassiveModels`, `internal/cmd/sdkgen/glance_metadef_resource_types_test.go:TestGlanceMetadefResourceTypesRegistryAndConcreteScope`.
- Original200 list only decoded, whole actualBody/Header/status and accepted read/Close/JSON/UTF8/canonical/contextErr/customCause/source errors preserved via ResponseError, onceClose/no replay; row/header/raw proof independently owned.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreFiniteConsumptionAndRootEvidence`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesResponseOwnershipAndNativeHooks`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesCanonicalModelsAndRawOwnership`.
- Configured native prebody retry/reauth/backoff and same-target redirects preserved; exact method/origin/path/query/serialized body/owned response fields and originalstatus gates. Native MoreHeaders mutation remains advanced provider policy; no ordinary-header hook rejection promise. Preserve read/Close/transport/ctx.Err/customCause and native ErrOriginal/ErrReauth fields without inventing Unwrap.
  Tests: `image/v2/metadefresourcetypes/core_test.go:TestMetadefResourceTypesCoreSourceLifetimeAndNativePolicies`, `image/v2/metadefresourcetypes/contracts_test.go:TestMetadefResourceTypesResponseOwnershipAndNativeHooks`.
- Declared untyped nullable fields, explicit id versus alternate name, collector-dependent computed location and fixed namespace URI are projected independently of complete Wire/page/headers/status. Connection Image and ImageV2 share client and captured current location; URI parent is an explicit additional Go view field, not default Python to_dict serialization.
  Tests: `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsFixedRoutesProjectionAndReceipts`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsIdentityPresenceAndLocation`, `connectiontest/connection_image_metadef_records_test.go:TestConnectionImageMetadefRecordsShareClientAndCurrentLocation`.
- Concrete options own callbacks/header/filter/pointer/slice snapshots and full replacement. Only limit/marker are server queries; declared Body filters are local, unknown captured encoding errors are ignored, path/session/version controls and typed/semantic collisions fail before HTTP. Raw MaxItems cap and wire limit hint precede filters and unconsumed rows.
  Tests: `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsLazyRepeatAndOptionSnapshots`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsSemanticFilters`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsCompletePreflight`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsPreflightOrderAndLocationFailures`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsRawCapAndEarlyStop`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsFullOptionsAndRetainedCallbacksOwnStorage`.
- Shared pager accepts body links/plural links/next/HTTP Link and singleton rows. Explicit-limit marker fallback follows short nonempty pages from retained wire identity, while cap/single-page/empty-page/early-break avoid unused continuation work. Same-origin/collection/filters, conflicting links and marker/cycle rules are explicit guarded Go choices; partial AllRecords survive later failure.
  Tests: `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsMarkerFallbackUsesWireIdentity`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsContinuationRepresentations`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsRejectedContinuationKeepsPartialReceipt`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsRawCapAndEarlyStop`.
- Actual200..399 requires whole-page UTF8 and mandatory plural JSON representation; empty204 and consumed malformed rows fail with actual page receipt. Accepted read/Close/cancellation causes and rejected native status evidence are preserved; no accepted-body replay or invented response.
  Tests: `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsStatusEnvelopeAndEmptyPolicy`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsAcceptedReadCloseAndCancelCauses`.
- Native retries retain the logical query/source/status policy. Sticky observed source and outer guards cover callbacks, physical roundtrip/read/Close/retry and post-yield boundaries even if a changed source is later restored; existing finite typed List/All stay strict200 and query-free.
  Tests: `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsNativeRetryAndOwnership`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsSourceAndOuterGuardAcrossPhysicalBoundaries`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsObservedGuardFailureSurvivesRestoration`, `image/v2/metadefresourcetypes/list_records_test.go:TestMetadefResourceTypeRecordsLegacyFiniteListRemainsSeparate`.

Python/Go differences:

- Named list declarations map to owned records, semantic query/body filtering and paging; Python mutable Resource lifecycle/cache/adapter overrides remain separate common SDK goals. Go uses fixed literal namespace and a selected service version, guarded headers and a copied declared view with namespace_name URI provenance beyond default Python to_dict.
- Go deliberately requires whole-page UTF8/object/plural JSON, one positive query limit and a nonempty string marker; local JSON filtering distinguishes bool from number and compares exact decimal numbers. Unknown row fields remain complete only in Wire, separately from declared Resource.
- Shared Go paging restricts origin/collection/filter changes, conflicting continuation links and repeated markers, and accepts links.next dictionaries as an additional representation. Source prioritizes links and permits broader URL/query changes including /v-prefix handling. The actual pinned Glance controller lists finite rows and ignores limit/marker; callers can choose Paginated=false.
- No live authenticated cloud, Python execution, server policy/DB/collation/transaction effects or dynamic session/cache lifecycle equivalence is claimed. Existing typed strict200 finite APIs and create/delete contracts are preserved; record lists accept actual200..399 with mandatory representation.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"

for value in conn.image.metadef_resource_types(
    limit=10, max_items=20, paginated=False
):
    print(value.to_dict())

for value in conn.image.metadef_resource_type_associations(
    namespace, limit=10, max_items=20, name="OS::Nova::Server", paginated=False
):
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
    types "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    name := flag.String("name", "", "optional association name filter")
    limit := flag.Int("limit", 0, "explicit server page limit")
    maxItems := flag.Int("max-items", 0, "maximum consumed rows per list")
    paginated := flag.Bool("paginated", false, "follow list continuations")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *name, *limit, *maxItems, *paginated); err != nil {
        log.Fatal(err)
    }
}

func printRecord(record *types.Record) error {
    if record == nil { return nil }
    data, err := json.MarshalIndent(map[string]any{
        "namespace": record.Namespace,
        "resource": record.Resource,
        "wire": record.Wire,
        "envelope": record.Envelope,
        "header": record.Header,
        "status_code": record.StatusCode,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, name string, limit, maxItems int, paginated bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    api := service.API.MetadefResourceTypes
    options := []types.RecordListOption{
        types.WithRecordListOpts(types.RecordListOpts{Limit: limit, MaxItems: maxItems}),
        types.WithRecordListHeader("X-Request-Source", "metadef-records-example"),
        types.WithRecordListPaginated(paginated),
    }
    for record, readErr := range api.ListRecords(ctx, options...) {
        if readErr != nil { return readErr }
        if err := printRecord(record); err != nil { return err }
    }
    scope, err := api.InNamespace(ctx, namespace)
    if err != nil { return err }
    scopedOptions := []types.RecordListOption{
        types.WithRecordListOpts(types.RecordListOpts{
            Limit: limit, MaxItems: maxItems,
            Filters: associationFilters(name),
        }),
        types.WithRecordListHeader("X-Request-Source", "metadef-records-example"),
        types.WithRecordListPaginated(paginated),
    }
    records, readErr := scope.AllRecords(ctx, scopedOptions...)
    for _, record := range records {
        if err := printRecord(record); err != nil { return errors.Join(readErr, err) }
    }
    return readErr
}

func associationFilters(name string) []resource.ListOption {
    if name == "" { return nil }
    return []resource.ListOption{resource.WithFilter("name", name)}
}
```

## References

- [Metadefresourcetypes](v2/metadefresourcetypes/README.md)
- [Resource collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L805-L844)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1696-L1786)
- [Resource class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_resource_type.py)
- [inherited list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2459)
- [List Records Test](v2/metadefresourcetypes/list_records_test.go)
- [Connection Image Metadef Records Test](../connectiontest/connection_image_metadef_records_test.go)
- [leaf core fixture](v2/metadefresourcetypes/core_test.go)
- [Contracts Test](v2/metadefresourcetypes/contracts_test.go)
- [List Control Test](../internal/rest/list_control_test.go)
- [List Short Page Test](../internal/rest/list_short_page_test.go)
- [Filter Test](../internal/jsonfilter/filter_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
