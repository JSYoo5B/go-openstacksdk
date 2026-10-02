# Volume metadata in Block Storage v3

`API.MetadataIn(ctx, resource.ID(id))` binds a fixed Volume metadata scope
without HTTP. An explicit `resource.Name` resolves once through the existing
collection lookup; this is a Go convenience beyond Python's ID string input.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/blockstorage/v3/volumes"
    "gophercloudsdk/resource"
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

`Get` reads metadata, `Merge` sends POST, and `Replace` sends PUT. Nil and empty
maps explicitly encode `metadata:{}`. `DeleteKeys(nil)` clears with one PUT;
`DeleteKeys([]string{})` validates without HTTP. Nonempty keys preserve order
and duplicates, return actual earlier acknowledgements on the first failure,
and do not ignore missing keys. The scope does not prefetch or cache the Volume.

Results retain the actual string map, raw response Body, copied Header and
StatusCode. The [shared metadata guide](../../metadata/README.md) includes an
explicit ETag example: v3 Volume GET emits ETag at 3.15+, and PUT can enforce
`If-Match`; POST and key DELETE do not use that validator. There is no new
metadata route gate, automatic version upgrade, implicit ETag fetch or 412
retry. Native parent Volume Create/Update metadata fields and image-metadata
actions remain separate APIs.

Python `fetch_volume_metadata` and deprecated `get_volume_metadata` correspond
to `scope.Get`; `set_volume_metadata` corresponds to `scope.Merge`. Python's
Resource cache, seeded return object and injected session differ from this
owned response scope. The [v2 guide](../../v2/volumes/README.md) uses the same
scoped methods for legacy v2 compatibility.

[HTTP contract tests](../../../api/cinder_metadata_scopes_test.go) exercise all
four Volume/Snapshot bindings. [Connection tests](../../../connection_metadata_test.go)
cover the shared v3 client and fixed metadata target.

The [release-pinned server contract](../../../docs/cinder-metadata-server-contracts.md)
separates these routes from Backup metadata and deployment-specific policies.
