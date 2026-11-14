# Snapshots

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_snapshots_test.go` in [example_snapshots_test.go](example_snapshots_test.go), compiled by `go test`.

## References

- [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)
- [snapshot proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [v2 example](../../v2/snapshots/README.md)
- [Generator documentation](../../../internal/cmd/sdkgen/README.md)
- [HTTP contract tests](../../../api/snapshot_metadata_results_test.go)
- [shared guide](../../metadata/README.md)
- [Scoped HTTP tests](../../../api/cinder_metadata_scopes_test.go)
- [Connection tests](../../../connectiontest/connection_metadata_test.go)
- [Service Waits](../../../docs/service-waits.md)
- [Python Manage](python-manage.md)
