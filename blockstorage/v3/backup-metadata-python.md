# Backup Metadata Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_backup_metadata_test.go` in [example_backup_metadata_test.go](example_backup_metadata_test.go), compiled by `go test`.

## References

- [v3 router](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/router.py#L183-L187)
- [Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L27)
- [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1772-L1867)
- [metadata_python_test.go](backups/metadata_python_test.go)
- [metadata.go](backups/metadata.go)
