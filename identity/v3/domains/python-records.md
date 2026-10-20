# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/domains`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.DomainRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordListLocation`

- List sends the single GET domains that Python domains() sends without arguments, maps the Python query names name and enabled through ListOpts or WithListQuery, sends a limit without a marker follow-up, and rejects local Body filters before HTTP.
  Tests: `identity/v3/domains/python_parity_test.go:TestPythonDomainListRequests`.
- Source Domain declares raw description/links/id/name, nullable CPython bool is_enabled from enabled and dict options (nonobject {}, null preserved), unknown Body discarded from declared view but complete Wire/Envelope/header/status retained; canonical server query alias is_enabled wins enabled presence and unknown kwargs are dropped before marshal.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsDeclaredDescriptorsQueryAliasesAndOwnedProof`.
- Pinned root links dictionary is ignored with links-key presence precedence, explicit null links is an iterable error with actual page proof, and short limit pages continue by raw last ID until empty. Raw max_items precedes filtering; break stops lookahead.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsSourceDictionaryLinksAndPresence`, `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsFilteredMarkerRawCapAndBreak`.
- Ordered recursive local filters use actual truthiness and CPython loaded float/bool/integer equality; early mismatch hides later bad shape and reached bad shape keeps physical proof.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsOrderedLoadedLocalEquality`.
- Proxy expressions see an opaque unread generator, with HTTP0 null/type/literal results, identity/nested generator preservation, no invented JSON and typed InvalidOption cause for syntax/type failure; falsey/nonstrings remain lazy.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsProxyExpressionsRemainOpaqueUntilConsumed`.
- Proxy formal headers/version/basepath and one-pass conflicting attrs normalization restore query/headers/version, retain original expression priority and unknown discard; bound control collisions remain HTTP0.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsProxyFormalRestorationAndBoundCollision`.
- Explicit location is owned and ignores unknown wire project/location; new SourceStatusPolicy accepts valid HTTP100..399 without changing older profiles/native statuses or triggering discovery.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsLocationBeforeLocalFilterAndStatusPolicy`.
- Captured options/Source/context guard each callback before and after; restoring callback cannot undo observed source mutation or custom cancellation and invalid preflight makes HTTP0/no proof.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsOwnedOptionsStickySourceAndCancellation`.
- 28 accepted/rejected physical Read/Close/Source restored by Close/custom cancel lanes preserve actual proof, terminal sticky causes, Close once and no retry/backoff/reauth. Native pointer401 has separately inspected value ErrOriginal and ErrReauth; no nonexistent Unwrap is assumed.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsFaultedActualRejectionsNeverFallbackOrRetry`.
- Clean native401/503/429/498 retries reuse live token and owned route/query/headers with bodyless GET, preserving successful HTTP203 proof.
  Tests: `identity/v3/domains/python_records_test.go:TestPythonDomainRecordsCleanNativeRetryUsesLiveAuthAndCapturedHeaders`.

Python/Go differences:

- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Python sends bool query values as True/False text; typed Go ListOpts send true/false, and WithListQuery reproduces the Python text.
- Python drops unknown query names; Go sends every WithListQuery key to the server.
- SDK-owned concrete record/options API preserves actual Wire/Envelope/metadata without builder interfaces; original native API is unchanged.
- Already selected Keystone v3 API required; no implicit arbitrary-v2 ensure_service_version conversion or automatic Connection current-location bridge. Explicit CloudLocation is cloned and current project retained because Domain declares no project/zone.
- Fixed origin/collection/query/headers/version bounds reject foreign route/traversal/cycles and malformed markers with actual receipt; amounts are nonnegative Go integers.
- Go iter.Seq2 can be reused and repeats captured requests rather than sharing Python generator single-use state. Proxy expression has SDK Source/option preflight before pure result.
- HTTP Link is deliberate Go interoperability for Source next[uri] versus ordinary Requests next[url] shape. JSON rootlink precedence/dict-ignore/null-error follows Source.
- Object-valued Cloud Proxy literal iterates sorted Go keys rather than Python dictionary insertion order. Opaque/nested generator remains Value/ValueObjects and is not forcibly encoded.
- Generic Connection/session GET response caching and arbitrary synchronized Resource lifecycle remain shared SDK-R1/C1 obligations; this finite Go mapping does not claim supported/full SDK parity. Connection.ListDomainRecords/SearchDomainRecords now supply cached service and explicit-or-default location.

### `python:cloud/_identity/IdentityCloudMixin/list_domains`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewDomainReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.DomainReadCloud.ListDomains`
- `github.com/JSYoo5B/go-openstacksdk/identity.DomainReadCloud.SearchDomains`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadNameOrID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadLocation`

- Cloud list_domains expands arbitrary keyword mapping into Source query/local filters; enabled aliases and nested options survive local processing and actual yielded inventory preserves response fields.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudListKeywordQueryAliasAndLocalFilters`.
- Cloud eagerly consumes Source Proxy result by Python iterable kind. Literal list/string/object results and nested generators stay HTTP0; opaque generators are not forcibly encoded, JSON exists only for complete JSON data.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudListProxyPolymorphicResultIsUnread`.
- Concrete already-v3 API factory captures Source/header state before callbacks, checks sticky selected API/source/custom cancel before and after options and prevents restore callbacks or fabricated preflight proof.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudOptionsStickySourceCancelAndHeaderSnapshot`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- SDK-owned concrete record/options API preserves actual Wire/Envelope/metadata without builder interfaces; original native API is unchanged.
- Already selected Keystone v3 API required; no implicit arbitrary-v2 ensure_service_version conversion or automatic Connection current-location bridge. Explicit CloudLocation is cloned and current project retained because Domain declares no project/zone.
- Fixed origin/collection/query/headers/version bounds reject foreign route/traversal/cycles and malformed markers with actual receipt; amounts are nonnegative Go integers.
- Go iter.Seq2 can be reused and repeats captured requests rather than sharing Python generator single-use state. Proxy expression has SDK Source/option preflight before pure result.
- HTTP Link is deliberate Go interoperability for Source next[uri] versus ordinary Requests next[url] shape. JSON rootlink precedence/dict-ignore/null-error follows Source.
- Object-valued Cloud Proxy literal iterates sorted Go keys rather than Python dictionary insertion order. Opaque/nested generator remains Value/ValueObjects and is not forcibly encoded.
- Generic Connection/session GET response caching and arbitrary synchronized Resource lifecycle remain shared SDK-R1/C1 obligations; this finite Go mapping does not claim supported/full SDK parity. Connection.ListDomainRecords/SearchDomainRecords now supply cached service and explicit-or-default location.

### `python:cloud/_identity/IdentityCloudMixin/search_domains`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewDomainReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.DomainReadCloud.ListDomains`
- `github.com/JSYoo5B/go-openstacksdk/identity.DomainReadCloud.SearchDomains`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadNameOrID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadLocation`

- name_or_id presence is preserved with a pointer: nil forwards mapping filters to list_domains; explicit empty/nonempty first materializes unfiltered domains and applies local exact/glob identifier plus mapping/JMESPath. Name/filters stay absent from server in that branch.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudSearchIdentifierPresenceSelectsSourceBranch`.
- Absent identifier kwargs expansion rejects string/false/zero/empty-string/array before HTTP; exactly None filters becomes {}. Identifier-present expression filters run on actual materialized rows and yield arbitrary scalar/null JSON with no fabricated records.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudSearchAbsentKeywordMappingVersusPresentJMESPath`.
- Absent-identifier Proxy nested generator remains an iterable ValueObjects result without invented JSON; public records remain separate from opaque/literal data.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudListProxyPolymorphicResultIsUnread`.
- Local loaded nested subset/glob selection keeps duplicates, CPython rounded float/bool/integer equality and selected record clones independent of physical inventory.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudLocalLoadedSubsetGlobAndIndependentRecord`.
- Complete HTTP materialization precedes invalid local expression, so later503 wins and physical receipt/partial Inventory survives.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudLateHTTPPrecedesLocalFilterError`.
- Explicit cloned location is applied before local selection and trusts configured current project rather than unknown wire extension.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudExplicitLocationOwnedBeforeLocalFiltering`.
- Sticky Source/context options and source header snapshot stop restore callbacks and HTTP on invalid preflight.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudOptionsStickySourceCancelAndHeaderSnapshot`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Without name_or_id Python passes filters to list_domains server-side and does not filter locally.
- SDK-owned concrete record/options API preserves actual Wire/Envelope/metadata without builder interfaces; original native API is unchanged.
- Already selected Keystone v3 API required; no implicit arbitrary-v2 ensure_service_version conversion or automatic Connection current-location bridge. Explicit CloudLocation is cloned and current project retained because Domain declares no project/zone.
- Fixed origin/collection/query/headers/version bounds reject foreign route/traversal/cycles and malformed markers with actual receipt; amounts are nonnegative Go integers.
- Go iter.Seq2 can be reused and repeats captured requests rather than sharing Python generator single-use state. Proxy expression has SDK Source/option preflight before pure result.
- HTTP Link is deliberate Go interoperability for Source next[uri] versus ordinary Requests next[url] shape. JSON rootlink precedence/dict-ignore/null-error follows Source.
- Object-valued Cloud Proxy literal iterates sorted Go keys rather than Python dictionary insertion order. Opaque/nested generator remains Value/ValueObjects and is not forcibly encoded.
- Generic Connection/session GET response caching and arbitrary synchronized Resource lifecycle remain shared SDK-R1/C1 obligations; this finite Go mapping does not claim supported/full SDK parity. Connection.ListDomainRecords/SearchDomainRecords now supply cached service and explicit-or-default location.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack


def main():
    conn = openstack.connect()
    for row in conn.identity.domains(
        is_enabled=True, description="operations", limit=100, max_items=100
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

    listed = conn.list_domains(is_enabled=True, description="operations")
    print(json.dumps([row.to_dict() for row in listed], indent=2))

    # Source distinguishes None from an explicitly empty identifier.
    server_filtered = conn.search_domains(filters={"name": "team"})
    locally_filtered = conn.search_domains(
        filters={"description": "operations"}, name_or_id="team-*"
    )
    print(json.dumps([row.to_dict() for row in server_filtered], indent=2))
    print(json.dumps([row.to_dict() for row in locally_filtered], indent=2))


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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("%s\n", data)
}

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.Domains.ListRecords(ctx,
        domains.WithRecordListQuery("is_enabled", true),
        domains.WithRecordListQuery("description", "operations"),
        domains.WithRecordListLimit(100),
        domains.WithRecordListMaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        printJSON(row.Resource.Body)
    }

    cloud := identity.NewDomainReadCloud(service.Domains)
    listed, err := cloud.ListDomains(ctx,
        identity.WithDomainReadFilters(json.RawMessage(`{"is_enabled":true,"description":"operations"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(listed.JSON)

    serverFiltered, err := cloud.SearchDomains(ctx,
        identity.WithDomainReadFilters(json.RawMessage(`{"name":"team"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(serverFiltered.Value)

    locallyFiltered, err := cloud.SearchDomains(ctx,
        identity.WithDomainReadNameOrID("team-*"),
        identity.WithDomainReadFilters(json.RawMessage(`{"description":"operations"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(locallyFiltered.Value)
}
```

## References

- [Policy](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)
- [Domain](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/domain.py)
- [Domains](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/domains.py)
- [Python Record Mutations](python-record-mutations.md)
- [Python Domain Config](python-domain-config.md)
