# Volume Id

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

func ReadVolumeID(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    result, err := conn.GetVolumeID(ctx,
        blockstorage.GetVolumeIDRequest{NameOrID: nameOrID})
    if err != nil {
        if result != nil {
            fmt.Println("admitted fallback pages:", len(result.Pages))
            if result.Observed != nil {
                fmt.Println("admitted member status:", result.Observed.StatusCode)
            }
        }
        return err
    }
    if result.ID == nil {
        fmt.Println("volume absent")
        return nil
    }
    fmt.Println("ID JSON:", string(result.ID))
    fmt.Println("normalized volume:", string(result.Value))
    copy := result.Volume.Clone()
    fmt.Println("original ID field:", string(copy.Body["id"]))
    return nil
}

func ReadVolumeIDWithLocation(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    project := "configured-project"
    location := resource.CloudLocation{
        Project: resource.CloudProject{
            ID: json.RawMessage(`"scope-id"`), Name: &project,
        },
    }
    result, err := conn.GetVolumeID(ctx,
        blockstorage.GetVolumeIDRequest{NameOrID: nameOrID},
        func(c *blockstorage.VolumeReadOpts) error { loc := location.Clone(); c.Location = &loc; return nil })
    if err != nil { return err }
    fmt.Println(string(result.ID))
    return nil
}

func ReadVolumeIDDirect(ctx context.Context, cinder *gophercloud.ServiceClient, nameOrID string) error {
    result, err := blockstorage.GetVolumeID(ctx, cinder,
        blockstorage.GetVolumeIDRequest{NameOrID: nameOrID})
    if err != nil { return err }
    if result.ID != nil { fmt.Println(string(result.ID)) }
    return nil
}
```

```python
import openstack

conn = openstack.connect(cloud="example")
volume_id = conn.get_volume_id("data")
if volume_id is None:
    print("no ID returned")
else:
    print(volume_id)
```

## References

- [Volume Reads](volume-reads.md)
- [Search Volumes](search-volumes.md)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L362-L371)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L82-L122)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
