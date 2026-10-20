# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/endpoints`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.ProxyListRecords`

- List with ListOpts sends GET endpoints?interface=&region_id=&service_id= like Python endpoints(interface=..., region_id=..., service_id=...), follows links.next, and the endpoints collection maps those query keys while rejecting limit and marker before HTTP.
  Tests: `identity/v3/endpoints/python_parity_test.go:TestPythonEndpointListQueryAndPaging`.
- Concrete Source Endpoint MRO owns eight Body descriptors, null defaults, bool enabled alias order, untyped links, unknown discard, separate Wire/page/header/status; query limit/marker/interface/region_id/service_id vs declared local Body filters. Python-loaded equality and raw cap precede local filtering. Source dict links ignored, root/plural/top/HTTP link precedence and limit-driven marker continue to empty page. Lazy opaque generator JMESPath preserves identity and zero HTTP until consumption. SDK-owned callback/header/microversion/route snapshots and source/context guards retain actual physical read/Close/rejection evidence without hiding faults in native auth/retry/backoff; clean retries and explicit Source status acceptance preserve old native API.
  Tests: `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsSourceMRODefaultsUntypedLinksAndActualEvidence`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsOrderedEnabledAliasesAndPythonBool`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsQueriesLocalBodyFiltersAndUnknownKwargs`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsSourceDictionaryLinksAndMarkerFallback`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsSourceLinkPrecedenceAndMalformedPageReceipt`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsRawCapLazyBreakAndCancellation`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsMalformedAndLaterErrorDiscardPartialCollection`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsOpaqueProxyJMESUsesActualGenerator`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsOwnOptionsFormalControlsAndIdentityVersion`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsSourceGuardBeforeOptionsAndAfterGeneratorCapture`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsPhysicalFaultsRetainActualProofWithoutHooks`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsCleanNativeRetryKeepsScopeQueryAndHeaders`, `identity/v3/endpoints/python_records_contracts_test.go:TestEndpointRecordsSourceStatusOptInPreservesNativeBindings`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Go concrete Source record splits defaults/projection from actual Wire/page/header/status; direct ServiceClient has SDK-owned location null. Optional HTTP cache and mutable Python Resource/Connection lifecycle remain shared SDK-R1/C1; no actual cloud acceptance claim.
- Go safe owned route/origin/cursor/auth guards reject Source arbitrary session/origin movement and invalid marker with actual page receipt. Generic options expose Source formal kwargs but do not permit receiver/session/auth replacement; Go JSON cannot construct arbitrary Python object operands.
- Opaque generator and serializable JSON are separate result carriers. Existing native API remains separate and unchanged; this named list is not full SDK parity.
- Verified Keystone27.0.0 list_endpoints is admin_required or role:reader and system_scope:all with system/project scope declarations. This is a system-reader user read, not general project-reader permission; deployments may override policy.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="system-reader")
    parser.add_argument("--service-id", default="")
    parser.add_argument("--region-id", default="")
    args = parser.parse_args()
    query = {"is_enabled": True, "max_items": 100}
    if args.service_id:
        query["service_id"] = args.service_id
    if args.region_id:
        query["region_id"] = args.region_id
    conn = openstack.connect(cloud=args.cloud)
    for row in conn.identity.endpoints(**query):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))


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
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
)

func main() {
	cloud := flag.String("cloud", "system-reader", "clouds.yaml cloud")
	service := flag.String("service-id", "", "optional service UUID")
	region := flag.String("region-id", "", "optional region ID")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	identity, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []endpoints.RecordListOption{
		endpoints.WithRecordListQuery("is_enabled", true),
		endpoints.WithRecordListMaxItems(100),
	}
	if *service != "" {
		options = append(options, endpoints.WithRecordListQuery("service_id", *service))
	}
	if *region != "" {
		options = append(options, endpoints.WithRecordListQuery("region_id", *region))
	}
	for row, err := range identity.Endpoints.ListRecords(ctx, options...) {
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(row.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	}
}
```

## References

- [`Proxy.endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py)
- [`Endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/endpoint.py)
- [Endpoint](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/endpoint.py)
- [Base](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/base.py)
