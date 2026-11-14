# Backup Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="source")
backup = conn.block_storage.find_backup(
    "nightly", ignore_missing=False, details=False
)
print(backup.id, backup.name, backup.created_at)
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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    identity := flag.String("backup", "nightly", "Backup name or ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud("source"))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    backup, err := service.Backups.FindRecord(ctx, *identity,
        request.IgnoreMissing(false),
        request.Custom(func(c *backups.RecordFindOpts) error { v := false; c.Details = &v; return nil }),
    )
    if err != nil { log.Fatal(err) }
    body, err := json.MarshalIndent(backup.Resource.Body, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}
```

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Backup Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py)
