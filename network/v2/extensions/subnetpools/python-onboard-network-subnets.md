# Python Onboard Network Subnets

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/onboard_network_subnets`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.OnboardNetworkSubnetsRecord`

- SDK-owned one-PUT onboard route sends unwrapped network_id without preliminary GETs, accepts arbitrary JSON under Source status policy, snapshots options and guards physical errors/source identity.
  Tests: `network/v2/extensions/subnetpools/python_onboard_network_test.go:TestPythonOnboardNetworkSubnetsUnwrappedBodyAndArbitraryJSON`, `network/v2/extensions/subnetpools/python_onboard_network_test.go:TestPythonOnboardNetworkSubnetsHTTPAndDecodeFailures`, `network/v2/extensions/subnetpools/python_onboard_network_test.go:TestPythonOnboardNetworkSubnetsOptionsOwnershipAndGuard`, `network/v2/extensions/subnetpools/python_onboard_network_test.go:TestPythonOnboardNetworkSubnetsPhysicalFaultsStopCallbacks`, `network/v2/extensions/subnetpools/python_onboard_network_test.go:TestPythonOnboardNetworkSubnetsPreflight`.

Python/Go differences:

- Go accepts literal string pool/network IDs, escapes the pool route and keeps independent JSON/HTTP proof; arbitrary Python Resource inputs and mutable cache/runtime are not modeled.
- SDK-R1/C1/S1, real-cloud compatibility and release validation remain; local tests and source review do not establish full SDK parity.

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
