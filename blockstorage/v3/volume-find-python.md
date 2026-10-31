# Volume Find Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="source")
volume = conn.block_storage.find_volume(
    "data", ignore_missing=False, details=False, all_projects=True
)
print(volume.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
)

func main() {
    identity := flag.String("volume", "data", "Volume name or ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud("source"))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    volume, err := service.Volumes.FindRecord(ctx, *identity,
        volumes.WithRecordFindIgnoreMissing(false),
        volumes.WithRecordFindDetails(false),
        volumes.WithRecordFindAllProjects(true),
    )
    if err != nil { log.Fatal(err) }
    body, err := json.MarshalIndent(volume.Resource.Body, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}
```

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Volume Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py)
