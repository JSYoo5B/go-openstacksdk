# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

### `python:identity/v3/groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.GroupRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithRecordListLocation`

- List sends the single GET groups that Python groups() sends without arguments, maps the Python query names domain_id and name through ListOpts or WithListQuery, sends a limit without a marker follow-up, and rejects local Body filters before HTTP.
  Tests: `identity/v3/groups/python_parity_test.go:TestPythonGroupListRequests`.
- Lazy Group records classify limit/marker/domain_id/name as server query, description/id as ordered local Body filters, discard unknown before marshaling and keep nullable raw Body separate from original independent Wire/Envelope/header/status.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsLazyUntypedDeclaredViewAndEvidence`.
- Source root links dictionary is ignored with present-key precedence over groups_links; explicit limit continues even short nonempty pages by last raw wire ID until empty. Local filtering does not change marker, raw max_items precedes filtering and break has no lookahead. Explicit root links:null is a Source iterable error and preserves actual page proof.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsSourceDictionaryLinksAndPresence`, `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsFilteredMarkerRawCapAndBreak`.
- Loaded local equality preserves CPython float rounding/underflow and bool/integer equality, ordered recursive subset/actual truthiness, mismatch short circuit and actual proof for reached bad shapes.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsOrderedLoadedLocalEquality`.
- Proxy JMESPath receives opaque unconsumed generator: projection/type/literal JSON HTTP0; @ keeps lazy generator; nested generator has no fabricated JSON; type/invalid expression does not request and falsey/nonstring remains lazy.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsProxyExpressionsRemainOpaqueUntilConsumed`.
- Owned request headers/microversion/base_path formal arguments and one-pass __conflicting_attrs restoration preserve restored query/header/version, original JMESPath priority, unknown attr discard and HTTP0 bound collisions.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsProxyFormalRestorationAndBoundCollision`.
- Immutable options plus sticky Source/context guards run before and after every callback; source-change then restore callback cannot escape and bad preflight creates HTTP0 without fabricated ResponseError.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsOwnedOptionsStickySourceAndCancellation`.
- Physical Read/Close/sticky Source restored by Close/custom cancel matrix for200/400/403/404/401/429/503 stops Retry/Backoff/Reauth/fallback with actual body/header/status and pointer401 wrapper and separately inspected ErrOriginal plus ErrReauth. Accepted200 keeps ResponseError; rejected400+ keeps native ErrUnexpectedResponseCode proof and pointer401 separates ErrOriginal/ErrReauth.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsFaultedActualRejectionsNeverFallbackOrRetry`.
- Clean native401/503/429/498 uses live provider token and captured header/route/query with bodyless GET; successful result retains actual HTTP203.
  Tests: `identity/v3/groups/python_records_test.go:TestPythonGroupRecordsCleanNativeRetryUsesLiveAuthAndCapturedHeaders`.

Python/Go differences:

- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Python sends bool query values as True/False text; typed Go ListOpts send true/false, and WithListQuery reproduces the Python text.
- Python drops unknown query names; Go sends every WithListQuery key to the server.
- groups.ListOpts.Filters adds a stray "-" query key next to the comparator filters; Python cannot send comparator filters at all.
- Concrete Go options own request/defaults, nullable raw Resource and independent actual response channels. Existing native List/Get/FindIdentity semantics are unchanged.
- Already selected Keystone v3 API required; no automatic v2 ensure_service_version conversion or Connection current location inference. Explicit immutable CloudLocation retains current project because Group declares no project_id/zone.
- Fixed origin/collection/query/header/version bounds reject foreign URL/traversal/cycle and malformed identities/markers using actual receipt. Paging amounts are nonnegative Go integers.
- iter.Seq2 can be iterated again and repeats captured requests rather than Python generator single-consumption state. Proxy pure expressions still perform SDK option/source preflight before returning.
- Connection cache and generic synchronized Resource object lifecycle remain separately tracked as SDK-R1/C1; finite Group read mapping is not full SDK parity.
- RFC HTTP Link is deliberate Go interoperability: Source requests next[uri] does not reproduce the ordinary Requests next[url] shape; root Source-link JSON precedence is exact.

### `python:cloud/_identity/IdentityCloudMixin/search_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewGroupReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.GroupReadCloud.SearchGroups`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchLocation`

- search_groups lists with the same GET /v3/groups (with domain_id when given) as list_groups before filtering locally.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityDomainGroupRoleHelpers`.
- Cloud materializes every Group page with only server domain_id, then exact/glob ID/name and ordered nested loaded local filtering; duplicate rows remain and selected records clone independently of physical inventory.
  Tests: `identity/group_search_records_test.go:TestPythonGroupCloudSearchDomainServerAndLocalGlobSubset`.
- Truthy string JMESPath sees materialized declared rows after identifier selection and yields arbitrary scalar/null/list JSON with no fabricated record association; falsey filters preserve rows and Python str(None) identifier behavior.
  Tests: `identity/group_search_records_test.go:TestPythonGroupCloudSearchStringExpressionsUseMaterializedRows`, `identity/group_search_records_test.go:TestPythonGroupCloudFalseyFiltersPreserveDeclaredRows`.
- Every page precedes local-filter evaluation, so invalid expression/unknown key/reached bad shape preserves Inventory after HTTP; ordered mismatch skips bad later shape and late HTTP503 wins over invalid expression.
  Tests: `identity/group_search_records_test.go:TestPythonGroupCloudLocalErrorsFollowEveryPageAndKeepInventory`, `identity/group_search_records_test.go:TestPythonGroupCloudOrderedMismatchHidesUnreachedBadShape`, `identity/group_search_records_test.go:TestPythonGroupCloudLateHTTPFailurePrecedesInvalidLocalFilter`.
- Explicit location is projected before local selection, unknown response project_id/location cannot replace current scope, and caller filters/domain/headers/location are captured without aliases.
  Tests: `identity/group_search_records_test.go:TestPythonGroupCloudLocationBeforeFilterAndOwnedArguments`.
- Selected Source/header snapshot precedes options; before/after each callback sticky Source/custom cancellation guards prevent restore callback/HTTP and invalid preflight has no invented receipt.
  Tests: `identity/group_search_records_test.go:TestPythonGroupCloudOptionsGuardCallbacksAndSelectedSourceSnapshot`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Concrete Go options own request/defaults, nullable raw Resource and independent actual response channels. Existing native List/Get/FindIdentity semantics are unchanged.
- Already selected Keystone v3 API required; no automatic v2 ensure_service_version conversion or Connection current location inference. Explicit immutable CloudLocation retains current project because Group declares no project_id/zone.
- Fixed origin/collection/query/header/version bounds reject foreign URL/traversal/cycle and malformed identities/markers using actual receipt. Paging amounts are nonnegative Go integers.
- iter.Seq2 can be iterated again and repeats captured requests rather than Python generator single-consumption state. Proxy pure expressions still perform SDK option/source preflight before returning.
- Connection cache and generic synchronized Resource object lifecycle remain separately tracked as SDK-R1/C1; finite Group read mapping is not full SDK parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack


def main():
    conn = openstack.connect()
    found = conn.identity.find_group(
        "ops", domain_id="domain-id", skip_cache=True,
        error_message="Group lookup failed",
    )
    if found is not None:
        print(json.dumps(found.to_dict(), indent=2, ensure_ascii=False))

    for row in conn.identity.groups(
        domain_id="domain-id", description="operations", limit=100,
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

    selected = conn.search_groups(
        name_or_id="ops-*", domain_id="domain-id",
        filters={"description": "operations"},
    )
    print(json.dumps([row.to_dict() for row in selected], indent=2))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
)

func printJSON(value any) {
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("%s\n", encoded)
}

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }

    found, err := service.Groups.FindRecord(ctx, "ops",
        groups.WithRecordFindQuery("domain_id", "domain-id"),
        groups.WithRecordFindQuery("skip_cache", true),
        groups.WithRecordFindQuery("error_message", "Group lookup failed"),
    )
    if err != nil { log.Fatal(err) }
    if found != nil { printJSON(found.Resource.Body) }

    for row, err := range service.Groups.ListRecords(ctx,
        groups.WithRecordListQuery("domain_id", "domain-id"),
        groups.WithRecordListQuery("description", "operations"),
        groups.WithRecordListLimit(100),
        groups.WithRecordListMaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        printJSON(row.Resource.Body)
    }

    cloud := identity.NewGroupReadCloud(service.Groups)
    selected, err := cloud.SearchGroups(ctx, "ops-*",
        identity.WithGroupSearchDomainID("domain-id"),
        identity.WithGroupSearchFilters(json.RawMessage(`{"description":"operations"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(selected.Value)
}
```

```go
proxy, err := service.Groups.ProxyListRecords(ctx,
    groups.WithRecordListJMESPath("[*].id"),
)
if err != nil { log.Fatal(err) }
fmt.Println(string(proxy.JSON))
```

## References

- [Policy](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)
- [Group](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/common/policies/group.py)
- [Groups](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/api/groups.py)
- [User Find](../users/python-record-find.md)
- [Project Find](../projects/python-record-reads.md)
