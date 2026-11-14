# Volume Backups

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "errors"
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

func InspectPages(pages []*blockstorage.VolumeBackupsPage) {
    for _, page := range pages {
        fmt.Println("admitted backup page:", page.StatusCode, "bytes:", len(page.Body))
    }
}

func Explain(err error) {
    if err == nil { return }
    var ambiguous *blockstorage.VolumeBackupSelectionError
    if errors.As(err, &ambiguous) {
        fmt.Println("filtered selection length:", ambiguous.Length)
    }
    fmt.Println("ambiguous:", errors.Is(err, resource.ErrAmbiguous), "error:", err)
}

func Inspect(result *blockstorage.GetVolumeBackupResult) {
    if result == nil { return }
    InspectPages(result.Pages)
    if result.Observed != nil {
        fmt.Println("member response:", result.Observed.StatusCode, len(result.Observed.Body))
    }
    fmt.Println("requested ID:", result.RequestedID, "seeded logical ID:", result.SeededID)
    if result.Backup != nil {
        owned := result.Backup.Clone()
        fmt.Println("actual wire ID JSON:", string(owned.Body["id"]))
        var projection struct {
            ID json.RawMessage `json:"id"`
            Name *string `json:"name"`
        }
        if err := owned.Decode(&projection); err != nil {
            fmt.Println("optional wire projection:", err)
        } else {
            fmt.Println("projected wire ID:", string(projection.ID))
        }
    }
}

func ViaConnection(ctx context.Context, conn *sdk.Connection) error {
    listed, err := conn.ListVolumeBackups(ctx,
        blockstorage.WithVolumeBackupListFilters(json.RawMessage(
            `{"status":"available","volume_id":"volume-id","description":"nightly"}`)))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }
    fmt.Println("complete normalized list:", string(listed.Value), "raw backups:", len(listed.Backups))

    searched, err := conn.SearchVolumeBackups(ctx,
        blockstorage.SearchVolumeBackupsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{"status":"available"}`)))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }
    fmt.Println("selected normalized backups:", string(searched.Value))

    selected, err := conn.GetVolumeBackup(ctx,
        blockstorage.GetVolumeBackupRequest{NameOrID: "nightly"})
    Inspect(selected) // Inspect actual proof before deciding what to do with err.
    Explain(err)
    if err != nil { return err }
    if selected.Value == nil {
        fmt.Println("completed lookup: no backup found")
    } else {
        fmt.Println("selected normalized JSON:", string(selected.Value))
    }
    return nil
}

func PreparedFirstPage(ctx context.Context, conn *sdk.Connection) error {
    calls := 0
    policy, err := blockstorage.PrepareVolumeBackupListOptions(ctx,
        request.Custom(func(value *blockstorage.VolumeBackupListOpts) error {
            calls++
            filters := json.RawMessage(`{"status":"available"}`)
            value.Filters = &filters
            return nil
        }),
        blockstorage.WithVolumeBackupListDetailed(false),
        request.Paginated(false),
        request.MaxItems(10),
        request.Headers(map[string]string{
            "X-Request-ID": "backup-read-example",
        }))
    if err != nil { return err }
    fmt.Println("original callback calls:", calls)
    result, err := conn.ListVolumeBackups(ctx,
        blockstorage.WithVolumeBackupListOptions(policy))
    if result != nil { InspectPages(result.Pages) }
    return err
}

func ProjectNames(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.SearchVolumeBackups(ctx,
        blockstorage.SearchVolumeBackupsRequest{},
        blockstorage.WithVolumeBackupSearchExpression("[].name"))
    if result != nil { InspectPages(result.Pages) }
    if err != nil { return err }
    fmt.Println("arbitrary expression JSON:", string(result.Value), "raw associations:", len(result.Backups))
    return nil
}

func FilteredGet(ctx context.Context, conn *sdk.Connection, name string) error {
    // Explicit {} chooses complete Search instead of ordinary GET-first find.
    result, err := conn.GetVolumeBackup(ctx,
        blockstorage.GetVolumeBackupRequest{NameOrID: name},
        blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{}`)))
    Inspect(result)
    Explain(err)
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    cloud, region, projectName := "owned-cloud", "RegionOne", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud, RegionName: &region,
        Project: resource.CloudProject{ID: json.RawMessage(`"project-id"`), Name: &projectName},
    }
    listed, err := blockstorage.ListVolumeBackups(ctx, cinder,
        request.Custom(func(c *blockstorage.VolumeBackupListOpts) error { loc := location.Clone(); c.Location = &loc; return nil }))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }

    policy, err := blockstorage.PrepareVolumeBackupSearchOptions(ctx,
        func(c *blockstorage.VolumeBackupSearchOpts) error { loc := location.Clone(); c.Location = &loc; return nil },
        blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{"status":"available"}`)))
    if err != nil { return err }
    searched, err := blockstorage.SearchVolumeBackups(ctx, cinder,
        blockstorage.SearchVolumeBackupsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeBackupSearchOptions(policy))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }

    selected, err := blockstorage.GetVolumeBackup(ctx, cinder,
        blockstorage.GetVolumeBackupRequest{NameOrID: "nightly"},
        func(c *blockstorage.VolumeBackupSearchOpts) error { loc := location.Clone(); c.Location = &loc; return nil })
    Inspect(selected)
    return err
}
```

```python
import openstack

conn = openstack.connect(cloud="my-cloud")
backups = conn.list_volume_backups(
    detailed=True,
    filters={"status": "available", "description": "nightly"},
)
matches = conn.search_volume_backups("nightly*", {"status": "available"})
backup = conn.get_volume_backup("nightly")
filtered_backup = conn.get_volume_backup("nightly", filters={})
names = conn.search_volume_backups(filters="[].name")
```

## References

- [List](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L733)
- [Search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L890)
- [Get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L668)
- [Proxy.backups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1631)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L540)
- [v3 Backup descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L26)
- [Requests parser](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Microversions](../docs/microversions.md)
- [V3](v3/README.md)
