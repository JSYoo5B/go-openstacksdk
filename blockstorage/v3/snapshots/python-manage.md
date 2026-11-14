# Python Manage

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--action", choices=("manage", "unmanage"), required=True)
parser.add_argument("--identity", required=True)
parser.add_argument("--reference", default='{"source-name":"backend-snapshot"}')
args = parser.parse_args()
with openstack.connect() as conn:
    if args.action == "manage":
        row = conn.block_storage.manage_snapshot(volume_id=args.identity, ref=json.loads(args.reference))
        print(json.dumps(row.to_dict(), indent=2))
    else:
        conn.block_storage.unmanage_snapshot(args.identity)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    sdk "github.com/JSYoo5B/go-openstacksdk"
)

func main() {
    action := flag.String("action", "", "manage or unmanage")
    identity := flag.String("identity", "", "volume ID for manage, snapshot ID for unmanage")
    reference := flag.String("reference", `{"source-name":"backend-snapshot"}`, "backend reference JSON")
    flag.Parse()
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    switch *action {
    case "manage":
        row, err := service.Snapshots.ManageRecord(ctx, *identity, json.RawMessage(*reference))
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    case "unmanage":
        receipt, err := service.Snapshots.UnmanageRecord(ctx, *identity)
        if err != nil { log.Fatal(err) }
        fmt.Println(receipt.StatusCode)
    default:
        log.Fatal("action must be manage or unmanage")
    }
}
```
