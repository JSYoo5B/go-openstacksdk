# Python Onboard Network Subnets

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--pool", required=True)
parser.add_argument("--network", required=True)
args = parser.parse_args()
with openstack.connect() as conn:
    result = conn.network.onboard_network_subnets(args.pool, args.network)
    print(json.dumps(result, indent=2))
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    sdk "github.com/JSYoo5B/go-openstacksdk"
)

func main() {
    pool := flag.String("pool", "", "subnet pool ID")
    network := flag.String("network", "", "network ID")
    flag.Parse()
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.API.SubnetPools.OnboardNetworkSubnetsRecord(ctx, *pool, *network)
    if err != nil { log.Fatal(err) }
    fmt.Printf("HTTP %d\n%s\n", result.StatusCode, result.Value)
}
```

## References

- [Proxy.onboard_network_subnets](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [SubnetPool.subnet_onboard](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet_pool.py)
