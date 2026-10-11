# Cinder v2 Python proxy 대응: volume·snapshot 외 리소스

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [block_storage v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py)에서 volume과 snapshot을 제외한 메서드 66개를 Go 호출과 비교합니다. 대상은 backup, transfer, limit, quota, backend pool, service, extension, volume type, QoS spec, consistency group입니다. 각 판정은 Python이 기본 인자와 문서화된 인자로 보내는 HTTP 요청을 [Proxy 공통 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)와 [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)에서 따라가 정했습니다. 비교 기준이 되는 테스트는 `backups`, `transfers`, `limits`, `quotasets`, `schedulerstats`, `services` 패키지의 `python_parity_test.go`입니다.

`go_mapping`은 `blockstorage/v2` 아래의 공개 Go API가 같은 요청을 보내고 같은 정보를 돌려준다는 뜻이며, 이때도 아래 절에 적은 차이는 남습니다. `unresolved`는 v2 package에 경로, 옵션, helper 동작이 없어서 Python이 보내는 요청이나 결과를 만들 수 없다는 뜻입니다. Cinder v3 package와 root `blockstorage` helper는 v3 discovery나 microversion header를 전제로 하므로 v2 대응으로 세지 않습니다. 표의 Go 호출은 `service, err := conn.BlockStorageV2(ctx)`로 얻은 service를 기준으로 적었습니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `backups` | `Backups.ListDetail(ctx, WithListDetailQuery...)`, `details=False`는 `Backups.List` | go_mapping |
| `get_backup` | `Backups.Get(ctx, id)` | go_mapping |
| `find_backup` | `Backups.Find(ctx, resource.ID(x))`와 `resource.Name(x)`를 따로 호출 | unresolved |
| `create_backup` | `Backups.Create(ctx, opts, WithCreateField)` | go_mapping |
| `delete_backup` | `Backups.Remove(ctx, resource.ID(id))`, `force=True`는 `Backups.ForceDelete` | go_mapping |
| `import_backup` | `Backups.Import(ctx, ImportOpts{BackupService, BackupURL})` | go_mapping |
| `export_backup` | `Backups.Export(ctx, id)` | go_mapping |
| `export_record` | `Backups.Export(ctx, id)` | go_mapping |
| `restore_backup` | `Backups.RestoreFromBackup(ctx, id, RestoreOpts{VolumeID, Name})` | go_mapping |
| `reset_backup_status` | `Backups.ResetStatus(ctx, id, ResetStatusOpts{Status})` | go_mapping |
| `reset_backup` | `Backups.ResetStatus(ctx, id, ResetStatusOpts{Status})` | go_mapping |
| `create_transfer` | `Transfers.Create(ctx, CreateOpts{VolumeID, Name})` | go_mapping |
| `get_transfer` | `Transfers.Get(ctx, id)` | go_mapping |
| `transfers` | `Transfers.List`는 `os-volume-transfer/detail`을 보냄 | unresolved |
| `find_transfer` | `Transfers.Find`는 목록 단계에서 상세 경로를 보냄 | unresolved |
| `accept_transfer` | `Transfers.Accept(ctx, id, AcceptOpts{AuthKey})` | go_mapping |
| `delete_transfer` | `Transfers.Remove(ctx, resource.ID(id))` | go_mapping |
| `get_limits` | `Limits.Get(ctx)` | go_mapping |
| `get_quota_set` | `QuotaSets.Get`, `QuotaSets.GetUsage`에 query 옵션 없음 | unresolved |
| `get_quota_set_defaults` | `QuotaSets.Get(ctx, "defaults")` | go_mapping |
| `update_quota_set` | `QuotaSets.Update(ctx, projectID, opts, WithUpdateField)` | go_mapping |
| `revert_quota_set` | `QuotaSets.Delete(ctx, projectID)`에 query 옵션 없음 | unresolved |
| `get_quota_class_set` | 없음 | unresolved |
| `update_quota_class_set` | 없음 | unresolved |
| `backend_pools` | `SchedulerStats.List(ctx, WithListQuery("detail", "True"))` | go_mapping |
| `get_capabilities` | 없음 | unresolved |
| `services` | `Services.List(ctx, WithListOptions(ListOpts{Binary, Host}))` | go_mapping |
| `find_service` | 없음 | unresolved |
| `enable_service` | 없음 | unresolved |
| `disable_service` | 없음 | unresolved |
| `thaw_service` | 없음 | unresolved |
| `freeze_service` | 없음 | unresolved |
| `failover_service` | 없음 | unresolved |
| `extensions` | 없음 | unresolved |
| `get_type` | 없음 | unresolved |
| `find_type` | 없음 | unresolved |
| `types` | 없음 | unresolved |
| `create_type` | 없음 | unresolved |
| `update_type` | 없음 | unresolved |
| `delete_type` | 없음 | unresolved |
| `update_type_extra_specs` | 없음 | unresolved |
| `delete_type_extra_specs` | 없음 | unresolved |
| `get_type_access` | 없음 | unresolved |
| `add_type_access` | 없음 | unresolved |
| `remove_type_access` | 없음 | unresolved |
| `get_type_encryption` | 없음 | unresolved |
| `create_type_encryption` | 없음 | unresolved |
| `update_type_encryption` | 없음 | unresolved |
| `delete_type_encryption` | 없음 | unresolved |
| `create_qos_spec` | 없음 | unresolved |
| `get_qos_spec` | 없음 | unresolved |
| `find_qos_spec` | 없음 | unresolved |
| `qos_specs` | 없음 | unresolved |
| `update_qos_spec` | 없음 | unresolved |
| `delete_qos_spec` | 없음 | unresolved |
| `associate_qos_spec` | 없음 | unresolved |
| `disassociate_qos_spec` | 없음 | unresolved |
| `disassociate_all_qos_spec` | 없음 | unresolved |
| `qos_spec_associations` | 없음 | unresolved |
| `get_consistency_group` | 없음 | unresolved |
| `find_consistency_group` | 없음 | unresolved |
| `consistency_groups` | 없음 | unresolved |
| `create_consistency_group` | 없음 | unresolved |
| `create_consistency_group_from_source` | 없음 | unresolved |
| `update_consistency_group` | 없음 | unresolved |
| `delete_consistency_group` | 없음 | unresolved |

## 공통 차이

Cinder v2 API에는 microversion이 없습니다. Python v2 Resource에도 `_max_microversion`이 없으므로 session 기본값을 따로 정하지 않으면 version header를 보내지 않습니다. Go v2 service의 client도 microversion을 설정하지 않으면 header를 보내지 않으며, 테스트는 이 상태의 요청을 비교합니다.

Python은 Resource 객체나 ID를 받지만 Go는 ID 문자열을 받고, `Remove`와 `Find`는 `resource.Ref`를 받습니다. 결과는 Python Resource 대신 Gophercloud typed model이라서 모델에 없는 응답 field는 남지 않습니다. Python은 2xx 응답을 모두 성공으로 보지만 Gophercloud 호출은 작업마다 정한 status만 받습니다.

Python delete 계열의 기본값은 `ignore_missing=True`입니다. Go에서는 `Remove(ctx, resource.ID(id))`가 404를 무시하고, `resource.WithMissingError()`를 붙이면 `ignore_missing=False`처럼 404를 돌려줍니다. 409 같은 다른 실패는 두 경우 모두 그대로 돌려줍니다.

## Backup

[backup 메서드](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1026-L1226)의 `backups()` 기본값 `details=True`는 `backups/detail`을 쓰므로 `ListDetail`에 대응하고, `details=False`는 `List`에 대응합니다. `ListDetailOpts`에는 `name`, `status`, `volume_id`, `project_id`, `sort_key`, `sort_dir` 필드가 없어서 `WithListDetailQuery`로 더합니다. Python의 `all_tenants=True`는 문자열 `True`가 되므로 같은 값을 보내려면 typed `AllTenants` 대신 `WithListDetailQuery("all_tenants", "True")`를 씁니다. 두 쪽 모두 `backups_links`를 따라 다음 페이지를 읽습니다.

`create_backup`은 Python이 `is_incremental`을 `incremental`로 바꿔 보내므로 Go `CreateOpts.Incremental`과 같은 본문이 됩니다. Go는 false 값을 생략하고, v2 Python Backup에 선언되지 않은 `metadata` 같은 속성은 Python이 버리지만 Go는 그대로 보냅니다. `delete_backup(force=True)`는 Python이 `{"os-force_delete": null}`을, Go `ForceDelete`가 `{"os-force_delete": {}}`를 보냅니다.

고정 Source의 `import_record`는 인자 `url`을 `backups/export_record` 경로 문자열로 덮어쓴 뒤 그 경로에 POST합니다. 그래서 Python v2 `import_backup`은 caller의 locator를 실제로 보내지 않습니다. Go `Import`는 v3 판정과 같이 이 결함을 고친 `POST backups/import_record` 요청을 보내며 `ImportOpts.BackupURL`은 bytes라서 standard base64로 다시 인코딩됩니다. export 결과의 `BackupURL`을 그대로 넘기면 Cinder가 돌려준 locator와 같은 값이 됩니다. 결과는 Python Backup 대신 ID와 Name만 담은 `ImportResponse`입니다.

`export_backup`과 deprecated 별칭 `export_record`는 같은 GET을 보냅니다. Python은 응답 JSON 전체 dict를 돌려주고, Go는 `backup-record`를 `BackupRecord`로 decode하면서 `backup_url`의 base64를 bytes로 풉니다. `restore_backup`은 Python이 `volume`과 `name`이 모두 없으면 요청 전에 예외를 내지만 Go는 빈 `restore` 객체를 그대로 보냅니다. 결과는 Python이 restore 응답으로 갱신한 Backup이고 Go는 `BackupID`, `VolumeID`, `VolumeName`을 가진 `Restore`입니다. `reset_backup_status`와 별칭 `reset_backup`은 같은 `os-reset_status` action을 보냅니다.

## Transfer

[transfer 메서드](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1902-L2029)의 생성, 조회, 수락, 삭제는 v2 Transfer의 `os-volume-transfer` 경로를 쓰므로 Go v2 `Transfers`와 같은 요청입니다. Python 생성 결과의 `auth_key`는 Go `Transfer.AuthKey`에 있습니다. 수락 결과는 Python이 새 Transfer로, Go가 `Transfer` 모델로 돌려줍니다.

## Limit와 quota

[`get_limits`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1230-L1248)는 project를 받아도 v2 Limits에 project URI나 query가 없어서 항상 `GET limits`를 보냅니다. Go `Limits.Get(ctx)`가 같은 요청이며 absolute와 rate 값을 typed 필드로 돌려줍니다.

[quota 메서드](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1516-L1622) 가운데 `get_quota_set_defaults`는 base path `/os-quota-sets/defaults`로 fetch하므로 project ID가 URL에 들어가지 않습니다. Go에서 같은 요청은 `QuotaSets.Get(ctx, "defaults")`이고, 이름이 비슷한 `GetDefaults(ctx, projectID)`는 `os-quota-sets/{project}/defaults`라는 다른 요청입니다. volume type별 quota key는 Go `QuotaSet.Extra`에 남습니다.

`update_quota_set(project, **attrs)`는 바뀐 속성만 `quota_set` envelope에 담아 PUT합니다. Go `UpdateOpts`의 pointer 필드와 `WithUpdateField`로 같은 본문을 만들 수 있습니다. Python은 바뀐 속성이 없으면 요청을 보내지 않지만 Go는 빈 envelope도 PUT하고, 응답 status는 200만 받습니다. deprecated signature에서 QuotaSet 객체를 넘기는 경우에도 Go는 그 project ID로 같은 PUT을 보내면 됩니다.

## Backend pool과 service

[`backend_pools`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1011-L1022)는 base path에 `?detail=True`가 들어 있습니다. 같은 문자열은 `WithListQuery("detail", "True")`가 보내고 typed `Detail: true`는 소문자 `true`를 보냅니다. Python은 `capabilities` dict 전체를 보존하지만 Go `Capabilities`는 정해진 필드만 decode하고 `infinite` 용량을 무한대로, 숫자가 아닌 다른 값을 0으로 바꿉니다.

[`services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1671-L1681)는 `binary`와 `host`를 query로 보내며 Go `ListOpts`의 같은 필드에 대응합니다. Python은 query mapping에 없는 Body 속성을 응답에서 local로 거르고 Go는 caller가 거릅니다.

## 대응하지 못한 메서드

Backup, transfer, quota, service에서 남은 메서드는 다음과 같습니다.

- `find_backup`: Python은 GET이 400·403·404이면 `backups/detail?name=...` 목록에서 ID나 이름을 찾습니다. Go `Find(resource.ID)`는 403과 400을 오류로 돌려주고 `Find(resource.Name)`은 name query 없이 목록을 읽으며, v2에는 이 검색을 재현하는 helper가 없습니다.
- `find_transfer`: Python의 목록 단계는 요약 경로 `os-volume-transfer`를 쓰지만 Go `Find(resource.Name)`은 `os-volume-transfer/detail`을 읽습니다.
- `transfers`: 고정 v2 Source는 `/volume-transfers/detail`(details=False이면 `/volume-transfers`)을 요청합니다. Go v2 `List`는 `os-volume-transfer/detail`만 보냅니다.
- `get_quota_set`: Python은 `usage=False` 또는 `usage=True`와 추가 `**query`를 query로 보냅니다. Go `Get`은 query를 보내지 않고 `GetUsage`는 소문자 `usage=true`만 보내며 추가 query 옵션이 없습니다. `QuotaUsageSet`에는 volume type별 usage key도 남지 않습니다.
- `revert_quota_set`: Python은 `**query`를 DELETE query로 보내지만 Go `Delete`에는 query 옵션이 없고 200 외의 2xx를 오류로 봅니다.
- `get_quota_class_set`, `update_quota_class_set`: Gophercloud v2.15.0의 blockstorage v2에는 `os-quota-class-sets` binding이 없습니다.
- `get_capabilities`: Gophercloud v2.15.0의 blockstorage v2에는 `GET capabilities/{host}` binding이 없습니다.
- `find_service`: Python은 GET 없이 `os-services` 목록에서 ID나 binary 이름과 host를 맞춰 하나를 고르는데, v2 `Services`에는 이 검색 helper가 없습니다.
- `enable_service`, `disable_service`, `thaw_service`, `freeze_service`, `failover_service`: Gophercloud v2.15.0에는 `PUT os-services/{action}` binding이 없습니다.

`extensions`와 volume type, QoS spec, consistency group 메서드 33개는 `blockstorage/v2`에 대응 package가 없습니다. Gophercloud v2.15.0의 blockstorage v2에는 `extensions`, `types`, `qos-specs`, `consistencygroups` binding이 없으므로 v2 SDK 소유 호출이 필요합니다. 두 v2 Source의 동작도 v3와 다른 부분이 있습니다. `find_type`은 GET과 목록 모두 query 없이 보내고, `update_type`은 v2 Type에 `allow_commit`이 없어서 HTTP 없이 `MethodNotSupported`를 냅니다. QoS spec과 consistency group의 Resource 파일과 proxy 본문은 v3와 같으므로 v2 package를 추가할 때 [v3 비교](../v3/python-parity.md)와 [consistency group 비교](../v3/consistency-groups-python.md)를 기준으로 삼을 수 있습니다.
