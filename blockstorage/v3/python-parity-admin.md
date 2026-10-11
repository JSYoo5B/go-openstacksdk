# Cinder v3 Python 관리 proxy 대응

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [block_storage v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py) 가운데 service, scheduler pool, extension, 기존 볼륨·스냅샷 가져오기, cluster, group type, default type, quota class, group 상태 초기화 등 관리 메서드 46개를 Go 호출과 비교합니다. volume·snapshot·backup 같은 핵심 resource는 [Cinder v3 Python proxy 대응](python-parity.md)에서 다룹니다. 각 Go 요청의 비교 기준은 아래 절에 적은 `python_parity_test.go` 테스트이며, native wire 계약은 [Cinder native 관리자 호출](native-admin.md)에 있습니다.

`go_mapping`은 Python 호출의 HTTP 요청과 인자를 공개 Go API로 재현할 수 있다는 뜻입니다. 이때도 아래 절의 차이는 남습니다. `unresolved`는 Go에 해당 경로나 helper가 없어서 같은 요청을 만들 수 없다는 뜻입니다. 표의 Go 호출은 `conn.BlockStorageV3(ctx)`가 돌려준 service의 필드를 기준으로 적었습니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `services` | `Services.List(ctx, WithListOptions(ListOpts{Binary, Host}), WithListQuery(...))` | go_mapping |
| `find_service` | `Services.List` 결과를 caller가 비교 | unresolved |
| `enable_service` | 없음 | unresolved |
| `disable_service` | 없음 | unresolved |
| `thaw_service` | 없음 | unresolved |
| `freeze_service` | 없음 | unresolved |
| `failover_service` | 없음 | unresolved |
| `set_service_log_levels` | 없음 | unresolved |
| `get_service_log_levels` | 없음 | unresolved |
| `cleanup_service_workers` | 없음 | unresolved |
| `backend_pools` | `SchedulerStats.List(ctx, WithListQuery("detail", "True"), ...)` | go_mapping |
| `get_capabilities` | 없음 | unresolved |
| `extensions` | `extensions.New(svc.RawClient()).List(ctx)` (`common/extensions`) | go_mapping |
| `manage_volume` | `ManageableVolumes.ManageExisting(ctx, opts)` | go_mapping |
| `manageable_volumes` | 없음 | unresolved |
| `manage_snapshot` | 없음 | unresolved |
| `unmanage_snapshot` | 없음 | unresolved |
| `manageable_snapshots` | 없음 | unresolved |
| `summary` | 없음 | unresolved |
| `get_cluster` | 없음 | unresolved |
| `find_cluster` | 없음 | unresolved |
| `clusters` | 없음 | unresolved |
| `enable_cluster` | 없음 | unresolved |
| `disable_cluster` | 없음 | unresolved |
| `get_group_type` | 없음 | unresolved |
| `find_group_type` | 없음 | unresolved |
| `group_types` | 없음 | unresolved |
| `create_group_type` | 없음 | unresolved |
| `update_group_type` | 없음 | unresolved |
| `delete_group_type` | 없음 | unresolved |
| `fetch_group_type_group_specs` | 없음 | unresolved |
| `create_group_type_group_specs` | 없음 | unresolved |
| `get_group_type_group_specs_property` | 없음 | unresolved |
| `update_group_type_group_specs_property` | 없음 | unresolved |
| `delete_group_type_group_specs_property` | 없음 | unresolved |
| `default_types` | 없음 | unresolved |
| `show_default_type` | 없음 | unresolved |
| `set_default_type` | 없음 | unresolved |
| `unset_default_type` | 없음 | unresolved |
| `get_quota_class_set` | 없음 | unresolved |
| `update_quota_class_set` | 없음 | unresolved |
| `resource_filters` | 없음 | unresolved |
| `reset_group_status` | 없음 | unresolved |
| `reset_group_state` | 없음 | unresolved |
| `reset_group_snapshot_status` | 없음 | unresolved |
| `reset_group_snapshot_state` | 없음 | unresolved |

## 공통 차이

Python 목록 호출은 Resource의 `QueryParameters`에 있는 key만 서버 query로 보내고, query에 없는 Body 속성은 받은 행을 로컬에서 거르며, 둘 다 아닌 key는 조용히 버립니다. 기본 `QueryParameters`에는 `limit`과 `marker`가 포함됩니다. Go generated 목록은 typed `ListOpts`와 `WithListQuery`로 준 값을 모두 서버로 보내므로, Python이 로컬에서 거르던 조건은 caller가 반복문에서 직접 비교합니다. Python은 `{resources_key}_links`의 next 링크를 따라가지만 이 문서의 Go 목록은 모두 한 페이지만 읽습니다. Cinder가 이 세 경로에 링크를 붙이지 않으므로 실제 결과는 같습니다.

Python은 Resource마다 `_max_microversion`과 서버 최대값 중 작은 값을 고르거나 고정 값을 보냅니다. Go generated 호출은 Connection에 설정한 client microversion을 그대로 씁니다. 같은 header가 필요하면 `RawClient()`의 복사본에 `Microversion`을 바꿔 새 API를 만듭니다. 반환 값은 Python Resource 대신 Gophercloud typed model이라서 모델에 없는 응답 field는 남지 않습니다.

## Service 목록

[`services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3470)는 `GET os-services`를 보내고 `binary`와 `host`를 서버 query로 씁니다. Go `Services.List`에 `ListOpts{Binary, Host}`를 주면 같은 query가 나가고, `limit`·`marker`는 `WithListQuery`로 더합니다. `zone`이나 `status` 같은 Body 속성 조건은 Python이 로컬에서 거르므로 Go에서는 결과 행의 `Zone`·`Status`를 직접 비교합니다.

[Service resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/service.py)는 3.32까지 협상하므로 Go client에 3.32를 설정하면 같은 header가 나갑니다. Go `Service`는 binary, host, zone, status, state, frozen, cluster, replication_status, active_backend_id, disabled_reason, updated_at을 읽습니다. 3.49에 추가된 `backend_state`와 Python의 `name` 별칭(binary와 같은 값)은 Go 모델에 없습니다. `updated_at`은 시간대 없는 형식만 받으므로 끝에 `Z`가 붙은 값은 decode 오류입니다.

`find_service`는 ID로 GET하지 않고 바로 목록을 읽습니다. 이때 `host` 같은 추가 query만 보내고, 받은 행에서 ID나 binary 이름이 같은 항목을 골라 둘 이상이면 중복 오류를, 없으면 기본값 `ignore_missing=True`에 따라 None을 돌려줍니다. Go `Services.List`는 같은 목록 요청을 보내지만 이 비교·중복 검사·missing 처리를 하는 helper가 없어서 unresolved로 둡니다.

이 대응은 `blockstorage/v3/services/python_parity_test.go`의 `TestPythonServicesListMatchesProxyQuery`, `TestPythonServicesListLocalFilterIsCallerSide`, `TestPythonServiceFindListPhase`가 확인합니다.

## Backend pool

[`backend_pools`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1616)는 [Pools](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/stats.py)의 base_path에 들어 있는 `?detail=True`를 항상 보냅니다. Go `SchedulerStats.List`는 아무 옵션이 없으면 `detail`을 생략해서 이름만 있는 pool 목록을 받으므로 기본값이 다릅니다. Python과 같은 문자열을 보내려면 `WithListQuery("detail", "True")`를 쓰고, typed `ListOpts{Detail: true}`는 소문자 `detail=true`를 보내며 Cinder는 두 값을 같은 bool로 읽습니다.

Python Pools의 서버 query는 `limit`과 `marker`뿐이라 `tenant_id`를 넘기면 버립니다. Go `ListOpts.TenantID`는 `tenant_id`를 실제로 보내므로 Python에는 없는 요청입니다. Python은 `capabilities` dict 전체를 보존하지만 Go `Capabilities`는 선언된 필드만 읽고 드라이버별 key를 버립니다. 용량 필드의 `infinite`·`unknown` 문자열 변환은 [native 문서](native-admin.md)에 설명한 대로입니다. Pools에는 `_max_microversion`이 없어 Python은 session 기본 microversion을 씁니다.

이 대응은 `blockstorage/v3/schedulerstats/python_parity_test.go`의 `TestPythonBackendPoolsDetailQuery`가 확인합니다.

## Extension

[`extensions`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3632)는 Cinder endpoint 아래 `GET extensions`를 보냅니다. Cinder service에는 extension API 필드가 따로 없으므로 `extensions.New(conn.BlockStorageV3(ctx).RawClient()).List(ctx)`처럼 `common/extensions` API를 같은 client로 만듭니다. 결과의 `Alias`, `Name`, `Description`, `Links`는 Python과 같고 Python `updated_at`은 Go `Updated` 문자열입니다. Go는 응답의 `namespace`도 읽습니다. Python [Extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/extension.py)에는 `_max_microversion`이 없어 session 기본 microversion을 씁니다.

이 대응은 `blockstorage/v3/python_parity_test.go`의 `TestPythonCinderExtensionsList`가 확인합니다.

## 기존 볼륨 가져오기

[`manage_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1282)은 [Volume.manage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py)로 `POST manageable_volumes`에 `{"volume": {...}}`를 보내고 202 응답의 volume을 돌려줍니다. Go `ManageableVolumes.ManageExisting`은 같은 경로와 envelope을 쓰며, 모든 인자를 채우면 본문 key와 값이 Python과 같습니다.

Python은 인자를 주지 않아도 `name`, `description`, `volume_type`, `availability_zone`, `metadata`를 null로, `bootable`을 false로 항상 보냅니다. Go는 빈 값을 생략하며 Cinder는 생략과 null·false를 같은 기본값으로 읽습니다. 이 key들은 typed 옵션에 선언되어 있어서 `WithManageExistingField`로 명시적 null을 보낼 수 없습니다. `Ref`는 `map[string]string`이라 문자열이 아닌 ref 값은 보낼 수 없습니다. Python은 header를 `Volume._max_microversion`인 3.71로 고정하고, 서버가 3.8을 지원하지 않으면 legacy `os-volume-manage` 경로로 바꿉니다. Go는 client microversion을 쓰고 legacy 경로를 고르지 않으므로 3.8 이상을 설정해야 합니다. 응답의 `created_at`은 시간대 없는 형식만 decode됩니다.

이 대응은 `blockstorage/v3/manageablevolumes/python_parity_test.go`의 `TestPythonManageVolumeRequest`가 확인합니다.

## unresolved 메서드

아래 메서드는 Gophercloud v2.15.0에 해당 경로의 binding이 없고 SDK가 소유한 호출도 아직 없습니다. 같은 요청을 보내려면 SDK 소유 API가 새로 필요합니다.

- Service 동작: `enable_service`·`disable_service`는 Service 객체의 `binary`·`host`를 `PUT os-services/enable`, `disable`, `disable-log-reason`에 보냅니다. `thaw_service`·`freeze_service`는 `host`를 `PUT os-services/thaw`, `freeze`에 보내고, `failover_service`는 3.26 이상에서 `failover`, 그 미만에서 `failover_host`를 씁니다. Go `Services`에는 `List`만 있습니다.
- 로그 수준과 worker 정리: `set_service_log_levels`·`get_service_log_levels`는 3.32 이상에서 `PUT os-services/set-log`, `get-log`에 `binary`(기본 빈 문자열)·`server`·`prefix`를 보냅니다. `cleanup_service_workers`는 3.24 이상에서 `PUT workers/cleanup`에 여덟 개 filter key를 보내며 `is_up`의 기본값은 false입니다.
- Service 검색: `find_service`는 위 Service 목록 절에서 설명한 목록 비교 helper가 없습니다.
- Capability: `get_capabilities`는 `GET capabilities/{host}`입니다.
- 가져오기 목록과 스냅샷: `manageable_volumes`·`manageable_snapshots`는 기본 `details=False`일 때 `GET manageable_volumes`, `manageable_snapshots`를, true일 때 `/detail` 경로를 쓰며 `host`, `cluster`, `marker`, `limit`, `offset`, `sort`를 query로 보냅니다. `manage_snapshot`은 `POST manageable_snapshots`에 `snapshot` envelope을 3.65로 보내고, `unmanage_snapshot`은 `POST snapshots/{id}/action`에 `{"os-unmanage": null}`을 보냅니다. Go `ManageableVolumes`에는 volume 가져오기만 있고 `Snapshots`에는 unmanage action이 없습니다.
- Volume 요약: `summary`는 3.12 이상에서 `GET volumes/summary?all_tenants=...`를 보내고 `volume-summary` envelope을 읽습니다. Go `Volumes.ListSummary`는 `GET volumes` 목록이라 다른 요청입니다.
- Cluster: `get_cluster`, `find_cluster`, `clusters`, `enable_cluster`, `disable_cluster`는 3.7 이상의 `clusters` 경로를 씁니다. `clusters`의 기본값은 `details=True`라 `clusters/detail`을 읽고, enable·disable은 `PUT clusters/enable`, `clusters/disable`에 `name`과 선택적 `binary`·`disabled_reason`을 보냅니다.
- Group type: `get_group_type`, `find_group_type`, `group_types`, `create_group_type`, `update_group_type`, `delete_group_type`는 3.11 이상의 `group_types` CRUD이며 `find_group_type`은 GET과 목록에 모두 `is_public=none`을 보냅니다. group spec 다섯 메서드는 `group_types/{id}/group_specs`와 그 아래 key 경로를 쓰고, `delete_group_type`과 `delete_group_type_group_specs_property`는 기본값 `ignore_missing=True`로 404를 무시합니다. SDK `groups` package에는 group record와 replication 호출만 있습니다.
- Default type: `default_types`, `show_default_type`, `set_default_type`, `unset_default_type`은 3.67 미만이면 요청 없이 SDKException을 내고, 그 이상에서 `default-types`와 `default-types/{project_id}`를 씁니다. 설정은 `PUT`에 `{"default_type": {"volume_type": ...}}`을 보내고 해제는 404를 무시하는 DELETE입니다.
- Quota class: `get_quota_class_set`은 기본 인자 `default`로 `GET os-quota-class-sets/default`를, `update_quota_class_set`은 바뀐 속성만 담은 `quota_class_set` envelope을 PUT합니다. SDK v3 `quotasets`는 project quota만 다룹니다.
- Resource filter: `resource_filters`는 3.33 이상에서 `GET resource_filters?resource=...`를 보냅니다.
- Group 상태 초기화: `reset_group_status`와 deprecated 별칭 `reset_group_state`는 `POST groups/{id}/action`, `reset_group_snapshot_status`와 deprecated 별칭 `reset_group_snapshot_state`는 `POST group_snapshots/{id}/action`에 `{"reset_status": {"status": ...}}`을 보냅니다. SDK `groups`·`groupsnapshots` package에는 이 action이 없습니다.

