# Volume Snapshot Mutations

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

func InspectCreate(result *blockstorage.CreateVolumeSnapshotResult) {
    if result == nil { return }
    fmt.Println("latest logical ID JSON:", string(result.SnapshotID))
    if result.Created != nil {
        fmt.Println("POST:", result.Created.StatusCode, "bytes:", len(result.Created.Body))
    }
    if result.LastAccepted != nil {
        fmt.Println("last admitted response:", result.LastAccepted.StatusCode)
    }
    fmt.Println("created-stage logical JSON:", string(result.CreatedValue))
    if result.CreatedSnapshot != nil {
        actual := result.CreatedSnapshot.Clone()
        fmt.Println("actual POST ID field:", string(actual.Body["id"]))
        var fields map[string]json.RawMessage
        if err := actual.Decode(&fields); err != nil {
            fmt.Println("optional projection:", err)
        } else {
            fmt.Println("actual POST field count:", len(fields))
        }
    }
    fmt.Println("final logical JSON:", string(result.Value), "ready:", len(result.Ready) != 0)
}

func InspectDelete(result *blockstorage.DeleteVolumeSnapshotResult) {
    if result == nil { return }
    if result.Deleted != nil { fmt.Println("requested deletion completed:", *result.Deleted) }
    if result.Resolved != nil {
        fmt.Println("lookup pages:", len(result.Resolved.Pages), "seeded ID:", result.Resolved.SeededID)
        if result.Resolved.Observed != nil {
            fmt.Println("lookup member:", result.Resolved.Observed.StatusCode)
        }
    }
    if result.Applied != nil { fmt.Println("DELETE acknowledged:", result.Applied.StatusCode) }
    if result.Absent != nil { fmt.Println("clean polling absence:", result.Absent.StatusCode) }
    fmt.Println("latest logical ID JSON:", string(result.SnapshotID))
}

func InspectError(err error) {
    if err == nil { return }
    var response *resource.ResponseError
    if errors.As(err, &response) {
        fmt.Println("accepted response failure:", response.StatusCode, len(response.Body))
    }
    var sdkTimeout *blockstorage.SnapshotWaitTimeoutError
    if errors.As(err, &sdkTimeout) {
        fmt.Println("SDK wait budget:", sdkTimeout.Timeout)
    }
    fmt.Println("deadline cause:", errors.Is(err, context.DeadlineExceeded))
}

func ViaConnection(ctx context.Context, cloudName, volumeID string) error {
    conn, err := Configured(ctx, cloudName)
    if err != nil { return err }
    created, err := conn.CreateVolumeSnapshot(ctx,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotName("nightly"),
        blockstorage.WithCreateVolumeSnapshotDescription("nightly volume snapshot"))
    InspectCreate(created) // Inspect completed physical stages even when err is nonnil.
    if err != nil { InspectError(err); return err }

    // Default deletion resolves exactly one snapshot and returns after acknowledgement.
    deleted, err := conn.DeleteVolumeSnapshot(ctx,
        blockstorage.DeleteVolumeSnapshotRequest{NameOrID: "nightly"})
    InspectDelete(deleted)
    InspectError(err)
    return err
}

func DirectNoWait(ctx context.Context, cinder *gophercloud.ServiceClient, volumeID string) error {
    result, err := blockstorage.CreateVolumeSnapshot(ctx, cinder,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotForce(false),
        blockstorage.WithCreateVolumeSnapshotWait(false),
        blockstorage.WithCreateVolumeSnapshotFields(map[string]json.RawMessage{
            "display_name": json.RawMessage(`"nightly"`),
            "description": json.RawMessage(`"created without polling"`),
        }))
    InspectCreate(result)
    InspectError(err)
    return err
}

func WaitForDeletion(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    budget, interval := 10*time.Minute, time.Second
    result, err := conn.DeleteVolumeSnapshot(ctx,
        blockstorage.DeleteVolumeSnapshotRequest{NameOrID: nameOrID},
        blockstorage.WithDeleteVolumeSnapshotWait(true),
        blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{
            Timeout: &budget, PollInterval: &interval,
        }))
    InspectDelete(result)
    InspectError(err)
    return err
}

func ZeroWaitBudget(ctx context.Context, conn *sdk.Connection, volumeID string) (*blockstorage.CreateVolumeSnapshotResult, error) {
    zero := time.Duration(0)
    result, err := conn.CreateVolumeSnapshot(ctx,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: &zero}))
    InspectCreate(result) // POST occurs first; an initially available result may already succeed.
    InspectError(err)
    return result, err
}

func WithCallerDeadline(ctx context.Context, conn *sdk.Connection, volumeID string) error {
    parent, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    result, err := conn.CreateVolumeSnapshot(parent,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID})
    InspectCreate(result)
    InspectError(err)
    return err
}

func OwnedLocation(ctx context.Context, conn *sdk.Connection, volumeID string) error {
    cloud, project := "configured-cloud", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud, Project: resource.CloudProject{
            ID: json.RawMessage(`"token-scope-id"`), Name: &project,
        },
    }
    wait := false
    result, err := conn.CreateVolumeSnapshot(ctx,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotOptions(blockstorage.CreateVolumeSnapshotOpts{
            Wait: &wait, Location: &location,
        }))
    InspectCreate(result)
    return err
}

func EmptyDelete(ctx context.Context) error {
    var conn *sdk.Connection
    result, err := conn.DeleteVolumeSnapshot(ctx, blockstorage.DeleteVolumeSnapshotRequest{})
    InspectDelete(result) // Valid context, no service selection, successful false.
    return err
}
```

```python
import openstack

conn = openstack.connect(cloud="configured-cloud")
snapshot = conn.create_volume_snapshot(
    "volume-id", name="nightly", description="nightly volume snapshot"
)  # force=False, wait=True, timeout=None
accepted = conn.create_volume_snapshot("volume-id", display_name="nightly", wait=False)
deleted = conn.delete_volume_snapshot("nightly")  # wait=False
deleted_after_wait = conn.delete_volume_snapshot("nightly", wait=True, timeout=600)
absent_input = conn.delete_volume_snapshot()  # False, no lookup
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L505)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L799)
- [pinned v3 Snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py)
- [Volume Snapshots](volume-snapshots.md)
