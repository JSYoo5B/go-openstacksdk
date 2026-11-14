# Volume Mutations

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

func InspectUpdate(result *blockstorage.UpdateVolumeResult) {
    if result == nil { return }
    fmt.Println("fixed mutation ID:", result.VolumeID)
    if result.Resolved != nil {
        fmt.Println("lookup pages:", len(result.Resolved.Pages))
        if result.Resolved.Observed != nil {
            fmt.Println("lookup status:", result.Resolved.Observed.StatusCode)
        }
    }
    if result.Applied != nil {
        fmt.Println("actual PUT response:", result.Applied.StatusCode,
            "raw bytes:", len(result.Applied.Body))
    }
}

func Update(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.UpdateVolume(ctx,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"},
        func(c *blockstorage.UpdateVolumeOpts) error { v := "renamed"; c.Attributes.Name = &v; return nil },
        blockstorage.WithUpdateVolumeMetadata(map[string]string{}),
        blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{
            "image_id": json.RawMessage(`"image-id"`),
            "scheduler_hints": json.RawMessage(`{"same_host":["peer-volume"]}`),
        }))
    InspectUpdate(result)
    if err != nil { return err }
    fmt.Println("merged normalized JSON:", string(result.Value))
    owned := result.Volume.Clone()
    fmt.Println("merged raw metadata:", string(owned.Body["metadata"]))

    // Empty kwargs still resolve the volume but do not send a PUT.
    unchanged, err := conn.UpdateVolume(ctx,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"})
    InspectUpdate(unchanged)
    if err != nil { return err }
    fmt.Println("no PUT:", unchanged.Applied == nil)
    return nil
}

func Bootable(ctx context.Context, conn *sdk.Connection) error {
    input := blockstorage.SetVolumeBootableRequest{NameOrID: "data"}
    enabled, err := conn.SetVolumeBootable(ctx, input) // Default: true.
    if enabled != nil && enabled.Applied != nil {
        fmt.Println("enable acknowledgement:", enabled.Applied.StatusCode)
    }
    if err != nil { return err }
    disabled, err := conn.SetVolumeBootable(ctx, input,
        func(c *blockstorage.SetVolumeBootableOpts) error { v := false; c.Bootable = &v; return nil })
    if disabled != nil && disabled.Applied != nil {
        fmt.Println("disable acknowledgement:", disabled.Applied.StatusCode)
    }
    return err
}

func UpdateWithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "configured-project"
    name := "renamed"
    location := resource.CloudLocation{
        Cloud: &cloud,
        Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
    }
    result, err := conn.UpdateVolume(ctx,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"},
        blockstorage.WithUpdateVolumeOptions(blockstorage.UpdateVolumeOpts{
            Attributes: blockstorage.UpdateVolumeAttributes{Name: &name},
            Location: &location,
        }))
    InspectUpdate(result)
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    result, err := blockstorage.UpdateVolume(ctx, cinder,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"},
        func(c *blockstorage.UpdateVolumeOpts) error { v := "updated"; c.Attributes.Description = &v; return nil })
    InspectUpdate(result)
    if err != nil { return err }
    _, err = blockstorage.SetVolumeBootable(ctx, cinder,
        blockstorage.SetVolumeBootableRequest{NameOrID: "data"},
        func(c *blockstorage.SetVolumeBootableOpts) error { v := false; c.Bootable = &v; return nil })
    return err
}
```

```python
updated = conn.update_volume("data", name="renamed", metadata={})
conn.set_volume_bootable("data")
conn.set_volume_bootable("data", False)
```

## References

- [pinned cloud update_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L234)
- [set_volume_bootable](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L254)
- [Volume Reads](volume-reads.md)
- [pinned _get_volume_kwargs](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L494)
- [pinned Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [Search Volumes](search-volumes.md)
- [pinned Volume request body](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L446)
- [pinned response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338)
- [pinned Volume bootable action](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L177)
