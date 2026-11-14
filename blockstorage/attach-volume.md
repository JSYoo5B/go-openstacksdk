# Attach Volume

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func attachDefault(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.AttachVolumeResult, error) {
    return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
        Server: resource.ID(serverID), Volume: resource.ID(volumeID),
    })
}

func attachWithoutWaiting(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.AttachVolumeResult, error) {
    return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
        Server: resource.ID(serverID), Volume: resource.ID(volumeID),
    }, func(c *blockstorage.AttachVolumeOpts) error { c.Device = "/dev/vdb"; return nil },
        blockstorage.WithAttachVolumeWait(false))
}

func attachWithDeadline(ctx context.Context, conn *sdk.Connection) (*blockstorage.AttachVolumeResult, error) {
    return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
        Server: resource.Name("worker"), Volume: resource.Name("data"),
    }, blockstorage.WithAttachVolumeWaitOptions(
        resource.WithTimeout(5*time.Minute), resource.WithPollInterval(time.Second),
    ))
}

func attachWithClients(ctx context.Context, nova, cinder *gophercloud.ServiceClient, input blockstorage.AttachVolumeRequest) (*blockstorage.AttachVolumeResult, error) {
    return blockstorage.AttachVolume(ctx, nova, cinder, input)
}
```

```python
server = conn.compute.get_server("server-id")
volume = conn.block_storage.get_volume("volume-id")
attachment = conn.attach_volume(server, volume)  # wait=True, timeout=None
```

## References

- [..](../README.md)
- [Nova VolumeAttachment API](../compute/v2/README.md)
- [BlockStorageCloudMixin.attach_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L431-L492)
- [Cinder wait_for_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3815)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591-L2667)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L102)
