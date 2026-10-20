# Python List Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/ips`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithRecordListOpts`

- All with no options sends GET /v2.0/floatingips with Accept application/json and follows floatingips_links, WithQuery forwards query-mapped keys such as floating_network_id and status, and WithName or WithBodyFilter return ErrUnsupported without HTTP.
  Tests: `network/v2/extensions/layer3/floatingips/python_parity_ips_test.go:TestPythonIPsListServerQueryAndPagesButNoLocalBodyFilters`.
- Source-shaped record lists send declared queries including tenant_id/tag wire aliases, keep status server-only, filter name/DNS/QoS/recursive port_details locally, discard unknown kwargs before encoding, and synchronize the shared name/address descriptor without changing native List/All.
  Tests: `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsQueryClassificationAndAliasOrder`.
- Record view applies nullable/list/dictionary/integer descriptors and preserves unknown extensions while Wire and the actual page status/header/envelope remain independently owned.
  Tests: `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsProjectionProofAndClone`, `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsDescriptorFaultKeepsPhysicalPage`.
- The shared Source pager consumes max_items before local filters, supports deferred negative-cap stopping and consumer break without constructing later malformed rows; later HTTP failures retain native evidence and AllRecords discards partial results.
  Tests: `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsCapsBeforeFiltersAndLazyBreak`, `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsPaginationLateErrorsAndResponseProof`.
- Concrete options snapshot raw maps/bytes/headers/version, cancellation stops HTTP, and Proxy JMESPath observes an unconsumed generator rather than materializing the list.
  Tests: `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsOwnedOptionsHeadersAndCancellation`, `network/v2/extensions/layer3/floatingips/python_list_records_test.go:TestPythonIPRecordsProxyGeneratorAndJMESPath`.

Python/Go differences:

- Native resource.WithStatus sends the status query and also filters case-insensitively in memory; the new record API instead keeps status server-only like Source. Native WithName/WithBodyFilter remain unsupported.
- Owned finite UTF-8 JSON records replace mutable Python Resource/runtime objects. Arbitrary Python objects, nonfinite JSON and full Requests/session/cache behavior remain explicit common profile boundaries.
- Go base_path overrides must be service-relative without traversal/query/fragment and continuation must remain in the selected collection; an empty override retains the default route.
- ListRecords/AllRecords reject truthy string JMESPath because it changes the return type. ProxyListRecords preserves Source unconsumed-generator evaluation and returns its separate opaque or JSON result.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--dns-name", default="app")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    rows = conn.network.ips(status="ACTIVE", dns_name=args.dns_name)
    print(json.dumps([row.to_dict() for row in rows], indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    dns := flag.String("dns-name", "app", "local DNS filter")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    rows, err := service.API.FloatingIPs.AllRecords(ctx,
        floatingips.WithRecordListQuery("status", "ACTIVE"),
        floatingips.WithRecordListQuery("dns_name", *dns))
    if err != nil { log.Fatal(err) }
    values := make([]any, 0, len(rows))
    for _, row := range rows { values = append(values, row.Resource) }
    raw, err := json.MarshalIndent(values, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}
```

## References

- [Install](../../../../../docs/install.md)
