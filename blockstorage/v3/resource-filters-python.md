# Resource Filters Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/resource_filters`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.API.ListResult`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListResource`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListControl`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListProxyJMESValue`

- ExactResourceFilter declared resource str and filters list descriptor plus inherited untyped id/name; scalar-list wrapping, null defaults, complete literal unknown fields and independent row/page/header/status evidence. Source constructor collision and malformed accepted envelope retain whole page proof.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersDeclaredProjectionOwnsLiteralPageEvidence`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersDefaultsNullableListAndPythonStrings`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersBadEnvelopeAndConstructorCollisionKeepAcceptedPage`.
- Source query mapping has include_pagination_defaultsFalse: resource onlyserverquery, repeated Requests/null expansion, localfilters/id/name via common Cinder MatchLocal, unknownkwargs and callerlimit/marker discarded without invokingmarshalers. Known query JSON and nil headers/callback outputs captured atowned boundaries; raw controls reject beforeanymarshal/HTTP.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersSourceQueryExpansionLocalPredicatesAndUnknownDiscard`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersControlsBeforeMarshalerAndOptionSnapshots`.
- Class ceiling3.33 uses commondiscovery absentselected default, highselected retained and explicitempty override skipsdiscovery/omitsversionheaders onallpages. Low selected3.32 retainsrealstock404 rather thanfabricatedlocalUnsupported.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersDiscoveryCeilingAndExplicitVersionOmission`.
- Lazy rawrow cap/limit hint operates beforelocalfilters, stops beforeunconsumedmalformedrows/next; iterator reuse/break andSinglePage retain defaults. Completefixedcollection pager preservesresourcequery and blocksforeignorigin/cycles; SourceemptyEOF/floatunderflownext falsey behavior is retained; All atomicallydropspartialrowsonfailure.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersLocalCapStopsBeforeInvalidRowsAndNext`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersPaginationPreservesFiltersAndBlocksForeignOrCycle`.
- Source generator-based JMESPath is implemented by ProxyListRecords and common OpaqueIdentity. Separate ListResult explicitly improves array-row evaluation by collection before arbitrary JSON transformation; scalar/string/array/null Value has no invented HTTP proof and Records retain actual receipts. Empty expression is no-op; typed row API routes nonempty expression to polymorphic/eager APIs and eager expression failure clears result after observed pages.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersListResultSupportsArbitraryJMESPathWithoutFakeProof`.
- Actual200/203/401/404/429/503 physicalRead/Close/mixed faults retainbody/header/status/cause; native401pointeroriginal/publicErrReauth remain, zeroRetryBackoffReauthcallbacks andoneHTTP/oneClose. Cleanrenewal/retry keepquery/protocol; read-boundarySource mutation stayssticky afterClose restoration and customcancel terminates. Body/version/foreign/auth/protectedomitretrymutations preservefirstrejection.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersPhysicalFaults401FieldsAndHooksStayTerminal`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersCleanRenewalAndRetryPreserveQueryAndProtocol`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersReadGuardAndCancellationSurviveCloseRestore`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersRetryCannotChangeBodyVersionOrHeaderAbsence`.
- Pinned Proxy JMESPath evaluation sees an opaque unconsumed generator before any version discovery/member HTTP. Projection/type return null, checked array functions fail, literal/equality preserve Source values, @ and nested array/object keep pointer identity and do not serialize as JSON. Falsey or non-string JSON filter values retain the lazy generator. Raw jmespath_filters is consumed privately and captured source remains guarded when generator starts discovery/list.
  Tests: `blockstorage/v3/resourcefilters/python_proxy_contracts_test.go:TestPythonCinderResourceFiltersProxyExpressionsObserveOpaqueGeneratorBeforeHTTP`, `blockstorage/v3/resourcefilters/python_proxy_contracts_test.go:TestPythonCinderResourceFiltersProxyPreservesFalseyNonStringAndNestedGeneratorIdentity`, `blockstorage/v3/resourcefilters/python_proxy_contracts_test.go:TestPythonCinderResourceFiltersProxyRawQueryJMESAndLazyDiscoveryKeepCapturedSource`.
- Common MatchLocalLoaded opt-in uses CPython decimal/exponent binary64 values, arbitrary integers and bool subclass equality through ordered nested Resource dictionary filters. Underflow is numeric zero and falsey, float overflow compares as infinity while adjacent big integer precision is retained, recursive array equality and dictionary subset use the same policy. Existing bindings exact-decimal policy stays unchanged.
  Tests: `blockstorage/v3/resourcefilters/python_proxy_contracts_test.go:TestPythonCinderResourceFiltersLoadedLocalFloatsRetainIntegerPrecisionAndNestedSubsets`.

Python/Go differences:

- Standalone computed locationnull is boundedSDK-R1; literal location andunknownfields retainedinWire, PythonConnection dynamicResource dirty/cache/session remainsprojectparitywork.
- Go ProxyListRecords preserves Source passing an opaque unconsumed generator into JMESPath; ListResult separately improves this by collecting declared JSON rows before evaluation. Python comparison main explicitly selects the eager list/to_dict repair. Source scalar/null/opaque nested pointer behavior is covered without service HTTP; arbitrary eager output retains actual Records separately.
- Ownedpager fixesdict links.next andRequestsHTTP Link uri-key mismatch, enforces sameorigin/fixedcollection/preservedfilters/cycles andonevalidwiremarker. Sourceunsafecontinuation/routecontrols arenotreplayed; malformedcontinuation retainsactualwholepageproof.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import jmespath
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.33")
    parser.add_argument("--resource", default=None)
    parser.add_argument("--single-page", action="store_true")
    parser.add_argument("--expression", default="")
    args = parser.parse_args()
    conn = openstack.connect(
        cloud=args.cloud,
        block_storage_default_microversion=args.microversion,
    )
    rows = list(conn.block_storage.resource_filters(
        resource=args.resource,
        paginated=not args.single_page,
    ))
    data = [row.to_dict() for row in rows]
    # Explicit repair: the pinned Proxy passes a generator to JMESPath.
    value = jmespath.search(args.expression, data) if args.expression else data
    print(json.dumps(value, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.33", "selected Cinder version")
    resourceName := flag.String("resource", "", "resource name; empty omits query")
    single := flag.Bool("single-page", false, "stop after the first page")
    expression := flag.String("expression", "", "optional JMESPath result")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    filters := resourcefilters.New(service.RawClient())
    options := []resourcefilters.RecordListOption{
        resourcefilters.WithRecordListControl(resource.ListControl{SinglePage: *single}),
        resourcefilters.WithRecordListJMESPath(*expression),
    }
    if *resourceName != "" {
        options = append(options, resourcefilters.WithRecordListResource(*resourceName))
    }
    result, err := filters.ListResult(ctx, options...)
    if err != nil { log.Fatal(err) }
    fmt.Println(string(result.Value))
}
```

## References

- [resource_filters Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3620-L3628)
- [ResourceFilter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/resource_filter.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2359)
- [Install](../../docs/install.md)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#list-resource-filters)
- [ResourceFiltersController.index](https://github.com/openstack/cinder/blob/master/cinder/api/v3/resource_filters.py)
- [Python Contracts Test](resourcefilters/python_contracts_test.go)
- [Python Proxy Contracts Test](resourcefilters/python_proxy_contracts_test.go)
