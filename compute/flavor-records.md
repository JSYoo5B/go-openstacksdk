# Flavor Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/find_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.FindFlavor`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.FindFlavor`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindLimit`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindMarker`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindPaginated`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindFilter`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindFilters`

- Pinned direct find_flavor accepts a string, ignore_missing=True, keyword get_extra_specs=False and query. Safe identities use native GET200 with the flavor envelope and native model/Swap decoding; compatible GET400/403/404 may fall back, with explicit strict, 404-only and GET-only Go policies.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsMissingPoliciesAndUnsafeNames`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`.
- Fallback uses native /flavors/detail and all FlavorPage continuations without an automatic name hint. SDK list-only is_public=None is added only for an absent caller key; direct GET remains the frozen caller query. Repeated and explicit nil/empty query keys retain precedence and raw wire extensions remain observable.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsListDefaultsAndCallerQuery`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsLookupClosesBeforeEnrichment`.
- Extra specs enrichment defaults to false. Explicit true enriches only a validated direct GET or a fully consumed unique ID-or-name match whose inline ExtraSpecs is nil/empty. It uses the returned canonical ID for native GET200 /os-extra_specs without caller query, clones the Flavor/model map and replaces fetched specs. Nonempty inline specs, missing lookup, ambiguity and late failures produce no enrichment request.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsLookupClosesBeforeEnrichment`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsUsesResolvedIDAndOwnsReturnedModel`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsInlineAndNativeNilEmptyDefaults`, `connection_find_identity_flavor_test.go:TestConnectionFlavorFindIdentityExtraSpecs`.
- Extra-specs GET403/404, undeclared status, typed decode, read/transport errors and cancellation remain terminal without another lookup or IgnoreMissing suppression. Native missing/null extra_specs wrappers remain nil maps; the separate GET is not incorrectly gated at inline-field introduction microversion2.61.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsEnrichmentFailuresAreTerminal`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsNativeFailuresRemainTerminal`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsKeepsLiveProviderVersionAndContext`.
- Bulk/custom options, bool pointers and query maps are frozen across callback phases and concurrent reuse. Generated leaf and handwritten Flavor collections retain the original provider, live token/headers/microversion/HTTP middleware/context and reverse ResourceBase. Extra-spec maps are independent; invalid controls, unsupported capabilities and unsafe GET-only names fail before HTTP.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsSnapshotsConcurrentReuseAndLastWins`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsKeepsLiveProviderVersionAndContext`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsSharesLiveSourceAndFreezesInput`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsRejectsInvalidInputAndCancellationBeforeHTTP`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsConcurrentCallsReturnIndependentMaps`.
- SDK-owned list default maps, bool options and enrichment callback are frozen before user/native callbacks; caller nil/empty keys retain precedence. Extra-specs capability and reserved controls fail before HTTP, callback errors/results remain terminal, and ordinary lookup plus secondary-search defaults remain isolated.
  Tests: `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsDefaultsOnlyAffectIdentityLists`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsCapabilityAndQueryPreflight`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsEnrichesValidatedUniqueSuccess`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsListFailuresAndAbsenceNeverEnrich`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsErrorsAndInvalidResultsAreTerminal`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsSnapshotsMetadataOptionsAndCallback`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsConcurrentOwnedOptionReuse`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsDefaultsPreserveOrdinaryAndSecondarySearches`.
- The generated Flavor Resources bridge corrects ordinary raw status forwarding as well as identity fallback. Ordinary All and explicit Name Find consume their native pages without implicit is_public=None, identity enrichment or extra member GET.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsOrdinaryCollectionsRetainQueryWithoutIdentityPolicies`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- The primary owned FindFlavor always attempts one escaped literal member GET, including names with spaces or reserved URL characters, without a name hint or list-only is_public default. Semantic attributes retain their original member query spellings and vendor parameters while known attributes seed the Resource view; semantic id conflicts fail before discovery or member HTTP.
  Tests: `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`, `resource/filters_test.go:TestSemanticFilterSelectionKeepsMemberSpellingSeedAndDeferredUnknownErrors`.
- Accepted member 200..399 retains its whole HTTP envelope, headers and status. A flavor envelope selects only the actual member object, otherwise the flat root is used; known response attributes overlay the input seed in parsed JSON alias order. Omission preserves seed, explicit null overrides it, and raw id/name/original_name truthiness plus response descriptors produce the owned view independently of Wire. Accepted empty, opaque or malformed JSON returns the seed with nil Wire; parsed null or array representations return a receipt error.
  Tests: `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDeclaredDefaultsDescriptorsAndRawIdentity`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordResponseAliasOrderAndIndependentReceipts`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDescriptorErrorsAndPassiveStringIdentity`.
- Only clean member 400/403/404 uses the prepared detailed fallback with transposed server queries, list-only is_public=None and local Body filters. Passive JSON string ID or name is compared exactly; numeric values are not stringified. A unique match requires successful consumption of the selected list, a second matching row immediately returns Ambiguous, a late list failure remains terminal, and zero matches returns nil by default or NotFound with IgnoreMissing(false).
  Tests: `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`.
- GetExtraSpecs defaults false. Owned true conditionally enriches only the successfully resolved unique result whose projected extra_specs is falsey; inline nonempty dictionaries, absence, duplicates and late list failures do not fetch. The separate read uses the logical string identity and the same prepared version without caller query, exposes independent Enrichment receipts, and preserves the original row or member Wire.
  Tests: `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- The owned find prepares options, source and effective version once across member, fallback and enrichment. Nil version retains the selected client or discovers at most 2.61, explicit empty stays versionless, and the original client remains unchanged. Service.FindFlavor captures Connection location before options and adds it only to the view. Live tokens continue across reads; accepted read/Close or source errors preserve partial receipts and never fall back, become missing, or invoke an SDK response resend.
  Tests: `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.

Python/Go differences:

- Retained native-shaped API.FindIdentity path: Go exposes SDK-owned typed GetExtraSpecs options and explicit raw wire query. Python list transposes min_ram/min_disk aliases and automatically classifies declared Body fields; Go uses minRam/minDisk and preserves explicit wire extensions instead. Raw get_extra_specs is an SDK control, not a service query.
- Retained native-shaped API.FindIdentity path: Go explicit nil/empty query slices retain key presence, suppress list defaults and omit the wire value; pinned Flavor.list instead converts missing or Python None is_public to the string None. Native missing/null fetched extra_specs remains a nil map, whereas Python fetch_extra_specs defaults a missing wrapper to an empty dict.
- Retained native-shaped API.FindIdentity path: The lookup keeps strict safe nonblank result-ID validation, native Flavor/Swap extraction and native full-page pagination. Details/AllProjects options are unsupported on Flavor; its actual list override has no details parameter, and its default base_path is /flavors/detail. Ordinary Get/List/explicit Ref lookup do not gain identity list defaults or enrichment. The generated Flavor Resources list bridge also fixes whole-map forwarding: ordinary List/All now preserve raw status instead of dropping it.
- The primary Python named-operation mapping is Service.FindFlavor or API.FindFlavor returning owned Resource/Wire/receipts; the retained native-shaped FindIdentity contracts above continue to document the independent typed ABI and its explicit fallback/enrichment policies. The owned leaf view has location=null; the Service wrapper adds one captured Connection CloudLocation to Resource and Enrichment.Resource without changing actual response evidence.
- Owned semantic filters transpose only on list fallback; original spellings and vendor attributes reach member GET. The common Go member query encoder accepts scalar, null and scalar-array values, so object-valued or nested-array semantic inputs fail before HTTP, including local Body predicates that Python may encode as arbitrary iterables. Source controls such as arbitrary session/path/resource_type, deprecated JMESPath and cache-bypass controls remain outside this fixed-route typed API and the broader SDK lifecycle goal.
- Go query booleans use lowercase true/false. The shared local matcher distinguishes JSON booleans from numbers and treats nested object-subset matching against a scalar as false; it does not reproduce Python empty-dict truthiness or nested scalar attribute errors. Source input dictionary order is unavailable in a Go map, so canonical seed spellings win over body wire aliases; actual selected response alias order is preserved.
- Executable input identities and markers must be nonblank UTF-8 literal strings without routing controls; passive numeric/container response identities are retained as JSON values and are never converted into string comparisons or specs request paths. The shared extra-specs target validator retains its stricter safe-ID policy.
- Owned reads accept 200..399 and validate nonnull UTF-8 object representations; member unavailable JSON including invalid UTF-8 is tolerated as a seed-only fetch, while physical read/Close and source/context failures remain terminal. Python charset replacement and mutable Resource/session/conditional-cache behavior are separate lifecycle scope. Raw and descriptor views own their bytes; nonfinite float/overflow cannot be represented in a finite JSON view, and the audited numeric conversions use explicit IEEE-754 and Unicode 16 rules.
- Dictionary next links are a deliberate Go compatibility extension. Initial positive limits enable logical last-raw-row marker fallback; the first advertised server limit may be accepted and fixed without enabling an absent initial fallback. Explicit row caps and first-page controls scope the uniqueness observation to the selected inventory.

### `python:compute/v2/flavors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ListFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListDetails`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListLimit`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListMarker`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListHeader`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListFilter`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListFilters`

- The owned flavor iterator defaults to /flavors/detail with Details=true and GetExtraSpecs=false; Details(false) selects /flavors. Semantic classification supports seven server-query attributes, min_disk/min_ram aliases, twelve local Body attributes and ignored unknown list attributes. Absent or semantic-null is_public sends literal None, while empty string and false remain distinct; canonical query spelling wins aliases in one bulk map, and conflicting semantic versus typed/raw query channels fail before HTTP.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `resource/filters_test.go:TestSemanticFilterSelectionKeepsMemberSpellingSeedAndDeferredUnknownErrors`, `resource/filters_test.go:TestSemanticFiltersCanonicalPrecedencePresenceAndExactQueryEncoding`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`.
- Flat list rows project the thirteen declared Body values plus location into an owned view: raw id/name/original_name truthy fallback, omission-only name alias, absent defaults, explicit null, integer/boolean/float/dictionary response conversions and parsed response alias order. Nested flavor and vendor values remain actual Wire fields. Wire, row envelope, page headers and status stay independent; Service.ListFlavors captures current location once before options and adds it only to view copies.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDeclaredDefaultsDescriptorsAndRawIdentity`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordResponseAliasOrderAndIndependentReceipts`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDescriptorErrorsAndPassiveStringIdentity`, `internal/jsonfilter/descriptor_float_test.go:TestDescriptorFloatJSONSourceConversionsAndOwnership`, `internal/jsonfilter/descriptor_float_test.go:TestDescriptorFloatJSONRejectsInvalidStringsAndNonfiniteWithCauses`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`.
- Each iterator consumption prepares options lazily once and follows rel/href links, flavors_links, top-level next and HTTP Link. Initial effective positive limits allow last-consumed-raw-row logical markers even for filtered rows and caller-mutated views; the first advertised server limit is fixed without creating an initial-limit fallback. Raw row caps precede filters and avoid decoding unconsumed bad rows; first-page, empty-page and break stop later reads, while consumed malformed rows and late HTTP errors remain observable with actual receipts.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`, `internal/rest/list_test.go:TestListFirstServerLimitIsOptInAndFixedAcrossContinuations`, `internal/rest/list_test.go:TestListHTTPLinksAreOptInAndConflictingNextLinksFail`, `internal/rest/list_test.go:TestListContinuationCannotWidenCollectionFiltersOrMarkers`.
- GetExtraSpecs(true) enriches only yielded, locally matched rows whose projected dictionary is falsey. Nonempty inline dictionaries skip GET; missing, null, empty or normalized nonobject dictionaries fetch through the existing prepared extra-specs reader. Filtering, cap and break precede enrichment. The child has independent actual response evidence and Resource copies, uses the logical string identity without list query, and actual HTTP404 or accepted Close404 remains terminal with the original row retained.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- Owned list/version policy preserves a selected client version, discovers at most 2.61 only when unselected, and supports explicit empty versionless override. One prepared policy and option application is shared with per-row enrichment; the original client is unchanged and live provider tokens remain in use. Model conversion, accepted-body failures, source/context failures and invalid controls retain their causes rather than fabricating successful rows or SDK resends.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDescriptorErrorsAndPassiveStringIdentity`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`.

Python/Go differences:

- Service.ListFlavors supplies Connection location; direct leaf API.ListRecords has location=null. FlavorRecord separates the computed owned response view from actual Wire, row envelope, page metadata and optional child receipt rather than exposing a mutable Python Resource/session.
- Go query booleans use lowercase true/false and the shared query encoder supports scalar/null/scalar-array values. The common local matcher preserves JSON type distinctions and scalar-versus-object subset policy instead of Python bool-number equivalence, empty-dict truthiness and nested scalar attribute failures.
- Typed Limit/MaxItems zero means omission/unlimited and negatives are rejected; an effective wire limit is one positive integer and a marker is one nonblank string. Semantic and typed/raw wire query collisions are errors even for equal values; raw wire query can replace the matching typed limit or marker independently. Arbitrary session/path/resource_type, cache-bypass and deprecated Resource/JMESPath controls remain separate SDK goal scope.
- Owned list reads accept 200..399 but require complete nonnull UTF-8 JSON objects and flavors arrays or singleton objects. Accepted empty204 is an error rather than native empty-page compatibility. Invalid representations and descriptors retain actual response receipts; nonfinite float/overflow fails the finite JSON view, with audited IEEE-754/Unicode 16 conversions and independent actual raw values.
- Dictionary next links are an explicit Go compatibility extension; safe continuation checks prevent origin/route/filter widening and cycles while preserving reverse proxy prefixes. Logical markers and executable specs targets require string identities; numeric/container identities remain raw descriptor values and are never stringified into request paths.
- The named flavor list/conditional-enrichment workflow is mapped; global mutable Resource, session/cache/discovery lifecycle and unrelated native/Cloud flavor operations are not claimed complete by this row.

### `python:cloud/_compute/ComputeCloudMixin/list_flavors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.AllFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`

- AllFlavors eagerly consumes the existing detailed owned Flavor list in source page order, retaining duplicates and producing a completed JSON array, including [] for an empty inventory. Its actual source get_extra default is false; an explicit true enriches falsey inline specs while false makes no child request. It sends the fixed detail list query, never Cloud filters or a name selector, and retains the chosen version and live token through all pages and child reads.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- The eager collection reuses the owned reader's guarded paging and conditional enrichment. A late page403 retains prior Inventory but cannot publish completed Value or Flavors. Requested child failure follows the same shared collection path: the Cloud Search composition proves the failing known row is retained, actual child404 is terminal, and an accepted child Close404 keeps the original physical cause plus separate row/child receipt without a completed collection.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- The shared Cloud read preparation applies captured typed/raw/header options once, checks the original service/client binding and preserves cancellation causes. AllFlavors rejects any present Filters pointer, including null and {}, before HTTP because its source declaration has no filter argument; nil explicitly clears an earlier filter. The reused inventory stage captures Connection location before callbacks and independently adds it to each projected Resource and child Resource while leaving actual Wire and receipts unchanged.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`.

Python/Go differences:

- The pinned source signature and body use get_extra=False and read no clouds.yaml override, despite the list_flavors docstring saying default True. Go follows the executed signature and exposes a concrete optional bool override.
- Python returns an eager list of connected mutable Flavor Resources. Go returns completed raw JSON Value/Flavors plus partial Inventory on failure; records retain an owned projected Resource, actual Wire/row receipt and separate child enrichment receipt. The owned response view can be modified by its caller; it is not an immutable Python Resource replacement.
- This Cloud composition reuses the separately mapped owned reader's documented JSON-domain descriptors, finite selected/discovered or explicit-versionless policy, route/control/continuation validation and Connection location snapshot. Generic mutable Resource/session/cache facilities and other native/Proxy declarations remain separate parts of the original SDK goal.
- AllFlavors names the eager Cloud operation without replacing Service.ListFlavors, which remains the lazy Proxy-list mapping. The shared options carrier does not add a filters parameter to this source operation: present filters are a Go preflight error and nil clears them.

### `python:cloud/_compute/ComputeCloudMixin/search_flavors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.SearchFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExpression`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`

- SearchFlavors defaults requested extra specs to true and completes the entire detailed inventory and conditional child reads before applying any identifier or Cloud filter. Explicit false skips enrichment. Even an ultimately unmatched row is enriched when required, and a dictionary can select newly fetched specs. No name or Cloud dictionary is forwarded into the early Proxy query/Body matcher. Empty successful inventory/selection is [] rather than an inventory error.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- After successful inventory completion, the existing Cloud selector applies per-row Python-shaped id/name exact-or-glob matching, preserving source order and duplicate matches, then ordered dictionary subset or JMESPath selection. Numeric logical IDs participate in Cloud string matching. Reached unknown predicates fail, predicates after an earlier mismatch remain unconsumed, and an empty identifier selection does not consume unknown dictionary keys. Expressions retain arbitrary scalar/object/array/false JSON Value and do not invent associated Flavor records.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifierPreservesPerRowExactGlobMatchesDuplicatesAndSourceOrder`, `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifiersUsePythonJSONDomainStringAndContainerRepresentation`, `internal/cloudfilter/mapping_test.go:TestCloudFilterOrderedMappingConsumesOnlyReachedKeysAndKeepsDuplicateFirstPosition`, `internal/cloudfilter/mapping_test.go:TestCloudFilterNestedEmptyObjectsUseActualTruthinessBeforeShapeOrMissingKeys`, `internal/cloudfilter/mapping_test.go:TestCloudFilterLeafEqualityKeepsExactArrayShapeAndPythonBooleanNumberPrecision`, `internal/cloudfilter/results_test.go:TestCloudFilterExpressionBridgeReturnsArbitraryOwnedJSONWithoutInventedRowIndices`.
- Any late inventory, requested child or local filter failure retains consumed Inventory without completed Value/Flavors. Accepted child failures preserve independent raw row/child receipts and their original read/Close causes. Reusable bulk pointer/raw filters and callback header carriers are owned, callbacks execute once, captured location is applied independently before local selection, and actual row Wire/Envelope is unchanged. Source mutation or callback cancellation stops later callbacks/HTTP and retains the custom cancellation cause; selection publication remains guarded.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `internal/cloudfilter/results_test.go:TestCloudFilterSelectOwnsViewsFiltersIndicesAndResultsBeforeGuardCallbacks`, `internal/cloudfilter/results_test.go:TestCloudFilterGuardInterruptsTraversalAndRetainsCancellationCauseWithoutPartialSelection`.

Python/Go differences:

- Python annotates a list result, but the executed string-filter branch can return arbitrary JMESPath values. Go exposes that raw JSON in Value and leaves Flavors unset for expression results; ordinary selector indices connect Flavors to the retained Inventory without synthesizing expression records.
- Dictionary filters are local Cloud predicates after complete enriched inventory, not early owned ListRecords query/Body filters. The shared JSON-domain Cloud selector represents Python truthiness, repr/glob, ordered predicate consumption and equality through finite documented Go rules rather than arbitrary Python object execution.
- Search uses the separately mapped reader's owned projected Resource plus captured Connection location, and keeps actual Wire/Envelope/Header/Status and child receipt separate. Partial Inventory is an additive diagnostic result on error. Selected/finite-discovered/explicit-empty version and typed shared-control differences remain documented in the owned guide; generic mutable Resource/session/cache lifecycle remains an independent SDK goal.

### `python:cloud/_compute/ComputeCloudMixin/get_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetFlavor`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`

- GetFlavor delegates directly to the owned FindFlavor reader with IgnoreMissing(true) and requested extra specs true by default; it does not call Cloud Search or First. Absent, explicit null and every source-falsey filter become an unfiltered Find. Truthy dictionaries retain original member query spelling and known seed attributes. Direct success keeps inline nonempty specs, or enriches the canonical returned ID when requested specs are falsey; explicit false skips the child and an explicit empty microversion remains versionless for both reads without discovery.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`, `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`.
- The direct Find delegation keeps its source-shaped clean400/403/404 detail-list fallback. Missing returns nil,nil by default, a second match immediately returns ambiguity without child enrichment, and only a unique fallback result is enriched. Captured location is applied to separate result/child Resources, and member and child status/header/envelope remain distinct. The existing reader preserves late fallback and accepted child failures as terminal partial evidence rather than inventing a successful Get.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- Truthy nonobject filters fail before HTTP instead of becoming Cloud expressions. Dictionary keys name_or_id, ignore_missing and get_extra_specs collide with explicit find arguments and are rejected before HTTP; the reused reader protects its explicit id argument and scalar original-query encoding. The shared captured read configuration and final binding check preserve reusable options, selected/live client state and guarded context/location preparation without a second application of Cloud callbacks.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`.

Python/Go differences:

- The source's deprecated filter docstring mentions strings/JMESPath, but its executed body normalizes falsey filters and expands truthy filters as Find kwargs. Go follows that direct delegation: truthy nonobjects and duplicate explicit kwargs are typed HTTP0 errors; it does not turn them into Cloud search. Python's deprecation warning is documented rather than emitted as a Go warning.
- Original member kwargs reuse the separately mapped Find encoder's scalar/null/scalar-array range. Dictionary-valued and nested-array values that Python may stringify or iterate are explicit Go input errors, distinct from SearchFlavors' nested dictionary predicates. The owned Find reader also retains its documented literal route validation, identity/filter/projection, version and continuation differences.
- Go returns an owned FlavorRecord and can retain a known partial record/child receipt with error, rather than returning a connected mutable Python Resource or silently losing physical evidence. Computed current location changes only Resource views, with Wire/Envelope/Header/Status kept actual. Generic mutable Resource/session/cache facilities and other native/Proxy/Cloud declarations remain separate portions of the original SDK goal.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(compute_api_version="2.55")
for flavor in conn.compute.flavors(min_ram=1024, disk=20):
    print(flavor.to_dict())

found = conn.compute.find_flavor("m1.small", ignore_missing=True,
                                 get_extra_specs=True)
if found is not None:
    print(found.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    identity := flag.String("flavor", "", "optional flavor ID or name to find after listing")
    extra := flag.Bool("extra", false, "conditionally fetch missing or empty extra specs")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *identity, *extra); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, identity string, extra bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    printRecord := func(operation string, record *flavors.FlavorRecord) error {
        if record == nil { return nil }
        var enrichment any
        if child := record.Enrichment; child != nil {
            enrichment = map[string]any{
                "resource": child.Resource, "wire": child.Wire,
                "extra_specs": child.ExtraSpecs, "envelope": string(child.Envelope),
                "header": child.Header, "status_code": child.StatusCode,
            }
        }
        data, err := json.MarshalIndent(map[string]any{
            "operation": operation, "resource": record.Resource, "wire": record.Wire,
            "envelope": string(record.Envelope), "header": record.Header,
            "status_code": record.StatusCode, "enrichment": enrichment,
        }, "", "  ")
        if err != nil { return err }
        fmt.Println(string(data))
        return nil
    }
    for record, err := range service.ListFlavors(ctx,
        flavors.WithFlavorListMaxItems(20),
        flavors.WithFlavorListExtraSpecs(extra)) {
        if printErr := printRecord("list", record); printErr != nil { return errors.Join(err, printErr) }
        if err != nil { return fmt.Errorf("list flavors: %w", err) }
    }
    if identity == "" { return nil }
    record, err := service.FindFlavor(ctx, identity, flavors.WithFlavorFindExtraSpecs(extra))
    if printErr := printRecord("find", record); printErr != nil { return errors.Join(err, printErr) }
    if err != nil { return fmt.Errorf("find flavor: %w", err) }
    if record == nil { fmt.Println("flavor not found") }
    return nil
}
```

## References

- [FetchExtraSpecs](flavor-extra-specs.md)
- [Console Selection](console-selection.md)
- [Flavor Property And Native](flavor-property-and-native.md)
- [Proxy flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L269-L296)
- [Proxy find_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L165-L199)
- [Flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py#L43-L138)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
