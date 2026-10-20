# Cloud Subnet Update Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/update_subnet`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateSubnetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetUpdateOpts`

- With attributes, FindRecord with ignore_missing false followed by UpdateWithAttributes sends the lookup GET and a PUT whose body carries name, enable_dhcp and gateway_ip null for disable_gateway_ip.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBUpdatePortAndSubnetChains`.
- SDK-owned seven formal kwargs use Source nonnull enable_dhcp versus truthiness for the remaining inputs, rename subnet_name to name, turn a truthy disable_gateway_ip into gateway_ip null, reject truthy gateway conflict before lookup, and skip service/identity validation and all HTTP when the effective body is empty. Go cancellation cannot be hidden by that no-op.
  Tests: `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateEffectiveValuesAndDirtySelection`, `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateIneffectiveAndConflictBeforeServiceLookup`, `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateOptionCancellationCannotBecomeNoOp`.
- Effective attrs resolve member/name lookup and compare ordinary raw wire values with Python equality, skip equal-value PUT and commit only changed fields to the resolved ID. Options snapshot JSON, reject unsupported kwargs before HTTP, retain lookup on native conflict without fabricated success and preserve configured provider hooks.
  Tests: `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateNameFallbackAndMissing`, `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateOwnedOptionsAndPreflight`, `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateConflictKeepsLookupAndSingleAttempt`.

Python/Go differences:

- Go returns the native Subnet model instead of a Subnet Resource.
- UpdateSubnetRecord owns argument defaults and effective-body selection. NoOp with nil Lookup corresponds to Source None; NoOp with Lookup corresponds to equal raw fields after lookup. Native Subnet is only the mutation result and is not a shared mutable Python Resource. Client capture begins after effective-argument selection; direct option-time client-internal mutation is outside the before-options guard contract.

Unresolved boundaries:

- Native mutation decoder omits unknown fields and does not merge partial/empty/malformed successful bodies into an existing Python Resource; this bridge exposes no raw mutation receipt.
- Member GET/LIST canonical-only or canonical/wire collision mapping, arbitrary Python kwargs/object/numeric type comparison, full Requests encoding/nonfinite JSON and SDK-R1/C1/S1 Resource/adapter/session/cache lifecycle remain separate.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--attrs", default="{}")
args = parser.parse_args()
attrs = json.loads(args.attrs)
if not isinstance(attrs, dict):
    parser.error("attrs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.update_subnet(args.identity, **attrs)
    print(json.dumps(None if result is None else result.to_dict(), indent=2))
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

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("identity", "", "subnet ID or exact name")
    attrs := flag.String("attrs", "{}", "cloud update kwargs as JSON")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.UpdateSubnetRecord(ctx, *identity,
        network.WithSubnetUpdateJSON(json.RawMessage(*attrs)))
    if err != nil { log.Fatal(err) }
    var value any = result.Subnet
    if result.NoOp {
        value = nil
        if result.Lookup != nil { value = result.Lookup.Resource }
    }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Install](../docs/install.md)
