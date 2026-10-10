# Cinder v3 Python proxy 대응

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [block_storage v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py) 가운데 volume, snapshot, backup, attachment, transfer, volume type, QoS spec, availability zone 메서드 66개를 Go 호출과 비교합니다. 각 판정은 Python이 기본 인자와 문서화된 인자로 보내는 HTTP 요청을 [Proxy 공통 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)와 [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)에서 따라가 정했고, 각 package의 `python_parity_test.go`가 Go 쪽 요청을 고정합니다.

`go_mapping`은 모든 Python 인자의 효과를 공개 Go API로 재현할 수 있다는 뜻입니다. 이때도 아래 절의 차이는 남습니다. `unresolved`는 Go에 경로, 본문 key, 기본값 변경 수단이나 helper 동작이 없어서 같은 요청을 만들 수 없다는 뜻입니다. 표의 Go 호출은 `conn.BlockStorageV3(ctx)`가 돌려준 service의 필드를 기준으로 적었습니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `get_volume` | `Volumes.Get(ctx, id)` | go_mapping |
| `volumes` | `Volumes.List(ctx, options...)` (상세 경로만 가능) | unresolved |
| `create_volume` | `Volumes.Create(ctx, opts, WithCreateField, WithCreateHintOpts)` | go_mapping |
| `update_volume` | `Volumes.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `delete_volume` | `Volumes.Remove(ctx, resource.ID(id))`, `Volumes.Delete(ctx, id, WithDeleteQuery)`, `Volumes.ForceDelete(ctx, id)` | unresolved |
| `init_volume_attachment` | `Volumes.InitializeConnection(ctx, id, opts)` | unresolved |
| `terminate_volume_attachment` | `Volumes.TerminateConnection(ctx, id, opts)` | unresolved |
| `get_snapshot` | `Snapshots.Get(ctx, id)` | go_mapping |
| `find_snapshot` | `Snapshots.Find(ctx, resource.ID(x))`와 `resource.Name(x)`를 따로 호출 | unresolved |
| `snapshots` | `Snapshots.ListDetail(ctx, options...)`, `Snapshots.List(ctx, options...)` | go_mapping |
| `create_snapshot` | `Snapshots.Create(ctx, opts, WithCreateField)` | go_mapping |
| `update_snapshot` | `Snapshots.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `delete_snapshot` | `Snapshots.Remove(ctx, resource.ID(id))`, `Snapshots.ForceDelete(ctx, id)` | go_mapping |
| `reset_snapshot_status` | `Snapshots.ResetStatus(ctx, id, opts)` | go_mapping |
| `reset_snapshot` | `Snapshots.ResetStatus(ctx, id, opts)` | go_mapping |
| `set_snapshot_status` | `Snapshots.UpdateStatus(ctx, id, opts)` | go_mapping |
| `backups` | `Backups.ListDetail(ctx, options...)`, `Backups.List(ctx, options...)` | go_mapping |
| `get_backup` | `Backups.Get(ctx, id)` | go_mapping |
| `find_backup` | `Backups.Find(ctx, resource.ID(x))`와 `resource.Name(x)`를 따로 호출 | unresolved |
| `create_backup` | `Backups.Create(ctx, opts, WithCreateField)` | go_mapping |
| `update_backup` | `Backups.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `delete_backup` | `Backups.Remove(ctx, resource.ID(id))`, `Backups.ForceDelete(ctx, id)` | go_mapping |
| `reset_backup` | `Backups.ResetBackupStatus(ctx, id, status)` | go_mapping |
| `fetch_backup_metadata` | 없음 | unresolved |
| `get_backup_metadata` | 없음 | unresolved |
| `set_backup_metadata` | 없음 | unresolved |
| `delete_backup_metadata` | 없음 | unresolved |
| `create_attachment` | `Attachments.Create(ctx, opts)` | unresolved |
| `get_attachment` | `Attachments.Get(ctx, id)` | go_mapping |
| `attachments` | `Attachments.List(ctx, options...)` (상세 경로 첫 페이지만) | unresolved |
| `update_attachment` | `Attachments.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `complete_attachment` | `Attachments.Complete(ctx, id)` | go_mapping |
| `delete_attachment` | `Attachments.Remove(ctx, resource.ID(id))` | go_mapping |
| `create_transfer` | `Transfers.Create(ctx, opts, WithCreateField)` | unresolved |
| `get_transfer` | `Transfers.Get(ctx, id)` | go_mapping |
| `transfers` | `Transfers.List(ctx, options...)` (상세 경로만 가능) | unresolved |
| `find_transfer` | `Transfers.Find(ctx, resource.ID(x))`와 `resource.Name(x)`를 따로 호출 | unresolved |
| `accept_transfer` | `Transfers.Accept(ctx, id, opts)` | go_mapping |
| `delete_transfer` | `Transfers.Remove(ctx, resource.ID(id))` | go_mapping |
| `get_type` | `VolumeTypes.Get(ctx, id)` | go_mapping |
| `find_type` | `VolumeTypes.Find(ctx, resource.ID(x))`와 `resource.Name(x)`를 따로 호출 | unresolved |
| `types` | `VolumeTypes.List(ctx, WithListOptions(ListOpts{IsPublic: VisibilityPublic}), ...)` | go_mapping |
| `create_type` | `VolumeTypes.Create(ctx, opts, WithCreateField)` | go_mapping |
| `update_type` | `VolumeTypes.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `delete_type` | `VolumeTypes.Remove(ctx, resource.ID(id))` | go_mapping |
| `update_type_extra_specs` | `VolumeTypes.CreateExtraSpecs(ctx, id, specs)` | go_mapping |
| `delete_type_extra_specs` | key마다 `VolumeTypes.DeleteExtraSpec(ctx, id, key)` | go_mapping |
| `get_type_access` | `VolumeTypes.ListAccesses(ctx, id)` | go_mapping |
| `add_type_access` | `VolumeTypes.AddAccess(ctx, id, opts)` | go_mapping |
| `remove_type_access` | `VolumeTypes.RemoveAccess(ctx, id, opts)` | go_mapping |
| `get_type_encryption` | `VolumeTypes.GetEncryption(ctx, id)` | go_mapping |
| `create_type_encryption` | `VolumeTypes.CreateEncryption(ctx, id, opts)` | unresolved |
| `update_type_encryption` | `VolumeTypes.GetEncryption`와 `VolumeTypes.UpdateEncryption(ctx, id, encryptionID, opts)` | unresolved |
| `delete_type_encryption` | `VolumeTypes.GetEncryption`와 `VolumeTypes.DeleteEncryption(ctx, id, encryptionID)` | unresolved |
| `create_qos_spec` | `QoS.Create(ctx, opts, WithCreateField)` | go_mapping |
| `get_qos_spec` | `QoS.Get(ctx, id)` | go_mapping |
| `find_qos_spec` | `QoS.Find(ctx, resource.ID(x))`와 `resource.Name(x)`를 따로 호출 | unresolved |
| `qos_specs` | `QoS.List(ctx, options...)` | go_mapping |
| `update_qos_spec` | `QoS.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `delete_qos_spec` | `QoS.Remove(ctx, resource.ID(id))`, `QoS.Delete(ctx, id, WithDeleteQuery("force", ...))` | unresolved |
| `associate_qos_spec` | `QoS.Associate(ctx, id, opts)` | go_mapping |
| `disassociate_qos_spec` | `QoS.Disassociate(ctx, id, opts)` | go_mapping |
| `disassociate_all_qos_spec` | `QoS.DisassociateAll(ctx, id)` | go_mapping |
| `delete_qos_spec_metadata` | `QoS.DeleteKeys(ctx, id, WithDeleteKeysOptions(keys))` | go_mapping |
| `qos_spec_associations` | `QoS.ListAssociations(ctx, id)` | go_mapping |
| `availability_zones` | `AvailabilityZones.List(ctx)` | go_mapping |

## 공통 차이

Python은 Resource 객체나 ID 문자열을 받지만 Go generated 호출은 ID 문자열을 받고, `Remove`와 `Find` 같은 collection 호출은 `resource.Ref`를 받습니다. 결과도 Python Resource 대신 Gophercloud typed model이라서 모델에 없는 응답 field는 남지 않습니다.

Python delete 계열은 `ignore_missing=True`가 기본이라 404를 무시합니다. Go에서는 `Remove(ctx, resource.ID(id))`가 같은 기본값을 갖고 `resource.WithMissingError()`가 `ignore_missing=False`에 해당합니다. 다만 `Remove`는 query 옵션을 받지 않으므로 force나 cascade가 필요하면 generated `Delete`를 쓰고, 404를 무시하려면 `gophercloud.ResponseCodeIs(err, http.StatusNotFound)`로 직접 걸러야 합니다.

Python은 resource마다 `_max_microversion`과 서버 최대값 중 작은 값을 고르고, snapshot과 backup action은 각각 3.65와 3.64를 고정해 보냅니다. Go는 Connection에 설정하거나 협상한 client microversion 하나를 모든 호출에 씁니다. 그래서 같은 header가 필요하면 `RawClient()`의 복사본에 `Microversion`을 바꿔 새 API를 만들어야 합니다. 예외로 `Backups.ResetBackupStatus`는 Python처럼 3.64를 강제합니다.

Python의 update 계열은 바뀐 attribute가 없으면 요청을 보내지 않지만 Go `Update`는 빈 envelope라도 PUT을 보냅니다. Python 목록 query의 `all_projects=True`는 `all_tenants=True`가 되며 Go typed `AllTenants`는 `all_tenants=true`를 보내므로, 같은 문자열이 필요하면 `WithListQuery("all_tenants", "True")`를 씁니다. Python은 query mapping에 없는 Body attribute를 받으면 응답을 local로 거릅니다. 이 Go 목록들에는 그런 local filter가 없으므로 caller가 결과를 직접 걸러야 합니다.

## Volume

`get_volume`, `create_volume`, `update_volume`은 같은 경로와 본문을 보냅니다. `create_volume(scheduler_hints=...)`은 `WithCreateHintOpts`로 `OS-SCH-HNT:scheduler_hints`를 volume envelope 밖에 두고, `is_multiattach`처럼 `CreateOpts`에 없는 attribute는 `WithCreateField("multiattach", true)`로 wire 이름을 직접 줍니다.

`delete_volume`은 microversion 3.23 이상에서 `cascade=False&force=False`를 항상 query로 보냅니다. Go `Remove`는 query 없이 DELETE를 보내며 서버 기본값은 같습니다. 같은 query가 필요하면 `Delete(ctx, id, WithDeleteQuery("cascade", "False"), WithDeleteQuery("force", "False"))`를 쓰고 404는 직접 거릅니다. 3.23 미만의 `force=True`는 `ForceDelete`에 대응하며 본문 값이 Python의 null 대신 `""`입니다.

`volumes`는 기본 `details=True`에서 `Volumes.List`와 같은 `volumes/detail` 요청과 `volumes_links` 순회를 합니다. 다만 `details=False`의 `GET /volumes` 요약 경로는 공개 Go 목록 호출이 없어서 unresolved입니다.

`init_volume_attachment`와 `terminate_volume_attachment`는 임의 connector dict를 보냅니다. Go `InitializeConnectionOpts`와 `TerminateConnectionOpts`는 ip, host, initiator, wwpns, wwnns, multipath, platform, os_type만 표현하고 `wwnns`는 문자열입니다. extension field는 connector 안이 아니라 action 객체에 붙으므로 nqn 같은 os-brick connector key를 보낼 수 없습니다.

## Snapshot

`snapshots`는 `details`에 따라 `ListDetail` 또는 `List`를 고르면 되고, 두 호출 모두 `snapshots_links`를 따라갑니다. `create_snapshot`의 `is_forced=False`는 Python이 `"force": false`로 보내지만 Go는 false를 생략하며 서버 기본값은 같습니다.

`delete_snapshot`은 `Remove`로 대응합니다. `force=True`는 Python이 `ignore_missing`을 적용하지 않고 `os-force_delete` action을 3.65로 보내는 경로라서 `ForceDelete`가 같은 action을 보냅니다. 본문 값은 Python의 null 대신 `{}`입니다. `reset_snapshot_status`, 그 별칭 `reset_snapshot`, `set_snapshot_status`는 같은 action 본문을 보내며 `progress`가 없으면 두 쪽 모두 생략합니다. Python은 이 세 action과 force delete에 3.65를 고정하므로 Go에서 같은 header를 원하면 client microversion을 3.65로 맞춥니다.

## Backup

`backups`의 기본 상세 목록은 `Backups.ListDetail`, `details=False`는 `Backups.List`입니다. `ListDetailOpts`에는 이름·상태·volume 필터가 없으므로 `WithListDetailQuery`로 더합니다. `create_backup`은 Python이 `is_incremental`을 `incremental`로 바꿔 보내므로 Go `CreateOpts.Incremental`과 같은 본문이 됩니다.

`delete_backup(force=True)`는 3.64로 `os-force_delete`를 보내며 Go `ForceDelete`는 client microversion과 `{}` 본문을 씁니다. `reset_backup`은 `reset_backup_status`의 deprecated 별칭이라 이미 검토된 `Backups.ResetBackupStatus`를 그대로 씁니다.

`fetch_backup_metadata`, `get_backup_metadata`, `set_backup_metadata`, `delete_backup_metadata`는 MetadataMixin의 `/backups/{id}/metadata` 경로를 씁니다. Go에는 backup metadata 하위 경로 호출이 없습니다. `Backups.Update`의 `Metadata`는 backup 본문 전체를 PUT하는 다른 요청이라 대응으로 보지 않았습니다.

## Attachment

`get_attachment`, `update_attachment`, `delete_attachment`는 같은 요청을 보냅니다. Go `UpdateOpts.Connector`는 omitempty가 없어서 connector 없이 호출하면 `"connector": null`을 보냅니다. `complete_attachment`는 Python이 `{"os-complete": "<attachment id>"}`, Go가 `{"os-complete": null}`을 보냅니다. 대상 attachment는 두 쪽 모두 URL의 ID로 정해집니다. attachment API는 3.27 이상이 필요하므로 Go client에 그 이상의 microversion을 설정해야 합니다.

`create_attachment`는 instance 없이 호출하면 `instance_uuid`를 생략합니다. Go `CreateOpts.InstanceUUID`는 항상 보내고 core field라 extension으로 지울 수도 없어서 unresolved입니다. `attachments`는 Python이 `GET /attachments` 요약 경로와 `attachments_links`를 쓰지만 Go `List`는 `attachments/detail`만 요청하고 첫 페이지에서 멈춥니다.

## Transfer

Python은 microversion 3.55를 쓸 수 있으면 `/volume-transfers`, 아니면 `/os-volume-transfer`를 씁니다. Go transfer 호출은 항상 legacy 경로 `/os-volume-transfer`를 씁니다. 그래서 `get_transfer`, `accept_transfer`, `delete_transfer`는 Python이 3.55 미만으로 보내는 요청과 같고, 3.55 이상 Python 요청과는 경로가 다릅니다. Go는 응답의 `created_at`이 시간대 없는 형식일 때만 decode합니다.

`create_transfer`의 `no_snapshots`는 3.55 경로에서만 의미가 있는데 Go는 그 경로로 보낼 수 없습니다. `transfers(details=False)`의 요약 목록도 Go에 없습니다. `find_transfer`는 Python이 ID GET 뒤 전체 목록에서 ID나 이름을 찾지만 Go `Find`는 한 번에 ID 또는 이름 한 가지만 찾습니다.

## Volume type

`types`는 query가 없으면 `is_public`을 보내지 않고, Cinder는 이를 공개 type만 보라는 뜻으로 읽습니다. Go `ListOpts`는 `IsPublic`이 비면 `is_public=None`을 보내 비공개 type까지 요청하므로 Python 기본값과 같은 결과를 원하면 `ListOpts{IsPublic: VisibilityPublic}`을 줍니다. `is_public="none"` 같은 Python query는 `WithListQuery("is_public", "none")`이 대신합니다. Go는 단수형 `volume_type_links`를 따라가고 Python은 `limit`이 있을 때 marker로 이어 가므로 여러 페이지 동작은 다를 수 있습니다.

`update_type_extra_specs`는 `POST types/{id}/extra_specs`이며 Go `CreateExtraSpecs`가 같은 요청을 보냅니다. Python은 attribute가 없으면 요청 없이 빈 extra_specs를 돌려주고 Go는 빈 map도 POST합니다. 이름이 비슷한 Go `UpdateExtraSpec`은 key 하나를 PUT하는 다른 경로입니다. `delete_type_extra_specs`는 key마다 DELETE를 보내므로 Go도 `DeleteExtraSpec`을 반복합니다. `get_type_access`는 Python이 dict 목록을, Go가 `VolumeTypeAccess{VolumeTypeID, ProjectID}`를 돌려줍니다.

`delete_type_encryption(volume_type=...)`은 encryption을 GET한 뒤 `encryption_id`로 DELETE합니다. Go도 `GetEncryption` 결과의 `EncryptionID`로 `DeleteEncryption`을 부르고 404는 직접 거릅니다. `create_type_encryption`과 `update_type_encryption`은 Python이 넘긴 attribute만 보냅니다. Go `CreateEncryptionOpts`는 cipher, key_size, control_location을, `UpdateEncryptionOpts`는 네 field를 항상 보내므로 생략한 값이 `""`나 0으로 저장되거나 덮어써질 수 있어서 unresolved입니다.

## QoS spec

`create_qos_spec`과 `update_qos_spec`은 spec key를 `qos_specs` 객체에 평평하게 보냅니다. Go `Specs` map이 같은 위치에 들어가고 `Update`는 응답의 `qos_specs` map을 돌려줍니다. `delete_qos_spec`은 항상 `force=False`를 query로 보냅니다. `Remove`는 force 없이 404를 무시하고, 같은 query가 필요하면 `Delete(ctx, id, WithDeleteQuery("force", "False"))`를 씁니다. association 세 호출은 Python과 마찬가지로 GET이며 `delete_qos_spec_metadata`는 `PUT qos-specs/{id}/delete_keys`입니다.

## Availability zone

`availability_zones`는 같은 `GET os-availability-zone` 한 번을 보냅니다. Go `ZoneState`는 `available`만 decode하고 Python `state`는 dict 전체를 보존합니다.

## find 계열

Python `find_*`는 ID로 GET을 먼저 보내고 404·400·403이면 목록에서 ID나 이름이 같은 항목을 하나 고릅니다. 결과가 없으면 기본값 `ignore_missing=True`에 따라 None을 돌려줍니다. 이 cinder3 범위의 snapshot, backup, type, QoS, transfer에는 Volumes의 `FindIdentity` 같은 helper가 없습니다. 그래서 `Find(resource.ID(x), resource.WithIgnoreMissing())`와 `Find(resource.Name(x))`를 caller가 이어 붙여야 하며, 400·403 fallback, `details`·`all_projects`, `find_type`의 `is_public=none`, `find_qos_spec`의 `**query`는 `Find`에 전달할 수 없습니다. Snapshot과 type의 이름 목록은 `name` query를 보내고 backup, QoS, transfer는 전체 목록을 받아 이름을 비교합니다.

## unresolved 목록

- `volumes`: `details=False`의 `GET /volumes` 요약 목록을 보내는 공개 Go 호출이 없습니다.
- `init_volume_attachment`, `terminate_volume_attachment`: connector는 Gophercloud의 typed field 8개만 보낼 수 있고 `wwnns`가 문자열이라서 nqn 같은 os-brick key나 목록형 wwnns를 보낼 수 없습니다.
- `find_snapshot`, `find_backup`, `find_type`, `find_qos_spec`, `find_transfer`: ID GET 뒤 목록 fallback을 하는 helper가 없고 `details`, `all_projects`, `is_public=none`, `**query`를 `Find`에 전달할 수 없습니다.
- `fetch_backup_metadata`, `get_backup_metadata`, `set_backup_metadata`, `delete_backup_metadata`: `/backups/{id}/metadata` 경로를 호출하는 Go API가 없습니다.
- `create_attachment`: instance 없는 요청에서 `instance_uuid`를 생략할 수 없습니다.
- `attachments`: `GET /attachments` 요약 경로가 없고 Go 목록은 `attachments_links`를 따라가지 않습니다.
- `create_transfer`: `/volume-transfers` 경로와 3.55의 `no_snapshots` 생성을 보낼 수 없습니다.
- `transfers`: `details=False` 요약 목록과 `/volume-transfers` 경로가 없습니다.
- `create_type_encryption`, `update_type_encryption`: 넘기지 않은 encryption field를 생략할 수 없어서 부분 생성과 부분 수정이 불가능합니다.
- `delete_volume`, `delete_qos_spec`: `cascade`·`force` query를 보낼 때는 `Remove`를 쓸 수 없고 generated `Delete`에는 404를 무시하는 옵션이 없습니다. 그래서 Python 기본값 `ignore_missing=True`를 호출자가 404를 직접 걸러야만 재현할 수 있습니다.
- `delete_type_encryption`: `DeleteEncryption`에는 `Remove`나 404 무시 옵션이 없어서 Python 기본값 `ignore_missing=True`를 호출자가 직접 처리해야 합니다.
