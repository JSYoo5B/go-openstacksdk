# Volume metadata in Block Storage v2

`API.MetadataIn(ctx, resource.ID(id))` binds a fixed Volume metadata scope
without HTTP. An explicit `resource.Name` resolves once through the existing
collection lookup; this is a Go convenience beyond Python's ID string input.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/blockstorage/v2/volumes"
    "github.com/JSYoo5B/gophercloudsdk/resource"
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
StatusCode. Header options and URI/source/error policies are described in the
[shared metadata guide](../../metadata/README.md). Native parent Volume
Create/Update metadata fields and image-metadata actions remain separate APIs.
v2 is legacy source compatibility; current Cinder v3 server evidence does not
promise a v2 endpoint or v3 Volume's ETag behavior on v2.

Python `fetch_volume_metadata` and deprecated `get_volume_metadata` correspond
to `scope.Get`; `set_volume_metadata` corresponds to `scope.Merge`. Python's
Resource cache, seeded return object and injected session differ from this
owned response scope. The [v3 guide](../../v3/volumes/README.md) uses the same
scoped methods.

[HTTP contract tests](../../../api/cinder_metadata_scopes_test.go) exercise all
four Volume/Snapshot bindings. [Connection tests](../../../connection_metadata_test.go)
cover the shared v3 client; v2 uses its existing leaf API.

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
