# QoS Policy Records Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindHeaders`

- Pinned direct find_qos_policy (openstack/network/v2/_proxy.py:5130) accepts a string, ignore_missing=True and query. Native GET200 /qos/policies/id and policy envelope preserve canonical IDs and typed model extraction; undeclared GET203/204 and accepted native decode errors remain terminal without fallback or resend.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesGETModelsAndNativeCodes`.
- Compatible GET400/403/404 may fall back, with explicit 404-only/GET-only and strict options. Logical absence defaults to nil,nil after successful complete native absence; LIST403/404, subsequent HTTP/decode/cycle/cancellation and same-ID/cross-project ambiguity over all native pages remain terminal.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesFallbackQueryAndMissingPolicies`, `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesAllPagesAndTerminalObservations`.
- Caller raw project/shared/fields/tags/status/vendor query is frozen and retained in both phases with repetitions, explicit nil/empty name precedence and concurrent reuse. Only an absent caller name gets a literal LIST hint; raw status alone is wire-only and no project scope is inferred.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesFallbackQueryAndMissingPolicies`, `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesQuerySnapshotsAndConcurrentReuse`.
- Leaf and cached Connection.Network.API retain live provider/token, configured headers/version, HTTP middleware, context and ResourceBase. Unsafe names are list-only, GET-only unsafe names and nil/canceled context fail before HTTP; explicit Details/AllProjects/ExtraSpecs false or true and typed WithStatus are unsupported before HTTP.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesLiveSourceAndCancellation`, `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesUnsafeNamesAndCapabilityPreflight`.
- Ordinary Resources.List/All preserve full raw wire query with native extraction and local cap-before-name/first-page controls, without adding a wire limit or identity lookup. Native typed List and explicit ID/Name Ref lookups remain separate.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesOrdinaryCollectionsQueriesAndControls`.
- Native collection uses policies_links rel=next and ignores top-level links.next alone. All advertised pages participate in identity matching and later errors. Policy preserves Shared/IsDefault bools, Rules map slices, project/tenant/revision/tags and RFC3339 time.Time; malformed bool, rules and old timestamp values remain decode errors.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesGETModelsAndNativeCodes`, `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesAllPagesAndTerminalObservations`.
- Exact proxy find_qos_policy performs one literal member GET with raw original kwargs and constructor seeds. Only clean400/403/404 fall back; complete ID/name equality search defaults ignore_missing true, explicit false returns logical ErrNotFound only after a successful full search. List HTTP failures remain terminal.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordMemberSeedDefaultsAndActualEvidence`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordFallbackQueryMappingAndLocalFilters`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordNamePresenceMissingAndPassiveResponseID`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordRetryCallbacksKeepCleanFallbackAndGuards`.
- GET constructor canonical/wire aliases are consumed in insertion order, response members override seed in their order and bulk maps use sorted keys; LIST query canonical alias precedence is distinct. Name omitted creates only the fallback name hint; explicit null/empty remains supplied.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordConstructorAliasOrderIsSeparateFromListQueryPrecedence`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordLocalFilterDescriptorAndAliasOrder`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordNamePresenceMissingAndPassiveResponseID`, `internal/neutronfind/find_policy_test.go:TestSourceProfileOwnsOrderedHeaderAndBodySeedWithoutChangingBaseline`.
- Source15 server query mappings include unchanged fields/id/is_default/project/sort/tag aliases; original GET kwargs include known and extension values. Fallback only discards unknown/all_projects and evaluates tenant_id/rules locally without native typed decoder coercion.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordFallbackQueryMappingAndLocalFilters`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordLocalFilterDescriptorAndAliasOrder`.
- Shared Source pager checks all matching pages before success, including late duplicates/errors, raw-row caps before local filtering, link formats, short-page marker fallback and scope/query/cycle proof. Truthy malformed continuation stays a physical page error.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordLateDuplicateErrorMarkerAndRawCap`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordMalformedPagesAndDetachedExtensionEvidence`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordContinuationFormatsAndCycleGuards`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordPaginationTruthiness`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyRecordsFalseyNumericNextUsesSourceTruthiness`.
- Explicit find microversion governs GET only; fallback LIST retains captured selected default even with canonical selected-version Header. Headers convenience applies safe snapshot to both phases without mutating original client.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordFallbackUsesSelectedVersionAndSourceReceipt`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyRecordsCanonicalVersionAliasSurvivesCleanRetriesAndPhases`, `internal/neutronfind/find_policy_test.go:TestSourceProfileOwnsFloatTruthinessAndFindVersionPhasesWithoutChangingBaseline`.
- QoSPolicy actual MRO is Resource plus TagMixin, no NetworkResource revision/timestamp/Header descriptors. Missing tags [], rules/bools/null, project_id alias tenant_id, explicit null, scalar list wrapping and unknown response Body precision remain independent from literal Wire and whole Envelope/receipt.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordMemberSeedDefaultsAndActualEvidence`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordLocalFilterDescriptorAndAliasOrder`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsLazyQueryClassificationAndMRODefaults`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsLiveReauthenticationAndSourceContext`.
- Options copy JSON values, map/pointer values and callback intermediates; fixed Source/context is captured before callbacks. Nil/control/malformed known options fail before HTTP, unsafe literal identity escapes or preflight rejects. Caller callbacks cannot replace the chosen route/source.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordOptionSnapshotsAndPreflightGuards`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordSourceCancellationAndConstructorPreflight`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsSnapshotsCallbacksAndPreflight`.
- Final200..399 success, actual400..599 native evidence, accepted and rejected Read/Close/cancel/sticky Source faults retain original response/cause and never become clean lookup absence.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordAllStatusAndPhysicalRejectionContracts`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordPhysicalFailuresPreserveActualEvidence`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordSourceCancellationAndConstructorPreflight`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsAcceptedAndRejectedStatusEvidence`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsPhysicalFaultCancellationAndStickySource`.
- Source profile CPython binary64 float truthiness is used in Boolean descriptors, paginated, recursive dictionary filtering and numeric next; exact decimal equality is deliberately separate. Plain integers remain arbitrary precision and no wire bytes change. Baseline core3 profile behavior remains frozen unless explicitly opted in.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordFloatTruthinessKeepsWireAndExactComparisonBoundary`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordBooleanDescriptorUsesFloatTruthiness`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsRulesAndTenantLocalFilterSemantics`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyRecordsFalseyNumericNextUsesSourceTruthiness`, `internal/neutronfind/find_policy_test.go:TestSourceProfileOwnsFloatTruthinessAndFindVersionPhasesWithoutChangingBaseline`, `internal/neutronfind/find_policy_test.go:TestSourceProfileNumericNextTruthinessDoesNotChangeBaselinePager`.
- Original selected Header version aliases are normalized only in private source/policy snapshots, safe list Accept and foreign/reserved headers remain fixed across retries. Clean503 permits exactly a second request and native live reauth changes token without changing selected source or version facts.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordLiveReauthenticationAndMicroversionOwnership`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordListAcceptAndRetryVersionPolicy`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyRecordsCanonicalVersionAliasSurvivesCleanRetriesAndPhases`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsOperationHeadersAndRetryOwnership`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsLiveReauthenticationAndSourceContext`.

Python/Go differences:

- SDK-owned concrete identity options provide missing/default/fallback/query ownership. Raw wire aliases and unknown keys are explicit Go extensions; Python query transposition, unknown-key discard and local Body classification are not automatic. No hidden/default/enrichment/parent or discovery policy is added.
- Unsafe names are list-only, GET-only rejects them before HTTP and consumed models require safe nonblank IDs. Native accepted statuses, model/extractor and advertised continuation semantics remain intact. Project scope is caller-selected.
- The audited ordinary common-list bridge preserves repeated/nil/empty raw query, including status and extensions. Both native models have no Status, so typed WithStatus is unsupported while raw status is wire-only. Native typed List and explicit Ref behavior are preserved.
- Python is_shared maps to wire shared; Go callers choose shared explicitly. Native Rules retain nested map keys but do not provide generic unknown outer Body attributes or seeded descriptor state. Native time.Time requires RFC3339 without an old timestamp normalizer.
- Immutable PolicyRecord distinguishes library Resource view, literal received Wire and actual whole response receipt; mutable Python Resource dirty/cache/session/Connection-derived location behavior is tracked separately. Existing typed native APIs remain independent.
- Go kwargs values are JSON-domain snapshots, bulk maps have deterministic sorted-key order, max_items is a nonnegative Go integer and fixed literal routes/session/constructor/cache/envelope controls are preflight rejected.
- Local equality preserves exact decimal JSON values rather than CPython binary64 rounding/underflow/overflow, although scalar truthiness matches the Source JSON parse domain and bool/numeric equality is allowed. This is an explicit Go comparison boundary, not full Python numeric equality parity.
- HTTP200..399 is the accepted completion range; Source final<400 informational100..199 is not treated as success. All actual400..599 and physical handler causes are preserved.
- Source external/rewritten continuations are restricted to the selected same-origin collection/query; malformed or cyclic continuation is an actual-page failure instead of forwarding shared credentials or silently changing filters.
- Safe Header options are explicit Go conveniences. Find uses GET-only explicit microversion and captured default for fallback List; standalone List applies its version to every page. Python headers kwargs query-vs-header implicit phase consumption is not inferred.
- Stock Neutron27 get_policy allows owner project reader or shared policy (also admin); create/update/delete are ADMIN_OR_PROJECT_MANAGER with historic admin-only deprecated policy. This wave covers only read named declarations and proves no live deployment role authorization.

### `python:network/v2/qos_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.List`
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
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListPaginated`

- Ordinary QoSPolicy.Resources.List/All classify the complete pinned declaration through SDK-owned WithFilter/WithFilters: fifteen canonical query fields and nineteen accepted canonical/wire spellings, two local Body fields rules/tenant_id and nine reserved list controls. QoSPolicy inherits Resource+TagMixin, not NetworkResource. Queried id/name/description/project_id/is_default/shared/tags remain server-only even when returned values differ; no implicit GET, tag fetch or caller builder is added. Generated leaf and cached Network/NetworkV2 facades share this descriptor.
  Tests: `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersEntireQueryAndAliasDescriptor`, `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersRawRulesAndTenantPresence`, `connection_qos_policy_filters_test.go:TestConnectionQoSPolicyFiltersSharedFacadesAndExactRawRules`.
- Canonical is_shared/any_tags/not_tags/not_any_tags transpose to shared/tags-any/not-tags/not-tags-any. Bulk canonical key presence wins over its wire alias, including null/false/empty string/empty array; individual options selecting the same wire target are last-wins. Semantic scalar strings, lowercase booleans and original JSON-number text encode without response-descriptor coercion, and flat arrays retain repeated scalar order while omitting null elements. Null/empty arrays omit URL values but retain collision presence. Selected objects/nested query arrays fail; unknown semantic names and their encoding errors are discarded.
  Tests: `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersEntireQueryAndAliasDescriptor`, `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersLazyPreflightAndNamespaceCollisions`, `connection_qos_policy_filters_test.go:TestConnectionQoSPolicyFiltersSharedFacadesAndExactRawRules`, `resource/filters_test.go:TestSemanticFiltersCanonicalPrecedencePresenceAndExactQueryEncoding`.
- Local rules matching reads the original row JSON after successful native whole-page extraction. Missing/null whole arrays match null; empty arrays remain distinct. Arrays require the full ordered length and values, including all keys inside nested rule objects and null map elements; dictionaries inside arrays are not subset predicates. Decimal comparisons are exact, so raw 9007199254740993 differs from 9007199254740992 while equivalent decimal/exponent values match. JSON booleans remain distinct from numbers. These raw comparisons do not replace returned native Rules or bypass its float64-range/type decode failures.
  Tests: `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersRawRulesAndTenantPresence`, `connection_qos_policy_filters_test.go:TestConnectionQoSPolicyFiltersSharedFacadesAndExactRawRules`, `api/network_body_filters_test.go:TestNetworkBodyFiltersQoSNativeNumberProjectionAndStrictTypes`, `resource/body_record_test.go:TestBodyRecordPairingPreservesOriginalRowsAndOwnsFields`, `internal/jsonfilter/filter_test.go:TestMatchFiltersArraysCompareExactly`, `internal/jsonfilter/filter_test.go:TestMatchFiltersExactDecimalNumbersAndHugeExponents`, `internal/jsonfilter/filter_test.go:TestMatchFiltersDeliberatelyDistinguishesBooleansAndNumbers`, `api/network_subnet_raw_body_filters_test.go:TestNetworkSubnetRawBodyFiltersWireAndNativeIsolation`.
- The deprecated local tenant_id reads its literal original field without a fallback to project_id; project_id response-descriptor aliasing does not create a tenant query alias. Missing/null tenant matches null, distinct from a present empty string, while returned native TenantID still uses its zero string and native ProjectID remains independent. Native Policy.Rules continues []map[string]any/float64 decoding, including preexisting preservation of null map elements and null values; this unit improves matching precision rather than the returned model. Native TenantID is a known string, so non-null numeric/object tenant responses fail before raw matching.
  Tests: `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersRawRulesAndTenantPresence`, `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersControlsAndNativeWholePageFailures`, `connection_qos_policy_filters_test.go:TestConnectionQoSPolicyFiltersSharedFacadesAndExactRawRules`, `api/network_body_filters_test.go:TestNetworkBodyFiltersLocalWireAndNativeSurfaces`.
- Options own deep JSON/query snapshots for independent concurrent reuse. Semantic bulk replacement/nil/empty clear affects only semantic state and validates only final selected values, so superseded encoding errors can disappear; explicit Body/raw query namespaces retain their separate snapshots and errors. Same-wire semantic/raw/name/page-size and same-Body semantic/explicit destinations fail lazily before HTTP even for null/empty/equal values. Reserved controls, selected invalid values, unsupported descriptors and canceled contexts yield terminal preflight errors. Raw queries and local Body conditions with the same text are independent destinations.
  Tests: `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersSnapshotsReplacementAndConcurrentReuse`, `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersLazyPreflightAndNamespaceCollisions`, `connection_qos_policy_filters_test.go:TestConnectionQoSPolicyFiltersPreflightAliasesAndNativeNameHint`, `resource/filters_test.go:TestSemanticFiltersSnapshotsConcurrentReuseAndServerOnlyName`, `resource/filters_test.go:TestSemanticFiltersValidateOnlyFinalSelectedValues`, `resource/filters_test.go:TestSemanticFiltersRejectSelectedErrorsAndReservedControlsBeforeIteration`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`, `resource/body_filters_test.go:TestCollectionBodyFiltersOptionApplicationOwnsRawBytes`, `resource/body_filters_test.go:TestCollectionBodyFiltersSnapshotsBulkAliasesAndConcurrentReuse`.
- Native extraction of the entire page precedes raw matching and cap consumption, preserving strict known string/bool/int/tag-array/rule-array/RFC3339 timestamp errors even beyond a cap, with native causes. A null row consumed by the Body lane is invalid; query-only null rows keep the native zero model and an empty object is valid. After native extraction succeeds a cap or early break can avoid consuming a later null row or continuation. MaxItems counts raw rows before local Body/name predicates without refill or an inferred wire limit, first-page/break stop before continuation, and All discards partial rows on terminal errors.
  Tests: `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersControlsAndNativeWholePageFailures`, `resource/body_record_test.go:TestBodyRecordPairingRejectsUnprovenShapeAndCount`, `resource/body_record_test.go:TestCollectionBodyRecordLaneIsSelectedOnlyForPreparedFilters`, `resource/body_filters_test.go:TestCollectionBodyFiltersRunAfterRawCapsOnBothIteratorPaths`, `api/network_body_filters_test.go:TestNetworkBodyFiltersRawControlsNameAndConsumerBreak`, `api/network_body_filters_test.go:TestNetworkBodyFiltersNativePageErrorsAndLateCancellation`.
- Ordinary native listing preserves HTTP 200/204/300 and rejects 201. A successfully parsed 204 is empty while empty application/json 204 retains native EOF. policies_links rel=next supplies continuation; top-level next/links.next alone is ignored and matching duplicates remain ordinary rows. Foreign advertised URLs retain native following, while later HTTP/decode/foreign errors, cycles and cancellation remain terminal. Configured headers/current microversion, original ProviderClient/live token, middleware, context and ResourceBase reverse prefix stay in use across pages.
  Tests: `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersPagingLiveSourceAndNativeStatus`, `connection_qos_policy_filters_test.go:TestConnectionQoSPolicyFiltersSharedFacadesAndExactRawRules`, `api/network_body_filters_test.go:TestNetworkBodyFiltersNativePageErrorsAndLateCancellation`.
- Ordinary raw WithQuery preserves complete literal wire parameters such as tenant_id/rules/is_shared/status/vendor separately from semantic transposition. Existing WithName remains a server hint plus exact local predicate; semantic queried name remains server-only. WithStatus stays unsupported because Policy has no Status, while raw status remains wire-only on both normal and Body lanes. Existing native typed List and FindIdentity keep their separate signatures/raw query policies without ambient semantic/Body options or new details/all-projects/enrichment capabilities.
  Tests: `api/qos_policy_list_filters_test.go:TestQoSPolicyListFiltersNativeAndIdentitySurfaceIsolation`, `connection_qos_policy_filters_test.go:TestConnectionQoSPolicyFiltersPreflightAliasesAndNativeNameHint`, `api/network_body_filters_test.go:TestNetworkBodyFiltersLocalWireAndNativeSurfaces`.
- Generation requires the checked-in QoSPolicy manifest and a live audited -openstacksdk-source checkout. Six source SHA files are checked before isolated AST-only extraction and full manifest comparison, including parser minor version, exact Resource+TagMixin C3 MRO, inherited limit/marker defaults, tag query expansion, Body classification, unknown discard and canonical precedence. Eighteen AST anchors independently prove list-typed rules, project_id alias/deprecated tenant, TagMixin query/default fields and inherited ID/accessor behavior. No SDK import or fabricated NetworkResource/tenant-query policy is used; missing/stale source or manifest fails closed and the prior five manifests stay exact.
  Tests: `internal/cmd/sdkgen/qos_policy_filters_test.go:TestQoSPolicyPythonFilterManifestKeepsTagQueryAliasesAndLocalTenantRules`, `internal/cmd/sdkgen/qos_policy_filters_test.go:TestQoSPolicyPythonFilterPinnedSourceRequiresTagMROAliasAndListCoercionProof`.
- The raw semantic plan requires exact native Policy 12/ListOpts 16/Link 2 field/type/tag shapes, no own model methods, the sole value query-builder and interface signatures, and linked-page/extractor contracts. Nineteen receiver-qualified native/root/pagination declaration hashes protect routes, native extraction including ExtractPolicysInto and slice-pointer decoding, GetBody pairing and UseNumber precision. Emission retains repeat/nil/empty whole query maps on both lanes, raw status, prior five audited bindings and distinct explicit Body/semantic inventory metadata. Identity opt-ins remain twenty, Body bindings eight and semantic bindings six.
  Tests: `internal/cmd/sdkgen/qos_policy_filters_test.go:TestQoSPolicyBodyFilterNativeContractRejectsModelBuilderInterfacePagerAndSourceDrift`, `internal/cmd/sdkgen/qos_policy_filters_test.go:TestQoSPolicySemanticFilterEmissionPreservesQueryMapsStatusAndPriorFiveBindings`, `internal/cmd/sdkgen/qos_policy_filters_test.go:TestQoSPolicyRawDependencyLoaderPreservesSliceDecoderNumbersAndNativeLinks`, `internal/cmd/sdkgen/qos_policy_filters_test.go:TestQoSPolicyBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods`.
- Exact proxy qos_policies has no name hint. Lazy ListRecords/AllRecords share the Find Record profile and concrete options; declared server query vs wire aliases/local tenant_id and rules vs unknown discarded kwargs are owned by the library. IDs/fields repeated values remain literal.
  Tests: `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsLazyQueryClassificationAndMRODefaults`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsRulesAndTenantLocalFilterSemantics`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsSnapshotsCallbacksAndPreflight`.
- List Query canonical alias wins wire alias, response constructor Body aliases use ordered received fields, raw rules retain exact nested numbers; list descriptor wraps nonlists and explicit null stays null without strict native time/int projection.
  Tests: `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsLazyQueryClassificationAndMRODefaults`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsRulesAndTenantLocalFilterSemantics`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsLiveReauthenticationAndSourceContext`.
- Raw row cap is counted before local filters; default full pagination, false first page, zero limit, max_items limit hint and iteration break have owned behavior. Partial rows may have been yielded on late failure; AllRecords discards partial slices.
  Tests: `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsRawCapsFirstPageAndEarlyBreak`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsPartialIteratorAndAllDropOnLateFailure`.
- All declared Source link forms and marker fallback are used through shared owned pager with fixed same-origin collection/query; changed filters/origin/routes/cycles and malformed pages preserve actual receipt. Numeric next underflow follows Source while ordinary truthy malformed next remains an error.
  Tests: `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsContinuationFormatsMarkerFallbackAndScope`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsPartialIteratorAndAllDropOnLateFailure`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyRecordsFalseyNumericNextUsesSourceTruthiness`.
- Standalone List per-call microversion governs all pages; default application/json Accept overrides configured source Header and explicit safe Accept wins. Retry/header ownership and canonical version aliases preserve selected Source, live token and original rejected evidence.
  Tests: `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyRecordsCanonicalVersionAliasSurvivesCleanRetriesAndPhases`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsOperationHeadersAndRetryOwnership`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsLiveReauthenticationAndSourceContext`.
- QoSPolicy actual MRO is Resource plus TagMixin, no NetworkResource revision/timestamp/Header descriptors. Missing tags [], rules/bools/null, project_id alias tenant_id, explicit null, scalar list wrapping and unknown response Body precision remain independent from literal Wire and whole Envelope/receipt.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordMemberSeedDefaultsAndActualEvidence`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordLocalFilterDescriptorAndAliasOrder`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsLazyQueryClassificationAndMRODefaults`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsLiveReauthenticationAndSourceContext`.
- Options copy JSON values, map/pointer values and callback intermediates; fixed Source/context is captured before callbacks. Nil/control/malformed known options fail before HTTP, unsafe literal identity escapes or preflight rejects. Caller callbacks cannot replace the chosen route/source.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordOptionSnapshotsAndPreflightGuards`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordSourceCancellationAndConstructorPreflight`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsSnapshotsCallbacksAndPreflight`.
- Final200..399 success, actual400..599 native evidence, accepted and rejected Read/Close/cancel/sticky Source faults retain original response/cause and never become clean lookup absence.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordAllStatusAndPhysicalRejectionContracts`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordPhysicalFailuresPreserveActualEvidence`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordSourceCancellationAndConstructorPreflight`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsAcceptedAndRejectedStatusEvidence`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsPhysicalFaultCancellationAndStickySource`.
- Source profile CPython binary64 float truthiness is used in Boolean descriptors, paginated, recursive dictionary filtering and numeric next; exact decimal equality is deliberately separate. Plain integers remain arbitrary precision and no wire bytes change. Baseline core3 profile behavior remains frozen unless explicitly opted in.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordFloatTruthinessKeepsWireAndExactComparisonBoundary`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordBooleanDescriptorUsesFloatTruthiness`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsRulesAndTenantLocalFilterSemantics`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyRecordsFalseyNumericNextUsesSourceTruthiness`, `internal/neutronfind/find_policy_test.go:TestSourceProfileOwnsFloatTruthinessAndFindVersionPhasesWithoutChangingBaseline`, `internal/neutronfind/find_policy_test.go:TestSourceProfileNumericNextTruthinessDoesNotChangeBaselinePager`.
- Original selected Header version aliases are normalized only in private source/policy snapshots, safe list Accept and foreign/reserved headers remain fixed across retries. Clean503 permits exactly a second request and native live reauth changes token without changing selected source or version facts.
  Tests: `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordLiveReauthenticationAndMicroversionOwnership`, `network/v2/extensions/qos/policies/python_records_find_contracts_test.go:TestPythonQoSPolicyFindRecordListAcceptAndRetryVersionPolicy`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyRecordsCanonicalVersionAliasSurvivesCleanRetriesAndPhases`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsOperationHeadersAndRetryOwnership`, `network/v2/extensions/qos/policies/python_records_list_contracts_test.go:TestPythonQoSPolicyListRecordsLiveReauthenticationAndSourceContext`.

Python/Go differences:

- The complete declared classification is now SDK-owned through concrete semantic and explicit Body options. id/name/project_id/bool/tag query fields remain server-only, while deprecated tenant_id and rules are local. The project response alias does not add a tenant query alias; raw WithQuery is an explicit wire extension. WithName retains its separate hint/local predicate, typed List and FindIdentity stay independent, and no caller builder/predicate is required.
- Individual semantic options are last-wins, bulk canonical presence wins over wire aliases and bulk replace/clear affects only semantic state. Only final selected semantic values are validated. Same-destination null/empty/equal collisions fail rather than permissive kwargs merging. Scalar query arrays become repeated values, null elements are omitted, booleans are lowercase, JSON-number spelling is retained and unknown semantic names/errors are discarded.
- Original rules/tenant JSON is compared only after native whole-page decoding. Raw matching distinguishes large precise rule numbers, but returned Rules remains the native float64 projection and has native finite-range errors. Null maps/values were already preserved in the native model. Known TenantID remains a strict string and missing/null uses a raw null condition distinct from a native empty string. Python list-descriptor wrapping of non-list rules is not imitated.
- JSON equality deliberately distinguishes bools from numbers and strings from numbers. Arrays compare full ordered length/elements, including complete nested objects. Recursive object filters keep the pinned empty-actual-object nonmatch; nonobject actual values return nonmatch rather than possible Python attribute errors. No Python string-to-number coercion is claimed for these local fields.
- Native 200/204/300, plural-links continuation and foreign URL following are preserved. Empty application/json 204 may fail EOF. Local caps count raw rows like the pinned implementation but do not infer a wire limit; first-page/break stop before continuation and All drops partial rows on errors. No same-origin/fixed-query or synthesized marker continuation policy is added.
- Raw selectors run only for consumed selected fields after native decoding. Body-lane consumed null rows are invalid while query-only null rows preserve the native zero model. This does not add Python Resource default/alias/unknown-response cache/dirty lifecycle, arbitrary Resource properties or Resource/session adapter injection. Source/manifest/native drift fails closed without enabling more identity resources or capabilities.
- Immutable PolicyRecord distinguishes library Resource view, literal received Wire and actual whole response receipt; mutable Python Resource dirty/cache/session/Connection-derived location behavior is tracked separately. Existing typed native APIs remain independent.
- Go kwargs values are JSON-domain snapshots, bulk maps have deterministic sorted-key order, max_items is a nonnegative Go integer and fixed literal routes/session/constructor/cache/envelope controls are preflight rejected.
- Local equality preserves exact decimal JSON values rather than CPython binary64 rounding/underflow/overflow, although scalar truthiness matches the Source JSON parse domain and bool/numeric equality is allowed. This is an explicit Go comparison boundary, not full Python numeric equality parity.
- HTTP200..399 is the accepted completion range; Source final<400 informational100..199 is not treated as success. All actual400..599 and physical handler causes are preserved.
- Source external/rewritten continuations are restricted to the selected same-origin collection/query; malformed or cyclic continuation is an actual-page failure instead of forwarding shared credentials or silently changing filters.
- Safe Header options are explicit Go conveniences. Find uses GET-only explicit microversion and captured default for fallback List; standalone List applies its version to every page. Python headers kwargs query-vs-header implicit phase consumption is not inferred.
- Stock Neutron27 get_policy allows owner project reader or shared policy (also admin); create/update/delete are ADMIN_OR_PROJECT_MANAGER with historic admin-only deprecated policy. This wave covers only read named declarations and proves no live deployment role authorization.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--identity", default="application-qos")
parser.add_argument("--project", default="")
args = parser.parse_args()
query = {"is_shared": False}
if args.project:
    query["project_id"] = args.project

with openstack.connect(cloud=args.cloud) as conn:
    policy = conn.network.find_qos_policy(
        args.identity, ignore_missing=False, **query
    )
    print(json.dumps(policy.to_dict(), indent=2))
    for policy in conn.network.qos_policies(**query):
        print(json.dumps(policy.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, mode, identity, project string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "find":
        options := []policies.RecordFindOption{
            policies.WithRecordFindIgnoreMissing(false),
            policies.WithRecordFindQuery("is_shared", false),
        }
        if project != "" {
            options = append(options, policies.WithRecordFindQuery("project_id", project))
        }
        record, err := network.QoSPolicies.FindRecord(ctx, identity, options...)
        if err != nil || record == nil { return nil, err }
        return record.Resource, nil
    case "list":
        options := []policies.RecordListOption{
            policies.WithRecordListQuery("is_shared", false),
        }
        if project != "" {
            options = append(options, policies.WithRecordListQuery("project_id", project))
        }
        records, err := network.QoSPolicies.AllRecords(ctx, options...)
        if err != nil { return nil, err }
        values := make([]*resource.RawResource, len(records))
        for i, record := range records { values[i] = record.Resource }
        return values, nil
    default:
        return nil, fmt.Errorf("unknown mode %q", mode)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    mode := flag.String("mode", "find", "find or list")
    identity := flag.String("identity", "application-qos", "exact policy ID or name")
    project := flag.String("project", "", "optional project filter")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *identity, *project)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## References

- [Neutron 27.0.0 QoS policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
