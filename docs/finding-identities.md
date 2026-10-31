# Finding Identities

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/find_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindDetails`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindAllProjects`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindDetails`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindAllProjects`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindLocation`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindLimit`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindMarker`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithRecordFindPaginated`

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
  Tests: `connectiontest/connection_find_identity_test.go:TestConnectionFindIdentitySharesPolicyAcrossHighLevelResources`.
- The audited query getter preserves the native GetResult extractor and declared success codes, reverse ResourceBase, live provider token, configured headers, HTTP middleware and caller context without cloning or mutating the shared service client; native reauthentication replays the same frozen request.
  Tests: `internal/nativefind/get_test.go:TestNativeIdentityGetPreservesQueryNativeResultCodesAndEvidence`, `internal/nativefind/get_test.go:TestNativeIdentityGetFreezesInputsAndUsesLiveProviderReauthentication`, `internal/nativefind/get_test.go:TestNativeIdentityGetAcceptedDecodeAndContextRemainTerminal`, `internal/nativefind/get_test.go:TestNativeIdentityGetRejectsUnsafeRoutesAndClientURLsBeforeHTTP`, `api/find_identity_identity_network_test.go:TestNativeFindIdentityDualPhaseQueryKeepsLiveProviderAndHTTPClient`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesPreserveSourceClientAndContext`.
- Details defaults to true; false selects the summary fallback route and true selects the detail fallback route, without changing direct member GET. A successful GET returns immediately even in summary mode; ordinary List keeps its original adapter, and summary rows are not followed by implicit detail GETs.
  Tests: `resource/find_identity_modes_test.go:TestCollectionFindIdentityModesSelectFallbackWithoutChangingGETOrList`, `resource/find_identity_modes_test.go:TestCollectionFindIdentityTypedModesKeepSuccessfulMemberGET`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesRoutesFrozenQueriesAndLiveSource`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesDefaultsRoutesAndListOnlyFlags`.
- Typed AllProjects defaults to false: nil/false omits the automatic cross-project field, and true adds all_tenants=true only to fallback LIST. Raw all_tenants with no typed value remains a dual-phase wire extension. Explicit typed values conflict with any case-insensitive raw all_tenants key presence, including nil/empty/repeated values, independently of option order; raw details/all_projects and unsupported explicit false/true fail before HTTP, including GET-only policy.
  Tests: `resource/find_identity_modes_test.go:TestCollectionFindIdentityTypedAllProjectsIsListOnlyAndRawQueryRemainsDualPhase`, `resource/find_identity_modes_test.go:TestCollectionFindIdentityModesPreflightUnsupportedAndConflicts`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesDefaultsRoutesAndListOnlyFlags`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesConflictsAndCapabilitiesBeforeHTTP`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesFallbackPoliciesNativeCodesAndRawScopeExtension`.
- Details/AllProjects/IgnoreMissing pointers and query arrays are independently snapshotted at option construction and after custom option application. Last scalar option wins; a bulk option replaces all fields. Captured configuration mutations during GET cannot change fallback mode/query, and shared options and mode iterators can be reused concurrently.
  Tests: `resource/find_identity_modes_test.go:TestCollectionFindIdentityModesSelectFallbackWithoutChangingGETOrList`, `resource/find_identity_modes_test.go:TestCollectionFindIdentityModesFreezeBulkAndCapturedOptions`, `resource/find_identity_modes_test.go:TestCollectionFindIdentityModeOptionsCanBeReusedConcurrently`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesReusableAndConcurrentIndependentSnapshots`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesOwnedSnapshotsLastWinsAndConcurrentReuse`.
- Both fallback modes retain native page codes and whole-page extraction. Duplicate rows and later HTTP/decode/read/transport/cycle errors remain terminal after a match; consumer break does not inspect continuation, context cancellation remains terminal, and native linked continuation behavior is preserved instead of inventing a stricter route or new response evidence.
  Tests: `resource/find_identity_modes_test.go:TestCollectionFindIdentityModesPreserveAllPageErrorsAndCancellation`, `resource/find_identity_modes_test.go:TestCollectionFindIdentityModeIteratorCancellationIsTerminal`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesNativeCodesAndWholePageErrors`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesBreakCancellationAndPreflight`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesCyclesForeignLinksAndLaterFailure`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesTransportAndAcceptedReadErrorsStayTerminal`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesSummaryPagesDuplicatesAndTerminalFailures`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesFallbackPoliciesNativeCodesAndRawScopeExtension`.
- Only the two audited Nova/Cinder bindings expose details/all-projects modes. Generated identity hooks use the owned mode iterator while retaining ordinary Get/List controls; inventory records both capabilities only for those bindings. Native List/ListSimple URL/page/extractor body, signature, page schema and receiver-qualified declaration drift are rejected before emission.
  Tests: `internal/cmd/sdkgen/identity_modes_test.go:TestIdentityListModesOnlyEnableAuditedConcretePagerContracts`, `internal/cmd/sdkgen/identity_modes_test.go:TestIdentityListModesEmitFallbackOnlyAndExplicitInventory`, `internal/cmd/sdkgen/identity_modes_test.go:TestIdentityListModesNativeDeclarationAndReceiverGuards`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Pinned find_server GET-first defaults/details/current-scope, literal escaping and only clean native400/403/404 fallback; strict/default complete logical absence, cross-project permission rejection and passive received ID.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordDirectDefaultsAliasesAndWire`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordFallbackStatusMissingAndAuthority`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordPassiveIdentityAndOriginalStatusPolicy`.
- Member Source constructor seed merged with successful raw response, Go final-representation statuses200..399 (pinned Source raise_from_response itself returns for <400), tolerated empty/malformed syntax versus valid wrongshape/UTF8/descriptor actual accepted receipt.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordEmptyMalformedShapeAndDescriptorProof`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordDirectDefaultsAliasesAndWire`.
- Owned Source Body/default/aka/computed/location projection uses the exact pinned Server inventory and actual nested Flavor/Image/VolumeAttachment _convert_type/default/alternate-ID/unknown-properties behavior, separate Resource/Wire/Envelope/Header/StatusCode receipts; outer unknown values remain Wire.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordNestedSourceDescriptors`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordNestedConversionsAndConstructorProfile`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordLocationScopeAndFalseyProject`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordListConstructorAndHeaderBoundary`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordDirectDefaultsAliasesAndWire`.
- Go raw kwargs extension preserves original GET query and known constructor seed, LIST Source query-client/wire alias precedence including explicit null, name-hint presence, all_tenants=True LIST only, ordered constructor alias last-input and Source server/local classification.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordDetailsScopeQueryPriorityAndOrderedSeed`.
- Local Source dictionary falsey/truthy scalar/nested subset/mapping.get errors and bool-number equality; Python binary64 scalar decimal truthiness while exact-decimal equality remains an explicit Go JSON profile.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordDictionaryTruthinessAndNumericEquality`.
- Nova captured default selected microversion is retained; absent default negotiates exact Server2.100 ceiling against eligible pinned advertisement with physical discovery proof and one negotiation. Explicit inherited Resource.find microversion is member-GET-only; fallback uses captured default/own discovery. Terminal discovery/source errors never become missing.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordDiscoveryCeilingAndSelectedVersions`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordExplicitVersionIsGETOnly`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordDiscoveryTerminalProofAndSourceGuard`.
- Full retained REST paging consumes later duplicates/errors, plural links/top-level next/HTTP Link and exact relative v2.1 alias, dictionary compatibility, singleton/stop-on-empty/short-page marker continuation and raw cap before localfilter/unconsumed badrows or badnext; fixed origin/collection/filter/cycle guards retain actual page proof.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordLateDuplicateAndPageFailure`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordMarkerCapAndSinglePage`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordPaginationRepresentationsAndGuards`.
- Concrete owned option pointers/maps/raw values/order/location and retained callback independence; unsafe literal/Source control and positional or constructor collision/negative caps/header/query/seed preflight; cancellation/custom and sticky captured Source errors retain causes. Reusable options and output views are independent in concurrent calls.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordInvalidControlsBeforeHTTP`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordSnapshotCancellationAndOptionSource`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordConcurrentOwnedOptions`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordAcceptExplicitAndSelectedAPIChange`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordListConstructorAndHeaderBoundary`.
- Accepted/rejected member and LIST read/Close/custom cancel/transient Source restore keep actual HTTP proof, all causes and exactly one physical Close, stopping lookup fallback/resend. Native retry body/version/status/header/absentauth/framing/foreignversion/Omitversion ownership keeps original rejection/customcause; clean retry/reauth preserve fixed route/query/body. Source LIST Accept application/json default can be overridden only by explicit safe Go Headers.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordPhysicalAcceptedRejectedAndListErrors`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordRetryKeepsOriginalRejectionAndRequest`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordRetryForbiddenHeadersAndOmittedVersion`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordDetailsScopeQueryPriorityAndOrderedSeed`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordAcceptExplicitAndSelectedAPIChange`.
- Source fields._convert_type int adapter reuses audited DescriptorIntegerJSON, preserving bool-as-int-subclass, arbitrary precision integers, Unicode decimal digit strings and binary64 float truncation while signed/padded/nonnumeric strings and containers become zero. Nondecimal isdigit strings/nonfinite conversion retain actual accepted member or whole-LIST proof; successful descriptor views stay separate from original Wire.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordSourceIntegerDescriptorsAndWireProof`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordDirectDefaultsAliasesAndWire`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordNestedSourceDescriptors`.
- Top-level next uses the owned Python binary64 truthiness hook: decimal underflow is falsey and ends paging, while truthy numeric continuation fails with actual whole-page receipt. Original decimal Wire and independently owned Resource/Envelope/headers survive successful underflow. Falsey next does not hide physical Read/Close causes or one-close ownership.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordNextBinary64TruthinessAndWholePageProof`.
- Explicit bounded Go status policy accepts final200..399 representations; pinned Source raise_from_response returns for any<400. Informational199 is intentionally rejected with actual native status/body/header in member and LIST, without false success or lookup fallback.
  Tests: `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordInformationalStatusIsBoundedGoDifference`, `compute/v2/servers/python_find_contracts_test.go:TestPythonServerFindRecordPassiveIdentityAndOriginalStatusPolicy`.

Python/Go differences:

- The concrete Go IdentityFindOpts and WithIdentityFind... helpers own defaults and snapshots. Existing explicit Ref-based Find remains strict; automatic FindIdentity defaults to ignoring logical absence like the pinned Python source.
- Names unsafe for a raw native member URL are resolved by list only, including NotFoundOnly; GET-only rejects them before HTTP. Names are neither trimmed nor URL-decoded, and every consumed row requires a safe ID.
- Caller Query is an explicit wire-parameter map used by both GET and LIST; the automatic exact name hint is added only to LIST and never replaces an explicitly present caller key. Native success codes, typed extraction and continuation formats remain intact; accepted raw HTTP metadata is not fabricated.
- Pinned find_server (openstack/compute/v2/_proxy.py:1023-1060) declares details/all_projects rather than **query; Go wire Query is an extra capability and is not the Python list-only all_projects policy. The owned Details and AllProjects options now implement those typed defaults and explicit changes: summary/detail is fallback-only, true is LIST-only all_tenants=true, and false omits the automatic query field.
- Typed mode options require explicit audited binding capabilities, including explicit false. The SDK rejects typed/raw cross-project conflicts and does not infer a details fetch for summary matches or modify ordinary List; this Go option-ownership policy makes ambiguous inputs fail before HTTP.
- Pinned find_server proxy exposes only identity/ignore_missing/details/all_projects and no **query. Go returns concrete owned ServerFindRecord snapshots; Query/Headers/Location/list controls are explicit Go conveniences inspired by inherited Resource behavior, not extra Python proxy parameters.
- Safe literal UTF8 string identities are escaped exactly once; arbitrary Resource/class/subclass/session/Connection arguments, full mutable Resource and metadata APIs and Python response/cache/session state are outside this named lookup record. Shared SDK-R1/C1/S1 foundations remain independently tracked rather than claimed complete.
- Outer location reads only recorded native Keystone v2/v3 authentication scope without I/O and permits a snapshotted concrete CloudLocation override; full Python cloud/region/auth-config names are not inferred. Nested Resource JSON constructors omit runtime Connection objects and project null location; nonnull nested connection is rejected. The exact Source nested descriptors and defaults are provided in the documented JSON profile.
- Sequential query options and literal response objects preserve Source dictionary alias order. Bulk Go maps/new callback keys/local filters use sorted keys because Go map has no kwargs insertion order; constructor last-alias and server canonical-query precedence are separate policies.
- Unknown/numeric/timestamp values remain in actual Wire. Source int/bool/dict/list/nested descriptor conversion uses shared audited Unicode16/CPython3.14 and binary64 truthiness; equality keeps shared exact decimals plus bool-number equality, not arbitrary Python float rounding/nonfinite/custom-runtime objects. Float descriptors must be finite JSON.
- Explicit member-only Microversion mirrors inherited Resource.find rather than the limited proxy signature; Headers applies both phases and typed Limit/Marker/MaxItems/Paginated is LIST-only. Raw positional/control and ctor collisions and typed all-projects/raw all_tenants ambiguity are rejected before HTTP. Source list row cls/connection/microversion/_synchronized collision remains an actual whole-page error.
- Fixed origin/collection/filter/request/protocol and one-time identity escaping intentionally reject Source callback/continuation retargeting. Exact relative /v2.1/servers[/detail] aliases are accepted; dictionary links are Go compatibility. Explicit cap/single-page lookup establishes uniqueness only over observed rows.
- Source server descriptors/defaults and actual Wire/Envelope are independent; no Header if_match descriptor is invented for Server. Valid wrongshape/UTF8/physical receipt errors fail with actual proof; malformed UTF8 is stricter than Requests replacement decoding. Successful malformed syntax/empty response retains only seed/default Resource and no fabricated actual Wire.
- Fixed Source docstring describes all_projects as admin-only by default; SDK sends the selected token and LIST all_tenants=True without acquiring/admin-faking permissions, preserving actual Nova rejection. Fixture/race/external consumer main compilation is distinct from authenticated deployment policy/endpoints.
- Pinned Source exceptions.raise_from_response returns for status<400; Go deliberately accepts final200..399 representations and rejects informational100..199 with actual native proof. This bounded transport status difference is documented and covered by explicit member/LIST199 rejection tests, rather than claimed Source status parity.

### `python:block_storage/v3/find_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindDetails`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindAllProjects`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordFindDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordFindAllProjects`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordFindLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordFindMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordFindPaginated`

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
  Tests: `connectiontest/connection_find_identity_test.go:TestConnectionFindIdentitySharesPolicyAcrossHighLevelResources`.
- The audited query getter preserves the native GetResult extractor and declared success codes, reverse ResourceBase, live provider token, configured headers, HTTP middleware and caller context without cloning or mutating the shared service client; native reauthentication replays the same frozen request.
  Tests: `internal/nativefind/get_test.go:TestNativeIdentityGetPreservesQueryNativeResultCodesAndEvidence`, `internal/nativefind/get_test.go:TestNativeIdentityGetFreezesInputsAndUsesLiveProviderReauthentication`, `internal/nativefind/get_test.go:TestNativeIdentityGetAcceptedDecodeAndContextRemainTerminal`, `internal/nativefind/get_test.go:TestNativeIdentityGetRejectsUnsafeRoutesAndClientURLsBeforeHTTP`, `api/find_identity_identity_network_test.go:TestNativeFindIdentityDualPhaseQueryKeepsLiveProviderAndHTTPClient`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesPreserveSourceClientAndContext`.
- Details defaults to true; false selects the summary fallback route and true selects the detail fallback route, without changing direct member GET. A successful GET returns immediately even in summary mode; ordinary List keeps its original adapter, and summary rows are not followed by implicit detail GETs.
  Tests: `resource/find_identity_modes_test.go:TestCollectionFindIdentityModesSelectFallbackWithoutChangingGETOrList`, `resource/find_identity_modes_test.go:TestCollectionFindIdentityTypedModesKeepSuccessfulMemberGET`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesRoutesFrozenQueriesAndLiveSource`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesDefaultsRoutesAndListOnlyFlags`.
- Typed AllProjects defaults to false: nil/false omits the automatic cross-project field, and true adds all_tenants=true only to fallback LIST. Raw all_tenants with no typed value remains a dual-phase wire extension. Explicit typed values conflict with any case-insensitive raw all_tenants key presence, including nil/empty/repeated values, independently of option order; raw details/all_projects and unsupported explicit false/true fail before HTTP, including GET-only policy.
  Tests: `resource/find_identity_modes_test.go:TestCollectionFindIdentityTypedAllProjectsIsListOnlyAndRawQueryRemainsDualPhase`, `resource/find_identity_modes_test.go:TestCollectionFindIdentityModesPreflightUnsupportedAndConflicts`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesDefaultsRoutesAndListOnlyFlags`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesConflictsAndCapabilitiesBeforeHTTP`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesFallbackPoliciesNativeCodesAndRawScopeExtension`.
- Details/AllProjects/IgnoreMissing pointers and query arrays are independently snapshotted at option construction and after custom option application. Last scalar option wins; a bulk option replaces all fields. Captured configuration mutations during GET cannot change fallback mode/query, and shared options and mode iterators can be reused concurrently.
  Tests: `resource/find_identity_modes_test.go:TestCollectionFindIdentityModesSelectFallbackWithoutChangingGETOrList`, `resource/find_identity_modes_test.go:TestCollectionFindIdentityModesFreezeBulkAndCapturedOptions`, `resource/find_identity_modes_test.go:TestCollectionFindIdentityModeOptionsCanBeReusedConcurrently`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesReusableAndConcurrentIndependentSnapshots`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesOwnedSnapshotsLastWinsAndConcurrentReuse`.
- Both fallback modes retain native page codes and whole-page extraction. Duplicate rows and later HTTP/decode/read/transport/cycle errors remain terminal after a match; consumer break does not inspect continuation, context cancellation remains terminal, and native linked continuation behavior is preserved instead of inventing a stricter route or new response evidence.
  Tests: `resource/find_identity_modes_test.go:TestCollectionFindIdentityModesPreserveAllPageErrorsAndCancellation`, `resource/find_identity_modes_test.go:TestCollectionFindIdentityModeIteratorCancellationIsTerminal`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesNativeCodesAndWholePageErrors`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesBreakCancellationAndPreflight`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesCyclesForeignLinksAndLaterFailure`, `internal/nativefind/list_modes_test.go:TestNativeIdentityListModesTransportAndAcceptedReadErrorsStayTerminal`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesSummaryPagesDuplicatesAndTerminalFailures`, `api/find_identity_modes_test.go:TestNativeFindIdentityModesFallbackPoliciesNativeCodesAndRawScopeExtension`.
- Only the two audited Nova/Cinder bindings expose details/all-projects modes. Generated identity hooks use the owned mode iterator while retaining ordinary Get/List controls; inventory records both capabilities only for those bindings. Native List/ListSimple URL/page/extractor body, signature, page schema and receiver-qualified declaration drift are rejected before emission.
  Tests: `internal/cmd/sdkgen/identity_modes_test.go:TestIdentityListModesOnlyEnableAuditedConcretePagerContracts`, `internal/cmd/sdkgen/identity_modes_test.go:TestIdentityListModesEmitFallbackOnlyAndExplicitInventory`, `internal/cmd/sdkgen/identity_modes_test.go:TestIdentityListModesNativeDeclarationAndReceiverGuards`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- GET-first clean400/403/404 fallback defaults to detail while explicit false selects summary; AllProjects defaults/false omit the automatic field and true adds all_tenants=True only to LIST, including successful GET bypass.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordFallbackCodesAndDetailsDefaults`, `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordAllProjectsIsFallbackOnlyAndFalseOmitted`.
- All fallback pages compare exact string ID or name; late duplicates/HTTP errors remain terminal, numeric IDs do not equal string identities, and missing defaults to nil with explicit strict policy.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordExactIdentityAndLateDuplicateOrError`, `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordMissingAndAcceptedTranslation`.
- Owned shared Cinder Volume view projects all37 Body descriptors plus computed location, preserving untyped IDs/timestamps, BoolStr/bool/int/list/dict conversions, ordered response aliases and independent actual Wire/Envelope/header/status.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordGetFirstOwnsSourceViewAndRawEvidence`, `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordMissingAndAcceptedTranslation`.
- Existing CloudLocation computes project/zone from the actual row; explicit location snapshots preserve configured names only for current/falsey projects and never trust HTTP location.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordOwnedLocationUsesRecordedScopeAndRowProjectZone`, `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordGetFirstOwnsSourceViewAndRawEvidence`.
- Source selected default bypasses discovery unchanged; absent default uses shared finite Cinder discovery and min(server maximum,3.71), with distinct discovery proof and physical rejection causes.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordDiscoveryDefaultAndExplicitVersion`, `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordDiscoveryRejectionPreservesPhysicalCauses`.
- Explicit Go Query extensions use only declared Volume server mappings; metadata is both server wire alias and Body-local filter, properties wins only its wire mapping, Requests repeats/null/bool spellings and unknown list drops are preserved, raw cap precedes local filtering.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordQueryClassesOwnedOptsAndRawCap`, `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordQuerySeedCanonicalPriorityAndNullOmission`.
- Shared HTTPLink and short-page marker traversal retain scope, with explicit first-page/cap controls and retained exact large-ID marker independent of public projection.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordPagingShortMarkerAndHTTPLinkKeepScope`, `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordMarkerUsesRetainedUntypedWireID`.
- Member400/403/404/500 and list500 rejected observations plus accepted list200 each retain separate Read/Close/mixed causes, exact HTTP proof and one Close without fallback or observation retry.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordRejectedAndAcceptedPhysicalCauses`.
- Response-close source changes on accepted200/rejected404/500 remain terminal with original HTTP proof and no fallback; invalid inputs/options/conflicting typed scope/JSON/location and callback cancellation fail before HTTP.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordResponseCloseSourceChangeNeverFallsBack`, `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordSelectionAndOptionFailures`.
- Resource.find constructor seed validates Volume BoolStr/int descriptors with shared cindervolume projection before discovery/member HTTP, even if an eventual response would overwrite invalid seed; selected/absent versions both fail with zero HTTP and no fabricated response proof.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordConstructorSeedDescriptorsBeforeDiscoveryOrHTTP`.
- Source status<400 policy accepts every valid100..399 including actual member/list199 with unchanged Resource/Wire/status evidence.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordAcceptsSourceInformationalStatus199`.
- Member/list429/503 mixed Read/Close with a nil native retry callback never replay; sticky source change restored on Close plus custom cancellation remain joined with original native status/method/URL/body/header and one Close.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordRejectedMixedFaultsNilRetryStickySourceAndCancel`.
- LIST constructor raw-row connection/microversion/_synchronized/cls presence conflicts fail before projection, even null/false and after earlier matching pages, with exact accepted HTTP200 body/header/status; Source-stripped self is allowed and direct GET does not run this list check.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordListConstructorCollisionsKeepActualPage`.
- Underlying Resource query extension retains the shared exact-decimal Go policy: 9007199254740992 differs from Source-rounded9007199254740993.0 while ordinary numeric/bool matches survive; bool descriptor1e-400 remains exact-nonzero true rather than Source binary64 underflow false, and actual Wire retains raw bytes.
  Tests: `blockstorage/v3/volumes/python_find_test.go:TestPythonVolumeFindRecordExactDecimalEqualityAndUnderflowAreExplicitGoPolicy`.

Python/Go differences:

- The concrete Go IdentityFindOpts and WithIdentityFind... helpers own defaults and snapshots. Existing explicit Ref-based Find remains strict; automatic FindIdentity defaults to ignoring logical absence like the pinned Python source.
- Names unsafe for a raw native member URL are resolved by list only, including NotFoundOnly; GET-only rejects them before HTTP. Names are neither trimmed nor URL-decoded, and every consumed row requires a safe ID.
- Caller Query is an explicit wire-parameter map used by both GET and LIST; the automatic exact name hint is added only to LIST and never replaces an explicitly present caller key. Native success codes, typed extraction and continuation formats remain intact; accepted raw HTTP metadata is not fabricated.
- Pinned find_volume (openstack/block_storage/v3/_proxy.py:855-893) declares details/all_projects rather than **query; Go wire Query is an extra capability and is not the Python list-only all_projects policy. The owned Details and AllProjects options now implement those typed defaults and explicit changes: summary/detail is fallback-only, true is LIST-only all_tenants=true, and false omits the automatic query field.
- Typed mode options require explicit audited binding capabilities, including explicit false. The SDK rejects typed/raw cross-project conflicts and does not infer a details fetch for summary matches or modify ordinary List; this Go option-ownership policy makes ambiguous inputs fail before HTTP.
- FindRecord supplements existing native typed APIs with an owned Python-shaped VolumeResourceRecord and independent raw HTTP evidence, not a mutable session-bound Resource or Python response/discovery cache.
- Leaf location uses already recorded token scope and actual row project/zone; unavailable Connection cloud/region/configured names remain null unless WithRecordFindLocation supplies them. It reuses shared CloudLocation and CinderVolume translators without extra authentication I/O.
- Query/Headers/Location/MaxItems/Paginated expose underlying Resource policies as Go extensions beyond the pinned proxy missing/details/all_projects arguments. Typed/raw scope conflicts are rejected before HTTP.
- Query descriptor seeds have canonical-attribute priority because a Go map has no kwargs order; incoming HTTP descriptor aliases still consume original JSON order. Non-object actual values fail object local matching without a Python type error.
- Go fixes finite per-call service discovery/route/header policy and rejects unsafe continuation origin/path/filter changes, cycles and negative limits; dictionary links compatibility is a Go extension. Selected explicit client defaults bypass discovery unchanged.
- Input identity segments are validated and URL-escaped; invalid UTF-8 or complete non-object accepted JSON fails with actual response evidence. Unknown fields stay in Wire rather than the declared to_dict view.
- Short-page marker matches Source Requests scalar-ID encoding. Array/object IDs use one Python repr marker in the guarded Go pager instead of Source iterable query expansion; the guide scopes that explicit Go policy.
- Shared EqualPythonJSON uses exact decimal equality rather than Python JSON binary64 rounding: integer9007199254740992 differs from decimal9007199254740993.0 in Go though Source compares them equal. Shared response bool conversion preserves exact nonzero truthiness, so1e-400 projects true in Go and false after Source floating-point underflow. Both boundaries are explicit owned Go policy, tested through actual FindRecord query extension and raw Wire evidence.

### `python:dns/v2/find_recordset`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/dns/v2/recordsets.RecordSetScope.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

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
- Every native/scoped contract fixture keeps its original fixed zone or pool and reverse-proxy ResourceBase through GET, fallback and continuation.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter`, `api/find_identity_native_test.go:TestNativeFindIdentityAllPagesDuplicatesAndTerminalErrorsAfterMatch`.
- The audited query getter preserves the native GetResult extractor and declared success codes, reverse ResourceBase, live provider token, configured headers, HTTP middleware and caller context without cloning or mutating the shared service client; native reauthentication replays the same frozen request.
  Tests: `internal/nativefind/get_test.go:TestNativeIdentityGetPreservesQueryNativeResultCodesAndEvidence`, `internal/nativefind/get_test.go:TestNativeIdentityGetFreezesInputsAndUsesLiveProviderReauthentication`, `internal/nativefind/get_test.go:TestNativeIdentityGetAcceptedDecodeAndContextRemainTerminal`, `internal/nativefind/get_test.go:TestNativeIdentityGetRejectsUnsafeRoutesAndClientURLsBeforeHTTP`, `api/find_identity_identity_network_test.go:TestNativeFindIdentityDualPhaseQueryKeepsLiveProviderAndHTTPClient`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Within a fixed zone, GET by name is rejected (400) then the zone's list is filtered by name, matching ID or name and returning nil by default when absent.
  Tests: `api/r2_dns_cloud_test.go:TestR2DNSRecordSetWaitsAndFindIdentityInZone`.

Python/Go differences:

- The concrete Go IdentityFindOpts and WithIdentityFind... helpers own defaults and snapshots. Existing explicit Ref-based Find remains strict; automatic FindIdentity defaults to ignoring logical absence like the pinned Python source.
- Names unsafe for a raw native member URL are resolved by list only, including NotFoundOnly; GET-only rejects them before HTTP. Names are neither trimmed nor URL-decoded, and every consumed row requires a safe ID.
- Caller Query is an explicit wire-parameter map used by both GET and LIST; the automatic exact name hint is added only to LIST and never replaces an explicitly present caller key. Native success codes, typed extraction and continuation formats remain intact; accepted raw HTTP metadata is not fabricated.
- The concrete parent scope fixes one zone/pool ID for GET, all list pages and fallback; no parent or source client is reconfigured by the child lookup.
- Python Resource.list classifies declared query aliases/unknown names and applies known Body attributes locally; Go sends an explicit wire Query and rejects reserved controls.
- Python Proxy.request may cache GET responses; the Go lookup is stateless.
- Python fetch merges into a seeded Resource and tolerates missing/invalid JSON differently; Go uses strict native GetResult extraction and requires safe nonblank result IDs.
- Python follows RFC Link headers and limit/marker continuation; Go keeps each native pager.
- Python accepts a Zone Resource and passes its ID; Go's scope is created from a zone reference (InZone) and keeps that fixed parent for GET, fallback and continuation.

### `python:load_balancer/v2/find_member`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/pools.API.Members`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/pools.MemberScope.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`

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
- Every native/scoped contract fixture keeps its original fixed zone or pool and reverse-proxy ResourceBase through GET, fallback and continuation.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter`, `api/find_identity_native_test.go:TestNativeFindIdentityAllPagesDuplicatesAndTerminalErrorsAfterMatch`.
- The audited query getter preserves the native GetResult extractor and declared success codes, reverse ResourceBase, live provider token, configured headers, HTTP middleware and caller context without cloning or mutating the shared service client; native reauthentication replays the same frozen request.
  Tests: `internal/nativefind/get_test.go:TestNativeIdentityGetPreservesQueryNativeResultCodesAndEvidence`, `internal/nativefind/get_test.go:TestNativeIdentityGetFreezesInputsAndUsesLiveProviderReauthentication`, `internal/nativefind/get_test.go:TestNativeIdentityGetAcceptedDecodeAndContextRemainTerminal`, `internal/nativefind/get_test.go:TestNativeIdentityGetRejectsUnsafeRoutesAndClientURLsBeforeHTTP`, `api/find_identity_identity_network_test.go:TestNativeFindIdentityDualPhaseQueryKeepsLiveProviderAndHTTPClient`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Fixed-pool Member uses existing LocalStatus policy for typed WithStatus: wire status is omitted, filtering remains local and raw cap precedes that filter. Raw WithQuery(status) alone remains wire-only. Combining typed and raw status keeps the activated local filter and compares the final option value in either order.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityMemberWireAndTypedStatusRemainSeparate`.
- Python-shaped lookup on the fixed pool: ID hit, ID miss with and without ignore-missing, exact name hit/miss and duplicate names behave like Python find semantics under explicit references.
  Tests: `api/py_lb_find_test.go:TestPyLBFindExplicitIDAndNameKeepIgnoreMissingAndDuplicateSemantics`.

Python/Go differences:

- Python member methods take the parent pool as an ID or Pool Resource and put pool_id into the URI; Go passes the pool ID per call or fixes it once in a pools.API.Members scope (an explicit ID needs no lookup request).
- Python find_* tries GET by ID, falls back to a list by name on 400/403/404 and matches ID or name; Go separates resource.ID and resource.Name references, resolves names through an exact-name list and reports duplicate names as an AmbiguousError (Python DuplicateResource).
- Python find_*/delete_* default to ignore_missing=True (find with a deprecation warning); Go Find is strict by default and needs resource.WithIgnoreMissing for the Python default, Remove ignores a missing object by default (WithMissingError selects strict) and native API.Delete keeps a 404 as a response error.
- The concrete Go IdentityFindOpts and WithIdentityFind... helpers own defaults and snapshots. Existing explicit Ref-based Find remains strict; automatic FindIdentity defaults to ignoring logical absence like the pinned Python source.
- Names unsafe for a raw native member URL are resolved by list only, including NotFoundOnly; GET-only rejects them before HTTP. Names are neither trimmed nor URL-decoded, and every consumed row requires a safe ID.
- Caller Query is an explicit wire-parameter map used by both GET and LIST; the automatic exact name hint is added only to LIST and never replaces an explicitly present caller key. Native success codes, typed extraction and continuation formats remain intact; accepted raw HTTP metadata is not fabricated.
- The concrete parent scope fixes one zone/pool ID for GET, all list pages and fallback; no parent or source client is reconfigured by the child lookup.
- Pinned find_member (openstack/load_balancer/v2/_proxy.py:542-569) exposes only name_or_id, pool and ignore_missing; explicit Go wire Query and fallback-policy controls are extensions.
- The fixed Member collection now forwards explicit raw wire status while typed WithStatus retains local filtering. Mixed typed/raw status uses the last value for the activated local filter; it is not the wire-only policy of raw query alone.
- Python find can serve a GET from the connection response cache (openstack/proxy.py request caching) and merges the fetched body into a seeded Resource; Go performs fresh stateless requests and returns a native typed model, with no response cache.
- Python continuation also follows RFC Link headers and limit/marker fallbacks (Resource._get_next_link); Go retains the native pager and its cycle policy for every list page.

### `python:identity/v3/find_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordFindListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordFindMicroversion`

- Safe string identities use the original native getter when query is absent and the audited query getter otherwise; both retain native success codes. Default logical absence is ignored, Compatible falls back only after GET400/403/404, with explicit 404-only/GET-only and strict missing options.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter`, `api/find_identity_native_test.go:TestNativeFindIdentityFallbackPoliciesAndMissingDefaults`.
- Pinned Project find declaration (openstack/identity/v3/_proxy.py:1063-1084) is explicitly opted into a string-ID/name binding. The original getter and query getter both accept only HTTP200; HTTP203 is terminal, with the same native singular envelope and plural linked-page format.
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
- Explicit domain_id reaches direct GET and every fallback page; no auth-domain filter is inferred. Same names across domains remain ambiguous, including matches on different pages, and filtering never bypasses exact SDK identity checks.
  Tests: `api/find_identity_identity_network_test.go:TestNativeFindIdentityKeystoneDomainFiltersAndCrossDomainDuplicates`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Member GET first retains constructor query seed/unknown extension and response alias order; successful member name is not re-matched, clean400/403/404 falls back to declared name/domain/boolean query and local Body filtering.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectFindRecordsGETFirstSeedAliasAndCleanFallback`.
- Default missing is nil, strict missing wraps ErrNotFound, duplicate actual rows across later pages are ErrAmbiguous, and a later rejected page retains actual503 proof instead of returning an earlier match.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectFindRecordsDefaultMissingAmbiguousAndLateHTTPProof`.
- ListBasePath affects only fallback LIST and is ignored by successful GET; explicit per-call find version applies only to GET while fallback keeps selected identity version. Actual199 succeeds with physical receipt.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectFindRecordsListBasePathOnlyAfterFallbackAndVersionPhase`.
- Accepted/rejected200/400/403/404/401/429/498/503 each preserve read/Close/both/source+custom cancellation errors and actual body/header/status with one HTTP and one Close, no fallback/RetryFunc/RetryBackoff/reauth.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsPhysicalReadCloseNative401AndCanceledGuardNeverFallback`.
- Selected route/provider/type/version and option/context guards remain owned through the compound lookup; raw Project view and whole page evidence remain detached.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsOptionsSnapshotAndSourceCancellationPreflight`, `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsDeclaredViewKeepsPhysicalEvidenceAndPresence`.
- Owned Project FindRecord explicitly adopts six raw fetch controls: collection GET/custom envelope preserve Project constructor seed including unknown controls, Project boolean/dict/tag projection and physical Wire/Envelope/header/status; malformed accepted selected envelopes retain ResponseError evidence instead of returning an invented record.
  Tests: `identity/v3/projects/project_fetch_controls_test.go:TestPythonProjectFindRecordFetchControlsCollectionProjectionAndConstructorSeed`, `identity/v3/projects/project_fetch_controls_test.go:TestPythonProjectFindRecordFetchControlsCustomEnvelopeFailureKeepsPhysicalEvidence`.
- Raw headers are GET query mapping keys and become fallback LIST formal headers with typed-header override; nonempty ListBasePath changes fallback only. Project fallback reconstructs list rows without GET constructor controls, preserving Project descriptors. Invalid truthy raw header values fail only after a rejected GET reaches LIST.
  Tests: `identity/v3/projects/project_fetch_controls_test.go:TestPythonProjectFindRecordFetchControlsHeadersAndListRouteAreStageSpecific`, `identity/v3/projects/project_fetch_controls_test.go:TestPythonProjectFindRecordFetchControlsBadHeadersWaitForFallback`.
- Owned raw options and HTTP headers remain detached; skip_cache=false still invokes the stateless reader for each call without an invented cache header. Custom fetch HTTP messages retain actual native500/body/header causes; physical read/Close/cancel faults remain terminal with one response close and no fallback.
  Tests: `identity/v3/projects/project_fetch_controls_test.go:TestPythonProjectFindRecordFetchControlsOwnOptionsRemainStatelessAndKeepErrorCause`, `identity/v3/projects/project_fetch_controls_test.go:TestPythonProjectFindRecordFetchControlsPhysicalFaultsNeverFallback`.

Python/Go differences:

- Legacy native FindIdentity: The concrete Go IdentityFindOpts and WithIdentityFind... helpers own defaults and snapshots. Existing explicit Ref-based Find remains strict; automatic FindIdentity defaults to ignoring logical absence like the pinned Python source.
- Legacy native FindIdentity: Names unsafe for a raw native member URL are resolved by list only, including NotFoundOnly; GET-only rejects them before HTTP. Names are neither trimmed nor URL-decoded, and every consumed row requires a safe ID.
- Legacy native FindIdentity: Caller Query is an explicit wire-parameter map used by both GET and LIST; the automatic exact name hint is added only to LIST and never replaces an explicitly present caller key. Native success codes, typed extraction and continuation formats remain intact; accepted raw HTTP metadata is not fabricated.
- Legacy native FindIdentity: Pinned declaration (openstack/identity/v3/_proxy.py:1063-1084) forwards **query into Resource.find/fetch. Go provides frozen wire query in both phases without silently inferring a domain or converting Body descriptors.
- Legacy native FindIdentity: The audited common-list bridge now forwards explicit raw wire status instead of silently dropping it. This also affects ordinary Resources.List/All raw query, while a statusless model still rejects typed WithStatus before HTTP.
- Concrete functional options own defaults and defensive copies. Runtime Resource class/session builders are SDK-owned; safe UTF-8 routes, nonnegative Go pagination caps, same-origin/path/filter continuation and stronger cycle bounds are deliberate Go policies.
- Source-selected literal link precedence, ignored Keystone dictionary links and limit-on-short-page marker fallback are enabled for this new Project profile without changing existing native List or older engine profiles. Valid RFC HTTP Link remains a documented Go interoperability correction of the pinned Requests uri/url failure.
- SDK-R1/C1/S1 remain separately tracked whole-SDK Connection Resource runtime, optional response cache and session/version negotiation limitations. Direct leaf location is null; typed ProjectReadLocation or Connection bridge supplies configured facts. This named read mapping is not whole-SDK parity or live-cloud validation.
- Owned Project FindRecord opts into the common six raw fetch controls while retaining Project unknown Body and tag/descriptor/link/status policies. Constructor seed controls may remain in the Resource view until overlaid by actual response; Wire and Envelope remain physical response evidence. Other leaf/native find bindings are not implicitly migrated.
- Owned FindRecord uses finite UTF-8 JSON/string identities and safe service-local routes with only %(id)s and %% URI formatting. Fallback headers require string-valued JSON objects; arbitrary Python objects, external URLs and other URI formatters are deliberate exclusions of this Go profile.
- Owned FindRecord has no SDK response cache or mutable Python session/Resource lifecycle. skip_cache is consumed without a wire query or invented cache header and each lookup invokes its HTTP reader; bypassing a caller custom transport cache is not claimed. error_message decorates original Go HTTP causes, without replacing transport/accepted decode/final missing errors or reproducing Python exception classes.

### `python:identity/v3/find_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordFindIgnoreMissing`

- Safe string identities use the original native getter when query is absent and the audited query getter otherwise; both retain native success codes. Default logical absence is ignored, Compatible falls back only after GET400/403/404, with explicit 404-only/GET-only and strict missing options.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter`, `api/find_identity_native_test.go:TestNativeFindIdentityFallbackPoliciesAndMissingDefaults`.
- Pinned User find declaration (openstack/identity/v3/_proxy.py:1379-1400) is explicitly opted into a string-ID/name binding. The original getter and query getter both accept only HTTP200; HTTP203 is terminal, with the same native singular envelope and plural linked-page format.
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
- Explicit domain_id reaches direct GET and every fallback page; no auth-domain filter is inferred. Same names across domains remain ambiguous, including matches on different pages, and filtering never bypasses exact SDK identity checks.
  Tests: `api/find_identity_identity_network_test.go:TestNativeFindIdentityKeystoneDomainFiltersAndCrossDomainDuplicates`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Owned User FindRecord performs literal GET first with constructor seed/response overlay, User declared boolean/list/dictionary descriptors/defaults and dropped unknown Resource fields while Wire/Envelope retain actual response; successful GET returns immediately without name recheck.
  Tests: `identity/v3/users/python_record_find_test.go:TestPythonUserFindRecordGETFirstSeedDescriptorsAndUnknownWire`.
- Clean400/403/404 fall back to all-page Source User list: query aliases transpose, known nonquery Body filters apply locally, unknown kwargs sent to GET are ignored by LIST, explicit name is retained, and exact ID-or-name duplicate/missing/late failure policy applies.
  Tests: `identity/v3/users/python_record_find_test.go:TestPythonUserFindRecordFallbackClassifiesFiltersAliasesAndAllPages`, `identity/v3/users/python_record_find_test.go:TestPythonUserFindRecordMissingDuplicatesAndLateFailures`, `identity/v3/users/python_record_find_test.go:TestPythonUserFindRecordOwnsOptionsAndNamesFallbackPath`.
- Concrete options snapshot JSON/map/header inputs, typed fallback basepath only changes fallback, and canceled context/nil options/invalid values or reserved runtime controls fail without HTTP. The six reachable raw fetch controls are handled by the separately opted-in owned User engine.
  Tests: `identity/v3/users/python_record_find_test.go:TestPythonUserFindRecordPreflightAndReservedControls`, `identity/v3/users/python_record_find_test.go:TestPythonUserFindRecordOwnsOptionsAndNamesFallbackPath`.
- Source-valid rel/href link arrays drive complete fallback paging and late duplicate/error detection; a links mapping containing only next is ignored like the pinned Resource._get_next_link one-key-dictionary iteration, rather than invented as continuation.
  Tests: `identity/v3/users/python_record_find_test.go:TestPythonUserFindRecordFallbackClassifiesFiltersAliasesAndAllPages`, `identity/v3/users/python_record_find_test.go:TestPythonUserFindRecordMissingDuplicatesAndLateFailures`, `identity/v3/users/python_record_find_test.go:TestPythonUserFindRecordIgnoresDictionaryNextLikeSource`.
- Owned User FindRecord consumes requires_id/base_path/resource_response_key/skip_cache/error_message in the GET stage, allowing collection GET and custom response envelopes while preserving physical envelope/header/status evidence and descriptor projection; raw base_path also controls fallback unless nonempty explicit ListBasePath overrides it.
  Tests: `identity/v3/users/user_fetch_controls_test.go:TestPythonUserFindRecordFetchControlsCollectionAndEnvelope`, `identity/v3/users/user_fetch_controls_test.go:TestPythonUserFindRecordFetchControlsBasePathAndFalseyEnvelope`, `internal/neutronfind/find_fetch_controls_test.go:TestFindWithFetchControlsRequiresIDUsesSourceTruthiness`, `internal/neutronfind/find_fetch_controls_test.go:TestFindWithFetchControlsBasePathOwnsBothPhasesAndFallbackOverride`, `internal/neutronfind/find_fetch_controls_test.go:TestFindWithFetchControlsResponseKeyAndAcceptedFailureEvidence`.
- Raw headers remain GET query values but become fallback HTTP headers with case-insensitive precedence over typed headers. Invalid truthy header maps are validated only when fallback is reached; falsey values retain defaults.
  Tests: `identity/v3/users/user_fetch_controls_test.go:TestPythonUserFindRecordFetchControlsHeadersAreStageSpecific`, `identity/v3/users/user_fetch_controls_test.go:TestPythonUserFindRecordFetchControlsHeadersValidateAtListStage`, `internal/neutronfind/find_fetch_controls_test.go:TestFindWithFetchControlsRawHeadersAreGETQueryThenFallbackHeaders`, `internal/neutronfind/find_fetch_controls_test.go:TestFindWithFetchControlsInvalidRawHeadersAreDeferredUntilFallback`.
- The stateless Go reader consumes skip_cache without inventing a wire parameter/cache header and performs fresh reader calls; error_message decorates original HTTP failures without hiding causes, altering clean fallback or replacing final missing errors. Snapshots, unsafe-route rejection and terminal physical read/close/cancel evidence remain explicit.
  Tests: `identity/v3/users/user_fetch_controls_test.go:TestPythonUserFindRecordFetchControlsOwnRawOptions`, `identity/v3/users/user_fetch_controls_test.go:TestPythonUserFindRecordFetchControlsPhysicalFailuresNeverFallback`, `internal/neutronfind/find_fetch_controls_test.go:TestFindWithFetchControlsSkipCacheIsLocalAndFetchesRemainStateless`, `internal/neutronfind/find_fetch_controls_test.go:TestFindWithFetchControlsCustomErrorPreservesNativeFailureAndCleanFallback`, `internal/neutronfind/find_fetch_controls_test.go:TestFindWithFetchControlsOwnsInputAndRejectsUnsafeRoutes`.

Python/Go differences:

- Legacy native FindIdentity: The concrete Go IdentityFindOpts and WithIdentityFind... helpers own defaults and snapshots. Existing explicit Ref-based Find remains strict; automatic FindIdentity defaults to ignoring logical absence like the pinned Python source.
- Legacy native FindIdentity: Names unsafe for a raw native member URL are resolved by list only, including NotFoundOnly; GET-only rejects them before HTTP. Names are neither trimmed nor URL-decoded, and every consumed row requires a safe ID.
- Legacy native FindIdentity: Caller Query is an explicit wire-parameter map used by both GET and LIST; the automatic exact name hint is added only to LIST and never replaces an explicitly present caller key. Native success codes, typed extraction and continuation formats remain intact; accepted raw HTTP metadata is not fabricated.
- Legacy native FindIdentity: Pinned declaration (openstack/identity/v3/_proxy.py:1379-1400) forwards **query into Resource.find/fetch. Go provides frozen wire query in both phases without silently inferring a domain or converting Body descriptors.
- Legacy native FindIdentity: The audited common-list bridge now forwards explicit raw wire status instead of silently dropping it. This also affects ordinary Resources.List/All raw query, while a statusless model still rejects typed WithStatus before HTTP.
- New owned FindRecord is separate from unchanged native FindIdentity and fixed-domain cloud ResolveRecord. It reuses pinned User record profile, Source status/link/paging and loaded JSON equality; User does not allow unknown Body fields, so only Wire/Envelope retain unknown response data. Go returns independent records and explicit JSON/safe-path options rather than mutable Python runtime/session state.
- Owned FindRecord consumes the six raw Resource.fetch controls via WithRecordFindQuery/WithRecordFindOpts, using finite UTF-8 JSON and safe service-local routes. URI formatting is deliberately limited to %(id)s and %%; fallback HTTP headers require string-valued JSON objects, rather than arbitrary Python objects, formatters or external URLs.
- Owned FindRecord has no SDK response cache or mutable Python session/Resource state. skip_cache is consumed and each lookup invokes its HTTP reader without an invented cache header; bypassing caches in a caller custom transport is not claimed. error_message decorates and preserves Go HTTP causes rather than constructing Python exception classes; final fallback missing errors retain their own message.
- Focused Source evidence executes AST-extracted actual find/fetch/list method bodies and response-envelope selection with bounded constructor/request/session/query/exception stubs. It is distinct from full Python runtime, Go local HTTP tests and unperformed live-cloud validation.

### `python:identity/v3/find_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.GroupRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordFindListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordFindLocation`

- Safe string identities use the original native getter when query is absent and the audited query getter otherwise; both retain native success codes. Default logical absence is ignored, Compatible falls back only after GET400/403/404, with explicit 404-only/GET-only and strict missing options.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter`, `api/find_identity_native_test.go:TestNativeFindIdentityFallbackPoliciesAndMissingDefaults`.
- Pinned Group find declaration (openstack/identity/v3/_proxy.py:782-803) is explicitly opted into a string-ID/name binding. The original getter and query getter both accept only HTTP200; HTTP203 is terminal, with the same native singular envelope and plural linked-page format.
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
- Explicit domain_id reaches direct GET and every fallback page; no auth-domain filter is inferred. Same names across domains remain ambiguous, including matches on different pages, and filtering never bypasses exact SDK identity checks.
  Tests: `api/find_identity_identity_network_test.go:TestNativeFindIdentityKeystoneDomainFiltersAndCrossDomainDuplicates`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Literal GET-first Source success projects four untyped nullable Group Body descriptors with declared seed, explicit null and passive response id, default null location; malformed/non-JSON body retains seeded state and actual receipt.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupFindRecordGETSeedAndPassiveNullID`.
- Only fully read/closed clean GET400/403/404 falls back; declared query versus ordered local Body filter, unknown GET-only params and name-key presence keep exact ID OR name matching.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupFindRecordCleanFallbackAndExplicitNamePresence`.
- Logical missing defaults nil, strict missing is ErrNotFound, all pages are consumed, repeated identical IDs remain ambiguous, and late LIST403/503 overrides an earlier match with actual proof.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupFindRecordMissingRepeatedIDAndLateFailure`.
- Per-call microversion applies GET only; fallback keeps native session identity version and optional safe ListBasePath changes only fallback.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupFindRecordsGETOnlyMicroversionAndListBasePath`.
- Explicit cloned location ignores unknown wire project_id/location and preserves configured current project, without trusting response id for routing.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsSourceLocationIsExplicitAndOwned`.
- Immutable options plus sticky Source/context guards run before and after every callback; source-change then restore callback cannot escape and bad preflight creates HTTP0 without fabricated ResponseError.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsOwnedOptionsStickySourceAndCancellation`.
- Physical Read/Close/sticky Source restored by Close/custom cancel matrix for200/400/403/404/401/429/503 stops Retry/Backoff/Reauth/fallback with actual body/header/status and pointer401 wrapper and separately inspected ErrOriginal plus ErrReauth. Accepted200 keeps ResponseError; rejected400+ keeps native ErrUnexpectedResponseCode proof and pointer401 separates ErrOriginal/ErrReauth.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsFaultedActualRejectionsNeverFallbackOrRetry`.
- Clean native401/503/429/498 uses live provider token and captured header/route/query with bodyless GET; successful result retains actual HTTP203.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsCleanNativeRetryUsesLiveAuthAndCapturedHeaders`.
- Owned Group FindRecord explicitly selects six raw fetch controls: collection GET/custom envelope preserve declared seed, passive id and explicit cloned CloudLocation; unknown constructor controls and response extension fields stay out of Resource Body while physical Wire/Envelope/header/status remain intact. Malformed accepted selected envelopes retain actual ResponseError evidence.
  Tests: `identity/v3/groups/group_fetch_controls_test.go:TestPythonGroupFindRecordFetchControlsCollectionDeclaredSeedAndLocation`, `identity/v3/groups/group_fetch_controls_test.go:TestPythonGroupFindRecordFetchControlsCustomEnvelopeFailureKeepsPhysicalEvidence`.
- Raw headers are member GET query mapping keys and fallback LIST HTTP headers with case-insensitive precedence over typed values; fallback-only ListBasePath and selected Group query/local filters remain separate. Bad truthy headers are rejected only when fallback is reached; successful GET still projects Group declared fields.
  Tests: `identity/v3/groups/group_fetch_controls_test.go:TestPythonGroupFindRecordFetchControlsHeadersAndListRouteAreStageSpecific`, `identity/v3/groups/group_fetch_controls_test.go:TestPythonGroupFindRecordFetchControlsBadHeadersWaitForFallback`.
- Raw options/header inputs are copied, repeated skip_cache=false lookups remain stateless reader calls with no invented cache header, and custom HTTP messages preserve native500/body/header causes. Physical read/Close/cancel faults stay terminal after one response Close without fallback.
  Tests: `identity/v3/groups/group_fetch_controls_test.go:TestPythonGroupFindRecordFetchControlsOwnOptionsRemainStatelessAndKeepErrorCause`, `identity/v3/groups/group_fetch_controls_test.go:TestPythonGroupFindRecordFetchControlsPhysicalFaultsNeverFallback`.

Python/Go differences:

- Legacy native FindIdentity: The concrete Go IdentityFindOpts and WithIdentityFind... helpers own defaults and snapshots. Existing explicit Ref-based Find remains strict; automatic FindIdentity defaults to ignoring logical absence like the pinned Python source.
- Legacy native FindIdentity: Names unsafe for a raw native member URL are resolved by list only, including NotFoundOnly; GET-only rejects them before HTTP. Names are neither trimmed nor URL-decoded, and every consumed row requires a safe ID.
- Legacy native FindIdentity: Caller Query is an explicit wire-parameter map used by both GET and LIST; the automatic exact name hint is added only to LIST and never replaces an explicitly present caller key. Native success codes, typed extraction and continuation formats remain intact; accepted raw HTTP metadata is not fabricated.
- Legacy native FindIdentity: Pinned declaration (openstack/identity/v3/_proxy.py:782-803) forwards **query into Resource.find/fetch. Go provides frozen wire query in both phases without silently inferring a domain or converting Body descriptors.
- Legacy native FindIdentity: The audited common-list bridge now forwards explicit raw wire status instead of silently dropping it. This also affects ordinary Resources.List/All raw query, while a statusless model still rejects typed WithStatus before HTTP.
- Concrete Go options own request/defaults, nullable raw Resource and independent actual response channels. Existing native List/Get/FindIdentity semantics are unchanged.
- Already selected Keystone v3 API required; no automatic v2 ensure_service_version conversion or Connection current location inference. Explicit immutable CloudLocation retains current project because Group declares no project_id/zone.
- Fixed origin/collection/query/header/version bounds reject foreign URL/traversal/cycle and malformed identities/markers using actual receipt. Paging amounts are nonnegative Go integers.
- iter.Seq2 can be iterated again and repeats captured requests rather than Python generator single-consumption state. Proxy pure expressions still perform SDK option/source preflight before returning.
- Connection cache, generic synchronized mutable Resource lifecycle and session/version negotiation remain separately tracked SDK-R1/C1/S1 goals. This named Go mapping explicitly uses immutable record snapshots and a stateless response reader, rather than claiming Python runtime/cache parity or live-cloud validation.
- Owned Group FindRecord opts into six raw fetch controls without changing Group unknown-Body rejection, descriptor/status/link policy or explicit cloned CloudLocation. Unknown constructor controls are dropped from Resource Body unlike Project; actual unknown response fields remain in physical Wire/Envelope. Other owned/native find bindings are not automatically migrated.
- Owned FindRecord uses finite UTF-8 JSON/string identities and safe service-local routes with only %(id)s and %% URI formatting. Fallback headers are string-valued JSON objects; arbitrary Python objects, external URLs and other formatter forms are deliberate boundaries of this Go profile.
- skip_cache is consumed without an invented wire query/cache header and each lookup invokes the HTTP reader. There is no SDK response cache, and bypassing caches in a caller custom transport is not claimed. error_message decorates preserved Go HTTP causes without emulating Python exception classes or changing transport/accepted decode/final missing errors.

### `python:identity/v3/find_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- Safe string identities use the original native getter when query is absent and the audited query getter otherwise; both retain native success codes. Default logical absence is ignored, Compatible falls back only after GET400/403/404, with explicit 404-only/GET-only and strict missing options.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter`, `api/find_identity_native_test.go:TestNativeFindIdentityFallbackPoliciesAndMissingDefaults`.
- Pinned Domain find declaration (openstack/identity/v3/_proxy.py:259-276) is explicitly opted into a string-ID/name binding. The original getter and query getter both accept only HTTP200; HTTP203 is terminal, with the same native singular envelope and plural linked-page format.
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
- Pinned find_domain defaults map to Domains.FindIdentity: an ID sends one GET /domains/<id>; a name sends that GET (404) and then GET /domains?name=<name>; no match returns nil for ignore_missing=True and resource.ErrNotFound with WithIdentityFindIgnoreMissing(false).
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeFindDomain`.

Python/Go differences:

- The concrete Go IdentityFindOpts and WithIdentityFind... helpers own defaults and snapshots. Existing explicit Ref-based Find remains strict; automatic FindIdentity defaults to ignoring logical absence like the pinned Python source.
- Names unsafe for a raw native member URL are resolved by list only, including NotFoundOnly; GET-only rejects them before HTTP. Names are neither trimmed nor URL-decoded, and every consumed row requires a safe ID.
- Caller Query is an explicit wire-parameter map used by both GET and LIST; the automatic exact name hint is added only to LIST and never replaces an explicitly present caller key. Native success codes, typed extraction and continuation formats remain intact; accepted raw HTTP metadata is not fabricated.
- Pinned find_domain (openstack/identity/v3/_proxy.py:259-276) accepts only string name_or_id and ignore_missing. Those direct inputs are implemented; Go Query and fallback-policy controls are extensions, not missing Python Resource inputs or **query arguments.
- The audited common-list bridge now forwards explicit raw wire status instead of silently dropping it. This also affects ordinary Resources.List/All raw query, while a statusless model still rejects typed WithStatus before HTTP.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Go decodes the native GetResult and requires a safe nonblank result ID instead of merging a seeded Domain Resource through Resource.fetch/_translate_response; missing or invalid JSON fails in Go, and descriptor defaults/coercion and retained unknown Resource fields differ.
- The list fallback keeps the native Keystone pager (links.next) instead of Python Resource._get_next_link RFC Link-header and limit/marker continuation; Keystone returns next null for domain lists, so the default requests are the same.
- Domain inherits _max_microversion=None, so neither side negotiates a microversion.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:identity/v3/find_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordFindListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordFindLocation`

- Safe string identities use the original native getter when query is absent and the audited query getter otherwise; both retain native success codes. Default logical absence is ignored, Compatible falls back only after GET400/403/404, with explicit 404-only/GET-only and strict missing options.
  Tests: `api/find_identity_native_test.go:TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter`, `api/find_identity_native_test.go:TestNativeFindIdentityFallbackPoliciesAndMissingDefaults`.
- Pinned Role find declaration (openstack/identity/v3/_proxy.py:1743-1764) is explicitly opted into a string-ID/name binding. The original getter and query getter both accept only HTTP200; HTTP203 is terminal, with the same native singular envelope and plural linked-page format.
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
- Explicit domain_id reaches direct GET and every fallback page; no auth-domain filter is inferred. Same names across domains remain ambiguous, including matches on different pages, and filtering never bypasses exact SDK identity checks.
  Tests: `api/find_identity_identity_network_test.go:TestNativeFindIdentityKeystoneDomainFiltersAndCrossDomainDuplicates`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Owned Role FindRecord performs literal GET-first with declared constructor seed/actual response overlay, untyped passive id/default/null and options dictionary projection, unknown Body rejection and explicit cloned location. Selected custom envelopes and malformed accepted-body failures retain physical Wire/Envelope/header/status evidence rather than fabricating successful Resources.
  Tests: `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordCollectionSeedProjectionAndLocation`, `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordCustomEnvelopeFailureKeepsPhysicalEvidence`, `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordGETSeedPresenceAndPhysicalIsolation`.
- Clean GET400/403/404 fallback uses Role declared name/domain_id/limit/marker queries and ordered description/options/links local filters, applies the automatic name hint, and consumes all pages for exact ID-or-name duplicate/missing/late-error behavior.
  Tests: `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordCleanFallbackKeepsDeclaredQueriesAndLocalFilters`, `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordCompleteSearchAmbiguityLateFailureAndMissing`.
- Six raw fetch controls separate member/collection GET route and response envelope from fallback LIST route/headers; raw mapping headers are GET query keys and case-insensitively override typed fallback headers only when LIST is reached, where invalid truthy values are validated.
  Tests: `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordCollectionSeedProjectionAndLocation`, `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordHeadersAndListRouteAreStageSpecific`, `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordBadHeadersWaitForFallback`.
- Find per-call microversion applies GET only, fallback retains source version and optional fallback path; nil/error/source-changing/cancel callbacks stop before later callbacks or HTTP. Copied inputs, stateless repeated reader calls, preserved native custom HTTP error causes and terminal physical read/Close/cancel evidence remain explicit.
  Tests: `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordOwnOptionsRemainStatelessAndKeepErrorCause`, `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordPhysicalFaultsNeverFallback`, `identity/v3/roles/python_record_find_test.go:TestPythonRoleFindRecordOptionLifecycleVersionsAndPreflight`.

Python/Go differences:

- Legacy native FindIdentity: The concrete Go IdentityFindOpts and WithIdentityFind... helpers own defaults and snapshots. Existing explicit Ref-based Find remains strict; automatic FindIdentity defaults to ignoring logical absence like the pinned Python source.
- Legacy native FindIdentity: Names unsafe for a raw native member URL are resolved by list only, including NotFoundOnly; GET-only rejects them before HTTP. Names are neither trimmed nor URL-decoded, and every consumed row requires a safe ID.
- Legacy native FindIdentity: Caller Query is an explicit wire-parameter map used by both GET and LIST; the automatic exact name hint is added only to LIST and never replaces an explicitly present caller key. Native success codes, typed extraction and continuation formats remain intact; accepted raw HTTP metadata is not fabricated.
- Legacy native FindIdentity: Pinned declaration (openstack/identity/v3/_proxy.py:1743-1764) forwards **query into Resource.find/fetch. Go provides frozen wire query in both phases without silently inferring a domain or converting Body descriptors.
- Legacy native FindIdentity: The audited common-list bridge now forwards explicit raw wire status instead of silently dropping it. This also affects ordinary Resources.List/All raw query, while a statusless model still rejects typed WithStatus before HTTP.
- New owned Role FindRecord is separate from unchanged native FindIdentity, Role record list/Cloud search and closed cloud lookup. It owns JSON kwargs, declared constructor seed, Role dictionary/default/untyped projection, query/local filters, Source link/status policy and raw fetch controls. Role drops unknown Body including fetch constructor controls while preserving physical Wire/Envelope; explicit immutable location preserves configured project and normalizes zone to null.
- Owned FindRecord uses finite UTF-8 JSON/string identities and safe service-local routes with only %(id)s and %% URI formatting. Fallback headers require string-valued JSON objects; arbitrary Python objects, external URLs and other formatter forms are deliberate boundaries of this Go profile. RFC Link uses the existing Go interoperability correction of pinned Requests uri/url behavior.
- Immutable records and a stateless response reader deliberately replace mutable Python Resource/session/cache runtime. skip_cache is consumed without a wire parameter or invented cache header; each lookup invokes the reader, without claiming bypass of caller custom transport caches. error_message decorates preserved Go HTTP causes rather than emulating Python exception classes, and leaves transport/accepted decode/final missing errors separate. SDK-R1/C1/S1 and live-cloud/release completion remain separately tracked.

### `python:image/v2/find_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.FindImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageRecordIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageRecordListOptions`

- The pinned direct proxy accepts a string name_or_id and ignore_missing=True. Safe strings use native GET first; only compatible GET400/403/404 fall back to the normal list, with explicit strict, 404-only and GET-only Go options. Root-object GET200 extraction preserves the native Image model, additional Properties and import/store response headers.
  Tests: `api/find_identity_images_test.go:TestNativeFindIdentityImagesGETContractsAndNativeModel`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesMissingDefaultsAndFallbackBoundaries`.
- Only a complete successful normal-list absence triggers a second native LIST. It starts from the frozen caller query, overlays mandatory os_hidden=true, omits the automatic name hint and never repeats GET. Explicit name filters and repeated/nil/empty query-key precedence survive; caller hidden=false is overridden only for that second search.
  Tests: `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListWaitsForCompleteSuccessfulAbsence`, `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListPreservesCallerQueryAndMandatoryOverlay`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesHiddenPassUsesOriginalCallerQuery`.
- Both searches consume all native ImagePage continuations and compare exact ID OR name. Repeated-ID duplicates, malformed whole-page native extraction, invalid result IDs, cycle, late HTTP/decode errors and cancellation remain terminal. Normal errors never start the hidden pass; list403/404 are never ignored.
  Tests: `api/find_identity_images_test.go:TestNativeFindIdentityImagesNormalSearchTerminalNeverRetriesHidden`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesHiddenPagesAndTerminalFailures`, `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListErrorsAndCancellationAreTerminal`.
- IgnoreMissing is applied only after both lists complete successfully without a match; strict absence has no suppressed GET400/403 cause. GET-only never performs either list. Unsafe names retain their literal list-only policy; SDK control keys, unsupported details/all_projects and canceled/nil context fail before HTTP.
  Tests: `api/find_identity_images_test.go:TestNativeFindIdentityImagesMissingDefaultsAndFallbackBoundaries`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesPreflightUnsafeNamesAndCancellation`, `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListStrictAndFallbackPolicies`, `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListPolicyPreflightAndOtherOperationsStayUnchanged`.
- SDK policy metadata and caller options are deep snapshots, including callback mutation boundaries and concurrent reuse. Generated leaf and handwritten Image collections preserve reverse ResourceBase, original live provider token, headers, HTTP middleware and caller context across GET and both searches. Ordinary Get/List/explicit Ref lookup are unchanged.
  Tests: `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListSnapshotsAndConcurrentReuse`, `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListFreezesOriginalQueryAcrossCallbacksAndUnsafeNames`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesQuerySnapshotsAndConcurrentReuse`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesKeepsLiveProviderHeadersAndCallerContext`, `connectiontest/connection_find_identity_image_test.go:TestConnectionImageFindIdentityHiddenSearch`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- FindImageRecord performs a literal escaped ID GET with the owned Image projector; only clean final native400/403/404 enter exact ID-or-name listing. A complete successful normal-list absence triggers a hidden list using the frozen original caller query plus mandatory os_hidden=True and without the auto-name hint. Visible duplicates and normal-list failures never enter hidden lookup.
  Tests: `image/image_record_find_test.go:TestFindImageRecordDefaultHiddenPhases`, `image/image_record_find_test.go:TestFindImageRecordErrorsNeverBecomeHiddenSearch`.
- Actual200..399 GET returns the owned seeded record, including tolerated invalid JSON; accepted malformed shapes/conversions and nonaccepted nonselected errors remain terminal. Default missing returns nil only after both successful empty searches; strict IgnoreMissing=false returns ErrNotFound without a suppressed GET cause.
  Tests: `image/image_record_find_test.go:TestFindImageRecordAcceptedGetAndAbsence`, `image/image_record_find_test.go:TestFindImageRecordErrorsNeverBecomeHiddenSearch`.
- Both fallback searches consume the shared owned list pager, compare exact passive id/name, continue after the first candidate to prove uniqueness, and discard it after duplicates or later page/row/transport failures. Invalid passive row IDs may be compared but never form a new URI. The selected row is not re-fetched.
  Tests: `image/image_record_find_test.go:TestFindImageRecordDefaultHiddenPhases`, `image/image_record_find_test.go:TestFindImageRecordErrorsNeverBecomeHiddenSearch`, `resource/find_identity_response_id_test.go:TestCollectionFindIdentityPassiveIDsAreOnlyCompared`.
- Rejected selected400/403/404 body Read/Close/transport/cancellation/native-hook/source failures are terminal rather than hidden absence. PreserveCleanRetry is applied only to these selected Glance lookup rejections; actual rejected evidence and underlying causes are retained.
  Tests: `image/image_record_find_test.go:TestFindImageRecordRejectedOwnershipFaultsAreTerminal`, `image/image_record_find_test.go:TestFindImageRecordErrorsNeverBecomeHiddenSearch`.
- One prepared source/location snapshot and one concrete option evaluation span GET, normal and hidden pages. Nested List options and filters affect lists only; List.Headers apply to all phases and Find.Headers override them. Option slices/maps/pointers are copied; source checks between nested callbacks stop mutation before HTTP.
  Tests: `image/image_record_find_test.go:TestFindImageRecordOptionsAndLocationOwnAllPhases`, `image/image_record_find_test.go:TestFindImageRecordNestedCallbackSourceChangeStopsBeforeHTTP`.
- Root Connection executes owned literal get, two-page list and hidden find through the same source and current location, retaining independent declared/wire channels and live original auth. Existing native FindIdentity behavior and direct typed Image APIs are preserved as separate profiles.
  Tests: `connectiontest/connection_image_records_test.go:TestConnectionImageRecordsShareSourceLocationAndHiddenDiscovery`, `image/image_record_find_test.go:TestFindImageRecordOptionsAndLocationOwnAllPhases`.

Python/Go differences:

- Go exposes SDK-owned concrete options and additional raw wire query on FindIdentity; the pinned direct find_image declaration (openstack/image/v2/_proxy.py:1013-1033) accepts only name_or_id and ignore_missing. Python query alias is_hidden is not automatically transposed; Go uses os_hidden. details/all_projects are not capabilities of this Image binding.
- Unsafe route strings use literal list-only lookup and invalid identities/result IDs fail before further HTTP. Go keeps the native Image extractor/pager and full-page native decode policy; this does not claim Python descriptor coercion, arbitrary-precision unknown numeric fields or every inherited continuation form.
- Earlier contracts and differences above describe the preserved native/typed API profiles. The additional owned ImageRecord default profile supplies the missing declared Image projector, constructor packing, descriptor conversion, aliases, generic paging and lookup behavior described by the new contracts below; existing strict typed/native behavior is unchanged.
- The added ImageRecord APIs are fresh immutable Go snapshots for the named default profile. They do not mutate a supplied Python-like Resource or implement its dirty state, same-instance fetch, configured Adapter cache/conditional GET, dynamic session/microversion/base-path replacement, or arbitrary Python descriptor objects. These full SDK contracts remain explicitly tracked as SDK-R1, SDK-C1, SDK-J1 and SDK-S1 in docs/implementation-plan.md; API mapping here does not mean complete Python runtime equivalence or live cloud validation.
- FindImageRecord List options are a Go extension; the pinned public find_image signature accepts only name_or_id and ignore_missing. Find uses the owned JSON-domain projector/pager rather than the preserved native Image extractor, allows safe escaped slash/space IDs, and keeps unsafe control/dot strings preflight errors. Arbitrary Python Resource identity and session/cache behavior remain common SDK work.

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
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsLookupClosesBeforeEnrichment`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsUsesResolvedIDAndOwnsReturnedModel`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsInlineAndNativeNilEmptyDefaults`, `connectiontest/connection_find_identity_flavor_test.go:TestConnectionFlavorFindIdentityExtraSpecs`.
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

### `python:network/v2/find_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindIgnoreMissing`

- Pinned direct find_router (openstack/network/v2/_proxy.py:5570) accepts a string, ignore_missing=True and query. Safe identities use native GET200 /routers/id with the router envelope and native nested model/timestamp extraction. GET203/204, accepted decode and missing result-ID errors remain terminal without fallback.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsGETModelsAndNativeCodes`.
- Compatible GET400/403/404 may fall back, with explicit 404-only/GET-only policies. Logical absence defaults to nil,nil and strict uses ErrNotFound after successful full native absence; LIST403/404 is never ignored. Native list200/204/300 semantics remain distinct from the member getter.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsFallbackQueryAndMissingPolicies`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsAllPagesAndTerminalObservations`.
- Caller raw query is frozen across both GET and LIST with repetitions, explicit nil/empty name precedence, vendor fields and concurrent option reuse. Only LIST receives an absent-key literal name hint. No project/domain filter is inferred; exact ID OR name matching and cross-project/repeated-ID ambiguity consume all native continuation pages, including later HTTP/decode/cycle errors.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsFallbackQueryAndMissingPolicies`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsQuerySnapshotsAndConcurrentReuse`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsAllPagesAndTerminalObservations`.
- Generated leaf and cached Connection.Network.API collections retain the original live provider, token, headers, HTTP middleware, microversion, caller context and reverse ResourceBase. Unsafe names are literal list-only; GET-only unsafe names, nil/canceled contexts and explicit Details/AllProjects/ExtraSpecs true or false fail before HTTP.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsLiveSourceAndCancellation`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsUnsafeNamesAndCapabilityPreflight`.
- Ordinary Resources.List/All preserve explicit raw status/fields/shared/vendor query and native paging while MaxItems applies before local name filtering and Paginated(false) stops continuation without a wire limit. Explicit Ref ID/Name lookup remains separate, with no automatic identity defaults or member lookup added to ordinary lists.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsOrdinaryCollectionsQueriesAndControls`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Exact pinned Router MRO: revision_number shadows NetworkResource with wire revision; ordered canonical/wire updates, project tenant fallback, null/false/zero, tag [] defaults, unknown extensions and untyped dates. Integer/list/dict/bool casts follow Source, with binary64 truthiness and integer casts.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindLiteralRawQuerySeedsAndHTTPHeader`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadSparseMROAliasesAndDescriptorCasts`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadIntegerUnicodeFloatAndExactLocalEquality`.
- Member raw kwargs both fetch and seed Resource; literal escaped one-segment route, None query omission and accepted parse failure/bodyless seed/default preservation. Canonical underscore Header spelling is exact, wire Header spelling case-insensitive; multi-value HTTP If-Match coalesces comma-space before list cast. Resource/Wire/whole Envelope/Header/status are independent snapshots and forged location projects null.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindLiteralRawQuerySeedsAndHTTPHeader`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindHeaderConstructorSpellingAndNullSeeds`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadMemberAndListStatusReceiptBoundaries`.
- Exact server query aliases and canonical precedence, original repeated fields, tag aliases, local canonical Body filters, unknown and Header-only List kwargs discarded before marshal. Nested dictionary falsey/subset/empty semantics, descriptor errors, exact-decimal local numeric equality and constructor collisions preserve whole page proof.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListExactAliasesRawFieldsTagsAndLocalFilters`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListConstructorDecodeAndLocalFilterFailures`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListUnknownsDiscardedAndNilMapComposition`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadIntegerUnicodeFloatAndExactLocalEquality`.
- GET-first Find permits only clean 400/403/404 fallback, adds automatic name hint only when name absent, distinguishes None presence, searches full pages for exact ID/name, defaults ignore_missing true, and keeps later duplicate/forbidden errors terminal.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindCleanFallbackNamePresenceAndMissing`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindWholeSearchRejectsLateDuplicateOrListError`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListPartialIterationAndAllDropLateError`.
- Lazy complete pager, raw cap before filters/zero-limit cap hint, single page/early break, same-origin/same-routers collection/original filters, raw marker unaffected by consumer changes, short-page fallback, binary64 falsey numeric next, and All discarding partial records on late error.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListRawCapSinglePageAndEarlyStop`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListContinuationScopeAndRawMarkerOwnership`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListFalseyNextAndWholePageProof`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListPartialIterationAndAllDropLateError`.
- Completed 200..399 and every actual 400..599 retain real status/body/Header. Malformed accepted shape/UTF8/descriptor/constructor and local filter failures preserve response proof. Physical Read/Close/custom-cancel/Source faults are terminal, including actual faulty404 no fallback and401 public ErrOriginal/ErrReauth causes, zero willing retry/reauth callbacks and one HTTP/Close.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListConstructorDecodeAndLocalFilterFailures`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadMemberAndListStatusReceiptBoundaries`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadPhysicalFaultsNeverInvokeRetryOrReauth`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadSourceMutationDuringReadRemainsTerminalAfterRestore`.
- Concrete JSON options clone maps/bytes/pointers/callback outputs; nil-map callback composition and unknown-before-marshal behavior, original MarshalerError/cause, selected Source capture before callbacks and unsafe identity/control/nil preflight avoid HTTP. Eight independent APIs concurrently reuse immutable options and isolated record evidence.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadOwnedOptionsMarshalCausesAndPreflight`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListUnknownsDiscardedAndNilMapComposition`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadReusableOptionsConcurrentOwnedViews`.
- Find per-call microversion is member-only; fallback all pages use captured selected default. Standalone List override applies every page, including explicit empty omission without Source mutation. Clean retry and live auth token refresh remain available while protocol/condition/version/body injection and protected Header omission preserve first physical rejection.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindVersionOverrideIsMemberOnly`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadRetryVersionAbsenceAndLiveTokenOwnership`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadExplicitListVersionAppliesToEveryPage`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadRetryCannotOmitOrInjectOwnedProtocolHeaders`.

Python/Go differences:

- Go provides SDK-owned concrete IdentityFindOpts and WithIdentityFind... defaults/snapshots. FindIdentity ignores logical absence by default like pinned Python, while existing explicit Ref Find stays strict. Unsafe names are list-only and every returned/consumed model needs a safe nonblank ID.
- Go raw wire Query is used by both GET and LIST and preserves unknown keys and repetitions; Python list query transposition/local Body filtering is not implicit. Router is_admin_state_up/is_distributed/is_ha and SecurityGroup is_shared require explicit wire admin_state_up/distributed/ha/shared, with tag aliases also explicit.
- Neither binding provides typed Details/AllProjects/ExtraSpecs, hidden lookup, list defaults or automatic microversion changes. Caller project filters are preserved without inference; declared native model/extractor/plural_links policies remain intact.
- The audited Router builder bridge preserves whole raw query in ordinary Resources.List/All as well as FindIdentity, including unknown keys, repetitions and explicit empty values. Its public native typed List API remains intact.
- Go accepts string identity and immutable owned JSON records; mutable Python Resource/Connection/cache/Adapter lifecycle and computed location remain separately tracked. RouterReadRecord aliases the existing RouterActionRecord.
- JSON-domain concrete options and deterministic bulk-map ordering replace arbitrary Python runtime kwargs. Deprecated JMESPath, allow_unknown_params and __conflicting_attrs restoration plus SDK-owned constructor/route/session controls are explicitly bounded preflight errors, not full runtime parity.
- General local numeric equality deliberately uses shared exact-decimal JSON rather than all CPython binary64 rounding/nonfinite/custom-object comparisons. Descriptor integer float cast and bool/numeric-next truthiness use binary64; Unicode integer-string tables use Unicode16 and do not emulate configurable Python string digit caps.
- Completed status200..399 is supported; arbitrary adapters final informational100..199 are outside this Go completion profile. Actual400..599 and all physical/Source/cancel causes retain actual receipts.
- Continuations require same origin, same fixed routers collection and original filters. Dictionary link forms and exact versioned /v2.0/routers alias are explicit Go conveniences; foreign/rewrite/cycle failures preserve actual whole-page proof.
- Safe Header controls are explicit Go policy. Per-call Find version affects only member GET and List version applies all pages; explicit empty version omits Header and selected client remains untouched.
- Neutron27 stock get_router permits project ADMIN_OR_PROJECT_READER; distributed/ha attributes have separate ADMIN branches. Sparse user responses retain Source missing null; this implementation does not elevate access or prove live deployment authorization/extensions/backend support.

### `python:network/v2/find_security_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordFindLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordFindMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordFindMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordFindPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordFindMicroversion`

- Pinned direct find_security_group (openstack/network/v2/_proxy.py:6598) accepts a string, ignore_missing=True and query. Safe identities use native GET200 /security-groups/id with the security_group envelope and native nested model/timestamp extraction. GET203/204, accepted decode and missing result-ID errors remain terminal without fallback.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsGETModelsAndNativeCodes`.
- Compatible GET400/403/404 may fall back, with explicit 404-only/GET-only policies. Logical absence defaults to nil,nil and strict uses ErrNotFound after successful full native absence; LIST403/404 is never ignored. Native list200/204/300 semantics remain distinct from the member getter.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsFallbackQueryAndMissingPolicies`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsAllPagesAndTerminalObservations`.
- Caller raw query is frozen across both GET and LIST with repetitions, explicit nil/empty name precedence, vendor fields and concurrent option reuse. Only LIST receives an absent-key literal name hint. No project/domain filter is inferred; exact ID OR name matching and cross-project/repeated-ID ambiguity consume all native continuation pages, including later HTTP/decode/cycle errors.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsFallbackQueryAndMissingPolicies`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsQuerySnapshotsAndConcurrentReuse`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsAllPagesAndTerminalObservations`.
- Generated leaf and cached Connection.Network.API collections retain the original live provider, token, headers, HTTP middleware, microversion, caller context and reverse ResourceBase. Unsafe names are literal list-only; GET-only unsafe names, nil/canceled contexts and explicit Details/AllProjects/ExtraSpecs true or false fail before HTTP.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsLiveSourceAndCancellation`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsUnsafeNamesAndCapabilityPreflight`.
- Ordinary Resources.List/All preserve explicit raw status/fields/shared/vendor query and native paging while MaxItems applies before local name filtering and Paginated(false) stops continuation without a wire limit. Explicit Ref ID/Name lookup remains separate, with no automatic identity defaults or member lookup added to ordinary lists.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsOrdinaryCollectionsQueriesAndControls`.
- SecurityGroup alone bridges the native concrete ListOpts limitation with an SDK-owned raw query pager using the original ServiceClient, SecGroupPage, ExtractGroups and StreamWithControl. Query and returned rows are independently owned; native full-page decode beyond a local cap, HTTP/transport/cycle/context failures remain observable.
  Tests: `internal/nativefind/security_groups_test.go:TestNativeSecurityGroupsPreserveQueriesModelsAndLiveSource`, `internal/nativefind/security_groups_test.go:TestNativeSecurityGroupsLocalControlsStopBeforeContinuation`, `internal/nativefind/security_groups_test.go:TestNativeSecurityGroupsCodesAndWholePageErrors`, `internal/nativefind/security_groups_test.go:TestNativeSecurityGroupsContinuationAndFailuresAreTerminal`, `internal/nativefind/security_groups_test.go:TestNativeSecurityGroupsInvalidSourceContextAndControlBeforeHTTP`, `internal/nativefind/security_groups_test.go:TestNativeSecurityGroupsConcurrentReuseHasOwnedRowsAndQuery`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Member-first lookup preserves raw kwargs and Source constructor seed independently from actual HTTP row. Source successful statuses200..399 are supported; malformed/empty JSON retains seed, valid nonobjects and malformed UTF8 retain accepted response proof.
  Tests: `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordGetFirstRawKwargsSeedAndWire`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordEmptyMalformedAndWrongShapes`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordNullableIdentityAndLiteralEscape`.
- Only clean native400/403/404 including an unchanged clean native retry permit fallback; successful full absence defaults nil,nil and strict ErrNotFound does not attach suppressed initial rejection. Fallback scans exact ID OR name across all observed pages and retains late duplicates/errors. Group has automatic name hint; Rule inherited name is compared but has no server hint.
  Tests: `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordFallbackCodesMissingAndCleanRetry`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordLateDuplicateAndPageFailure`.
- GET keeps original kwargs while LIST applies Source server/alias classification and declared nonserver Body filters. Client query alias wins even as null; unknown kwargs seed/member query only; consumed all_projects has no Neutron wire effect. Rule name-null is a local filter, Group name-null omits automatic/server hint.
  Tests: `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordQueryAliasesScopeAndLocalFilters`.
- Record-owned Body/default/header descriptor view and actual Wire/Envelope/status/header stay independent. Header wire if-match aliases are case-insensitive while canonical if_match is exact; IF_MATCH remains unknown Body in constructor/LIST and is not synthesized from actual HTTP Header. Ordered aliases retain last-value precedence. LIST cls/connection/microversion/_synchronized constructor conflicts retain actual whole-page proof.
  Tests: `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordGetFirstRawKwargsSeedAndWire`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordListHeadersAndConstructorControls`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordNullableIdentityAndLiteralEscape`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordHeaderAliasDomains`.
- Source dictionary filtering preserves falsey/empty/nested subset semantics, truthy scalar empty-object matches, mapping.get failures and boolean-number equality. Dictionary scalar decimal truthiness follows Python binary64 underflow/overflow; shared exact-decimal JSON equality is the documented numeric profile.
  Tests: `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordDictionaryTruthinessAndEquality`.
- Shared REST pager preserves plural links/top-level next/HTTP Link, exact Neutron versioned relative aliases, singleton rows, stop-on-empty and explicit limit short-page marker fallback. Raw caps precede local filters and unconsumed bad rows/next parsing. Fixed origin/collection/filter/cursor guards retain page proof.
  Tests: `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordMarkerCapAndSinglePage`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordPaginationRepresentationsAndGuards`.
- Concrete options own maps/raw bytes/pointers/order and retained callback values. Source positional/control ignore_missing/name_or_id/session/cls and constructor conflicts, unsafe literal IDs, malformed seed, negative typed list controls and forbidden auth/version headers fail before HTTP. Context custom and option/marshal causes remain joined; captured source changes are terminal. Reusable options/records remain independent during concurrent use.
  Tests: `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordInvalidControlsAndDescriptorFailBeforeHTTP`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordOptionsSnapshotCancellationAndMarshal`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordConcurrentOptionAndRecordOwnership`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordSourceArgumentCollisions`.
- No class-ceiling discovery is added. Pinned Resource.find applies explicit per-call microversion only to member GET; fallback LIST retains captured service default, including independently owned selected version header. Phase headers/value or absence stay fixed; Source LIST Accept application/json overrides configured service Accept unless an explicit safe Go header option is supplied. Original selected client remains unchanged.
  Tests: `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordProtocolAbsenceAndPerCallVersion`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordRetryKeepsOriginalRejectionAndRequest`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordMicroversionPhasesAndAcceptDefault`.
- Accepted/rejected member and list Read/Close/custom cancel/sticky source restoration errors keep actual status/body/header/causes with one physical close and prevent incorrect lookup fallback/resend. Native request/protocol mutation retains the original HTTP rejection; clean retry and token reauthentication keep route/query/body. Required-absent Authorization/X-Service-Token/framing/hop/foreign-version headers and selected-version OmitHeaders are protected on both member and list native retries; original rejection and custom callback causes remain intact before another physical attempt.
  Tests: `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordPhysicalAcceptedRejectedAndListErrors`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordRetryKeepsOriginalRejectionAndRequest`, `network/v2/extensions/security/groups/python_find_contracts_test.go:TestPythonSecurityGroupFindRecordRetryForbiddenHeadersAndOmittedVersion`.

Python/Go differences:

- Go provides SDK-owned concrete IdentityFindOpts and WithIdentityFind... defaults/snapshots. FindIdentity ignores logical absence by default like pinned Python, while existing explicit Ref Find stays strict. Unsafe names are list-only and every returned/consumed model needs a safe nonblank ID.
- Go raw wire Query is used by both GET and LIST and preserves unknown keys and repetitions; Python list query transposition/local Body filtering is not implicit. Router is_admin_state_up/is_distributed/is_ha and SecurityGroup is_shared require explicit wire admin_state_up/distributed/ha/shared, with tag aliases also explicit.
- Neither binding provides typed Details/AllProjects/ExtraSpecs, hidden lookup, list defaults or automatic microversion changes. Caller project filters are preserved without inference; declared native model/extractor/plural_links policies remain intact.
- The raw-query pager deliberately extends ordinary common Resources.List/All beyond the concrete native ListOpts whitelist, including status and vendor keys. The public native typed List and its existing ListOpts contract are retained.
- Go returns an owned Source-shaped Resource/Wire/Envelope/Header/StatusCode record from a validated literal string identity, rather than mutating/accepting arbitrary Python Resource/class/subclass/session objects. Connection location is null; global Resource/cache/session SDK-R1/C1/S1 work remains separately tracked.
- Sequential query options retain Source constructor kwargs insertion order; bulk Go map/unregistered callback keys and multiple local filters use sorted keys because Go map has no kwargs order. Ordered literal JSON response alias precedence is retained.
- Named Microversion retains the Source GET-only inherited override; fallback LIST uses captured service default. Header applies explicitly in both phases and typed fallback Limit/Marker/MaxItems/Paginated are Go controls. Raw ignore_missing/name_or_id/session/cls/id positional/control/constructor conflicts and route/cache/response-key/header/runtime controls are rejected before HTTP; supported controls use named options. all_projects is consumed without Neutron wire effect.
- Fixed origin/collection/filter/request/protocol and literal one-time escaping guards intentionally reject Source callback/continuation retargeting; raw pagination values are validated by the shared pager. Dictionary links are a Go compatibility extension. Explicit cap/single-page returns only the observed subset and cannot rule out unobserved duplicates.
- Unknown values, original numeric tokens/timestamps and nested extensions stay lossless in Wire. Source int descriptors use audited Unicode16/CPython3.14 profile; bool and dictionary scalar truthiness use binary64 underflow/overflow. Shared JSON equality uses exact decimals plus Python boolean-number equality rather than all Python float rounding/nonfinite/custom-object runtime behavior.
- Accepted malformed UTF8 fails rather than Requests replacement decoding; valid wrong shapes and physical body failures retain actual ACK. Fixture/race and external main compilation are distinct from authenticated deployment Neutron authorization/endpoint validation.

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

### `python:compute/v2/get_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindExtraSpecs`

- The documented explicit-ID get_flavor mapping combines FindFallbackNever with IgnoreMissing(false). Native Flavor GET200 returns the typed ID/name/RAM/disk/VCPU/swap/bandwidth/public/ephemeral/description fields; GET400/403/404 remain errors without a flavor list or extra-specs request under the Never/strict policy.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsMissingPoliciesAndUnsafeNames`.
- Omitted or false ExtraSpecs makes no extra request. True fetches the returned canonical ID once when inline specs are missing/null/empty, retains inline nonempty specs without another GET, preserves unrelated Flavor fields and forwards no caller query to enrichment. Missing/null extra_specs retain native nil-map behavior while an explicit object remains nonnil; the selected Compute version is not upgraded to2.61.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`.
- Extra-spec HTTP403/404/203/204, typed decode, read/transport and cancellation failures are terminal and cannot trigger another lookup or become ignored missing. Native GET203/204, invalid typed Flavor fields, null result or unsafe returned ID are rejected before enrichment or fallback.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsEnrichmentFailuresAreTerminal`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`.
- Nil/canceled contexts, nil/unsupported/invalid options and an unsafe direct route under FindFallbackNever fail without HTTP. The shared direct/enrichment engine preserves caller context, current provider token, configured middleware/headers, endpoint/resource-base and selected microversion rather than adding a new service or discovery policy.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsKeepsLiveProviderVersionAndContext`.

Python/Go differences:

- Python get_flavor accepts an ID or Flavor Resource; this Go mapping requires an explicit safe ID string and the documented Never/strict options. Ordinary FindIdentity retains its independent list-fallback defaults. This row neither closes find_flavor nor treats native flavors_links paging as the whole Python inherited continuation contract.
- The result is native Flavor, with native typed field/swap extraction and a required safe nonblank returned ID, rather than a seeded mutable Python Flavor Resource. Native missing/null ExtraSpecs are nil maps; missing IsPublic is false rather than the source descriptor default true. Python ID/name/original-input fallbacks, nullable/unknown/location descriptors and connected mutable state are deliberate DTO differences.
- Go retains the selected client microversion and native strict200 GET/specs policy, without automatic source Resource discovery/version selection. Full generic Resource/session/cache APIs remain in the SDK goal, independently of this explicit-ID GET and optional enrichment. Caller raw query and Find controls are Go engine extensions, not extra required arguments of Python get_flavor.

### `python:compute/v2/find_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`

- find_aggregate sends GET /os-aggregates/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_aggregate lists /os-aggregates and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- IdentityLiteralGet sends the literal identity through the query getter even though the native Get takes an integer, so os-aggregates/rack-a is requested before the list; explicit Find(resource.ID) keeps its numeric check.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindAggregateSendsLiteralIdentityToMemberRoute`, `resource/find_identity_name_test.go:TestCollectionFindIdentityLiteralGetUsesQueryHookWithoutQuery`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python compares the integer Aggregate id with the string name_or_id in the list fallback, so a numeric string never matches there; Go compares the ID as text after a GET 400 or 403.
- Python negotiates min(Aggregate _max_microversion 2.81, server maximum) per request; Go sends the microversion configured or negotiated on the Connection client.
- Pinned find_aggregate declares no **query; a Go caller query reaches the GET but the native single-page list rejects it with ErrUnsupported.

### `python:identity/v3/find_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`

- find_policy sends GET /policies/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_policy lists /policies without a name query and matches the row ID only, because Keystone returns no name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- An IdentityName that returns an empty string makes FindIdentity match IDs only in the list fallback.
  Tests: `resource/find_identity_name_test.go:TestCollectionFindIdentityNameOverrideMatchesAlternateFieldOrIDOnly`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- The native model has no name field, so a nonstandard name key in a Keystone response is not compared; Python Resource.name would match it.
- Pinned find_policy declares no **query; Go still accepts resource.WithIdentityFindQuery, which Python cannot send.

### `python:network/v2/find_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_ip sends GET /floatingips/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_ip lists /floatingips without a name query and matches the row ID or floating_ip_address (Python FloatingIP.name); any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- resource.WithIdentityFindQuery values are sent on both the member GET and the /floatingips list fallback, before the automatic name hint.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.
- resource.Adapter.IdentityName lets FindIdentity compare floating_ip_address while explicit Find(resource.Name) stays unsupported.
  Tests: `resource/find_identity_name_test.go:TestCollectionFindIdentityNameOverrideMatchesAlternateFieldOrIDOnly`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.
- Only FindIdentity compares floating_ip_address; Find(ctx, resource.Name(x)) still returns ErrUnsupported because the native model has no Name field.

### `python:network/v2/find_tap_mirror`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_tap_mirror sends GET /taas/tap_mirrors/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_tap_mirror lists /taas/tap_mirrors?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- resource.WithIdentityFindQuery values are sent on both the member GET and the /taas/tap_mirrors list fallback, before the automatic name hint.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.

### `python:identity/v3/find_region`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/regions.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`

- find_region sends GET /regions/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_region lists /regions without a name query and matches the row ID only, because Keystone returns no name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- An IdentityName that returns an empty string makes FindIdentity match IDs only in the list fallback.
  Tests: `resource/find_identity_name_test.go:TestCollectionFindIdentityNameOverrideMatchesAlternateFieldOrIDOnly`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- The native model has no name field, so a nonstandard name key in a Keystone response is not compared; Python Resource.name would match it.
- Pinned find_region declares no **query; Go still accepts resource.WithIdentityFindQuery, which Python cannot send.

### `python:network/v2/find_segment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_segment sends GET /segments/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_segment lists /segments?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- resource.WithIdentityFindQuery values are sent on both the member GET and the /segments list fallback, before the automatic name hint.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.

### `python:network/v2/find_vpn_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_vpn_service sends GET /vpn/vpnservices/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_vpn_service lists /vpn/vpnservices?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- resource.WithIdentityFindQuery values are sent on both the member GET and the /vpn/vpnservices list fallback, before the automatic name hint.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.

### `python:network/v2/find_vpn_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_vpn_endpoint_group sends GET /vpn/endpoint-groups/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_vpn_endpoint_group lists /vpn/endpoint-groups?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- resource.WithIdentityFindQuery values are sent on both the member GET and the /vpn/endpoint-groups list fallback, before the automatic name hint.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.

### `python:network/v2/find_vpn_ike_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_vpn_ike_policy sends GET /vpn/ikepolicies/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_vpn_ike_policy lists /vpn/ikepolicies?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- resource.WithIdentityFindQuery values are sent on both the member GET and the /vpn/ikepolicies list fallback, before the automatic name hint.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.

### `python:network/v2/find_vpn_ipsec_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_vpn_ipsec_policy sends GET /vpn/ipsecpolicies/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_vpn_ipsec_policy lists /vpn/ipsecpolicies?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- resource.WithIdentityFindQuery values are sent on both the member GET and the /vpn/ipsecpolicies list fallback, before the automatic name hint.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.

### `python:network/v2/find_vpn_ipsec_site_connection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_vpn_ipsec_site_connection sends GET /vpn/ipsec-site-connections/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_vpn_ipsec_site_connection lists /vpn/ipsec-site-connections?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- resource.WithIdentityFindQuery values are sent on both the member GET and the /vpn/ipsec-site-connections list fallback, before the automatic name hint.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.

### `python:network/v2/find_bgp_peer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithRecordFindIgnoreMissing`

- Retained native API: find_bgp_peer sends GET /bgp-peers/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- Retained native API: After a member GET 400, 403 or 404 find_bgp_peer lists /bgp-peers without a name query and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- Retained native API: A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- Retained native API: The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- Retained native API: A caller query reaches the member GET, but the list fallback stops with ErrUnsupported before HTTP because the native BGP pager takes no query.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.
- GET-first lookup preserves supplied constructor attributes and tenant alias; clean rejection falls back without a server name hint and applies declared Body filters locally.
  Tests: `network/v2/extensions/bgp/peers/python_records_test.go:TestBGPPeerFindGETSeedAndFilteredFallbackKeepProfilesSeparate`.
- Missing policy, repeated-match ambiguity and literal ID equality are distinct; raw fetch controls retain unknown constructor seed while physical errors never become fallback.
  Tests: `network/v2/extensions/bgp/peers/python_records_test.go:TestBGPPeerFindCompletesDuplicateMissingAndLiteralIDSearch`, `network/v2/extensions/bgp/peers/python_records_test.go:TestBGPPeerFindFetchControlsKeepUnknownConstructorSeed`, `network/v2/extensions/bgp/peers/python_records_test.go:TestBGPPeerFindPhysicalFaultsAndPreflightAreTerminal`.

Python/Go differences:

- Go uses owned resource views, explicit string identities and finite JSON options rather than mutable Python Resource instances or arbitrary object inputs. Typed headers and service-relative safe routes are library-owned adaptations. Shared session/Resource and live-cloud/release contracts remain open.

### `python:network/v2/find_bgp_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithRecordFindIgnoreMissing`

- Retained native API: find_bgp_speaker sends GET /bgp-speakers/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- Retained native API: After a member GET 400, 403 or 404 find_bgp_speaker lists /bgp-speakers without a name query and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- Retained native API: A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- Retained native API: The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- Retained native API: A caller query reaches the member GET, but the list fallback stops with ErrUnsupported before HTTP because the native BGP pager takes no query.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.
- GET-first lookup preserves supplied constructor attributes and tenant alias; clean rejection falls back without a server name hint and applies declared Body filters locally.
  Tests: `network/v2/extensions/bgp/speakers/python_records_test.go:TestBGPSpeakerFindGETSeedAndFilteredFallbackKeepProfilesSeparate`.
- Missing policy, repeated-match ambiguity and literal ID equality are distinct; raw fetch controls retain unknown constructor seed while physical errors never become fallback.
  Tests: `network/v2/extensions/bgp/speakers/python_records_test.go:TestBGPSpeakerFindCompletesDuplicateMissingAndLiteralIDSearch`, `network/v2/extensions/bgp/speakers/python_records_test.go:TestBGPSpeakerFindFetchControlsKeepUnknownConstructorSeed`, `network/v2/extensions/bgp/speakers/python_records_test.go:TestBGPSpeakerFindPhysicalFaultsAndPreflightAreTerminal`.

Python/Go differences:

- Go uses owned resource views, explicit string identities and finite JSON options rather than mutable Python Resource instances or arbitrary object inputs. Typed headers and service-relative safe routes are library-owned adaptations. Shared session/Resource and live-cloud/release contracts remain open.

### `python:network/v2/find_bgpvpn`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_bgpvpn sends GET /bgpvpn/bgpvpns/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_bgpvpn lists /bgpvpn/bgpvpns?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- resource.WithIdentityFindQuery values are sent on both the member GET and the /bgpvpn/bgpvpns list fallback, before the automatic name hint.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.
- The native BGPVPN ListOpts has no Name field; the SDK adds the name= hint itself, so explicit Find(resource.Name) also sends name= now.

### `python:cloud/_compute/ComputeCloudMixin/get_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.FindIdentity`

- get_aggregate(name_or_id) sends GET os-aggregates/{name_or_id} with the literal identity, falls back to GET os-aggregates after a 404 and matches the name, and returns nil without error when nothing matches.
  Tests: `compute/v2/aggregates/python_cloud_find_test.go:TestPythonCloudGetAggregateFindsByLiteralIdentityThenList`, `api/find_identity_python_finds_test.go:TestPythonFindAggregateSendsLiteralIdentityToMemberRoute`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python compares the integer Aggregate id with the string name_or_id in the list fallback, so a numeric string never matches there; Go compares the ID as text after a GET 400 or 403.
- Python negotiates min(Aggregate _max_microversion 2.81, server maximum) per request; Go sends the microversion configured or negotiated on the Connection client.
- Python's deprecated filters argument only emits RemovedInSDK50Warning and does not change the lookup; Go has no such argument.

### `python:cloud/_compute/ComputeCloudMixin/update_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.UpdateAggregate`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordAvailabilityZone`

- get_aggregate(name_or_id) is GET os-aggregates/{value} then the list; UpdateAggregate performs it strictly and sends one PUT with only supplied attributes, including availability_zone null.
  Tests: `connectiontest/connection_compute_aggregates_test.go:TestPyCloudComputeUpdateAggregateFindsThenSendsOnlySuppliedNull`.
- A missing aggregate returns ErrNotFound and no PUT is sent.
  Tests: `connectiontest/connection_compute_aggregates_test.go:TestPyCloudComputeUpdateAggregateMissingSendsNoPut`.
- The delegated proxy update (null availability zone, empty name, extension attributes, no-op without attributes) is mapped.
  Tests: `compute/v2/aggregates/python_update_record_test.go:TestPythonAggregateUpdateRecordSendsNullAndEmptyValues`.

Python/Go differences:

- Python raises SDKException for a missing aggregate; Go returns resource.ErrNotFound.
- Python takes **kwargs; Go takes typed UpdateRecord options.
- Identities unsafe as a URL segment skip the member GET in Go; Python sends them encoded first. Python compares the integer id with the string in the list fallback so a numeric string never matches there; Go compares text after a 400 or 403.
- Python returns a Resource; Go returns an AggregateRecord with typed decode, raw fields, headers and status.

### `python:cloud/_compute/ComputeCloudMixin/delete_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteAggregate`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.Delete`

- A digit-only identity is deleted with one DELETE and no lookup, and a 404 on it is false; a 500 is an error.
  Tests: `connectiontest/connection_compute_aggregates_test.go:TestPyCloudComputeDeleteAggregateDigitsSkipLookupAndMapMissingToFalse`, `connectiontest/connection_compute_aggregates_test.go:TestPyCloudComputeDeleteAggregateByNameAndFailures`.
- A name is found first (GET then list), a missing aggregate returns false with no DELETE, and cancellation or a blank identity fails before any request.
  Tests: `connectiontest/connection_compute_aggregates_test.go:TestPyCloudComputeDeleteAggregateByNameAndFailures`.

Python/Go differences:

- Python str.isdigit also accepts non-ASCII digits (for example superscripts) that Go sends through the name lookup; Go parses digit-only ASCII/Unicode-decimal text as an integer ID.
- Python accepts an int input; Go takes a string.
- The generated Delete binding accepts only HTTP 200 from Nova while Python accepts any status below 400; other 2xx codes return an error in Go.
- Python returns True/False; Go returns the same boolean plus an error and never replays the DELETE.

### `python:cloud/_compute/ComputeCloudMixin/set_aggregate_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.SetMetadata`

- get_aggregate(name_or_id) sends GET os-aggregates/{name_or_id} with the literal identity, falls back to GET os-aggregates after a 404 and matches the name, and returns nil without error when nothing matches.
  Tests: `compute/v2/aggregates/python_cloud_find_test.go:TestPythonCloudGetAggregateFindsByLiteralIdentityThenList`, `api/find_identity_python_finds_test.go:TestPythonFindAggregateSendsLiteralIdentityToMemberRoute`.
- set_aggregate_metadata(name_or_id, ...) resolves the aggregate like get_aggregate and then sends POST os-aggregates/{id}/action with {"set_metadata": {"metadata": ...}} through SetMetadata; a missing aggregate stops with resource.ErrNotFound before the action.
  Tests: `compute/v2/aggregates/python_cloud_find_test.go:TestPythonCloudAggregateActionsResolveTheAggregateFirst`, `compute/v2/aggregates/python_parity_test.go:TestPythonAggregateHostAndMetadataActionsReturnAggregate`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python compares the integer Aggregate id with the string name_or_id in the list fallback, so a numeric string never matches there; Go compares the ID as text after a GET 400 or 403.
- Python negotiates min(Aggregate _max_microversion 2.81, server maximum) per request; Go sends the microversion configured or negotiated on the Connection client.
- Python raises SDKException('Host aggregate ... not found.') for a missing aggregate; Go strict FindIdentity returns resource.ErrNotFound and the caller does not call SetMetadata.
- Python passes the found Aggregate object; Go passes the integer ID of the found model.

### `python:cloud/_compute/ComputeCloudMixin/add_host_to_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.AddHost`

- get_aggregate(name_or_id) sends GET os-aggregates/{name_or_id} with the literal identity, falls back to GET os-aggregates after a 404 and matches the name, and returns nil without error when nothing matches.
  Tests: `compute/v2/aggregates/python_cloud_find_test.go:TestPythonCloudGetAggregateFindsByLiteralIdentityThenList`, `api/find_identity_python_finds_test.go:TestPythonFindAggregateSendsLiteralIdentityToMemberRoute`.
- add_host_to_aggregate(name_or_id, ...) resolves the aggregate like get_aggregate and then sends POST os-aggregates/{id}/action with {"add_host": {"host": h}} through AddHost; a missing aggregate stops with resource.ErrNotFound before the action.
  Tests: `compute/v2/aggregates/python_cloud_find_test.go:TestPythonCloudAggregateActionsResolveTheAggregateFirst`, `compute/v2/aggregates/python_parity_test.go:TestPythonAggregateHostAndMetadataActionsReturnAggregate`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python compares the integer Aggregate id with the string name_or_id in the list fallback, so a numeric string never matches there; Go compares the ID as text after a GET 400 or 403.
- Python negotiates min(Aggregate _max_microversion 2.81, server maximum) per request; Go sends the microversion configured or negotiated on the Connection client.
- Python raises SDKException('Host aggregate ... not found.') for a missing aggregate; Go strict FindIdentity returns resource.ErrNotFound and the caller does not call AddHost.
- Python passes the found Aggregate object; Go passes the integer ID of the found model.

### `python:cloud/_compute/ComputeCloudMixin/remove_host_from_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.RemoveHost`

- get_aggregate(name_or_id) sends GET os-aggregates/{name_or_id} with the literal identity, falls back to GET os-aggregates after a 404 and matches the name, and returns nil without error when nothing matches.
  Tests: `compute/v2/aggregates/python_cloud_find_test.go:TestPythonCloudGetAggregateFindsByLiteralIdentityThenList`, `api/find_identity_python_finds_test.go:TestPythonFindAggregateSendsLiteralIdentityToMemberRoute`.
- remove_host_from_aggregate(name_or_id, ...) resolves the aggregate like get_aggregate and then sends POST os-aggregates/{id}/action with {"remove_host": {"host": h}} through RemoveHost; a missing aggregate stops with resource.ErrNotFound before the action.
  Tests: `compute/v2/aggregates/python_cloud_find_test.go:TestPythonCloudAggregateActionsResolveTheAggregateFirst`, `compute/v2/aggregates/python_parity_test.go:TestPythonAggregateHostAndMetadataActionsReturnAggregate`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python compares the integer Aggregate id with the string name_or_id in the list fallback, so a numeric string never matches there; Go compares the ID as text after a GET 400 or 403.
- Python negotiates min(Aggregate _max_microversion 2.81, server maximum) per request; Go sends the microversion configured or negotiated on the Connection client.
- Python raises SDKException('Host aggregate ... not found.') for a missing aggregate; Go strict FindIdentity returns resource.ErrNotFound and the caller does not call RemoveHost.
- Python passes the found Aggregate object; Go passes the integer ID of the found model.

### `python:cloud/_network/NetworkCloudMixin/get_firewall_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_firewall_rule sends GET /fwaas/firewall_rules/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_firewall_rule lists /fwaas/firewall_rules?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- get_firewall_rule(name_or_id, filters) sends every filter on GET /fwaas/firewall_rules/{name_or_id} and on the name list, and returns nil without error when nothing matches, as find_firewall_rule(ignore_missing=True, **filters).
  Tests: `api/find_identity_python_finds_test.go:TestPythonCloudGetFirewallForwardsFiltersToBothRequests`, `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.
- Python takes filters as one dict; Go takes one resource.WithIdentityFindQuery(key, value) per filter with string values, and nested dict filter values have no Go form.

### `python:cloud/_network/NetworkCloudMixin/get_firewall_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_firewall_policy sends GET /fwaas/firewall_policies/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_firewall_policy lists /fwaas/firewall_policies?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- get_firewall_policy(name_or_id, filters) sends every filter on GET /fwaas/firewall_policies/{name_or_id} and on the name list, and returns nil without error when nothing matches, as find_firewall_policy(ignore_missing=True, **filters).
  Tests: `api/find_identity_python_finds_test.go:TestPythonCloudGetFirewallForwardsFiltersToBothRequests`, `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.
- Python takes filters as one dict; Go takes one resource.WithIdentityFindQuery(key, value) per filter with string values, and nested dict filter values have no Go form.

### `python:cloud/_network/NetworkCloudMixin/get_firewall_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- find_firewall_group sends GET /fwaas/firewall_groups/{name_or_id} with the literal identity and returns that resource without listing when the GET succeeds.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindGETByIDReturnsWithoutList`.
- After a member GET 400, 403 or 404 find_firewall_group lists /fwaas/firewall_groups?name={name_or_id} and matches the row ID or name; any other GET error is returned without listing.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- A missing resource is nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false); two matching rows are resource.ErrAmbiguous.
  Tests: `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`, `api/find_identity_python_finds_test.go:TestPythonFindDuplicateMatchesAreAmbiguous`.
- The sdkgen opt-in pins the native member GET route, accepted codes, name policy and list query policy, and rejects model or signature drift.
  Tests: `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedCollectionsRejectNativeShapeDrift`, `internal/cmd/sdkgen/identity_extended_collections_test.go:TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet`.
- get_firewall_group(name_or_id, filters) sends every filter on GET /fwaas/firewall_groups/{name_or_id} and on the name list, and returns nil without error when nothing matches, as find_firewall_group(ignore_missing=True, **filters).
  Tests: `api/find_identity_python_finds_test.go:TestPythonCloudGetFirewallForwardsFiltersToBothRequests`, `api/find_identity_python_finds_test.go:TestPythonFindForwardsQueryToGETAndList`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- Python list fallback drops **query keys missing from the Resource query mapping and filters known Body attributes locally; Go sends every resource.WithIdentityFindQuery key on the wire for both the GET and the list and applies no local Body filter.
- Go query values are strings; Python stringifies values such as booleans (True) when building the request.
- Python takes filters as one dict; Go takes one resource.WithIdentityFindQuery(key, value) per filter with string values, and nested dict filter values have no Go form.

### `python:cloud/_network/NetworkCloudMixin/delete_firewall_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteFirewallRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteCurrentLocation`

- The strict lookup of delete_firewall_rule sends the filters on GET /fwaas/firewall_rules/{name_or_id} and on the name list and reports a missing resource as resource.ErrNotFound before any DELETE.
  Tests: `api/find_identity_python_finds_test.go:TestPythonCloudGetFirewallForwardsFiltersToBothRequests`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- Library-owned concrete JSON/functional options snapshot raw filters, headers, version and location. Falsey filters become empty kwargs, truthy filters require mapping despite the historical JMES docstring; bound name_or_id/ignore_missing collide by presence before HTTP. No caller builder/interface or extension/has_service gate is needed.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- A closed Source profile reuses Engine.Find with strict ignore_missing=False: initial member GET, clean400/403/404 complete list fallback, declared query versus local Body filters and exact original ID/name match; unknown kwargs reach GET only and explicit name overrides the server hint. Full fallback duplicate/late error blocks DELETE. It uses plain Resource fields without inherited NetworkResource Header/tag defaults.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Found actual Resource ID selects one strict bodyless DELETE. Clean missing Find, clean LIST404 and clean actual DELETE404 return Deleted=false; normal accepted100..399 DELETE returns true regardless of empty/non-JSON body. Mutation Resource/Wire are not fabricated; lookup and deletion actual envelope/header/status remain separate. Conflicts and non404 rejections remain errors.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`.
- Source fetch accepted204/empty or200/malformed response keeps the original requested ID seed and can proceed to DELETE; explicit returned null ID errors before DELETE. Profile SourceStatusPolicy allows100..399, cloned SourceLinkPolicy and PythonLoadedEquality leave old native/generated paths unchanged. Untyped fields remain raw, Group.ports/Policy.firewall_rules wrap scalar list values, Rule Computed.summary reads pinned default while raw summary remains only in Wire.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteAcceptedEmptyLookupAndFalseyReturnedIDBoundaries`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Physical accepted200/rejected401/404/503 GET and DELETE faults preserve actual full bytes/header/status and no successful Resource. Three-family×two-method×four-status×Read+Close/cancel/source72lane fixture closes once, invokes no native retry/reauth hook and keeps custom context cause. Native401 pointer does not unwrap: actual value ErrOriginal and ErrReauth read/close causes are checked independently while retaining any outer joined causes.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.
- Clean missing classification delegates through the existing resource Collection Get/Delete policies and accepts only a single clean missing chain. Joined terminal callback/physical/source/context errors cannot become false merely because they contain404. A callback-fabricated404 over actual503 remains error; selected original client/API/route facts are guarded around cloned option callbacks and physical operations.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteOptionsSourceGuardsAndFabricated404NeverFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.
- Filters reach Resource.find through the shared Source fetch-control profile: filters.headers is an ordinary initial member GET query value and becomes HTTP headers only on the fallback LIST; filters.microversion versions only the initial GET while the fallback LIST and DELETE use the adapter default; base_path/resource_response_key/skip_cache/requires_id/error_message control only the initial fetch and list_base_path only the fallback LIST.
  Tests: `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallDeleteFindHeadersAndMicroversionPhases`.
- Raw max_items is a Source cap compared before each fallback row and also sent as an initial GET query value, with Requests limit hint spelling. A cap equal to a whole page performs Python's next marker GET (its failure stops before DELETE), true counts as 1 and a negative truthy cap stops at the first row so the strict Find reports a clean missing resource.
  Tests: `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallDeleteFindCapEqualsWholePage`, `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallDeleteFindRawCapValues`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- SDK-R1: Go returns bool plus owned Lookup/Delete receipt views instead of mutable Python Resource/cache/lifecycle objects. All declared getter fields are projected before return, so an unused malformed field may fail earlier. Source informational status and empty/malformed fetch seed behavior are implemented only on this new closed profile; legacy typed/generated APIs keep their old contracts.
- SDK-C1: JSON bytes retain large integers and Source untyped values without native bool/int coercion. Local equality uses shared CPython-loaded numeric policy and exact raw Wire/Envelope proof. Non-JSON custom objects and unrestricted Python runtime/representation/marshal behaviors remain separately tracked; structured Go errors keep actual native status and physical/context causes.
- SDK-S1: Go captures the original Neutron service/client/route and uses safe escaped literal member IDs with fixed protocol/header/origin guards. Arbitrary adapter/class retargeting and URI percent-format are restricted. WithFirewallDeleteHeaders/Microversion are explicit Go transport extras (headers across phases, microversion on initial GET+DELETE while fallback LIST keeps captured default). The raw Source Find controls in filters are implemented separately; filters.microversion cannot be combined with WithFirewallDeleteMicroversion because they would select two initial GET versions.
- Source has no FWaaS extension/has_service pre-gate for these deletes, and neither does this workflow. Shared-reader access is not permission to delete: official FWaaS2025.2 sample gives owner project member/admin ordinary delete and separate admin-only shared deletion. SDK does not elevate RBAC or unlink attached rules/policies/groups; backend conflict remains actual409.
- Actual pinned Source156cases, standalone Python AST/byte-exact Go+Python mains and prepared fixture groups are distinct evidence. Owner did not execute Go/Git or authenticate a live cloud.
- Raw limit filter values in the Find fallback must be nonnegative integers for the marker-fallback decision; Python's truthiness of boolean, fractional or string limits, which Neutron rejects with HTTP 400, is not reproduced; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- A found ID that is empty or otherwise falsey but not null ("", 0, false, [] or {}) is rejected before DELETE. Python only rejects None, so utils.urljoin turns such an ID into an empty segment and sends DELETE to the collection route, which Neutron refuses with HTTP 405 and Python raises as an HttpException; Go never sends a collection-route DELETE.

### `python:cloud/_network/NetworkCloudMixin/delete_firewall_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteFirewallPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteCurrentLocation`

- The strict lookup of delete_firewall_policy sends the filters on GET /fwaas/firewall_policies/{name_or_id} and on the name list and reports a missing resource as resource.ErrNotFound before any DELETE.
  Tests: `api/find_identity_python_finds_test.go:TestPythonCloudGetFirewallForwardsFiltersToBothRequests`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- Library-owned concrete JSON/functional options snapshot raw filters, headers, version and location. Falsey filters become empty kwargs, truthy filters require mapping despite the historical JMES docstring; bound name_or_id/ignore_missing collide by presence before HTTP. No caller builder/interface or extension/has_service gate is needed.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- A closed Source profile reuses Engine.Find with strict ignore_missing=False: initial member GET, clean400/403/404 complete list fallback, declared query versus local Body filters and exact original ID/name match; unknown kwargs reach GET only and explicit name overrides the server hint. Full fallback duplicate/late error blocks DELETE. It uses plain Resource fields without inherited NetworkResource Header/tag defaults.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Found actual Resource ID selects one strict bodyless DELETE. Clean missing Find, clean LIST404 and clean actual DELETE404 return Deleted=false; normal accepted100..399 DELETE returns true regardless of empty/non-JSON body. Mutation Resource/Wire are not fabricated; lookup and deletion actual envelope/header/status remain separate. Conflicts and non404 rejections remain errors.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`.
- Source fetch accepted204/empty or200/malformed response keeps the original requested ID seed and can proceed to DELETE; explicit returned null ID errors before DELETE. Profile SourceStatusPolicy allows100..399, cloned SourceLinkPolicy and PythonLoadedEquality leave old native/generated paths unchanged. Untyped fields remain raw, Group.ports/Policy.firewall_rules wrap scalar list values, Rule Computed.summary reads pinned default while raw summary remains only in Wire.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteAcceptedEmptyLookupAndFalseyReturnedIDBoundaries`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Physical accepted200/rejected401/404/503 GET and DELETE faults preserve actual full bytes/header/status and no successful Resource. Three-family×two-method×four-status×Read+Close/cancel/source72lane fixture closes once, invokes no native retry/reauth hook and keeps custom context cause. Native401 pointer does not unwrap: actual value ErrOriginal and ErrReauth read/close causes are checked independently while retaining any outer joined causes.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.
- Clean missing classification delegates through the existing resource Collection Get/Delete policies and accepts only a single clean missing chain. Joined terminal callback/physical/source/context errors cannot become false merely because they contain404. A callback-fabricated404 over actual503 remains error; selected original client/API/route facts are guarded around cloned option callbacks and physical operations.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteOptionsSourceGuardsAndFabricated404NeverFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.
- Filters reach Resource.find through the shared Source fetch-control profile: filters.headers is an ordinary initial member GET query value and becomes HTTP headers only on the fallback LIST; filters.microversion versions only the initial GET while the fallback LIST and DELETE use the adapter default; base_path/resource_response_key/skip_cache/requires_id/error_message control only the initial fetch and list_base_path only the fallback LIST.
  Tests: `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallDeleteFindHeadersAndMicroversionPhases`.
- Raw max_items is a Source cap compared before each fallback row and also sent as an initial GET query value, with Requests limit hint spelling. A cap equal to a whole page performs Python's next marker GET (its failure stops before DELETE), true counts as 1 and a negative truthy cap stops at the first row so the strict Find reports a clean missing resource.
  Tests: `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallDeleteFindCapEqualsWholePage`, `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallDeleteFindRawCapValues`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- SDK-R1: Go returns bool plus owned Lookup/Delete receipt views instead of mutable Python Resource/cache/lifecycle objects. All declared getter fields are projected before return, so an unused malformed field may fail earlier. Source informational status and empty/malformed fetch seed behavior are implemented only on this new closed profile; legacy typed/generated APIs keep their old contracts.
- SDK-C1: JSON bytes retain large integers and Source untyped values without native bool/int coercion. Local equality uses shared CPython-loaded numeric policy and exact raw Wire/Envelope proof. Non-JSON custom objects and unrestricted Python runtime/representation/marshal behaviors remain separately tracked; structured Go errors keep actual native status and physical/context causes.
- SDK-S1: Go captures the original Neutron service/client/route and uses safe escaped literal member IDs with fixed protocol/header/origin guards. Arbitrary adapter/class retargeting and URI percent-format are restricted. WithFirewallDeleteHeaders/Microversion are explicit Go transport extras (headers across phases, microversion on initial GET+DELETE while fallback LIST keeps captured default). The raw Source Find controls in filters are implemented separately; filters.microversion cannot be combined with WithFirewallDeleteMicroversion because they would select two initial GET versions.
- Source has no FWaaS extension/has_service pre-gate for these deletes, and neither does this workflow. Shared-reader access is not permission to delete: official FWaaS2025.2 sample gives owner project member/admin ordinary delete and separate admin-only shared deletion. SDK does not elevate RBAC or unlink attached rules/policies/groups; backend conflict remains actual409.
- Actual pinned Source156cases, standalone Python AST/byte-exact Go+Python mains and prepared fixture groups are distinct evidence. Owner did not execute Go/Git or authenticate a live cloud.
- Raw limit filter values in the Find fallback must be nonnegative integers for the marker-fallback decision; Python's truthiness of boolean, fractional or string limits, which Neutron rejects with HTTP 400, is not reproduced; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- A found ID that is empty or otherwise falsey but not null ("", 0, false, [] or {}) is rejected before DELETE. Python only rejects None, so utils.urljoin turns such an ID into an empty segment and sends DELETE to the collection route, which Neutron refuses with HTTP 405 and Python raises as an HttpException; Go never sends a collection-route DELETE.

### `python:cloud/_network/NetworkCloudMixin/delete_firewall_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteFirewallGroupRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteCurrentLocation`

- The strict lookup of delete_firewall_group sends the filters on GET /fwaas/firewall_groups/{name_or_id} and on the name list and reports a missing resource as resource.ErrNotFound before any DELETE.
  Tests: `api/find_identity_python_finds_test.go:TestPythonCloudGetFirewallForwardsFiltersToBothRequests`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- Library-owned concrete JSON/functional options snapshot raw filters, headers, version and location. Falsey filters become empty kwargs, truthy filters require mapping despite the historical JMES docstring; bound name_or_id/ignore_missing collide by presence before HTTP. No caller builder/interface or extension/has_service gate is needed.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- A closed Source profile reuses Engine.Find with strict ignore_missing=False: initial member GET, clean400/403/404 complete list fallback, declared query versus local Body filters and exact original ID/name match; unknown kwargs reach GET only and explicit name overrides the server hint. Full fallback duplicate/late error blocks DELETE. It uses plain Resource fields without inherited NetworkResource Header/tag defaults.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Found actual Resource ID selects one strict bodyless DELETE. Clean missing Find, clean LIST404 and clean actual DELETE404 return Deleted=false; normal accepted100..399 DELETE returns true regardless of empty/non-JSON body. Mutation Resource/Wire are not fabricated; lookup and deletion actual envelope/header/status remain separate. Conflicts and non404 rejections remain errors.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`.
- Source fetch accepted204/empty or200/malformed response keeps the original requested ID seed and can proceed to DELETE; explicit returned null ID errors before DELETE. Profile SourceStatusPolicy allows100..399, cloned SourceLinkPolicy and PythonLoadedEquality leave old native/generated paths unchanged. Untyped fields remain raw, Group.ports/Policy.firewall_rules wrap scalar list values, Rule Computed.summary reads pinned default while raw summary remains only in Wire.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteAcceptedEmptyLookupAndFalseyReturnedIDBoundaries`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Physical accepted200/rejected401/404/503 GET and DELETE faults preserve actual full bytes/header/status and no successful Resource. Three-family×two-method×four-status×Read+Close/cancel/source72lane fixture closes once, invokes no native retry/reauth hook and keeps custom context cause. Native401 pointer does not unwrap: actual value ErrOriginal and ErrReauth read/close causes are checked independently while retaining any outer joined causes.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.
- Clean missing classification delegates through the existing resource Collection Get/Delete policies and accepts only a single clean missing chain. Joined terminal callback/physical/source/context errors cannot become false merely because they contain404. A callback-fabricated404 over actual503 remains error; selected original client/API/route facts are guarded around cloned option callbacks and physical operations.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteOptionsSourceGuardsAndFabricated404NeverFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.
- Filters reach Resource.find through the shared Source fetch-control profile: filters.headers is an ordinary initial member GET query value and becomes HTTP headers only on the fallback LIST; filters.microversion versions only the initial GET while the fallback LIST and DELETE use the adapter default; base_path/resource_response_key/skip_cache/requires_id/error_message control only the initial fetch and list_base_path only the fallback LIST.
  Tests: `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallDeleteFindHeadersAndMicroversionPhases`.
- Raw max_items is a Source cap compared before each fallback row and also sent as an initial GET query value, with Requests limit hint spelling. A cap equal to a whole page performs Python's next marker GET (its failure stops before DELETE), true counts as 1 and a negative truthy cap stops at the first row so the strict Find reports a clean missing resource.
  Tests: `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallDeleteFindCapEqualsWholePage`, `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallDeleteFindRawCapValues`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- SDK-R1: Go returns bool plus owned Lookup/Delete receipt views instead of mutable Python Resource/cache/lifecycle objects. All declared getter fields are projected before return, so an unused malformed field may fail earlier. Source informational status and empty/malformed fetch seed behavior are implemented only on this new closed profile; legacy typed/generated APIs keep their old contracts.
- SDK-C1: JSON bytes retain large integers and Source untyped values without native bool/int coercion. Local equality uses shared CPython-loaded numeric policy and exact raw Wire/Envelope proof. Non-JSON custom objects and unrestricted Python runtime/representation/marshal behaviors remain separately tracked; structured Go errors keep actual native status and physical/context causes.
- SDK-S1: Go captures the original Neutron service/client/route and uses safe escaped literal member IDs with fixed protocol/header/origin guards. Arbitrary adapter/class retargeting and URI percent-format are restricted. WithFirewallDeleteHeaders/Microversion are explicit Go transport extras (headers across phases, microversion on initial GET+DELETE while fallback LIST keeps captured default). The raw Source Find controls in filters are implemented separately; filters.microversion cannot be combined with WithFirewallDeleteMicroversion because they would select two initial GET versions.
- Source has no FWaaS extension/has_service pre-gate for these deletes, and neither does this workflow. Shared-reader access is not permission to delete: official FWaaS2025.2 sample gives owner project member/admin ordinary delete and separate admin-only shared deletion. SDK does not elevate RBAC or unlink attached rules/policies/groups; backend conflict remains actual409.
- Actual pinned Source156cases, standalone Python AST/byte-exact Go+Python mains and prepared fixture groups are distinct evidence. Owner did not execute Go/Git or authenticate a live cloud.
- Raw limit filter values in the Find fallback must be nonnegative integers for the marker-fallback decision; Python's truthiness of boolean, fractional or string limits, which Neutron rejects with HTTP 400, is not reproduced; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- A found ID that is empty or otherwise falsey but not null ("", 0, false, [] or {}) is rejected before DELETE. Python only rejects None, so utils.urljoin turns such an ID into an empty segment and sends DELETE to the collection route, which Neutron refuses with HTTP 405 and Python raises as an HttpException; Go never sends a collection-route DELETE.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    computev2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
    storagev3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindApplicationResources(ctx context.Context, compute *computev2.Service,
    storage *storagev3.Service, network *networkv2.Service) error {
    strict := resource.WithIdentityFindIgnoreMissing(false)
    server, err := compute.Servers.FindIdentity(ctx, "web-01", strict)
    if err != nil { return err }
    volume, err := storage.Volumes.FindIdentity(ctx, "data-01", strict)
    if err != nil { return err }
    port, err := network.Ports.FindIdentity(ctx, "web-port", strict)
    if err != nil { return err }
    _, _, _ = server, volume, port
    return nil
}
```

```go
package example

import (
    "context"

    dnsv2 "github.com/JSYoo5B/go-openstacksdk/dns/v2"
    lbv2 "github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindScopedResources(ctx context.Context, dns *dnsv2.Service,
    loadbalancer *lbv2.Service) error {
    records, err := dns.RecordSets.InZone(ctx, resource.Name("example.org."))
    if err != nil { return err }
    record, err := records.FindIdentity(ctx, "www.example.org.",
        resource.WithIdentityFindIgnoreMissing(false))
    if err != nil { return err }
    members, err := loadbalancer.Pools.Members(ctx, resource.ID("pool-id"))
    if err != nil { return err }
    member, err := members.FindIdentity(ctx, "backend-01",
        resource.WithIdentityFindIgnoreMissing(false))
    if err != nil { return err }
    _, _ = record, member
    return nil
}
```

```go
package example

import (
    "context"

    computev2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
    blockstoragev3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindAcrossProjects(ctx context.Context, compute *computev2.Service,
    storage *blockstoragev3.Service) error {
    summary := resource.WithIdentityFindDetails(false)
    allProjects := resource.WithIdentityFindAllProjects(true)
    strict := resource.WithIdentityFindIgnoreMissing(false)
    server, err := compute.Servers.FindIdentity(ctx, "web-01", summary, allProjects, strict)
    if err != nil { return err }
    volume, err := storage.Volumes.FindIdentity(ctx, "data-01", summary, allProjects, strict)
    if err != nil { return err }
    _, _ = server, volume
    return nil
}
```

```go
flavor, err := compute.Flavors.FindIdentity(ctx, flavorID,
    resource.WithIdentityFindFallback(resource.FindFallbackNever),
    resource.WithIdentityFindIgnoreMissing(false),
    resource.WithIdentityFindExtraSpecs(true))
```

```go
package example

import (
    "context"

    computev2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindFlavor(ctx context.Context, compute *computev2.Service) error {
    flavor, err := compute.Flavors.FindIdentity(ctx, "small",
        resource.WithIdentityFindExtraSpecs(true),
        resource.WithIdentityFindIgnoreMissing(false),
        resource.WithIdentityFindQuery("minRam", "256"))
    if err != nil { return err }
    _, _ = flavor.ID, flavor.ExtraSpecs
    return nil
}
```

```go
package example

import (
    "context"

    imagev2 "github.com/JSYoo5B/go-openstacksdk/image/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindImage(ctx context.Context, images *imagev2.Service) error {
    image, err := images.Images.FindIdentity(ctx, "ubuntu",
        resource.WithIdentityFindIgnoreMissing(false))
    if err != nil { return err }
    _, _ = image.ID, image.Hidden
    return nil
}
```

```go
package example

import (
    "context"

    identityv3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindTenantResources(ctx context.Context, identity *identityv3.Service,
    network *networkv2.Service) error {
    strict := resource.WithIdentityFindIgnoreMissing(false)
    domain, err := identity.Domains.FindIdentity(ctx, "Default", strict)
    if err != nil { return err }
    domainQuery := resource.WithIdentityFindQuery("domain_id", domain.ID)
    project, err := identity.Projects.FindIdentity(ctx, "app", strict, domainQuery)
    if err != nil { return err }
    user, err := identity.Users.FindIdentity(ctx, "app-user", strict, domainQuery)
    if err != nil { return err }
    group, err := identity.Groups.FindIdentity(ctx, "app-group", strict, domainQuery)
    if err != nil { return err }
    role, err := identity.Roles.FindIdentity(ctx, "reader", strict, domainQuery)
    if err != nil { return err }
    net, err := network.Networks.FindIdentity(ctx, "web-net", strict)
    if err != nil { return err }
    subnet, err := network.Subnets.FindIdentity(ctx, "web-subnet", strict,
        resource.WithIdentityFindQuery("network_id", net.ID))
    if err != nil { return err }
    _, _, _, _, _ = project, user, group, role, subnet
    return nil
}
```

```go
package example

import (
    "context"
    "net/url"

    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindNetworkExtensions(ctx context.Context, network *networkv2.Service,
    projectID string) error {
    filter := resource.WithIdentityFindOptions(resource.IdentityFindOpts{
        Query: url.Values{"project_id": {projectID}, "fields": {"id", "name"}},
    })
    strict := resource.WithIdentityFindIgnoreMissing(false)
    router, err := network.Routers.FindIdentity(ctx, "app-router", filter, strict)
    if err != nil { return err }
    group, err := network.SecurityGroups.FindIdentity(ctx, "app-sg", filter, strict)
    if err != nil { return err }
    _, _ = router, group
    return nil
}
```

```go
package example

import (
    "context"

    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindSubnetPoolAndTrunk(ctx context.Context, network *networkv2.Service,
    projectID string) error {
    filter := resource.WithIdentityFindQuery("project_id", projectID)
    strict := resource.WithIdentityFindIgnoreMissing(false)
    pool, err := network.SubnetPools.FindIdentity(ctx, "app-pool", filter, strict)
    if err != nil { return err }
    trunk, err := network.Trunks.FindIdentity(ctx, "app-trunk", filter, strict)
    if err != nil { return err }
    _, _ = pool.Prefixes, trunk.Subports
    return nil
}
```

```go
package example

import (
    "context"

    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindQoSAndAddresses(ctx context.Context, network *networkv2.Service,
    projectID string) error {
    strict := resource.WithIdentityFindIgnoreMissing(false)
    policy, err := network.QoSPolicies.FindIdentity(ctx, "app-qos",
        resource.WithIdentityFindQuery("shared", "true"), strict)
    if err != nil { return err }
    group, err := network.SecurityAddressGroups.FindIdentity(ctx, "app-addresses",
        resource.WithIdentityFindQuery("project_id", projectID), strict)
    if err != nil { return err }
    _, _ = policy.Rules, group.Addresses
    return nil
}
```

## References

- [Finding](../clustering/v1/finding/README.md)
- [User Read APIs](../compute/user-read-apis.md)
- [SDK Support Ledger](sdk-support-ledger.md)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2439)
- [Nova find_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1023)
- [Cinder find_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L855)
- [Neutron find_port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3950)
- [Designate find_recordset](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/_proxy.py#L337)
- [Octavia find_member](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/load_balancer/v2/_proxy.py#L542)
- [find_network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3214)
- [find_subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7487)
- [find_project](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1063)
- [find_user](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1379)
- [find_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L782)
- [find_domain](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L259)
- [find_role](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1743)
- [Listing](../network/v2/extensions/layer3/routers/listing/README.md)
- [Listing](../network/v2/extensions/security/groups/listing/README.md)
- [find_router](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5570)
- [find_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6598)
- [Router query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/router.py#L36)
- [SecurityGroup query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group.py#L29)
- [find_subnet_pool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7621)
- [find_trunk](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7869)
- [SubnetPool query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet_pool.py#L34)
- [Trunk query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/trunk.py#L33)
- [Listing](../network/v2/extensions/subnetpools/listing/README.md)
- [Listing](../network/v2/extensions/trunks/listing/README.md)
- [find_qos_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5130)
- [find_address_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L346)
- [QoSPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py#L21)
- [AddressGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/address_group.py#L22)
- [find_tap_mirror](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9096)
- [find_segment](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7192)
- [find_vpn_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L8592)
- [find_firewall_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6437)
- [find_bgpvpn](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1087)
- [find_bgp_peer](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L848)
- [find_ip](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L2177)
- [find_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L956)
- [find_region](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1636)
- [find_aggregate](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L482)
- [get_aggregate](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1922)
- [get_firewall_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L846)
- [users.API.FindRecord](../identity/v3/users/python-record-find.md)
- [Python Record Find](../identity/v3/users/python-record-find.md)
- [Python Record Reads](../identity/v3/projects/python-record-reads.md)
- [Python Records](../identity/v3/groups/python-records.md)
- [Roles.FindRecord](../identity/v3/roles/python-record-find.md)
