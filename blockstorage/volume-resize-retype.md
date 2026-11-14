# Volume Resize Retype

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
    action := os.Getenv("RESIZE_ACTION")
    if action != "extend" && action != "retype" && action != "complete-extend" {
        return fmt.Errorf("RESIZE_ACTION must be extend, retype or complete-extend")
    }
    path := os.Getenv("RESIZE_ACTION_PATH")
    if path == "" {
        path = "connection"
    }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown RESIZE_ACTION_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var size int
    var newType string
    var retype blockstorage.VolumeRetypeOpts
    var completion blockstorage.VolumeExtendCompletionOpts
    switch action {
    case "extend":
        text, supplied := os.LookupEnv("EXTEND_SIZE")
        if !supplied {
            return fmt.Errorf("EXTEND_SIZE is required")
        }
        parsed, err := strconv.Atoi(text)
        if err != nil {
            return fmt.Errorf("EXTEND_SIZE: %w", err)
        }
        size = parsed
    case "retype":
        text, supplied := os.LookupEnv("NEW_VOLUME_TYPE")
        if !supplied {
            return fmt.Errorf("NEW_VOLUME_TYPE is required; an explicit empty string is allowed")
        }
        newType = text
        var originals []blockstorage.VolumeRetypeOption
        if policy, supplied := os.LookupEnv("MIGRATION_POLICY"); supplied {
            originals = append(originals, volumes.WithVolumeRetypeMigrationPolicy(policy))
        }
        prepared, err := blockstorage.PrepareVolumeRetypeOptions(ctx, originals...)
        if err != nil {
            return err
        }
        retype = prepared
    case "complete-extend":
        var originals []blockstorage.VolumeExtendCompletionOption
        if text, supplied := os.LookupEnv("COMPLETION_ERROR"); supplied {
            flag, err := strconv.ParseBool(text)
            if err != nil {
                return fmt.Errorf("COMPLETION_ERROR: %w", err)
            }
            originals = append(originals, volumes.WithVolumeExtendCompletionError(flag))
        }
        prepared, err := blockstorage.PrepareVolumeExtendCompletionOptions(ctx, originals...)
        if err != nil {
            return err
        }
        completion = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, action, path, input, size, newType, retype, completion)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest, size int, newType string, retype blockstorage.VolumeRetypeOpts, completion blockstorage.VolumeExtendCompletionOpts) (*blockstorage.VolumeActionResult, error) {
    r := volumes.WithVolumeRetypeOptions(retype)
    c := volumes.WithVolumeExtendCompletionOptions(completion)
    if path == "connection" {
        switch action {
        case "extend": return conn.ExtendVolume(ctx, input, size)
        case "retype": return conn.RetypeVolume(ctx, input, newType, r)
        default: return conn.CompleteVolumeExtend(ctx, input, c)
        }
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil {
        return nil, err
    }
    if path == "direct" {
        switch action {
        case "extend": return blockstorage.ExtendVolume(ctx, cinder.RawClient(), input, size)
        case "retype": return blockstorage.RetypeVolume(ctx, cinder.RawClient(), input, newType, r)
        default: return blockstorage.CompleteVolumeExtend(ctx, cinder.RawClient(), input, c)
        }
    }
    switch action {
    case "extend": return cinder.Volumes.ExtendVolume(ctx, input.VolumeID, size)
    case "retype": return cinder.Volumes.RetypeVolume(ctx, input.VolumeID, newType, r)
    default: return cinder.Volumes.CompleteVolumeExtend(ctx, input.VolumeID, c)
    }
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
            fmt.Printf("opaque body: %q\n", result.Applied.Body)
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
conn.block_storage.extend_volume(volume_id, 20)
conn.block_storage.retype_volume(volume_id, "actual-type-id", migration_policy="never")
conn.block_storage.complete_volume_extend(volume_id, error=False)
```

## References

- [extend/completion Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1084)
- [retype Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1126)
- [Volume.extend/complete_extend](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L165)
- [Volume.retype](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L317)
- [Resource._get_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1016)
- [Volume Flags](volume-flags.md)
- [Volume Actions](volume-actions.md)
