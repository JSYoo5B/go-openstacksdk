# Volume Reads

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

func ReadVolumes(ctx context.Context, conn *sdk.Connection, id, name string) error {
    listed, err := conn.ListVolumes(ctx)
    if err != nil {
        if listed != nil { fmt.Println("admitted list pages:", len(listed.Pages)) }
        return err
    }
    fmt.Println(string(listed.Value), len(listed.Volumes))

    found, err := conn.GetVolumeByID(ctx,
        blockstorage.GetVolumeByIDRequest{ID: id})
    if err != nil {
        if found != nil && found.Observed != nil {
            fmt.Println("admitted member response:", found.Observed.StatusCode)
        }
        return err
    }
    fmt.Println(string(found.Value), string(found.Volume.Body["id"]))
    copy := found.Volume.Clone()
    var projection struct { ID *string `json:"id"` }
    if err := copy.Decode(&projection); err != nil { return err }

    exists, err := conn.VolumeExists(ctx,
        blockstorage.VolumeExistsRequest{NameOrID: name})
    if err != nil {
        if exists != nil { fmt.Println("admitted fallback pages:", len(exists.Pages)) }
        return err
    }
    if exists.Exists == nil { return fmt.Errorf("missing completed existence result") }
    fmt.Println("exists:", *exists.Exists)
    return nil
}

func ListWithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud,
        Project: resource.CloudProject{
            ID: json.RawMessage(`"project-id"`), Name: &project,
        },
    }
    result, err := conn.ListVolumes(ctx, func(c *blockstorage.VolumeReadOpts) error { loc := location.Clone(); c.Location = &loc; return nil })
    if err != nil { return err }
    fmt.Println(string(result.Value))
    return nil
}

func ReadDirect(ctx context.Context, cinder *gophercloud.ServiceClient, id string) error {
    listed, err := blockstorage.ListVolumes(ctx, cinder)
    if err != nil { return err }
    fmt.Println(len(listed.Volumes), len(listed.Pages))
    volume, err := blockstorage.GetVolumeByID(ctx, cinder,
        blockstorage.GetVolumeByIDRequest{ID: id})
    if err != nil { return err }
    exists, err := blockstorage.VolumeExists(ctx, cinder,
        blockstorage.VolumeExistsRequest{NameOrID: id})
    if err != nil { return err }
    fmt.Println(string(volume.Value), *exists.Exists)
    return nil
}
```

```python
import openstack

conn = openstack.connect(cloud="example")
volumes = conn.list_volumes()
volume = conn.get_volume_by_id("volume-id")
exists = conn.volume_exists("data")

volumes = conn.list_volumes(cache=False)
```

## References

- [Volume Id](volume-id.md)
- [Search Volumes](search-volumes.md)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py)
- [v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [v3 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py)
