# Volume Backup Import

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cloudName := os.Getenv("CLOUD_NAME")
    if cloudName == "" {
        return fmt.Errorf("CLOUD_NAME is required")
    }
    backupService, servicePresent := os.LookupEnv("BACKUP_SERVICE")
    backupURL, urlPresent := os.LookupEnv("BACKUP_URL")
    if !servicePresent || !urlPresent {
        return fmt.Errorf("BACKUP_SERVICE and BACKUP_URL must be defined; empty values are allowed")
    }
    path := os.Getenv("BACKUP_IMPORT_PATH")
    if path == "" {
        path = "connection"
    }
    switch path {
    case "connection", "direct", "service":
    default:
        return fmt.Errorf("unknown BACKUP_IMPORT_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.ImportVolumeBackupRequest{
        BackupService: backupService,
        BackupURL: backupURL,
    }
    var result *blockstorage.ImportVolumeBackupResult
    if path == "connection" {
        result, err = conn.ImportVolumeBackup(ctx, input)
    } else {
        cinder, selectErr := conn.BlockStorageV3(ctx)
        if selectErr != nil {
            return selectErr
        }
        if path == "direct" {
            result, err = blockstorage.ImportVolumeBackup(ctx, cinder.RawClient(), input)
        } else {
            result, err = cinder.Backups.ImportBackup(ctx, backupService, backupURL)
        }
    }
    inspect(result, err)
    return err
}

func inspect(result *blockstorage.ImportVolumeBackupResult, err error) {
    if result != nil {
        fmt.Printf("logical backup ID JSON: %s\n", result.BackupID)
        fmt.Printf("import microversion: %q\n", result.Microversion)
        for i, page := range result.Discovery {
            if page != nil {
                fmt.Println("admitted discovery response:", i, page.StatusCode, len(page.Body))
            }
        }
        if result.Applied != nil {
            fmt.Println("admitted import response:", result.Applied.StatusCode, len(result.Applied.Body))
        }
        if result.Backup != nil {
            actual := result.Backup.Clone()
            fmt.Println("actual selected fields:", len(actual.Body))
            fmt.Printf("actual response ID JSON: %s\n", actual.Body["id"])
        }
    }
    if err != nil {
        fmt.Println("import error:", err)
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("parent context interrupted import")
        }
        var physical *resource.ResponseError
        if errors.As(err, &physical) {
            fmt.Println("current admitted error response:", physical.StatusCode, len(physical.Body))
        }
        return
    }
    if result != nil && result.Value != nil {
        pretty, formatErr := json.MarshalIndent(result.Value, "", "  ")
        if formatErr == nil {
            fmt.Println("nullable Backup:", string(pretty))
        }
    }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
backup = conn.block_storage.import_backup(
    service="cinder.backup.drivers.swift.SwiftBackupDriver",
    url="opaque-record",
)
print(backup.id, backup.to_dict())
```

## References

- [`Proxy.import_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1802)
- [`Backup.import_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L146)
- [`Resource._get_microversion`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418)
- [`maximum_supported_microversion`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L358)
- [native Import](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L306)
- [Volume Backups](volume-backups.md)
- [Backup export record](volume-backup-export-record.md)
- [Backup restore/reset](volume-backup-actions.md)
