# Volume Backup Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Configured(ctx context.Context, cloudName string) (*sdk.Connection, error) {
    return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func Explain(err error) {
    if err == nil { return }
    fmt.Println("workflow error:", err)
    if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("SDK wait timeout or parent context deadline")
    }
    if errors.Is(err, context.Canceled) {
        fmt.Println("parent context canceled")
    }
    if errors.Is(err, resource.ErrInvalidOption) {
        fmt.Println("local input, location, status or route error")
    }
}

func InspectCreate(result *blockstorage.CreateVolumeBackupResult) {
    if result == nil { return }
    fmt.Println("latest logical ID JSON:", string(result.BackupID))
    if result.Created != nil {
        fmt.Println("POST:", result.Created.StatusCode, "bytes:", len(result.Created.Body))
    }
    if result.LastAccepted != nil {
        fmt.Println("last admitted response:", result.LastAccepted.StatusCode)
    }
    fmt.Println("created-stage logical JSON:", string(result.CreatedValue))
    if result.CreatedBackup != nil {
        actual := result.CreatedBackup.Clone()
        fmt.Println("actual POST ID:", string(actual.Body["id"]))
        var fields map[string]json.RawMessage
        if err := actual.Decode(&fields); err != nil {
            fmt.Println("optional projection:", err)
        } else {
            fmt.Println("actual POST field count:", len(fields))
        }
    }
    fmt.Println("final logical JSON:", string(result.Value), "ready:", len(result.Ready) != 0)
}

func InspectDelete(result *blockstorage.DeleteVolumeBackupResult) {
    if result == nil { return }
    if result.Deleted == nil {
        fmt.Println("requested delete policy did not complete")
    } else {
        fmt.Println("deleted:", *result.Deleted)
    }
    fmt.Println("latest logical ID JSON:", string(result.BackupID))
    if result.Resolved != nil {
        fmt.Println("resolved logical JSON:", string(result.Resolved.Value))
        fmt.Println("resolved list pages:", len(result.Resolved.Pages))
        if result.Resolved.Observed != nil {
            fmt.Println("resolved member:", result.Resolved.Observed.StatusCode)
        }
    }
    if result.Applied != nil {
        fmt.Println("DELETE or force POST:", result.Applied.StatusCode, "bytes:", len(result.Applied.Body))
    }
    if result.LastAccepted != nil {
        fmt.Println("last admitted response:", result.LastAccepted.StatusCode)
    }
    if result.Absent != nil {
        fmt.Println("clean polling GET absence:", result.Absent.StatusCode)
    }
    fmt.Println("deleted-status logical JSON:", string(result.Ready))
}

func CreateDefault(ctx context.Context, conn *sdk.Connection, volumeID string) error {
    result, err := conn.CreateVolumeBackup(ctx,
        blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID})
    InspectCreate(result)
    Explain(err)
    return err
}

func CreateWithoutWait(ctx context.Context, conn *sdk.Connection, volumeID, snapshotID string) error {
    policy, err := blockstorage.PrepareCreateVolumeBackupOptions(ctx,
        blockstorage.WithCreateVolumeBackupName(""),
        blockstorage.WithCreateVolumeBackupDescription(""),
        blockstorage.WithCreateVolumeBackupForce(false),
        blockstorage.WithCreateVolumeBackupIncremental(true),
        blockstorage.WithCreateVolumeBackupSnapshotID(snapshotID),
        blockstorage.WithCreateVolumeBackupWait(false))
    if err != nil { return err }
    result, err := conn.CreateVolumeBackup(ctx,
        blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeBackupOptions(policy))
    InspectCreate(result)
    return err
}

func RemoveDefault(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    result, err := conn.DeleteVolumeBackup(ctx,
        blockstorage.DeleteVolumeBackupRequest{NameOrID: nameOrID})
    InspectDelete(result)
    Explain(err)
    return err
}

func ForceAndWait(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    timeout := 10 * time.Minute
    policy, err := blockstorage.PrepareDeleteVolumeBackupOptions(ctx,
        blockstorage.WithDeleteVolumeBackupForce(true),
        blockstorage.WithDeleteVolumeBackupWait(true),
        blockstorage.WithDeleteVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{
            Timeout: &timeout,
        }))
    if err != nil { return err }
    result, err := conn.DeleteVolumeBackup(ctx,
        blockstorage.DeleteVolumeBackupRequest{NameOrID: nameOrID},
        blockstorage.WithDeleteVolumeBackupOptions(policy))
    InspectDelete(result)
    Explain(err)
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient, volumeID string) error {
    cloud, region := "owned-cloud", "RegionOne"
    location := resource.CloudLocation{
        Cloud: &cloud, RegionName: &region,
        Project: resource.CloudProject{ID: json.RawMessage(`"project-id"`)},
    }
    result, err := blockstorage.CreateVolumeBackup(ctx, cinder,
        blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID},
        func(c *blockstorage.CreateVolumeBackupOpts) error { loc := location.Clone(); c.Location = &loc; return nil },
        blockstorage.WithCreateVolumeBackupForce(false),
        blockstorage.WithCreateVolumeBackupWait(false))
    InspectCreate(result)
    return err
}

func DirectDelete(ctx context.Context, cinder *gophercloud.ServiceClient, nameOrID string) error {
    result, err := blockstorage.DeleteVolumeBackup(ctx, cinder,
        blockstorage.DeleteVolumeBackupRequest{NameOrID: nameOrID},
        blockstorage.WithDeleteVolumeBackupForce(false),
        blockstorage.WithDeleteVolumeBackupWait(false))
    InspectDelete(result)
    return err
}

func EmptyDelete(ctx context.Context) error {
    var conn *sdk.Connection
    result, err := conn.DeleteVolumeBackup(ctx, blockstorage.DeleteVolumeBackupRequest{})
    InspectDelete(result)
    if err != nil { return err }
    if result == nil || result.Deleted == nil || *result.Deleted {
        return errors.New("empty delete must return a completed false result")
    }
    return nil
}
```

```python
import openstack

conn = openstack.connect(cloud="my-cloud")
created = conn.create_volume_backup(volume_id)
queued = conn.create_volume_backup(
    volume_id, name="", description="", force=False, wait=False,
    incremental=True, snapshot_id=snapshot_id,
)
deleted = conn.delete_volume_backup("nightly")
forced = conn.delete_volume_backup("nightly", force=True, wait=True, timeout=600)
empty = conn.delete_volume_backup()
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L600)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L758)
- [Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L116)
- [V3](v3/README.md)
- [v3 Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L26)
- [Volume Backups](volume-backups.md)
- [Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L132)
- [native Backup Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L60)
- [ForceDelete](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L357)
- [Microversions](../docs/microversions.md)
