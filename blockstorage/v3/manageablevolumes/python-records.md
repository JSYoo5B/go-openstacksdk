# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack

with openstack.connect() as conn:
    for row in conn.block_storage.manageable_volumes(details=False, host="host@backend", limit=20, max_items=20):
        print(json.dumps(row.to_dict(), indent=2))
    for row in conn.block_storage.manageable_snapshots(details=True, host="host@backend", max_items=20):
        print(json.dumps(row.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablesnapshots"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.ManageableVolumes.ListRecords(ctx,
        request.Custom(func(c *manageablevolumes.RecordListOpts) error { c.Details = false; return nil }),
        request.Query("host", "host@backend"),
        request.Limit(20), request.MaxItems(20)) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
    for row, err := range service.ManageableSnapshots.ListRecords(ctx,
        request.Custom(func(c *manageablesnapshots.RecordListOpts) error { c.Details = true; return nil }),
        request.Query("host", "host@backend"),
        request.MaxItems(20)) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```
