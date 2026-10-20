# QoS Minimum Packet Rate Records Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_qos_minimum_packet_rate_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.FindMinimumPacketRateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateFindHeaders`

- Literal parent-first one-segment routes; Get uses URI constructor only and no parent query. Find member query includes fixed qos_policy_id plus original kwargs, with None query omission and explicit-null/default constructor seed preservation.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateGetNoFallbackAndFindMissingCompleteSearch`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateNoneQuerySeedsAndLocalDescriptorFilters`.
- Direct Resource MRO: untyped direction/min_kpps/id/name with missing null, no Header/TagMixin/NetworkResource descriptors. Unknown extensions and null/false/zero/string/object values survive. Mutable Connection location is represented by null even if actual Wire forges location.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateSparseDefaultsAndForgedLocation`.
- URI qos_policy_id remains separate in QoSPolicyID, is excluded from default constructor/list to_dict Body, and projects bound parent only when actual member unknown Body introduces the key. Raw foreign/null parent and computed location stay in actual Wire/Envelope evidence.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateSparseDefaultsAndForgedLocation`.
- Only clean member400/403/404 permit same-parent full exact ID/name fallback; default logical absence nil and explicit IgnoreMissing false ErrNotFound follow entirely successful search. Later duplicate/list forbidden/late error remain terminal.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateGetNoFallbackAndFindMissingCompleteSearch`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindLaterDuplicateAndListFailureRemainTerminal`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListPartialIterationAndAllDropLateError`.
- Inherited server query is only limit/marker. Known Body kwargs filter locally; fields/tags/other unknown List kwargs are discarded before marshal/JSON validation. Source dictionary subset/falsey/truthy scalar semantics and bool-number equality preserve local filter behavior.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateNoneQuerySeedsAndLocalDescriptorFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListConstructorCollisionsAndDictionaryFilters`.
- Lazy complete pager owns parent/origin/filter scope, short-page marker and full raw-ID identity, falsey binary64 numeric next, raw cap before filters, explicit limit-zero cap hint, single page and early break. All drops earlier records on late error.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListRawCapSinglePageAndEarlyStop`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListContinuationCannotChangeParentOrFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListMarkerFallbackAndFalseyNumericNext`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListPartialIterationAndAllDropLateError`.
- Source constructor cls/null/false, connection/microversion/_synchronized collisions, malformed shape/JSON and actual invalid-UTF8 byte preserve full accepted page proof. List rejects malformed receipt; accepted bodyless member retains seed/defaults with Wire nil.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateNoneQuerySeedsAndLocalDescriptorFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListConstructorCollisionsAndDictionaryFilters`.
- Completed200..399 plus every actual400..599 native rejection retain HTTP status/header/body. Accepted/rejected Read/Close/context/custom-cancel/Source causes are observable; faulty404 does not trigger fallback.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRatePhysicalReadCloseCancelAndSourceProof`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRatePhysicalFaultsNeverInvokeRetryOrReauth`.
- StopFaultRetryCallbacks:true prevents RetryFunc and ReauthFunc after actual physical/Source/cancel faults, with exactly one physical request and Close for accepted/rejected statuses incl401/403/404/503/599; clean HTTP retry and live auth token refresh remain allowed.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRatePhysicalFaultsNeverInvokeRetryOrReauth`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateRetryVersionAbsenceAndLiveTokenOwnership`.
- Find explicit microversion affects member only; fallback uses captured selected service default. Get version is per-call and standalone List override applies all pages; captured matching header aliases/default Accept are owned without selected-client mutation.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateVersionPhasesCleanRetryAndOwnedHeaderPolicy`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateRetryVersionAbsenceAndLiveTokenOwnership`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateExplicitListVersionAppliesToEveryPage`.
- Concrete owned options snapshot values/maps/pointers/callback outputs before HTTP and support concurrent reuse with independent record/receipt snapshots. Nil/parent/control/Source mutation preflight errors avoid HTTP. Native retry cannot inject auth/framing/foreign/version/body changes, including originally absent version.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateVersionPhasesCleanRetryAndOwnedHeaderPolicy`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateRetryVersionAbsenceAndLiveTokenOwnership`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateReusableOptionsConcurrentOwnedViews`.

Python/Go differences:

- Go accepts literal parent policy IDs in parent-first order; Python Resource instance mutation/cache/Adapter/session and full Connection computed location lifecycles remain separately tracked. QoSPolicyID exposes URI state outside Body/to_dict.
- JSON-domain options and deterministic bulk-map ordering replace arbitrary Python kwargs/runtime values. Deprecated Proxy JMESPath and __conflicting_attrs restoration are global remaining operations, not completed by this record leaf.
- Known server query only limit/marker; concrete Source local Body filters differ from native rule query builders. Negative/invalid limits/caps and SDK-owned constructor/route controls are rejected before HTTP.
- CPython binary64 truthiness and bool-number equality are followed, but ordinary local equality deliberately uses exact-decimal JSON. All CPython float rounding/nonfinite/custom-object comparisons are not claimed.
- Completion status200..399 is supported; final informational100..199 from arbitrary adapters remains outside this Go completion profile. Actual400..599 and all physical/Source/cancel causes preserve receipt.
- Paging requires same origin, same fixed parent collection and original filters; rewritten/foreign continuations fail with actual page evidence. Dictionary continuation links and exact versioned route alias are explicit Go conveniences.
- Safe headers and Get Microversion controls are explicit Go conveniences absent from pinned public Get kwargs. Empty explicit version omits the version header; Find GET override and captured fallback default are distinct phases.
- Stock Neutron27 get_policy_minimum_packet_rate_rule permits project admin or parent-policy-owner reader. Shared-policy visibility does not itself grant rule access; actual deployment authorization/extensions/backend acceptance is not proven by fixtures.

### `python:network/v2/get_qos_minimum_packet_rate_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetMinimumPacketRateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateGetHeaders`

- Literal parent-first one-segment routes; Get uses URI constructor only and no parent query. Find member query includes fixed qos_policy_id plus original kwargs, with None query omission and explicit-null/default constructor seed preservation.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateGetNoFallbackAndFindMissingCompleteSearch`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateNoneQuerySeedsAndLocalDescriptorFilters`.
- Direct Resource MRO: untyped direction/min_kpps/id/name with missing null, no Header/TagMixin/NetworkResource descriptors. Unknown extensions and null/false/zero/string/object values survive. Mutable Connection location is represented by null even if actual Wire forges location.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateSparseDefaultsAndForgedLocation`.
- URI qos_policy_id remains separate in QoSPolicyID, is excluded from default constructor/list to_dict Body, and projects bound parent only when actual member unknown Body introduces the key. Raw foreign/null parent and computed location stay in actual Wire/Envelope evidence.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateSparseDefaultsAndForgedLocation`.
- Only clean member400/403/404 permit same-parent full exact ID/name fallback; default logical absence nil and explicit IgnoreMissing false ErrNotFound follow entirely successful search. Later duplicate/list forbidden/late error remain terminal.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateGetNoFallbackAndFindMissingCompleteSearch`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindLaterDuplicateAndListFailureRemainTerminal`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListPartialIterationAndAllDropLateError`.
- Inherited server query is only limit/marker. Known Body kwargs filter locally; fields/tags/other unknown List kwargs are discarded before marshal/JSON validation. Source dictionary subset/falsey/truthy scalar semantics and bool-number equality preserve local filter behavior.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateNoneQuerySeedsAndLocalDescriptorFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListConstructorCollisionsAndDictionaryFilters`.
- Lazy complete pager owns parent/origin/filter scope, short-page marker and full raw-ID identity, falsey binary64 numeric next, raw cap before filters, explicit limit-zero cap hint, single page and early break. All drops earlier records on late error.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListRawCapSinglePageAndEarlyStop`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListContinuationCannotChangeParentOrFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListMarkerFallbackAndFalseyNumericNext`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListPartialIterationAndAllDropLateError`.
- Source constructor cls/null/false, connection/microversion/_synchronized collisions, malformed shape/JSON and actual invalid-UTF8 byte preserve full accepted page proof. List rejects malformed receipt; accepted bodyless member retains seed/defaults with Wire nil.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateNoneQuerySeedsAndLocalDescriptorFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListConstructorCollisionsAndDictionaryFilters`.
- Completed200..399 plus every actual400..599 native rejection retain HTTP status/header/body. Accepted/rejected Read/Close/context/custom-cancel/Source causes are observable; faulty404 does not trigger fallback.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRatePhysicalReadCloseCancelAndSourceProof`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRatePhysicalFaultsNeverInvokeRetryOrReauth`.
- StopFaultRetryCallbacks:true prevents RetryFunc and ReauthFunc after actual physical/Source/cancel faults, with exactly one physical request and Close for accepted/rejected statuses incl401/403/404/503/599; clean HTTP retry and live auth token refresh remain allowed.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRatePhysicalFaultsNeverInvokeRetryOrReauth`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateRetryVersionAbsenceAndLiveTokenOwnership`.
- Find explicit microversion affects member only; fallback uses captured selected service default. Get version is per-call and standalone List override applies all pages; captured matching header aliases/default Accept are owned without selected-client mutation.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateVersionPhasesCleanRetryAndOwnedHeaderPolicy`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateRetryVersionAbsenceAndLiveTokenOwnership`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateExplicitListVersionAppliesToEveryPage`.
- Concrete owned options snapshot values/maps/pointers/callback outputs before HTTP and support concurrent reuse with independent record/receipt snapshots. Nil/parent/control/Source mutation preflight errors avoid HTTP. Native retry cannot inject auth/framing/foreign/version/body changes, including originally absent version.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateVersionPhasesCleanRetryAndOwnedHeaderPolicy`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateRetryVersionAbsenceAndLiveTokenOwnership`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateReusableOptionsConcurrentOwnedViews`.

Python/Go differences:

- Go accepts literal parent policy IDs in parent-first order; Python Resource instance mutation/cache/Adapter/session and full Connection computed location lifecycles remain separately tracked. QoSPolicyID exposes URI state outside Body/to_dict.
- JSON-domain options and deterministic bulk-map ordering replace arbitrary Python kwargs/runtime values. Deprecated Proxy JMESPath and __conflicting_attrs restoration are global remaining operations, not completed by this record leaf.
- Known server query only limit/marker; concrete Source local Body filters differ from native rule query builders. Negative/invalid limits/caps and SDK-owned constructor/route controls are rejected before HTTP.
- CPython binary64 truthiness and bool-number equality are followed, but ordinary local equality deliberately uses exact-decimal JSON. All CPython float rounding/nonfinite/custom-object comparisons are not claimed.
- Completion status200..399 is supported; final informational100..199 from arbitrary adapters remains outside this Go completion profile. Actual400..599 and all physical/Source/cancel causes preserve receipt.
- Paging requires same origin, same fixed parent collection and original filters; rewritten/foreign continuations fail with actual page evidence. Dictionary continuation links and exact versioned route alias are explicit Go conveniences.
- Safe headers and Get Microversion controls are explicit Go conveniences absent from pinned public Get kwargs. Empty explicit version omits the version header; Find GET override and captured fallback default are distinct phases.
- Stock Neutron27 get_policy_minimum_packet_rate_rule permits project admin or parent-policy-owner reader. Shared-policy visibility does not itself grant rule access; actual deployment authorization/extensions/backend acceptance is not proven by fixtures.

### `python:network/v2/qos_minimum_packet_rate_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ListMinimumPacketRateRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.AllMinimumPacketRateRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumPacketRateListPaginated`

- Literal parent-first one-segment routes; Get uses URI constructor only and no parent query. Find member query includes fixed qos_policy_id plus original kwargs, with None query omission and explicit-null/default constructor seed preservation.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateGetNoFallbackAndFindMissingCompleteSearch`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateNoneQuerySeedsAndLocalDescriptorFilters`.
- Direct Resource MRO: untyped direction/min_kpps/id/name with missing null, no Header/TagMixin/NetworkResource descriptors. Unknown extensions and null/false/zero/string/object values survive. Mutable Connection location is represented by null even if actual Wire forges location.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateSparseDefaultsAndForgedLocation`.
- URI qos_policy_id remains separate in QoSPolicyID, is excluded from default constructor/list to_dict Body, and projects bound parent only when actual member unknown Body introduces the key. Raw foreign/null parent and computed location stay in actual Wire/Envelope evidence.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateSparseDefaultsAndForgedLocation`.
- Only clean member400/403/404 permit same-parent full exact ID/name fallback; default logical absence nil and explicit IgnoreMissing false ErrNotFound follow entirely successful search. Later duplicate/list forbidden/late error remain terminal.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateGetNoFallbackAndFindMissingCompleteSearch`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindLaterDuplicateAndListFailureRemainTerminal`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListPartialIterationAndAllDropLateError`.
- Inherited server query is only limit/marker. Known Body kwargs filter locally; fields/tags/other unknown List kwargs are discarded before marshal/JSON validation. Source dictionary subset/falsey/truthy scalar semantics and bool-number equality preserve local filter behavior.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateNoneQuerySeedsAndLocalDescriptorFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListConstructorCollisionsAndDictionaryFilters`.
- Lazy complete pager owns parent/origin/filter scope, short-page marker and full raw-ID identity, falsey binary64 numeric next, raw cap before filters, explicit limit-zero cap hint, single page and early break. All drops earlier records on late error.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListRawCapSinglePageAndEarlyStop`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListContinuationCannotChangeParentOrFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListMarkerFallbackAndFalseyNumericNext`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListPartialIterationAndAllDropLateError`.
- Source constructor cls/null/false, connection/microversion/_synchronized collisions, malformed shape/JSON and actual invalid-UTF8 byte preserve full accepted page proof. List rejects malformed receipt; accepted bodyless member retains seed/defaults with Wire nil.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateNoneQuerySeedsAndLocalDescriptorFilters`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateListConstructorCollisionsAndDictionaryFilters`.
- Completed200..399 plus every actual400..599 native rejection retain HTTP status/header/body. Accepted/rejected Read/Close/context/custom-cancel/Source causes are observable; faulty404 does not trigger fallback.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRatePhysicalReadCloseCancelAndSourceProof`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRatePhysicalFaultsNeverInvokeRetryOrReauth`.
- StopFaultRetryCallbacks:true prevents RetryFunc and ReauthFunc after actual physical/Source/cancel faults, with exactly one physical request and Close for accepted/rejected statuses incl401/403/404/503/599; clean HTTP retry and live auth token refresh remain allowed.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRatePhysicalFaultsNeverInvokeRetryOrReauth`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateRetryVersionAbsenceAndLiveTokenOwnership`.
- Find explicit microversion affects member only; fallback uses captured selected service default. Get version is per-call and standalone List override applies all pages; captured matching header aliases/default Accept are owned without selected-client mutation.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateVersionPhasesCleanRetryAndOwnedHeaderPolicy`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateRetryVersionAbsenceAndLiveTokenOwnership`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateExplicitListVersionAppliesToEveryPage`.
- Concrete owned options snapshot values/maps/pointers/callback outputs before HTTP and support concurrent reuse with independent record/receipt snapshots. Nil/parent/control/Source mutation preflight errors avoid HTTP. Native retry cannot inject auth/framing/foreign/version/body changes, including originally absent version.
  Tests: `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateVersionPhasesCleanRetryAndOwnedHeaderPolicy`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateRetryVersionAbsenceAndLiveTokenOwnership`, `network/v2/extensions/qos/rules/python_minimum_packet_rate_contracts_test.go:TestPythonMinimumPacketRateReusableOptionsConcurrentOwnedViews`.

Python/Go differences:

- Go accepts literal parent policy IDs in parent-first order; Python Resource instance mutation/cache/Adapter/session and full Connection computed location lifecycles remain separately tracked. QoSPolicyID exposes URI state outside Body/to_dict.
- JSON-domain options and deterministic bulk-map ordering replace arbitrary Python kwargs/runtime values. Deprecated Proxy JMESPath and __conflicting_attrs restoration are global remaining operations, not completed by this record leaf.
- Known server query only limit/marker; concrete Source local Body filters differ from native rule query builders. Negative/invalid limits/caps and SDK-owned constructor/route controls are rejected before HTTP.
- CPython binary64 truthiness and bool-number equality are followed, but ordinary local equality deliberately uses exact-decimal JSON. All CPython float rounding/nonfinite/custom-object comparisons are not claimed.
- Completion status200..399 is supported; final informational100..199 from arbitrary adapters remains outside this Go completion profile. Actual400..599 and all physical/Source/cancel causes preserve receipt.
- Paging requires same origin, same fixed parent collection and original filters; rewritten/foreign continuations fail with actual page evidence. Dictionary continuation links and exact versioned route alias are explicit Go conveniences.
- Safe headers and Get Microversion controls are explicit Go conveniences absent from pinned public Get kwargs. Empty explicit version omits the version header; Find GET override and captured fallback default are distinct phases.
- Stock Neutron27 get_policy_minimum_packet_rate_rule permits project admin or parent-policy-owner reader. Shared-policy visibility does not itself grant rule access; actual deployment authorization/extensions/backend acceptance is not proven by fixtures.

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
parser.add_argument("--mode", choices=("get", "find", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "get":
        value = conn.network.get_qos_minimum_packet_rate_rule(args.rule, args.policy)
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "find":
        value = conn.network.find_qos_minimum_packet_rate_rule(
            args.rule, args.policy, ignore_missing=False
        )
        print(json.dumps(value.to_dict(), indent=2))
    else:
        for value in conn.network.qos_minimum_packet_rate_rules(
            args.policy, direction="any", max_items=100
        ):
            print(json.dumps(value.to_dict(), indent=2))
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

func run(ctx context.Context, cloud, mode, policyID, identity string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "get":
        value, err := network.QoSRules.GetMinimumPacketRateRecord(ctx, policyID, identity)
        if err != nil { return nil, err }
        return value.Resource, nil
    case "find":
        value, err := network.QoSRules.FindMinimumPacketRateRecord(
            ctx, policyID, identity, rules.WithMinimumPacketRateFindIgnoreMissing(false),
        )
        if err != nil || value == nil { return nil, err }
        return value.Resource, nil
    case "list":
        values, err := network.QoSRules.AllMinimumPacketRateRecords(
            ctx, policyID, rules.WithMinimumPacketRateListQuery("direction", "any"),
            rules.WithMinimumPacketRateListMaxItems(100),
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
    policyID := flag.String("policy", "", "literal QoS policy ID")
    identity := flag.String("rule", "", "literal rule ID or Find identity")
    mode := flag.String("mode", "list", "get, find or list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *policyID, *identity)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## References

- [Install](../../docs/install.md)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L488)
- [parent-owner base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L65)
