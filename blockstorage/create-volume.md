# Create Volume

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func createDefault(ctx context.Context, conn *sdk.Connection) (*blockstorage.CreateVolumeResult, error) {
    return conn.CreateVolume(ctx, blockstorage.CreateVolumeRequest{Size: 10},
        func(c *blockstorage.CreateVolumeOpts) error { v := "data"; c.Attributes.Name = &v; return nil },
        blockstorage.WithCreateVolumeMetadata(map[string]string{"owner": "worker"}))
}

func createWithoutWaiting(ctx context.Context, cinder *gophercloud.ServiceClient) (*blockstorage.CreateVolumeResult, error) {
    return blockstorage.CreateVolume(ctx, cinder, nil,
        blockstorage.CreateVolumeRequest{Size: 10},
        blockstorage.WithCreateVolumeWait(false))
}

func createBootableFromImage(ctx context.Context, conn *sdk.Connection) (*blockstorage.CreateVolumeResult, error) {
    image := resource.Name("ubuntu")
    return conn.CreateVolume(ctx,
        blockstorage.CreateVolumeRequest{Size: 20, Image: &image},
        func(c *blockstorage.CreateVolumeOpts) error { v := true; c.Bootable = &v; return nil })
}

func createWithPolicy(ctx context.Context, conn *sdk.Connection, progress func(int) error) (*blockstorage.CreateVolumeResult, error) {
    return conn.CreateVolume(ctx, blockstorage.CreateVolumeRequest{Size: 30},
        func(c *blockstorage.CreateVolumeOpts) error { v := "snapshot-id"; c.Attributes.SnapshotID = &v; return nil },
        blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{
            "volume_type_id": json.RawMessage(`"type-id"`),
            "is_multiattach": json.RawMessage(`false`),
        }),
        blockstorage.WithCreateVolumeSchedulerHints(map[string]json.RawMessage{
            "same_host": json.RawMessage(`["source-volume-id"]`),
        }),
        blockstorage.WithCreateVolumeWaitOptions(
            resource.WithTimeout(5*time.Minute), resource.WithPollInterval(time.Second),
            resource.WithProgressCallback(progress),
        ))
}
```

```python
volume = conn.create_volume(10, name="data", metadata={"owner": "worker"})
boot_volume = conn.create_volume(20, image="ubuntu", bootable=True)
```

## References

- [..](../README.md)
- [Volume SetBootable API](v3/README.md)
- [BlockStorageCloudMixin.create_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L175-L232)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L494-L503)
- [Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54-L144)
- [hint request preparation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L446-L464)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L102)
- [Attach Volume](attach-volume.md)
- [Detach Volume](detach-volume.md)
