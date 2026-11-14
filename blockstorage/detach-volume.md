# Detach Volume

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

func detachDefault(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.DetachVolumeResult, error) {
    return conn.DetachVolume(ctx, blockstorage.DetachVolumeRequest{
        Server: resource.ID(serverID), Volume: resource.ID(volumeID),
    })
}

func detachNovaOnly(ctx context.Context, nova *gophercloud.ServiceClient, serverID, volumeID string) (*blockstorage.DetachVolumeResult, error) {
    return blockstorage.DetachVolume(ctx, nova, nil,
        blockstorage.DetachVolumeRequest{
            Server: resource.ID(serverID), Volume: resource.ID(volumeID),
        }, blockstorage.WithDetachVolumeWait(false))
}

func detachWithDeadline(ctx context.Context, conn *sdk.Connection, input blockstorage.DetachVolumeRequest, progress func(int) error) (*blockstorage.DetachVolumeResult, error) {
    return conn.DetachVolume(ctx, input,
        blockstorage.WithDetachVolumeWaitOptions(
            resource.WithTimeout(5*time.Minute), resource.WithPollInterval(time.Second),
            resource.WithProgressCallback(progress),
        ))
}

func detachWithClients(ctx context.Context, nova, cinder *gophercloud.ServiceClient) (*blockstorage.DetachVolumeResult, error) {
    return blockstorage.DetachVolume(ctx, nova, cinder,
        blockstorage.DetachVolumeRequest{
            Server: resource.Name("worker"), Volume: resource.Name("data"),
        })
}
```

```python
server = conn.compute.get_server("server-id")
volume = conn.block_storage.get_volume("volume-id")
conn.detach_volume(server, volume)  # wait=True; success returns None
```

## References

- [..](../README.md)
- [BlockStorageCloudMixin.detach_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L398-L429)
- [Compute delete_volume_attachment](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2542-L2578)
- [Cinder waiter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3815)
- [Resource waiter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591-L2667)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2502-L2540)
- [cloud get_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L82-L122)
- [Attach Volume](attach-volume.md)
