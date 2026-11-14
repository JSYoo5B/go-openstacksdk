# Cloud QoS Policies Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("list", "search"), default="list")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="null")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    filters = json.loads(args.filters)
    if args.mode == "search":
        values = conn.search_qos_policies(name_or_id=args.name, filters=filters)
    else:
        values = conn.list_qos_policies(filters=filters)
    for value in values:
        if hasattr(value, "to_dict"):
            value = value.to_dict()
        print(json.dumps(value, indent=2))
```

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "list", "list or search")
    name := flag.String("name", "", "literal server name query")
    filters := flag.String("filters", "null", "JSON filters")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    options := []network.QoSPolicyReadOption{
        network.WithQoSPolicyFilters(json.RawMessage(*filters)),
    }
    var result *network.QoSPolicyRecords
    switch *mode {
    case "list":
        result, err = service.ListQoSPolicyRecords(ctx, options...)
    case "search":
        result, err = service.SearchQoSPolicyRecords(ctx, *name, options...)
    default:
        log.Fatal("mode must be list or search")
    }
    if err != nil {
        if result != nil {
            log.Printf("completed extension records: %d", len(result.Extensions))
        }
        log.Fatal(err)
    }
    if result.JSON == nil {
        fmt.Printf("opaque iterable result: %T\n", result.Value)
        return
    }
    var readable any
    decoder := json.NewDecoder(bytes.NewReader(result.JSON))
    decoder.UseNumber()
    if err := decoder.Decode(&readable); err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(readable, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L42-L71)
- [Install](../docs/install.md)
- [Cloud list/search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L253-L348)
- [extension gate](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L43-L50)
- [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866-L916)
- [QoSPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py)
