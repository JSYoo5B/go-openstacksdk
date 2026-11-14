# Volumes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func mergeVolumeMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]string, error) {
    scope, err := volumes.New(client).MetadataIn(ctx, resource.ID(id))
    if err != nil {
        return nil, err
    }
    result, err := scope.Merge(ctx, map[string]string{"owner": "team-a"})
    if err != nil {
        return nil, err
    }
    return result.Metadata, nil // full actual server response, including retained keys
}
```

## References

- [shared metadata guide](../../metadata/README.md)
- [v3 guide](../../v3/volumes/README.md)
- [HTTP contract tests](../../../api/cinder_metadata_scopes_test.go)
- [Connection tests](../../../connectiontest/connection_metadata_test.go)
- [release-pinned server contract](../../../docs/cinder-metadata-server-contracts.md)
- [Service Waits](../../../docs/service-waits.md)
