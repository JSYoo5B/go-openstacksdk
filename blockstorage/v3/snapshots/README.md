# Snapshots

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
)

func replaceSnapshotMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
    return snapshots.New(client).UpdateMetadata(ctx, id, snapshots.UpdateMetadataOpts{
        Metadata: map[string]any{"owner": "team-a", "purpose": "backup"},
    })
}
```

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
