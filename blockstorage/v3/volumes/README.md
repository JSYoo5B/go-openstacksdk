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

## 서비스별 상태·삭제 대기

`WaitForAvailable(ctx, ref, options...)`는 available/error와 2초 간격을
SDK가 적용하며 SDK 시간 제한이 없습니다. `WaitForState`는 caller가 대상
상태를 지정하고, `WaitForDelete`는 삭제 요청 없이 실제 미존재를 기본
120초 기다립니다. parent context와 호출별 `resource.WaitOption`이 항상
적용됩니다. 기존 공통 `WaitFor`/`WaitForDeletion`의 5분 정책과 네이티브
`WaitForStatus`는 유지됩니다. 전체 예제와 Python Resource/cache의 차이는
[서비스 대기 가이드](../../../docs/service-waits.md)를 참고하세요.

The service API adds `ReserveVolume(ctx, id)`, `UnreserveVolume(ctx, id)`, `BeginVolumeDetaching(ctx, id)` and `AbortVolumeDetaching(ctx, id)`. Each sends the pinned source's null action value and returns owned opaque acknowledgement plus separate discovery proof. Existing native `Reserve`, `Unreserve` and `BeginDetaching` retain their original body/status policy. See [the package and Connection examples](../../volume-actions.md).

The service API adds `SetVolumeBootableStatus(ctx, id, bool)` and `SetVolumeReadonly(ctx, id, options...)`. Bootable requires an explicit bool; readonly defaults to true and `WithVolumeReadonly(false)` sends false. `WithVolumeReadonlyOptions` and `PrepareVolumeReadonlyOptions` own reusable policies without caller builders. Both actions return `VolumeActionResult` with separate discovery and opaque acknowledgement proof. Existing native `SetBootable` and attach modes keep their original contracts. See [the Connection and package flag examples](../../volume-flags.md).

The direct-ID service methods `ExtendVolume(ctx, id, size)`, `RetypeVolume(ctx, id, newType, options...)` and `CompleteVolumeExtend(ctx, id, options...)` preserve typed literal action inputs and return `VolumeActionResult`. Retype defaults to migration policy never; an explicit empty policy omits it. Extend completion defaults to false and always includes the error bool. `WithVolumeRetypeOptions`/`PrepareVolumeRetypeOptions` and `WithVolumeExtendCompletionOptions`/`PrepareVolumeExtendCompletionOptions` own reusable policies without caller builders. Existing native `ExtendSize` and `ChangeType` keep their contracts. See [the Connection and package examples](../../volume-resize-retype.md).

`API.ResetVolumeStatus`·`API.MigrateVolume`·`API.CompleteVolumeMigration`은 명시 ID의 opaque action helper입니다. [사용법과 Python 비교](../../volume-migration-reset.md)를 참고하세요. 기존 native `ResetStatus`의 status·202/error-only 계약은 유지됩니다.

`API.RevertVolumeToSnapshot`·`API.UnmanageVolume`은 SDK가 소유한 action 결과를 반환합니다. [사용법과 Python 비교](../../volume-revert-unmanage.md)를 참고하세요. 기존 native `Unmanage`의 `{}` body·202/error-only 계약은 유지됩니다.

[직접 Cinder attach·detach](../../cinder-volume-attachment.md)의 `API.AttachCinderVolume`·`API.DetachCinderVolume`은 required literal body 인자와 owned functional options를 받고 `VolumeActionResult`를 반환합니다. 기존 native `API.Attach`·`API.Detach`의 Mode/omitempty/202/error-only 계약과 Nova+Cinder `Connection.AttachVolume`·`DetachVolume` workflow는 별도 API로 유지합니다.
