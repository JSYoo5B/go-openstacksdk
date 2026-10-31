# Python Record Reads

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

### `python:identity/v3/projects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithRecordListJMESPath`

- List sends the single GET projects that Python projects() sends without arguments, maps the Python query names domain_id, enabled, is_domain, name, parent_id, tags, tags-any, not-tags and not-tags-any through ListOpts or WithListQuery, sends a limit without a marker follow-up, and rejects local Body filters before HTTP.
  Tests: `identity/v3/projects/python_parity_test.go:TestPythonProjectListRequests`.
- Default bodyless /projects GET, selected identity role, Source Body aliases/casts/defaults and explicit-null presence retain independent Resource/Wire/page/header/status evidence including large integer extensions.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsDeclaredViewKeepsPhysicalEvidenceAndPresence`.
- Server canonical/wire QueryParameters use Requests boolean/repeated/null values; declared non-query Body filters use ordered nested CPython loaded equality, empty-dict truthiness and unknown-query discard before marshaling.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsQueryAliasesAndOrderedLoadedLocalFilters`.
- Truthy limit repeats even short pages without links using the last raw row marker, raw cap counts locally excluded rows, empty page ends and consumer break stops before another request.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsLimitMarkerWithoutLinkRawCapAndBreak`.
- Pinned Keystone dictionary-next is ignored; dangerous advertised top-level origin/filter changes and cycles preserve the actual whole page receipt.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsKeystoneDictionaryNextAndRouteGuards`.
- Source opaque Proxy JMESPath runs before HTTP and preserves null/literal values, generator identity and nested unconsumed generators; typed list remains resource-only.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsProxyOpaqueJMESIsUnreadAndRetainsGenerator`.
- DTO/functional options snapshot values and headers; nil context/options, selected source mutation and custom cancellation fail before physical HTTP without fabricated receipt.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsOptionsSnapshotAndSourceCancellationPreflight`.
- Malformed plural/null/colliding row and nonmapping local filter failures retain accepted page status/body/header. Clean native reauthentication keeps identity version and live token.
  Tests: `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsInvalidRepresentationRetainsActualHeaderAndPage`, `identity/v3/projects/python_records_test.go:TestPythonProjectRecordsCleanReauthenticationAndVersionHeaderAreIdentityOwned`.

Python/Go differences:

- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Python sends bool query values as True/False text; typed Go ListOpts send true/false, and WithListQuery reproduces the Python text.
- Python drops unknown query names; Go sends every WithListQuery key to the server.
- projects.ListOpts.Filters adds a stray "-" query key next to the comparator filters; Python cannot send comparator filters at all.
- Concrete functional options own defaults and defensive copies. Runtime Resource class/session builders are SDK-owned; safe UTF-8 routes, nonnegative Go pagination caps, same-origin/path/filter continuation and stronger cycle bounds are deliberate Go policies.
- Source-selected literal link precedence, ignored Keystone dictionary links and limit-on-short-page marker fallback are enabled for this new Project profile without changing existing native List or older engine profiles. Valid RFC HTTP Link remains a documented Go interoperability correction of the pinned Requests uri/url failure.
- SDK-R1/C1/S1 remain separately tracked whole-SDK Connection Resource runtime, optional response cache and session/version negotiation limitations. Direct leaf location is null; typed ProjectReadLocation or Connection bridge supplies configured facts. This named read mapping is not whole-SDK parity or live-cloud validation.

### `python:cloud/_identity/IdentityCloudMixin/list_projects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewProjectReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.ProjectReadCloud.ListProjects`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ListProjects`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadNameOrID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadLocation`

- Concrete Keystone v3 Project facade expands filters in insertion order then truthy explicit name/domain override corresponding filter values; declared Body filters run in the leaf and actual203 record/header is retained.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudListTruthfulNameDomainOverrideAndLocalBodyFilter`.
- Falsey null/bool/number/underflow/string/list/map filters become empty kwargs. Every truthy string/bool/number/list fails before HTTP because pinned cloud.list_projects uses dict(**filters), despite its JMESPath documentation.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudFalseyFiltersAndTruthyNonMappingRejectBeforeHTTP`.
- Proxy literal lists/strings/objects are consumed without HTTP or invented Project association; nested unconsumed generator remains in Value with nil JSON. Null/noniterable/type-error Proxy values fail before HTTP.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudProxyLiteralAndOpaqueListResultsDoNotInventHTTP`.
- Captured source headers predate callbacks and remain fixed through actual late503; callback executes once, prior physically exposed Inventory remains, actual response proof and custom cancellation survive.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudSourceHeaderSnapshotCallbackCancellationAndLatePageEvidence`.
- Connection uses cached IdentityV3 Projects client, injects configured CurrentLocation before selection and after opts/nil reset, and preserves owned per-call location. Nil/canceled/missing catalog returns real configuration/cancel causes without fake HTTP receipt.
  Tests: `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsShareCachedIdentityAndInjectLocationBeforeSelection`, `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsPerCallLocationSnapshotAndNoNamePushdownForSearch`, `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsPreflightAndMissingEndpointPreserveRealConfigurationCause`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Concrete functional options own defaults and defensive copies. Runtime Resource class/session builders are SDK-owned; safe UTF-8 routes, nonnegative Go pagination caps, same-origin/path/filter continuation and stronger cycle bounds are deliberate Go policies.
- Source-selected literal link precedence, ignored Keystone dictionary links and limit-on-short-page marker fallback are enabled for this new Project profile without changing existing native List or older engine profiles. Valid RFC HTTP Link remains a documented Go interoperability correction of the pinned Requests uri/url failure.
- SDK-R1/C1/S1 remain separately tracked whole-SDK Connection Resource runtime, optional response cache and session/version negotiation limitations. Direct leaf location is null; typed ProjectReadLocation or Connection bridge supplies configured facts. This named read mapping is not whole-SDK parity or live-cloud validation.
- ProjectListResult.Value is []any and retains opaque Source generators; JSON is populated only for wholly representable results. Object-valued Proxy expressions iterate sorted Go keys rather than Python insertion order, an explicit deterministic Go result policy.

### `python:cloud/_identity/IdentityCloudMixin/search_projects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity.ProjectReadCloud.SearchProjects`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchProjects`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadLocation`

- Search calls list_projects without a server name hint and then applies exact/glob ID/name plus ordered nested loaded-JSON filters. Wire enabled passes server query but fails local Source Resource membership; is_enabled is the owned attribute name.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudSearchGlobNestedLoadedEqualityAndWireAliasFailures`.
- Truthful Source cloud list expansion rejects truthy JMESPath strings/nonmapping filters before HTTP; falsey filters use the ordinary owned list and literal empty Proxy list produces a nonnil empty search inventory.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudFalseyFiltersAndTruthyNonMappingRejectBeforeHTTP`, `identity/project_record_cloud_test.go:TestPythonProjectCloudProxyLiteralAndOpaqueListResultsDoNotInventHTTP`.
- Typed owned location is supplied before local selection, foreign project IDs clear current names, response location remains untouched in Wire, and caller mutation cannot change location options.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudLocationBeforeLocalSelectionAndOwnedOverrides`.
- Connection cache and CurrentLocation defaults are applied before filtering; per-call location stays owned, Source list-only name option is not pushed from Search, and preflight/catalog failures preserve actual configuration causes.
  Tests: `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsShareCachedIdentityAndInjectLocationBeforeSelection`, `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsPerCallLocationSnapshotAndNoNamePushdownForSearch`, `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsPreflightAndMissingEndpointPreserveRealConfigurationCause`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python sends filters both to the server through list_projects and again to the local filter.
- Concrete functional options own defaults and defensive copies. Runtime Resource class/session builders are SDK-owned; safe UTF-8 routes, nonnegative Go pagination caps, same-origin/path/filter continuation and stronger cycle bounds are deliberate Go policies.
- Source-selected literal link precedence, ignored Keystone dictionary links and limit-on-short-page marker fallback are enabled for this new Project profile without changing existing native List or older engine profiles. Valid RFC HTTP Link remains a documented Go interoperability correction of the pinned Requests uri/url failure.
- SDK-R1/C1/S1 remain separately tracked whole-SDK Connection Resource runtime, optional response cache and session/version negotiation limitations. Direct leaf location is null; typed ProjectReadLocation or Connection bridge supplies configured facts. This named read mapping is not whole-SDK parity or live-cloud validation.
- Search result JSON is separate from actual Projects/Inventory. Source opaque/nonmapping list items fail as concrete Go option errors when local Resource filtering cannot inspect them; no fake receipt or row association is added.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import os
import openstack

conn = openstack.connect(cloud=os.environ["OS_CLOUD"])
domain = os.environ["OS_PROJECT_DOMAIN_ID"]

for project in conn.identity.projects(domain_id=domain, limit=2, max_items=2):
    print(json.dumps(project.to_dict(), indent=2))

project = conn.identity.find_project(
    "demo", domain_id=domain, skip_cache=True,
    error_message="Project lookup failed",
)
if project is not None:
    print(json.dumps(project.to_dict(), indent=2))

listed = conn.list_projects(domain_id=domain, filters={"is_enabled": True})
print(json.dumps([project.to_dict() for project in listed], indent=2))
searched = conn.search_projects("demo*", domain_id=domain,
                                filters={"is_enabled": True})
print(json.dumps([project.to_dict() for project in searched], indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    domain := os.Getenv("OS_PROJECT_DOMAIN_ID")

    for project, err := range service.Projects.ListRecords(ctx,
        projects.WithRecordListQuery("domain_id", domain),
        projects.WithRecordListLimit(2), projects.WithRecordListMaxItems(2)) {
        if err != nil { log.Fatal(err) }
        printJSON(project.Resource)
    }
    project, err := service.Projects.FindRecord(ctx, "demo",
        projects.WithRecordFindQuery("domain_id", domain),
        projects.WithRecordFindQuery("skip_cache", true),
        projects.WithRecordFindQuery("error_message", "Project lookup failed"))
    if err != nil { log.Fatal(err) }
    if project != nil { printJSON(project.Resource) }

    filters := identity.WithProjectReadFilters(json.RawMessage(`{"is_enabled":true}`))
    listed, err := conn.ListProjects(ctx,
        identity.WithProjectReadDomainID(domain), filters)
    if err != nil { log.Fatal(err) }
    printJSON(listed.JSON)
    searched, err := conn.SearchProjects(ctx, "demo*",
        identity.WithProjectReadDomainID(domain), filters)
    if err != nil { log.Fatal(err) }
    printJSON(searched.Value)
}
```

## References

- [openstacksdk ef55d7d](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/project.py)
- [Python Record Find](../users/python-record-find.md)
- [Source list_projects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L82)
- [Project](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py#L157-L174)
- [Base](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)
