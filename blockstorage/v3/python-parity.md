# Cinder v3 Python proxy 대응

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [block_storage v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py) 가운데 volume, snapshot, backup, attachment, transfer, volume type, QoS spec, availability zone 메서드 66개를 Go 호출과 비교합니다. 각 판정은 Python이 기본 인자와 문서화된 인자로 보내는 HTTP 요청을 [Proxy 공통 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)와 [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)에서 따라가 정했고, 기존 `python_parity_test.go`와 아래 개별 가이드가 참조하는 계약 테스트가 각 Go 요청의 비교 기준입니다.

`go_mapping`은 고정한 named Python 호출의 HTTP 동작·기본값·옵션에 대응하는 공개 Go API가 있다는 뜻입니다. 이때도 아래 절의 차이는 남습니다. `unresolved`는 Go에 경로, 본문 key, 기본값 변경 수단이나 helper 동작이 없어서 같은 요청을 만들 수 없다는 뜻입니다. 표의 Go 호출은 `conn.BlockStorageV3(ctx)`가 돌려준 service의 필드를 기준으로 적었습니다. 표는 공개 API의 구현 대응을 설명하며 전체 검증 완료나 Python Resource/session 전체 동등성을 뜻하지 않습니다. 최종 실행 결과는 [검증 기록](../../docs/sdk-support-ledger.md)에서 별도로 확인합니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `get_volume` | `Volumes.Get(ctx, id)` | go_mapping |
| `volumes` | `Volumes.List(ctx, options...)`, `ListSummary`, `ListWithDetails` | go_mapping |
| `create_volume` | `Volumes.Create(ctx, opts, WithCreateField, WithCreateHintOpts)` | go_mapping |
| `update_volume` | `Volumes.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `delete_volume` | `Volumes.Remove(ctx, resource.ID(id))`, `Volumes.Delete(ctx, id, WithDeleteQuery)`, `Volumes.ForceDelete(ctx, id)` | go_mapping |
| `init_volume_attachment` | `Volumes.InitializeWithConnector(ctx, id, connector)` ([가이드](connector-python.md)) | go_mapping |
| `terminate_volume_attachment` | `Volumes.TerminateWithConnector(ctx, id, connector)` ([가이드](connector-python.md)) | go_mapping |
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
| `fetch_backup_metadata` | `Backups.PythonMetadata().FetchMetadata` 명시적 Source 프로파일; stock child route 없음 | unresolved |
| `get_backup_metadata` | `Backups.PythonMetadata().GetMetadata` 명시적 Source 프로파일; stock child route 없음 | unresolved |
| `set_backup_metadata` | `Backups.PythonMetadata().SetMetadata` 명시적 Source 프로파일; stock child route 없음 | unresolved |
| `delete_backup_metadata` | `Backups.PythonMetadata().DeleteMetadata` 명시적 Source 프로파일; stock child route 없음 | unresolved |
| `create_attachment` | `Attachments.CreateRecord(ctx, volumeID, options...)` ([가이드](attachments-python.md)) | go_mapping |
| `get_attachment` | `Attachments.Get(ctx, id)` | go_mapping |
| `attachments` | `Attachments.ListRecords(ctx, options...)` ([가이드](attachments-python.md)) | go_mapping |
| `update_attachment` | `Attachments.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `complete_attachment` | `Attachments.Complete(ctx, id)` | go_mapping |
| `delete_attachment` | `Attachments.Remove(ctx, resource.ID(id))` | go_mapping |
| `create_transfer` | `Transfers.CreateRecord(ctx, options...)` ([가이드](transfers-python.md)) | go_mapping |
| `get_transfer` | `Transfers.GetRecord(ctx, id)` ([가이드](transfers-python.md)) | go_mapping |
| `transfers` | `Transfers.ListRecords(ctx, options...)` ([가이드](transfers-python.md)) | go_mapping |
| `find_transfer` | `Transfers.FindRecord(ctx, nameOrID, options...)` ([가이드](transfers-python.md)) | go_mapping |
| `accept_transfer` | `Transfers.AcceptRecord(ctx, id, transfers.WithRecordAcceptAuthKey(key))` ([가이드](transfers-python.md)) | go_mapping |
| `delete_transfer` | `Transfers.DeleteRecord(ctx, id, options...)` ([가이드](transfers-python.md)) | go_mapping |
| `get_type` | `VolumeTypes.Get(ctx, id)` | go_mapping |
| `find_type` | `VolumeTypes.FindRecord(ctx, nameOrID, options...)` ([가이드](type-encryption-python.md)) | go_mapping |
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
| `create_type_encryption` | `VolumeTypes.CreateEncryptionWithAttributes(ctx, typeID, options...)` ([가이드](type-encryption-python.md)) | go_mapping |
| `update_type_encryption` | `VolumeTypes.UpdateEncryptionFromType` 또는 `UpdateEncryptionRecord` ([가이드](type-encryption-python.md)) | go_mapping |
| `delete_type_encryption` | `VolumeTypes.GetEncryption`와 `VolumeTypes.DeleteEncryption(ctx, id, encryptionID)` | go_mapping |
| `create_qos_spec` | `QoS.Create(ctx, opts, WithCreateField)` | go_mapping |
| `get_qos_spec` | `QoS.Get(ctx, id)` | go_mapping |
| `find_qos_spec` | `QoS.FindRecord(ctx, nameOrID, options...)` ([가이드](type-encryption-python.md)) | go_mapping |
| `qos_specs` | `QoS.List(ctx, options...)` | go_mapping |
| `update_qos_spec` | `QoS.Update(ctx, id, opts, WithUpdateField)` | go_mapping |
| `delete_qos_spec` | `QoS.Remove(ctx, resource.ID(id))`, `QoS.Delete(ctx, id, WithDeleteQuery("force", ...))` | go_mapping |
| `associate_qos_spec` | `QoS.Associate(ctx, id, opts)` | go_mapping |
| `disassociate_qos_spec` | `QoS.Disassociate(ctx, id, opts)` | go_mapping |
| `disassociate_all_qos_spec` | `QoS.DisassociateAll(ctx, id)` | go_mapping |
| `delete_qos_spec_metadata` | `QoS.DeleteKeys(ctx, id, WithDeleteKeysOptions(keys))` | go_mapping |
| `qos_spec_associations` | `QoS.ListAssociations(ctx, id)` | go_mapping |
| `availability_zones` | `AvailabilityZones.List(ctx)` | go_mapping |

## 공통 차이

Python은 Resource 객체나 ID 문자열을 받지만 Go generated 호출은 ID 문자열을 받고, `Remove`와 `Find` 같은 collection 호출은 `resource.Ref`를 받습니다. generated 결과는 Python Resource 대신 Gophercloud typed model이라서 모델에 없는 응답 field는 남지 않습니다. 새 attachment·transfer·type/QoS/encryption record는 선언된 Python 속성의 `Resource`, 원본 unknown 필드를 포함한 `Wire`, 실제 `Envelope`·`Header`·`StatusCode`를 별도로 제공합니다. mutable Resource/session과 전체 cloud location은 별도 공통 범위이며 record의 location은 nullable null입니다.

Python delete 계열은 `ignore_missing=True`가 기본이라 404를 무시합니다. Go에서는 `Remove(ctx, resource.ID(id))`가 같은 기본값을 갖고 `resource.WithMissingError()`가 `ignore_missing=False`에 해당합니다. 다만 `Remove`는 query 옵션을 받지 않으므로 force나 cascade가 필요하면 generated `Delete`를 쓰고, 404를 무시하려면 `resource.IgnoreMissing(api.Delete(...))`를 사용합니다. 이 helper는 순수 404만 성공으로 바꾸고 read/Close·취소 등의 다른 원인이 섞인 실패는 보존합니다.

Python은 resource마다 `_max_microversion`과 서버 최대값 중 작은 값을 고르고, snapshot과 backup action은 각각 3.65와 3.64를 고정해 보냅니다. 기존 generated Go 호출은 Connection에 설정하거나 협상한 client microversion을 씁니다. 같은 header가 필요하면 `RawClient()`의 복사본에 `Microversion`을 바꿔 새 API를 만듭니다. `Backups.ResetBackupStatus`는 Python처럼 3.64를 강제합니다. 새 attachment create·transfer record는 공통 discovery로 capability와 미선택 버전을 관리하고 공유 client를 변경하지 않습니다. 명시된 버전과 Resource ceiling의 처리 차이는 각 가이드에 설명합니다. Connector는 명시된 버전이 있으면 그대로 쓰고 없으면 3.71 ceiling으로 discovery하며, type/QoS/encryption record는 captured client의 선택을 유지합니다.

Python의 update 계열은 바뀐 attribute가 없으면 요청을 보내지 않지만 Go `Update`는 빈 envelope라도 PUT을 보냅니다. Python 목록 query의 `all_projects=True`는 `all_tenants=True`가 되며 Go typed `AllTenants`는 `all_tenants=true`를 보내므로, 같은 문자열이 필요하면 `WithListQuery("all_tenants", "True")`를 씁니다. Python은 query mapping에 없는 Body attribute를 받으면 응답을 local로 거릅니다. 기존 native 목록에서는 caller가 local filter를 적용합니다. 새 record 목록·Find는 해당 Source의 서버 query와 Body filter를 분리합니다. 특히 Transfer는 Source가 `all_projects`를 버리므로 위의 일반 all_tenants 변환을 적용하지 않습니다.

## Volume

`get_volume`, `create_volume`, `update_volume`은 같은 경로와 본문을 보냅니다. `create_volume(scheduler_hints=...)`은 `WithCreateHintOpts`로 `OS-SCH-HNT:scheduler_hints`를 volume envelope 밖에 두고, `is_multiattach`처럼 `CreateOpts`에 없는 attribute는 `WithCreateField("multiattach", true)`로 wire 이름을 직접 줍니다.

`delete_volume`은 microversion 3.23 이상에서 `cascade=False&force=False`를 항상 query로 보냅니다. Go `Remove`는 query 없이 DELETE를 보내며 서버 기본값은 같습니다. 같은 query가 필요하면 `Delete(ctx, id, WithDeleteQuery("cascade", "False"), WithDeleteQuery("force", "False"))`를 쓰고 404는 직접 거릅니다. 3.23 미만의 `force=True`는 `ForceDelete`에 대응하며 본문 값이 Python의 null 대신 `""`입니다.

`volumes`는 기본 `details=True`에서 `Volumes.List`와 같은 `volumes/detail` 요청과 `volumes_links` 순회를 합니다. `details=False`는 `Volumes.ListSummary` 또는 `ListWithDetails(ctx, false, options...)`의 `GET /volumes`를 사용합니다. native cap·페이지 차이는 [Volume summary 비교](volume-summary-python.md)에 설명합니다.

`InitializeWithConnector`·`TerminateWithConnector`는 임의 connector dict를 nested `connector`에 보냅니다. `wwnns` 배열, `nqn`, os-brick/backend 확장과 명시 null·false·0·빈 값을 보존합니다. `VolumeConnectorResult`는 실제 본문·헤더·status를 유지하며 공통 source capture와 JSON snapshot을 사용합니다. 기존 typed `InitializeConnection`·`TerminateConnection`의 여덟 필드 한계는 유지됩니다. [Connector Python/Go 비교](connector-python.md)에 옵션과 반환 값을 설명합니다.

## Snapshot

`snapshots`는 `details`에 따라 `ListDetail` 또는 `List`를 고르면 되고, 두 호출 모두 `snapshots_links`를 따라갑니다. `create_snapshot`의 `is_forced=False`는 Python이 `"force": false`로 보내지만 Go는 false를 생략하며 서버 기본값은 같습니다.

`delete_snapshot`은 `Remove`로 대응합니다. `force=True`는 Python이 `ignore_missing`을 적용하지 않고 `os-force_delete` action을 3.65로 보내는 경로라서 `ForceDelete`가 같은 action을 보냅니다. 본문 값은 Python의 null 대신 `{}`입니다. `reset_snapshot_status`, 그 별칭 `reset_snapshot`, `set_snapshot_status`는 같은 action 본문을 보내며 `progress`가 없으면 두 쪽 모두 생략합니다. Python은 이 세 action과 force delete에 3.65를 고정하므로 Go에서 같은 header를 원하면 client microversion을 3.65로 맞춥니다.

## Backup

`backups`의 기본 상세 목록은 `Backups.ListDetail`, `details=False`는 `Backups.List`입니다. `ListDetailOpts`에는 이름·상태·volume 필터가 없으므로 `WithListDetailQuery`로 더합니다. `create_backup`은 Python이 `is_incremental`을 `incremental`로 바꿔 보내므로 Go `CreateOpts.Incremental`과 같은 본문이 됩니다.

`delete_backup(force=True)`는 3.64로 `os-force_delete`를 보내며 Go `ForceDelete`는 client microversion과 `{}` 본문을 씁니다. `reset_backup`은 `reset_backup_status`의 deprecated 별칭이라 이미 검토된 `Backups.ResetBackupStatus`를 그대로 씁니다.

`fetch_backup_metadata`, `get_backup_metadata`, `set_backup_metadata`, `delete_backup_metadata`는 MetadataMixin의 `/backups/{id}/metadata` 경로를 씁니다. `Backups.PythonMetadata()`가 명시적 Source compatibility profile을 제공합니다. stock Cinder 27.0.0에는 해당 child route가 없으므로 네 선언은 일반 stock mapping에서 unresolved로 유지합니다. `Backups.Update`의 `Metadata`는 backup 본문 전체를 PUT하는 다른 요청입니다. [선택적 프로파일의 경계](backup-metadata-python.md)를 확인합니다.

## Attachment

`get_attachment`, `update_attachment`, `delete_attachment`는 같은 요청을 보냅니다. Go `UpdateOpts.Connector`는 omitempty가 없어서 connector 없이 호출하면 `"connector": null`을 보냅니다. `complete_attachment`는 Python이 `{"os-complete": "<attachment id>"}`, Go가 `{"os-complete": null}`을 보냅니다. 대상 attachment는 두 쪽 모두 URL의 ID로 정해집니다. attachment API는 3.27 이상이 필요하므로 Go client에 그 이상의 microversion을 설정해야 합니다.

`Attachments.CreateRecord(ctx, volumeID, options...)`는 instance 없는 reservation의 `instance_uuid`를 생략합니다. 옵션으로 임의 connector와 mode를 보내고 mode capability 판단은 요청 버전과 별도로 수행합니다. `ListRecords`는 Source summary `GET /attachments`, 서버 query·Body 필터, `attachments_links`와 marker fallback, 필터 전 raw cap을 제공합니다. 기존 generated `Create`와 native `/attachments/detail` `List`의 입력·페이지 한계는 유지됩니다. [Attachment 비교](attachments-python.md)에 optional instance, discovery와 실제 ACK·raw record 계약을 설명합니다.

## Transfer

`Transfers.CreateRecord`·`GetRecord`·`ListRecords`·`FindRecord`·`AcceptRecord`·`DeleteRecord`는 공통 Cinder discovery와 선택한 버전으로 `/volume-transfers` 또는 `/os-volume-transfer`를 결정합니다. `ListRecords`의 기본 details=true, `WithRecordListDetails(false)`의 요약 경로와 lazy paging을 제공하고, `FindRecord`는 clean 400·403·404 뒤 정확한 ID/이름 전체 검색을 수행합니다. raw `created_at`과 큰 JSON 숫자는 typed decoder로 손실되지 않습니다. native transfer 호출은 legacy 경로와 기존 typed decoder 계약을 유지합니다.

고정 Source에는 `all_projects`가 실제로 무시되는 동작, legacy create에도 입력 `no_snapshots`가 남는 동작, Find fallback이 legacy 선택에서도 `/volume-transfers` summary를 쓰는 동작이 있습니다. 새 기본 profile은 이를 재현합니다. `WithRecordCreateSuppressLegacySnapshots(true)`와 `WithRecordListServerQuery`는 명시적 Go 개선/확장이며 Source 기본 동작과 구분합니다. [Transfer 여섯 호출의 비교](transfers-python.md)에 경로 선택·omission·부분 결과·raw view와 제약을 설명합니다.

## Volume type

`types`는 query가 없으면 `is_public`을 보내지 않고, Cinder는 이를 공개 type만 보라는 뜻으로 읽습니다. Go `ListOpts`는 `IsPublic`이 비면 `is_public=None`을 보내 비공개 type까지 요청하므로 Python 기본값과 같은 결과를 원하면 `ListOpts{IsPublic: VisibilityPublic}`을 줍니다. `is_public="none"` 같은 Python query는 `WithListQuery("is_public", "none")`이 대신합니다. Go는 단수형 `volume_type_links`를 따라가고 Python은 `limit`이 있을 때 marker로 이어 가므로 여러 페이지 동작은 다를 수 있습니다.

`update_type_extra_specs`는 `POST types/{id}/extra_specs`이며 Go `CreateExtraSpecs`가 같은 요청을 보냅니다. Python은 attribute가 없으면 요청 없이 빈 extra_specs를 돌려주고 Go는 빈 map도 POST합니다. 이름이 비슷한 Go `UpdateExtraSpec`은 key 하나를 PUT하는 다른 경로입니다. `delete_type_extra_specs`는 key마다 DELETE를 보내므로 Go도 `DeleteExtraSpec`을 반복합니다. `get_type_access`는 Python이 dict 목록을, Go가 `VolumeTypeAccess{VolumeTypeID, ProjectID}`를 돌려줍니다.

`delete_type_encryption(volume_type=...)`은 encryption을 GET한 뒤 `encryption_id`로 DELETE합니다. Go도 `GetEncryption` 결과의 `EncryptionID`로 `DeleteEncryption`을 부르고 `resource.IgnoreMissing`으로 순수 404를 처리합니다. 새 `CreateEncryptionWithAttributes`는 명시한 속성만 POST합니다. `UpdateEncryptionFromType`는 GET 후 바뀐 속성만 PUT하고, `UpdateEncryptionRecord`는 이미 받은 record를 복사해 사용합니다. omission·null·0·false·빈 값을 구분하고 빈/동일 변경에는 PUT을 보내지 않습니다. 기존 generated encryption 옵션의 강제 기본값은 이 새 API에 적용되지 않습니다. [Type encryption·Find 비교](type-encryption-python.md)에 raw 응답과 dirty 판단의 범위를 설명합니다.

## QoS spec

`create_qos_spec`과 `update_qos_spec`은 spec key를 `qos_specs` 객체에 평평하게 보냅니다. Go `Specs` map이 같은 위치에 들어가고 `Update`는 응답의 `qos_specs` map을 돌려줍니다. `delete_qos_spec`은 항상 `force=False`를 query로 보냅니다. `Remove`는 force 없이 404를 무시하고, 같은 query가 필요하면 `Delete(ctx, id, WithDeleteQuery("force", "False"))`를 씁니다. association 세 호출은 Python과 마찬가지로 GET이며 `delete_qos_spec_metadata`는 `PUT qos-specs/{id}/delete_keys`입니다.

## Availability zone

`availability_zones`는 같은 `GET os-availability-zone` 한 번을 보냅니다. Go `ZoneState`는 `available`만 decode하고 Python `state`는 dict 전체를 보존합니다.

## find 계열

Python `find_*`는 ID로 GET을 먼저 보내고 404·400·403이면 목록에서 ID나 이름이 같은 항목을 하나 고릅니다. 결과가 없으면 기본값 `ignore_missing=True`에 따라 None을 돌려줍니다. `VolumeTypes.FindRecord`, `QoS.FindRecord`, `Transfers.FindRecord`가 이 GET-first 검색을 소유합니다. Type GET와 fallback 목록에는 모두 `is_public=none`을 보내며, QoS는 `**query`의 GET kwargs와 목록의 서버 query·local Body filter를 분리합니다. 기본 missing은 nil,nil이고 중복·뒤 page 오류와 원래 HTTP/물리 원인을 보존합니다. 구현별 query·descriptor 차이는 [Type/QoS](type-encryption-python.md)와 [Transfer](transfers-python.md) 가이드에 설명합니다. Snapshot과 Backup의 Source Find는 아직 unresolved이며 기존 `Find(resource.ID(...))`와 `Find(resource.Name(...))`를 합쳐도 전체 Source 계약의 대응으로 세지 않습니다.

`cascade`·`force` query를 보내는 volume·QoS spec 삭제와 encryption 삭제는 `Remove`를 쓸 수 없으므로 `resource.IgnoreMissing(api.Delete(...))`처럼 직접 호출을 감싸 Python 기본값 `ignore_missing=True`를 고릅니다. 이 helper는 순수 404만 성공으로 바꾸고 다른 실패는 그대로 돌려줍니다.

## 권한 조건

이번에 대응한 기존 누락 9개와 connector 2개에는 user·admin·서비스 내부용 API가 함께 있습니다. [Cinder 2025.2 기본 정책](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html)과 27.0.0 정책 소스를 기준으로 다음을 구분합니다.

- Type Find는 project reader 경로이고 Transfer의 일반 조회는 reader, 생성·수락·삭제는 member 경로입니다. init/terminate connector도 `xena_system_admin_or_project_member`이므로 low-level 연결 API라는 이유로 admin-only로 분류하지 않습니다.
- Type encryption create/get/update와 QoS GET/list는 `admin_api`입니다. 따라서 Type/QoS 네 신규 mapping은 user 1개·admin 3개입니다.
- Attachment create/list는 [고정 Python proxy가 내부 서비스용으로 설명](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1495-L1552)합니다. 실제 create 정책은 project member를 허용하고 [27.0.0 list controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/attachments.py#L65-L106)는 일반 project 목록과 관리자 all_tenants를 나눕니다. 서비스용 사용 의도와 service-role-only 권한을 동일하게 취급하지 않습니다.

서버 정책의 기준 release는 SDK source pin과 별개입니다. 배포 override·리소스 소유권·상태는 서버가 판단하며, 이 핵심 서비스의 합산 mapping 수를 모두 user 단계 완료 수로 표시하지 않습니다.

## unresolved 목록과 공통 한계

- `find_snapshot`, `find_backup`: Source의 GET-first clean 400·403·404 목록 fallback과 전체 검색 옵션을 소유하는 공개 helper가 없습니다.
- `fetch_backup_metadata`, `get_backup_metadata`, `set_backup_metadata`, `delete_backup_metadata`: 명시적 `PythonMetadata()` profile은 있지만 stock Cinder 27.0.0 child route가 없어 일반 stock 지원 mapping으로 더하지 않습니다.

표에서 새 mapping으로 표시한 named API에도 mutable Python Resource/session, 전체 cloud location, adapter discovery cache와 인증된 실제 클라우드 acceptance 동등성은 남습니다. native API와 새 record API의 버전·JSON·페이지·오류 차이는 [Connector](connector-python.md), [Attachment](attachments-python.md), [Transfer](transfers-python.md), [Type/QoS](type-encryption-python.md)에 유지합니다.

현재 summary 경로와 native cap 사용법은 [Volume summary 비교](volume-summary-python.md), stock Cinder 경계를 보존한 선택적 Backup child route는 [명시적 Python compatibility profile](backup-metadata-python.md)에 설명합니다. 이 선택적 프로파일 4개는 일반 stock 지원 mapping 수에 더하지 않습니다.
