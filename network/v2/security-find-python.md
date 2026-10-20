# Security Find Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

### `python:network/v2/find_security_group_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordFindLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordFindMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordFindMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordFindPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithRecordFindMicroversion`

- Member-first lookup preserves raw kwargs and Source constructor seed independently from actual HTTP row. Source successful statuses200..399 are supported; malformed/empty JSON retains seed, valid nonobjects and malformed UTF8 retain accepted response proof.
  Tests: `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordGetFirstRawKwargsSeedAndWire`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordEmptyMalformedAndWrongShapes`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordNullableIdentityAndLiteralEscape`.
- Only clean native400/403/404 including an unchanged clean native retry permit fallback; successful full absence defaults nil,nil and strict ErrNotFound does not attach suppressed initial rejection. Fallback scans exact ID OR name across all observed pages and retains late duplicates/errors. Group has automatic name hint; Rule inherited name is compared but has no server hint.
  Tests: `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordFallbackCodesMissingAndCleanRetry`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordLateDuplicateAndPageFailure`.
- GET keeps original kwargs while LIST applies Source server/alias classification and declared nonserver Body filters. Client query alias wins even as null; unknown kwargs seed/member query only; consumed all_projects has no Neutron wire effect. Rule name-null is a local filter, Group name-null omits automatic/server hint.
  Tests: `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordQueryAliasesScopeAndLocalFilters`.
- Record-owned Body/default/header descriptor view and actual Wire/Envelope/status/header stay independent. Header wire if-match aliases are case-insensitive while canonical if_match is exact; IF_MATCH remains unknown Body in constructor/LIST and is not synthesized from actual HTTP Header. Ordered aliases retain last-value precedence. LIST cls/connection/microversion/_synchronized constructor conflicts retain actual whole-page proof.
  Tests: `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordGetFirstRawKwargsSeedAndWire`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordListHeadersAndConstructorControls`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordNullableIdentityAndLiteralEscape`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordHeaderAliasDomains`.
- Source dictionary filtering preserves falsey/empty/nested subset semantics, truthy scalar empty-object matches, mapping.get failures and boolean-number equality. Dictionary scalar decimal truthiness follows Python binary64 underflow/overflow; shared exact-decimal JSON equality is the documented numeric profile.
  Tests: `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordDictionaryTruthinessAndEquality`.
- Shared REST pager preserves plural links/top-level next/HTTP Link, exact Neutron versioned relative aliases, singleton rows, stop-on-empty and explicit limit short-page marker fallback. Raw caps precede local filters and unconsumed bad rows/next parsing. Fixed origin/collection/filter/cursor guards retain page proof.
  Tests: `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordMarkerCapAndSinglePage`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordPaginationRepresentationsAndGuards`.
- Concrete options own maps/raw bytes/pointers/order and retained callback values. Source positional/control ignore_missing/name_or_id/session/cls and constructor conflicts, unsafe literal IDs, malformed seed, negative typed list controls and forbidden auth/version headers fail before HTTP. Context custom and option/marshal causes remain joined; captured source changes are terminal. Reusable options/records remain independent during concurrent use.
  Tests: `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordInvalidControlsAndDescriptorFailBeforeHTTP`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordOptionsSnapshotCancellationAndMarshal`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordConcurrentOptionAndRecordOwnership`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordSourceArgumentCollisions`.
- No class-ceiling discovery is added. Pinned Resource.find applies explicit per-call microversion only to member GET; fallback LIST retains captured service default, including independently owned selected version header. Phase headers/value or absence stay fixed; Source LIST Accept application/json overrides configured service Accept unless an explicit safe Go header option is supplied. Original selected client remains unchanged.
  Tests: `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordProtocolAbsenceAndPerCallVersion`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordRetryKeepsOriginalRejectionAndRequest`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordMicroversionPhasesAndAcceptDefault`.
- Accepted/rejected member and list Read/Close/custom cancel/sticky source restoration errors keep actual status/body/header/causes with one physical close and prevent incorrect lookup fallback/resend. Native request/protocol mutation retains the original HTTP rejection; clean retry and token reauthentication keep route/query/body. Required-absent Authorization/X-Service-Token/framing/hop/foreign-version headers and selected-version OmitHeaders are protected on both member and list native retries; original rejection and custom callback causes remain intact before another physical attempt.
  Tests: `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordPhysicalAcceptedRejectedAndListErrors`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordRetryKeepsOriginalRejectionAndRequest`, `network/v2/extensions/security/rules/python_find_contracts_test.go:TestPythonSecurityRuleFindRecordRetryForbiddenHeadersAndOmittedVersion`.

Python/Go differences:

- Go returns an owned Source-shaped Resource/Wire/Envelope/Header/StatusCode record from a validated literal string identity, rather than mutating/accepting arbitrary Python Resource/class/subclass/session objects. Connection location is null; global Resource/cache/session SDK-R1/C1/S1 work remains separately tracked.
- Sequential query options retain Source constructor kwargs insertion order; bulk Go map/unregistered callback keys and multiple local filters use sorted keys because Go map has no kwargs order. Ordered literal JSON response alias precedence is retained.
- Named Microversion retains the Source GET-only inherited override; fallback LIST uses captured service default. Header applies explicitly in both phases and typed fallback Limit/Marker/MaxItems/Paginated are Go controls. Raw ignore_missing/name_or_id/session/cls/id positional/control/constructor conflicts and route/cache/response-key/header/runtime controls are rejected before HTTP; supported controls use named options. all_projects is consumed without Neutron wire effect.
- Fixed origin/collection/filter/request/protocol and literal one-time escaping guards intentionally reject Source callback/continuation retargeting; raw pagination values are validated by the shared pager. Dictionary links are a Go compatibility extension. Explicit cap/single-page returns only the observed subset and cannot rule out unobserved duplicates.
- Unknown values, original numeric tokens/timestamps and nested extensions stay lossless in Wire. Source int descriptors use audited Unicode16/CPython3.14 profile; bool and dictionary scalar truthiness use binary64 underflow/overflow. Shared JSON equality uses exact decimals plus Python boolean-number equality rather than all Python float rounding/nonfinite/custom-object runtime behavior.
- Accepted malformed UTF8 fails rather than Requests replacement decoding; valid wrong shapes and physical body failures retain actual ACK. Fixture/race and external main compilation are distinct from authenticated deployment Neutron authorization/endpoint validation.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
group = conn.network.find_security_group("web", project_id="PROJECT_ID")
rule = conn.network.find_security_group_rule("RULE_ID", security_group_id="GROUP_ID")
strict = conn.network.find_security_group("web", ignore_missing=False)
print(None if group is None else group.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printJSON(value *resource.RawResource) {
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    group := flag.String("group", "", "security group exact name or ID")
    rule := flag.String("rule", "", "security group rule exact ID or name")
    project := flag.String("project", "", "optional project_id query")
    groupFilter := flag.String("security-group", "", "optional security_group_id rule query")
    strict := flag.Bool("strict", false, "return an error when missing")
    flag.Parse()
    if *group == "" && *rule == "" { log.Fatal("-group or -rule is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    if *group != "" {
        options := []groups.RecordFindOption{groups.WithRecordFindIgnoreMissing(!*strict)}
        if *project != "" {
            options = append(options, groups.WithRecordFindQuery("project_id", *project))
        }
        value, err := service.SecurityGroups.FindRecord(ctx, *group, options...)
        if err != nil { log.Fatal(err) }
        if value == nil { printJSON(nil) } else { printJSON(value.Resource) }
    }
    if *rule != "" {
        options := []rules.RecordFindOption{rules.WithRecordFindIgnoreMissing(!*strict)}
        if *project != "" {
            options = append(options, rules.WithRecordFindQuery("project_id", *project))
        }
        if *groupFilter != "" {
            options = append(options, rules.WithRecordFindQuery("security_group_id", *groupFilter))
        }
        value, err := service.SecurityRules.FindRecord(ctx, *rule, options...)
        if err != nil { log.Fatal(err) }
        if value == nil { printJSON(nil) } else { printJSON(value.Resource) }
    }
}
```

## References

- [Security Group proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6598-L6623)
- [Rule proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6768-L6793)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)
- [Install](../../docs/install.md)
