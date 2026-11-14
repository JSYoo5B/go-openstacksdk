# Volume Snapshots

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
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Configured(ctx context.Context, cloudName string) (*sdk.Connection, error) {
    return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func InspectPages(pages []*blockstorage.VolumeSnapshotsPage) {
    for _, page := range pages {
        fmt.Println("admitted snapshot page:", page.StatusCode, "bytes:", len(page.Body))
    }
}

func Inspect(result *blockstorage.GetVolumeSnapshotResult) {
    if result == nil { return }
    InspectPages(result.Pages)
    if result.Observed != nil {
        fmt.Println("member response:", result.Observed.StatusCode, len(result.Observed.Body))
    }
    fmt.Println("requested ID:", result.RequestedID, "seeded logical ID:", result.SeededID)
    if result.Snapshot != nil {
        owned := result.Snapshot.Clone()
        fmt.Println("actual response ID JSON:", string(owned.Body["id"]))
        var wire map[string]json.RawMessage
        if err := owned.Decode(&wire); err != nil {
            fmt.Println("optional wire projection:", err)
        } else {
            fmt.Println("wire fields:", len(wire))
        }
    }
}

func ViaConnection(ctx context.Context, conn *sdk.Connection) error {
    listed, err := conn.ListVolumeSnapshots(ctx,
        blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(
            `{"status":"available","volume_id":"volume-id","description":"nightly"}`)))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }
    fmt.Println("complete normalized list:", string(listed.Value), "raw rows:", len(listed.Snapshots))

    searched, err := conn.SearchVolumeSnapshots(ctx,
        blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(`{"status":"available"}`)))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }
    fmt.Println("selected normalized snapshots:", string(searched.Value))

    selected, err := conn.GetVolumeSnapshot(ctx,
        blockstorage.GetVolumeSnapshotRequest{NameOrID: "nightly"})
    Inspect(selected) // Physical proof remains useful when err is nonnil.
    if err != nil { return err }
    fmt.Println("selected normalized JSON:", string(selected.Value))

    byID, err := conn.GetVolumeSnapshotByID(ctx,
        blockstorage.GetVolumeSnapshotByIDRequest{ID: "snapshot-id"})
    Inspect(byID)
    return err
}

func ProjectNames(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.SearchVolumeSnapshots(ctx,
        blockstorage.SearchVolumeSnapshotsRequest{},
        blockstorage.WithVolumeSnapshotSearchExpression("[].name"))
    if result != nil { InspectPages(result.Pages) }
    if err != nil { return err }
    fmt.Println("arbitrary expression JSON:", string(result.Value))
    return nil
}

func FirstPage(ctx context.Context, conn *sdk.Connection) error {
    detailed, paginated, max := false, false, 10
    result, err := conn.ListVolumeSnapshots(ctx,
        blockstorage.WithVolumeSnapshotListOptions(blockstorage.VolumeSnapshotListOpts{
            Detailed: &detailed, Paginated: &paginated, MaxItems: &max,
            Headers: map[string]string{"X-Request-ID": "snapshot-read-example"},
        }))
    if result != nil { InspectPages(result.Pages) }
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    location := resource.CloudLocation{}
    listed, err := blockstorage.ListVolumeSnapshots(ctx, cinder,
        request.Custom(func(c *blockstorage.VolumeSnapshotListOpts) error { loc := location.Clone(); c.Location = &loc; return nil }))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }

    searched, err := blockstorage.SearchVolumeSnapshots(ctx, cinder,
        blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "nightly*"},
        func(c *blockstorage.VolumeSnapshotSearchOpts) error { loc := location.Clone(); c.Location = &loc; return nil })
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }

    // Explicit {} filters choose full Search, rather than ordinary GET-first find.
    selected, err := blockstorage.GetVolumeSnapshot(ctx, cinder,
        blockstorage.GetVolumeSnapshotRequest{NameOrID: "nightly"},
        blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(`{}`)))
    Inspect(selected)
    if err != nil { return err }

    byID, err := blockstorage.GetVolumeSnapshotByID(ctx, cinder,
        blockstorage.GetVolumeSnapshotByIDRequest{ID: "snapshot-id"},
        blockstorage.WithVolumeSnapshotReadOptions(blockstorage.VolumeSnapshotReadOpts{Location: &location}))
    Inspect(byID)
    return err
}
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L546)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L710)
- [Search helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L864)
- [Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2165)
- [Proxy.get_snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L119)
- [pinned Snapshot descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py#L62)
- [Fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86)
- [Utils](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)
- [Requests Response.links](https://requests.readthedocs.io/en/latest/_modules/requests/models/#Response.links)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2256)
- [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Microversions](../docs/microversions.md)
- [Snapshots](v3/snapshots/README.md)
- [Metadata](metadata/README.md)
