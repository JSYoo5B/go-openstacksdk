# Backups

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
)

func clearBackupMetadata(ctx context.Context, cinder *v3.Service, backupID string) (*backups.Backup, error) {
    opts := backups.UpdateOpts{
        Metadata: map[string]string{},
    }
    return cinder.Backups.Update(ctx, backupID, backups.UpdateOpts{},
        backups.WithUpdateOptions(opts),
    )
}
```

## References

- [Volume Backup Export Record](../../volume-backup-export-record.md)
- [Volume Backup Actions](../../volume-backup-actions.md)
- [Volume Backup Import](../../volume-backup-import.md)
