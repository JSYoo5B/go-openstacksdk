# Volume Revert Unmanage

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
    action := os.Getenv("VOLUME_LIFECYCLE_ACTION")
    if action != "revert" && action != "unmanage" {
        return fmt.Errorf("VOLUME_LIFECYCLE_ACTION must be revert or unmanage")
    }
    path := os.Getenv("VOLUME_LIFECYCLE_PATH")
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown VOLUME_LIFECYCLE_PATH %q", path)
    }
    var snapshotID string
    if action == "revert" {
        value, supplied := os.LookupEnv("SNAPSHOT_ID")
        if !supplied { return fmt.Errorf("SNAPSHOT_ID is required; explicit empty is allowed") }
        snapshotID = value
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, path, action, input, snapshotID)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path, action string, input blockstorage.VolumeActionRequest, snapshotID string) (*blockstorage.VolumeActionResult, error) {
    if path == "connection" {
        if action == "revert" { return conn.RevertVolumeToSnapshot(ctx, input, snapshotID) }
        return conn.UnmanageVolume(ctx, input)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" {
        if action == "revert" { return blockstorage.RevertVolumeToSnapshot(ctx, cinder.RawClient(), input, snapshotID) }
        return blockstorage.UnmanageVolume(ctx, cinder.RawClient(), input)
    }
    if action == "revert" { return cinder.Volumes.RevertVolumeToSnapshot(ctx, input.VolumeID, snapshotID) }
    return cinder.Volumes.UnmanageVolume(ctx, input.VolumeID)
}

func inspect(result *blockstorage.VolumeActionResult, err error) {
    if result != nil {
        fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
        for index, page := range result.Discovery {
            if page != nil { fmt.Println("admitted discovery response:", index, page.StatusCode, len(page.Body)) }
        }
        if result.Applied != nil {
            fmt.Println("admitted action response:", result.Applied.StatusCode)
            fmt.Printf("opaque body: %q\n", result.Applied.Body)
        }
        fmt.Println("helper acknowledgement completed:", result.Completed)
    }
    if err != nil {
        var proof *resource.ResponseError
        if errors.As(err, &proof) {
            fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body))
        }
    }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.revert_volume_to_snapshot(volume_id, snapshot_id)
conn.block_storage.unmanage_volume(volume_id)
```

## References

- [revert Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1220-L1239)
- [unmanage Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1293-L1303)
- [Volume.revert_to_snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L226-L232)
- [Volume.unmanage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L311-L315)
- [Keystoneauth5.16.0 version_match](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)
- [Volume Actions](volume-actions.md)
- [Volume Migration Reset](volume-migration-reset.md)
- [service API](v3/volumes/README.md)
