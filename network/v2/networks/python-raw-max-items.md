# Python Raw Max Items

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/networks`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.New`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPaginated`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/network.New`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatus`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.ResourceAdapter`
- `github.com/JSYoo5B/go-openstacksdk/resource.NewCollection`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`
- `github.com/JSYoo5B/go-openstacksdk/network.NewWithDependencies`
- `github.com/JSYoo5B/go-openstacksdk.Connection.Network`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordListConflictingAttrs`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.ProxyCloudListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithCloudRecordListQuery`

- Pinned direct networks (openstack/network/v2/_proxy.py:3258) delegates query to Resource.list. The declared non-query Body attribute subnet_ids remains available through explicit SDK-owned ordinary Resources.List/All filters; it now compares original lowercase subnets JSON rather than only projecting native Network.Subnets. Leaf, cached Connection.API and manual Network.Networks retain native routes, returned models, shared provider and existing manual Kind/ERROR wait policy.
  Tests: `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersNativeValueSemantics`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersLocalWireAndNativeSurfaces`, `internal/cmd/sdkgen/body_filter_collections_test.go:TestBodyFilterCollectionSelectorsMarshalOnlyTheSelectedNativeField`, `api/network_list_filters_test.go:TestNetworkListFiltersRawDescriptorsAndBooleanTruthiness`, `connection_network_filters_test.go:TestConnectionNetworkFiltersSharedFacadesAndRawDescriptors`.
- Complete ordered array values still compare exact length and elements without identifier/CIDR normalization. Missing/null selected fields match null while present empty arrays remain distinct; original null array elements now remain null for comparison although returned native []string elements become empty strings. JSON validation, creation snapshots, independent concurrent reuse, bulk replacement/clear and lazy unsupported/unknown preflight remain SDK-owned.
  Tests: `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersNativeValueSemantics`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersAliasesSnapshotsAndConcurrentReuse`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersLazyPreflightAndExclusions`, `api/network_list_filters_test.go:TestNetworkListFiltersSnapshotsReplacementAndConcurrentReuse`.
- Raw caps count rows before local Body/name/status filters and do not refill with matches or infer a wire limit. Explicit first-page and consumer break preserve native continuation behavior. Native plural_links, whole-current-page decoding including malformed rows beyond the cap, later HTTP failures/cycles and cancellation retain their existing terminal behavior.
  Tests: `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersRawControlsAndConsumerBreak`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersNativePageErrorsAndContinuation`, `api/network_list_filters_test.go:TestNetworkListFiltersControlsAndNativeWholePageFailures`, `api/network_list_filters_test.go:TestNetworkListFiltersPagingLiveSourceAndNativeStatus`.
- Raw subnet_ids/subnets/prefixes/status/vendor query remains independently frozen wire data without automatic alias conversion or Body classification. Native typed List and FindIdentity do not acquire the ordinary-list Body option. Trunk sub_ports remains query-mapped and is not registered as a local Body filter.
  Tests: `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersLocalWireAndNativeSurfaces`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersLazyPreflightAndExclusions`, `api/network_list_filters_test.go:TestNetworkListFiltersNativeAndIdentitySurfaceIsolation`.
- Audited native model/query-builder/result/page/extractor/decoder shapes and receiver-qualified declaration hashes are still required. The expanded Network filter lane pairs native whole-page extraction with original row fields; inventory distinguishes canonical raw Body fields from semantic Python property names. No unsupported properties or unrelated bindings are advertised, and existing SubnetPool/other filters remain independent.
  Tests: `internal/cmd/sdkgen/body_filter_collections_test.go:TestBodyFilterCollectionGateRejectsNativeShapeAndUnauditedFields`, `internal/cmd/sdkgen/body_filter_collections_test.go:TestBodyFilterCollectionStillRequiresPinnedNativeContracts`, `internal/cmd/sdkgen/body_filter_collections_test.go:TestBodyFilterCollectionInventoryHasOnlyOwnedCanonicalFields`, `internal/cmd/sdkgen/network_filters_test.go:TestNetworkBodyFilterNativeContractRejectsModelBuilderDecoderAndSourceDrift`, `internal/cmd/sdkgen/network_filters_test.go:TestNetworkBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods`.
- The Python attribute subnet_ids and Go canonical subnets refer to one original lowercase subnets field for comparison; the returned native Subnets projection is unchanged. Sequential canonical/alias options use canonical last-wins; bulk containing both names fails deterministically before HTTP even if values agree. Network local name/status conditions compose with the Body predicate on both generated controlled and manual native-pager paths. Native old-format and RFC3339 timestamps retain their decoder behavior.
  Tests: `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersAliasesSnapshotsAndConcurrentReuse`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersRawControlsAndConsumerBreak`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersNativePageErrorsAndContinuation`, `api/network_list_filters_test.go:TestNetworkListFiltersSnapshotsReplacementAndConcurrentReuse`, `api/network_list_filters_test.go:TestNetworkListFiltersLazyPreflightAndNamespaceCollisions`, `connection_network_filters_test.go:TestConnectionNetworkFiltersPreflightAndLocalStatus`.
- SDK-owned WithFilter/WithFilters classify all 23 canonical QueryParameters attributes and 35 accepted canonical/wire spellings, including inherited limit/marker/tag aliases, without application builders. Query name/status/id are server-only; flags, provider aliases and address-scope aliases transpose to their exact wire keys. Semantic tenant_id and raw Body wire aliases are unknown and discarded, with no project/tenant inference.
  Tests: `api/network_list_filters_test.go:TestNetworkListFiltersEntireQueryAndAliasDescriptor`, `api/network_list_filters_test.go:TestNetworkListFiltersLazyPreflightAndNamespaceCollisions`, `api/network_list_filters_test.go:TestNetworkListFiltersNativeAndIdentitySurfaceIsolation`, `internal/cmd/sdkgen/network_filters_test.go:TestNetworkPythonFilterManifestKeepsQueryAliasesLocalTypesAndNetworkResourceMRO`.
- A semantic bulk map selects canonical query keys by presence ahead of wire aliases, including null/false/empty values; individual options are last-wins. Query scalars and scalar arrays retain exact number text, lowercase booleans and repeated values, skipping null array elements. Nil/empty arrays omit wire values but retain key presence. Maps/nested arrays and nine reserved source controls fail lazily; unknown semantic values are discarded before validation. Semantic/raw-query/page-size/name/status and semantic/explicit-Body collisions reject before HTTP, even when equal.
  Tests: `api/network_list_filters_test.go:TestNetworkListFiltersEntireQueryAndAliasDescriptor`, `api/network_list_filters_test.go:TestNetworkListFiltersLazyPreflightAndNamespaceCollisions`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`, `resource/filters_test.go:TestSemanticFiltersValidateOnlyFinalSelectedValues`.
- The complete 14 non-query Body properties use four response-only booleans, two exact-integer responses and eight original-JSON fields, including inherited revision_number and extension fields absent from the native model. All local descriptor defaults are None, so missing/null bools remain null rather than false; query-mapped security/external False defaults do not become local predicates. Raw original timestamps, nested large numbers and array-null elements are compared after native decoding, without changing the returned native Network model.
  Tests: `api/network_list_filters_test.go:TestNetworkListFiltersRawDescriptorsAndBooleanTruthiness`, `api/network_list_filters_test.go:TestNetworkListFiltersControlsAndNativeWholePageFailures`.
- Response-only boolean truthiness keeps null and maps exact numeric zero and empty JSON strings/arrays/objects to false; nonzero numbers and nonempty values become true, including the string false. Caller types remain unchanged, so bool, number and string conditions stay distinct. Integer responses accept only exact integral numbers or decimal integer strings; selected fractional/bool/container values fail with cause. Only consumed selected fields are normalized; raw caps/break can avoid unconsumed normalization while native known-field errors on the whole current page still precede filtering.
  Tests: `api/network_list_filters_test.go:TestNetworkListFiltersRawDescriptorsAndBooleanTruthiness`, `api/network_list_filters_test.go:TestNetworkListFiltersControlsAndNativeWholePageFailures`, `internal/jsonfilter/boolean_test.go:TestBooleanJSONResponseTruthinessAndExactNumericZero`, `internal/jsonfilter/boolean_test.go:TestBooleanJSONOwnsBytesAndLeavesCallerEqualityUnchanged`, `resource/body_record_test.go:TestBodyRecordFieldNormalizationPresenceAndCauses`, `resource/body_record_boolean_test.go:TestBodyRecordBooleanPresenceOwnershipAndErrorCauses`, `resource/body_record_boolean_test.go:TestBodyRecordBooleanKeepsJSONAndIntegerDescriptorContracts`, `internal/jsonfilter/boolean_test.go:TestBooleanJSONRejectsMalformedAndMultipleValuesWithCauses`.
- Semantic options snapshot maps/slices/raw JSON and can be reused concurrently; bulk replacement clears only semantic state and can supersede earlier invalid selected values. Explicit Body state and raw query remain independent. Raw WithQuery status is preserved in both lanes, WithStatus retains its native local comparison, including the final last-wins status value supplied by a later raw WithQuery, and typed API.List/FindIdentity use their existing raw query/native extraction instead of acquiring semantic list options. Live token, configured source headers/microversion, middleware, context and native foreign continuation remain intact; All never returns a partially collected slice with a terminal page error. ResourceAdapter returns fresh SDK-owned metadata and NewCollection clones it; mutating a separate adapter does not alter any existing facade. Manual singular network errors and case-folded exact ERROR failure checks remain distinct from the generated networks/ERROR-prefix waiter.
  Tests: `api/network_list_filters_test.go:TestNetworkListFiltersSnapshotsReplacementAndConcurrentReuse`, `api/network_list_filters_test.go:TestNetworkListFiltersPagingLiveSourceAndNativeStatus`, `api/network_list_filters_test.go:TestNetworkListFiltersNativeAndIdentitySurfaceIsolation`, `connection_network_filters_test.go:TestConnectionNetworkFiltersAdapterOwnershipAndWaitPolicies`, `internal/cmd/sdkgen/network_filters_test.go:TestNetworkSemanticFilterEmissionUsesFreshResourceAdapterAndPreservesPriorSevenBindings`.
- The checked-in Network semantic descriptor is verified against seven live pinned Python file SHAs and 43 independent AST anchors covering exact C3 inheritance, all effective Body declarations/defaults, QueryParameters aliases/unknown-discard/precedence, bool conversion/null shortcut, integer/list conversion and direct Proxy controls. Generation requires fresh AST extraction to match the manifest and audited native raw-row/source contracts; full-file hashes do not themselves implement inherited Resource/cache/pagination behavior. Native checks cover 28 receiver-qualified declarations plus the NoZ timestamp layout constant, full model/list-option/result/page/root dependency shapes and exact receiver/signature policy; Network has no resourcePath constant to guard. Fresh ResourceAdapter emission retains generic adapters for unrelated resources.
  Tests: `api/network_list_filters_test.go:TestNetworkListFiltersEntireQueryAndAliasDescriptor`, `internal/cmd/sdkgen/network_filters_test.go:TestNetworkPythonFilterPinnedSourceRequiresBooleanDefaultsAndCompleteClassificationProof`, `internal/cmd/sdkgen/network_filters_test.go:TestNetworkRawDependencyLoaderPreservesNativeTimeCodesAndNumbers`.
- New Source-shaped Network ListRecords/AllRecords uses the existing 29-field effective Body and 23-query inventory through a cloned private neutronfind profile; no implicit name hint, member fetch, native typed decoder, or application builder is introduced. GET is bodyless at the selected service route; actual page JSON/Header/status remains independent of canonical Resource and original Wire. AllRecords returns a nonnil empty slice on success and discards partial values on terminal errors.
  Tests: `network/v2/networks/python_list_records_test.go:TestPythonNetworkListRecordsDefaultRequestAndExtensionFields`, `network/v2/networks/python_list_records_test.go:TestPythonNetworkListRecordsQueryAliasesAndLocalBodyFilters`, `network/v2/networks/python_list_records_test.go:TestPythonNetworkListRecordsStatusesAndOperationContext`.
- The complete 29 Body defaults plus if_match/location are projected: missing port-security/external booleans false, tags empty array, other declarations null; explicit null overrides defaults. Ordered canonical/wire response aliases are last-value-wins. Raw unknown extensions, timestamps and large integers survive; Network tenant_id has no project_id alias. Row Header canonical case/wire-insensitive spelling is distinct from whole-page HTTP Header. self is removed from Resource and remains in Wire; spoofed location remains actual Wire while the owned view is null.
  Tests: `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsEntireBodyDefaultsAndOrderedAliases`, `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsDeclaredCoercionUnknownBodyAndPrivateEvidence`.
- All 23 canonical query attributes and 35 accepted canonical/wire spellings are classified from the pinned profile. Canonical values win by presence, including null/false/zero; query bools use True/False and lists preserve Requests repeat/null omission. fields remain literal list strings. Body-only/header/unknown wire spellings cannot become undeclared server filters or implicit project/tenant scope.
  Tests: `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsAllQueryNamesWireAliasesAndCanonicalPresence`.
- The full 14 non-query Body descriptor inventory filters locally after owned projection, including missing/null, bool/int coercion, array wrapping and untyped dictionary subset. Predicates retain insertion order and short-circuit; actual mapping-get failure includes the original whole page. Large unknown integers retain original raw evidence and list-row callers do not seed missing query values.
  Tests: `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsCompleteLocalDescriptorInventory`, `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsLocalOrderShortCircuitAndDictionaryFailureProof`.
- Paging reuses inherited links/plural links/HTTP Link/limit-marker fallback and no automatic member lookup. Positive cap hints limit and counts raw excluded rows; consumer break and single-page controls avoid continuation and out-of-range constructor faults. Late errors discard AllRecords slices while iterators retain previously yielded rows. JSON envelope, UTF8, null-row/constructor/descriptor failures keep actual page status/header/body/Cause without creating caller-error or success receipts.
  Tests: `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsContinuationRawCapsBreakAndAtomicAll`, `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsMalformedRowsConstructorsAndActualPage`.
- Owned factories, DTO maps/raw bytes/version pointers and option callbacks are snapshotted. Initial and callback-reset nil maps are initialized, retained callback maps cannot alter the eventual request, and options are concurrently reusable. Invocation is lazy; selected source/route/version guard surrounds callbacks and cancellation retains custom causes before HTTP.
  Tests: `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsPreflightControlPresenceAndUnknownDiscard`, `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsOwnedOptionsNilMapsLazinessAndConcurrentReuse`, `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsSourceMutationAndCanceledOptionAreSticky`.
- Accepted and rejected physical read/Close/cancellation/source faults do not call native RetryFunc or ReauthFunc, use one attempt/one Close, and retain all causes. Native pointer ErrUnableToReauthenticate keeps actual method/URL/status/body/header in ErrOriginal and fault/cancel/source causes in ErrReauth. Clean503 retry and401 token renewal remain allowed; body/protocol/version/auth/conditional mutations and expanded final acceptance never fabricate success.
  Tests: `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsPhysicalFaultsPreserveNative401AndNeverInvokeCallbacks`, `network/v2/networks/python_list_records_contracts_test.go:TestPythonNetworkListRecordsCleanRetryReauthenticationAndForbiddenMutation`.
- ProxyListRecords evaluates deprecated truthy-string JMESPath against the actual opaque lazy generator rather than a collected array: projection/index/type null, literal arbitrary JSON and runtime errors all use zero HTTP; @/not_null return the generator and to_array/hash can contain that same object. Opaque identity equality, nested equality and contains preserve object identity; JSON exists only for fully serializable output, and to_string/generator marshal rejects without consumption. Generator iteration starts real list HTTP.
  Tests: `network/v2/networks/python_list_records_proxy_test.go:TestPythonNetworkProxyListRecordsJSONResultsDoNotReadGenerator`, `network/v2/networks/python_list_records_proxy_test.go:TestPythonNetworkProxyListRecordsJMESRuntimeFailuresBeforeHTTP`, `network/v2/networks/python_list_records_proxy_test.go:TestPythonNetworkProxyListRecordsGeneratorIdentityAndNestedResultsStayLazy`, `network/v2/networks/python_list_records_proxy_test.go:TestPythonNetworkListRecordsFalseyAndNonstringJMESAreIgnored`.
- NormalizeProxyListOptions separates the original Proxy expression before one ordered __conflicting_attrs restore. Falsey conflict values are ignored; truthy nonobjects and restored bound arguments fail before HTTP. Original key positions and appended member order remain stable, self-restoration is not recursive, and restored jmespath_filters is discarded as Resource.list unknown attrs instead of replacing the original expression. Header/version/max_items restored formals are consumed by the SDK; allow_unknown_params is ignored because Source always validates with allow_unknown_params=True.
  Tests: `network/v2/networks/python_list_records_controls_test.go:TestPythonNetworkListRecordsConflictingAttrsFormalOverridesAndOriginalJMES`, `network/v2/networks/python_list_records_controls_test.go:TestPythonNetworkListRecordsUnknownConstructorNamesDiscardBeforeMarshal`.
- Source named headers/microversion/base_path and concrete factories use owned final values, null/default semantics, cloned route/version state and fixed request metadata. Base-path continuation locks to the selected collection; foreign/query/fragment/traversal paths fail before HTTP. The new Source profile opts into CPython loaded equality for rounded floats/underflow/overflow/nested bool-number comparisons, retaining arbitrary precision integers and unchanged literal Resource/Wire/Envelope. Existing profile zero values retain old exact-decimal comparison.
  Tests: `network/v2/networks/python_list_records_controls_test.go:TestPythonNetworkListRecordsBasePathAndNamedHeadersVersionsUseOwnedFinalValues`, `network/v2/networks/python_list_records_controls_test.go:TestPythonNetworkListRecordsUsesLoadedCPythonFloatEqualityWithoutChangingRawEvidence`.
- Shared Source list formal normalization independently verifies original JMES separation, one-pass ordered restoration, bound argument presence failures, falsey values, ignored unknown controls without parsing, concrete header string maps, nullable version/path and complete snapshot ownership. This closes Source kwargs through a common library contract rather than new per-service decoder/builder responsibilities.
  Tests: `internal/neutronfind/proxy_list_options_test.go:TestNormalizeProxyListOptionsPreservesOriginalExpressionAndOrderedOnePassMerge`, `internal/neutronfind/proxy_list_options_test.go:TestNormalizeProxyListOptionsFalseyConflictAndUnusedFormalIgnoreValues`, `internal/neutronfind/proxy_list_options_test.go:TestNormalizeProxyListOptionsRejectsOnlyBoundAndRestoredDuplicateArguments`, `internal/neutronfind/proxy_list_options_test.go:TestNormalizeProxyListOptionsHeaderFalseyStringMapAndNullableVersionPath`, `internal/neutronfind/proxy_list_options_test.go:TestNormalizeProxyListOptionsJMESNonstringsAreReturnedWithoutInventingQueries`.
- New connected Source lane uses cloned SourceStatusPolicy/SourceLinkPolicy and the complete existing Network descriptor/query profile. Original row/full page/header/status proof remains separate from projected defaults, large integers, Python-loaded local bool/number equality and owned connected location. Unknown availability_zone is read as the Source runtime attribute, while tenant_id is not inferred as Network.project_id.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListRawControlsLocalLoadedEqualityPagingAndReceipts`, `network/cloud_network_lists_test.go:TestCloudNetworkListRawMicroversionTransportAndUnknownLocationZone`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListSourceStatusDefaultsAndConnectedLocation`.
- Original bound self/resource_type fail during Proxy call; one-pass ordered conflicting attrs then reject restored base_path/paginated/cls/session collisions but ignore restored self/resource_type and restored JMES spelling. Raw headers/base_path/microversion are evaluated only when the Resource generator is consumed. Explicit empty base path returns empty with no HTTP but still checks iteration-start cancellation/source guard.
  Tests: `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListRawFormalsRemainLazyAcrossProxyPhases`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListOriginalAndRestoredBindingHaveDifferentPhases`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListEmptyPathSkipsHTTPAndKeepsIterationGuards`, `network/cloud_network_lists_test.go:TestCloudNetworkListOpaqueJMESListConversionAndDeferredFormals`.
- Opaque Proxy JMES values retain Source generator identity and HTTP0 timing; @ is consumed by cloud list(), array/string/object literals become iterable values, projection null/noniterable results and type errors fail without HTTP. Nested generator values retain laziness/location/source and no fabricated JSON/HTTP receipts. Go object keys iterate lexical order rather than Source insertion order, asserted explicitly.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListOpaqueJMESListConversionAndDeferredFormals`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListOpaqueProxyStaysLazyWithOwnedLocation`.
- Raw JSON microversion falsey nonnull values suppress headers, null uses selected version, numeric/bool and iterable tuple values follow the separately audited Keystoneauth JSON-domain normalization. Concrete Headers/Microversion/Location options and raw filters snapshot caller inputs before guarded callback/response use; original selected source and live token are preserved.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListRawMicroversionTransportAndUnknownLocationZone`, `network/cloud_network_lists_test.go:TestCloudNetworkListLateFailureIsAtomicAndOptionsOwnSource`.
- Cloud collection discards partial public outputs on late failure while actual page rejection/physical proof is retained. Accepted200 and rejected401/404/503 read+Close/cancel/source lanes close physical bodies once, make no retry/reauth calls and preserve actual status/body/header. Native401 independent ErrOriginal value and ErrReauth causes remain available; clean retry/token renewal are guarded and bounded in fixtures.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListLateFailureIsAtomicAndOptionsOwnSource`, `network/cloud_network_lists_test.go:TestCloudNetworkListPhysicalFaultsKeepActualReceiptAndNoNativeHooks`, `network/cloud_network_lists_test.go:TestCloudNetworkListCleanRetryTokenRenewalAndGuardedRetry`.
- The explicit Source-shaped list lane now accepts JSON falsey, true, negative and fractional max_items values, counts raw rows before local filtering, and defers truthy nonnumeric comparison errors until a nonempty raw row; falsey/container and raw limit hints preserve their separate wire forms.
  Tests: `network/v2/networks/python_raw_max_items_test.go:TestPythonNetworkRawMaxItemsJSONDomainAndDeferredComparison`, `network/v2/networks/python_raw_max_items_test.go:TestPythonNetworkRawMaxItemsFalseyAndRawLimitHint`, `network/v2/networks/python_raw_max_items_test.go:TestPythonNetworkRawMaxItemsOpaqueJMESAndOwnedJSON`.
- The cap is checked before each next raw-row constructor. Exact-page cap boundaries may fetch a following page and expose its HTTP/read/Close/context failure; malformed rows beyond the cap are ignored and filtered rows count without refill. Legacy Find/native/typed pager policies remain separate.
  Tests: `network/v2/networks/python_raw_max_items_test.go:TestPythonNetworkRawMaxItemsExactPageBoundaryObservesNextResponse`, `network/v2/networks/python_raw_max_items_test.go:TestPythonNetworkRawMaxItemsCountsFilteredRowsAndStopsBeforeConstructor`.
- ProxyListRecords (Proxy.networks) and ProxyCloudListRecords: Raw limit values reach the wire with Requests spelling (False, -1, 1.5, x, repeated values) and Resource.list keeps the original limit for its loop: only a Python-truthy limit performs the marker GET after a nonempty page without a next link, so false/0/0.0/""/null send one GET. Integer max_items equal to one whole page is compared before the next raw row, so the marker GET is performed and its failure is observed; raw true/negative/fractional caps follow the same before-row comparison. base_path="" never enters Resource.list's while loop: no HTTP and no KSA microversion normalization, so an invalid raw microversion is unobserved and the result is empty; a nonempty route still rejects it before HTTP.
  Tests: `network/v2/networks/python_source_list_limits_test.go:TestPythonNetworkProxyRawLimitSpellingAndTruthiness`, `network/v2/networks/python_source_list_limits_test.go:TestPythonNetworkProxyCapEqualsWholePage`, `network/v2/networks/python_source_list_limits_test.go:TestPythonNetworkProxyEmptyBasePathSkipsVersionNormalization`.

Python/Go differences:

- Go uses concrete WithFilter/WithFilters and explicit WithBodyFilter options with frozen values instead of Python keyword arguments and mutable Resource objects. Semantic names follow Python properties; raw subnets/vlan_qinq/vlan_transparent are accepted only by explicit Body filters or raw wire query. Body bulk alias collisions reject; semantic bulk query canonical keys win by presence and sequential options are last-wins. Namespace conflicts reject deterministically.
- Local comparisons use original row JSON after native whole-page decoding, retaining selected large numbers, timestamp spelling and null array elements that native return projections normalize or omit. Returned models remain native. Query-only null rows retain native zero models; consumed null rows with Body conditions reject, while empty object rows are valid.
- Boolean response normalization follows JSON truthiness and preserves missing/null instead of false. Go exact decimals preserve tiny nonzero truthiness where Python float parsing can underflow. Caller filters are not coerced: Go distinguishes bool/number and string/number equality. Exact-integral response policy rejects fractions/bools instead of Python integer truncation/bool-as-int, and accepts signed decimal integer strings rather than digit-only/zero fallback.
- Python list descriptors wrap non-list values; Go selected raw JSON does not wrap. Arrays compare complete ordered values, standalone dictionaries use recursive subset matching on nonempty objects, and unknown extension fields do not become returned native model fields. Exact timestamp text differs even when instants are equivalent.
- Known query scalar arrays become repeated wire values without Python formatting arbitrary containers. Nil/empty arrays omit URL values but preserve key presence. Unknown semantic names are discarded, while raw WithQuery preserves caller wire names/extensions/repeated data and does not classify aliases or Body fields.
- max_items caps raw rows locally without Python Resource.list automatic limit hints. Native accepted 200/204/300, JSON-empty-204 EOF, full-page typed extraction and advertised networks_links/foreign following are retained. Explicit first page and early break avoid continuation; typed native List, FindIdentity and other CRUD are separate surfaces.
- New record API is a sibling to existing typed List/All and semantic Collection APIs. It implements Source owned descriptors, whole-page/private receipts, paging, per-call controls and actual Proxy generator JMES results; prior exact-decimal/native typed-detection differences describe those existing surfaces and do not override the new record contracts.
- Record JSON is immutable owned view rather than lazy mutable Python Resource. Every declared field is projected before row return, so an unaccessed malformed descriptor can fail earlier. Unknown Body extensions are preserved; location is still owned null rather than Python current_location or computed constructor synthesis. Shared SDK Resource/cache/lifecycle parity remains independently tracked.
- Source complete kwargs behavior is implemented in the JSON domain with concrete header string-map and microversion/base-path string types. None-valued headers/adapter/subclass injection, runtime non-JSON objects and arbitrary URI percent-format are not reintroduced. Default microversion has no Network class maximum, and unknown LIST skip_cache is ignored exactly rather than inventing a per-call cache override.
- Go collection continuation retains fixed service origin/path/filter/cursor guards. Per-call base_path selects a service-relative collection; typed ListRecords/AllRecords choose the default collection for an empty path, while ProxyListRecords and ProxyCloudListRecords follow Source and make no request. Source arbitrary adapter retargeting and informational final100..199 remain explicit differences. Original selected clients, live tokens and approved clean retries retain source ownership.
- Typed ListRecords/AllRecords yield resources only; truthy-string JMES calls use ProxyListRecords with arbitrary Value and optional JSON. Fully JSON null is JSON null, generator-containing values keep their opaque generator identity and no fabricated Wire/HTTP metadata. This mirrors the pinned generator behavior, which differs from cloud expression filtering over collected arrays.
- Source general numeric equality is opt-in through PythonLoadedEquality for this new profile: float tokens use CPython binary64 rounding/nonfinite overflow/underflow, integer tokens remain arbitrary precision, nested bool-number equality is preserved. Raw actual bytes are not rewritten. The rest of the repository is not silently migrated.
- Neutron27 stock get_network allows project-reader/service/shared/external/advsvc branches with separate ADMIN provider/segments attributes. SDK does not elevate access or infer all-project scope; missing user attributes retain declared defaults. Fixture/race/source/main evidence does not prove authenticated deployment policy/extension/backend behavior.
- Earlier typed/ProxyListRecords lane only: Record JSON is immutable owned view rather than lazy mutable Python Resource. Every declared field is projected before row return, so an unaccessed malformed descriptor can fail earlier. Unknown Body extensions are preserved; location is still owned null rather than Python current_location or computed constructor synthesis. Shared SDK Resource/cache/lifecycle parity remains independently tracked.
- Earlier typed/ProxyListRecords lane only: The earlier typed lane has concrete header string-map and microversion/base-path string types; the new raw Source cloud lane accepts audited numeric/iterable microversion and deferred formals. None-valued headers/adapter/subclass injection, runtime non-JSON objects and arbitrary URI percent-format are not reintroduced. Default microversion has no Network class maximum, and unknown LIST skip_cache is ignored exactly rather than inventing a per-call cache override.
- Earlier typed/ProxyListRecords lane only: Go collection continuation retains fixed service origin/path/filter/cursor guards. Per-call base_path selects a service-relative collection; typed ListRecords/AllRecords choose the default collection for an empty path, while ProxyListRecords and ProxyCloudListRecords follow Source and make no request. Source arbitrary adapter retargeting and informational final100..199 remain explicit differences. Original selected clients, live tokens and approved clean retries retain source ownership.
- SDK-R1: Go returns owned RawResource/receipt views instead of a mutable Python Resource and cache/lifecycle object. Declared descriptors are projected before row return and may fail earlier for an unaccessed malformed property. Actual Wire/Envelope/header/status are never forged from normalized values; runtime Resource.microversion attribute/object identity remain generic Resource parity scope.
- SDK-C1: JSON bytes retain exact integers and raw proof; local comparisons use audited CPython-loaded float/bool/int policy. Non-JSON custom objects, nonfinite JSON runtime values and exact unrestricted Python representation/Unicode/marshal behavior remain independent generic contracts. Go wraps structured errors while preserving actual response and physical/context causes.
- SDK-S1: source origin, service-relative path/filter/continuation guards and concrete transport options retain the original selected client. Arbitrary adapter/class injection, URI percent-mapping retargeting, foreign/query/fragment/traversal base paths are restricted; existing typed/legacy list APIs retain their baseline rather than receiving hidden Source flags.
- New ProxyCloudListRecords raw WithCloudRecordListQuery preserves Resource formal validation timing and original/restored binding behavior. It is the Source-connected lane; older WithRecordListQuery/ProxyListRecords keep the prior eager concrete control contract. Informational final100..199 and Source continuation-link policy apply only to the new cloned profile.
- Source cloud list() turns dict results into insertion-ordered keys; Go uses lexical keys and has an explicit z,a→a,z assertion. Opaque nested generators remain in Values with JSON=nil; ordinary outputs have Networks/Values/JSON. Typed-only callers can select Networks while arbitrary JMES callers inspect Values.
- Neutron27 stock get_network project-reader/service/shared/external/advsvc paths do not guarantee ADMIN-only provider/segments attributes or all-project visibility. SDK injects current-location facts but does not elevate access. Actual pinned Source/AST/fixture/race/build receipts and live authenticated deployment evidence remain distinct.
- Raw limit spelling and Python truthiness, raw max_items caps and the cap-equals-page marker GET are implemented on the Source-shaped Proxy lanes (shared opt-in Source list controls); typed ListRecords/AllRecords and the semantic Collection lanes keep strict Go integer limit/max_items preflight.
- Raw max_items behavior and exact-boundary cap timing are now owned only by the explicit Source-shaped list path. Other named nonfinite microversion and global runtime/lifecycle contracts remain unresolved; historical typed/native contracts above retain their original boundaries.

Unresolved boundaries:

- Raw JSON microversion exponent overflow remains a precise named runtime boundary: actual CPython json.loads + runtime KSA normalization accepts positive-overflow 1e400/[1e400] as network latest, while the current Go finite-number conversion rejects it on a nonempty route. KSA-only12 header-normalizer evidence is separately recorded; it is not claimed as pinned Source physical HTTP proof.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--max-items-json", default="null")
args = parser.parse_args()
cap = json.loads(args.max_items_json)
with openstack.connect(cloud=args.cloud) as conn:
    for row in conn.network.networks(limit=10, max_items=cap):
        print(json.dumps(row.to_dict(), sort_keys=True))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    capJSON := flag.String("max-items-json", "null", "raw JSON max_items")
    flag.Parse()
    raw := json.RawMessage(*capJSON)
    if !json.Valid(raw) { log.Fatal("max-items-json must be complete JSON") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []networks.RecordListOption{
        networks.WithCloudRecordListQuery("max_items", raw),
        networks.WithRecordListLimit(10),
    }
    proxy, err := network.Networks.ProxyCloudListRecords(ctx, nil, options...)
    if err != nil { log.Fatal(err) }
    generator, ok := proxy.Value.(*networks.RecordListGenerator)
    if !ok { log.Fatal("expected a Resource list generator") }
    for row, err := range generator.Records {
        if err != nil { log.Fatal(err) }
        encoded, err := json.Marshal(row.Resource)
        if err != nil { log.Fatal(err) }
        fmt.Println(string(encoded))
    }
}
```

## References

- [Install](../../../docs/install.md)
