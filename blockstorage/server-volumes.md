# Server Volumes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func getForServer(ctx context.Context, conn *sdk.Connection, serverID string) error {
    result, err := conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{ServerID: serverID})
    return inspectResult(result, err)
}

func getWithClient(ctx context.Context, cinder *gophercloud.ServiceClient, serverID string) (*blockstorage.GetVolumesResult, error) {
    return blockstorage.GetVolumes(ctx, cinder,
        blockstorage.GetVolumesRequest{ServerID: serverID})
}

func getForRawServer(ctx context.Context, conn *sdk.Connection) (*blockstorage.GetVolumesResult, error) {
    // An explicit null ID can match a null attachment server_id.
    server := map[string]json.RawMessage{"id": json.RawMessage(`null`)}
    return conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{},
        blockstorage.WithGetVolumesServerFields(server))
}

func prepareServerFields(ctx context.Context, server map[string]json.RawMessage) (blockstorage.GetVolumesOpts, error) {
    // Runs options locally once and owns their maps/raw bytes; no HTTP.
    return blockstorage.PrepareGetVolumesOptions(ctx,
        blockstorage.WithGetVolumesServerFields(server))
}

func getWithPreparedOptions(ctx context.Context, conn *sdk.Connection, prepared blockstorage.GetVolumesOpts) (*blockstorage.GetVolumesResult, error) {
    return conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{},
        blockstorage.WithGetVolumesOptions(prepared))
}

func inspectResult(result *blockstorage.GetVolumesResult, callErr error) error {
    if callErr != nil {
        if result != nil {
            // Volumes is nil on every workflow error. Pages is actual response evidence.
            for i, page := range result.Pages {
                fmt.Printf("accepted page %d: status=%d request=%q bytes=%d\n",
                    i, page.StatusCode, page.Header.Get("X-Openstack-Request-Id"), len(page.Body))
            }
        }
        return callErr
    }
    for i, volume := range result.Volumes {
        fmt.Printf("match %d: raw id=%s raw status=%s\n", i,
            volume.Body["id"], volume.Body["status"])

        var projected blockstorage.VolumeInfo
        if err := volume.Decode(&projected); err != nil {
            // Selection succeeded; this optional stricter typed projection failed.
            fmt.Printf("typed projection %d: %v\n", i, err)
        } else {
            fmt.Printf("typed projection %d: status=%d request=%q\n", i,
                projected.StatusCode, projected.Header.Get("X-Openstack-Request-Id"))
        }

        privateCopy := volume.Clone()
        privateCopy.Body["name"] = json.RawMessage(`"private-copy"`)
        fmt.Printf("copy %d name=%s; original name=%s\n", i,
            privateCopy.Body["name"], volume.Body["name"])
    }
    return nil
}

func independentMatches(result *blockstorage.GetVolumesResult) []*resource.RawResource {
    copies := make([]*resource.RawResource, len(result.Volumes))
    for i, volume := range result.Volumes {
        copies[i] = volume.Clone()
    }
    return copies
}
```

```python
server = {"id": "server-id"}
volumes = conn.get_volumes(server)
for volume in volumes:
    print(volume["id"])
```

## References

- [..](../README.md)
- [Volume Attachment Device](volume-attachment-device.md)
- [Reference](README.md)
- [get_volumes](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L323-L346)
- [list_volumes](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L52-L64)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L890-L921)
- [attachments descriptor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54)
- [list conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L111)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2340-L2359)
- [VolumePage.NextPageURL](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/results.go#L126-L135)
