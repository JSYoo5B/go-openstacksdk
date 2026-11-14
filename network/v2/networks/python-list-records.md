# Python List Records

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
parser.add_argument("--name")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--jmes")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    query = {}
    if args.name is not None:
        query["name"] = args.name
    if args.max_items:
        query["max_items"] = args.max_items
    if args.jmes is not None:
        query["jmespath_filters"] = args.jmes
    result = conn.network.networks(**query)
    if isinstance(result, types.GeneratorType):
        for value in result:
            print(json.dumps(value.to_dict(), indent=2))
    else:
        print(result)
```

```go
package main

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    name := flag.String("name", "", "server-side network name filter")
    maximum := flag.Int("max-items", 0, "raw row cap; zero is unbounded")
    expression := flag.String("jmes", "", "deprecated Python Proxy expression")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []networks.RecordListOption{
        request.MaxItems(*maximum),
        networks.WithRecordListQuery("jmespath_filters", *expression),
    }
    if *name != "" {
        options = append(options, networks.WithRecordListQuery("name", *name))
    }
    result, err := network.Networks.ProxyListRecords(ctx, options...)
    if err != nil { log.Fatal(err) }
    if result.JSON != nil {
        fmt.Println(string(result.JSON))
        return
    }
    generator, ok := result.Value.(*networks.RecordListGenerator)
    if !ok {
        // Expressions such as to_array(@) retain nested generator values.
        // They have no JSON representation and do not issue a request here.
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

- [Install](../../../docs/install.md)
- [Network](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/network.py#L196-L265)
- [Proxy.networks](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3258)
- [Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py#L17)
- [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155)
