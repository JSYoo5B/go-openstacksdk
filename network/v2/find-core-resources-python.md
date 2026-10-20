# Find Core Resources Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindMicroversion`

- Safe string identities use the original native getter when query is absent and the audited query getter otherwise; both retain native success codes. Default logical absence is ignored, Compatible falls back only after GET400/403/404, with explicit 404-only/GET-only and strict missing options.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter`, `api/find_identity_native_test.go:TestNativeFindIdentityFallbackPoliciesAndMissingDefaults`.
- Caller wire query is frozen once and independently copied to the direct native GET and fallback LIST; repeated values, nil/empty-key precedence and custom option snapshots survive both phases. Query GET requires the explicit adapter hook before any HTTP, while no-query GET retains the original native getter.
  Tests: `resource/find_identity_test.go:TestCollectionFindIdentityQueryHookCapabilityBeforeHTTP`, `resource/find_identity_test.go:TestCollectionFindIdentityQueryHookMutationsCannotChangeFallback`, `resource/find_identity_test.go:TestCollectionFindIdentityQueryHookKeepsNativeAndTerminalCauses`, `api/find_identity_native_test.go:TestNativeFindIdentityQuerySnapshotsRepeatedValuesAndConcurrentReuse`.
- Unsafe ordinary names are literal list-only queries; exact ID OR name matching is retained even with a caller name filter, and bulk query arrays, nil/empty name-key precedence, snapshots and concurrent option reuse are preserved.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentityUnsafeNamesRemainLiteralQueryOnlyAndIDMatches`, `api/find_identity_native_test.go:TestNativeFindIdentityQuerySnapshotsRepeatedValuesAndConcurrentReuse`, `resource/find_identity_test.go:TestCollectionFindIdentitySnapshotsBulkAndCapturedCustomOptions`.
- All advertised native pages are consumed; duplicate rows including repeated IDs are ambiguous, and later HTTP/decode/cycle errors remain terminal. List403/404, decode, transport and context errors are never hidden by the missing policy.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentityAllPagesDuplicatesAndTerminalErrorsAfterMatch`, `api/find_identity_native_test.go:TestNativeFindIdentityListHTTPAbsenceIsNeverIgnored`, `api/find_identity_native_test.go:TestNativeFindIdentityDecodeAndTransportFailuresNeverFallback`.
- Invalid options, reserved query controls, blank/control identities and canceled context fail before HTTP. Every consumed result requires a safe nonblank ID; accepted ResponseError causes and cancellation do not trigger status fallback.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentityValidatesOptionsInputAndContextBeforeHTTP`, `resource/find_identity_test.go:TestCollectionFindIdentityRespectsBindingIDPolicyAndConsumedRows`, `resource/find_identity_test.go:TestCollectionFindIdentityAcceptedAndNestedTerminalErrorsNeverFallback`, `resource/find_identity_test.go:TestCollectionFindIdentityChecksCancellationBetweenPhasesAndAfterEmptyList`.
- Existing Connection high-level Server/Volume/Port/Network collections share the identity policy; repeated frozen wire filters reach both direct GET and fallback LIST, and only LIST gains the native detail/literal name hint. GET403 followed by a successful empty list obeys default/strict logical absence.
  Tests: `connection_find_identity_test.go:TestConnectionFindIdentitySharesPolicyAcrossHighLevelResources`.
- The audited query getter preserves the native GetResult extractor and declared success codes, reverse ResourceBase, live provider token, configured headers, HTTP middleware and caller context without cloning or mutating the shared service client; native reauthentication replays the same frozen request.
  Tests: `internal/nativefind/get_test.go:TestNativeIdentityGetPreservesQueryNativeResultCodesAndEvidence`, `internal/nativefind/get_test.go:TestNativeIdentityGetFreezesInputsAndUsesLiveProviderReauthentication`, `internal/nativefind/get_test.go:TestNativeIdentityGetAcceptedDecodeAndContextRemainTerminal`, `internal/nativefind/get_test.go:TestNativeIdentityGetRejectsUnsafeRoutesAndClientURLsBeforeHTTP`, `api/find_identity_identity_network_test.go:TestNativeFindIdentityDualPhaseQueryKeepsLiveProviderAndHTTPClient`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Literal escaped string identity always GETs first with untransposed JSON kwargs. Accepted partial/204/invalid-JSON responses retain constructor seed; explicit response null can replace seeded identity. Raw records preserve Source bool/int/list/dict descriptors, declared missing defaults, ordered canonical/wire response aliases, unknown Neutron extensions, large numbers and actual HTTP evidence.
  Tests: `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordMemberFirstSeedAndResponseProjection`, `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordMalformedPagesAndDetachedExtensionEvidence`.
- Only clean member GET400/403/404 falls back. LIST uses pinned ordered QueryParameters canonical/wire aliases (including Port tenant_id later-declaration precedence), repeated fields/tag values, unknown discard and declared non-query Body local filters. all_projects is excluded from fetch and discarded by these Neutron lists; unrelated Source control overrides fail before HTTP.
  Tests: `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordFallbackAliasesAndLocalBodyFilters`, `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordOptionSnapshotsAndPreflightGuards`.
- An absent name keyword injects the identity hint into LIST only; null and empty explicitly suppress or preserve it. Exact string ID OR name is matched while missing/numeric response IDs remain passive. Default logical absence returns nil,nil; strict absence errors, and LIST404 never becomes missing success.
  Tests: `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordNamePresenceMissingAndPassiveResponseID`.
- The lookup consumes linked/HTTP Link/dictionary/top-level next and version-prefixed pages; late duplicate/error cannot return an earlier match. Explicit positive limit continues short pages using the raw ID marker, raw max_items precedes local filtering, pagination uses Source JSON truthiness, and loops or route/filter changes retain actual-page errors.
  Tests: `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordLateDuplicateErrorMarkerAndRawCap`, `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordContinuationFormatsAndCycleGuards`, `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordPaginationTruthinessAndHeaderConstructorProjection`.
- Local recursive dictionary filters follow Source truthiness: empty dict accepts a truthy scalar/list, falsey observations fail, and nonempty dictionary traversal of scalar/list is an actual-page error. Python bool/number equality is retained. Local filters stop on the first false in concrete option order; owned bulk maps use deterministic key order.
  Tests: `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts`, `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordFallbackAliasesAndLocalBodyFilters`.
- Options snapshot raw bytes, map/array/header values, missing/version pointers and captured callback state. Invalid JSON, nil option, negative cap, identity/Resource/fetch controls, protected Header, source changes and caller cancellation fail before HTTP with their original causes.
  Tests: `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordOptionSnapshotsAndPreflightGuards`, `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordLiveReauthenticationAndMicroversionOwnership`.
- GET inflates actual If-Match over Header seed and cannot be shadowed by Body if_match; LIST consumes row constructor Header canonical/wire attrs without inflating page HTTP headers into each row. LIST connection/microversion/_synchronized keyword collisions, null/scalar/missing/malformed plural representations retain complete actual page receipts.
  Tests: `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordPaginationTruthinessAndHeaderConstructorProjection`, `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordMalformedPagesAndDetachedExtensionEvidence`.
- Native unexpected-status errors preserve actual method/URL/body/header/status and rejected Read/Close causes across member and fallback LIST, including statuses500. Accepted read/close/late cancellation failures retain original response receipts and never establish absence or resend.
  Tests: `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordPhysicalFailuresPreserveActualEvidence`.
- A retry returning the exact original clean rejection remains fallback-eligible; physical rejection errors remain terminal. Actual retry-admitted404 is rejected by the original SDK200..399 status policy, retains actual native receipt and never triggers LIST. Live token and native reauthentication replay the same owned query/Header/microversion while leaving selected service settings unchanged.
  Tests: `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordRetryCallbacksKeepCleanFallbackAndGuards`, `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordLiveReauthenticationAndMicroversionOwnership`.
- Source LIST supplies application/json Accept over configured service headers unless caller explicitly selects another Accept. Retry callbacks cannot alter operation-owned selected version Header or its required absence; failed changes retain the original rejected HTTP evidence without an extra physical attempt or fallback.
  Tests: `network/v2/ports/python_find_records_contracts_test.go:TestPythonPortFindRecordListAcceptAndRetryVersionPolicy`.

Python/Go differences:

- The Go concrete options accept JSON-domain values and escaped UTF-8 string routes, rejecting blank/control segments, id collisions, session/route/fetch control overrides, protected headers and malformed known input. They do not emulate arbitrary Python objects or subclass hooks. Caller Go maps have no insertion order: bulk local filters use sorted keys and GET seed canonical values win over their own wire aliases; WithRecordFindQuery option order and response JSON dictionary order are retained.
- Python mutable Resource/Munch lifecycle, shared proxy response caching, warning/subclass/session hooks and connection-derived location remain explicit project-wide common scope. These owned records retain independent canonical Resource, raw Wire and original HTTP evidence; literal location is retained where supplied and otherwise location is null. This callable mapping is not a claim of full SDK cache or mutable identity parity.
- The operation owns optional selected microversion/header settings instead of negotiating resource maxima or changing the shared client. Source inherited requires_id/error_message/skip_cache/resource_response_key and arbitrary list_base_path are rejected for this fixed literal search. These Neutron resources have no parent URI descriptors; route retargeting is not inferred from JSON kwargs.
- The shared guarded pager rejects cross-origin/collection/filter mutations, repeated pages, conflicting links and invalid cursor representations. JSON-domain Requests expansion and Python equality/descriptor helpers are reused; runtime-dependent non-JSON values, CPython configurable integer limits and full Python Unicode runtime choice are not implemented by this callable.

### `python:network/v2/find_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordFindMicroversion`

- Safe string identities use the original native getter when query is absent and the audited query getter otherwise; both retain native success codes. Default logical absence is ignored, Compatible falls back only after GET400/403/404, with explicit 404-only/GET-only and strict missing options.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter`, `api/find_identity_native_test.go:TestNativeFindIdentityFallbackPoliciesAndMissingDefaults`.
- Pinned Network find declaration (openstack/network/v2/_proxy.py:3214-3239) is explicitly opted into a string-ID/name binding. The original getter and query getter both accept only HTTP200; HTTP203 is terminal, with the same native singular envelope and plural linked-page format.
  Tests: `api/find_identity_identity_network_test.go:TestNativeFindIdentityIdentityAndNetworkGETStatusContracts`.
- Caller wire query is frozen once and independently copied to the direct native GET and fallback LIST; repeated values, nil/empty-key precedence and custom option snapshots survive both phases. Query GET requires the explicit adapter hook before any HTTP, while no-query GET retains the original native getter.
  Tests: `resource/find_identity_test.go:TestCollectionFindIdentityQueryHookCapabilityBeforeHTTP`, `resource/find_identity_test.go:TestCollectionFindIdentityQueryHookMutationsCannotChangeFallback`, `resource/find_identity_test.go:TestCollectionFindIdentityQueryHookKeepsNativeAndTerminalCauses`, `api/find_identity_native_test.go:TestNativeFindIdentityQuerySnapshotsRepeatedValuesAndConcurrentReuse`.
- Unsafe ordinary names are literal list-only queries; exact ID OR name matching is retained even with a caller name filter, and bulk query arrays, nil/empty name-key precedence, snapshots and concurrent option reuse are preserved.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentityUnsafeNamesRemainLiteralQueryOnlyAndIDMatches`, `api/find_identity_native_test.go:TestNativeFindIdentityQuerySnapshotsRepeatedValuesAndConcurrentReuse`, `resource/find_identity_test.go:TestCollectionFindIdentitySnapshotsBulkAndCapturedCustomOptions`.
- All advertised native pages are consumed; duplicate rows including repeated IDs are ambiguous, and later HTTP/decode/cycle errors remain terminal. List403/404, decode, transport and context errors are never hidden by the missing policy.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentityAllPagesDuplicatesAndTerminalErrorsAfterMatch`, `api/find_identity_native_test.go:TestNativeFindIdentityListHTTPAbsenceIsNeverIgnored`, `api/find_identity_native_test.go:TestNativeFindIdentityDecodeAndTransportFailuresNeverFallback`.
- Invalid options, reserved query controls, blank/control identities and canceled context fail before HTTP. Every consumed result requires a safe nonblank ID; accepted ResponseError causes and cancellation do not trigger status fallback.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentityValidatesOptionsInputAndContextBeforeHTTP`, `resource/find_identity_test.go:TestCollectionFindIdentityRespectsBindingIDPolicyAndConsumedRows`, `resource/find_identity_test.go:TestCollectionFindIdentityAcceptedAndNestedTerminalErrorsNeverFallback`, `resource/find_identity_test.go:TestCollectionFindIdentityChecksCancellationBetweenPhasesAndAfterEmptyList`.
- The audited query getter preserves the native GetResult extractor and declared success codes, reverse ResourceBase, live provider token, configured headers, HTTP middleware and caller context without cloning or mutating the shared service client; native reauthentication replays the same frozen request.
  Tests: `internal/nativefind/get_test.go:TestNativeIdentityGetPreservesQueryNativeResultCodesAndEvidence`, `internal/nativefind/get_test.go:TestNativeIdentityGetFreezesInputsAndUsesLiveProviderReauthentication`, `internal/nativefind/get_test.go:TestNativeIdentityGetAcceptedDecodeAndContextRemainTerminal`, `internal/nativefind/get_test.go:TestNativeIdentityGetRejectsUnsafeRoutesAndClientURLsBeforeHTTP`, `api/find_identity_identity_network_test.go:TestNativeFindIdentityDualPhaseQueryKeepsLiveProviderAndHTTPClient`.
- The cached Connection high-level Network collection uses the same dual-phase frozen query, literal name hint and default/strict logical missing policy as the generated facade.
  Tests: `connection_find_identity_test.go:TestConnectionFindIdentitySharesPolicyAcrossHighLevelResources`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Literal escaped string identity always GETs first with untransposed JSON kwargs. Accepted partial/204/invalid-JSON responses retain constructor seed; explicit response null can replace seeded identity. Raw records preserve Source bool/int/list/dict descriptors, declared missing defaults, ordered canonical/wire response aliases, unknown Neutron extensions, large numbers and actual HTTP evidence.
  Tests: `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordMemberFirstSeedAndResponseProjection`, `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordMalformedPagesAndDetachedExtensionEvidence`.
- Only clean member GET400/403/404 falls back. LIST uses pinned ordered QueryParameters canonical/wire aliases (including Port tenant_id later-declaration precedence), repeated fields/tag values, unknown discard and declared non-query Body local filters. all_projects is excluded from fetch and discarded by these Neutron lists; unrelated Source control overrides fail before HTTP.
  Tests: `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordFallbackAliasesAndLocalBodyFilters`, `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordOptionSnapshotsAndPreflightGuards`.
- An absent name keyword injects the identity hint into LIST only; null and empty explicitly suppress or preserve it. Exact string ID OR name is matched while missing/numeric response IDs remain passive. Default logical absence returns nil,nil; strict absence errors, and LIST404 never becomes missing success.
  Tests: `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordNamePresenceMissingAndPassiveResponseID`.
- The lookup consumes linked/HTTP Link/dictionary/top-level next and version-prefixed pages; late duplicate/error cannot return an earlier match. Explicit positive limit continues short pages using the raw ID marker, raw max_items precedes local filtering, pagination uses Source JSON truthiness, and loops or route/filter changes retain actual-page errors.
  Tests: `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordLateDuplicateErrorMarkerAndRawCap`, `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordContinuationFormatsAndCycleGuards`, `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordPaginationTruthinessAndHeaderConstructorProjection`.
- Local recursive dictionary filters follow Source truthiness: empty dict accepts a truthy scalar/list, falsey observations fail, and nonempty dictionary traversal of scalar/list is an actual-page error. Python bool/number equality is retained. Local filters stop on the first false in concrete option order; owned bulk maps use deterministic key order.
  Tests: `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts`, `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordFallbackAliasesAndLocalBodyFilters`.
- Options snapshot raw bytes, map/array/header values, missing/version pointers and captured callback state. Invalid JSON, nil option, negative cap, identity/Resource/fetch controls, protected Header, source changes and caller cancellation fail before HTTP with their original causes.
  Tests: `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordOptionSnapshotsAndPreflightGuards`, `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordLiveReauthenticationAndMicroversionOwnership`.
- GET inflates actual If-Match over Header seed and cannot be shadowed by Body if_match; LIST consumes row constructor Header canonical/wire attrs without inflating page HTTP headers into each row. LIST connection/microversion/_synchronized keyword collisions, null/scalar/missing/malformed plural representations retain complete actual page receipts.
  Tests: `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordPaginationTruthinessAndHeaderConstructorProjection`, `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordMalformedPagesAndDetachedExtensionEvidence`.
- Native unexpected-status errors preserve actual method/URL/body/header/status and rejected Read/Close causes across member and fallback LIST, including statuses500. Accepted read/close/late cancellation failures retain original response receipts and never establish absence or resend.
  Tests: `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordPhysicalFailuresPreserveActualEvidence`.
- A retry returning the exact original clean rejection remains fallback-eligible; physical rejection errors remain terminal. Actual retry-admitted404 is rejected by the original SDK200..399 status policy, retains actual native receipt and never triggers LIST. Live token and native reauthentication replay the same owned query/Header/microversion while leaving selected service settings unchanged.
  Tests: `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordRetryCallbacksKeepCleanFallbackAndGuards`, `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordLiveReauthenticationAndMicroversionOwnership`.
- Source LIST supplies application/json Accept over configured service headers unless caller explicitly selects another Accept. Retry callbacks cannot alter operation-owned selected version Header or its required absence; failed changes retain the original rejected HTTP evidence without an extra physical attempt or fallback.
  Tests: `network/v2/networks/python_find_records_contracts_test.go:TestPythonNetworkFindRecordListAcceptAndRetryVersionPolicy`.

Python/Go differences:

- The Go concrete options accept JSON-domain values and escaped UTF-8 string routes, rejecting blank/control segments, id collisions, session/route/fetch control overrides, protected headers and malformed known input. They do not emulate arbitrary Python objects or subclass hooks. Caller Go maps have no insertion order: bulk local filters use sorted keys and GET seed canonical values win over their own wire aliases; WithRecordFindQuery option order and response JSON dictionary order are retained.
- Python mutable Resource/Munch lifecycle, shared proxy response caching, warning/subclass/session hooks and connection-derived location remain explicit project-wide common scope. These owned records retain independent canonical Resource, raw Wire and original HTTP evidence; literal location is retained where supplied and otherwise location is null. This callable mapping is not a claim of full SDK cache or mutable identity parity.
- The operation owns optional selected microversion/header settings instead of negotiating resource maxima or changing the shared client. Source inherited requires_id/error_message/skip_cache/resource_response_key and arbitrary list_base_path are rejected for this fixed literal search. These Neutron resources have no parent URI descriptors; route retargeting is not inferred from JSON kwargs.
- The shared guarded pager rejects cross-origin/collection/filter mutations, repeated pages, conflicting links and invalid cursor representations. JSON-domain Requests expansion and Python equality/descriptor helpers are reused; runtime-dependent non-JSON values, CPython configurable integer limits and full Python Unicode runtime choice are not implemented by this callable.

### `python:network/v2/find_subnet`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordFindMicroversion`

- Safe string identities use the original native getter when query is absent and the audited query getter otherwise; both retain native success codes. Default logical absence is ignored, Compatible falls back only after GET400/403/404, with explicit 404-only/GET-only and strict missing options.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter`, `api/find_identity_native_test.go:TestNativeFindIdentityFallbackPoliciesAndMissingDefaults`.
- Pinned Subnet find declaration (openstack/network/v2/_proxy.py:7487-7514) is explicitly opted into a string-ID/name binding. The original getter and query getter both accept only HTTP200; HTTP203 is terminal, with the same native singular envelope and plural linked-page format.
  Tests: `api/find_identity_identity_network_test.go:TestNativeFindIdentityIdentityAndNetworkGETStatusContracts`.
- Caller wire query is frozen once and independently copied to the direct native GET and fallback LIST; repeated values, nil/empty-key precedence and custom option snapshots survive both phases. Query GET requires the explicit adapter hook before any HTTP, while no-query GET retains the original native getter.
  Tests: `resource/find_identity_test.go:TestCollectionFindIdentityQueryHookCapabilityBeforeHTTP`, `resource/find_identity_test.go:TestCollectionFindIdentityQueryHookMutationsCannotChangeFallback`, `resource/find_identity_test.go:TestCollectionFindIdentityQueryHookKeepsNativeAndTerminalCauses`, `api/find_identity_native_test.go:TestNativeFindIdentityQuerySnapshotsRepeatedValuesAndConcurrentReuse`.
- Unsafe ordinary names are literal list-only queries; exact ID OR name matching is retained even with a caller name filter, and bulk query arrays, nil/empty name-key precedence, snapshots and concurrent option reuse are preserved.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentityUnsafeNamesRemainLiteralQueryOnlyAndIDMatches`, `api/find_identity_native_test.go:TestNativeFindIdentityQuerySnapshotsRepeatedValuesAndConcurrentReuse`, `resource/find_identity_test.go:TestCollectionFindIdentitySnapshotsBulkAndCapturedCustomOptions`.
- All advertised native pages are consumed; duplicate rows including repeated IDs are ambiguous, and later HTTP/decode/cycle errors remain terminal. List403/404, decode, transport and context errors are never hidden by the missing policy.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentityAllPagesDuplicatesAndTerminalErrorsAfterMatch`, `api/find_identity_native_test.go:TestNativeFindIdentityListHTTPAbsenceIsNeverIgnored`, `api/find_identity_native_test.go:TestNativeFindIdentityDecodeAndTransportFailuresNeverFallback`.
- Invalid options, reserved query controls, blank/control identities and canceled context fail before HTTP. Every consumed result requires a safe nonblank ID; accepted ResponseError causes and cancellation do not trigger status fallback.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentityValidatesOptionsInputAndContextBeforeHTTP`, `resource/find_identity_test.go:TestCollectionFindIdentityRespectsBindingIDPolicyAndConsumedRows`, `resource/find_identity_test.go:TestCollectionFindIdentityAcceptedAndNestedTerminalErrorsNeverFallback`, `resource/find_identity_test.go:TestCollectionFindIdentityChecksCancellationBetweenPhasesAndAfterEmptyList`.
- The audited query getter preserves the native GetResult extractor and declared success codes, reverse ResourceBase, live provider token, configured headers, HTTP middleware and caller context without cloning or mutating the shared service client; native reauthentication replays the same frozen request.
  Tests: `internal/nativefind/get_test.go:TestNativeIdentityGetPreservesQueryNativeResultCodesAndEvidence`, `internal/nativefind/get_test.go:TestNativeIdentityGetFreezesInputsAndUsesLiveProviderReauthentication`, `internal/nativefind/get_test.go:TestNativeIdentityGetAcceptedDecodeAndContextRemainTerminal`, `internal/nativefind/get_test.go:TestNativeIdentityGetRejectsUnsafeRoutesAndClientURLsBeforeHTTP`, `api/find_identity_identity_network_test.go:TestNativeFindIdentityDualPhaseQueryKeepsLiveProviderAndHTTPClient`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Literal escaped string identity always GETs first with untransposed JSON kwargs. Accepted partial/204/invalid-JSON responses retain constructor seed; explicit response null can replace seeded identity. Raw records preserve Source bool/int/list/dict descriptors, declared missing defaults, ordered canonical/wire response aliases, unknown Neutron extensions, large numbers and actual HTTP evidence.
  Tests: `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordMemberFirstSeedAndResponseProjection`, `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordMalformedPagesAndDetachedExtensionEvidence`.
- Only clean member GET400/403/404 falls back. LIST uses pinned ordered QueryParameters canonical/wire aliases (including Port tenant_id later-declaration precedence), repeated fields/tag values, unknown discard and declared non-query Body local filters. all_projects is excluded from fetch and discarded by these Neutron lists; unrelated Source control overrides fail before HTTP.
  Tests: `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordFallbackAliasesAndLocalBodyFilters`, `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordOptionSnapshotsAndPreflightGuards`.
- An absent name keyword injects the identity hint into LIST only; null and empty explicitly suppress or preserve it. Exact string ID OR name is matched while missing/numeric response IDs remain passive. Default logical absence returns nil,nil; strict absence errors, and LIST404 never becomes missing success.
  Tests: `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordNamePresenceMissingAndPassiveResponseID`.
- The lookup consumes linked/HTTP Link/dictionary/top-level next and version-prefixed pages; late duplicate/error cannot return an earlier match. Explicit positive limit continues short pages using the raw ID marker, raw max_items precedes local filtering, pagination uses Source JSON truthiness, and loops or route/filter changes retain actual-page errors.
  Tests: `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordLateDuplicateErrorMarkerAndRawCap`, `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordContinuationFormatsAndCycleGuards`, `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordPaginationTruthinessAndHeaderConstructorProjection`.
- Local recursive dictionary filters follow Source truthiness: empty dict accepts a truthy scalar/list, falsey observations fail, and nonempty dictionary traversal of scalar/list is an actual-page error. Python bool/number equality is retained. Local filters stop on the first false in concrete option order; owned bulk maps use deterministic key order.
  Tests: `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts`, `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordFallbackAliasesAndLocalBodyFilters`.
- Options snapshot raw bytes, map/array/header values, missing/version pointers and captured callback state. Invalid JSON, nil option, negative cap, identity/Resource/fetch controls, protected Header, source changes and caller cancellation fail before HTTP with their original causes.
  Tests: `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordOptionSnapshotsAndPreflightGuards`, `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordLiveReauthenticationAndMicroversionOwnership`.
- GET inflates actual If-Match over Header seed and cannot be shadowed by Body if_match; LIST consumes row constructor Header canonical/wire attrs without inflating page HTTP headers into each row. LIST connection/microversion/_synchronized keyword collisions, null/scalar/missing/malformed plural representations retain complete actual page receipts.
  Tests: `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordPaginationTruthinessAndHeaderConstructorProjection`, `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordMalformedPagesAndDetachedExtensionEvidence`.
- Native unexpected-status errors preserve actual method/URL/body/header/status and rejected Read/Close causes across member and fallback LIST, including statuses500. Accepted read/close/late cancellation failures retain original response receipts and never establish absence or resend.
  Tests: `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordPhysicalFailuresPreserveActualEvidence`.
- A retry returning the exact original clean rejection remains fallback-eligible; physical rejection errors remain terminal. Actual retry-admitted404 is rejected by the original SDK200..399 status policy, retains actual native receipt and never triggers LIST. Live token and native reauthentication replay the same owned query/Header/microversion while leaving selected service settings unchanged.
  Tests: `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordRetryCallbacksKeepCleanFallbackAndGuards`, `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordLiveReauthenticationAndMicroversionOwnership`.
- Source LIST supplies application/json Accept over configured service headers unless caller explicitly selects another Accept. Retry callbacks cannot alter operation-owned selected version Header or its required absence; failed changes retain the original rejected HTTP evidence without an extra physical attempt or fallback.
  Tests: `network/v2/subnets/python_find_records_contracts_test.go:TestPythonSubnetFindRecordListAcceptAndRetryVersionPolicy`.

Python/Go differences:

- The Go concrete options accept JSON-domain values and escaped UTF-8 string routes, rejecting blank/control segments, id collisions, session/route/fetch control overrides, protected headers and malformed known input. They do not emulate arbitrary Python objects or subclass hooks. Caller Go maps have no insertion order: bulk local filters use sorted keys and GET seed canonical values win over their own wire aliases; WithRecordFindQuery option order and response JSON dictionary order are retained.
- Python mutable Resource/Munch lifecycle, shared proxy response caching, warning/subclass/session hooks and connection-derived location remain explicit project-wide common scope. These owned records retain independent canonical Resource, raw Wire and original HTTP evidence; literal location is retained where supplied and otherwise location is null. This callable mapping is not a claim of full SDK cache or mutable identity parity.
- The operation owns optional selected microversion/header settings instead of negotiating resource maxima or changing the shared client. Source inherited requires_id/error_message/skip_cache/resource_response_key and arbitrary list_base_path are rejected for this fixed literal search. These Neutron resources have no parent URI descriptors; route retargeting is not inferred from JSON kwargs.
- The shared guarded pager rejects cross-origin/collection/filter mutations, repeated pages, conflicting links and invalid cursor representations. JSON-domain Requests expansion and Python equality/descriptor helpers are reused; runtime-dependent non-JSON values, CPython configurable integer limits and full Python Unicode runtime choice are not implemented by this callable.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--network", default="app-net")
    parser.add_argument("--port", default="")
    parser.add_argument("--subnet", default="")
    parser.add_argument("--project", default="")
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {"project_id": args.project} if args.project else {}
    finders = [
        ("network", args.network, conn.network.find_network),
        ("port", args.port, conn.network.find_port),
        ("subnet", args.subnet, conn.network.find_subnet),
    ]
    for kind, identity, find in finders:
        if not identity:
            continue
        result = find(identity, ignore_missing=not args.strict, **query)
        print(kind)
        print(json.dumps(result.to_dict() if result is not None else None,
                         indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printResource(kind string, value *resource.RawResource) {
    fmt.Println(kind)
    if value == nil { fmt.Println("null"); return }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    networkName := flag.String("network", "app-net", "network ID or exact name")
    portName := flag.String("port", "", "optional port ID or exact name")
    subnetName := flag.String("subnet", "", "optional subnet ID or exact name")
    project := flag.String("project", "", "optional project filter")
    strict := flag.Bool("strict", false, "fail on logical absence")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    if *networkName != "" {
        options := []networks.RecordFindOption{networks.WithRecordFindIgnoreMissing(!*strict)}
        if *project != "" { options = append(options, networks.WithRecordFindQuery("project_id", *project)) }
        result, err := service.Networks.FindRecord(ctx, *networkName, options...)
        if err != nil { log.Fatal(err) }
        if result == nil { printResource("network", nil) } else { printResource("network", result.Resource) }
    }
    if *portName != "" {
        options := []ports.RecordFindOption{ports.WithRecordFindIgnoreMissing(!*strict)}
        if *project != "" { options = append(options, ports.WithRecordFindQuery("project_id", *project)) }
        result, err := service.Ports.FindRecord(ctx, *portName, options...)
        if err != nil { log.Fatal(err) }
        if result == nil { printResource("port", nil) } else { printResource("port", result.Resource) }
    }
    if *subnetName != "" {
        options := []subnets.RecordFindOption{subnets.WithRecordFindIgnoreMissing(!*strict)}
        if *project != "" { options = append(options, subnets.WithRecordFindQuery("project_id", *project)) }
        result, err := service.Subnets.FindRecord(ctx, *subnetName, options...)
        if err != nil { log.Fatal(err) }
        if result == nil { printResource("subnet", nil) } else { printResource("subnet", result.Resource) }
    }
}
```

```python
port = conn.network.find_port(
    "app-port", binding_profile={"nested": {"enabled": True}})
```

```go
port, err := service.Ports.FindRecord(ctx, "app-port",
    ports.WithRecordFindQuery("binding_profile",
        map[string]any{"nested": map[string]any{"enabled": true}}))
```

## References

- [Install](../../docs/install.md)
- [Network find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3214)
- [Port find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3950)
- [Subnet find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7487)
- [Resource.find/list/fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [NetworkResource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_base.py)
- [Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py)
- [Port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port.py)
- [Subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet.py)
