# Volume Limits

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
    nativeLimits "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/limits"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Inspect(result *blockstorage.GetVolumeLimitsResult) {
    if result == nil { return }
    if result.Project != nil {
        fmt.Println("project ID JSON:", string(result.Project.ID),
            "seeded:", result.Project.SeededID,
            "lookup pages:", len(result.Project.Pages))
        if result.Project.Observed != nil {
            fmt.Println("project member status:", result.Project.Observed.StatusCode)
        }
    }
    fmt.Println("requested project ID JSON:", string(result.RequestedProjectID))
    if result.Observed != nil {
        fmt.Println("limits status:", result.Observed.StatusCode,
            "raw bytes:", len(result.Observed.Body))
    }
}

func Current(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.GetVolumeLimits(ctx, blockstorage.GetVolumeLimitsRequest{})
    Inspect(result) // Completed physical proof also remains available on error.
    if err != nil { return err }
    fmt.Println("normalized nullable JSON:", string(result.Value))
    owned := result.Limits.Clone()
    fmt.Println("original absolute JSON:", string(owned.Body["absolute"]))

    // This optional native projection enforces the native int/string schema.
    // A projection error does not invalidate the successful raw workflow.
    var projection nativeLimits.Limits
    if projectionErr := owned.Decode(&projection); projectionErr != nil {
        fmt.Println("native projection:", projectionErr)
    } else {
        fmt.Println("native maximum volumes:", projection.Absolute.MaxTotalVolumes)
    }
    return nil
}

func OtherProject(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    result, err := conn.GetVolumeLimits(ctx,
        blockstorage.GetVolumeLimitsRequest{NameOrID: nameOrID})
    Inspect(result)
    if err != nil { return err }
    fmt.Println("normalized limits:", string(result.Value))
    return nil
}

func WithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "authenticated-project"
    location := resource.CloudLocation{
        Cloud: &cloud, Zone: json.RawMessage(`"owned-zone"`),
        Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
    }
    result, err := conn.GetVolumeLimits(ctx,
        blockstorage.GetVolumeLimitsRequest{NameOrID: "other-project"},
        blockstorage.WithGetVolumeLimitsOptions(blockstorage.GetVolumeLimitsOpts{
            Location: &location,
        }))
    Inspect(result)
    if err != nil { return err }
    fmt.Println("limits with owned authenticated location:", string(result.Value))
    return nil
}

func Direct(ctx context.Context, cinder, identity *gophercloud.ServiceClient) error {
    // Empty input needs no Identity client and sends no project filter.
    current, err := blockstorage.GetVolumeLimits(ctx, cinder, nil,
        blockstorage.GetVolumeLimitsRequest{})
    Inspect(current)
    if err != nil { return err }
    selected, err := blockstorage.GetVolumeLimits(ctx, cinder, identity,
        blockstorage.GetVolumeLimitsRequest{NameOrID: "project-name-or-id"},
        func(c *blockstorage.GetVolumeLimitsOpts) error { c.Location = &resource.CloudLocation{}; return nil })
    Inspect(selected)
    return err
}
```

## References

- [pinned cloud get_volume_limits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L348)
- [pinned Cinder Proxy.get_limits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1918)
- [Resource paging](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2345)
- [Utils](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)
- [Models](https://requests.readthedocs.io/en/latest/_modules/requests/models/#RequestEncodingMixin._encode_params)
- [Urllib.Parse](https://docs.python.org/3/library/urllib.parse.html#urllib.parse.urlencode)
- [Resource._get_microversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418)
- [Cinder v3 limits singleton](v3/limits/README.md)
- [pinned Limits hierarchy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/limits.py)
- [pinned field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86)
- [Resource constructor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538)
