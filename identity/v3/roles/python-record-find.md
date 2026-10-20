# Python Record Find

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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
    role = conn.identity.find_role(
        args.identity, ignore_missing=False, domain_id=args.domain,
        skip_cache=True, error_message="Role lookup failed",
    )
    print(json.dumps(role.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("identity", "", "Role exact name or ID")
    domain := flag.String("domain", "", "explicit domain ID")
    flag.Parse()
    if *identity == "" || *domain == "" { log.Fatal("identity and domain are required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    row, err := service.Roles.FindRecord(ctx, *identity,
        roles.WithRecordFindIgnoreMissing(false),
        roles.WithRecordFindQuery("domain_id", *domain),
        roles.WithRecordFindQuery("skip_cache", true),
        roles.WithRecordFindQuery("error_message", "Role lookup failed"))
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(row.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Python Records](python-records.md)
- [Install](../../../docs/install.md)
