# Service Waits

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func waitForServer(ctx context.Context, client *gophercloud.ServiceClient, id string) (*servers.Server, error) {
    return servers.New(client).WaitForServer(ctx, resource.ID(id),
        resource.WithTimeout(time.Minute))
}
```

```python
server = conn.compute.wait_for_server(server, wait=60)
```

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func waitForVolume(ctx context.Context, client *gophercloud.ServiceClient, id string) (*volumes.Volume, error) {
    bounded, cancel := context.WithTimeout(ctx, 10*time.Minute)
    defer cancel()
    return volumes.New(client).WaitForAvailable(bounded, resource.ID(id))
}

func waitForVolumeDeletion(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
    return volumes.New(client).WaitForDelete(ctx, resource.ID(id),
        resource.WithUnlimitedWait())
}
```

```python
volume = conn.block_storage.wait_for_status(volume)  # available, wait=None
conn.block_storage.wait_for_delete(volume, wait=None)
```

## References

- [Image Record Waits](../image/image-record-waits.md)
- [Server Ready](../compute/server-ready.md)
- [Resource](../resource/README.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L3244-L3349)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2373-L2435)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3837)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2299-L2358)
- [Tasks](../image/v2/tasks/README.md)
