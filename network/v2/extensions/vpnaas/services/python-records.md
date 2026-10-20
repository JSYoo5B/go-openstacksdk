# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/vpn_services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithRecordListOpts`

- List sends GET vpn/vpnservices with WithListOptions and WithListQuery values as query and follows the vpnservices_links next href.
  Tests: `network/v2/extensions/vpnaas/services/python_parity_test.go:TestPythonVPNServiceListQueryAndLinks`.
- SDK-owned record listing owns all-page iteration and classifies declared QueryParameters/aliases as server query versus declared Body-only local descriptor filters. Missing/null, project alias, raw unknown fields and class-specific boolean/list projection are separate from original page evidence.
  Tests: `network/v2/extensions/vpnaas/services/python_record_contracts_test.go:TestVPNServiceRecordsAllPagesServerQueryAndLocalFilter`, `network/v2/extensions/vpnaas/services/python_record_contracts_test.go:TestVPNServiceRecordsProjectionDefaultsAndUnknownFields`.
- ProxyListRecords retains actual opaque Resource.list generator semantics: projection/type may return null without HTTP and @ returns a generator. Owned options discard unknown keys before serialization, guard cancellation/invalid controls before HTTP, and propagate failed page reads without fabricated collected results.
  Tests: `network/v2/extensions/vpnaas/services/python_record_contracts_test.go:TestVPNServiceProxyJMESReturnsOpaqueGeneratorWithoutHTTP`, `network/v2/extensions/vpnaas/services/python_record_contracts_test.go:TestVPNServiceRecordsOptionsCancellationAndFailures`.

Python/Go differences:

- Legacy native List follows {resources}_links but lacks Source marker fallback. The new record pager also retries with the last physical ID when a positive limit is set and no next link exists.
- The SDK owns Source-shaped ListRecords/AllRecords local Body filters and an explicit ProxyListRecords opaque-generator result. Existing native typed List remains a separate ABI; immutable raw views are not Python mutable Resource instances.
- Explicit Go profile boundaries include configured guarded HTTP/route/constructor decoding, full Requests charset/nonfinite JSON and arbitrary Python session/cache/dynamic object behavior. These do not retain the now-closed absence of SDK local filtering.

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
    values = conn.network.vpn_services(status="ACTIVE")
    for value in values:
        print(json.dumps(value.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services"
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
    values, err := service.API.VPNServices.AllRecords(ctx,
        services.WithRecordListQuery("status", "ACTIVE"))
    if err != nil { log.Fatal(err) }
    for _, value := range values {
        raw, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(raw))
    }
}
```

## References

- [Install](../../../../../docs/install.md)
