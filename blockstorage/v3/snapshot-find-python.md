# Snapshot Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev", block_storage_default_microversion="3.65")
snapshot = conn.block_storage.find_snapshot("daily")
summary = conn.block_storage.find_snapshot("daily", details=False)
strict = conn.block_storage.find_snapshot(
    "daily", details=True, all_projects=True, ignore_missing=False)
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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("snapshot", "daily", "snapshot name or ID")
    details := flag.Bool("details", true, "use detailed fallback list")
    allProjects := flag.Bool("all-projects", false, "search across projects")
    strict := flag.Bool("strict", false, "return an error when missing")
    version := flag.String("microversion", "3.65", "Cinder selected version; empty enables discovery")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    value, err := service.Snapshots.FindRecord(ctx, *identity,
        request.Custom(func(c *snapshots.RecordFindOpts) error { v := *details; c.Details = &v; return nil }),
        request.Custom(func(c *snapshots.RecordFindOpts) error { v := *allProjects; c.AllProjects = &v; return nil }),
        request.IgnoreMissing(!*strict))
    if err != nil { log.Fatal(err) }
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [find_snapshot proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L154-L193)
- [Snapshot descriptor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py)
- [Install](../../docs/install.md)
- [Python Find Contracts Test](snapshots/python_find_contracts_test.go)
