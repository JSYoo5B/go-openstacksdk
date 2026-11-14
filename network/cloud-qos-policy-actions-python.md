# Cloud QoS Policy Actions Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("get", "delete"), default="get")
parser.add_argument("--identity", required=True)
parser.add_argument("--filters", default='{"skip_cache":true,"error_message":"QoS policy lookup failed"}')
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "delete":
        print(json.dumps(conn.delete_qos_policy(args.identity)))
    else:
        value = conn.get_qos_policy(args.identity, filters=json.loads(args.filters))
        print(json.dumps(None if value is None else value.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "get", "get or delete")
    identity := flag.String("identity", "", "literal ID or exact name")
    filters := flag.String("filters", `{"skip_cache":true,"error_message":"QoS policy lookup failed"}`, "get-only JSON kwargs")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    switch *mode {
    case "get":
        result, err := service.GetQoSPolicyRecord(
            ctx, *identity, network.WithQoSPolicyFilters(json.RawMessage(*filters)),
        )
        if err != nil { log.Fatal(err) }
        if result.Policy == nil { printJSON(nil); return }
        printJSON(result.Policy.Resource)
    case "delete":
        result, err := service.DeleteQoSPolicyRecord(ctx, *identity)
        if err != nil { log.Fatal(err) }
        printJSON(result.Deleted)
    default:
        log.Fatal("mode must be get or delete")
    }
}
```

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L42-L165)
- [Install](../docs/install.md)
- [Cloud get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L220-L250)
- [Cloud delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1512-L1536)
- [Resource.find/fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Proxy delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L670-L713)
