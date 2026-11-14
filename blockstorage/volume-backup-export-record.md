# Volume Backup Export Record

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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Inspect(response *blockstorage.VolumeBackupExportResponse, value json.RawMessage, err error) {
    if response != nil {
        fmt.Println("export HTTP:", response.StatusCode, "bytes:", len(response.Body))
        fmt.Println("content type:", response.Header.Get("Content-Type"))
    }
    if err != nil {
        fmt.Println("export error:", err)
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("parent context interrupted export")
        }
        var physical *resource.ResponseError
        if errors.As(err, &physical) {
            fmt.Println("accepted error proof:", physical.StatusCode, len(physical.Body))
        }
        return
    }
    if value != nil {
        // Literal null is successful nonnil bytes "null"; nil means no Value.
        fmt.Printf("JSON record: %s\n", value)
    }
}

func ConfiguredRecord(parent context.Context, cloudName, backupID string) (*blockstorage.ExportVolumeBackupRecordResult, error) {
    ctx, cancel := context.WithTimeout(parent, 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return nil, err }
    result, err := conn.ExportVolumeBackupRecord(ctx,
        blockstorage.ExportVolumeBackupRecordRequest{BackupID: backupID})
    if result != nil { Inspect(result.Exported, result.Value, err) }
    return result, err
}

func ServiceRecord(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.ExportBackupResult, error) {
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    result, err := cinder.Backups.ExportBackup(ctx, backupID)
    if result != nil { Inspect(result.Exported, result.Value, err) }
    return result, err
}

func ServiceRaw(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.ExportRecordResult, error) {
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    result, err := cinder.Backups.ExportRecord(ctx, backupID)
    if result != nil { Inspect(result.Exported, nil, err) }
    return result, err
}

func DirectRecord(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*blockstorage.ExportVolumeBackupRecordResult, error) {
    result, err := blockstorage.ExportVolumeBackupRecord(ctx, cinder,
        blockstorage.ExportVolumeBackupRecordRequest{BackupID: backupID})
    if result != nil { Inspect(result.Exported, result.Value, err) }
    return result, err
}

func DirectServiceRecord(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*backups.ExportBackupResult, error) {
    return backups.New(cinder).ExportBackup(ctx, backupID)
}

// Native typed Export is an alternative for the native BackupRecord schema.
func NativeTyped(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.BackupRecord, error) {
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    return cinder.Backups.Export(ctx, backupID)
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack", block_storage_api_version="3")
value = conn.block_storage.export_backup(backup_id)
print(value)

response = conn.block_storage.export_record(backup_id)
print(response.status_code, response.headers, response.content)
```

## References

- [`Proxy.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1824)
- [`Proxy.export_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1812)
- [`Backup.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L173)
- [`Backup.export`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L184)
- [Response.json](https://github.com/psf/requests/blob/v2.32.5/src/requests/models.py#L947)
- [compat](https://github.com/psf/requests/blob/v2.32.5/src/requests/compat.py#L57-L64)
- [guess_json_utf](https://github.com/psf/requests/blob/v2.32.5/src/requests/utils.py#L947)
- [`backups.Export`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L281)
- [`BackupRecord`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/results.go#L201)
- [Cloud raw export](volume-backup-export.md)
- [Volume Backups](volume-backups.md)
- [Volume Backup Mutations](volume-backup-mutations.md)
- [..](../README.md)
