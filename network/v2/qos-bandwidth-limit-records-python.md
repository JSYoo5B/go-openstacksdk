# QoS Bandwidth Limit Records Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.FindBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitFindHeaders`

- GET-first uses literal escaped parent and rule, fixed qos_policy_id in member query, raw original kwargs and constructor Body seeds. Only clean400/403/404 fallback to same parent; no default name server hint. Default logical absence is nil and explicit ignore_missing false returns ErrNotFound only after complete successful search.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitGetNoFallbackAndFindMissingCompleteSearch`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitFindBodySeedsAndListLocalFilters`.
- Exact ID/name list search sees later duplicates/errors before success. Parent URI seed is excluded from Body; actual member unknown qos_policy_id is projected through fixed URI while Wire keeps raw receipt. Only limit/marker are server query fields and declared Body kwargs are local filters.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitFindLaterDuplicateAndListFailureRemainTerminal`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitFindBodySeedsAndListLocalFilters`.
- Find per-call microversion governs member only and fallback retains captured selected default including canonical selected header alias. Clean native retry succeeds and unsafe retry request/header/version/status mutations preserve original proof.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitVersionPhasesCleanRetryAndOwnedHeaderPolicy`.
- Exact class Resource MRO has untyped max_kbps/max_burst_kbps/direction plus inherited id/name, all missing null, unknown received Body retained, no invented TagMixin/NetworkResource Header/default descriptors. Actual literal row, parent URI and physical response snapshots remain separate.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitListLazyQueryLocalFiltersAndUnknowns`.
- Final200..399 completion and every actual400..599 native rejection preserve status/body/header. Accepted malformed shape, Read/Close/custom cancellation/captured Source faults preserve actual receipt and causes; one physical Close. Faulty404 never becomes clean fallback.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitPhysicalReadCloseCancelAndSourceProof`.
- Concrete options snapshot JSON/maps/pointers/callback outputs, capture fixed selected Source before callbacks, reject nil/malformed parent/proxy/constructor/route controls before HTTP. Safe owned headers/version do not mutate the selected client or enable retry route/status/auth/version changes.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitVersionPhasesCleanRetryAndOwnedHeaderPolicy`.

Python/Go differences:

- Parent-first Go argument order follows existing native rule APIs. Parent policy is a literal ID; Python Resource instance/dirty/cache/session/Connection lifecycles are represented by immutable record snapshots, not fully reimplemented. QoSPolicyID exposes URI state separately from the default Body/to_dict view.
- JSON-domain options and deterministic sorted bulk map keys replace arbitrary Python kwargs/runtime objects. Source default QueryParameters only limit/marker; max_kbps/max_burst_kbps/direction/id/name local filtering differs from native upstream rule LIST query builders.
- Local numeric equality uses shared exact-decimal JSON, with bool-number equality and CPython float truthiness/falsey next handling. All CPython binary64 rounding/nonfinite/custom-object comparisons are not claimed.
- HTTP200..399 is the completion success range; Source final<400 informational100..199 is outside this profile. Actual400..599 and accepted/rejected physical causes stay observable.
- Paging restricts same origin, fixed parent collection and original filters; exact versioned-path alias only. Dictionary continuation links are a Go compatibility extension, and rewritten/foreign Source continuations are errors with actual proof.
- Safe Header options and Get Microversion options are explicit Go conveniences absent from the public pinned get proxy. Find per-call version applies to GET only; fallback keeps captured selected default and standalone List version controls all pages.
- Stock Neutron27 get_policy_bandwidth_limit_rule is project-scoped ADMIN_OR_PARENT_OWNER_READER. Shared QoSPolicy visibility does not alone grant rule read access; fixture403 preservation does not prove live deployment authorization. Other rule kinds/Create/Update/Delete are excluded from this wave.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

### `python:network/v2/get_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitGetHeaders`

- Exact public get proxy constructs literal parent URI without a parent fetch or member parent query, performs one rule GET, seeds only identity, and returns every actual member error without fallback or ignore-missing. Get has no Source query kwargs; safe Header/version convenience is explicit Go extension.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitGetNoFallbackAndFindMissingCompleteSearch`.
- URI qos_policy_id is outside default to_dict unless member response introduces an unknown Body key, then Source getattr(URI) projects the bound parent. Literal raw unknown values, untyped defaults and actual response metadata remain separately owned.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitMemberLiteralRoutesSourceQueryAndURIView`.
- Exact class Resource MRO has untyped max_kbps/max_burst_kbps/direction plus inherited id/name, all missing null, unknown received Body retained, no invented TagMixin/NetworkResource Header/default descriptors. Actual literal row, parent URI and physical response snapshots remain separate.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitListLazyQueryLocalFiltersAndUnknowns`.
- Final200..399 completion and every actual400..599 native rejection preserve status/body/header. Accepted malformed shape, Read/Close/custom cancellation/captured Source faults preserve actual receipt and causes; one physical Close. Faulty404 never becomes clean fallback.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitPhysicalReadCloseCancelAndSourceProof`.
- Concrete options snapshot JSON/maps/pointers/callback outputs, capture fixed selected Source before callbacks, reject nil/malformed parent/proxy/constructor/route controls before HTTP. Safe owned headers/version do not mutate the selected client or enable retry route/status/auth/version changes.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitVersionPhasesCleanRetryAndOwnedHeaderPolicy`.

Python/Go differences:

- Parent-first Go argument order follows existing native rule APIs. Parent policy is a literal ID; Python Resource instance/dirty/cache/session/Connection lifecycles are represented by immutable record snapshots, not fully reimplemented. QoSPolicyID exposes URI state separately from the default Body/to_dict view.
- JSON-domain options and deterministic sorted bulk map keys replace arbitrary Python kwargs/runtime objects. Source default QueryParameters only limit/marker; max_kbps/max_burst_kbps/direction/id/name local filtering differs from native upstream rule LIST query builders.
- Local numeric equality uses shared exact-decimal JSON, with bool-number equality and CPython float truthiness/falsey next handling. All CPython binary64 rounding/nonfinite/custom-object comparisons are not claimed.
- HTTP200..399 is the completion success range; Source final<400 informational100..199 is outside this profile. Actual400..599 and accepted/rejected physical causes stay observable.
- Paging restricts same origin, fixed parent collection and original filters; exact versioned-path alias only. Dictionary continuation links are a Go compatibility extension, and rewritten/foreign Source continuations are errors with actual proof.
- Safe Header options and Get Microversion options are explicit Go conveniences absent from the public pinned get proxy. Find per-call version applies to GET only; fallback keeps captured selected default and standalone List version controls all pages.
- Stock Neutron27 get_policy_bandwidth_limit_rule is project-scoped ADMIN_OR_PARENT_OWNER_READER. Shared QoSPolicy visibility does not alone grant rule read access; fixture403 preservation does not prove live deployment authorization. Other rule kinds/Create/Update/Delete are excluded from this wave.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

### `python:network/v2/qos_bandwidth_limit_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ListBandwidthLimitRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.AllBandwidthLimitRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListPaginated`

- Lazy list uses fixed literal parent without lookup. Only default limit/marker transpose to server query; Body id/name/direction/bandwidth fields filter locally. Unknown kwargs including fields/tags are discarded before transport-value evaluation, even function/malformed unknown values.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitListRawCapSinglePageAndEarlyStop`.
- Source list injects URI parent then consumes it outside Body; Wire owns foreign original row value and whole page proof. Positive raw cap counts before filtering, limit=0 plus cap creates limit hint; single page and early break avoid another request.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitListRawCapSinglePageAndEarlyStop`.
- Complete pager handles short-page ID marker fallback, relative and exact versioned continuations, Source falsey float next and malformed truthy next proofs. Parent/origin/filter retargeting fails with actual page evidence; iterator yields earlier rows but All drops partial rows after late failure.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitListContinuationCannotChangeParentOrFilters`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitListMarkerFallbackAndFalseyNumericNext`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitListPartialIterationAndAllDropLateError`.
- Exact class Resource MRO has untyped max_kbps/max_burst_kbps/direction plus inherited id/name, all missing null, unknown received Body retained, no invented TagMixin/NetworkResource Header/default descriptors. Actual literal row, parent URI and physical response snapshots remain separate.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitListLazyQueryLocalFiltersAndUnknowns`.
- Final200..399 completion and every actual400..599 native rejection preserve status/body/header. Accepted malformed shape, Read/Close/custom cancellation/captured Source faults preserve actual receipt and causes; one physical Close. Faulty404 never becomes clean fallback.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitPhysicalReadCloseCancelAndSourceProof`.
- Concrete options snapshot JSON/maps/pointers/callback outputs, capture fixed selected Source before callbacks, reject nil/malformed parent/proxy/constructor/route controls before HTTP. Safe owned headers/version do not mutate the selected client or enable retry route/status/auth/version changes.
  Tests: `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_bandwidth_limit_records_contracts_test.go:TestPythonBandwidthLimitVersionPhasesCleanRetryAndOwnedHeaderPolicy`.

Python/Go differences:

- Parent-first Go argument order follows existing native rule APIs. Parent policy is a literal ID; Python Resource instance/dirty/cache/session/Connection lifecycles are represented by immutable record snapshots, not fully reimplemented. QoSPolicyID exposes URI state separately from the default Body/to_dict view.
- JSON-domain options and deterministic sorted bulk map keys replace arbitrary Python kwargs/runtime objects. Source default QueryParameters only limit/marker; max_kbps/max_burst_kbps/direction/id/name local filtering differs from native upstream rule LIST query builders.
- Local numeric equality uses shared exact-decimal JSON, with bool-number equality and CPython float truthiness/falsey next handling. All CPython binary64 rounding/nonfinite/custom-object comparisons are not claimed.
- HTTP200..399 is the completion success range; Source final<400 informational100..199 is outside this profile. Actual400..599 and accepted/rejected physical causes stay observable.
- Paging restricts same origin, fixed parent collection and original filters; exact versioned-path alias only. Dictionary continuation links are a Go compatibility extension, and rewritten/foreign Source continuations are errors with actual proof.
- Safe Header options and Get Microversion options are explicit Go conveniences absent from the public pinned get proxy. Find per-call version applies to GET only; fallback keeps captured selected default and standalone List version controls all pages.
- Stock Neutron27 get_policy_bandwidth_limit_rule is project-scoped ADMIN_OR_PARENT_OWNER_READER. Shared QoSPolicy visibility does not alone grant rule read access; fixture403 preservation does not prove live deployment authorization. Other rule kinds/Create/Update/Delete are excluded from this wave.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

### `python:cloud/_network/NetworkCloudMixin/list_qos_bandwidth_limit_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListQoSBandwidthLimitRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyBandwidthLimitRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitProxyQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListBasePath`

- Complete QoS extension gate then exactly one parent policy resolution, owned untyped defaults/unknown fields/computed-null location, bound URI parent and local Body filtering with shared actual paging.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthListOwnsGateParentPagingAndRawFields`.
- Extension gate consumes every alias including later unhashable alias or failed page after an early QoS match; absence has typed unavailable extension error and no policy/callback/rule request.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`.
- Policy missing and ambiguity prevent rules/callback/filter expansion, preserve real extension inventory and typed missing/ambiguity without inventing policy None HTTP evidence.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPolicyMissingAndAmbiguityRetainActualGate`.
- Source proxy generator JMESPath literals/Unicode/object keys/nested generator/null/type errors distinguish rule HTTP from completed prerequisite HTTP; owned base_path succeeds while existing typed raw base_path rejection stays unchanged.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`.
- CPython falsey filters, truthy nonobject pre-rule errors, initialized owned callback maps, joined option error/cancellation or original captured-source mutation and real prerequisite receipts.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthFilterOwnershipCallbackGuardAndFalseyCases`.
- Actual truncated401 rule response keeps native reauthentication pointer/UnexpectedResponseCode/body/header/EOF, stops willing retry/reauth callbacks; late failed page preserves only earlier actual Inventory and completed parent/extensions.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`.

Python/Go differences:

- Concrete JSON/context/captured-client Go APIs and protected route/header policy replace dynamic Python session/Resource class/constructor/callable substitutions; arbitrary Python-only objects are not accepted.
- SourceProxy lane accepts100..399 with Source loaded local equality and computed location:null. Existing typed list/Get record status200..399 and typed raw base_path rejection remain compatibility APIs; concrete typed BasePath is additive.
- JMESPath literal object key iteration uses Go lexical order vs Python insertion order; existing strict syntax/exact JSON numeric evaluator/custom Resource function type policies remain explicit differences.
- Literal and nested generator values never acquire fake BandwidthLimitRecord or HTTP receipts. Actual extension/policy/rule proofs and observed partial inventory are retained with Go errors.
- Standalone Service computed location is Source null, including ignoring raw location in logical view. Full Connection computed location and mutable Resource/session model are tracked shared SDK-C1/R1 scope.
- Source policy and extension/user authorization are checked by the real Neutron deployment; these are read-only named helpers and no creation/update/delete or server-side policy emulation is added.

### `python:cloud/_network/NetworkCloudMixin/search_qos_bandwidth_limit_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchQoSBandwidthLimitRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyBandwidthLimitRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitProxyQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListBasePath`

- Complete QoS extension gate then exactly one parent policy resolution, owned untyped defaults/unknown fields/computed-null location, bound URI parent and local Body filtering with shared actual paging.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthListOwnsGateParentPagingAndRawFields`.
- Search delegates exactly one complete list workflow, preserves identifier/glob selection and CPython loaded bool/integer local equality, applies original ordered dictionary filter after list and retains actual Inventory/proof on unknown-key failure.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSearchKeepsRuleIDPhaseAndFilterPresence`.
- Extension gate consumes every alias including later unhashable alias or failed page after an early QoS match; absence has typed unavailable extension error and no policy/callback/rule request.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`.
- Policy missing and ambiguity prevent rules/callback/filter expansion, preserve real extension inventory and typed missing/ambiguity without inventing policy None HTTP evidence.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPolicyMissingAndAmbiguityRetainActualGate`.
- Source proxy generator JMESPath literals/Unicode/object keys/nested generator/null/type errors distinguish rule HTTP from completed prerequisite HTTP; owned base_path succeeds while existing typed raw base_path rejection stays unchanged.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`.
- CPython falsey filters, truthy nonobject pre-rule errors, initialized owned callback maps, joined option error/cancellation or original captured-source mutation and real prerequisite receipts.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthFilterOwnershipCallbackGuardAndFalseyCases`.
- Actual truncated401 rule response keeps native reauthentication pointer/UnexpectedResponseCode/body/header/EOF, stops willing retry/reauth callbacks; late failed page preserves only earlier actual Inventory and completed parent/extensions.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`.
- Source search without identifier skips falsey literal values; nested generator membership consumes actual rule records before unknown-key failure, while identifier phase rejects opaque generator without rule HTTP.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSearchSourceFalseyLiteralsAndNestedMembership`.

Python/Go differences:

- Concrete JSON/context/captured-client Go APIs and protected route/header policy replace dynamic Python session/Resource class/constructor/callable substitutions; arbitrary Python-only objects are not accepted.
- SourceProxy lane accepts100..399 with Source loaded local equality and computed location:null. Existing typed list/Get record status200..399 and typed raw base_path rejection remain compatibility APIs; concrete typed BasePath is additive.
- JMESPath literal object key iteration uses Go lexical order vs Python insertion order; existing strict syntax/exact JSON numeric evaluator/custom Resource function type policies remain explicit differences.
- Literal and nested generator values never acquire fake BandwidthLimitRecord or HTTP receipts. Actual extension/policy/rule proofs and observed partial inventory are retained with Go errors.
- Standalone Service computed location is Source null, including ignoring raw location in logical view. Full Connection computed location and mutable Resource/session model are tracked shared SDK-C1/R1 scope.
- Source policy and extension/user authorization are checked by the real Neutron deployment; these are read-only named helpers and no creation/update/delete or server-side policy emulation is added.

### `python:cloud/_network/NetworkCloudMixin/get_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetQoSBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitMicroversion`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Literal rule getter gates and resolves policy once, returns raw untyped record/default location on success, and preserves native404 without rule Find fallback or ignored missing.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGetResolvesPolicyOnceAndKeeps404`.
- Extension gate consumes every alias including later unhashable alias or failed page after an early QoS match; absence has typed unavailable extension error and no policy/callback/rule request.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`.
- Policy missing and ambiguity prevent rules/callback/filter expansion, preserve real extension inventory and typed missing/ambiguity without inventing policy None HTTP evidence.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPolicyMissingAndAmbiguityRetainActualGate`.
- CPython falsey filters, truthy nonobject pre-rule errors, initialized owned callback maps, joined option error/cancellation or original captured-source mutation and real prerequisite receipts.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthFilterOwnershipCallbackGuardAndFalseyCases`.
- Actual truncated401 rule response keeps native reauthentication pointer/UnexpectedResponseCode/body/header/EOF, stops willing retry/reauth callbacks; late failed page preserves only earlier actual Inventory and completed parent/extensions.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`.

Python/Go differences:

- Concrete JSON/context/captured-client Go APIs and protected route/header policy replace dynamic Python session/Resource class/constructor/callable substitutions; arbitrary Python-only objects are not accepted.
- SourceProxy lane accepts100..399 with Source loaded local equality and computed location:null. Existing typed list/Get record status200..399 and typed raw base_path rejection remain compatibility APIs; concrete typed BasePath is additive.
- JMESPath literal object key iteration uses Go lexical order vs Python insertion order; existing strict syntax/exact JSON numeric evaluator/custom Resource function type policies remain explicit differences.
- Literal and nested generator values never acquire fake BandwidthLimitRecord or HTTP receipts. Actual extension/policy/rule proofs and observed partial inventory are retained with Go errors.
- Standalone Service computed location is Source null, including ignoring raw location in logical view. Full Connection computed location and mutable Resource/session model are tracked shared SDK-C1/R1 scope.
- Source policy and extension/user authorization are checked by the real Neutron deployment; these are read-only named helpers and no creation/update/delete or server-side policy emulation is added.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--policy", required=True)
parser.add_argument("--rule", default="")
parser.add_argument("--mode", choices=("find", "get", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        rule = conn.network.find_qos_bandwidth_limit_rule(
            args.rule, args.policy, ignore_missing=False
        )
        print(json.dumps(rule.to_dict(), indent=2))
    elif args.mode == "get":
        rule = conn.network.get_qos_bandwidth_limit_rule(args.rule, args.policy)
        print(json.dumps(rule.to_dict(), indent=2))
    else:
        for rule in conn.network.qos_bandwidth_limit_rules(
            args.policy, direction="egress", max_items=100
        ):
            print(json.dumps(rule.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, mode, parent, identity string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "find":
        value, err := network.QoSRules.FindBandwidthLimitRecord(
            ctx, parent, identity, rules.WithBandwidthLimitFindIgnoreMissing(false),
        )
        if err != nil || value == nil { return nil, err }
        return value.Resource, nil
    case "get":
        value, err := network.QoSRules.GetBandwidthLimitRecord(ctx, parent, identity)
        if err != nil { return nil, err }
        return value.Resource, nil
    case "list":
        values, err := network.QoSRules.AllBandwidthLimitRecords(
            ctx, parent, rules.WithBandwidthLimitListQuery("direction", "egress"),
            rules.WithBandwidthLimitListMaxItems(100),
        )
        if err != nil { return nil, err }
        rows := make([]*resource.RawResource, 0, len(values))
        for _, value := range values { rows = append(rows, value.Resource) }
        return rows, nil
    default:
        return nil, fmt.Errorf("unknown mode %q", mode)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    parent := flag.String("policy", "", "literal QoS policy ID")
    identity := flag.String("rule", "", "literal rule ID or Find identity")
    mode := flag.String("mode", "list", "find, get or list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *parent, *identity)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L205)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L65)
