# Volume Actions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
    if cloudName == "" || volumeID == "" {
        return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
    }
    action := os.Getenv("STATE_ACTION")
    if action == "" {
        action = "reserve"
    }
    switch action {
    case "reserve", "unreserve", "begin-detaching", "abort-detaching":
    default:
        return fmt.Errorf("unknown STATE_ACTION %q", action)
    }
    path := os.Getenv("STATE_ACTION_PATH")
    if path == "" {
        path = "connection"
    }
    switch path {
    case "connection", "direct", "service":
    default:
        return fmt.Errorf("unknown STATE_ACTION_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, action, path, input)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
    if path == "connection" {
        calls := map[string]func(context.Context, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error){
            "reserve": conn.ReserveVolume,
            "unreserve": conn.UnreserveVolume,
            "begin-detaching": conn.BeginVolumeDetaching,
            "abort-detaching": conn.AbortVolumeDetaching,
        }
        return calls[action](ctx, input)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil {
        return nil, err
    }
    if path == "direct" {
        calls := map[string]func(context.Context, *gophercloud.ServiceClient, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error){
            "reserve": blockstorage.ReserveVolume,
            "unreserve": blockstorage.UnreserveVolume,
            "begin-detaching": blockstorage.BeginVolumeDetaching,
            "abort-detaching": blockstorage.AbortVolumeDetaching,
        }
        return calls[action](ctx, cinder.RawClient(), input)
    }
    calls := map[string]func(context.Context, string) (*blockstorage.VolumeActionResult, error){
        "reserve": cinder.Volumes.ReserveVolume,
        "unreserve": cinder.Volumes.UnreserveVolume,
        "begin-detaching": cinder.Volumes.BeginVolumeDetaching,
        "abort-detaching": cinder.Volumes.AbortVolumeDetaching,
    }
    return calls[action](ctx, input.VolumeID)
}

func inspect(result *blockstorage.VolumeActionResult, err error) {
    if result != nil {
        fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
        for i, page := range result.Discovery {
            if page != nil {
                fmt.Println("admitted discovery response:", i, page.StatusCode, len(page.Body))
            }
        }
        if result.Applied != nil {
            fmt.Println("admitted action response:", result.Applied.StatusCode)
            fmt.Printf("opaque action body: %q\n", result.Applied.Body)
        }
        fmt.Println("helper acknowledgement completed:", result.Completed)
    }
    if err != nil {
        fmt.Println("action error:", err)
        var physical *resource.ResponseError
        if errors.As(err, &physical) {
            fmt.Println("current admitted error response:", physical.StatusCode, len(physical.Body))
        }
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("parent context interrupted the call")
        }
    }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
volume_id = "actual-volume-id"
conn.block_storage.reserve_volume(volume_id)
conn.block_storage.unreserve_volume(volume_id)
conn.block_storage.begin_volume_detaching(volume_id)
conn.block_storage.abort_volume_detaching(volume_id)
```

## References

- [v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1422)
- [Volume methods](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L405)
- [Source `_get_resource`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563)
- [Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L146)
- [Source utility](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L358)
- [native Reserve/Unreserve/BeginDetaching](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go#L363)
- [cloud bootable helper](volume-mutations.md)
- [Volume Flags](volume-flags.md)
