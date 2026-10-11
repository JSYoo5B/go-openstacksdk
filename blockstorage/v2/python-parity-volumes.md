# Cinder v2 Python proxy 대응: volume·snapshot·cgsnapshot

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [block_storage v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py) 가운데 volume, snapshot, consistency group snapshot 메서드와 `delete_qos_spec_metadata`까지 36개를 Go 호출과 비교합니다. 판정은 Python이 기본 인자와 문서화된 인자로 보내는 HTTP 요청을 [Proxy 공통 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py), [Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/volume.py), [Snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/snapshot.py) Resource에서 따라가 정했습니다. 비교 기준은 [volumes 계약 테스트](volumes/python_parity_test.go)와 [snapshots 계약 테스트](snapshots/python_parity_test.go)입니다.

표의 Go 호출은 `conn.BlockStorageV2(ctx)`가 돌려준 service의 `Volumes`·`Snapshots` 필드를 기준으로 적었습니다. `go_mapping`은 모든 Python 인자의 효과를 공개 Go API로 재현할 수 있다는 뜻이고, 이때도 아래 절의 차이는 남습니다. `unresolved`는 v2 경로에 대한 Go 호출이 없거나, 기본값을 바꿀 수단이 없어 Python과 같은 요청을 만들 수 없다는 뜻입니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`get_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L594) | `Volumes.Get(ctx, id)` | go_mapping |
| [`find_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L626) | `Volumes.Find(ctx, resource.ID(x), resource.WithIgnoreMissing())` 뒤 `Volumes.Find(ctx, resource.Name(x))` | unresolved |
| [`volumes`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L666) | `Volumes.List(ctx, options...)` (상세 목록만) | unresolved |
| [`create_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L694) | `Volumes.Create(ctx, opts, WithCreateField)` | unresolved |
| [`update_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L705) | `Volumes.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| [`delete_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L720) | `resource.IgnoreMissing(Volumes.Delete(ctx, id, WithDeleteQuery("cascade", "False")))`, `Volumes.Remove(ctx, resource.ID(id))`, `Volumes.ForceDelete(ctx, id)` | go_mapping |
| [`extend_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L761) | `Volumes.ExtendSize(ctx, id, ExtendSizeOpts{NewSize})` | go_mapping |
| [`set_volume_readonly`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L773) | 없음 | unresolved |
| [`retype_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L788) | `Volumes.ChangeType(ctx, id, ChangeTypeOpts{NewType, MigrationPolicy: MigrationPolicyNever})` | go_mapping |
| [`set_volume_bootable_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L811) | `Volumes.SetBootable(ctx, id, BootableOpts{Bootable})` | go_mapping |
| [`set_volume_image_metadata`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L826) | `Volumes.SetImageMetadata(ctx, id, ImageMetadataOpts{Metadata})` | go_mapping |
| [`delete_volume_image_metadata`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L841) | 없음 | unresolved |
| [`reset_volume_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L861) | `Volumes.ResetStatus(ctx, id, ResetStatusOpts{...})` (status 생략 불가) | unresolved |
| [`attach_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L882) | `Volumes.Attach(ctx, id, AttachOpts{MountPoint, InstanceUUID 또는 HostName})` | go_mapping |
| [`detach_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L902) | `Volumes.Detach(ctx, id, DetachOpts{AttachmentID})` (force 없음) | unresolved |
| [`unmanage_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L923) | 없음 | unresolved |
| [`migrate_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L935) | 없음 | unresolved |
| [`complete_volume_migration`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L961) | 없음 | unresolved |
| [`upload_volume_to_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L980) | `Volumes.UploadImage(ctx, id, UploadImageOpts{...})` | go_mapping |
| [`create_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L59) | `Volumes.UploadImage` 뒤 `image.Service.ExistingImageRecord(ctx, imageID)` | go_mapping |
| [`get_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L91) | `Snapshots.Get(ctx, id)` | go_mapping |
| [`find_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L126) | `Snapshots.Find(ctx, resource.ID(x), ...)` 뒤 `resource.Name(x)` (요약 경로) | unresolved |
| [`snapshots`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L170) | `Snapshots.List(ctx, options...)` (요약 경로 한 페이지) | unresolved |
| [`create_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L201) | `Snapshots.Create(ctx, opts, WithCreateField)` | go_mapping |
| [`update_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L212) | 없음 | unresolved |
| [`delete_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L225) | `Snapshots.Remove(ctx, resource.ID(id))` (force 없음) | unresolved |
| [`reset_snapshot_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L255) | 없음 | unresolved |
| [`reset_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L269) | 없음 | unresolved |
| [`manage_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L279) | 없음 | unresolved |
| [`unmanage_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L291) | 없음 | unresolved |
| [`delete_qos_spec_metadata`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1436) | 없음 | unresolved |
| [`get_consistency_group_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2231) | 없음 | unresolved |
| [`find_consistency_group_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2270) | 없음 | unresolved |
| [`consistency_group_snapshots`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2303) | 없음 | unresolved |
| [`create_consistency_group_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2330) | 없음 | unresolved |
| [`delete_consistency_group_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2346) | 없음 | unresolved |

## 공통 차이

Python은 Resource 객체나 ID 문자열을 받고 Go generated 호출은 ID 문자열을 받습니다. `Find`와 `Remove` 같은 collection 호출은 `resource.Ref`를 받습니다. 결과는 Python Resource 대신 Gophercloud v2 typed model이라서 모델에 없는 응답 field는 남지 않습니다. 예를 들어 v2 `Volume`에는 `volume_image_metadata`, `os-vol-mig-status-attr:*`, `os-volume-replication:*` field가 없고 v2 `Snapshot`에는 `force` field가 없습니다.

Cinder v2 API에는 microversion이 없고 Python v2 Proxy도 version header를 보내지 않습니다. `conn.BlockStorageV2(ctx)`의 client는 microversion이 비어 있으므로 generated 호출도 header를 보내지 않으며, 테스트에서 이를 확인합니다. 다만 [blockstorage 루트 helper](../README.md)(`SetVolumeReadonly`, `MigrateVolume`, `UnmanageVolume` 등)는 Cinder v3 전용입니다. 이 helper에 v2 client를 넘기면 major 3 version discovery를 먼저 보내고 v3 microversion header를 v2 경로에 붙일 수 있습니다. 각 helper 문서도 v2 지원 근거가 아니라고 밝히므로 이 문서는 v2 대응에 쓰지 않습니다. `blockstorage/v3/*` package를 v2 client로 만드는 방법도 같은 이유로 대응에 넣지 않았습니다.

Python v2 [`Volume._action`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/volume.py#L109)은 `raise_from_response`를 호출하지 않습니다. 그래서 extend, retype, bootable, image metadata, attach, detach, reset, force delete action은 HTTP 오류 응답을 받아도 예외 없이 None을 돌려줍니다. Go generated 호출은 고정된 성공 status(대부분 202, bootable과 image metadata는 200)만 받아들이고 그 밖의 응답은 오류로 돌려줍니다. Snapshot action은 Python도 오류를 예외로 올립니다.

## Volume

`get_volume`과 `update_volume`은 같은 경로와 envelope을 보냅니다. Python update는 바뀐 attribute가 없으면 요청을 보내지 않지만 Go `Update`는 빈 `{"volume":{}}`도 PUT합니다. `UpdateOpts`에 없는 attribute는 `WithUpdateField`에 wire 이름으로 줍니다.

`delete_volume`은 force가 없으면 항상 `?cascade=False`(또는 `True`) query를 붙인 DELETE를 보내고 404를 무시합니다. Go에서 같은 요청은 `resource.IgnoreMissing(Volumes.Delete(ctx, id, volumes.WithDeleteQuery("cascade", "False")))`입니다. 이 helper는 순수 404만 nil로 바꾸고 409 같은 다른 실패는 그대로 돌려줍니다. `Remove(ctx, resource.ID(id))`도 404를 무시하지만 query를 보내지 않으며 `resource.WithMissingError()`가 `ignore_missing=False`에 해당합니다. `DeleteOpts{Cascade: true}`는 소문자 `cascade=true`를 보내므로 Python과 같은 문자열이 필요하면 `WithDeleteQuery`를 씁니다. `force=True`는 `ForceDelete`가 같은 `os-force_delete` action을 보내며 값만 Python의 null 대신 `""`입니다.

`create_volume`은 지정한 attribute만 `volume` envelope에 넣고, `scheduler_hints`도 v2 Resource에서는 일반 Body attribute라서 envelope 안의 `OS-SCH-HNT:scheduler_hints`가 됩니다. Go에서는 `WithCreateField("OS-SCH-HNT:scheduler_hints", hints)`가 같은 위치에 넣습니다. `WithCreateHintOpts`는 envelope 밖에 두므로 v2 Python과 다릅니다. 판정을 미완료로 둔 이유는 `CreateOpts.Size`에 omitempty가 없어서 Go가 항상 `"size"`를 보내기 때문입니다. Python의 `create_volume(snapshot_id=...)`처럼 size를 생략해 서버가 원본 크기를 쓰게 하는 요청은 만들 수 없고, `WithCreateField("size", ...)`도 core field라서 거부됩니다.

`volumes`의 기본 `details=True`는 `Volumes.List`와 같은 `GET volumes/detail` 요청과 `volumes_links` 순회를 합니다. `all_projects=True`는 Python이 `all_tenants=True`로 보내므로 `WithListQuery("all_tenants", "True")`가 같은 문자열을 만듭니다. 하지만 `details=False`의 `GET volumes`는 v2 Go에 없어서 판정은 미완료입니다.

`find_volume`의 기본 동작은 `GET volumes/{x}`가 404이면 `GET volumes/detail?name=x`를 보냅니다. Go에서 `Find(resource.ID(x), resource.WithIgnoreMissing())`와 `Find(resource.Name(x))`를 차례로 부르면 같은 두 요청이 나갑니다. 그러나 Python은 400과 403에서도 목록으로 넘어가고 ID나 이름이 같은 행을 고르며, `details=False`와 `all_projects=True`도 받습니다. Go 연쇄 호출은 이 셋을 재현하지 못하고 v2 adapter는 `FindIdentity`를 지원하지 않으므로 미완료로 둡니다.

## Volume action

`extend_volume`, `retype_volume`, `set_volume_bootable_status`, `set_volume_image_metadata`, `attach_volume`은 v2 generated action이 같은 `POST volumes/{id}/action` 본문을 보냅니다. `retype_volume`의 기본 `migration_policy="never"`는 Go에서 `MigrationPolicyNever`를 직접 넣어야 본문에 들어갑니다. Python의 `new_type`은 Type 객체도 받지만 Go는 ID 문자열만 받습니다. `attach_volume`은 Python이 instance가 있으면 host_name을 버리고 둘 다 없으면 `ValueError`를 냅니다. Go는 받은 값을 그대로 보내므로 둘 중 하나만 채워야 합니다. 빈 mountpoint는 Python이 `""`로 보내지만 Go는 생략합니다. `set_volume_image_metadata`는 빈 map이면 두 쪽 모두 `{"metadata":{}}`를 보내며 nil map이면 Go는 null을 보냅니다.

`detach_volume`은 force가 없을 때 `Detach`가 같은 `os-detach` 본문을 보냅니다. `force=True`의 `os-force_detach`(attachment_id와 connector)는 v2 Go에 없어서 미완료입니다. `reset_volume_status`도 `ResetStatus`가 같은 action을 보내지만 `ResetStatusOpts.Status`에 omitempty가 없어서 status 없이 attach_status만 바꾸는 Python 호출을 재현하지 못합니다.

`upload_volume_to_image`는 Python이 `force`, `disk_format`, `container_format`을 false나 null이어도 항상 보내고, Go `UploadImageOpts`는 빈 값을 생략합니다. Python은 응답의 `os-volume_upload_image` dict를 그대로 돌려주고 Go는 같은 객체를 `VolumeImage`로 decode합니다. `create_image`는 이 action 뒤에 Glance 요청 없이 `Image.existing(id=image_id)`를 만듭니다. Go에서는 `UploadImage` 결과의 `ImageID`로 `image.Service.ExistingImageRecord`를 부르면 HTTP 없이 같은 record를 얻습니다. Python은 `disk_format`이 비면 cloud 설정의 `image_format`(기본 `qcow2`)을, `container_format`이 비면 `bare`를 쓰지만 v2 Go 호출은 설정을 읽지 않으므로 호출자가 값을 직접 넣습니다. 고정 Proxy가 받는 `wait`와 `timeout`은 Python도 사용하지 않습니다.

## Snapshot

`get_snapshot`은 같은 `GET snapshots/{id}`를 보냅니다. `create_snapshot`은 `is_forced`를 `force`로 바꾼 `snapshot` envelope을 보내고 Go `CreateOpts`가 같은 key를 씁니다. Python은 `is_forced=False`를 `"force": false`로 보내지만 Go는 false를 생략하고 `WithCreateField("force", false)`도 core field라서 거부합니다. 서버 기본값은 같습니다. Go `CreateOpts.VolumeID`는 필수라서 비어 있으면 요청 전에 오류가 납니다.

`snapshots`의 기본 `details=True`는 `GET snapshots/detail`인데 v2 Go `List`는 요약 경로 `GET snapshots`만 읽고, 그 응답도 `snapshots_links`를 따라가지 않는 한 페이지입니다. `find_snapshot`도 같은 이유로 Python의 `snapshots/detail?name=x` fallback을 만들 수 없습니다. `delete_snapshot`은 force가 없으면 `Remove(ctx, resource.ID(id))`가 404를 무시하는 같은 DELETE를 보내지만, `force=True`의 `os-force_delete` action이 v2 Go에 없어서 미완료입니다.

## 미완료 메서드

다음 메서드는 v2 경로에 대한 Go 호출이 없습니다. Gophercloud v2.15.0의 `blockstorage/v2` package에 binding이 없으므로 SDK 소유 v2 호출이 필요합니다.

- `set_volume_readonly`: `os-update_readonly_flag` action이 없습니다.
- `delete_volume_image_metadata`: `os-unset_image_metadata` action이 없습니다. 고정 Python은 keys가 None이면 image metadata가 아니라 일반 `metadata`를 순회합니다.
- `unmanage_volume`, `migrate_volume`, `complete_volume_migration`: `os-unmanage`, `os-migrate_volume`, `os-migrate_volume_completion` action이 없습니다.
- `update_snapshot`: `PUT snapshots/{id}`가 없습니다. `UpdateMetadata`는 `snapshots/{id}/metadata`라는 다른 경로입니다.
- `reset_snapshot_status`, `reset_snapshot`, `unmanage_snapshot`: snapshot의 `os-reset_status`, `os-unmanage` action이 없습니다.
- `manage_snapshot`: `POST os-snapshot-manage`가 없습니다.
- `delete_qos_spec_metadata`: v2 QoS package 자체가 없어서 `PUT qos-specs/{id}/delete_keys`를 보낼 수 없습니다.
- consistency group snapshot 다섯 메서드: v2 `cgsnapshots` binding이 없습니다. [v3 consistencygroupsnapshots](../v3/consistency-group-snapshots-python.md)는 같은 legacy 경로를 다루지만 v3 Proxy 대응으로만 검증되었습니다.

다음 메서드는 Go 호출이 있지만 기본값이나 인자를 재현하지 못합니다.

- `create_volume`: size를 생략할 수 없습니다.
- `volumes`, `snapshots`, `find_volume`, `find_snapshot`: 요약/상세 경로 선택, 400·403 fallback, ID 일치 검색과 `all_projects` 인자를 재현하지 못합니다.
- `reset_volume_status`: status를 생략할 수 없습니다.
- `detach_volume`, `delete_snapshot`: force 경로가 없습니다.
