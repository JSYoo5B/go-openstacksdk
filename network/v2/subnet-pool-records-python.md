# Subnet Pool Records Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_subnet_pool`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordFindLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordFindMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordFindMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordFindPaginated`

- Pinned direct find_subnet_pool (openstack/network/v2/_proxy.py:7621) accepts a string, ignore_missing=True and query. Native GET200 /subnetpools/id and subnetpool envelope preserve canonical IDs and typed nested model/timestamp extraction; undeclared GET203/204 and accepted decode errors remain terminal without fallback or resend.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksGETModelsAndNativeCodes`.
- Compatible GET400/403/404 may fall back, with explicit 404-only/GET-only and strict options. Logical absence defaults to nil,nil after successful complete native absence, while LIST403/404 and later HTTP/decode/cycle/cancellation remain terminal. Exact ID OR name matches include repeated-ID and cross-project ambiguity over all native continuations.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksFallbackQueryAndMissingPolicies`, `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksAllPagesAndTerminalObservations`.
- Caller raw query, including status/fields/tags/vendor/project filters, is frozen and retained in both phases with repetitions, explicit nil/empty name precedence and concurrent reuse. Only an absent caller name gets a literal LIST hint; no project scope is inferred and raw status alone never becomes a local response filter.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksFallbackQueryAndMissingPolicies`, `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksQuerySnapshotsAndConcurrentReuse`.
- Leaf and cached Connection.Network.API retain the original provider/live token, configured headers/version, HTTP middleware, context and ResourceBase. Unsafe names are literal list-only; nil/canceled context, GET-only unsafe names and explicit Details/AllProjects/ExtraSpecs true or false fail before HTTP.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksLiveSourceAndCancellation`, `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksUnsafeNamesAndCapabilityPreflight`.
- Ordinary Resources.List/All preserve full raw wire query with native page extraction and local cap-before-name/first-page controls, without adding a wire limit or identity lookup. Existing explicit ID/Name Ref lookups and the native typed List API remain separate.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksOrdinaryCollectionsQueriesAndControls`.
- Native SubnetPool requires all default_prefixlen/min_prefixlen/max_prefixlen values and decodes string or number representations into native int fields. Missing/null/bool or malformed timestamps remain accepted decode errors without lookup fallback. Native old and RFC3339 timestamp forms, prefixes/IPVersion/default flag, and subnetpools_links rel=next are preserved; top-level links.next alone is ignored.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksGETModelsAndNativeCodes`, `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksAllPagesAndTerminalObservations`, `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksUnsafeNamesAndCapabilityPreflight`.
- Update sends only raw supplied attrs in subnetpool envelope, resolves canonical/wire aliases in option order, preserves null/false/zero/unknown attrs and sparse member seed while retaining independent literal Wire/Envelope/Header/status. No attrs is a no-HTTP commit, tags default to [] and missing Body fields become null; this Resource has no NetworkResource conditional Header descriptors.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsUpdateAliasesNoopAndSeedWireProjection`, `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsMemberDescriptorsSparseAndMalformedEvidence`.
- Member kwargs remain original query plus constructor seed; fallback/direct lists transpose only Source declared query aliases, canonical keys override wire aliases even when null, known nonserver canonical Body attrs filter descriptor views, unknown list attrs are discarded, and all_projects is consumed without granting access. fields list values remain literal, tags and repeated query values preserve Requests JSON-domain representation. LIST Accept defaults override source Accept while explicit Headers prevail.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsQueryClassificationAliasesAndLocalDescriptors`.
- Lazy owned full pagination supports body/header continuation, marker fallback on short pages, raw-row caps before local filters and single-page choice. Find checks all candidate rows/pages before returning one exact ID/name; late duplicates/rejected/malformed pages remain errors. Cycles and conflicting/foreign continuation are terminal; standard dictionary next links are a declared Go improvement.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsPaginationFullSearchAndRawCaps`.
- Only clean native400/403/404 member rejection falls back; default missing requires a completely successful search and strict missing retains no suppressed native rejection cause. Nonobject row/envelope and constructor cls/connection/microversion/_synchronized collisions retain actual page proof. Source response descriptors retain integer/bool/list/null/tag/tenant aliases; malformed accepted member syntax follows seed while valid wrong shapes fail.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsCleanMissingAndConstructorFailures`, `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsMemberDescriptorsSparseAndMalformedEvidence`.
- Accepted/rejected Read+Close/cancel/source-changed-and-restored-on-Close failures preserve every cause and physical body/header/status, close once, stop before fallback/missing and never resend. Member result receipts for accepted physical failures hold actual metadata without a fabricated descriptor acknowledgement.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsPhysicalFailuresPreserveAcceptedAndRejectedProof`.
- Native retry cannot change/delete/alias/omit selected network version, inject version when absent, add foreign-service/auth/framing headers or mutate JSON body/Content-Type. One503 proof plus native callback causes survives; ordinary retry headers, exact method/URL/JSON, live token and reauthentication are retained.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsRetriesKeepProtocolBodyAndNativeCauses`.
- Source is captured before callbacks; concrete options snapshot raw JSON/maps/pointers and retained callbacks cannot mutate requests/results. Invalid controls/headers/IDs/descriptors/context/options fail before HTTP and preserve causes. One immutable option supports concurrent requests with independent records/evidence.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsOptionsSourceContextAndConcurrentOwnership`.
- Source Resource.find consumes the explicit microversion for member fetch only; fallback LIST and all its pages keep the captured original selected/default microversion. Ordinary Header options apply to both phases. Fallback retry cannot inject the member-only override when selected version differs or is absent; one original LIST503 body/header/status survives. Direct List/Update per-operation overrides remain effective and never mutate source.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsFindMicroversionIsMemberPhaseOnly`.
- Library-owned PagePolicy.NextTruthy reuses SubnetPool response binary64 truthiness: top-level next1e-400/-1e-400/0.0 are falsey and do not request another page, while truthy numeric1e400/1.0 fail URL decoding with exact whole actual page body/header/status. Normal bool/null next remain falsey; independently yielded row receipts retain their own page snapshots.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsNextUsesBinary64TruthinessAndWholePageProof`.

Python/Go differences:

- SDK-owned concrete identity options preserve missing/default/fallback policies and query ownership. Raw wire aliases and unknown keys are explicit Go extensions; Python query transposition/local Body classification is not automatic. No hidden/default/enrichment/parent or microversion-discovery policy is added.
- Unsafe names are list-only, GET-only rejects them before HTTP and consumed models require safe nonblank IDs. Native accepted statuses, model/extractor and advertised native continuation semantics remain intact. Project scope is caller-selected rather than inferred.
- The audited ordinary common-list bridge preserves repeated/empty raw query, including status and extensions, instead of collapsing values or silently deleting status. Native typed List options and explicit Ref semantics are retained.
- Native SubnetPool decoding requires three prefix lengths even when a caller fields projection requests only id/name. Such a projected response fails instead of providing a permissive partially seeded Python Resource. Typed WithStatus is unsupported because the native model has no Status; raw status remains wire-only.
- Bounded named mappings cover literal identity/name and JSON-domain attrs, not arbitrary Python mutable Resource instances, Connection/session/cache side effects, resource location calculation or deprecation warnings. location is null; whole Source Resource/Connection parity remains separately tracked.
- Go rejects SDK constructor/control attr collisions rather than allowing caller retargeting, and supplies per-call owned Headers/Microversion and typed list-only pagination conveniences. Raw unknown Body/Find kwargs remain data; unsupported list kwargs are discarded.
- Bulk Go maps are sorted when conflicting canonical/wire alias insertion order matters; individual mutation options preserve order. Local map filter evaluation is sorted. Shared JSON query float repr/Unicode runtime and CPython int-string-limit differences remain explicit common representation bounds.
- Owned pager additionally follows standard links:{next:URL}, which pinned Source dict-to-item loop ignores, and rejects ambiguous/foreign/cyclic links with physical proof. Raw caps, marker-on-short-page and complete find search match Source supported behavior.
- Fixed Neutron27.0.0 policy permits admin/owning-project reader or shared-pool reads and admin/owning-project member updates; is_default update is separately admin-only. Examples omit that admin attr and all implicit defaults. No explicit service-only branch exists. neutron-lib3.21.1 rejects shared/ip_version/tenant_id/id PUT fields; raw SDK attrs stay unpoliced and backend rejection authoritative.
- Find typed Microversion applies only to GET, matching Resource.find; fallback LIST keeps original/default version. Per-call Headers apply to both phases under SDK-owned protocol restrictions.

### `python:network/v2/subnet_pools`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPaginated`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatus`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.Network`
- `github.com/JSYoo5B/go-openstacksdk.Connection.NetworkV2`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordListPaginated`

- Ordinary SubnetPool.Resources.List/All classify the complete pinned declaration through SDK-owned WithFilter/WithFilters: sixteen canonical query fields and twenty accepted canonical/wire spellings, ten local Body properties and nine reserved list controls. SubnetPool inherits Resource+TagMixinNetwork, not NetworkResource. id remains local; queried name/project_id/default/shared/ip_version/address_scope/tags stay server-only even when returned fields differ. Generated leaf and cached Network/NetworkV2 facades share the descriptor without implicit detail GET, prefix mutation or caller builder.
  Tests: `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersEntireQueryAndAliasDescriptor`, `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersRawPresenceIntegersAndTimestamps`, `connection_subnet_pool_filters_test.go:TestConnectionSubnetPoolFiltersSharedFacadesAndRawDescriptors`.
- Canonical is_shared/any_tags/not_tags/not_any_tags transpose to shared/tags-any/not-tags/not-tags-any. Bulk canonical key presence wins over a wire alias even for null/false/empty string/empty array; individual options targeting the same query are last-wins. Scalar strings, lowercase booleans and original JSON-number text encode without response-descriptor coercion. Flat arrays retain repeated scalar order and omit null elements; null/empty arrays omit URL values while keeping collision presence. Selected query objects/nested arrays fail, while unknown semantic names and their encoding errors are discarded.
  Tests: `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersEntireQueryAndAliasDescriptor`, `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersLazyPreflightAndNamespaceCollisions`, `connection_subnet_pool_filters_test.go:TestConnectionSubnetPoolFiltersSharedFacadesAndRawDescriptors`, `resource/filters_test.go:TestSemanticFiltersCanonicalPrecedencePresenceAndExactQueryEncoding`.
- Local id/tenant_id/prefixes/created_at/updated_at read original row JSON after successful native whole-page extraction. Literal missing/null ID or tenant matches null, distinct from a present empty string; tenant never falls back to project_id and the project response alias does not create a tenant query alias. Full prefix arrays compare ordered length/elements and preserve null elements for matching while the returned native []string uses an empty string. No CIDR normalization is added. Timestamp conditions compare original text, so equivalent instants with different text differ, while returned CreatedAt/UpdatedAt retain native time.Time decoding.
  Tests: `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersRawPresenceIntegersAndTimestamps`, `connection_subnet_pool_filters_test.go:TestConnectionSubnetPoolFiltersSharedFacadesAndRawDescriptors`, `connection_subnet_pool_filters_test.go:TestConnectionSubnetPoolFiltersPreflightAndIndependentRawID`, `resource/body_record_test.go:TestBodyRecordPairingPreservesOriginalRowsAndOwnsFields`, `internal/jsonfilter/filter_test.go:TestMatchFiltersArraysCompareExactly`.
- The five selected integer response descriptors default_prefix_length/maximum_prefix_length/minimum_prefix_length/default_quota/revision_number use exact-integral raw normalization. Integral decimal/exponent numbers and signed decimal integer strings compare mathematically when permitted by preceding native decoding; large original prefix numbers remain distinct from rounded native float64/int projection, and filter values are not coerced. Selected fractional prefix responses are terminal invalid-option errors rather than Python int truncation; unselected native fractional prefixes retain native truncation. Missing/null quota/revision can match null, while missing/null native prefix fields fail native decoding first.
  Tests: `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersRawPresenceIntegersAndTimestamps`, `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersControlsAndNativeWholePageFailures`, `connection_subnet_pool_filters_test.go:TestConnectionSubnetPoolFiltersSharedFacadesAndRawDescriptors`, `resource/body_record_test.go:TestBodyRecordFieldNormalizationPresenceAndCauses`, `internal/jsonfilter/filter_test.go:TestMatchFiltersExactDecimalNumbersAndHugeExponents`, `internal/jsonfilter/filter_test.go:TestMatchFiltersDeliberatelyDistinguishesBooleansAndNumbers`.
- Options own deep JSON/query snapshots for independent concurrent reuse. Bulk nil/empty replacement clears only semantic state and validates final selected values, allowing overwritten errors to disappear; explicit Body/raw-query namespaces remain separate. Same-wire semantic/raw/name/page-size and same-Body semantic/explicit destinations fail lazily before HTTP even for null/empty/equal values. Reserved controls, selected invalid encodings, unsupported descriptors and canceled contexts yield terminal preflight errors. Local Body id/tenant conditions and raw wire id/tenant parameters are independent destinations.
  Tests: `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersSnapshotsReplacementAndConcurrentReuse`, `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersLazyPreflightAndNamespaceCollisions`, `connection_subnet_pool_filters_test.go:TestConnectionSubnetPoolFiltersPreflightAndIndependentRawID`, `resource/filters_test.go:TestSemanticFiltersSnapshotsConcurrentReuseAndServerOnlyName`, `resource/filters_test.go:TestSemanticFiltersValidateOnlyFinalSelectedValues`, `resource/filters_test.go:TestSemanticFiltersRejectSelectedErrorsAndReservedControlsBeforeIteration`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`.
- Native extraction of the entire page precedes raw matching and cap consumption. All three native prefix lengths must be numbers or Atoi-compatible strings: missing/null/bool/object/invalid-string values, null rows and empty objects fail even on the query-only lane or after a cap. Known quota/revision/string/bool/tag/prefix-array/timestamp errors keep native causes; mixed incompatible timestamp modes also fail. Only after extraction succeeds can raw cap or break avoid a later selected fractional normalization or continuation. MaxItems counts raw rows before local Body/name predicates without refill or invented wire limit, and All discards partial rows on terminal errors.
  Tests: `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersControlsAndNativeWholePageFailures`, `resource/body_record_test.go:TestBodyRecordPairingRejectsUnprovenShapeAndCount`, `resource/body_record_test.go:TestCollectionBodyRecordLaneIsSelectedOnlyForPreparedFilters`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersRawControlsAndConsumerBreak`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersNativePageErrorsAndContinuation`.
- Ordinary native listing preserves HTTP 200/204/300 and rejects 201. A successfully parsed 204 is empty while empty application/json 204 retains native EOF. subnetpools_links rel=next supplies continuation; top-level next/links.next alone is ignored and duplicate matching rows remain ordinary rows. Foreign advertised URLs retain native following. Later HTTP/decode/foreign failures, cycles and cancellation are terminal. Configured headers/current microversion, original ProviderClient/live token, middleware, context and ResourceBase reverse prefix remain in use across pages.
  Tests: `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersPagingLiveSourceAndNativeStatus`, `connection_subnet_pool_filters_test.go:TestConnectionSubnetPoolFiltersSharedFacadesAndRawDescriptors`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersNativePageErrorsAndContinuation`.
- Raw WithQuery remains a literal wire extension, including id/tenant_id/prefixes/raw prefix spellings/status/vendor parameters; semantic raw prefix spellings remain unknown rather than invented property aliases. WithName retains server hint plus exact local predicate while semantic queried name is server-only. WithStatus stays unsupported on the statusless model. Native typed List, FindIdentity and AddPrefixes/RemovePrefixes keep their existing signatures and request behavior; ordinary list filters do not leak into their GET or prefix-operation requests.
  Tests: `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersNativeAndIdentitySurfaceIsolation`, `api/subnet_pool_list_filters_test.go:TestSubnetPoolListFiltersLazyPreflightAndNamespaceCollisions`, `connection_subnet_pool_filters_test.go:TestConnectionSubnetPoolFiltersPreflightAndIndependentRawID`, `api/network_subnet_body_filters_test.go:TestNetworkSubnetBodyFiltersLocalWireAndNativeSurfaces`.
- Generation requires the checked-in SubnetPool manifest and live audited -openstacksdk-source checkout. Seven source SHA files are verified before isolated AST-only extraction and full manifest comparison, including parser minor version, exact Resource+TagMixinNetwork C3 MRO, inherited limit/marker defaults, tag query expansion, all local classification, unknown discard and canonical query precedence. Thirty-three AST anchors independently prove the five integer descriptors, prefixes list descriptor, project alias/deprecated tenant, untyped timestamps, inherited ID/accessor/default behavior and local-dictionary filtering. SubnetPool.name is ordinary rather than alternate_id; no SDK import, NetworkResource hierarchy or tenant query is fabricated. Missing/stale source or manifest fails closed, and the prior six manifests stay exact.
  Tests: `internal/cmd/sdkgen/subnet_pool_filters_test.go:TestSubnetPoolPythonFilterManifestKeepsTagAliasesLocalIntegersAndInheritedMRO`, `internal/cmd/sdkgen/subnet_pool_filters_test.go:TestSubnetPoolPythonFilterPinnedSourceRequiresIntegerCoercionAndClassificationProof`.
- The raw semantic plan requires exact native SubnetPool 18/ListOpts 22/Link 2 field/type/tag shapes, sole pointer model/time-wrapper decoders, the sole value query-builder and exact interface signature, and native linked-page/extractor/result contracts. Twenty-two receiver-qualified native/root/pagination declaration hashes protect routes, native extraction and NoZ/RFC3339 decoding, page creation/GetBody/UseNumber pairing, accepted page codes and handler/continuation ordering. Two constants protect resourcePath and the NoZ time layout. Emission retains repeat/nil/empty whole query maps on both lanes, raw status, the prior six audited bindings and distinct explicit Body/semantic inventory metadata. Identity opt-ins remain twenty, Body bindings eight and semantic bindings seven; actual compiled native export schemas are also checked.
  Tests: `internal/cmd/sdkgen/subnet_pool_filters_test.go:TestSubnetPoolBodyFilterNativeContractRejectsModelBuilderDecoderAndSourceDrift`, `internal/cmd/sdkgen/subnet_pool_filters_test.go:TestSubnetPoolSemanticFilterEmissionPreservesQueryMapsStatusAndPriorSixBindings`, `internal/cmd/sdkgen/subnet_pool_filters_test.go:TestSubnetPoolRawDependencyLoaderPreservesNativeTimeCodesAndNumbers`, `internal/cmd/sdkgen/subnet_pool_filters_test.go:TestSubnetPoolBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods`.
- Update sends only raw supplied attrs in subnetpool envelope, resolves canonical/wire aliases in option order, preserves null/false/zero/unknown attrs and sparse member seed while retaining independent literal Wire/Envelope/Header/status. No attrs is a no-HTTP commit, tags default to [] and missing Body fields become null; this Resource has no NetworkResource conditional Header descriptors.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsUpdateAliasesNoopAndSeedWireProjection`, `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsMemberDescriptorsSparseAndMalformedEvidence`.
- Member kwargs remain original query plus constructor seed; fallback/direct lists transpose only Source declared query aliases, canonical keys override wire aliases even when null, known nonserver canonical Body attrs filter descriptor views, unknown list attrs are discarded, and all_projects is consumed without granting access. fields list values remain literal, tags and repeated query values preserve Requests JSON-domain representation. LIST Accept defaults override source Accept while explicit Headers prevail.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsQueryClassificationAliasesAndLocalDescriptors`.
- Lazy owned full pagination supports body/header continuation, marker fallback on short pages, raw-row caps before local filters and single-page choice. Find checks all candidate rows/pages before returning one exact ID/name; late duplicates/rejected/malformed pages remain errors. Cycles and conflicting/foreign continuation are terminal; standard dictionary next links are a declared Go improvement.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsPaginationFullSearchAndRawCaps`.
- Only clean native400/403/404 member rejection falls back; default missing requires a completely successful search and strict missing retains no suppressed native rejection cause. Nonobject row/envelope and constructor cls/connection/microversion/_synchronized collisions retain actual page proof. Source response descriptors retain integer/bool/list/null/tag/tenant aliases; malformed accepted member syntax follows seed while valid wrong shapes fail.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsCleanMissingAndConstructorFailures`, `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsMemberDescriptorsSparseAndMalformedEvidence`.
- Accepted/rejected Read+Close/cancel/source-changed-and-restored-on-Close failures preserve every cause and physical body/header/status, close once, stop before fallback/missing and never resend. Member result receipts for accepted physical failures hold actual metadata without a fabricated descriptor acknowledgement.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsPhysicalFailuresPreserveAcceptedAndRejectedProof`.
- Native retry cannot change/delete/alias/omit selected network version, inject version when absent, add foreign-service/auth/framing headers or mutate JSON body/Content-Type. One503 proof plus native callback causes survives; ordinary retry headers, exact method/URL/JSON, live token and reauthentication are retained.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsRetriesKeepProtocolBodyAndNativeCauses`.
- Source is captured before callbacks; concrete options snapshot raw JSON/maps/pointers and retained callbacks cannot mutate requests/results. Invalid controls/headers/IDs/descriptors/context/options fail before HTTP and preserve causes. One immutable option supports concurrent requests with independent records/evidence.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsOptionsSourceContextAndConcurrentOwnership`.
- Source Resource.find consumes the explicit microversion for member fetch only; fallback LIST and all its pages keep the captured original selected/default microversion. Ordinary Header options apply to both phases. Fallback retry cannot inject the member-only override when selected version differs or is absent; one original LIST503 body/header/status survives. Direct List/Update per-operation overrides remain effective and never mutate source.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsFindMicroversionIsMemberPhaseOnly`.
- Library-owned PagePolicy.NextTruthy reuses SubnetPool response binary64 truthiness: top-level next1e-400/-1e-400/0.0 are falsey and do not request another page, while truthy numeric1e400/1.0 fail URL decoding with exact whole actual page body/header/status. Normal bool/null next remain falsey; independently yielded row receipts retain their own page snapshots.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsNextUsesBinary64TruthinessAndWholePageProof`.

Python/Go differences:

- The complete declared query/local classification is now SDK-owned through concrete semantic and explicit Body options. id is local rather than a Python query field; queried name/project/default/shared/ip_version/address_scope/tags are server-only. The project response alias does not create a tenant query alias. WithName retains separate hint/local behavior and raw WithQuery is a wire extension, without a caller builder or predicate.
- Individual semantic options are last-wins and bulk canonical presence wins over wire aliases. Only final selected semantic values are validated; same-destination collisions reject even null/empty/equal values. Query arrays encode repeated scalar values with null omission, booleans are lowercase, JSON-number spelling is retained and unknown semantic keys/errors are discarded. Raw prefix spellings are available through explicit Body aliases or raw query, rather than invented semantic property names.
- The five integer response descriptors use exact-integral Go normalization after native decoding. Python int descriptor behavior can truncate fractions, retain bool-as-int values, and default ordinary signed/non-digit strings differently; these conversions are not replicated. Filter values stay uncoerced. Native quota/revision int decoding remains strict, and native prefix floats/strings retain their returned int/float64 or Atoi limits independently of raw comparisons.
- Original timestamps, literal ID/tenant presence and prefix-array null elements drive matching, while returned typed times, native zero strings and []string null-to-empty projection remain unchanged. Native known TenantID is a string and malformed values fail first; no arbitrary tenant-object or precision claim is made. Arrays compare full ordered elements, bools remain distinct from numbers, and Python scalar-to-list wrapping/default/alias projection is not added.
- Whole-page native decoding remains authoritative before raw caps: every row requires all three valid prefix-length fields, so null/empty rows fail on both query and Body lanes. Selected raw integer normalization runs only on consumed fields after that boundary; caps/break can avoid later normalization but cannot hide native type/required-prefix/time errors.
- Native 200/204/300, plural-links continuation and foreign URL following are preserved, including application/json empty-204 EOF. Raw caps count rows like pinned Resource.list but do not infer wire limit; first-page/break stop before continuation and All drops partial results on errors. No RFC-Link parsing, same-origin/fixed-query or synthesized marker policy is added.
- Bounded named mappings cover literal identity/name and JSON-domain attrs, not arbitrary Python mutable Resource instances, Connection/session/cache side effects, resource location calculation or deprecation warnings. location is null; whole Source Resource/Connection parity remains separately tracked.
- Go rejects SDK constructor/control attr collisions rather than allowing caller retargeting, and supplies per-call owned Headers/Microversion and typed list-only pagination conveniences. Raw unknown Body/Find kwargs remain data; unsupported list kwargs are discarded.
- Bulk Go maps are sorted when conflicting canonical/wire alias insertion order matters; individual mutation options preserve order. Local map filter evaluation is sorted. Shared JSON query float repr/Unicode runtime and CPython int-string-limit differences remain explicit common representation bounds.
- Owned pager additionally follows standard links:{next:URL}, which pinned Source dict-to-item loop ignores, and rejects ambiguous/foreign/cyclic links with physical proof. Raw caps, marker-on-short-page and complete find search match Source supported behavior.
- Fixed Neutron27.0.0 policy permits admin/owning-project reader or shared-pool reads and admin/owning-project member updates; is_default update is separately admin-only. Examples omit that admin attr and all implicit defaults. No explicit service-only branch exists. neutron-lib3.21.1 rejects shared/ip_version/tenant_id/id PUT fields; raw SDK attrs stay unpoliced and backend rejection authoritative.

### `python:network/v2/update_subnet_pool`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordMutationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordMutationMicroversion`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Update sends only raw supplied attrs in subnetpool envelope, resolves canonical/wire aliases in option order, preserves null/false/zero/unknown attrs and sparse member seed while retaining independent literal Wire/Envelope/Header/status. No attrs is a no-HTTP commit, tags default to [] and missing Body fields become null; this Resource has no NetworkResource conditional Header descriptors.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsUpdateAliasesNoopAndSeedWireProjection`, `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsMemberDescriptorsSparseAndMalformedEvidence`.
- Member kwargs remain original query plus constructor seed; fallback/direct lists transpose only Source declared query aliases, canonical keys override wire aliases even when null, known nonserver canonical Body attrs filter descriptor views, unknown list attrs are discarded, and all_projects is consumed without granting access. fields list values remain literal, tags and repeated query values preserve Requests JSON-domain representation. LIST Accept defaults override source Accept while explicit Headers prevail.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsQueryClassificationAliasesAndLocalDescriptors`.
- Lazy owned full pagination supports body/header continuation, marker fallback on short pages, raw-row caps before local filters and single-page choice. Find checks all candidate rows/pages before returning one exact ID/name; late duplicates/rejected/malformed pages remain errors. Cycles and conflicting/foreign continuation are terminal; standard dictionary next links are a declared Go improvement.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsPaginationFullSearchAndRawCaps`.
- Only clean native400/403/404 member rejection falls back; default missing requires a completely successful search and strict missing retains no suppressed native rejection cause. Nonobject row/envelope and constructor cls/connection/microversion/_synchronized collisions retain actual page proof. Source response descriptors retain integer/bool/list/null/tag/tenant aliases; malformed accepted member syntax follows seed while valid wrong shapes fail.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsCleanMissingAndConstructorFailures`, `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsMemberDescriptorsSparseAndMalformedEvidence`.
- Accepted/rejected Read+Close/cancel/source-changed-and-restored-on-Close failures preserve every cause and physical body/header/status, close once, stop before fallback/missing and never resend. Member result receipts for accepted physical failures hold actual metadata without a fabricated descriptor acknowledgement.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsPhysicalFailuresPreserveAcceptedAndRejectedProof`.
- Native retry cannot change/delete/alias/omit selected network version, inject version when absent, add foreign-service/auth/framing headers or mutate JSON body/Content-Type. One503 proof plus native callback causes survives; ordinary retry headers, exact method/URL/JSON, live token and reauthentication are retained.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsRetriesKeepProtocolBodyAndNativeCauses`.
- Source is captured before callbacks; concrete options snapshot raw JSON/maps/pointers and retained callbacks cannot mutate requests/results. Invalid controls/headers/IDs/descriptors/context/options fail before HTTP and preserve causes. One immutable option supports concurrent requests with independent records/evidence.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsOptionsSourceContextAndConcurrentOwnership`.
- Source Resource.find consumes the explicit microversion for member fetch only; fallback LIST and all its pages keep the captured original selected/default microversion. Ordinary Header options apply to both phases. Fallback retry cannot inject the member-only override when selected version differs or is absent; one original LIST503 body/header/status survives. Direct List/Update per-operation overrides remain effective and never mutate source.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsFindMicroversionIsMemberPhaseOnly`.
- Library-owned PagePolicy.NextTruthy reuses SubnetPool response binary64 truthiness: top-level next1e-400/-1e-400/0.0 are falsey and do not request another page, while truthy numeric1e400/1.0 fail URL decoding with exact whole actual page body/header/status. Normal bool/null next remain falsey; independently yielded row receipts retain their own page snapshots.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsNextUsesBinary64TruthinessAndWholePageProof`.

Python/Go differences:

- Go updates a concrete ID with typed UpdateOpts and permitted body fields and returns a native typed value/error. Python accepts ID or existing mutable Resource, applies attrs/descriptor state and commits that Resource.
- Typed Go RevisionNumber nil/zero/signed semantics use the existing native conditional-header capability without a new public helper. This pinned Python resource does not inherit NetworkResource, so forwarded if_revision binds the helper but does not generate If-Match; the Go conditional capability is a native extension difference.
- Go preserves raw native MoreHeaders/OmitHeaders/callback override and retry behavior and reports native412 without SDK unconditional fallback; generic Python adapter/session and mutable/dirty header behavior remain separate.
- Bounded named mappings cover literal identity/name and JSON-domain attrs, not arbitrary Python mutable Resource instances, Connection/session/cache side effects, resource location calculation or deprecation warnings. location is null; whole Source Resource/Connection parity remains separately tracked.
- Go rejects SDK constructor/control attr collisions rather than allowing caller retargeting, and supplies per-call owned Headers/Microversion and typed list-only pagination conveniences. Raw unknown Body/Find kwargs remain data; unsupported list kwargs are discarded.
- Bulk Go maps are sorted when conflicting canonical/wire alias insertion order matters; individual mutation options preserve order. Local map filter evaluation is sorted. Shared JSON query float repr/Unicode runtime and CPython int-string-limit differences remain explicit common representation bounds.
- Owned pager additionally follows standard links:{next:URL}, which pinned Source dict-to-item loop ignores, and rejects ambiguous/foreign/cyclic links with physical proof. Raw caps, marker-on-short-page and complete find search match Source supported behavior.
- Fixed Neutron27.0.0 policy permits admin/owning-project reader or shared-pool reads and admin/owning-project member updates; is_default update is separately admin-only. Examples omit that admin attr and all implicit defaults. No explicit service-only branch exists. neutron-lib3.21.1 rejects shared/ip_version/tenant_id/id PUT fields; raw SDK attrs stay unpoliced and backend rejection authoritative.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
pool = conn.network.find_subnet_pool("POOL_NAME_OR_ID")
rows = list(conn.network.subnet_pools(is_shared=False, default_quota=128))
if pool is not None:
    updated = conn.network.update_subnet_pool(pool.id, name="renamed", default_quota=0)
    unchanged = conn.network.update_subnet_pool(pool.id)
```

```sh
go run . -cloud dev -action find -pool POOL_NAME_OR_ID
go run . -cloud dev -action list
go run . -cloud dev -action update -pool POOL_ID -name renamed -quota 0
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    action := flag.String("action", "find", "find, list, update")
    pool := flag.String("pool", "", "find identity or update literal ID")
    name := flag.String("name", "", "optional update name")
    quota := flag.Int("quota", 0, "optional update quota; supply -quota to send zero")
    flag.Parse()
    sent := map[string]bool{}
    flag.Visit(func(f *flag.Flag) { sent[f.Name] = true })
    if (*action == "find" || *action == "update") && *pool == "" {
        log.Fatal("-pool is required")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    var rows []*subnetpools.SubnetPoolRecord
    switch *action {
    case "find":
        row, findErr := network.SubnetPools.FindRecord(ctx, *pool)
        err = findErr
        if row != nil { rows = append(rows, row) }
    case "list":
        rows, err = network.SubnetPools.AllRecords(ctx)
    case "update":
        options := []subnetpools.RecordMutationOption{}
        if sent["name"] { options = append(options, subnetpools.WithRecordAttribute("name", *name)) }
        if sent["quota"] { options = append(options, subnetpools.WithRecordAttribute("default_quota", *quota)) }
        row, updateErr := network.SubnetPools.UpdateRecord(ctx, *pool, options...)
        err = updateErr
        if row != nil { rows = append(rows, row) }
    default:
        log.Fatal("select find, list or update")
    }
    if err != nil { log.Fatal(err) }
    for _, row := range rows {
        raw, marshalErr := json.MarshalIndent(row.Resource.Body, "", "  ")
        if marshalErr != nil { log.Fatal(marshalErr) }
        fmt.Printf("HTTP %d\n%s\n", row.StatusCode, raw)
    }
}
```

## References

- [Install](../../docs/install.md)
- [Subnetpool](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/subnetpool.py)
- [neutron-lib 3.21.1 schema](https://github.com/openstack/neutron-lib/blob/3.21.1/neutron_lib/api/definitions/subnetpool.py)
- [address-scope extension](https://github.com/openstack/neutron-lib/blob/3.21.1/neutron_lib/api/definitions/address_scope.py)
