# Volume Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="source")
volume = conn.block_storage.find_volume(
    "data", ignore_missing=False, details=False, all_projects=True
)
print(volume.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    identity := flag.String("volume", "data", "Volume name or ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud("source"))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    volume, err := service.Volumes.FindRecord(ctx, *identity,
        request.IgnoreMissing(false),
        request.Custom(func(c *volumes.RecordFindOpts) error { v := false; c.Details = &v; return nil }),
        request.Custom(func(c *volumes.RecordFindOpts) error { v := true; c.AllProjects = &v; return nil }),
    )
    if err != nil { log.Fatal(err) }
    body, err := json.MarshalIndent(volume.Resource.Body, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}
```

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Volume Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py)
