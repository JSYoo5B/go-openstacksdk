# Python Raw Max Items

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--max-items-json", default="null")
args = parser.parse_args()
cap = json.loads(args.max_items_json)
with openstack.connect(cloud=args.cloud) as conn:
    for row in conn.network.networks(limit=10, max_items=cap):
        print(json.dumps(row.to_dict(), sort_keys=True))
```

```go
package main

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    capJSON := flag.String("max-items-json", "null", "raw JSON max_items")
    flag.Parse()
    raw := json.RawMessage(*capJSON)
    if !json.Valid(raw) { log.Fatal("max-items-json must be complete JSON") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []networks.RecordListOption{
        networks.WithCloudRecordListQuery("max_items", raw),
        request.Limit(10),
    }
    proxy, err := network.Networks.ProxyCloudListRecords(ctx, nil, options...)
    if err != nil { log.Fatal(err) }
    generator, ok := proxy.Value.(*networks.RecordListGenerator)
    if !ok { log.Fatal("expected a Resource list generator") }
    for row, err := range generator.Records {
        if err != nil { log.Fatal(err) }
        encoded, err := json.Marshal(row.Resource)
        if err != nil { log.Fatal(err) }
        fmt.Println(string(encoded))
    }
}
```

## References

- [Install](../../../docs/install.md)
