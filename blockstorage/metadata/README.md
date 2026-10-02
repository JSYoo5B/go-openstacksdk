# Volume and Snapshot metadata scopes

Block Storage v2/v3 Volume and Snapshot APIs expose
`MetadataIn(ctx, resource.Ref)`. An explicit ID binds without a request; an
explicit Name resolves once through the existing collection lookup and then
binds its ID. Python's metadata proxy methods treat a string as an ID, so Name
binding is a Go convenience. Each scope captures the selected collection URL
and ID. Later response data or changes to `ResourceBase` do not retarget it.

| Python operation | Scoped Go operation | Request |
| --- | --- | --- |
| `fetch_volume_metadata` / `fetch_snapshot_metadata` | `Get(ctx)` | GET metadata collection |
| deprecated `get_volume_metadata` / `get_snapshot_metadata` | `Get(ctx)` | same GET |
| `set_volume_metadata` / `set_snapshot_metadata` | `Merge(ctx, values)` | POST; preserve other keys |
| inherited `replace_metadata` | `Replace(ctx, values)` | PUT; replace the complete map |
| `delete_*_metadata(..., keys=None)` | `DeleteKeys(ctx, nil)` | one clear-all PUT |
| `delete_*_metadata(..., keys=[])` | `DeleteKeys(ctx, []string{})` | no HTTP after validation |
| `delete_*_metadata(..., keys=keys)` | `DeleteKeys(ctx, keys)` | ordered per-key DELETEs |

GET, POST and PUT accept HTTP 200 with an exact lowercase `metadata` object.
Metadata keys and values are strings. Both nil and empty input maps send
`{"metadata":{}}`: POST retains existing entries and PUT clears them. This
differs from native Snapshot `UpdateMetadata`'s `omitempty` input. No operation
implicitly fetches, merges client-side, or retries an accepted mutation.

For example, bind a v3 Volume and explicitly pass a returned ETag when replacing:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/blockstorage/v3/volumes"
    "gophercloudsdk/resource"
)

func replaceVolumeMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
    scope, err := volumes.New(client).MetadataIn(ctx, resource.ID(id))
    if err != nil {
        return err
    }
    current, err := scope.Get(ctx)
    if err != nil {
        return err
    }
    var options []volumes.MetadataOption
    if etag := current.Header.Get("ETag"); etag != "" {
        options = append(options, volumes.WithMetadataHeader("If-Match", etag))
    }
    result, err := scope.Replace(ctx, map[string]string{"owner": "team-a"}, options...)
    if err != nil {
        return err
    }
    _ = result.Metadata
    _ = result.Body // original response JSON, including envelope extensions
    return nil
}
```

Deletion preserves input order and duplicates. Each successful key DELETE has
an actual HTTP 200 acknowledgement. The first error stops the sequence and
returns earlier acknowledgements; this is not a transaction and missing keys
are not ignored. The SDK does not manufacture metadata from acknowledgements.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/blockstorage/v2/snapshots"
    "gophercloudsdk/resource"
)

func deleteSnapshotMetadataKeys(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
    scope, err := snapshots.New(client).MetadataIn(ctx, resource.ID(id))
    if err != nil {
        return err
    }
    result, err := scope.DeleteKeys(ctx, []string{"temporary", "obsolete"})
    if result != nil {
        for _, deleted := range result.Deleted {
            _ = deleted.Key // requested key with its actual response evidence
            _ = deleted.StatusCode
        }
    }
    return err // successful earlier deletes remain recorded on failure
}
```

`Result` retains the actual `Metadata` string map, complete raw `Body`, copied
`Header` and `StatusCode`. `DeleteResult` has `Cleared *Result` for clear-all PUT
and `Deleted []DeletedKey` for ordered acknowledgements. Empty-key calls have no
fabricated body, header or status. A malformed accepted envelope, non-string or
null metadata value, read failure or cancellation retains accepted-response
evidence through the SDK response error; native HTTP and transport causes remain
available through error unwrapping. Raw JSON retains unknown envelope fields.

Request keys and values must be valid UTF-8. Maps and key slices are copied
before custom options and HTTP; header options
snapshot their values when created. `WithHeader` and `WithHeaders`, also exposed as leaf
`WithMetadataHeader` / `WithMetadataHeaders`, are concrete SDK-owned options.
They cannot change auth, body, routing or version ownership, and do not add
query/session/cache/base-path options. Source `MoreHeaders` retains native
precedence for ordinary headers; conflicting source/request `If-Match` values
are rejected rather than silently dropping the condition. Ambiguous case
variants in one header map are rejected. The original Provider, live auth and
middleware remain shared; configure the source client before concurrent use.

The metadata routes require no new metadata-specific microversion. v3 starts
at 3.0; legacy v2 compatibility does not imply that a current deployment serves
v2. In the audited Cinder 27.0.0 server, **v3 Volume** metadata GET emits ETag at
3.15+, and collection PUT checks `If-Match` and can return 412. POST and key
DELETE do not use that validator. Snapshot/v2 have no equivalent guarantee.
The SDK never upgrades a version, fetches an ETag automatically, or retries an
unconditioned mutation after 412. An absent ETag in the example leaves the PUT
unconditioned; server support and policy determine concurrency behavior.

IDs and deletion keys are encoded as literal components once, without treating
`%` as already encoded or turning a slash key into another route. Mock transport
tests establish SDK URL construction; they do not establish how a deployment's
proxy/WSGI handles encoded slashes or dot components. Server key/length rules
remain authoritative, and Volume's key regex is not applied to Snapshot.

Pinned Python returns and updates a seeded mutable Volume/Snapshot Resource.
Its set method caches the request delta and ignores the full merged response;
its delete method may fail while updating an incomplete local cache after
successful HTTP. This Go scope returns owned actual response evidence and has
no Resource cache/dirty lifecycle. Its strict string result, explicit Ref,
input ownership and partial deletion evidence are intentional Go policies;
arbitrary Resource subclasses, injected sessions and inherited cache behavior
remain separate. Backup metadata child routes are excluded: the audited stock
server has no `/backups/{id}/metadata` route.

Sources: pinned [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py),
[v2 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py),
[v3 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py),
and fixed Cinder 27.0.0 [Volume controller](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/volume_metadata.py)
and [Snapshot controller](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/snapshot_metadata.py).

Leaf guides: [v2 Volume](../v2/volumes/README.md),
[v3 Volume](../v3/volumes/README.md), [v2 Snapshot](../v2/snapshots/README.md),
[v3 Snapshot](../v3/snapshots/README.md).

The [four-binding HTTP tests](../../api/cinder_metadata_scopes_test.go) cover
actual response ownership, empty-map and deletion branches, Name lookup,
header/ETag errors, fixed routes, live auth and native API isolation.
[Connection tests](../../connection_metadata_test.go) cover the cached v3
Volume/Snapshot client, and [generator tests](../../internal/cmd/sdkgen/cinder_metadata_scopes_test.go)
check that scope discovery supplements exactly the four existing collections.

The [fixed server contract](../../docs/cinder-metadata-server-contracts.md)
records the release-pinned route, ETag and Backup limitations.
