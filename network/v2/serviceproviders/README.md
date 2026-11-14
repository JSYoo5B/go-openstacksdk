# Serviceproviders

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

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
        options = append(options, request.Query("service_type", *serviceType))
    }
    switch *defaultMode {
    case "any":
    case "yes", "no":
        options = append(options, request.Query("is_default", *defaultMode == "yes"))
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
