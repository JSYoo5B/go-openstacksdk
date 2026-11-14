# Volume Flags

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
    "strconv"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
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
    action := os.Getenv("FLAG_ACTION")
    if action != "bootable" && action != "readonly" {
        return fmt.Errorf("FLAG_ACTION must be bootable or readonly")
    }
    path := os.Getenv("FLAG_ACTION_PATH")
    if path == "" {
        path = "connection"
    }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown FLAG_ACTION_PATH %q", path)
    }
    text, supplied := os.LookupEnv("FLAG_VALUE")
    if action == "bootable" && !supplied {
        return fmt.Errorf("FLAG_VALUE is required for bootable")
    }
    flag := true
    if supplied {
        parsed, err := strconv.ParseBool(text)
        if err != nil {
            return fmt.Errorf("FLAG_VALUE: %w", err)
        }
        flag = parsed
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var policy blockstorage.VolumeReadonlyOpts
    if action == "readonly" {
        var originals []blockstorage.VolumeReadonlyOption
        if supplied {
            originals = append(originals, volumes.WithVolumeReadonly(flag))
        }
        prepared, err := blockstorage.PrepareVolumeReadonlyOptions(ctx, originals...)
        if err != nil {
            return err
        }
        policy = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, action, path, input, flag, policy)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest, flag bool, policy blockstorage.VolumeReadonlyOpts) (*blockstorage.VolumeActionResult, error) {
    options := []blockstorage.VolumeReadonlyOption{volumes.WithVolumeReadonlyOptions(policy)}
    if path == "connection" {
        if action == "bootable" {
            return conn.SetVolumeBootableStatus(ctx, input, flag)
        }
        return conn.SetVolumeReadonly(ctx, input, options...)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil {
        return nil, err
    }
    if path == "direct" {
        if action == "bootable" {
            return blockstorage.SetVolumeBootableStatus(ctx, cinder.RawClient(), input, flag)
        }
        return blockstorage.SetVolumeReadonly(ctx, cinder.RawClient(), input, options...)
    }
    if action == "bootable" {
        return cinder.Volumes.SetVolumeBootableStatus(ctx, input.VolumeID, flag)
    }
    return cinder.Volumes.SetVolumeReadonly(ctx, input.VolumeID, options...)
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
    }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
volume_id = "actual-volume-id"
conn.block_storage.set_volume_bootable_status(volume_id, False)
conn.block_storage.set_volume_readonly(volume_id)       # default True
conn.block_storage.set_volume_readonly(volume_id, False)
```

## References

- [bootable Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1149)
- [readonly Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1111)
- [Volume methods](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L177)
- [Volume class maximum 3.71](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L146)
- [Volume Actions](volume-actions.md)
- [cloud `SetVolumeBootable`](volume-mutations.md)
- [native `SetBootable`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go#L644)
