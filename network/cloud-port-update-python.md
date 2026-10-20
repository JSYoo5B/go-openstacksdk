# Cloud Port Update Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/update_port`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdatePortRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithPortUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithPortUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithPortUpdateOpts`

- With attributes, FindRecord with ignore_missing false (member GET, then name list) followed by UpdateWithAttributes sends Python's lookup and PUT body, and a missing port fails before any PUT.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBUpdatePortAndSubnetChains`.
- SDK-owned cloud helper snapshots strict thirteen-key kwargs, validates before lookup, always resolves even empty kwargs, skips PUT for no attributes or Python-equal raw wire values, and PUTs only changed fields to the resolved response ID. Explicit null/false/empty collections remain distinct from omission; raw string boolean values are not replaced by descriptor getter views.
  Tests: `network/cloud_port_update_test.go:TestCloudPortUpdateLookupOnlyAndDirtySelection`, `network/cloud_port_update_test.go:TestCloudPortUpdateNameFallbackAndMissing`.
- Options own caller JSON, unsupported canonical aliases/revision/extensions and malformed inputs fail before HTTP; canceled context or selected-service mutation cannot send lookup/PUT. A default-client conflict retains a separate lookup record and makes one PUT attempt without fabricated native success. No new mutation retry policy is added; configured provider retry/reauthentication hooks remain in force.
  Tests: `network/cloud_port_update_test.go:TestCloudPortUpdateOwnedOptionsAndPreflight`, `network/cloud_port_update_test.go:TestCloudPortUpdateConflictKeepsLookupAndSingleAttempt`.

Python/Go differences:

- Python raises SDKException for a missing port; Go FindRecord returns its own not-found error.
- Go returns the native Port model, which omits extension attributes.
- Cloud UpdatePortRecord returns a separate Lookup PortRecord and native PUT Port; NoOp has Lookup and no fabricated Port. This callable JSON profile owns defaults/options and does not return the same Python mutable Resource instance.

Unresolved boundaries:

- Native PUT decoder omits unknown extension fields and does not preserve/merge the Python Resource on partial, empty or malformed success JSON; no raw mutation receipt is exposed by this bridge.
- Ordinary wire-key lookup response dirty comparison is covered; GET/LIST canonical-only attributes and canonical/wire key collision precedence and original arbitrary Python kwargs numeric/object types remain outside this bounded JSON profile. Full Resource lifecycle, adapter/session/cache policy and SDK-R1/C1/S1 remain separate. Native configured retry/auth hooks keep existing behavior; this helper does not establish universal mutation no-replay guarantees.

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
    result = conn.update_port(args.identity, **attrs)
    print(json.dumps(result.to_dict(), indent=2))
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
    identity := flag.String("identity", "", "port ID or exact name")
    attrs := flag.String("attrs", "{}", "cloud update kwargs as JSON")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.UpdatePortRecord(ctx, *identity,
        network.WithPortUpdateJSON(json.RawMessage(*attrs)))
    if err != nil { log.Fatal(err) }
    var value any = result.Port
    if result.NoOp { value = result.Lookup.Resource }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Install](../docs/install.md)
