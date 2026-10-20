# Python Record Find

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--domain", required=True)
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.identity.find_user(
        args.identity, ignore_missing=False, domain_id=args.domain,
        skip_cache=True, error_message="User lookup failed",
    )
    print(json.dumps(result.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("identity", "", "User exact name or ID")
    domain := flag.String("domain", "", "explicit domain ID")
    flag.Parse()
    if *identity == "" || *domain == "" { log.Fatal("identity and domain are required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    row, err := service.Users.FindRecord(ctx, *identity,
        users.WithRecordFindIgnoreMissing(false),
        users.WithRecordFindQuery("domain_id", *domain),
        users.WithRecordFindQuery("skip_cache", true),
        users.WithRecordFindQuery("error_message", "User lookup failed"))
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(row.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Install](../../../docs/install.md)
