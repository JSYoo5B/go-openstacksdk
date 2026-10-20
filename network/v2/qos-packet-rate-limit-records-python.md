# QoS Packet Rate Limit Records Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_qos_packet_rate_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.FindPacketRateLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitFindHeaders`

- GET-first uses literal escaped parent and rule, fixed qos_policy_id in member query, raw original kwargs and constructor Body seeds. Only clean400/403/404 fallback to same parent; no default name server hint. Default logical absence is nil and explicit ignore_missing false returns ErrNotFound only after complete successful search.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitGetNoFallbackAndFindMissingCompleteSearch`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitFindBodySeedsAndListLocalFilters`.
- Exact ID/name list search sees later duplicates/errors before success. Parent URI seed is excluded from Body; actual member unknown qos_policy_id is projected through fixed URI while Wire keeps raw receipt. Only limit/marker are server query fields and declared Body kwargs are local filters.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitFindLaterDuplicateAndListFailureRemainTerminal`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitFindBodySeedsAndListLocalFilters`.
- Find per-call microversion governs member only and fallback retains captured selected default including canonical selected header alias. Clean native retry succeeds and unsafe retry request/header/version/status mutations preserve original proof.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitVersionPhasesCleanRetryAndOwnedHeaderPolicy`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitClean401LiveAuthExplicitHeadersAndVersionAbsence`.
- Exact class Resource MRO has untyped max_kpps/max_burst_kpps/direction plus inherited id/name, all missing null, unknown received Body retained, no invented TagMixin/NetworkResource Header/default descriptors. Actual literal row, parent URI and physical response snapshots remain separate. Captured callback nil maps normalize before subsequent concrete options; known JSON conversion adds ErrInvalidOption without losing original MarshalerError/caller cause, whereas unknown list kwargs discard before conversion. Combined accepted/rejected read/close/cancel/sticky Source failures suppress retry/reauth hooks before replay, preserve exactly1HTTP/1Close, and native401 ErrOriginal/ErrReauth retain independent actual receipt and all causes.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitOwnedNilMapsMarshalCausesAndCanceledPreflight`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMixedPhysicalFaultsStopCallbacksAndNative401Proof`.
- Final200..399 completion and every actual400..599 native rejection preserve status/body/header. Accepted malformed shape, Read/Close/custom cancellation/captured Source faults preserve actual receipt and causes; one physical Close. Faulty404 never becomes clean fallback. Combined accepted/rejected read/close/cancel/sticky Source failures suppress retry/reauth hooks before replay, preserve exactly1HTTP/1Close, and native401 ErrOriginal/ErrReauth retain independent actual receipt and all causes.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitPhysicalReadCloseCancelAndSourceProof`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMixedPhysicalFaultsStopCallbacksAndNative401Proof`.
- Concrete options snapshot JSON/maps/pointers/callback outputs, capture fixed selected Source before callbacks, reject nil/malformed parent/proxy/constructor/route controls before HTTP. Safe owned headers/version do not mutate the selected client or enable retry route/status/auth/version changes. Captured callback nil maps normalize before subsequent concrete options; known JSON conversion adds ErrInvalidOption without losing original MarshalerError/caller cause, whereas unknown list kwargs discard before conversion.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitVersionPhasesCleanRetryAndOwnedHeaderPolicy`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitOwnedNilMapsMarshalCausesAndCanceledPreflight`.
- Effective Source direct Resource MRO has computed location independently owned as null, including forged response location. Missing untyped defaults, arbitrary extension precision and independent Wire/Envelope survive; no NetworkResource/TagMixin descriptors are fabricated. Actual203 body evidence remains separate from Resource view.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitComputedLocationAndMissingDefaults`.
- Accepted203 malformed JSON or empty204 member response returns the constructed default/seeded view without a received Wire row, retaining actualEnvelope/header/status; valid wrong shapes remain receipt-backed errors. An actual member microversion extension projects the selected fetched version or null while preserving its raw Wire value.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListConstructorCollisionsAndMemberSelectedVersion`.

Python/Go differences:

- Parent-first Go argument order follows existing native rule APIs. Parent policy is a literal ID; Python Resource instance/dirty/cache/session/Connection lifecycles are represented by immutable record snapshots, not fully reimplemented. QoSPolicyID exposes URI state separately from the default Body/to_dict view.
- JSON-domain options and deterministic sorted bulk map keys replace arbitrary Python kwargs/runtime objects. Source default QueryParameters only limit/marker; max_kpps/max_burst_kpps/direction/id/name local filtering differs from native upstream rule LIST query builders.
- Local numeric equality uses shared exact-decimal JSON, with bool-number equality and CPython float truthiness/falsey next handling. All CPython binary64 rounding/nonfinite/custom-object comparisons are not claimed.
- HTTP200..399 is the completion success range; Source final<400 informational100..199 is outside this profile. Actual400..599 and accepted/rejected physical causes stay observable.
- Paging restricts same origin, fixed parent collection and original filters; exact versioned-path alias only. Dictionary continuation links are a Go compatibility extension, and rewritten/foreign Source continuations are errors with actual proof.
- Safe Header options and Get Microversion options are explicit Go conveniences absent from the public pinned get proxy. Find per-call version applies to GET only; fallback keeps captured selected default and standalone List version controls all pages.
- Stock Neutron27 get_policy_packet_rate_limit_rule is project-scoped ADMIN_OR_PARENT_OWNER_READER. Shared QoSPolicy visibility does not alone grant rule read access; fixture403 preservation does not prove live deployment authorization. Other rule kinds/Create/Update/Delete are excluded from this wave.
- Gophercloud v2.15.0 has no PacketRateLimitRule native typed functions or model. These callable SDK-owned record APIs reuse the established QoSRules selected client, shared neutronfind Engine and owned REST response/pager contracts rather than claiming generated/native wrappers for this family.
- Computed location is an immutable null SDK connection-view profile even when response JSON forges it. Full dynamic Connection.current_location and mutable Resource/dirty/subclass/session lifecycle are tracked separately; member microversion shadowing reflects actual selected fetch version, not the raw response extension.
- Pinned optional Proxy cache_enabled/skip_cache GET caching and shared invalidation remain SDK-C1. Every iteration observes fresh physical HTTP; full cache equivalence is not claimed by these finite named APIs.
- allow_unknown_params and deprecated jmespath_filters remain explicit rejected shared-foundation controls; arbitrary runtime objects and dynamic session/base_path are outside concrete JSON options. Known builder JSON failure preserves ErrInvalidOption plus original typed/caller cause.
- StopFaultRetryCallbacks is a separate true private profile flag, independent of SourceFindPolicy. Native ErrUnableToReauthenticate deliberately has no Unwrap; actual401 receipt lives in ErrOriginal and independent guarded physical/source/cancel cause in ErrReauth. Clean401 still uses live token without changing source client.

### `python:network/v2/get_qos_packet_rate_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetPacketRateLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitGetHeaders`

- Exact public get proxy constructs literal parent URI without a parent fetch or member parent query, performs one rule GET, seeds only identity, and returns every actual member error without fallback or ignore-missing. Get has no Source query kwargs; safe Header/version convenience is explicit Go extension.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitGetNoFallbackAndFindMissingCompleteSearch`.
- URI qos_policy_id is outside default to_dict unless member response introduces an unknown Body key, then Source getattr(URI) projects the bound parent. Literal raw unknown values, untyped defaults and actual response metadata remain separately owned.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMemberLiteralRoutesSourceQueryAndURIView`.
- Exact class Resource MRO has untyped max_kpps/max_burst_kpps/direction plus inherited id/name, all missing null, unknown received Body retained, no invented TagMixin/NetworkResource Header/default descriptors. Actual literal row, parent URI and physical response snapshots remain separate. Captured callback nil maps normalize before subsequent concrete options; known JSON conversion adds ErrInvalidOption without losing original MarshalerError/caller cause, whereas unknown list kwargs discard before conversion. Combined accepted/rejected read/close/cancel/sticky Source failures suppress retry/reauth hooks before replay, preserve exactly1HTTP/1Close, and native401 ErrOriginal/ErrReauth retain independent actual receipt and all causes.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitOwnedNilMapsMarshalCausesAndCanceledPreflight`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMixedPhysicalFaultsStopCallbacksAndNative401Proof`.
- Final200..399 completion and every actual400..599 native rejection preserve status/body/header. Accepted malformed shape, Read/Close/custom cancellation/captured Source faults preserve actual receipt and causes; one physical Close. Faulty404 never becomes clean fallback. Combined accepted/rejected read/close/cancel/sticky Source failures suppress retry/reauth hooks before replay, preserve exactly1HTTP/1Close, and native401 ErrOriginal/ErrReauth retain independent actual receipt and all causes.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitPhysicalReadCloseCancelAndSourceProof`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMixedPhysicalFaultsStopCallbacksAndNative401Proof`.
- Concrete options snapshot JSON/maps/pointers/callback outputs, capture fixed selected Source before callbacks, reject nil/malformed parent/proxy/constructor/route controls before HTTP. Safe owned headers/version do not mutate the selected client or enable retry route/status/auth/version changes. Captured callback nil maps normalize before subsequent concrete options; known JSON conversion adds ErrInvalidOption without losing original MarshalerError/caller cause, whereas unknown list kwargs discard before conversion.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitVersionPhasesCleanRetryAndOwnedHeaderPolicy`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitOwnedNilMapsMarshalCausesAndCanceledPreflight`.
- Effective Source direct Resource MRO has computed location independently owned as null, including forged response location. Missing untyped defaults, arbitrary extension precision and independent Wire/Envelope survive; no NetworkResource/TagMixin descriptors are fabricated. Actual203 body evidence remains separate from Resource view.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitComputedLocationAndMissingDefaults`.
- Accepted203 malformed JSON or empty204 member response returns the constructed default/seeded view without a received Wire row, retaining actualEnvelope/header/status; valid wrong shapes remain receipt-backed errors. An actual member microversion extension projects the selected fetched version or null while preserving its raw Wire value.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListConstructorCollisionsAndMemberSelectedVersion`.

Python/Go differences:

- Parent-first Go argument order follows existing native rule APIs. Parent policy is a literal ID; Python Resource instance/dirty/cache/session/Connection lifecycles are represented by immutable record snapshots, not fully reimplemented. QoSPolicyID exposes URI state separately from the default Body/to_dict view.
- JSON-domain options and deterministic sorted bulk map keys replace arbitrary Python kwargs/runtime objects. Source default QueryParameters only limit/marker; max_kpps/max_burst_kpps/direction/id/name local filtering differs from native upstream rule LIST query builders.
- Local numeric equality uses shared exact-decimal JSON, with bool-number equality and CPython float truthiness/falsey next handling. All CPython binary64 rounding/nonfinite/custom-object comparisons are not claimed.
- HTTP200..399 is the completion success range; Source final<400 informational100..199 is outside this profile. Actual400..599 and accepted/rejected physical causes stay observable.
- Paging restricts same origin, fixed parent collection and original filters; exact versioned-path alias only. Dictionary continuation links are a Go compatibility extension, and rewritten/foreign Source continuations are errors with actual proof.
- Safe Header options and Get Microversion options are explicit Go conveniences absent from the public pinned get proxy. Find per-call version applies to GET only; fallback keeps captured selected default and standalone List version controls all pages.
- Stock Neutron27 get_policy_packet_rate_limit_rule is project-scoped ADMIN_OR_PARENT_OWNER_READER. Shared QoSPolicy visibility does not alone grant rule read access; fixture403 preservation does not prove live deployment authorization. Other rule kinds/Create/Update/Delete are excluded from this wave.
- Gophercloud v2.15.0 has no PacketRateLimitRule native typed functions or model. These callable SDK-owned record APIs reuse the established QoSRules selected client, shared neutronfind Engine and owned REST response/pager contracts rather than claiming generated/native wrappers for this family.
- Computed location is an immutable null SDK connection-view profile even when response JSON forges it. Full dynamic Connection.current_location and mutable Resource/dirty/subclass/session lifecycle are tracked separately; member microversion shadowing reflects actual selected fetch version, not the raw response extension.
- Pinned optional Proxy cache_enabled/skip_cache GET caching and shared invalidation remain SDK-C1. Every iteration observes fresh physical HTTP; full cache equivalence is not claimed by these finite named APIs.
- allow_unknown_params and deprecated jmespath_filters remain explicit rejected shared-foundation controls; arbitrary runtime objects and dynamic session/base_path are outside concrete JSON options. Known builder JSON failure preserves ErrInvalidOption plus original typed/caller cause.
- StopFaultRetryCallbacks is a separate true private profile flag, independent of SourceFindPolicy. Native ErrUnableToReauthenticate deliberately has no Unwrap; actual401 receipt lives in ErrOriginal and independent guarded physical/source/cancel cause in ErrReauth. Clean401 still uses live token without changing source client.

### `python:network/v2/qos_packet_rate_limit_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ListPacketRateLimitRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.AllPacketRateLimitRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateLimitListPaginated`

- Lazy list uses fixed literal parent without lookup. Only default limit/marker transpose to server query; Body id/name/direction/bandwidth fields filter locally. Unknown kwargs including fields/tags are discarded before transport-value evaluation, even function/malformed unknown values.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListRawCapSinglePageAndEarlyStop`.
- Source list injects URI parent then consumes it outside Body; Wire owns foreign original row value and whole page proof. Positive raw cap counts before filtering, limit=0 plus cap creates limit hint; single page and early break avoid another request.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListRawCapSinglePageAndEarlyStop`.
- Complete pager handles short-page ID marker fallback, relative and exact versioned continuations, Source falsey float next and malformed truthy next proofs. Parent/origin/filter retargeting fails with actual page evidence; iterator yields earlier rows but All drops partial rows after late failure.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListContinuationCannotChangeParentOrFilters`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListMarkerFallbackAndFalseyNumericNext`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListPartialIterationAndAllDropLateError`.
- Exact class Resource MRO has untyped max_kpps/max_burst_kpps/direction plus inherited id/name, all missing null, unknown received Body retained, no invented TagMixin/NetworkResource Header/default descriptors. Actual literal row, parent URI and physical response snapshots remain separate. Captured callback nil maps normalize before subsequent concrete options; known JSON conversion adds ErrInvalidOption without losing original MarshalerError/caller cause, whereas unknown list kwargs discard before conversion. Combined accepted/rejected read/close/cancel/sticky Source failures suppress retry/reauth hooks before replay, preserve exactly1HTTP/1Close, and native401 ErrOriginal/ErrReauth retain independent actual receipt and all causes.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitOwnedNilMapsMarshalCausesAndCanceledPreflight`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMixedPhysicalFaultsStopCallbacksAndNative401Proof`.
- Final200..399 completion and every actual400..599 native rejection preserve status/body/header. Accepted malformed shape, Read/Close/custom cancellation/captured Source faults preserve actual receipt and causes; one physical Close. Faulty404 never becomes clean fallback. Combined accepted/rejected read/close/cancel/sticky Source failures suppress retry/reauth hooks before replay, preserve exactly1HTTP/1Close, and native401 ErrOriginal/ErrReauth retain independent actual receipt and all causes.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitPhysicalReadCloseCancelAndSourceProof`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitMixedPhysicalFaultsStopCallbacksAndNative401Proof`.
- Concrete options snapshot JSON/maps/pointers/callback outputs, capture fixed selected Source before callbacks, reject nil/malformed parent/proxy/constructor/route controls before HTTP. Safe owned headers/version do not mutate the selected client or enable retry route/status/auth/version changes. Captured callback nil maps normalize before subsequent concrete options; known JSON conversion adds ErrInvalidOption without losing original MarshalerError/caller cause, whereas unknown list kwargs discard before conversion.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitVersionPhasesCleanRetryAndOwnedHeaderPolicy`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitOwnedNilMapsMarshalCausesAndCanceledPreflight`.
- Effective Source direct Resource MRO has computed location independently owned as null, including forged response location. Missing untyped defaults, arbitrary extension precision and independent Wire/Envelope survive; no NetworkResource/TagMixin descriptors are fabricated. Actual203 body evidence remains separate from Resource view.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitComputedLocationAndMissingDefaults`.
- LIST only constructor duplicate cls/connection/microversion/_synchronized fails after preceding rows with whole actual203 page/header receipt; AllRecords discards partial rows and retains outer named operation.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListConstructorCollisionsAndMemberSelectedVersion`, `network/v2/extensions/qos/rules/python_packet_rate_limit_contracts_test.go:TestPythonPacketRateLimitListPartialIterationAndAllDropLateError`.

Python/Go differences:

- Parent-first Go argument order follows existing native rule APIs. Parent policy is a literal ID; Python Resource instance/dirty/cache/session/Connection lifecycles are represented by immutable record snapshots, not fully reimplemented. QoSPolicyID exposes URI state separately from the default Body/to_dict view.
- JSON-domain options and deterministic sorted bulk map keys replace arbitrary Python kwargs/runtime objects. Source default QueryParameters only limit/marker; max_kpps/max_burst_kpps/direction/id/name local filtering differs from native upstream rule LIST query builders.
- Local numeric equality uses shared exact-decimal JSON, with bool-number equality and CPython float truthiness/falsey next handling. All CPython binary64 rounding/nonfinite/custom-object comparisons are not claimed.
- HTTP200..399 is the completion success range; Source final<400 informational100..199 is outside this profile. Actual400..599 and accepted/rejected physical causes stay observable.
- Paging restricts same origin, fixed parent collection and original filters; exact versioned-path alias only. Dictionary continuation links are a Go compatibility extension, and rewritten/foreign Source continuations are errors with actual proof.
- Safe Header options and Get Microversion options are explicit Go conveniences absent from the public pinned get proxy. Find per-call version applies to GET only; fallback keeps captured selected default and standalone List version controls all pages.
- Stock Neutron27 get_policy_packet_rate_limit_rule is project-scoped ADMIN_OR_PARENT_OWNER_READER. Shared QoSPolicy visibility does not alone grant rule read access; fixture403 preservation does not prove live deployment authorization. Other rule kinds/Create/Update/Delete are excluded from this wave.
- Gophercloud v2.15.0 has no PacketRateLimitRule native typed functions or model. These callable SDK-owned record APIs reuse the established QoSRules selected client, shared neutronfind Engine and owned REST response/pager contracts rather than claiming generated/native wrappers for this family.
- Computed location is an immutable null SDK connection-view profile even when response JSON forges it. Full dynamic Connection.current_location and mutable Resource/dirty/subclass/session lifecycle are tracked separately; member microversion shadowing reflects actual selected fetch version, not the raw response extension.
- Pinned optional Proxy cache_enabled/skip_cache GET caching and shared invalidation remain SDK-C1. Every iteration observes fresh physical HTTP; full cache equivalence is not claimed by these finite named APIs.
- allow_unknown_params and deprecated jmespath_filters remain explicit rejected shared-foundation controls; arbitrary runtime objects and dynamic session/base_path are outside concrete JSON options. Known builder JSON failure preserves ErrInvalidOption plus original typed/caller cause.
- StopFaultRetryCallbacks is a separate true private profile flag, independent of SourceFindPolicy. Native ErrUnableToReauthenticate deliberately has no Unwrap; actual401 receipt lives in ErrOriginal and independent guarded physical/source/cancel cause in ErrReauth. Clean401 still uses live token without changing source client.

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
        rule = conn.network.find_qos_packet_rate_limit_rule(
            args.rule, args.policy, ignore_missing=False
        )
        print(json.dumps(rule.to_dict(), indent=2))
    elif args.mode == "get":
        rule = conn.network.get_qos_packet_rate_limit_rule(args.rule, args.policy)
        print(json.dumps(rule.to_dict(), indent=2))
    else:
        for rule in conn.network.qos_packet_rate_limit_rules(
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
        value, err := network.QoSRules.FindPacketRateLimitRecord(
            ctx, parent, identity, rules.WithPacketRateLimitFindIgnoreMissing(false),
        )
        if err != nil || value == nil { return nil, err }
        return value.Resource, nil
    case "get":
        value, err := network.QoSRules.GetPacketRateLimitRecord(ctx, parent, identity)
        if err != nil { return nil, err }
        return value.Resource, nil
    case "list":
        values, err := network.QoSRules.AllPacketRateLimitRecords(
            ctx, parent, rules.WithPacketRateLimitListQuery("direction", "egress"),
            rules.WithPacketRateLimitListMaxItems(100),
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

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L280)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L65)
