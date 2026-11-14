# Cinder Volume Attachment

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
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
    if cloudName == "" || volumeID == "" { return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required") }
    action, path := os.Getenv("CINDER_ATTACHMENT_ACTION"), os.Getenv("CINDER_ATTACHMENT_PATH")
    if action != "attach" && action != "detach" { return fmt.Errorf("CINDER_ATTACHMENT_ACTION must be attach or detach") }
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" { return fmt.Errorf("unknown CINDER_ATTACHMENT_PATH %q", path) }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var bodyValue string
    var attach blockstorage.CinderVolumeAttachOpts
    var detach blockstorage.CinderVolumeDetachOpts
    if action == "attach" {
        value, supplied := os.LookupEnv("CINDER_MOUNTPOINT")
        if !supplied { return fmt.Errorf("CINDER_MOUNTPOINT is required; explicit empty is allowed") }
        bodyValue = value
        var options []blockstorage.CinderVolumeAttachOption
        if value, supplied := os.LookupEnv("CINDER_INSTANCE"); supplied {
            options = append(options, volumes.WithCinderVolumeAttachInstance(value))
        }
        if value, supplied := os.LookupEnv("CINDER_HOST_NAME"); supplied {
            options = append(options, volumes.WithCinderVolumeAttachHostName(value))
        }
        prepared, err := blockstorage.PrepareCinderVolumeAttachOptions(ctx, options...)
        if err != nil { return err }
        attach = prepared
    } else {
        value, supplied := os.LookupEnv("CINDER_ATTACHMENT_ID")
        if !supplied { return fmt.Errorf("CINDER_ATTACHMENT_ID is required; explicit empty is allowed") }
        bodyValue = value
        force := false
        if text, supplied := os.LookupEnv("CINDER_FORCE"); supplied {
            value, err := strconv.ParseBool(text)
            if err != nil { return fmt.Errorf("CINDER_FORCE: %w", err) }
            force = value
        }
        options := []blockstorage.CinderVolumeDetachOption{volumes.WithCinderVolumeDetachForce(force)}
        if text, supplied := os.LookupEnv("CINDER_CONNECTOR_JSON"); force && supplied {
            var connector map[string]json.RawMessage
            if err := json.Unmarshal([]byte(text), &connector); err != nil { return fmt.Errorf("CINDER_CONNECTOR_JSON: %w", err) }
            options = append(options, volumes.WithCinderVolumeDetachConnector(connector))
        }
        prepared, err := blockstorage.PrepareCinderVolumeDetachOptions(ctx, options...)
        if err != nil { return err }
        detach = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    result, err := apply(ctx, conn, path, action, blockstorage.VolumeActionRequest{VolumeID: volumeID}, bodyValue, attach, detach)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path, action string, input blockstorage.VolumeActionRequest, bodyValue string, attach blockstorage.CinderVolumeAttachOpts, detach blockstorage.CinderVolumeDetachOpts) (*blockstorage.VolumeActionResult, error) {
    a := volumes.WithCinderVolumeAttachOptions(attach)
    d := volumes.WithCinderVolumeDetachOptions(detach)
    if path == "connection" {
        if action == "attach" { return conn.AttachCinderVolume(ctx, input, bodyValue, a) }
        return conn.DetachCinderVolume(ctx, input, bodyValue, d)
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" {
        if action == "attach" { return blockstorage.AttachCinderVolume(ctx, cinder.RawClient(), input, bodyValue, a) }
        return blockstorage.DetachCinderVolume(ctx, cinder.RawClient(), input, bodyValue, d)
    }
    if action == "attach" { return cinder.Volumes.AttachCinderVolume(ctx, input.VolumeID, bodyValue, a) }
    return cinder.Volumes.DetachCinderVolume(ctx, input.VolumeID, bodyValue, d)
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
        if errors.As(err, &proof) { fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body)) }
    }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.attach_volume(volume_id, "/dev/vdb", instance=server_id)
conn.block_storage.attach_volume(volume_id, "", instance="", host_name="ignored")
conn.block_storage.detach_volume(volume_id, attachment_id)
conn.block_storage.detach_volume(volume_id, attachment_id, force=True,
                                 connector={"host": "", "multipath": False, "extra": None})
```

## References

- [Proxy attach/detach](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1241-L1280)
- [Volume.attach/detach](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L234-L271)
- [AttachVolume](attach-volume.md)
- [DetachVolume](detach-volume.md)
- [Volume Actions](volume-actions.md)
- [service API](v3/volumes/README.md)
