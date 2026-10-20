# Python Available

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_available_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.FindAvailableIPRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithAvailableIPOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithAvailableIPHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithAvailableIPMicroversion`

- List sends GET floatingips without a query, follows floatingips_links lazily and stops fetching when the caller breaks at the first row with an empty PortID; no row yields nil.
  Tests: `network/v2/extensions/layer3/floatingips/python_parity_core_test.go:TestPythonParityCoreFloatingIPFindAvailableStopsAtFirstFreeRow`.
- FindAvailableIPRecord owns unfiltered Source-profile floating IP list iteration, returns the first falsey untyped port_id including missing/null/empty string/false/zero/underflow/empty collections, preserves raw extension/resource views, stops before later malformed row construction and continuation parsing, and returns nil on exhausted inventory.
  Tests: `network/v2/extensions/layer3/floatingips/python_available_test.go:TestPythonAvailableIPFalseyPortIDAndRawExtensionProjection`, `network/v2/extensions/layer3/floatingips/python_available_test.go:TestPythonAvailableIPEarlyStopPagingAndExhaustion`.
- Owned transport-only options do not add server filters, snapshot caller headers, and guard cancellation/selected source mutation before HTTP. Accepted status203 retains actual page evidence; malformed consumed pages/rows and rejected500 yield no fabricated free IP.
  Tests: `network/v2/extensions/layer3/floatingips/python_available_test.go:TestPythonAvailableIPAcceptedStatusAndFailures`, `network/v2/extensions/layer3/floatingips/python_available_test.go:TestPythonAvailableIPOwnedTransportOptionsAndCancellation`.

Python/Go differences:

- FindAvailableIPRecord is the SDK-owned no-query first-falsey-port helper; native typed List remains available as a separate ABI. Returned immutable RawResource views and page receipt do not return a Python mutable Resource object.
- network.FloatingIPs.Available remains a different workflow that filters by network/project and may allocate; this helper only observes existing inventory.
- Explicit Go profile boundaries include transport-only headers/microversion extras absent from Source formals, guarded continuation/route policy, full Requests charset/nonfinite JSON/adapter-session-cache and arbitrary Python object behavior. These are not a remaining absence of the named helper.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    value = conn.network.find_available_ip()
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
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.API.FloatingIPs.FindAvailableIPRecord(ctx)
    if err != nil { log.Fatal(err) }
    var value any
    if result != nil { value = result.Resource }
    raw, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}
```

## References

- [Install](../../../../../docs/install.md)
