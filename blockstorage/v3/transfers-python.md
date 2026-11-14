# Transfers Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="source")
transfer = conn.block_storage.create_transfer(
    volume_id="v1", name="handoff", no_snapshots=True
)
for candidate in conn.block_storage.transfers(details=False):
    print(candidate.id, candidate.name)
found = conn.block_storage.find_transfer("handoff", ignore_missing=False)
print(found.id)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/JSYoo5B/go-openstacksdk/request"
    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx, openstack.WithCloud("source"))
    if err != nil { log.Fatal(err) }
    storage, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    created, err := storage.Transfers.CreateRecord(ctx,
        transfers.WithRecordCreateVolumeID("v1"),
        transfers.WithRecordCreateName("handoff"),
        transfers.WithRecordCreateNoSnapshots(true),
        request.Custom(func(c *transfers.RecordCreateOpts) error { c.SuppressLegacySnapshots = true; return nil }),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(created.Resource.Body["id"]))
    for candidate, err := range storage.Transfers.ListRecords(ctx,
        request.Custom(func(c *transfers.RecordListOpts) error { v := false; c.Details = &v; return nil }),
    ) {
        if err != nil { log.Fatal(err) }
        fmt.Println(string(candidate.Resource.Body["id"]), string(candidate.Resource.Body["name"]))
    }
    found, err := storage.Transfers.FindRecord(ctx, "handoff",
        request.IgnoreMissing(false),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(found.Resource.Body["id"]))
}
```

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Transfer Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/transfer.py)
