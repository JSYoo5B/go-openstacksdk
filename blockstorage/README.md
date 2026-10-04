# Block Storage

Cinder v3 볼륨의 조회, iterator, 삭제, 상태 대기를 제공합니다. `conn`과 `ctx`는 [전체 README](../README.md)처럼 준비합니다. Go 조각은 `fmt`, `time`, `resource` 등을 import한 오류 반환 함수 안에서 사용합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.block_storage.get_volume(id)` | `service.Volumes.Get(ctx, id)` |
| `conn.block_storage.find_volume(name, ignore_missing=False)` | `service.Volumes.Find(ctx, resource.Name(name))` |
| `conn.block_storage.volumes(status="available")` | `service.Volumes.List(ctx, resource.WithStatus("available"))` |
| `conn.block_storage.wait_for_status(volume, status="available", wait=300)` | `service.Volumes.Wait(ctx, resource.ID(volume.ID), "available", resource.WithTimeout(5*time.Minute))` |
| `conn.block_storage.delete_volume(id)` | `service.Volumes.Delete(ctx, resource.ID(id))` |
| `conn.attach_volume(server, volume, wait=True)` | `conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{Server: resource.ID(serverID), Volume: resource.ID(volumeID)})` |
| `conn.detach_volume(server, volume, wait=True)` | `conn.DetachVolume(ctx, blockstorage.DetachVolumeRequest{Server: resource.ID(serverID), Volume: resource.ID(volumeID)})` |
| `conn.create_volume(size, wait=True, image=None, bootable=None, **kwargs)` | `conn.CreateVolume(ctx, blockstorage.CreateVolumeRequest{Size: size}, options...)` |
| `conn.update_volume(name_or_id, **kwargs)` | `conn.UpdateVolume(ctx, blockstorage.UpdateVolumeRequest{NameOrID: name}, options...)` |
| `conn.set_volume_bootable(name_or_id, bootable=True)` | `conn.SetVolumeBootable(ctx, blockstorage.SetVolumeBootableRequest{NameOrID: name}, options...)` |
| `conn.delete_volume(name_or_id, wait=True, force=False)` | `conn.DeleteVolume(ctx, blockstorage.DeleteVolumeRequest{Volume: resource.ID(volumeID)}, options...)` |
| `conn.get_volume_attach_device(volume, server_id)` | `blockstorage.GetVolumeAttachDevice(volume, serverID)` 또는 `GetVolumeAttachDeviceFields(fields, serverID)` |
| `conn.get_volumes(server)` | `conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{ServerID: serverID})` |
| `conn.search_volumes(name_or_id, filters)` | `conn.SearchVolumes(ctx, blockstorage.SearchVolumesRequest{NameOrID: name}, options...)` |
| `conn.get_volume(name_or_id, filters=None)` | `conn.GetVolume(ctx, blockstorage.GetVolumeRequest{NameOrID: name}, options...)` |
| `conn.list_volumes()` | `conn.ListVolumes(ctx, options...)` |
| `conn.get_volume_by_id(id)` | `conn.GetVolumeByID(ctx, blockstorage.GetVolumeByIDRequest{ID: id}, options...)` |
| `conn.get_volume_id(name_or_id)` | `conn.GetVolumeID(ctx, blockstorage.GetVolumeIDRequest{NameOrID: name}, options...)` |
| `conn.get_volume_limits(name_or_id=None)` | `conn.GetVolumeLimits(ctx, blockstorage.GetVolumeLimitsRequest{NameOrID: name})` |
| `conn.list_volume_snapshots(detailed=True, filters=None)` | `conn.ListVolumeSnapshots(ctx, options...)` |
| `conn.search_volume_snapshots(name_or_id, filters)` | `conn.SearchVolumeSnapshots(ctx, blockstorage.SearchVolumeSnapshotsRequest{NameOrID: name}, options...)` |
| `conn.get_volume_snapshot(name_or_id, filters=None)` | `conn.GetVolumeSnapshot(ctx, blockstorage.GetVolumeSnapshotRequest{NameOrID: name}, options...)` |
| `conn.get_volume_snapshot_by_id(id)` | `conn.GetVolumeSnapshotByID(ctx, blockstorage.GetVolumeSnapshotByIDRequest{ID: id}, options...)` |
| `conn.create_volume_backup(volume_id, force=False, wait=True, incremental=False)` | `conn.CreateVolumeBackup(ctx, blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID}, options...)` |
| `conn.delete_volume_backup(name_or_id, force=False, wait=False)` | `conn.DeleteVolumeBackup(ctx, blockstorage.DeleteVolumeBackupRequest{NameOrID: name}, options...)` |
| `conn.export_volume_backup(backup_id)` | `conn.ExportVolumeBackup(ctx, blockstorage.ExportVolumeBackupRequest{BackupID: backupID})` |
| `conn.list_volume_backups(detailed=True, filters=None)` | `conn.ListVolumeBackups(ctx, options...)` |
| `conn.search_volume_backups(name_or_id, filters)` | `conn.SearchVolumeBackups(ctx, blockstorage.SearchVolumeBackupsRequest{NameOrID: name}, options...)` |
| `conn.get_volume_backup(name_or_id, filters=None)` | `conn.GetVolumeBackup(ctx, blockstorage.GetVolumeBackupRequest{NameOrID: name}, options...)` |
| `conn.list_volume_types()` | `conn.ListVolumeTypes(ctx, options...)` |
| `conn.search_volume_types(name_or_id, filters)` | `conn.SearchVolumeTypes(ctx, blockstorage.SearchVolumeTypesRequest{NameOrID: name}, options...)` |
| `conn.get_volume_type(name_or_id, filters=None)` | `conn.GetVolumeType(ctx, blockstorage.GetVolumeTypeRequest{NameOrID: name}, options...)` |
| `conn.get_volume_type_access(name_or_id)` | `conn.GetVolumeTypeAccess(ctx, blockstorage.GetVolumeTypeAccessRequest{NameOrID: name}, options...)` |
| `conn.add_volume_type_access(name_or_id, project_id)` | `conn.AddVolumeTypeAccess(ctx, blockstorage.VolumeTypeAccessRequest{NameOrID: name, ProjectID: projectID}, options...)` |
| `conn.remove_volume_type_access(name_or_id, project_id)` | `conn.RemoveVolumeTypeAccess(ctx, blockstorage.VolumeTypeAccessRequest{NameOrID: name, ProjectID: projectID}, options...)` |
| `conn.volume_exists(name_or_id)` | `conn.VolumeExists(ctx, blockstorage.VolumeExistsRequest{NameOrID: name}, options...)` |
| `conn.block_storage.fetch_volume_metadata(id)` | `conn.VolumeMetadata(ctx, resource.ID(id))`의 `Get(ctx)` |
| `conn.block_storage.set_volume_metadata(id, owner="worker")` | 같은 범위의 `Merge(ctx, map[string]string{"owner":"worker"})` |
| `conn.block_storage.delete_volume_metadata(id, keys)` | 같은 범위의 `DeleteKeys(ctx, keys)` |

SDK에서 볼륨 목록은 상세 목록 API를 사용합니다. Cloud 수준 삭제의 `WithDeleteVolumeForce`는 선택된 microversion에 맞는 요청을 사용합니다. Proxy의 별도 `cascade` 옵션은 이 workflow에 포함하지 않으며 snapshot까지 삭제하는 옵션은 native API 계층에서 명시적으로 선택합니다. [공식 Block Storage API](https://docs.openstack.org/openstacksdk/latest/user/proxies/block_storage_v3.html)

볼륨 수정의 변경 필드 처리와 부분 응답 병합, bootable 기본값과 명시적인 false는 [사용법과 Python 비교](volume-mutations.md)에 설명합니다. SDK가 옵션과 실제 단계별 응답을 소유하며, 인터페이스 builder를 구현할 필요는 없습니다.

## 조회와 목록

Python:

```python
volume = conn.block_storage.find_volume("data", ignore_missing=False)
for volume in conn.block_storage.volumes(status="available"):
    print(volume.id, volume.size)
```

Go:

```go
service, err := conn.BlockStorage(ctx)
if err != nil { return err }
volume, err := service.Volumes.Find(ctx, resource.Name("data"))
if err != nil { return err }
fmt.Println(volume.ID, volume.Size)

for volume, err := range service.Volumes.List(ctx,
    resource.WithStatus("available"), resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(volume.ID, volume.Size)
}
```

query 확장은 `resource.WithQuery("key", "value")`를 사용합니다. 이름 검색은 정확한 일치를 확인하며 모든 페이지에 걸쳐 중복을 검사합니다. 이름 조회는 현재 클라이언트가 볼 수 있는 기본 조회 범위를 사용합니다.

## 대기와 삭제

```go
ready, err := service.Volumes.Wait(ctx, resource.ID(volume.ID), "available",
    resource.WithTimeout(5*time.Minute))
if err != nil { return err }
fmt.Println(ready.Status)

if err := service.Volumes.Delete(ctx, resource.ID(volume.ID)); err != nil {
    return err
}
```

`error`, `error_deleting`처럼 `error`로 시작하는 상태는 즉시 실패로 처리합니다. Delete는 삭제 요청의 성공을 반환하고 실제 삭제 완료까지 기다리지 않습니다.

microversion이 필요하면 연결 시 `sdk.WithMicroversion(sdk.BlockStorage, "3.60")`으로 정확한 버전을 지정하거나, `sdk.WithMicroversionRange(sdk.BlockStorage, "3.0", "3.60")`으로 클라우드와 겹치는 가장 높은 버전을 선택합니다. [협상과 캐시 정책](../docs/microversions.md)을 참고하세요. 직접 endpoint를 설정한다면 `/v3/<project-id>/`가 포함된 URL을 사용합니다.

## 전체 API와 서버 부팅

볼륨 생성·수정·크기 변경·attachment·snapshot·backup·volume type 관리는 `service.API`의 [Block Storage v3 API](v3/README.md)에서 제공합니다. 기존 볼륨으로 서버를 부팅할 때는 [Compute의 `WithBootVolume`](../compute/README.md)을 사용하며, 볼륨 이름은 이 패키지의 `Volumes.ResolveID`로 해석합니다. 기존 데이터 볼륨 연결은 `conn.AttachVolume`이 연결 전 Cinder 조회, Nova 요청과 기본 완료 대기를 묶습니다. [연결 사용법과 Python 비교](attach-volume.md)에 옵션·기본값·오류 시 부분 결과를 설명합니다. 기존 데이터 볼륨 분리는 `conn.DetachVolume`이 Nova DELETE와 선택적인 Cinder 대기를 묶습니다. [분리 사용법과 Python 비교](detach-volume.md)는 Nova만 사용하는 호출과 접수 후 대기 오류를 설명합니다. `conn.CreateVolume`은 이미지 이름·생성 속성·기본 완료 대기와 선택적인 bootable 액션을 처리합니다. [생성 사용법과 Python 비교](create-volume.md)에 알려진 속성 매핑과 단계별 결과를 설명합니다.

`service.API.Snapshots.UpdateMetadata`와 버전별 Snapshot API는 PUT 응답의 메타데이터를
`map[string]any`로 반환합니다. 이전 `*Snapshot` 반환형은 다른 envelope를 읽어 정상 응답도
nil로 반환하던 오류였습니다. Snapshot 전체 모델이 필요하면 `Get`을 명시적으로 호출합니다.
숫자는 native `json.Number`로 유지하고 잘못된 metadata object는 panic 대신 오류로 처리합니다.
native options의 nil/빈 Metadata map은 `omitempty`로 생략되며, core 필드를 extension으로
덮어쓰는 기존 금지 정책도 유지합니다. [v2 사용법](v2/snapshots/README.md)과
[v3 사용법](v3/snapshots/README.md)에 반환형 변경과 Python의 POST 병합·캐시 차이를 설명합니다.

[blockstorage_test.go](blockstorage_test.go)는 Cinder의 실패 상태 패턴과 microversion 헤더를, [전체 통합 테스트](../collections_test.go)는 서비스 공통 정책을 검증합니다.

Volume·Snapshot의 메타데이터는 v2/v3 버전별 `API.MetadataIn(ctx, ref)`로 고정합니다. v3는 `conn.VolumeMetadata`·`conn.SnapshotMetadata`가 공유 클라이언트를 연결합니다. Get은 실제 map을 조회하고 Merge는 POST 병합, Replace는 PUT 전체 교체를 실행합니다. nil/빈 map도 명시적인 객체를 보내며 DeleteKeys의 nil은 전체 삭제, 빈 slice는 요청 없음입니다. 문자열 map·header 옵션·순서별 부분 성공과 Python Resource/cache 차이는 [공통 사용법](metadata/README.md), [v3 Volume](v3/volumes/README.md)·[Snapshot](v3/snapshots/README.md)을 참고합니다. [서버 계약](../docs/cinder-metadata-server-contracts.md)은 Cinder의 ETag와 Backup 경로 차이를 고정 소스로 설명합니다.

`blockstorage.WaitForAvailable(ctx, service.Volumes, ref)`는 available·정확한 error 실패 상태와 무제한 SDK timeout을 기본으로 사용합니다. `WaitForState`는 다른 대상을 명시하고 `WaitForDelete`는 삭제 요청 없이 기본 120초 동안 삭제 완료를 관찰합니다. v2/v3 Volume·Snapshot leaf에도 같은 메서드가 있습니다. [서비스별 대기 비교](../docs/service-waits.md)에 옵션·context·Python 대응과 남은 차이를 설명합니다.

`conn.DeleteVolume`은 초기 조회·선택적 강제 삭제·기본 완료 대기를 묶고 초기 부재와 조회 후 삭제 경합을 구별합니다. [삭제 사용법과 Python 비교](delete-volume.md)에 선택된 microversion, `Deleted` 반환값과 오류 시 단계별 결과를 설명합니다.

`blockstorage.GetVolumeAttachDevice`는 owned observation·VolumeInfo와 native Cinder v2/v3 Volume의 현재 attachment를 읽는 순수 helper입니다. 서버 ID의 첫 정확한 일치가 반환되며 HTTP와 refresh는 없습니다. raw JSON 값, nil·빈 문자열과 전체 decoder의 경계는 [서버별 device 조회](volume-attachment-device.md)에 설명합니다.

`conn.GetVolumes`는 Cinder v3 상세 목록을 모두 읽은 뒤 서버별 association을 선택합니다. Device나 상태를 조건으로 삼지 않고 같은 볼륨의 중복 일치도 유지합니다. RawResource 결과, 선택적인 typed projection과 오류 시 실제 페이지 증거는 [서버에 연결된 볼륨 목록](server-volumes.md)에 설명합니다.

`conn.SearchVolumes`는 glob·중첩 JSON 필터·전체 JMESPath와 계산된 location을 제공합니다.
`conn.GetVolume`은 필터 생략·null의 GET-first 조회와 명시적 필터의 전체 검색을 구분합니다.
임의 JSON 결과와 원본 row, 기본값, Python 비교는 [볼륨 검색과 단일 선택](search-volumes.md)에 설명합니다.

전체 목록·ID 직접 조회·존재 확인은 [읽기 사용법과 Python 비교](volume-reads.md)에 설명합니다. `GetVolumeByID`의 404는 오류이며, `VolumeExists`의 `false`는 정상 부재만 나타냅니다. `WithVolumeReadLocation`으로 각 호출의 기본 location을 교체할 수 있습니다.

`GetVolumeID`는 이름·ID를 실제 조회하고 응답의 ID JSON을 반환합니다. [ID 조회와 Python 비교](volume-id.md)에 정상 부재, 발견한 null ID, 오류와 실제 응답 증거의 구분을 설명합니다.

볼륨 타입의 [목록·검색·조회 사용법과 Python 비교](volume-types.md)에 `is_public=none`을 사용하는 기본 조회, nonnull 필터의 전체 검색, Type location과 원본 응답 증거를 설명합니다. `WithVolumeTypeReadLocation`과 `WithVolumeTypeSearchLocation`으로 호출별 location을 지정합니다.

타입의 프로젝트 접근 권한은 `GetVolumeTypeAccess`·`AddVolumeTypeAccess`·`RemoveVolumeTypeAccess`로 조회·변경합니다. [접근 권한 사용법과 Python 비교](volume-type-access.md)에 타입 조회 후의 실제 응답 ID 사용, 원본 JSON 값, literal 프로젝트 ID, 단계별 응답 증거와 오류를 설명합니다. 프로젝트 존재 확인은 서버가 처리하며, 변경 응답은 권한 변경의 완료를 보장하지 않습니다.

[볼륨 limits 조회](volume-limits.md)는 빈 입력의 Identity 생략, 다른 프로젝트의 strict 조회, raw/nullable 모델과 단계별 증거를 설명합니다. 기본값과 옵션 적용은 SDK가 담당합니다.

Snapshot의 상세 목록·서버/local 필터와 Search 필터, exact 이름/ID 조회·literal ID 조회는 [Snapshot 사용법과 Python 비교](volume-snapshots.md)에 설명합니다. `Value`는 nullable 16-field logical JSON이고 일반 조회의 `Snapshot`·`Snapshots`는 actual wire rows입니다. 일반 빈 목록의 `Snapshots`는 nonnil 빈 slice이며 expression 결과에는 raw row 연결을 만들지 않습니다. 오류에도 실제 `Observed`·`Pages` 증거를 유지합니다.

Snapshot 생성·삭제는 `CreateVolumeSnapshot`·`DeleteVolumeSnapshot`으로 호출합니다. 생성의 기본 `wait=true`와 삭제의 기본 `wait=false`, 네 가지 생성 속성, exact 이름/ID 해석과 단계별 결과는 [Snapshot 생성·삭제와 Python 비교](volume-snapshot-mutations.md)에 설명합니다.


Backup의 상세 목록·서버/local 필터·Search 필터와 exact 이름/ID 조회는 [Backup 사용법과 Python 비교](volume-backups.md)에
설명합니다. `Value`는 nullable 24-field logical JSON이고 `Backup`·`Backups`는 actual wire rows입니다.
`"false"` 문자열도 true인 ordinary Boolean, row project와 zone을 사용하는 location, normal absence,
오류 시 `Observed`·`Pages`와 선택적인 raw `Clone`·`Decode`를 구분합니다. 이 세 cloud read helper는
기존 native Backup, v2, 생성·삭제·restore/import/export/action의 지원 검토와 독립적입니다.


Backup 생성·삭제는 `CreateVolumeBackup`·`DeleteVolumeBackup`으로 호출합니다. 여섯 생성 field의
null/false 보존, 기본 wait와 timeout, 조회·변경·poll의 실제 결과는
[Backup 생성·삭제와 Python 비교](volume-backup-mutations.md)에 설명합니다. force는
`os-force_delete: null` POST와 독립 3.64 action이며 normal selected microversion과 다릅니다.
이 두 cloud workflow와 v2·native·restore/import/export/reset/update/metadata 검토는 독립적입니다.

Backup export는 `ExportVolumeBackup`으로 호출합니다. [Backup export와 Python 비교](volume-backup-export.md)에
이름 조회·wait·location 변환 없이 ID를 요청하는 API, 실제 opaque 응답과 부분 오류 증거를 설명합니다.
Python v3 cloud의 raw Response를 대상으로 하며 native typed Export와 별도 parsed Proxy 선언의
지원 검토는 독립적입니다.
