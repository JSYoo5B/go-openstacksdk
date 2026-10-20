# Volume Summary Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/volumes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ListWithDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ListSummary`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ListSummaryWithControl`

- Volumes.List always requests volumes/detail, sends name, status and all_tenants=True as given and follows volumes_links.
  Tests: `blockstorage/v3/volumes/python_parity_test.go:TestPythonVolumeListDefaultUsesDetailRouteOnly`.
- details=false selects GET /volumes, details=true and existing List retain /volumes/detail; selected query/native options, ResourceBase and authentication are preserved and linked native pages yield typed summary rows.
  Tests: `blockstorage/v3/volumes/summary_test.go:TestPythonVolumeSummarySelectsDetailsAndPreservesQuery`.
- Shared raw-row MaxItems and SinglePage never rewrite an explicit wire limit; consumer break stops continuation and whole native page decode errors remain observable beyond a cap.
  Tests: `blockstorage/v3/volumes/summary_test.go:TestPythonVolumeSummaryLocalControlsAndConsumerBreak`.
- Invalid receiver/context/control/options do not issue HTTP, callback failure and cancellation cause remain joined, and a later page native HTTP error preserves prior rows and ListSummary operation identity.
  Tests: `blockstorage/v3/volumes/summary_test.go:TestPythonVolumeSummaryPreflightAndPartialFailures`.

Python/Go differences:

- Python all_projects=True sends all_tenants=True; typed Go AllTenants sends all_tenants=true, so WithListQuery("all_tenants", "True") is needed for the identical string.
- Python filters returned rows locally for Body attributes outside the query mapping; Go sends only query parameters and leaves local filtering to the caller.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.
- The shared native summary cap is local and does not supply Python max_items wire-limit hints; native linked URL/query and whole-page decode policy remain explicit. The aggregate /volumes/summary API is a different operation.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    for volume, err := range service.Volumes.ListSummaryWithControl(ctx,
        resource.ListControl{MaxItems: 100},
        volumes.WithListOptions(volumes.ListOpts{Limit: 20})) {
        if err != nil { log.Fatal(err) }
        fmt.Println(volume.ID, volume.Name)
    }
}
```

## References

- [volumes Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L894-L920)
- [summary_test.go](volumes/summary_test.go)
