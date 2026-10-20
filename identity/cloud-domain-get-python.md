# Cloud Domain Get Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/get_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.DomainRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordGetLocation`

- Get sends GET domains/{id} like Python get_domain and decodes the domain envelope.
  Tests: `identity/v3/domains/python_parity_test.go:TestPythonDomainCreateGetUpdateRequests`.
- Literal member GET uses existing private Domain Source profile/converter with six nullable raw/bool/dict descriptors, preserves options missing/null/nonobject/object and constructor ID seed/explicit response null; unknown response remains Wire and independent Envelope/header/status/Clone records actual100/199/200/203/399.
  Tests: `identity/v3/domains/python_record_get_test.go:TestPythonDomainGetRecordNullableRawOptionsSeedAndActualEvidence`.
- Direct rejected403/404 never falls back; actual native response proof retains method/URL/status/body/header. Accepted malformed singular envelope keeps actual ResponseError.
  Tests: `identity/v3/domains/python_record_get_test.go:TestPythonDomainGetRecordRejectedNeverFallsBackAndBadEnvelopeKeepsProof`.
- Transport/location options snapshot maps/pointers, retain configured project despite unknown response scope, and per-callback selected source/context/cancel checks prevent restore callbacks and HTTP0 fake receipts.
  Tests: `identity/v3/domains/python_record_get_test.go:TestPythonDomainGetRecordOwnedOptionsLocationAndStickyGuard`.
- 28 physical accepted/rejected Read/Close/source restored on Close/custom cancel lanes retain immutable actual proof, Close once and zero Retry/Backoff/Reauth; native pointer401 ErrOriginal value and ErrReauth graph are separately checked. Clean401/503/429/498 uses live token and owned bodyless request.
  Tests: `identity/v3/domains/python_record_get_test.go:TestPythonDomainGetRecordPhysicalFaultsKeepActualProofAndNoRetry`, `identity/v3/domains/python_record_get_test.go:TestPythonDomainGetRecordCleanRetryPreservesLiveAuthAndFrozenRequest`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Python accepts any status below 400 and raises NotFoundException on 404; the native Get accepts only 200 and returns a Gophercloud 404 error.
- SDK-owned typed Go transport/location conveniences and raw nullable Domain record preserve actual Wire/Envelope/header/status; existing native Domain Get/Create/Update and full find_domain judgments stay unchanged.
- Go uses required literal nonblank UTF-8 string identity, immutable escaped path and selected native-v3 client. Source empty/numeric/false IDs may request domains/ collection; mutable Resource/dict/object inputs and unsafe raw URL composition are narrowed to documented Go/SDK-C1/S1 boundaries.
- Direct Source actual both-None finder raises no-ID InvalidRequest despite misleading cloud comment; Go ErrInvalidOption preflight preserves HTTP0 instead of pretending None success.
- Ignored filters are not evaluated. Source warnings.warn is represented by DeprecatedFilters presence flag; global Python warning-as-error configuration is not cloned.
- Explicit immutable location is a Go convenience and default standalone view is null; automatic Connection bridge/current scope inference is absent.
- Generic cache/synchronized arbitrary Resource lifecycle/session/Connection remain SDK-R1/C1/S1 differences; finite named get implementation is not full SDK supported. No arbitrary fetch controls or full Proxy.find_domain audit/row promotion.

### `python:cloud/_identity/IdentityCloudMixin/get_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewDomainReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.DomainReadCloud.GetDomain`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetNameOrID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetLocation`

- get_domain(domain_id=...) is a direct GET /v3/domains/<id>, and get_domain(name_or_id=...) maps to Domains FindIdentity which sends the GET then the name list.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityDomainGroupRoleHelpers`, `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- DomainID presence wins NameOrID; nonnil direct GET ignores filters completely without marshaler/parse/IO, and records only deprecated warning presence flag. Direct missing404 is an error; nil DomainID selects closed ignore-missing finder and logical absence is nil record/no error.
  Tests: `identity/domain_record_get_test.go:TestPythonCloudDomainGetIDPresenceWinsAndFiltersAreNeverConsumed`, `identity/domain_record_get_test.go:TestPythonCloudDomainGetDirectMissingVersusClosedFinderMissing`.
- Actual Source Resource.find(None)/fetch/prepare_request raises InvalidRequest before HTTP despite cloud doc comment; both identifiers omitted maps to typed preflight error/no invented proof. Empty/numeric Source route edge is deliberately narrowed to safe nonblank Go strings.
  Tests: `identity/domain_record_get_test.go:TestPythonCloudDomainGetBothOmittedMatchesActualNoIDPreflight`.
- Closed cloud-only Domain resolver uses clean400/403/404 GET-first name hint then all-page ID OR name; repeated ID is ambiguous, list199 is accepted, late503 overrides prior match, GET-only version differs from list session default, and existing list profile remains unchanged.
  Tests: `identity/v3/domains/python_record_get_test.go:TestPythonDomainResolveRecordClosedCleanFallbackAbsentDuplicatesAndLateProof`, `identity/v3/domains/python_record_get_test.go:TestPythonDomainResolveRecordGetOnlyVersionAndProfileIsolation`.
- Cloud closed fallback preserves duplicate/late native proof and ignored invalid filters, with no search/expression branch or full Proxy.find_domain mapping growth.
  Tests: `identity/domain_record_get_test.go:TestPythonCloudDomainGetClosedFallbackDuplicatesAndLateHTTPProof`.
- Captured source/header state precedes per-callback cloned options, explicit location is projected with owned configured scope and unknown response location remains Wire; sticky source/custom cancel/nil option creates no HTTP and no fabricated receipt.
  Tests: `identity/domain_record_get_test.go:TestPythonCloudDomainGetOwnOptionLocationHeadersAndSourceSnapshot`, `identity/domain_record_get_test.go:TestPythonCloudDomainGetOptionsStickySourceCancelAndNil`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- The deprecated filters argument is a no-op in Python.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- SDK-owned typed Go transport/location conveniences and raw nullable Domain record preserve actual Wire/Envelope/header/status; existing native Domain Get/Create/Update and full find_domain judgments stay unchanged.
- Go uses required literal nonblank UTF-8 string identity, immutable escaped path and selected native-v3 client. Source empty/numeric/false IDs may request domains/ collection; mutable Resource/dict/object inputs and unsafe raw URL composition are narrowed to documented Go/SDK-C1/S1 boundaries.
- Direct Source actual both-None finder raises no-ID InvalidRequest despite misleading cloud comment; Go ErrInvalidOption preflight preserves HTTP0 instead of pretending None success.
- Ignored filters are not evaluated. Source warnings.warn is represented by DeprecatedFilters presence flag; global Python warning-as-error configuration is not cloned.
- Explicit immutable location is a Go convenience and default standalone view is null; automatic Connection bridge/current scope inference is absent.
- Generic cache/synchronized arbitrary Resource lifecycle/session/Connection remain SDK-R1/C1/S1 differences; finite named get implementation is not full SDK supported. No arbitrary fetch controls or full Proxy.find_domain audit/row promotion.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--domain-id", required=True)
    parser.add_argument("--domain-name", default="")
    args = parser.parse_args()
    conn = openstack.connect()
    row = conn.identity.get_domain(args.domain_id)
    print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    row = conn.get_domain(domain_id=args.domain_id,
                          filters={"ignored": "without evaluation"})
    print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    if args.domain_name:
        found = conn.get_domain(name_or_id=args.domain_name)
        if found is not None:
            print(json.dumps(found.to_dict(), indent=2, ensure_ascii=False))


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
    "github.com/JSYoo5B/go-openstacksdk/identity"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("%s\n", data)
}

func main() {
    id := flag.String("domain-id", "", "required domain UUID")
    name := flag.String("domain-name", "", "optional domain name")
    flag.Parse()
    if *id == "" { log.Fatal("-domain-id is required") }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    row, err := service.Domains.GetRecord(ctx, *id)
    if err != nil { log.Fatal(err) }
    printJSON(row.Resource.Body)

    cloud := identity.NewDomainReadCloud(service.Domains)
    result, err := cloud.GetDomain(ctx,
        identity.WithDomainGetID(*id),
        identity.WithDomainGetFilters(json.RawMessage(`{invalid-but-ignored`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(result.Record.Resource.Body)
    fmt.Println("deprecated filters ignored:", result.DeprecatedFilters)
    if *name != "" {
        found, err := cloud.GetDomain(ctx, identity.WithDomainGetNameOrID(*name))
        if err != nil { log.Fatal(err) }
        if found.Record != nil { printJSON(found.Record.Resource.Body) }
    }
}
```

## References

- [Python Records](v3/domains/python-records.md)
- [Domain](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/domain.py)
- [list_domains](v3/domains/python-records.md)
