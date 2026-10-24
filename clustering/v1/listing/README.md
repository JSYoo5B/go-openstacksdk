# Listing

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/actions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.WithListMicroversion`

- Concrete/default query values and defensive snapshots preserve service-specific filters, raw responses and actual list capability.
  Tests: `api/clustering_read_test.go:TestClusteringReadActionListMarkersDefaultsAndOptionsSnapshot`.
- With an explicit positive limit, every nonempty page including a short page advances by its retained raw response ID; consumer model mutation cannot replace the wire marker and an empty page ends the stream.
  Tests: `api/clustering_read_test.go:TestClusteringReadShortPagesContinueUsingUnchangedWireIDs`, `internal/rest/list_short_page_test.go:TestListShortPageMarkerPolicyUsesWireRowsAndStopsOnEmptyPage`.
- MaxItems counts validated wire rows before local filtering and zero is unlimited; positive caps supply a limit hint only when no explicit limit exists. Paginated=False consumes the first response, reusable options own bool pointers and later options win. Reaching a cap or an empty page stops before unused rows/continuations while defaults, context and next-page source guards remain active.
  Tests: `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSevenFacadesCountRawRowsBeforeFilters`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSinglePageIgnoresUnneededContinuation`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsLimitHintLastWinsAndExactBoundary`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsOwnPaginatedPointersAndOverrideOrder`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsZeroUnlimitedTrueOverridesFalseAndEmptyStops`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsNegativeAndReservedQueryFailBeforeHTTP`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsConsumerBreakAndNextPageSourceGuard`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Known pinned Body fields and Python attribute aliases can be selected explicitly for local raw JSON comparison, including query-capable fields; unknown/source-excluded fields and raw Body query extensions fail before child HTTP.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSourceAliasesAndQueryCapableFieldsStayLocal`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersKnownQueryUnknownFieldsAndNamespacesFailBeforeChildHTTP`.
- Filter JSON and canonical maps are owned snapshots, alias/canonical options use last-wins and each iterator traversal owns its prepared arguments; missing and explicit null compare as null without erasing raw presence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSixFacadesSnapshotAliasesAndReuse`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersMissingAndExplicitNullRemainDistinctFromValues`, `internal/senlin/filter_options_test.go:TestBodyFilterOptionsSnapshotCanonicalAliasesAndOwnRawMaps`.
- Raw pagination and validation precede filtering: nonmatching rows consume caps and determine markers, single-page/cap stops skip unused continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCapsAndSinglePageBeforeMatching`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersFilteredPagesKeepRawMarkersAndParent`.
- Nested object subsets, exact large decimal values, arrays of complete objects, bool/number distinction and empty-object behavior are verified using the real typed facade.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersNestedSubsetsExactNumbersArraysAndBooleans`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go separates concrete server query inputs from explicit WithListFilter raw Body inputs rather than automatically partitioning Python keyword arguments. Query-capable fields can also be selected for local filtering; only known pinned fields and owned namespaces are accepted.
- Raw JSON comparison keeps exact decimal numbers and distinguishes bool from number; object filters match recursive subsets of nonempty objects, while arrays remain ordered complete values. Missing scalar fields and explicit null compare as null without descriptor coercion or cached Resource mutation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python maps the keyword target_id to the wire key target (query keys: name, action, status, sort, global_project, cluster_id); Go uses the wire name target and applies no alias.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/events`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.API.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.WithListMicroversion`

- Concrete/default query values and defensive snapshots preserve service-specific filters, raw responses and actual list capability.
  Tests: `api/clustering_read_test.go:TestClusteringReadEventRepeatedFiltersSnapshotAndExactBody`.
- With an explicit positive limit, every nonempty page including a short page advances by its retained raw response ID; consumer model mutation cannot replace the wire marker and an empty page ends the stream.
  Tests: `api/clustering_read_test.go:TestClusteringReadShortPagesContinueUsingUnchangedWireIDs`, `internal/rest/list_short_page_test.go:TestListShortPageMarkerPolicyUsesWireRowsAndStopsOnEmptyPage`.
- MaxItems counts validated wire rows before local filtering and zero is unlimited; positive caps supply a limit hint only when no explicit limit exists. Paginated=False consumes the first response, reusable options own bool pointers and later options win. Reaching a cap or an empty page stops before unused rows/continuations while defaults, context and next-page source guards remain active.
  Tests: `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSevenFacadesCountRawRowsBeforeFilters`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSinglePageIgnoresUnneededContinuation`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsLimitHintLastWinsAndExactBoundary`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsOwnPaginatedPointersAndOverrideOrder`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsZeroUnlimitedTrueOverridesFalseAndEmptyStops`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsNegativeAndReservedQueryFailBeforeHTTP`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsConsumerBreakAndNextPageSourceGuard`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Known pinned Body fields and Python attribute aliases can be selected explicitly for local raw JSON comparison, including query-capable fields; unknown/source-excluded fields and raw Body query extensions fail before child HTTP.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSourceAliasesAndQueryCapableFieldsStayLocal`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersKnownQueryUnknownFieldsAndNamespacesFailBeforeChildHTTP`.
- Filter JSON and canonical maps are owned snapshots, alias/canonical options use last-wins and each iterator traversal owns its prepared arguments; missing and explicit null compare as null without erasing raw presence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSixFacadesSnapshotAliasesAndReuse`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersMissingAndExplicitNullRemainDistinctFromValues`, `internal/senlin/filter_options_test.go:TestBodyFilterOptionsSnapshotCanonicalAliasesAndOwnRawMaps`.
- Raw pagination and validation precede filtering: nonmatching rows consume caps and determine markers, single-page/cap stops skip unused continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCapsAndSinglePageBeforeMatching`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersFilteredPagesKeepRawMarkersAndParent`.
- Nested object subsets, exact large decimal values, arrays of complete objects, bool/number distinction and empty-object behavior are verified using the real typed facade.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersNestedSubsetsExactNumbersArraysAndBooleans`.
- A numeric raw level matches the corresponding exact numeric filter and does not match the typed-model string spelling.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersEventLevelUsesRawJSONRatherThanTypedString`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go separates concrete server query inputs from explicit WithListFilter raw Body inputs rather than automatically partitioning Python keyword arguments. Query-capable fields can also be selected for local filtering; only known pinned fields and owned namespaces are accepted.
- Raw JSON comparison keeps exact decimal numbers and distinguishes bool from number; object filters match recursive subsets of nonempty objects, while arrays remain ordered complete values. Missing scalar fields and explicit null compare as null without descriptor coercion or cached Resource mutation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python maps obj_id, obj_name and obj_type to the wire keys oid, oname and otype (other keys: cluster_id, action, level, sort, global_project); Go uses the wire names. Raw Event.level filtering stays separate from the typed string conversion.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/policy_types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes.API.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes.WithListMicroversion`

- Catalog lists follow inherited body/header links within their origin/path with sticky filters and lazy breaks; no undocumented marker fallback is inferred from type names.
  Tests: `api/clustering_discovery_test.go:TestClusteringDiscoveryCatalogLinksPreserveExtensionsAndNoMarkerFallback`, `api/clustering_discovery_test.go:TestClusteringDiscoveryHTTPLinksBreakAndTargetGuard`.
- MaxItems supplies a wire-limit hint only when no explicit limit exists, caps validated rows locally and zero is unlimited. Paginated=False consumes the first response; bool pointers/query inputs are owned across lazy repeated iterations and later options win. Cap/single/empty stops skip unused decoding or continuation errors while unbounded row/link errors and per-page selected-version guards retain accepted-page evidence.
  Tests: `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsHintsCapsAndSnapshots`, `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsStopBeforeUnusedRowsAndLinks`, `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsPreflightAndPageVersions`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Known pinned Body fields and Python attribute aliases can be selected explicitly for local raw JSON comparison, including query-capable fields; unknown/source-excluded fields and raw Body query extensions fail before child HTTP.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSourceAliasesAndQueryCapableFieldsStayLocal`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersKnownQueryUnknownFieldsAndNamespacesFailBeforeChildHTTP`.
- Filter JSON and canonical maps are owned snapshots, alias/canonical options use last-wins and each iterator traversal owns its prepared arguments; missing and explicit null compare as null without erasing raw presence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSixFacadesSnapshotAliasesAndReuse`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersMissingAndExplicitNullRemainDistinctFromValues`, `internal/senlin/filter_options_test.go:TestBodyFilterOptionsSnapshotCanonicalAliasesAndOwnRawMaps`.
- Raw pagination and validation precede filtering: nonmatching rows consume caps and determine markers, single-page/cap stops skip unused continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCapsAndSinglePageBeforeMatching`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersFilteredPagesKeepRawMarkersAndParent`.
- Nested object subsets, exact large decimal values, arrays of complete objects, bool/number distinction and empty-object behavior are verified using the real typed facade.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersNestedSubsetsExactNumbersArraysAndBooleans`.
- Raw type id filters do not synthesize a name fallback, while the binding scope still requires a valid raw binding ID before local filtering and preserves failure evidence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCatalogIDDoesNotFallbackAndBindingValidationPrecedesFilter`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count. The inherited positive-cap limit hint is deployment-dependent because the official catalog/service endpoint declares no limit/marker protocol; Go still does not invent an ID marker fallback.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go separates concrete server query inputs from explicit WithListFilter raw Body inputs rather than automatically partitioning Python keyword arguments. Query-capable fields can also be selected for local filtering; only known pinned fields and owned namespaces are accepted.
- Raw JSON comparison keeps exact decimal numbers and distinguishes bool from number; object filters match recursive subsets of nonempty objects, while arrays remain ordered complete values. Missing scalar fields and explicit null compare as null without descriptor coercion or cached Resource mutation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python type-catalog and service resources forward limit and marker and, with an explicit limit, continue with the last row's Resource.id (the alternate name ID); Go sends limit hints only through opt-in local caps, has no marker fallback because these endpoints document no paging protocol, and follows only links the server returns.
- Python PolicyType.name is the alternate ID; Go does not apply a raw id-to-name fallback.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/profile_types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.API.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.WithListMicroversion`

- Catalog lists follow inherited body/header links within their origin/path with sticky filters and lazy breaks; no undocumented marker fallback is inferred from type names.
  Tests: `api/clustering_discovery_test.go:TestClusteringDiscoveryCatalogLinksPreserveExtensionsAndNoMarkerFallback`, `api/clustering_discovery_test.go:TestClusteringDiscoveryHTTPLinksBreakAndTargetGuard`.
- MaxItems supplies a wire-limit hint only when no explicit limit exists, caps validated rows locally and zero is unlimited. Paginated=False consumes the first response; bool pointers/query inputs are owned across lazy repeated iterations and later options win. Cap/single/empty stops skip unused decoding or continuation errors while unbounded row/link errors and per-page selected-version guards retain accepted-page evidence.
  Tests: `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsHintsCapsAndSnapshots`, `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsStopBeforeUnusedRowsAndLinks`, `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsPreflightAndPageVersions`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Known pinned Body fields and Python attribute aliases can be selected explicitly for local raw JSON comparison, including query-capable fields; unknown/source-excluded fields and raw Body query extensions fail before child HTTP.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSourceAliasesAndQueryCapableFieldsStayLocal`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersKnownQueryUnknownFieldsAndNamespacesFailBeforeChildHTTP`.
- Filter JSON and canonical maps are owned snapshots, alias/canonical options use last-wins and each iterator traversal owns its prepared arguments; missing and explicit null compare as null without erasing raw presence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSixFacadesSnapshotAliasesAndReuse`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersMissingAndExplicitNullRemainDistinctFromValues`, `internal/senlin/filter_options_test.go:TestBodyFilterOptionsSnapshotCanonicalAliasesAndOwnRawMaps`.
- Raw pagination and validation precede filtering: nonmatching rows consume caps and determine markers, single-page/cap stops skip unused continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCapsAndSinglePageBeforeMatching`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersFilteredPagesKeepRawMarkersAndParent`.
- Nested object subsets, exact large decimal values, arrays of complete objects, bool/number distinction and empty-object behavior are verified using the real typed facade.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersNestedSubsetsExactNumbersArraysAndBooleans`.
- Raw type id filters do not synthesize a name fallback, while the binding scope still requires a valid raw binding ID before local filtering and preserves failure evidence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCatalogIDDoesNotFallbackAndBindingValidationPrecedesFilter`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count. The inherited positive-cap limit hint is deployment-dependent because the official catalog/service endpoint declares no limit/marker protocol; Go still does not invent an ID marker fallback.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go separates concrete server query inputs from explicit WithListFilter raw Body inputs rather than automatically partitioning Python keyword arguments. Query-capable fields can also be selected for local filtering; only known pinned fields and owned namespaces are accepted.
- Raw JSON comparison keeps exact decimal numbers and distinguishes bool from number; object filters match recursive subsets of nonempty objects, while arrays remain ordered complete values. Missing scalar fields and explicit null compare as null without descriptor coercion or cached Resource mutation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python type-catalog and service resources forward limit and marker and, with an explicit limit, continue with the last row's Resource.id (the alternate name ID); Go sends limit hints only through opt-in local caps, has no marker fallback because these endpoints document no paging protocol, and follows only links the server returns.
- Python ProfileType.name is the alternate ID; Go does not apply a raw id-to-name fallback.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/services.API.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/services.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/services.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/services.WithListMicroversion`

- Concrete/default query values and defensive snapshots preserve service-specific filters, raw responses and actual list capability.
  Tests: `api/clustering_read_test.go:TestClusteringReadServicesVersionGateAndLocalStatusFilter`.
- MaxItems supplies a wire-limit hint only when no explicit limit exists, caps validated rows locally and zero is unlimited. Paginated=False consumes the first response; bool pointers/query inputs are owned across lazy repeated iterations and later options win. Cap/single/empty stops skip unused decoding or continuation errors while unbounded row/link errors and per-page selected-version guards retain accepted-page evidence.
  Tests: `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsHintsCapsAndSnapshots`, `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsStopBeforeUnusedRowsAndLinks`, `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsPreflightAndPageVersions`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links. Services retains its numeric1.7 preflight gate even with list controls.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsServicesVersionGateBeforeHTTP`.
- Known pinned Body fields and Python attribute aliases can be selected explicitly for local raw JSON comparison, including query-capable fields; unknown/source-excluded fields and raw Body query extensions fail before child HTTP.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSourceAliasesAndQueryCapableFieldsStayLocal`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersKnownQueryUnknownFieldsAndNamespacesFailBeforeChildHTTP`.
- Filter JSON and canonical maps are owned snapshots, alias/canonical options use last-wins and each iterator traversal owns its prepared arguments; missing and explicit null compare as null without erasing raw presence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSixFacadesSnapshotAliasesAndReuse`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersMissingAndExplicitNullRemainDistinctFromValues`, `internal/senlin/filter_options_test.go:TestBodyFilterOptionsSnapshotCanonicalAliasesAndOwnRawMaps`.
- Raw pagination and validation precede filtering: nonmatching rows consume caps and determine markers, single-page/cap stops skip unused continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCapsAndSinglePageBeforeMatching`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersFilteredPagesKeepRawMarkersAndParent`.
- An entirely filtered first page cannot bypass the selected numeric 1.7 service gate before advertised continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersServicesVersionGateOnFilteredContinuation`, `internal/senlin/catalog_filter_test.go:TestBodyFilterCatalogFacadesUseRawFieldsAndOwnNamespaces`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.
- Services>=1.7 and Receiver user>=1.4 gates evaluate the effective per-call client, allowing documented explicit upgrades and rejecting downgrades/default resets without changing source configuration.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestEffectiveVersionGates`, `internal/senlin/list_request_test.go:TestListRequestRevalidatesSourceBeforeContinuationAndGatesEffectiveVersion`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count. The inherited positive-cap limit hint is deployment-dependent because the official catalog/service endpoint declares no limit/marker protocol; Go still does not invent an ID marker fallback.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go separates concrete server query inputs from explicit WithListFilter raw Body inputs rather than automatically partitioning Python keyword arguments. Query-capable fields can also be selected for local filtering; only known pinned fields and owned namespaces are accepted.
- Raw JSON comparison keeps exact decimal numbers and distinguishes bool from number; object filters match recursive subsets of nonempty objects, while arrays remain ordered complete values. Missing scalar fields and explicit null compare as null without descriptor coercion or cached Resource mutation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python type-catalog and service resources forward limit and marker and, with an explicit limit, continue with the last row's Resource.id (the alternate name ID); Go sends limit hints only through opt-in local caps, has no marker fallback because these endpoints document no paging protocol, and follows only links the server returns.
- Python Service declares no query keys beyond limit and marker, so every other keyword is a client-side filter on Service attributes; Go provides explicit local filters for id, name, state, binary, host, updated_at, disabled_reason and status and the numeric 1.7 version gate.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/profiles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.WithListMicroversion`

- Lazy typed queries preserve selected filters and explicit false across links and last-wire-ID marker fallback, including short pages; local Body subsets, arrays, aliases and exact numeric filters never become server queries.
  Tests: `api/clustering_profiles_test.go:TestClusteringProfilesListShortPagesLocalFiltersAndSnapshot`, `api/clustering_profiles_test.go:TestClusteringProfilesLocalJSONFilterTypesArraysAndNull`, `api/clustering_profiles_test.go:TestClusteringProfilesPaginationGuardsAndConsumerControl`.
- MaxItems counts validated wire rows before local filtering and zero is unlimited; positive caps supply a limit hint only when no explicit limit exists. Paginated=False consumes the first response, reusable options own bool pointers and later options win. Reaching a cap or an empty page stops before unused rows/continuations while defaults, context and next-page source guards remain active.
  Tests: `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSevenFacadesCountRawRowsBeforeFilters`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSinglePageIgnoresUnneededContinuation`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsLimitHintLastWinsAndExactBoundary`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsOwnPaginatedPointersAndOverrideOrder`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsZeroUnlimitedTrueOverridesFalseAndEmptyStops`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsNegativeAndReservedQueryFailBeforeHTTP`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsConsumerBreakAndNextPageSourceGuard`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go takes concrete inputs, context and explicit ID/Name references instead of mutable Python Resource inputs and implicit combined-string identity guesses.
- The documented envelope is strict; raw fields, exact JSON numbers and accepted-response HTTP evidence are preserved rather than Python flat/empty fallback decoding and cached Resource mutation.
- Numeric selected microversions and protected body/query/header extensions replace implicit adapter negotiation; optional JSON values preserve omission, null and empty objects and the deployment validates null support.
- Explicit WithListFilter selects known local Body fields. JSON booleans stay distinct from numbers, decimal numbers compare exactly without float64, and non-object actual values fail object-subset filters without Python attribute errors.
- Go keeps continuation links on the same collection origin/path, retains original filters and selected limits, and guards URL/marker cycles rather than following arbitrary Python next-link parameters.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python profile query keys are sort, global_project, type and name; unknown keywords such as the documented metadata are discarded or become client filters, while Go reports them as filters or extensions explicitly.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithListMicroversion`

- Lazy typed queries preserve selected filters and explicit false across links and last-wire-ID marker fallback, including short pages; local Body subsets, arrays, aliases and exact numeric filters never become server queries.
  Tests: `api/clustering_policies_test.go:TestClusteringPolicyListShortPagesRetainFiltersAndSnapshot`, `api/clustering_policies_test.go:TestClusteringPolicyLocalFiltersSnapshotExactNumbersAndNull`, `api/clustering_policies_test.go:TestClusteringPolicyListDefaultBreakAndInvalidExtensions`, `api/clustering_policies_test.go:TestClusteringPolicyNameGrammarAndInFlightListCancellation`.
- MaxItems counts validated wire rows before local filtering and zero is unlimited; positive caps supply a limit hint only when no explicit limit exists. Paginated=False consumes the first response, reusable options own bool pointers and later options win. Reaching a cap or an empty page stops before unused rows/continuations while defaults, context and next-page source guards remain active.
  Tests: `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSevenFacadesCountRawRowsBeforeFilters`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSinglePageIgnoresUnneededContinuation`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsLimitHintLastWinsAndExactBoundary`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsOwnPaginatedPointersAndOverrideOrder`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsZeroUnlimitedTrueOverridesFalseAndEmptyStops`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsNegativeAndReservedQueryFailBeforeHTTP`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsConsumerBreakAndNextPageSourceGuard`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go takes concrete inputs, context and explicit ID/Name references instead of mutable Python Resource inputs and implicit combined-string identity guesses.
- The documented envelope is strict; raw fields, exact JSON numbers and accepted-response HTTP evidence are preserved rather than Python flat/empty fallback decoding and cached Resource mutation.
- Numeric selected microversions and protected body/query/header extensions replace implicit adapter negotiation; optional JSON values preserve omission, null and empty objects and the deployment validates null support.
- Explicit WithListFilter selects known local Body fields. JSON booleans stay distinct from numbers, decimal numbers compare exactly without float64, and non-object actual values fail object-subset filters without Python attribute errors.
- Go keeps continuation links on the same collection origin/path, retains original filters and selected limits, and guards URL/marker cycles rather than following arbitrary Python next-link parameters.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python policy query keys are name, type, sort and global_project.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/clusters`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithListMicroversion`

- Lazy owned query options retain explicit false, sort grammar, vendor query and selected filters across linked and short-page last-wire-ID continuation; constructor snapshots and consumer mutation cannot change raw markers, and local Body subsets/aliases/exact decimal filters do not become server query.
  Tests: `api/clustering_clusters_test.go:TestClusteringClustersListShortPageFiltersAndSnapshots`, `api/clustering_clusters_pagination_test.go:TestClusteringClustersLocalFilterJSONTypesAndNull`, `api/clustering_clusters_pagination_test.go:TestClusteringClustersContinuationGuardsBreakAndCancellation`, `api/clustering_clusters_pagination_test.go:TestClusteringClustersSharedStatusAndSortUseActualWireQuery`.
- MaxItems counts validated wire rows before local filtering and zero is unlimited; positive caps supply a limit hint only when no explicit limit exists. Paginated=False consumes the first response, reusable options own bool pointers and later options win. Reaching a cap or an empty page stops before unused rows/continuations while defaults, context and next-page source guards remain active.
  Tests: `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSevenFacadesCountRawRowsBeforeFilters`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSinglePageIgnoresUnneededContinuation`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsLimitHintLastWinsAndExactBoundary`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsOwnPaginatedPointersAndOverrideOrder`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsZeroUnlimitedTrueOverridesFalseAndEmptyStops`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsNegativeAndReservedQueryFailBeforeHTTP`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsConsumerBreakAndNextPageSourceGuard`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go uses concrete owned options, context and explicit ID/Name references rather than mutable Python Resource input; With scalar/null helpers preserve omission, explicit zero/false/empty and null without requiring caller builders.
- Documented singular envelopes and success codes are strict; raw HTTP/header/body evidence and exact JSON numbers are retained instead of Python permissive fallback decoding or cached Resource mutation.
- The selected numeric microversion and protected field/query/header extensions are explicit; null field support remains deployment-defined and no unverified force version gate is invented.
- Known local Body filters support recursive object subsets, arrays and exact decimal comparison while keeping JSON boolean and numeric types distinct; arbitrary vendor query keys are explicit server extensions.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python cluster query keys are name, status, sort and global_project.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/nodes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithListMicroversion`

- Lazy owned query options retain explicit false, sort grammar, vendor query and selected filters across linked and short-page last-wire-ID continuation; constructor snapshots and consumer mutation cannot change raw markers, and local Body subsets/aliases/exact decimal filters do not become server query.
  Tests: `api/clustering_nodes_pagination_test.go:TestClusteringNodesListWireOptionsAndRawMarkerSnapshots`, `api/clustering_nodes_pagination_test.go:TestClusteringNodesListPreflightAndSharedStatus`.
- MaxItems counts validated wire rows before local filtering and zero is unlimited; positive caps supply a limit hint only when no explicit limit exists. Paginated=False consumes the first response, reusable options own bool pointers and later options win. Reaching a cap or an empty page stops before unused rows/continuations while defaults, context and next-page source guards remain active.
  Tests: `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSevenFacadesCountRawRowsBeforeFilters`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSinglePageIgnoresUnneededContinuation`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsLimitHintLastWinsAndExactBoundary`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsOwnPaginatedPointersAndOverrideOrder`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsZeroUnlimitedTrueOverridesFalseAndEmptyStops`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsNegativeAndReservedQueryFailBeforeHTTP`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsConsumerBreakAndNextPageSourceGuard`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go uses concrete owned options, context and explicit ID/Name references rather than mutable Python Resource input; With scalar/null helpers preserve omission, explicit zero/false/empty and null without requiring caller builders.
- Documented singular envelopes and success codes are strict; raw HTTP/header/body evidence and exact JSON numbers are retained instead of Python permissive fallback decoding or cached Resource mutation.
- The selected numeric microversion and protected field/query/header extensions are explicit; null field support remains deployment-defined and no unverified force version gate is invented.
- Known local Body filters support recursive object subsets, arrays and exact decimal comparison while keeping JSON boolean and numeric types distinct; arbitrary vendor query keys are explicit server extensions.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python node query keys are show_details, name, sort, global_project, cluster_id and status; Go exposes the same through typed options (ShowDetails, ClusterID, Status) without aliases.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/receivers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithListMicroversion`

- Lazy/reusable GET200 preserves typed query, explicit global_project=false, user_id-to-user at selected>=1.4 in API preparation, public Resources and subsequent pages, local exact JSON filters and pre-filter wire-ID continuation on linked/short pages.
  Tests: `api/clustering_receivers_test.go:TestClusteringReceiversSynchronousCRUDAndResponseEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversMessagePresenceSnapshotsAndReuse`, `api/clustering_receivers_test.go:TestClusteringReceiversUpdateNameLookupFreezesInputsAndRechecksSource`, `api/clustering_receivers_test.go:TestClusteringReceiversUserGateResourcesBypassAndPageRecheck`, `api/clustering_receivers_test.go:TestClusteringReceiversLazyPagingLocalFiltersAndWireMarkers`, `api/clustering_receivers_test.go:TestClusteringReceiversFindDeleteMissingAmbiguousAndInvalidResolvedIDs`, `api/clustering_receivers_test.go:TestClusteringReceiversFindContinuationKeepsNameAndLaterPageOutcomes`, `api/clustering_receivers_test.go:TestClusteringReceiversAcceptedMalformedAndWrongCodesKeepEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversPreflightOwnedFieldsOptionsAndContext`.
- Connection exposes synchronous Receivers and asynchronous cluster policy commands with the same selected1.4, refreshed authentication token and reverse-proxy prefix. user reaches its wire query, global_project=false survives and no incidental Location/channel URL or accepted action is followed automatically.
  Tests: `connection_senlin_receivers_policy_test.go:TestConnectionSenlinReceiversAndPolicyCommandsShareSelectedSource`.
- MaxItems counts validated wire rows before local filtering and zero is unlimited; positive caps supply a limit hint only when no explicit limit exists. Paginated=False consumes the first response, reusable options own bool pointers and later options win. Reaching a cap or an empty page stops before unused rows/continuations while defaults, context and next-page source guards remain active.
  Tests: `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSevenFacadesCountRawRowsBeforeFilters`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSinglePageIgnoresUnneededContinuation`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsLimitHintLastWinsAndExactBoundary`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsOwnPaginatedPointersAndOverrideOrder`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsZeroUnlimitedTrueOverridesFalseAndEmptyStops`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsNegativeAndReservedQueryFailBeforeHTTP`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsConsumerBreakAndNextPageSourceGuard`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.
- Services>=1.7 and Receiver user>=1.4 gates evaluate the effective per-call client, allowing documented explicit upgrades and rejecting downgrades/default resets without changing source configuration.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestEffectiveVersionGates`, `internal/senlin/list_request_test.go:TestListRequestRevalidatesSourceBeforeContinuationAndGatesEffectiveVersion`.

Python/Go differences:

- Go uses stateless owned concrete options/context and explicit ID/Name references instead of Python mutable Resource input. Nonempty vendor receiver types pass through to the controller, known webhook inputs require cluster/action and no Python CLI default type or unproved message microversion gate is invented.
- Published201/200 strict receiver/receivers envelopes and204 deletion are enforced. Exact JSON Body, Header and StatusCode are retained, nullable action/cluster fields remain distinct from a Senlin action UUID, and channel or incidental Location URLs are never followed.
- Owned extension fields/query/headers are explicit Go capabilities rather than Python silently consuming unknown Resource attributes. Concrete/readonly fields and SDK authentication/version/transport headers are protected; null and deployment plugin semantics stay server decisions.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python maps user_id to the wire key user (keys: name, type, cluster_id, action, sort, global_project) and accepts an arbitrary sort string; Go validates the published sort key/direction set and sends user through the typed UserID option.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/cluster_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.API.InCluster`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.Scope.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.Scope.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.WithListMicroversion`

- Lazy GET200 list queries map is_enabled to enabled, preserve explicit false and policy_name/policy_type/sort plus owned vendor-query snapshots, repeat independently and stop after consumer break. Only advertised next links supply a marker; response binding or policy IDs never invent a marker fallback.
  Tests: `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesLazyLinkedPagesSnapshotsAndNoFallback`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesPreflightAndScopedWaits`.
- Each row retains independent raw JSON/HTTP metadata and separate canonical binding/policy/parent identities. Foreign continuation parents, malformed rows and source changes stop the collection with original accepted-page or native HTTP evidence.
  Tests: `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesRequiredRowsRetainWholeResponse`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactBodyIdentityOwnsRoutes`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExplicitLinksKeepParentAndSource`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactNameIgnoresCaseFoldedExtensions`.
- Binding MaxItems is a local raw-row cap without a wire-limit hint; Paginated=False consumes the first response. Snapshot bool pointers, later-option precedence and zero-unlimited defaults remain independent across lazy repeated iterations. Raw control query keys and initial limit/marker fail before list HTTP, while cap/empty stops ignore unused invalid rows/links and uncapped row errors retain whole-page evidence.
  Tests: `api/clustering_catalog_list_controls_test.go:TestClusteringBindingListControlsStayLocalAndOwnSnapshots`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links. Binding caps supply no wire-limit hint and initial page size remains unsupported.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Known pinned Body fields and Python attribute aliases can be selected explicitly for local raw JSON comparison, including query-capable fields; unknown/source-excluded fields and raw Body query extensions fail before child HTTP.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSourceAliasesAndQueryCapableFieldsStayLocal`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersKnownQueryUnknownFieldsAndNamespacesFailBeforeChildHTTP`.
- Filter JSON and canonical maps are owned snapshots, alias/canonical options use last-wins and each iterator traversal owns its prepared arguments; missing and explicit null compare as null without erasing raw presence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSixFacadesSnapshotAliasesAndReuse`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersMissingAndExplicitNullRemainDistinctFromValues`, `internal/senlin/filter_options_test.go:TestBodyFilterOptionsSnapshotCanonicalAliasesAndOwnRawMaps`.
- Raw pagination and validation precede filtering: nonmatching rows consume caps and determine markers, single-page/cap stops skip unused continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCapsAndSinglePageBeforeMatching`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersFilteredPagesKeepRawMarkersAndParent`.
- Nested object subsets, exact large decimal values, arrays of complete objects, bool/number distinction and empty-object behavior are verified using the real typed facade.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersNestedSubsetsExactNumbersArraysAndBooleans`.
- Raw type id filters do not synthesize a name fallback, while the binding scope still requires a valid raw binding ID before local filtering and preserves failure evidence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCatalogIDDoesNotFallbackAndBindingValidationPrecedesFilter`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.
- Per-call list settings retain the once-resolved canonical cluster and apply only to child list requests; the parent resolution and later scoped Get continue using source settings.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestBindingScopeAndOtherOperationsUnchanged`.

Python/Go differences:

- Go confirms and fixes a canonical parent through explicit ID/Name references instead of Python _proxy.py:1078-1081 extracting the supplied cluster ID and interpolating it directly.
- Senlin16.0.0 ClusterPolicyController.index whitelists enabled, policy_name, policy_type and sort. Go rejects initial limit/marker even through Resources and follows only same-origin, same-parent collection links preserving initial filters; advertised deployment markers remain allowed. This intentionally does not simulate unsupported stock-server pagination.
- WithListQuery is a protected, explicit deployment extension and reaches the server; pinned Resource.list discards unknown query keys. Go rejects body/foreign-argument options and accepts owned local-filter and per-call request namespaces through WithListFilter and concrete-query collisions; stock Senlin vendor support is not promised.
- Go requires a cluster_policies array, separate valid canonical raw identities and a matching parent, preserving unknown/null/raw JSON fields. Python Resource.list wraps non-array collections, removes self and overlays URI fields on row dictionaries; Go stores fixed URIClusterID separately and uses PolicyID only as the collection identity callback.
- Published filters use concrete typed inputs and grammar-checked sort, while null/omission and exact numbers remain visible in Body rather than being hidden by Python Resource bool/dict descriptor conversions.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count. Unlike Python Resource.list max_items, the binding cap sends no limit hint because the audited stock controller whitelists only enabled/policy_name/policy_type/sort.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go separates concrete server query inputs from explicit WithListFilter raw Body inputs rather than automatically partitioning Python keyword arguments. Query-capable fields can also be selected for local filtering; only known pinned fields and owned namespaces are accepted.
- Raw JSON comparison keeps exact decimal numbers and distinguishes bool from number; object filters match recursive subsets of nonempty objects, while arrays remain ordered complete values. Missing scalar fields and explicit null compare as null without descriptor coercion or cached Resource mutation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python cluster_policies passes cluster_id=Resource._get_id(cluster) without resolution (the URL accepts what the server accepts) and declares sort, policy_name, policy_type and is_enabled (sent as enabled); with an explicit limit its fallback marker is Resource.id, the alternate policy_id. Go resolves the cluster once, requires the canonical binding identity before any filter and rejects an initial limit or marker because the audited server whitelist lacks them.
- Go keeps JSON booleans distinct from numbers in raw Body filters and compares exact decimals; Python compares attribute scalars by equality after descriptor coercion.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
for profile in conn.clustering.profiles(
    limit=20, max_items=50, paginated=True, metadata={"team": "infra"},
):
    print(profile.id)
```

```go
package example

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListProfiles(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    for profile, err := range service.Profiles.List(ctx,
        profiles.WithListOptions(profiles.ListOpts{Limit: 20}),
        profiles.WithListMaxItems(50), profiles.WithListPaginated(true),
        profiles.WithListFilter("metadata", map[string]any{"team": "infra"})) {
        if err != nil { return err }
        fmt.Println(profile.ID)
    }
    return nil
}

func FirstProfilesPage(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    for profile, err := range service.Profiles.Resources.List(ctx,
        resource.WithMaxItems(5), resource.WithPaginated(false)) {
        if err != nil { return err }
        fmt.Println(profile.ID)
    }
    return nil
}

func FirstClusterPoliciesPage(ctx context.Context, conn *sdk.Connection,
    cluster resource.Ref) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    scope, err := service.ClusterPolicies.InCluster(ctx, cluster)
    if err != nil { return err }
    for binding, err := range scope.List(ctx,
        clusterpolicies.WithListMaxItems(5), clusterpolicies.WithListPaginated(false)) {
        if err != nil { return err }
        fmt.Println(binding.ID, binding.PolicyID)
    }
    return nil
}
```

```python
for service in conn.clustering.services(
    microversion="1.7", headers={"X-Trace-ID": "engine-list"}, paginated=False,
):
    print(service.id)
```

```go
package example

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/services"
)

func ListServicesWithCallOptions(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    for engine, err := range service.Services.List(ctx,
        services.WithListMicroversion("1.7"),
        services.WithListHeader("X-Trace-ID", "engine-list"),
        services.WithListPaginated(false)) {
        if err != nil { return err }
        fmt.Println(engine.ID, engine.State)
    }
    return nil
}
```

## References

- [Attribute collection](../clusterattributes/README.md)
- [pinned Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2358)
- [Resource](../../../resource/README.md)
- [List Control Test](../../../internal/rest/list_control_test.go)
- [Actions](../actions/README.md)
- [Events](../events/README.md)
- [Services](../services/README.md)
- [ProfileTypes](../profiletypes/README.md)
- [PolicyTypes](../policytypes/README.md)
- [ClusterPolicies](../clusterpolicies/README.md)
- [pinned Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2248-L2338)
- [Filter](../../../internal/senlin/filter.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)

## Explicit raw Body filters

The eleven typed Senlin List/All APIs accept package-specific WithListFilter for pinned known Body fields. Comparisons use owned JSON snapshots, declared aliases, exact numbers, and recursive object subsets after raw caps and ID validation. Concrete list options select server queries; raw Body keys cannot be forwarded through WithListQuery. Additional Go model fields and URI attributes do not become semantic filters automatically.

## Per-call list headers and versions

Typed List/All APIs accept WithListHeader and WithListMicroversion without mutating the shared client. Options persist across pages while authentication uses the current shared token. Service gates are checked against the selected call version. Authentication, version, and transport headers remain protected, and local control keys cannot become extension queries. Common Resources, parent resolution, and other Get/Wait calls do not accept these typed list options.
