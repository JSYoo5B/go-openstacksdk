# Cloud Endpoints Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_identity/IdentityCloudMixin/search_endpoints`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewEndpointReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.EndpointReadCloud.SearchEndpoints`

- search_endpoints lists with the same GET /v3/endpoints as list_endpoints before filtering locally.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityEndpointHelpers`.
- search_endpoints materializes unfiltered selected v3 Endpoint Source records across all pages before id/name exact-or-glob and ordered local dict/loaded float/bool equality or arbitrary materialized-list JMESPath; falsey filters skip mapping and invalid local filters run after pages. Actual rows preserve Wire/page/header/status, independent selected copies/Inventory; no server filter hint. Library-owned immutable options/header/client/provider/route/version/location source capture and chained REST operation guard preserve native bodyless GET and every physical error/cause without fallback/retry/auth hiding.
  Tests: `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudSearchGlobLocalFiltersAndActualRecords`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudSearchFalseyFiltersAndPythonIdentifiers`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudLocalFilterErrorsFollowEveryPageAndKeepInventory`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudNestedEmptyFilterUsesActualTruthiness`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudJMESUsesMaterializedListAndKeepsNoInventedReceipt`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetNoneBypassesDeprecatedFalseySearch`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetCleanFallbackExactIDNameDuplicatesAndMissing`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetSeedAliasesWireAndExplicitLocation`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetFilteredArbitraryTruthLenAndIndex`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudOwnedHeadersOptionsAndConcurrentReuse`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudPreflightSourceCancelAndOuterGuard`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudLateNativeErrorWinsOverInvalidLocalFilter`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudPhysicalFaultsNeverFallbackRetryOrInventRows`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Concrete Go typed v3 factory/JSON operands and safe owned route/context/header guards constrain arbitrary Python proxy/session/object behavior. Runtime mutable proxy API-version metadata is shared SDK-S1; Resource/Connection mutable lifecycle remains SDK-R1 and optional HTTP cache SDK-C1.
- Source None vs present falsey filters branch is explicit in EndpointReadOpts and DeprecatedFilters; Go exposes branch metadata rather than executing Python warning categories. Source get_endpoint does not use _get_entity and no Resource/dict identity passthrough is invented.
- Separate RawMessage Value, real EndpointRecord and Inventory preserve arbitrary JMESPath output without fabricating HTTP association. Search/deprecated get retains physically read rows on late/local failure; default private find exposes returned record receipt, not an invented complete inventory of its rejected/member/fallback observations.
- SDK-owned explicit CloudLocation uses captured/current scope; direct ServiceClient default is null. Stock Keystone27.0.0 get/list policy is admin_required or role:reader and system_scope:all, with system/project scope declaration; project reader alone and actual deployed cloud acceptance are not claimed.

### `python:cloud/_identity/IdentityCloudMixin/get_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewEndpointReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.EndpointReadCloud.GetEndpoint`

- get_endpoint(id) with the default filters=None sends GET /v3/endpoints/<id> and, after 404, the unfiltered GET /v3/endpoints because Endpoint declares no name query.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityEndpointHelpers`.
- get_endpoint omitted/literal-null filters executes private Source GET-first Resource.find, only clean400403404 fallback to complete exact id-or-name list with default missing nil and duplicate ErrAmbiguous. Every other filter value including falsey enters deprecated materialized search, then loaded truthiness/len/integer index0; arbitrary values and actual records have independent result carriers and no invented receipt. Library-owned immutable options/header/client/provider/route/version/location source capture and chained REST operation guard preserve native bodyless GET and every physical error/cause without fallback/retry/auth hiding.
  Tests: `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudSearchGlobLocalFiltersAndActualRecords`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudSearchFalseyFiltersAndPythonIdentifiers`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudLocalFilterErrorsFollowEveryPageAndKeepInventory`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudNestedEmptyFilterUsesActualTruthiness`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudJMESUsesMaterializedListAndKeepsNoInventedReceipt`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetNoneBypassesDeprecatedFalseySearch`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetCleanFallbackExactIDNameDuplicatesAndMissing`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetSeedAliasesWireAndExplicitLocation`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetFilteredArbitraryTruthLenAndIndex`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudOwnedHeadersOptionsAndConcurrentReuse`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudPreflightSourceCancelAndOuterGuard`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudLateNativeErrorWinsOverInvalidLocalFilter`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudPhysicalFaultsNeverFallbackRetryOrInventRows`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.find does the ID GET and list fallback in one call and also falls back after GET 400 or 403; Go chains Find(resource.ID(x), resource.WithIgnoreMissing()) and Find(resource.Name(x), resource.WithIgnoreMissing()), which falls through only after a 404.
- The Python list match compares the ID or the name and raises DuplicateResource; the Go name lookup compares only the name and returns resource.ErrAmbiguous.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Concrete Go typed v3 factory/JSON operands and safe owned route/context/header guards constrain arbitrary Python proxy/session/object behavior. Runtime mutable proxy API-version metadata is shared SDK-S1; Resource/Connection mutable lifecycle remains SDK-R1 and optional HTTP cache SDK-C1.
- Source None vs present falsey filters branch is explicit in EndpointReadOpts and DeprecatedFilters; Go exposes branch metadata rather than executing Python warning categories. Source get_endpoint does not use _get_entity and no Resource/dict identity passthrough is invented.
- Separate RawMessage Value, real EndpointRecord and Inventory preserve arbitrary JMESPath output without fabricating HTTP association. Search/deprecated get retains physically read rows on late/local failure; default private find exposes returned record receipt, not an invented complete inventory of its rejected/member/fallback observations.
- SDK-owned explicit CloudLocation uses captured/current scope; direct ServiceClient default is null. Stock Keystone27.0.0 get/list policy is admin_required or role:reader and system_scope:all, with system/project scope declaration; project reader alone and actual deployed cloud acceptance are not claimed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def jsonable(value):
    if hasattr(value, "to_dict"):
        return value.to_dict()
    if isinstance(value, list):
        return [jsonable(item) for item in value]
    if isinstance(value, dict):
        return {key: jsonable(item) for key, item in value.items()}
    return value


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="system-reader")
    parser.add_argument("--id", default="")
    parser.add_argument("--filters", default="")
    parser.add_argument("--get", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    filters = json.loads(args.filters) if args.filters else None
    if args.get:
        value = conn.get_endpoint(args.id, filters=filters)
    else:
        value = conn.search_endpoints(id=args.id, filters=filters)
    print(json.dumps(jsonable(value), indent=2, ensure_ascii=False))


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
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
)

func main() {
	cloud := flag.String("cloud", "system-reader", "clouds.yaml cloud")
	id := flag.String("id", "", "literal ID or search glob")
	filters := flag.String("filters", "", "JSON dict or JSON encoded JMESPath string")
	get := flag.Bool("get", false, "get one endpoint")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	api, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	reader := identity.NewEndpointReadCloud(api.Endpoints)
	var options []identity.EndpointReadOption
	if *filters != "" {
		options = append(options, identity.WithEndpointReadFilters(json.RawMessage(*filters)))
	}
	var value json.RawMessage
	if *get {
		result, err := reader.GetEndpoint(ctx, *id, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	} else {
		result, err := reader.SearchEndpoints(ctx, *id, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [`IdentityCloudMixin.search_endpoints/get_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py)
- [`_utils._filter_list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py)
- [Python Records](v3/endpoints/python-records.md)
- [Endpoint](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/endpoint.py)
- [Base](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/base.py)
