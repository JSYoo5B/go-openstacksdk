# Volume Migration Reset

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
    action := os.Getenv("VOLUME_MIGRATION_ACTION")
    if action != "reset" && action != "migrate" && action != "complete" {
        return fmt.Errorf("VOLUME_MIGRATION_ACTION must be reset, migrate or complete")
    }
    path := os.Getenv("VOLUME_MIGRATION_PATH")
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" {
        return fmt.Errorf("unknown VOLUME_MIGRATION_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    var reset blockstorage.VolumeStatusResetOpts
    var migration blockstorage.VolumeMigrationOpts
    var completion blockstorage.VolumeMigrationCompletionOpts
    var newVolume string
    switch action {
    case "reset":
        var options []blockstorage.VolumeStatusResetOption
        if value, supplied := os.LookupEnv("RESET_STATUS"); supplied {
            options = append(options, volumes.WithVolumeStatusResetStatus(value))
        }
        if value, supplied := os.LookupEnv("RESET_ATTACH_STATUS"); supplied {
            options = append(options, volumes.WithVolumeStatusResetAttachStatus(value))
        }
        if value, supplied := os.LookupEnv("RESET_MIGRATION_STATUS"); supplied {
            options = append(options, volumes.WithVolumeStatusResetMigrationStatus(value))
        }
        prepared, err := blockstorage.PrepareVolumeStatusResetOptions(ctx, options...)
        if err != nil { return err }
        reset = prepared
    case "migrate":
        var options []blockstorage.VolumeMigrationOption
        if value, supplied := os.LookupEnv("MIGRATION_HOST"); supplied {
            options = append(options, volumes.WithVolumeMigrationHost(value))
        }
        if value, supplied := os.LookupEnv("MIGRATION_CLUSTER"); supplied {
            options = append(options, volumes.WithVolumeMigrationCluster(value))
        }
        for _, control := range []struct {
            name string
            option func(bool) blockstorage.VolumeMigrationOption
        }{
            {"MIGRATION_FORCE_HOST_COPY", volumes.WithVolumeMigrationForceHostCopy},
            {"MIGRATION_LOCK_VOLUME", volumes.WithVolumeMigrationLockVolume},
        } {
            if text, supplied := os.LookupEnv(control.name); supplied {
                value, err := strconv.ParseBool(text)
                if err != nil { return fmt.Errorf("%s: %w", control.name, err) }
                options = append(options, control.option(value))
            }
        }
        prepared, err := blockstorage.PrepareVolumeMigrationOptions(ctx, options...)
        if err != nil { return err }
        migration = prepared
    case "complete":
        value, supplied := os.LookupEnv("NEW_VOLUME_ID")
        if !supplied { return fmt.Errorf("NEW_VOLUME_ID is required; explicit empty is allowed") }
        newVolume = value
        var options []blockstorage.VolumeMigrationCompletionOption
        if text, supplied := os.LookupEnv("MIGRATION_ERROR"); supplied {
            value, err := strconv.ParseBool(text)
            if err != nil { return fmt.Errorf("MIGRATION_ERROR: %w", err) }
            options = append(options, volumes.WithVolumeMigrationCompletionError(value))
        }
        prepared, err := blockstorage.PrepareVolumeMigrationCompletionOptions(ctx, options...)
        if err != nil { return err }
        completion = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    result, err := apply(ctx, conn, path, action, input, newVolume, reset, migration, completion)
    inspect(result, err)
    return err
}

func apply(ctx context.Context, conn *sdk.Connection, path, action string, input blockstorage.VolumeActionRequest, newVolume string, reset blockstorage.VolumeStatusResetOpts, migration blockstorage.VolumeMigrationOpts, completion blockstorage.VolumeMigrationCompletionOpts) (*blockstorage.VolumeActionResult, error) {
    r := volumes.WithVolumeStatusResetOptions(reset)
    m := volumes.WithVolumeMigrationOptions(migration)
    c := volumes.WithVolumeMigrationCompletionOptions(completion)
    if path == "connection" {
        switch action {
        case "reset": return conn.ResetVolumeStatus(ctx, input, r)
        case "migrate": return conn.MigrateVolume(ctx, input, m)
        default: return conn.CompleteVolumeMigration(ctx, input, newVolume, c)
        }
    }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" {
        switch action {
        case "reset": return blockstorage.ResetVolumeStatus(ctx, cinder.RawClient(), input, r)
        case "migrate": return blockstorage.MigrateVolume(ctx, cinder.RawClient(), input, m)
        default: return blockstorage.CompleteVolumeMigration(ctx, cinder.RawClient(), input, newVolume, c)
        }
    }
    switch action {
    case "reset": return cinder.Volumes.ResetVolumeStatus(ctx, input.VolumeID, r)
    case "migrate": return cinder.Volumes.MigrateVolume(ctx, input.VolumeID, m)
    default: return cinder.Volumes.CompleteVolumeMigration(ctx, input.VolumeID, newVolume, c)
    }
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
conn.block_storage.reset_volume_status(volume_id, status="available", attach_status="detached")
conn.block_storage.migrate_volume(volume_id, host="host@backend", force_host_copy=True)
conn.block_storage.complete_volume_migration(volume_id, new_volume_id, error=False)
```

## References

- [reset Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1199)
- [migration/completion Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1336)
- [Volume.reset_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L209)
- [Volume.migrate/complete_migration](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L330)
- [Keystoneauth5.16.0 version_match](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)
- [Volume Actions](volume-actions.md)
- [Volume Resize Retype](volume-resize-retype.md)
- [service API](v3/volumes/README.md)
