# Service Waits

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Shared rules

**Common waits.** `Collection.Wait` polls until the status equals the target, compared case-insensitively, and `Collection.WaitDeleted` polls until a 404 or a `deleted` status. Both default to a five-minute timeout and a two-second interval. A resource that disappears during `Wait` ends it with `resource.FailedStateError` for status `missing`.

**Options.** Every wait takes `resource.WaitOption` values:

- `WithTimeout` and `WithUnlimitedWait` set the time budget;
- `WithPollInterval` sets a fixed interval, and `WithWaitStrategy` picks a fixed, linear or exponential schedule with optional jitter (a Go-only extension; Python waits use a fixed interval);
- `WithFailureStates` replaces the failure statuses, and `WithStatusAttribute` selects another string field as the status;
- `WithProgressCallback` runs a `func(int) error` after each nonterminal poll, and a returned error stops the wait; `WithProgress` sends the same value on a caller-owned channel, blocking until it is received or the context ends, never closing it and sending nothing after the wait returns. With both set, the callback runs first and an error suppresses that step's send.

Later options override earlier ones.

**Service waits.** Service packages add Python-style helpers with their own defaults: state waits such as `WaitForState` fail on the exact `ERROR` or `error` status and have no SDK timeout, deletion waits such as `WaitForDelete` stop after 120 seconds, and both poll every two seconds. Compute `WaitForServer` waits for `ACTIVE` for up to 120 seconds; Cinder `WaitForAvailable` waits for `available`. Caller options apply after these defaults.

**Operation waits.** Operations that wait as one of their steps, such as creating a volume or an image, take their wait settings as `With<Op>Wait(...resource.WaitOption)` or a `Wait`/`WaitOptions` field. `resource.ResolveWait` applies the operation's own defaults, and a control the operation does not honor (failure states, status attribute or progress) fails with `resource.ErrInvalidOption` before any request.

**Cancellation.** A done context ends the current pause and the wait immediately. Cancelling a wait never rolls back the remote operation, and a wait never infers completion from an accepted request.

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
