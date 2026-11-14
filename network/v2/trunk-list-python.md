# Trunk List Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--name", default="")
parser.add_argument("--port", default="")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--single-page", action="store_true")
args = parser.parse_args()

conn = openstack.connect(cloud=args.cloud)
query = {"paginated": not args.single_page, "max_items": args.max_items}
if args.name:
    query["name"] = args.name
if args.port:
    query["port_id"] = args.port
for trunk in conn.network.trunks(**query):
    print(json.dumps(trunk.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    name := flag.String("name", "", "server-side trunk name query")
    port := flag.String("port", "", "parent port ID query")
    cap := flag.Int("max-items", 0, "raw row cap; 0 means unlimited")
    single := flag.Bool("single-page", false, "observe first page only")
    version := flag.String("microversion", "", "optional selected service version")
    flag.Parse()

    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    connOptions := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" {
        connOptions = append(connOptions, sdk.WithMicroversion(sdk.Network, *version))
    }
    conn, err := sdk.Connect(ctx, connOptions...)
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []trunks.RecordListOption{
        request.MaxItems(*cap),
        request.Paginated(!*single),
    }
    if *name != "" { options = append(options, trunks.WithRecordListQuery("name", *name)) }
    if *port != "" { options = append(options, trunks.WithRecordListQuery("port_id", *port)) }
    rows, err := service.Trunks.AllRecords(ctx, options...)
    if err != nil { log.Fatal(err) }
    for _, row := range rows {
        body, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(body))
        fmt.Printf("HTTP %d request-id=%s\n", row.StatusCode,
            row.Header.Get("X-Openstack-Request-Id"))
    }
}
```

## References

- [Install](../../docs/install.md)
- [Trunk](https://opendev.org/openstack/neutron/src/tag/27.0.0/neutron/conf/policies/trunk.py)
