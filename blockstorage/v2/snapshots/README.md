# Snapshot metadata in Block Storage v2

`API.UpdateMetadata` sends `PUT /snapshots/{id}/metadata` and returns the
server's `metadata` object as `map[string]any`. The request uses concrete
`UpdateMetadataOpts` and library-owned `WithUpdateMetadataOptions` /
`WithUpdateMetadataField` options; callers do not implement a builder.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots"
)

func replaceSnapshotMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
    return snapshots.New(client).UpdateMetadata(ctx, id, snapshots.UpdateMetadataOpts{
        Metadata: map[string]any{"owner": "team-a", "purpose": "backup"},
    })
}
```

This is the native PUT replacement operation. Supply the metadata to retain;
there is no read/merge/write sequence or follow-up Snapshot GET. The native
`metadata,omitempty` request field remains unchanged: a nil or empty map is
omitted, producing `{}`, rather than `{"metadata": {}}`.
`WithUpdateMetadataOptions` retains the native map reference; it does not deep
copy typed metadata. `WithUpdateMetadataField` snapshots its extension JSON,
protects the core `metadata` key, and follows the existing builder's nesting
rules. An extension added to a populated metadata object goes inside that object.

The return type corrects the earlier `*Snapshot` wrapper, which looked for a
`snapshot` response envelope and could return nil for a valid metadata response.
Callers using that return type should read the returned map directly instead of
accessing `Snapshot.Metadata`. Other Snapshot-returning methods retain their
own result types. This native result correction is separate from the scoped
metadata APIs below.

HTTP 200 is the native success status. The SDK requires an exact lowercase
`metadata` object, including an empty object, and reports an error for a missing,
null or nonobject envelope instead of triggering the native extractor's type
assertion panic. Values inside a valid object retain decoded nulls, arrays,
objects and `json.Number` numbers without another JSON decode. Native HTTP,
transport, read and context errors keep their original cause through the SDK's
operation error. This map result does not expose original response bytes or a
new status/header result, and malformed responses do not trigger another request.

Pinned openstacksdk's `set_snapshot_metadata` uses POST to merge metadata and
updates a cached Resource. Its replacement path sends an explicit metadata
object. The Go native PUT operation and actual response map do not implement
that cached lifecycle, POST merge, or the separate metadata GET/delete family.
See the pinned [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)
and [snapshot proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py).

The [v3 example](../../v3/snapshots/README.md) uses the same corrected result
contract. [Generator documentation](../../../internal/cmd/sdkgen/README.md)
describes the pinned native source checks.

[HTTP contract tests](../../../api/snapshot_metadata_results_test.go) cover both
versions, response envelopes and number precision, native input ownership,
empty-map omission, error causes and the shared client.

## Scoped metadata operations

`API.MetadataIn(ctx, resource.ID(id))` binds without HTTP. `scope.Get` reads
metadata; `scope.Merge` POSTs a delta; `scope.Replace` PUTs a complete map.
These SDK-owned operations accept `map[string]string` and return actual string
metadata plus raw Body, copied Header and StatusCode. They explicitly send
`metadata:{}` for nil or empty maps, while native `UpdateMetadata` keeps the
input omission and decoded `map[string]any` contract described above.

`scope.DeleteKeys(ctx, nil)` clears with one PUT. A nonnil empty slice makes no
request after validation. Other keys are deleted in original order, including
duplicates; the first failure returns earlier actual acknowledgements. There
is no automatic GET, missing-key suppression or Resource-cache update. An
explicit Name binds once through the existing collection lookup, a Go
convenience beyond Python's ID string input.

The [shared guide](../../metadata/README.md) contains complete examples and
explains fixed targets, header/error ownership, escaping and Python cache
differences. Snapshot has no promised Volume 3.15 ETag behavior. Legacy v2
compatibility is separate from current server v3 endpoint support.

[Scoped HTTP tests](../../../api/cinder_metadata_scopes_test.go) cover all four
bindings, including native Snapshot result isolation.
[Connection tests](../../../connection_metadata_test.go) cover the shared v3
client; v2 remains available through its existing leaf API.

The [release-pinned server contract](../../../docs/cinder-metadata-server-contracts.md)
separates these routes from Backup metadata and deployment-specific policies.

## 서비스별 상태·삭제 대기

`WaitForAvailable(ctx, ref, options...)`는 available/error와 2초 간격을
SDK가 적용하며 SDK 시간 제한이 없습니다. `WaitForState`는 caller가 대상
상태를 지정하고, `WaitForDelete`는 삭제 요청 없이 실제 미존재를 기본
120초 기다립니다. parent context와 호출별 `resource.WaitOption`이 항상
적용됩니다. 기존 공통 `WaitFor`/`WaitForDeletion`의 5분 정책과 네이티브
`WaitForStatus`는 유지됩니다. 전체 예제와 Python Resource/cache의 차이는
[서비스 대기 가이드](../../../docs/service-waits.md)를 참고하세요.
