# Blockstorage

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
volume = conn.block_storage.find_volume("data", ignore_missing=False)
for volume in conn.block_storage.volumes(status="available"):
    print(volume.id, volume.size)
```

See `Example_volumesFind` in [example_blockstorage_fragment1_test.go](example_blockstorage_fragment1_test.go), compiled by `go test`.

See `Example_waitAndDelete` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [..](../README.md)
- [Block Storage V3](https://docs.openstack.org/openstacksdk/latest/user/proxies/block_storage_v3.html)
- [Volume Mutations](volume-mutations.md)
- [Microversions](../docs/microversions.md)
- [Block Storage v3 API](v3/README.md)
- [Compute](../compute/README.md)
- [Attach Volume](attach-volume.md)
- [Detach Volume](detach-volume.md)
- [Create Volume](create-volume.md)
- [Snapshots](v2/snapshots/README.md)
- [Snapshots](v3/snapshots/README.md)
- [blockstorage_test.go](blockstorage_test.go)
- [Collections Test](../connectiontest/collections_test.go)
- [Metadata](metadata/README.md)
- [v3 Volume](v3/volumes/README.md)
- [Snapshot](v3/snapshots/README.md)
- [Service Waits](../docs/service-waits.md)
- [Delete Volume](delete-volume.md)
- [Volume Attachment Device](volume-attachment-device.md)
- [Server Volumes](server-volumes.md)
- [Search Volumes](search-volumes.md)
- [Volume Reads](volume-reads.md)
- [Volume Id](volume-id.md)
- [Volume Types](volume-types.md)
- [Volume Type Access](volume-type-access.md)
- [Volume Limits](volume-limits.md)
- [Volume Snapshots](volume-snapshots.md)
- [Volume Snapshot Mutations](volume-snapshot-mutations.md)
- [Volume Backups](volume-backups.md)
- [Volume Backup Mutations](volume-backup-mutations.md)
- [Volume Backup Export](volume-backup-export.md)
- [Volume Backup Export Record](volume-backup-export-record.md)
- [Volume Backup Actions](volume-backup-actions.md)
- [Volume Backup Import](volume-backup-import.md)
- [Volume Actions](volume-actions.md)
- [Volume Flags](volume-flags.md)
- [Volume Resize Retype](volume-resize-retype.md)
- [Volume Migration Reset](volume-migration-reset.md)
- [Volume revert·unmanage](volume-revert-unmanage.md)
- [Cinder Volume Attachment](cinder-volume-attachment.md)
- [Volume image metadata](volume-image-metadata.md)
- [Volume Upload Image](volume-upload-image.md)
