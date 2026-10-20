# Trunk Address Find Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_trunk`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordFindHeaders`

- Pinned direct find_trunk (openstack/network/v2/_proxy.py:7869) accepts a string, ignore_missing=True and query. Native GET200 /trunks/id and trunk envelope preserve canonical IDs and typed nested model/timestamp extraction; undeclared GET203/204 and accepted decode errors remain terminal without fallback or resend.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksGETModelsAndNativeCodes`.
- Compatible GET400/403/404 may fall back, with explicit 404-only/GET-only and strict options. Logical absence defaults to nil,nil after successful complete native absence, while LIST403/404 and later HTTP/decode/cycle/cancellation remain terminal. Exact ID OR name matches include repeated-ID and cross-project ambiguity over all native continuations.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksFallbackQueryAndMissingPolicies`, `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksAllPagesAndTerminalObservations`.
- Caller raw query, including status/fields/tags/vendor/project filters, is frozen and retained in both phases with repetitions, explicit nil/empty name precedence and concurrent reuse. Only an absent caller name gets a literal LIST hint; no project scope is inferred and raw status alone never becomes a local response filter.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksFallbackQueryAndMissingPolicies`, `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksQuerySnapshotsAndConcurrentReuse`.
- Leaf and cached Connection.Network.API retain the original provider/live token, configured headers/version, HTTP middleware, context and ResourceBase. Unsafe names are literal list-only; nil/canceled context, GET-only unsafe names and explicit Details/AllProjects/ExtraSpecs true or false fail before HTTP.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksLiveSourceAndCancellation`, `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksUnsafeNamesAndCapabilityPreflight`.
- Ordinary Resources.List/All preserve full raw wire query with native page extraction and local cap-before-name/first-page controls, without adding a wire limit or identity lookup. Existing explicit ID/Name Ref lookups and the native typed List API remain separate.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksOrdinaryCollectionsQueriesAndControls`.
- Native Trunk preserves port/subport segmentation, status, revision and RFC3339 time.Time values. Its embedded LinkedPageBase with no override uses top-level links.next; trunks_links alone is ignored. All advertised native pages still participate in identity ambiguity/late-failure checks.
  Tests: `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksGETModelsAndNativeCodes`, `api/find_identity_network_pools_trunks_test.go:TestNativeFindIdentityNetworkPoolsTrunksAllPagesAndTerminalObservations`.
- Literal fixed Source member route accepts 200..399 and preserves constructor kwargs, singular-envelope or flat fields, ordered canonical/wire aliases, explicit null and JSON precision separately from actual Envelope/Header/StatusCode. Non-JSON accepted bodies retain seeded Resource defaults; JSON null/scalar decode failures retain actual accepted receipt. Constructor seeds consume query insertion order and last Body alias wins, while response fields overwrite seeds. Unordered bulk Query maps use deterministic sorted keys.
  Tests: `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordMemberSeedDefaultsAndActualEvidence`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordLocalFilterDescriptorAndAliasOrder`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordAllStatusAndPhysicalRejectionContracts`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordConstructorAliasOrderIsSeparateFromListQueryPrecedence`.
- SDK-owned Source query profile sends raw kwargs to GET, maps only actual declared query aliases to LIST, prefers canonical over wire names for LIST only, discards unknown LIST kwargs and applies declared non-query Body values locally. all_projects is consumed and no project scope is inferred.
  Tests: `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordFallbackQueryMappingAndLocalFilters`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordLocalFilterDescriptorAndAliasOrder`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordConstructorAliasOrderIsSeparateFromListQueryPrecedence`.
- Only clean direct GET400/403/404 allows list fallback. Omitted ignore_missing=True permits nil,nil only after a complete successful no-match list; strict mode yields logical ErrNotFound. Missing/empty/null explicit name presence overrides automatic list hint; received IDs are passively compared, not turned into hidden follow-up requests.
  Tests: `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordNamePresenceMissingAndPassiveResponseID`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordAllStatusAndPhysicalRejectionContracts`.
- All Source-advertised pages participate in exact ID-or-name matching and duplicate detection. resource plural-links, top-level next/dictionary links, HTTP Link, versioned paths and limit/marker fallback are supported. max_items counts raw rows before local/name filtering and acts as limit hint when no wire limit was provided. paginated scalar truthiness uses Source binary64 underflow/overflow/signed-zero semantics, separate from exact-decimal equality.
  Tests: `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordLateDuplicateErrorMarkerAndRawCap`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordContinuationFormatsAndCycleGuards`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordPaginationTruthiness`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordFloatTruthinessKeepsWireAndExactComparisonBoundary`.
- Nonquery local dict filters follow Python actual-value truthiness and recursive mapping behavior. Empty filter dict can accept a truthy scalar/list; nonempty dict requiring .get on a nonmapping fails with actual list page evidence. Descriptor projections apply before equality and non-dict equality uses the explicit Go exact-decimal JSON profile with bool/numeric interoperability; complete Python float-rounding equality parity is not claimed. Scalar truthiness treats JSON fractional/exponent underflow as false and overflow as true; plain integers retain arbitrary precision and received Wire bytes stay exact.
  Tests: `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordLocalFilterDescriptorAndAliasOrder`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordFallbackQueryMappingAndLocalFilters`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordFloatTruthinessKeepsWireAndExactComparisonBoundary`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordBooleanDescriptorUsesFloatTruthiness`.
- Owned request options copy JSON, headers, pointers and retained callback state. Invalid context/identity/kwargs/Resource-constructor controls, malformed JSON, negative max_items and forbidden auth/header controls fail before HTTP. Source/client selection facts are sticky across callbacks.
  Tests: `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordOptionSnapshotsAndPreflightGuards`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordSourceCancellationAndConstructorPreflight`.
- Malformed rows/envelopes and constructor duplicate connection/microversion/_synchronized keys retain entire actual page. Actual HTTP proof and received extension JSON are separate from descriptor/default/constructor Resource fields; caller result/header mutation does not change sibling evidence.
  Tests: `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordMalformedPagesAndDetachedExtensionEvidence`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordMemberSeedDefaultsAndActualEvidence`.
- Every rejected400..599 retains native actual status/body/header and physical Read/Close causes, including compatible-fallback statuses. Accepted physical Read/Close, custom cancellation causes and source faults are terminal and keep actual receipt; they never become fallback or logical absence.
  Tests: `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordAllStatusAndPhysicalRejectionContracts`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordPhysicalFailuresPreserveActualEvidence`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordSourceCancellationAndConstructorPreflight`.
- Native retry ownership remains fixed: original selected service/provider/token source is guarded, clean rejection can fall back, physical/callback faults cannot; callbacks may not expand success codes, change phase version or alter fixed target. Reauthentication observes updated provider token and keeps original selected client unchanged.
  Tests: `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordRetryCallbacksKeepCleanFallbackAndGuards`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordListAcceptAndRetryVersionPolicy`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordLiveReauthenticationAndMicroversionOwnership`.
- Pinned Source explicit microversion applies only to member fetch; fallback list uses captured session/service default and no class maximum discovery is invented. Original source version header is independently owned when override differs. LIST Accept default application/json wins configured default while explicit safe SDK header options override both phases.
  Tests: `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordFallbackUsesSelectedVersionAndSourceReceipt`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordListAcceptAndRetryVersionPolicy`, `network/v2/extensions/trunks/python_find_records_contracts_test.go:TestPythonTrunkFindRecordLiveReauthenticationAndMicroversionOwnership`.

Python/Go differences:

- SDK-owned concrete identity options preserve missing/default/fallback policies and query ownership. Raw wire aliases and unknown keys are explicit Go extensions; Python query transposition/local Body classification is not automatic. No hidden/default/enrichment/parent or microversion-discovery policy is added.
- Unsafe names are list-only, GET-only rejects them before HTTP and consumed models require safe nonblank IDs. Native accepted statuses, model/extractor and advertised native continuation semantics remain intact. Project scope is caller-selected rather than inferred.
- The audited ordinary common-list bridge preserves repeated/empty raw query, including status and extensions, instead of collapsing values or silently deleting status. Native typed List options and explicit Ref semantics are retained.
- Native Trunk time.Time decoding requires RFC3339, without a custom old timestamp normalizer. Its default pager reads links.next rather than Python trunks_links, and the SDK does not synthesize another continuation format.
- Go returns an owned concrete Source record and context/error rather than a mutable Python Resource. Resource default/seed view, received Wire object and full HTTP receipt remain distinct. Connection-derived location is null unless explicit literal state exists; Python dirty-state/session GET cache/full Resource lifetime remain separately tracked shared SDK work.
- Go fixed literal UTF-8 routes escape slash/percent and reject empty/dot/control segments, arbitrary base_path/list_base_path, duplicate id and constructor/session/cache controls before HTTP. Safe route ownership and all origin/filter/cycle checks are explicit Go constraints rather than arbitrary caller builder responsibilities.
- WithRecordFindHeaders is an explicit safe Go option applied in both phases; it does not infer Python headers kwargs, which fetch treats as query/constructor input while list treats as request headers. Caller version/auth/framing header conflicts are guarded. Explicit microversion remains Source GET-only.
- Go max_items accepts nonnegative Go integers and preserves JSON kwargs/query conversion without Python deprecation warnings. Invalid UTF-8/incompatible JSON values return actual response or preflight errors instead of invoking arbitrary user-defined descriptor behavior.
- These classes inherit Resource directly, so no NetworkResource If-Match descriptor or custom timestamp normalization is invented. Native typed FindIdentity/Get/List remain separate callable APIs; this new record profile implements actual declared Source aliases/defaults/continuation semantics.
- Scalar truthiness follows CPython JSON binary64 for fraction/exponent values, including underflow, overflow and signed zero, with arbitrary precision plain integers. Numeric equality deliberately uses exact-decimal JSON instead: 1e-4000 differs from 0, 1e4000 differs from 2e4000, and 9007199254740993.0 equals integer 9007199254740993. These comparisons may differ after Python float parsing; no full numeric equality parity is claimed. Wire/Envelope bytes remain unchanged.
- GET constructor Body aliases follow frozen kwargs insertion order and last alias value wins, distinct from LIST canonical query precedence. SDK-owned bulk Query maps have no insertion order and are sorted deterministically.

### `python:network/v2/find_address_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordFindHeaders`

- Pinned direct find_address_group (openstack/network/v2/_proxy.py:346) accepts a string, ignore_missing=True and query. Native GET200 /address-groups/id and address_group envelope preserve canonical IDs and typed model extraction; undeclared GET203/204 and accepted native decode errors remain terminal without fallback or resend.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesGETModelsAndNativeCodes`.
- Compatible GET400/403/404 may fall back, with explicit 404-only/GET-only and strict options. Logical absence defaults to nil,nil after successful complete native absence; LIST403/404, subsequent HTTP/decode/cycle/cancellation and same-ID/cross-project ambiguity over all native pages remain terminal.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesFallbackQueryAndMissingPolicies`, `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesAllPagesAndTerminalObservations`.
- Caller raw project/shared/fields/tags/status/vendor query is frozen and retained in both phases with repetitions, explicit nil/empty name precedence and concurrent reuse. Only an absent caller name gets a literal LIST hint; raw status alone is wire-only and no project scope is inferred.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesFallbackQueryAndMissingPolicies`, `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesQuerySnapshotsAndConcurrentReuse`.
- Leaf and cached Connection.Network.API retain live provider/token, configured headers/version, HTTP middleware, context and ResourceBase. Unsafe names are list-only, GET-only unsafe names and nil/canceled context fail before HTTP; explicit Details/AllProjects/ExtraSpecs false or true and typed WithStatus are unsupported before HTTP.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesLiveSourceAndCancellation`, `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesUnsafeNamesAndCapabilityPreflight`.
- Ordinary Resources.List/All preserve full raw wire query with native extraction and local cap-before-name/first-page controls, without adding a wire limit or identity lookup. Native typed List and explicit ID/Name Ref lookups remain separate.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesOrdinaryCollectionsQueriesAndControls`.
- Native collection uses address_groups_links rel=next and ignores top-level links.next alone. All advertised pages participate in identity matching and later errors. AddressGroup preserves project and Addresses string slices; malformed address collection values remain decode errors.
  Tests: `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesGETModelsAndNativeCodes`, `api/find_identity_network_qos_addresses_test.go:TestNativeFindIdentityNetworkQoSAddressesAllPagesAndTerminalObservations`.
- Literal fixed Source member route accepts 200..399 and preserves constructor kwargs, singular-envelope or flat fields, ordered canonical/wire aliases, explicit null and JSON precision separately from actual Envelope/Header/StatusCode. Non-JSON accepted bodies retain seeded Resource defaults; JSON null/scalar decode failures retain actual accepted receipt.
  Tests: `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordMemberSeedDefaultsAndActualEvidence`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordLocalFilterDescriptorAndAliasOrder`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordAllStatusAndPhysicalRejectionContracts`.
- SDK-owned Source query profile sends raw kwargs to GET, maps only actual declared query aliases to LIST, prefers canonical over wire names for LIST only, discards unknown LIST kwargs and applies declared non-query Body values locally. all_projects is consumed and no project scope is inferred.
  Tests: `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordFallbackQueryMappingAndLocalFilters`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordLocalFilterDescriptorAndAliasOrder`.
- Only clean direct GET400/403/404 allows list fallback. Omitted ignore_missing=True permits nil,nil only after a complete successful no-match list; strict mode yields logical ErrNotFound. Missing/empty/null explicit name presence overrides automatic list hint; received IDs are passively compared, not turned into hidden follow-up requests.
  Tests: `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordNamePresenceMissingAndPassiveResponseID`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordAllStatusAndPhysicalRejectionContracts`.
- All Source-advertised pages participate in exact ID-or-name matching and duplicate detection. resource plural-links, top-level next/dictionary links, HTTP Link, versioned paths and limit/marker fallback are supported. max_items counts raw rows before local/name filtering and acts as limit hint when no wire limit was provided. paginated scalar truthiness uses Source binary64 underflow/overflow/signed-zero semantics, separate from exact-decimal equality.
  Tests: `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordLateDuplicateErrorMarkerAndRawCap`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordContinuationFormatsAndCycleGuards`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordPaginationTruthiness`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordFloatTruthinessKeepsWireAndExactComparisonBoundary`.
- Nonquery local dict filters follow Python actual-value truthiness and recursive mapping behavior. Empty filter dict can accept a truthy scalar/list; nonempty dict requiring .get on a nonmapping fails with actual list page evidence. Descriptor projections apply before equality and non-dict equality uses the explicit Go exact-decimal JSON profile with bool/numeric interoperability; complete Python float-rounding equality parity is not claimed. Scalar truthiness treats JSON fractional/exponent underflow as false and overflow as true; plain integers retain arbitrary precision and received Wire bytes stay exact.
  Tests: `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordLocalFilterDescriptorAndAliasOrder`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordFallbackQueryMappingAndLocalFilters`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordFloatTruthinessKeepsWireAndExactComparisonBoundary`.
- Owned request options copy JSON, headers, pointers and retained callback state. Invalid context/identity/kwargs/Resource-constructor controls, malformed JSON, negative max_items and forbidden auth/header controls fail before HTTP. Source/client selection facts are sticky across callbacks.
  Tests: `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordOptionSnapshotsAndPreflightGuards`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordSourceCancellationAndConstructorPreflight`.
- Malformed rows/envelopes and constructor duplicate connection/microversion/_synchronized keys retain entire actual page. Actual HTTP proof and received extension JSON are separate from descriptor/default/constructor Resource fields; caller result/header mutation does not change sibling evidence.
  Tests: `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordMalformedPagesAndDetachedExtensionEvidence`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordMemberSeedDefaultsAndActualEvidence`.
- Every rejected400..599 retains native actual status/body/header and physical Read/Close causes, including compatible-fallback statuses. Accepted physical Read/Close, custom cancellation causes and source faults are terminal and keep actual receipt; they never become fallback or logical absence.
  Tests: `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordAllStatusAndPhysicalRejectionContracts`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordPhysicalFailuresPreserveActualEvidence`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordSourceCancellationAndConstructorPreflight`.
- Native retry ownership remains fixed: original selected service/provider/token source is guarded, clean rejection can fall back, physical/callback faults cannot; callbacks may not expand success codes, change phase version or alter fixed target. Reauthentication observes updated provider token and keeps original selected client unchanged.
  Tests: `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordRetryCallbacksKeepCleanFallbackAndGuards`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordListAcceptAndRetryVersionPolicy`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordLiveReauthenticationAndMicroversionOwnership`.
- Pinned Source explicit microversion applies only to member fetch; fallback list uses captured session/service default and no class maximum discovery is invented. Original source version header is independently owned when override differs. LIST Accept default application/json wins configured default while explicit safe SDK header options override both phases.
  Tests: `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordFallbackUsesSelectedVersionAndSourceReceipt`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordListAcceptAndRetryVersionPolicy`, `network/v2/extensions/security/addressgroups/python_find_records_contracts_test.go:TestPythonAddressGroupFindRecordLiveReauthenticationAndMicroversionOwnership`.

Python/Go differences:

- SDK-owned concrete identity options provide missing/default/fallback/query ownership. Raw wire aliases and unknown keys are explicit Go extensions; Python query transposition, unknown-key discard and local Body classification are not automatic. No hidden/default/enrichment/parent or discovery policy is added.
- Unsafe names are list-only, GET-only rejects them before HTTP and consumed models require safe nonblank IDs. Native accepted statuses, model/extractor and advertised continuation semantics remain intact. Project scope is caller-selected.
- The audited ordinary common-list bridge preserves repeated/nil/empty raw query, including status and extensions. Both native models have no Status, so typed WithStatus is unsupported while raw status is wire-only. Native typed List and explicit Ref behavior are preserved.
- Python addresses is a declared Body local filter; Go raw addresses remains a wire query and native Addresses is a string slice. Native extraction does not provide generic unknown outer Body attributes or seeded descriptor state.
- Go returns an owned concrete Source record and context/error rather than a mutable Python Resource. Resource default/seed view, received Wire object and full HTTP receipt remain distinct. Connection-derived location is null unless explicit literal state exists; Python dirty-state/session GET cache/full Resource lifetime remain separately tracked shared SDK work.
- Go fixed literal UTF-8 routes escape slash/percent and reject empty/dot/control segments, arbitrary base_path/list_base_path, duplicate id and constructor/session/cache controls before HTTP. Safe route ownership and all origin/filter/cycle checks are explicit Go constraints rather than arbitrary caller builder responsibilities.
- WithRecordFindHeaders is an explicit safe Go option applied in both phases; it does not infer Python headers kwargs, which fetch treats as query/constructor input while list treats as request headers. Caller version/auth/framing header conflicts are guarded. Explicit microversion remains Source GET-only.
- Go max_items accepts nonnegative Go integers and preserves JSON kwargs/query conversion without Python deprecation warnings. Invalid UTF-8/incompatible JSON values return actual response or preflight errors instead of invoking arbitrary user-defined descriptor behavior.
- These classes inherit Resource directly, so no NetworkResource If-Match descriptor or custom timestamp normalization is invented. Native typed FindIdentity/Get/List remain separate callable APIs; this new record profile implements actual declared Source aliases/defaults/continuation semantics.
- Scalar truthiness follows CPython JSON binary64 for fraction/exponent values, including underflow, overflow and signed zero, with arbitrary precision plain integers. Numeric equality deliberately uses exact-decimal JSON instead: 1e-4000 differs from 0, 1e4000 differs from 2e4000, and 9007199254740993.0 equals integer 9007199254740993. These comparisons may differ after Python float parsing; no full numeric equality parity is claimed. Wire/Envelope bytes remain unchanged.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack

with openstack.connect(cloud="devstack") as conn:
    trunk = conn.network.find_trunk(
        "application-trunk", ignore_missing=False,
        project_id="project-id", is_admin_state_up=False,
        fields=["id", "name", "admin_state_up"],
    )
    address_group = conn.network.find_address_group(
        "application-addresses", ignore_missing=False,
        project_id="project-id", addresses=["10.0.0.0/24"],
    )
    print(json.dumps(trunk.to_dict(), indent=2))
    print(json.dumps(address_group.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, kind, identity, project string) (*resource.RawResource, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch kind {
    case "trunk":
        options := []trunks.RecordFindOption{
            trunks.WithRecordFindIgnoreMissing(false),
            trunks.WithRecordFindQuery("is_admin_state_up", false),
        }
        if project != "" { options = append(options, trunks.WithRecordFindQuery("project_id", project)) }
        record, err := network.Trunks.FindRecord(ctx, identity, options...)
        if err != nil || record == nil { return nil, err }
        return record.Resource, nil
    case "address-group":
        options := []addressgroups.RecordFindOption{
            addressgroups.WithRecordFindIgnoreMissing(false),
        }
        if project != "" { options = append(options, addressgroups.WithRecordFindQuery("project_id", project)) }
        record, err := network.SecurityAddressGroups.FindRecord(ctx, identity, options...)
        if err != nil || record == nil { return nil, err }
        return record.Resource, nil
    default:
        return nil, fmt.Errorf("unknown kind %q", kind)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    kind := flag.String("kind", "trunk", "trunk or address-group")
    identity := flag.String("identity", "application-trunk", "exact ID or name")
    project := flag.String("project", "", "optional project filter")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *kind, *identity, *project)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```
