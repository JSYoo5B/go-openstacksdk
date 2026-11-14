# Volume Backup Export

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
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func Inspect(result *blockstorage.ExportVolumeBackupResult, err error) {
    if result != nil {
        fmt.Println("requested backup ID:", result.BackupID)
        if result.Exported != nil {
            response := result.Exported
            fmt.Println("export HTTP:", response.StatusCode, "bytes:", len(response.Body))
            fmt.Println("content type:", response.Header.Get("Content-Type"))
        }
    }
    if err == nil { return }
    fmt.Println("export error:", err)
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("parent context interrupted export")
    }
    if errors.Is(err, resource.ErrInvalidOption) {
        fmt.Println("invalid local input or changed request source")
    }
}

func ConfiguredExport(parent context.Context, cloudName, backupID string) (*blockstorage.ExportVolumeBackupResult, error) {
    ctx, cancel := context.WithTimeout(parent, 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return nil, err }
    result, err := conn.ExportVolumeBackup(ctx,
        blockstorage.ExportVolumeBackupRequest{BackupID: backupID})
    Inspect(result, err)
    return result, err
}

func DirectExport(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*blockstorage.ExportVolumeBackupResult, error) {
    result, err := blockstorage.ExportVolumeBackup(ctx, cinder,
        blockstorage.ExportVolumeBackupRequest{BackupID: backupID})
    Inspect(result, err)
    return result, err
}

// Optional application policy: call after a successful export only when
// the application expects JSON. An opaque export need not contain JSON.
func ApplicationJSON(result *blockstorage.ExportVolumeBackupResult) (json.RawMessage, error) {
    if result == nil || result.Exported == nil {
        return nil, fmt.Errorf("no admitted export response")
    }
    var value json.RawMessage
    if err := json.Unmarshal(result.Exported.Body, &value); err != nil {
        return nil, err
    }
    return value, nil
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack", block_storage_api_version="3")
response = conn.export_volume_backup(backup_id)
print(response.status_code, response.headers, response.content)
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L648)
- [deprecated v3 `Proxy.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1824)
- [`Backup.export`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L184)
- [`Proxy.export_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1812)
- [`Backup.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L173)
- [JSON result](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L281)
- [typed `BackupRecord`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/results.go#L201)
- [Volume Backups](volume-backups.md)
- [Volume Backup Mutations](volume-backup-mutations.md)
- [..](../README.md)
