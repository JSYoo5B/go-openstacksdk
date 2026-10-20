# Dscp Marking Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.FindDSCPMarkingRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingFindIgnoreMissing`

- Required qos_policy_id is Source constructor URI and member fetch parameter: GET forwards it plus original kwargs, no parent GET. All_projects is consumed separately. Bound URI is removed from Body defaults. GET unknown Body seeds remain Source-shaped while response explicit values override them; caller bound policy/identity kwargs collision fails before HTTP.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingFindBoundRequiredKwargsAndUnknownConstructorSeeds`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPreflightBoundControlsCancellationAndSnapshots`.
- GET-first clean400/403/404 fallback scans exact ID or name across all pages for complete uniqueness and default ignore-missing. No name server hint is invented because Source inherits limit/marker-only QueryParameters. dscp_mark/name/id are local filters; unknown/member-only kwargs are dropped on fallback. List errors/duplicate/later page rejection remain visible; clean empty search alone yields defaultnil or strictErrNotFound.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingFindCleanFallbackLocalFiltersAndNoNameHint`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingFindCompletesUniquenessAndMissingSearch`.
- Owned engine SourceFindPolicy retains ordered local predicates, constructor/default/unknown response behavior and guarded raw-row cap/paging; local dictionary scalar faults cannot become missing. Actual Source kwargs use provided JSON request values, with documented shared exact-decimal rather than binary64-rounding equality policy.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingListOrderedDictionaryFiltersAndExactNumbers`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingListPagingBoundsAndConstructorProof`.
- Bound policy literal ID controls escaped parent/member path without parent GET or name resolution. Source URI is exposed as independent PolicyID, not invented Body default. id/name/dscp_mark are untyped with null defaults, inherited location null and no foreign Neutron tags/Header/revision/timestamp descriptors. Unknown Body extension and exact original JSON values/Wire/self/evidence are owned independently.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingGetLiteralPolicyRouteRawProjectionAndDefaults`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingFindBoundRequiredKwargsAndUnknownConstructorSeeds`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingListServerClassificationRawCapAndOwnedURI`.
- Captured selected Neutron source and concrete JSON/header/pointer options remain owned across option callbacks, retained configs, physical Read/Close, cancellation, native clean retry and live authentication. Accepted physical failures keep actual response proof; rejected400..599 retain native method/URL/status/body/Header and physical causes, one request/Close and no lookup fallback or manufactured decoded Resource.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPhysicalFaultsPreventFallbackAndKeepEvidence`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPreflightBoundControlsCancellationAndSnapshots`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPublicFixtureAndConcurrentOwnedReuse`.
- Selected/default network microversion is retained without invented discovery/ceiling; explicit Find version affects GET only and fallback retains Source service default. Get/List explicit version owns its invocation. Fixed source bodyless method/path/query and network version/absence plus auth/framing/foreign Nova/Cinder headers prevent native retry injection while ordinary retry Header/live provider tokens work.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingVersionsAndRetryHeadersRemainSelected`.
- Public Gophercloud testhelper HTTP fixture and existing nativeRuleAPI/transport are reused. Concurrent factory reuse, actual escaped member path and query-free GET are fixture-proven; no additional transport or paging engine is implemented.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPublicFixtureAndConcurrentOwnedReuse`.
- Source Computed location remains ownednull for GET/Find/List even if physical response includes forged object, false, zero, emptystring or null. The actual rawlocation and wholepage Envelope stay in independent Wire/receipt, and RawResource.Clone plus mutation of cloned/original wire/header cannot contaminate Resource view or original physical Envelope. Shared SourceFindPolicy projection enforces computed ownership rather than treating location as ordinary unknown Body.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingComputedLocationStaysOwnedAcrossReads`.

Python/Go differences:

- Actual generated leaf is network/v2/extensions/qos/rules; no dscpmarking subpackage exists. Exactly three unresolved named Source operations are mapped; existing native typed DSCP CRUD/List and generated parent Collection are not recounted. Shared neutronfind.Engine Get/Find/List with SourceFindPolicy plus cloudread/REST owns transport/pagination/defaults, with no new engine/mock server.
- Pinned QoSDSCPMarkingRule directly inherits Resource, allows unknown Body and declares untyped dscp_mark and URI qos_policy_id. Go exposes URI as PolicyID separately from Source-shaped Body; original response URI-looking values remain in Wire while member Resource getters reflect bound parent. Actual response location remains raw in Wire/Envelope while owned Resource computed location is null across GET/Find/List; no server location-to-Body leakage. Mutable Resource/Munch parent seeds/dirty-state/cache/Connection/location synthesis/general Adapter/session remain global SDK-R1/C1/S1.
- Source Proxy parent _get_resource seeds identity without parent HTTP; Go accepts a validated literal parentID string and escapes parent/child once. Blank/dot/control/invalidUTF8 parent/child, caller bound kwargs, arbitrary base_path/session/constructor overrides are explicit Go guards. Shared policy read is not shared rule-read authority.
- Source get_qos_dscp_marking_rule signature is query-free. Additional Go concrete GetQuery/Header/Microversion options are transport extensions and never become response Body seed. Find GET forwards original kwargs including required qos_policy_id while fallback/list classify only inherited limit/marker and actual local Body properties; user-supplied QoS policy/identity required kwargs collisions are rejected.
- Shared Engine accepted200..399 is narrower than pinned generic raise_from_response valid100..399 for transport-exposed1xx. Existing SDK-C1 exact-decimal JSON equality preserves numeric evidence rather than complete CPython binary64 equality. API profiles retain these explicit shared policies; no fake selected microversion limit/discovery or network rule scalar casts are invented.
- Pinned Neutron27 get_policy_dscp_marking_rule uses ADMIN_OR_PARENT_OWNER_READER/project scope for collection and member. The shared_qos_policy condition belongs to get_policy, not rule read. Actual HTTP/RBAC/deployment extension errors remain server observations; local fixtures do not establish authenticated cloud acceptance or writable project-manager/admin parity.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

### `python:network/v2/get_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetDSCPMarkingRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingGetQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingGetMicroversion`

- One query-free named default GET at the bound policy/member path; no parent lookup, fallback or404 ignore. Optional Go transport query does not seed Body: accepted blank/malformed JSON keeps identity/defaults while actual null/scalar/wrong singular/invalidUTF8 are response evidence errors. Source getter has no kwargs; wider Go query/Header/version extensions are explicit.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingGetLiteralPolicyRouteRawProjectionAndDefaults`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingGetPartialReceiptsWithoutQuerySeedsOrFallback`.
- Bound policy literal ID controls escaped parent/member path without parent GET or name resolution. Source URI is exposed as independent PolicyID, not invented Body default. id/name/dscp_mark are untyped with null defaults, inherited location null and no foreign Neutron tags/Header/revision/timestamp descriptors. Unknown Body extension and exact original JSON values/Wire/self/evidence are owned independently.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingGetLiteralPolicyRouteRawProjectionAndDefaults`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingFindBoundRequiredKwargsAndUnknownConstructorSeeds`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingListServerClassificationRawCapAndOwnedURI`.
- Captured selected Neutron source and concrete JSON/header/pointer options remain owned across option callbacks, retained configs, physical Read/Close, cancellation, native clean retry and live authentication. Accepted physical failures keep actual response proof; rejected400..599 retain native method/URL/status/body/Header and physical causes, one request/Close and no lookup fallback or manufactured decoded Resource.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPhysicalFaultsPreventFallbackAndKeepEvidence`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPreflightBoundControlsCancellationAndSnapshots`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPublicFixtureAndConcurrentOwnedReuse`.
- Selected/default network microversion is retained without invented discovery/ceiling; explicit Find version affects GET only and fallback retains Source service default. Get/List explicit version owns its invocation. Fixed source bodyless method/path/query and network version/absence plus auth/framing/foreign Nova/Cinder headers prevent native retry injection while ordinary retry Header/live provider tokens work.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingVersionsAndRetryHeadersRemainSelected`.
- Public Gophercloud testhelper HTTP fixture and existing nativeRuleAPI/transport are reused. Concurrent factory reuse, actual escaped member path and query-free GET are fixture-proven; no additional transport or paging engine is implemented.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPublicFixtureAndConcurrentOwnedReuse`.
- Source Computed location remains ownednull for GET/Find/List even if physical response includes forged object, false, zero, emptystring or null. The actual rawlocation and wholepage Envelope stay in independent Wire/receipt, and RawResource.Clone plus mutation of cloned/original wire/header cannot contaminate Resource view or original physical Envelope. Shared SourceFindPolicy projection enforces computed ownership rather than treating location as ordinary unknown Body.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingComputedLocationStaysOwnedAcrossReads`.

Python/Go differences:

- Actual generated leaf is network/v2/extensions/qos/rules; no dscpmarking subpackage exists. Exactly three unresolved named Source operations are mapped; existing native typed DSCP CRUD/List and generated parent Collection are not recounted. Shared neutronfind.Engine Get/Find/List with SourceFindPolicy plus cloudread/REST owns transport/pagination/defaults, with no new engine/mock server.
- Pinned QoSDSCPMarkingRule directly inherits Resource, allows unknown Body and declares untyped dscp_mark and URI qos_policy_id. Go exposes URI as PolicyID separately from Source-shaped Body; original response URI-looking values remain in Wire while member Resource getters reflect bound parent. Actual response location remains raw in Wire/Envelope while owned Resource computed location is null across GET/Find/List; no server location-to-Body leakage. Mutable Resource/Munch parent seeds/dirty-state/cache/Connection/location synthesis/general Adapter/session remain global SDK-R1/C1/S1.
- Source Proxy parent _get_resource seeds identity without parent HTTP; Go accepts a validated literal parentID string and escapes parent/child once. Blank/dot/control/invalidUTF8 parent/child, caller bound kwargs, arbitrary base_path/session/constructor overrides are explicit Go guards. Shared policy read is not shared rule-read authority.
- Source get_qos_dscp_marking_rule signature is query-free. Additional Go concrete GetQuery/Header/Microversion options are transport extensions and never become response Body seed. Find GET forwards original kwargs including required qos_policy_id while fallback/list classify only inherited limit/marker and actual local Body properties; user-supplied QoS policy/identity required kwargs collisions are rejected.
- Shared Engine accepted200..399 is narrower than pinned generic raise_from_response valid100..399 for transport-exposed1xx. Existing SDK-C1 exact-decimal JSON equality preserves numeric evidence rather than complete CPython binary64 equality. API profiles retain these explicit shared policies; no fake selected microversion limit/discovery or network rule scalar casts are invented.
- Pinned Neutron27 get_policy_dscp_marking_rule uses ADMIN_OR_PARENT_OWNER_READER/project scope for collection and member. The shared_qos_policy condition belongs to get_policy, not rule read. Actual HTTP/RBAC/deployment extension errors remain server observations; local fixtures do not establish authenticated cloud acceptance or writable project-manager/admin parity.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

### `python:network/v2/qos_dscp_marking_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ListDSCPMarkingRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.AllDSCPMarkingRecords`

- Lazy bound policy list defaults Accept application/json. Only inherited limit/marker are server queries; class-declared id/name/dscp_mark filter locally and unknowns are discarded before marshaling. Raw MaxItems includes filtered rows and supplies falsey-limit hint; Source URI is fixed independently of physical row qos_policy_id. Declared DSCP mark is untyped and exact false/null/zero/large extension values remain owned.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingListServerClassificationRawCapAndOwnedURI`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingListOrderedDictionaryFiltersAndExactNumbers`.
- Shared list pager handles plural/singleton rows, body next and versioned route, link/count-free truthy-limit short-page marker fallback, raw caps/single-page and exact fixed policy path/filter/cycle guards. Full JSON malformed/missing/null envelope and consumed wrong/cls/connection constructor rows retain actual page proof; ignored unused bad rows and unsafe continuation after raw cap remain unevaluated.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingListServerClassificationRawCapAndOwnedURI`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingListPagingBoundsAndConstructorProof`.
- WithListQuery preserves top-level local predicate option order and raw nested dictionary order; firstfalse short-circuits later mapping faults. Shared SDK-C1 exact JSON equality intentionally does not round actual9007199254740992 to rawfilter9007199254740993.0; truthful empty dictionary/scalar and nonempty mapping errors preserve Source mechanics.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingListOrderedDictionaryFiltersAndExactNumbers`.
- Bound policy literal ID controls escaped parent/member path without parent GET or name resolution. Source URI is exposed as independent PolicyID, not invented Body default. id/name/dscp_mark are untyped with null defaults, inherited location null and no foreign Neutron tags/Header/revision/timestamp descriptors. Unknown Body extension and exact original JSON values/Wire/self/evidence are owned independently.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingGetLiteralPolicyRouteRawProjectionAndDefaults`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingFindBoundRequiredKwargsAndUnknownConstructorSeeds`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingListServerClassificationRawCapAndOwnedURI`.
- Captured selected Neutron source and concrete JSON/header/pointer options remain owned across option callbacks, retained configs, physical Read/Close, cancellation, native clean retry and live authentication. Accepted physical failures keep actual response proof; rejected400..599 retain native method/URL/status/body/Header and physical causes, one request/Close and no lookup fallback or manufactured decoded Resource.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPhysicalFaultsPreventFallbackAndKeepEvidence`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPreflightBoundControlsCancellationAndSnapshots`, `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPublicFixtureAndConcurrentOwnedReuse`.
- Selected/default network microversion is retained without invented discovery/ceiling; explicit Find version affects GET only and fallback retains Source service default. Get/List explicit version owns its invocation. Fixed source bodyless method/path/query and network version/absence plus auth/framing/foreign Nova/Cinder headers prevent native retry injection while ordinary retry Header/live provider tokens work.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingVersionsAndRetryHeadersRemainSelected`.
- Public Gophercloud testhelper HTTP fixture and existing nativeRuleAPI/transport are reused. Concurrent factory reuse, actual escaped member path and query-free GET are fixture-proven; no additional transport or paging engine is implemented.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingPublicFixtureAndConcurrentOwnedReuse`.
- Source Computed location remains ownednull for GET/Find/List even if physical response includes forged object, false, zero, emptystring or null. The actual rawlocation and wholepage Envelope stay in independent Wire/receipt, and RawResource.Clone plus mutation of cloned/original wire/header cannot contaminate Resource view or original physical Envelope. Shared SourceFindPolicy projection enforces computed ownership rather than treating location as ordinary unknown Body.
  Tests: `network/v2/extensions/qos/rules/python_dscp_marking_contracts_test.go:TestPythonDSCPMarkingComputedLocationStaysOwnedAcrossReads`.

Python/Go differences:

- Actual generated leaf is network/v2/extensions/qos/rules; no dscpmarking subpackage exists. Exactly three unresolved named Source operations are mapped; existing native typed DSCP CRUD/List and generated parent Collection are not recounted. Shared neutronfind.Engine Get/Find/List with SourceFindPolicy plus cloudread/REST owns transport/pagination/defaults, with no new engine/mock server.
- Pinned QoSDSCPMarkingRule directly inherits Resource, allows unknown Body and declares untyped dscp_mark and URI qos_policy_id. Go exposes URI as PolicyID separately from Source-shaped Body; original response URI-looking values remain in Wire while member Resource getters reflect bound parent. Actual response location remains raw in Wire/Envelope while owned Resource computed location is null across GET/Find/List; no server location-to-Body leakage. Mutable Resource/Munch parent seeds/dirty-state/cache/Connection/location synthesis/general Adapter/session remain global SDK-R1/C1/S1.
- Source Proxy parent _get_resource seeds identity without parent HTTP; Go accepts a validated literal parentID string and escapes parent/child once. Blank/dot/control/invalidUTF8 parent/child, caller bound kwargs, arbitrary base_path/session/constructor overrides are explicit Go guards. Shared policy read is not shared rule-read authority.
- Source get_qos_dscp_marking_rule signature is query-free. Additional Go concrete GetQuery/Header/Microversion options are transport extensions and never become response Body seed. Find GET forwards original kwargs including required qos_policy_id while fallback/list classify only inherited limit/marker and actual local Body properties; user-supplied QoS policy/identity required kwargs collisions are rejected.
- Shared Engine accepted200..399 is narrower than pinned generic raise_from_response valid100..399 for transport-exposed1xx. Existing SDK-C1 exact-decimal JSON equality preserves numeric evidence rather than complete CPython binary64 equality. API profiles retain these explicit shared policies; no fake selected microversion limit/discovery or network rule scalar casts are invented.
- Pinned Neutron27 get_policy_dscp_marking_rule uses ADMIN_OR_PARENT_OWNER_READER/project scope for collection and member. The shared_qos_policy condition belongs to get_policy, not rule read. Actual HTTP/RBAC/deployment extension errors remain server observations; local fixtures do not establish authenticated cloud acceptance or writable project-manager/admin parity.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--policy", required=True)
    parser.add_argument("--operation", choices=("list", "get", "find"), default="list")
    parser.add_argument("--rule", default="")
    parser.add_argument("--page-size", type=int, default=0)
    parser.add_argument("--max-items", type=int, default=0)
    parser.add_argument("--mark", type=int)
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect()
    if args.operation == "get":
        result = conn.network.get_qos_dscp_marking_rule(args.rule, args.policy)
        print(result.to_dict())
    elif args.operation == "find":
        result = conn.network.find_qos_dscp_marking_rule(
            args.rule, args.policy, ignore_missing=not args.strict
        )
        print(None if result is None else result.to_dict())
    else:
        query = {"limit": args.page_size, "max_items": args.max_items}
        if args.mark is not None:
            query["dscp_mark"] = args.mark
        for rule in conn.network.qos_dscp_marking_rules(args.policy, **query):
            print(rule.qos_policy_id, rule.to_dict())


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

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
)

func show(record *rules.DSCPMarkingRecord) {
    if record == nil { fmt.Println("missing"); return }
    body, err := json.Marshal(record.Resource.Body)
    if err != nil { log.Fatal(err) }
    fmt.Printf("policy=%s HTTP=%d body=%s\n", record.PolicyID, record.StatusCode, body)
}

func main() {
    policy := flag.String("policy", "", "parent policy ID")
    operation := flag.String("operation", "list", "list, get or find")
    rule := flag.String("rule", "", "literal rule ID or find identity")
    pageSize := flag.Int("page-size", 0, "server page size")
    maxItems := flag.Int("max-items", 0, "raw row cap; zero means no cap")
    mark := flag.Int("mark", -1, "optional local DSCP mark filter")
    strict := flag.Bool("strict", false, "find absence is an error")
    flag.Parse()
    if *policy == "" { log.Fatal("--policy is required") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    switch *operation {
    case "get":
        record, err := network.QoSRules.GetDSCPMarkingRecord(ctx, *policy, *rule)
        if err != nil { log.Fatal(err) }
        show(record)
    case "find":
        record, err := network.QoSRules.FindDSCPMarkingRecord(ctx, *policy, *rule,
            rules.WithDSCPMarkingFindIgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        show(record)
    case "list":
        options := []rules.DSCPMarkingListOption{
            rules.WithDSCPMarkingListLimit(*pageSize),
            rules.WithDSCPMarkingListMaxItems(*maxItems),
        }
        if *mark >= 0 {
            options = append(options, rules.WithDSCPMarkingListQuery("dscp_mark", *mark))
        }
        for record, err := range network.QoSRules.ListDSCPMarkingRecords(ctx, *policy, options...) {
            if err != nil { log.Fatal(err) }
            show(record)
        }
    default:
        log.Fatal("--operation must be list, get or find")
    }
}
```

## References

- [Source rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_dscp_marking_rule.py)
- [Neutron 27.0.0 policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L337-L357)
- [Python Dscp Marking Contracts Test](python_dscp_marking_contracts_test.go)
