# Serviceproviders

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/service_providers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListConflictingAttrs`

- Library-owned service provider list uses the common neutronfind engine for GET /service-providers and full Source defaults, independent actual Resource/Wire/whole-page/Header/status, raw unknown extensions and sparse rows without inventing provider IDs. Source Resource has no Neutron revision/tag/Header mixin. is_default aliases default, bool coercion follows CPython and omission/null remains null.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderListOwnedDefaultsAliasesAndNoNetworkMixins`.
- All six canonical query attributes including inherited limit/marker, literal fields, service_type/name and canonical is_default/default presence are server queries; id is a local Body condition and unknown caller kwargs are discarded before marshaling. Canonical null/false precedence and server-only name/default behavior remain explicit.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderListFullQueriesDefaultPresenceAndLocalID`.
- Actual Proxy JMES evaluation receives an opaque unconsumed generator. Projection null/literal/identity results and runtime failures occur with zero HTTP, nested generator identity remains Value with no fake JSON/evidence, and real source HTTP starts only when the generator is consumed.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderProxyJMESPreservesOpaqueGeneratorBeforeHTTP`.
- Shared owned Source kwargs consume headers/version/base_path, one-pass conflicting attrs and selected collection continuation, preserving original client facts. Accepted malformed envelope/null-row/constructor failures retain actual status/body/Header. Endpoint route is service-relative and request source/token remain owned.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderListSourceControlsPagingAndActualErrors`.
- Raw cap counts excluded rows and stops before out-of-range constructor or invalid continuation; option callback maps remain owned through nil reset and retained callback mutation, without requiring user builders or transport classes.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderListCapsAndOwnedCallbackMaps`.
- Real Gophercloud testhelper HTTP transport response read/Close faults never call RetryFunc or ReauthFunc and preserve one HTTP/one Close plus all physical causes. Accepted faults retain actual ResponseError, rejected native401 pointer keeps ErrOriginal receipt and ErrReauth failures; native503 actual status/body/Header is preserved.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderListPhysicalProofNeverReauthsOrRetries`.

Python/Go differences:

- Gophercloud2.15 has no native binding for this resource; concrete SDK-owned API/profile/options reuse common reader, projection, paging and JMES engines. Source readonly capabilities are preserved and provider names do not become IDs.
- Immutable eager JSON view differs from mutable Python Resource/lazy descriptors/cache/current_location; location is an owned null view and Wire remains original. None-valued headers, arbitrary adapters/subclasses/runtime non-JSON objects and URI percent-format are outside the concrete JSON option boundary, separately tracked through shared SDK foundations.
- Same-origin selected collection/filter/cursor guards bound broad Source routing. Empty base_path falls back to the default in Go whereas Source empty URI returns without HTTP. Missing stock provider IDs do not become invented marker IDs; unsupported limit/marker continuation fails with actual page evidence. Explicit single-page is available.
- Completed200..399 are supported, informational100..199/replacement UTF8 are explicit differences. Nativephysical/source/cancel causes and clean retry/token renewal retain actual metadata and method ownership rather than fabricating successful Resource data.
- Actual Source Proxy generator JMES results are distinct from cloud filter arrays. Arbitrary JSON or nested opaque generators are returned by ProxyListRecords.Value/JSON; typed ListRecords/AllRecords remain resource iterators and redirect truthy expressions to that API.
- Neutron27 stock get_service_provider is project role:reader without owner project_id because providers are not project resources. SDK does not create drivers, elevate privileges or prove actual deployment backend support; fixture/race/source/main evidence has that explicit scope.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import types
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--service-type")
parser.add_argument("--default", choices=("any", "yes", "no"), default="any")
parser.add_argument("--jmes")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    query = {}
    if args.service_type is not None:
        query["service_type"] = args.service_type
    if args.default != "any":
        query["is_default"] = args.default == "yes"
    if args.jmes is not None:
        query["jmespath_filters"] = args.jmes
    result = conn.network.service_providers(**query)
    if isinstance(result, types.GeneratorType):
        for value in result:
            print(json.dumps(value.to_dict(), indent=2))
    else:
        print(result)
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    serviceType := flag.String("service-type", "", "server service type")
    defaultMode := flag.String("default", "any", "any, yes or no")
    expression := flag.String("jmes", "", "deprecated Python Proxy expression")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []serviceproviders.RecordListOption{
        serviceproviders.WithRecordListJMESPath(*expression),
    }
    if *serviceType != "" {
        options = append(options, serviceproviders.WithRecordListQuery("service_type", *serviceType))
    }
    switch *defaultMode {
    case "any":
    case "yes", "no":
        options = append(options, serviceproviders.WithRecordListQuery("is_default", *defaultMode == "yes"))
    default:
        log.Fatal("default must be any, yes or no")
    }
    result, err := network.ServiceProviders.ProxyListRecords(ctx, options...)
    if err != nil { log.Fatal(err) }
    if result.JSON != nil {
        fmt.Println(string(result.JSON))
        return
    }
    generator, ok := result.Value.(*serviceproviders.RecordListGenerator)
    if !ok {
        fmt.Printf("opaque expression result: %T\n", result.Value)
        return
    }
    for value, err := range generator.Records {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```

## References

- [Service Type](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/service_type.py#L22-L41)
- [Install](../../../docs/install.md)
- [ServiceProvider](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/service_provider.py#L16)
- [Proxy.service_providers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7274)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
