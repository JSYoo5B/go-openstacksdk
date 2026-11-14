# Python List Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

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
