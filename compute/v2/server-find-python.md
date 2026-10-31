# Server Find Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--server", required=True)
parser.add_argument("--basic", action="store_true")
parser.add_argument("--all-projects", action="store_true")
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()

conn = openstack.connect(cloud=args.cloud)
server = conn.compute.find_server(
    args.server,
    ignore_missing=not args.strict,
    details=not args.basic,
    all_projects=args.all_projects,
)
print(json.dumps(None if server is None else server.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("server", "", "exact server ID or name")
    basic := flag.Bool("basic", false, "use basic fallback list")
    allProjects := flag.Bool("all-projects", false, "request cross-project lookup")
    strict := flag.Bool("strict", false, "return an error for missing server")
    version := flag.String("microversion", "", "optional selected service microversion")
    flag.Parse()
    if *identity == "" { log.Fatal("-server is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    connectionOptions := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" {
        connectionOptions = append(connectionOptions,
            sdk.WithMicroversion(sdk.Compute, *version))
    }
    conn, err := sdk.Connect(ctx, connectionOptions...)
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    record, err := service.Servers.FindRecord(ctx, *identity,
        servers.WithRecordFindIgnoreMissing(!*strict),
        servers.WithRecordFindDetails(!*basic),
        servers.WithRecordFindAllProjects(*allProjects))
    if errors.Is(err, resource.ErrNotFound) { log.Fatal("server not found") }
    if errors.Is(err, resource.ErrAmbiguous) { log.Fatal("server name is ambiguous") }
    if err != nil { log.Fatal(err) }
    if record == nil { fmt.Println("null"); return }
    body, err := json.MarshalIndent(record.Resource.Body, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
    fmt.Printf("HTTP %d microversion=%s request-id=%s\n",
        record.StatusCode, record.Microversion,
        record.Header.Get("X-Openstack-Request-Id"))
}
```

## References

- [Install](../../docs/install.md)
