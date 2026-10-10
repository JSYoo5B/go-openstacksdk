# Nova v2 Python proxy 대응: 관리자 호출

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [compute v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py) 가운데 host aggregate, flavor 관리, extension, hypervisor, compute service, migration, 서버 관리자 action, 진단, 사용량, quota class, share attachment, server external event 메서드 52개를 Go 호출과 비교합니다. Python 요청은 [Proxy 공통 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)와 [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py), 그리고 각 resource 파일([Aggregate](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/aggregate.py), [Flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py), [Extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/extension.py), [Hypervisor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/hypervisor.py), [Service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/service.py), [Migration](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/migration.py), [ServerMigration](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_migration.py), [Server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py), [ServerDiagnostics](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_diagnostics.py), [Usage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/usage.py))에서 따라갔습니다. 각 package의 `python_parity_test.go`와 `servers/python_parity_admin_test.go`가 Go 쪽 요청을 고정합니다. 서버 사용자 호출은 [Nova v2 Python proxy 대응: 서버 사용자 호출](python-parity.md)에 따로 정리했습니다.

`go_mapping`은 모든 Python 인자의 효과를 공개 Go API로 재현할 수 있다는 뜻이고, 이때도 아래 절의 차이는 남습니다. `unresolved`는 Go에 경로, 본문 값이나 helper 동작이 없어서 같은 요청이나 같은 기본 동작을 만들 수 없다는 뜻입니다. 표의 Go 호출은 `conn.ComputeV2(ctx)`가 돌려준 service의 필드를 기준으로 적었습니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `abort_server_migration` | 없음 | unresolved |
| `add_host_to_aggregate` | `Aggregates.AddHost(ctx, id, AddHostOpts{Host})` | go_mapping |
| `aggregate_precache_images` | 없음 | unresolved |
| `aggregates` | `Aggregates.All(ctx, resource.WithName(...))`, `Aggregates.List(ctx)` | go_mapping |
| `create_aggregate` | `Aggregates.Create(ctx, CreateOpts{Name, AvailabilityZone})` | go_mapping |
| `create_flavor` | `Flavors.Create(ctx, opts, WithCreateField)` | go_mapping |
| `create_flavor_extra_specs` | `Flavors.CreateExtraSpecs(ctx, id, ExtraSpecsOpts)` | go_mapping |
| `create_server_external_events` | 없음 | unresolved |
| `create_share_attachment` | 없음 | unresolved |
| `delete_aggregate` | `Aggregates.Remove(ctx, resource.ID(id))` | go_mapping |
| `delete_flavor` | `Flavors.Remove(ctx, resource.ID(id))` | go_mapping |
| `delete_flavor_extra_specs_property` | `Flavors.DeleteExtraSpec(ctx, id, key)` (404를 무시하는 기본값 없음) | unresolved |
| `delete_service` | `Services.Delete(ctx, id)` (404를 무시하는 기본값 없음) | unresolved |
| `delete_share_attachment` | 없음 | unresolved |
| `disable_service` | `Services.Update(ctx, id, UpdateOpts{Status, DisabledReason})` (2.53 이상만) | unresolved |
| `enable_service` | `Services.Update(ctx, id, UpdateOpts{Status})` (2.53 이상만) | unresolved |
| `evacuate_server` | `Servers.Evacuate(ctx, id, opts)` (2.14 미만에서만 성공) | unresolved |
| `extensions` | `Extensions.List(ctx)` | go_mapping |
| `find_aggregate` | `Aggregates.Find(ctx, resource.ID(x))`와 `resource.Name(x)`를 따로 호출 | unresolved |
| `find_extension` | `Extensions.Get(ctx, alias)` (목록 fallback 없음) | unresolved |
| `find_hypervisor` | `Hypervisors.Find(ctx, resource.ID(x))` (이름 조회 없음) | unresolved |
| `find_service` | `Services.List(ctx, ...)`를 직접 걸러야 함 | unresolved |
| `flavor_add_tenant_access` | `Flavors.AddAccess(ctx, id, AddAccessOpts{Tenant})` | go_mapping |
| `flavor_remove_tenant_access` | `Flavors.RemoveAccess(ctx, id, RemoveAccessOpts{Tenant})` | go_mapping |
| `force_complete_server_migration` | 없음 | unresolved |
| `get_aggregate` | `Aggregates.Get(ctx, id)` | go_mapping |
| `get_flavor_access` | `Flavors.ListAccesses(ctx, id)` | go_mapping |
| `get_hypervisor` | `Hypervisors.Get(ctx, id)` | go_mapping |
| `get_hypervisor_uptime` | `Hypervisors.GetUptime(ctx, id)` | go_mapping |
| `get_quota_class_set` | 없음 | unresolved |
| `get_server_diagnostics` | `Diagnostics.Get(ctx, serverID)` | go_mapping |
| `get_server_migration` | 없음 | unresolved |
| `get_share_attachment` | 없음 | unresolved |
| `get_usage` | `Usage.SingleTenant(ctx, projectID, SingleTenantOpts{Start, End})`의 첫 값 | go_mapping |
| `hypervisors` | `Hypervisors.List(ctx, WithListOptions)` (상세 경로만) | unresolved |
| `live_migrate_server` | `Servers.LiveMigrate(ctx, id, opts, WithLiveMigrateField)` (`"auto"` 불가) | unresolved |
| `migrate_server` | `Servers.Migrate(ctx, id)` (대상 host 불가) | unresolved |
| `migrations` | `Migrations.List(ctx, WithListOptions, WithListQuery)` | go_mapping |
| `remove_host_from_aggregate` | `Aggregates.RemoveHost(ctx, id, RemoveHostOpts{Host})` | go_mapping |
| `reset_server_state` | `Servers.ResetState(ctx, id, servers.ServerState(state))` | go_mapping |
| `server_migrations` | 없음 | unresolved |
| `services` | `Services.List(ctx, WithListOptions(ListOpts{Binary, Host}), WithListQuery)` | go_mapping |
| `set_aggregate_metadata` | `Aggregates.SetMetadata(ctx, id, SetMetadataOpts{Metadata})` | go_mapping |
| `share_attachments` | 없음 | unresolved |
| `trigger_server_crash_dump` | 없음 | unresolved |
| `update_aggregate` | `Aggregates.Update(ctx, id, UpdateOpts)` (`availability_zone` null 불가) | unresolved |
| `update_flavor` | `Flavors.Update(ctx, id, UpdateOpts{Description})` (설명 지우기 불가) | unresolved |
| `update_flavor_extra_specs_property` | `Flavors.UpdateExtraSpec(ctx, id, ExtraSpecsOpts{key: value})` | go_mapping |
| `update_quota_class_set` | 없음 | unresolved |
| `update_service` | `Services.Update(ctx, id, UpdateOpts, WithUpdateField)` (`forced_down` false 불가) | unresolved |
| `update_service_forced_down` | `Services.Update(ctx, id, UpdateOpts{ForcedDown: true})` (true만) | unresolved |
| `usages` | `Usage.AllTenants(ctx, AllTenantsOpts{Start, End, Detailed})` | go_mapping |

## 공통 차이

Python은 Resource 객체나 ID 문자열을 받지만 Go generated 호출은 ID 값을 직접 받고, `Remove`와 `Find`는 `resource.Ref`를 받습니다. aggregate ID는 Go에서 `int`라서 `Get`, `Update`, action 호출에는 정수를 넘기고, `Remove(ctx, resource.ID("7"))`와 `Find`는 문자열을 정수로 바꾼 뒤 요청합니다. 결과는 Python Resource 대신 Gophercloud typed model이라서 모델에 없는 응답 field는 남지 않습니다. 예를 들어 `flavors.Flavor`에는 `OS-FLV-DISABLED:disabled`가 없고, `hypervisors.Hypervisor`에는 2.88부터 상세 응답에 오는 `uptime`이 없습니다.

Python은 resource의 `_max_microversion`(Aggregate 2.81, Flavor 2.61, Hypervisor 2.88, Service 2.69, Migration 2.80, Usage 2.75, ServerDiagnostics 2.48, Server 2.100)과 서버 최대값 가운데 작은 값을 요청마다 고릅니다. aggregate action, flavor 접근 권한 action과 `get_flavor_access`, Extension 요청은 microversion을 지정하지 않습니다. Go는 Connection에 설정하거나 협상한 client microversion 하나를 모든 호출에 보내므로, 같은 header가 필요하면 `RawClient()`의 복사본에 `Microversion`을 바꿔 새 API를 만들어야 합니다.

Python delete 계열은 `ignore_missing=True`가 기본이라 404를 무시합니다. Go에서 `Remove(ctx, resource.ID(id))`가 같은 기본값을 갖고 `resource.WithMissingError()`가 `ignore_missing=False`에 해당합니다. generated `Delete`를 직접 부르면 404가 오류로 돌아옵니다. `Remove`가 없는 `Flavors.DeleteExtraSpec`과 `Services.Delete`는 Python 기본값을 고를 수 없어서 unresolved로 두었습니다.

Python `find_*`는 ID로 GET을 먼저 보내고 404, 400, 403이면 목록에서 ID나 이름이 같은 항목을 하나 고릅니다. 이 단원의 Go package에는 이 두 단계를 묶은 helper가 없으므로 find 메서드는 모두 unresolved입니다. Python update는 바뀐 attribute가 없으면 요청을 보내지 않지만 Go `Update`는 빈 envelope라도 PUT을 보냅니다.

## Host aggregate

`create_aggregate`, `get_aggregate`, `add_host_to_aggregate`, `remove_host_from_aggregate`, `set_aggregate_metadata`는 같은 경로와 본문을 보내며 두 쪽 모두 응답의 `aggregate`를 돌려줍니다. `set_aggregate_metadata`에서 값이 None인 key는 Go map의 nil 값이 되어 `null`로 나가고, 인자를 주지 않은 Python 호출의 `{"metadata": {}}`는 빈 map으로 만듭니다. Go `SetMetadata`는 nil map을 HTTP 전에 거부하므로 빈 map을 넘겨야 합니다. Go `AddHost`와 `RemoveHost`도 빈 host를 HTTP 전에 거부하지만, Python이 보내는 빈 문자열은 어차피 Nova가 거부합니다.

`aggregates()`는 `GET os-aggregates`이며, Python이 이름 같은 Body attribute를 query로 받으면 응답을 받은 뒤 걸러 냅니다. Go에서는 `All(ctx, resource.WithName("rack2"))`가 같은 지역 필터입니다. Python은 `limit`과 `marker`를 query로 보낼 수 있지만 Nova의 aggregate 목록은 이를 쓰지 않으며, Go 목록에는 query를 넣을 수 없습니다.

`delete_aggregate`는 `Remove`가 404를 무시하며 같은 DELETE를 보냅니다. Go generated `Delete`는 200만 성공으로 보지만 Nova도 200을 돌려줍니다.

`update_aggregate`는 넘긴 `name`과 `availability_zone`만 `{"aggregate": {...}}`에 담습니다. Go `UpdateOpts`는 빈 문자열을 생략하고 확장 필드로 typed key를 덮을 수 없어서, Nova가 aggregate의 가용 영역을 지우는 데 쓰는 `availability_zone=None`의 null을 보낼 수 없습니다. 그래서 unresolved입니다. `create_aggregate`의 null 가용 영역은 생략과 결과가 같아서 차이로만 남깁니다.

`find_aggregate`는 `GET os-aggregates/{name_or_id}`를 먼저 보냅니다. Go `Find(ctx, resource.ID(x))`는 숫자가 아닌 값을 HTTP 전에 `resource.ErrInvalidOption`으로 거부하고, `Find(ctx, resource.Name(x))`는 목록에서 이름만 맞춰 봅니다. `aggregate_precache_images`의 `POST os-aggregates/{id}/images`(2.81)는 Go 호출이 없습니다.

## Flavor

`create_flavor`는 넘긴 attribute만 `{"flavor": {...}}`에 담고 Go `CreateOpts`는 `name`, `ram`, `vcpus`, `disk`를 항상 보냅니다. Nova가 이 네 key를 필수로 요구하므로 유효한 Python 호출에서는 본문이 같습니다. `is_public`과 `ephemeral`은 `IsPublic`, `Ephemeral` pointer가 같은 서버 key(`os-flavor-access:is_public`, `OS-FLV-EXT-DATA:ephemeral`)로 보내고, `CreateOpts` 밖의 key는 `WithCreateField`로 넣습니다. Go는 빈 `description`과 0인 `rxtx_factor`를 생략합니다.

`update_flavor`는 `PUT flavors/{id}`에 `{"flavor": {"description": ...}}`를 보냅니다. Nova는 description의 null과 빈 문자열을 받아 설명을 지우지만 Go `UpdateOpts.Description`은 빈 값을 생략하고 `WithUpdateField("description", nil)`도 HTTP 전에 거부하므로 unresolved입니다. `delete_flavor`는 `Remove`가 404를 무시하며 Nova의 202를 성공으로 받습니다.

`flavor_add_tenant_access`와 `flavor_remove_tenant_access`는 같은 `POST flavors/{id}/action` 본문을 보냅니다. Python은 `Accept` header를 비워 보내고 None을 돌려주지만 Go는 응답의 `flavor_access` 목록을 돌려줍니다. `get_flavor_access`는 같은 GET을 보내며 Python의 dict 목록 대신 `FlavorAccess` 행을 하나씩 내보냅니다.

`create_flavor_extra_specs`는 `POST flavors/{id}/os-extra_specs`로 병합하고, Python은 응답 spec을 담은 Flavor를, Go는 응답 map을 돌려줍니다. Go `ExtraSpecsOpts`는 문자열 값만 받으므로 숫자 값은 문자열로 바꿔 넘깁니다. Nova도 숫자 값을 문자열로 저장합니다. `update_flavor_extra_specs_property`는 `PUT flavors/{id}/os-extra_specs/{key}`에 `{key: value}`를 보내며, Python은 응답의 그 값을, Go는 응답 map을 돌려줍니다. `delete_flavor_extra_specs_property`는 Python이 404를 무시하지만 Go `DeleteExtraSpec`은 옵션 없이 404를 오류로 돌려줍니다.

## Extension

`extensions()`는 `GET extensions`이고 Go `Extensions.List`가 alias, 이름, namespace, `updated`, 설명, link를 그대로 읽습니다. `find_extension`은 `GET extensions/{alias}` 뒤 404이면 목록에서 alias나 이름을 찾습니다. Go `Extensions.Get`은 첫 GET만 보내고 404를 그대로 돌려주며 목록 fallback helper가 없습니다.

## Hypervisor

`get_hypervisor`는 `GET os-hypervisors/{id}`이며 Go `Get`이 같은 요청을 보냅니다. `get_hypervisor_uptime`은 Python이 서버가 2.88을 지원하면 요청 없이 `SDKException`을 일으키고, 그보다 낮으면 `GET os-hypervisors/{id}/uptime`을 보냅니다. Go `GetUptime`은 client microversion과 관계없이 요청을 보내므로 2.88 이상이면 Nova의 404가 오류로 돌아옵니다. Python은 uptime을 Hypervisor에 채워 돌려주고 Go는 `Uptime` 값을 돌려줍니다.

`hypervisors()`는 기본값 `details=False`에서 `GET os-hypervisors` 요약 목록을 읽습니다. Go `List`는 항상 `os-hypervisors/detail`을 읽어서 같은 요청을 보낼 수 없습니다. 2.53 이상에서 `hypervisor_hostname_pattern`과 `with_servers`는 Go `ListOpts`의 pointer 필드가 같은 query로 보냅니다. 다만 Python의 `with_servers=True`는 `True`로, Go는 `true`로 나갑니다. 2.53 미만에서 Python은 pattern을 `os-hypervisors/{pattern}/search` 경로로 보내는데 Go에는 이 경로가 없습니다.

`find_hypervisor`는 ID GET이 404이면 기본값 `details=True`에 따라 `os-hypervisors/detail`에서 ID나 `hypervisor_hostname`을 찾습니다. Go `Find(ctx, resource.ID(x))`는 그 GET에서 멈추고, hypervisor collection에는 이름 binding이 없어서 `Find(ctx, resource.Name(x))`는 `resource.ErrUnsupported`입니다.

## Compute service

`services()`는 `GET os-services`이며 Python의 `name`은 `binary` query로 바뀝니다. Go에서는 `ListOpts{Binary, Host}`가 같은 query를 보내고 `limit`, `marker` 같은 다른 key는 `WithListQuery`로 넣습니다. Python은 `status`처럼 query가 아닌 Body attribute를 응답에서 거르지만 Go `Services.List`는 지역 필터가 없어서 호출자가 걸러야 합니다.

`enable_service`, `disable_service`, `update_service_forced_down`, `update_service`는 2.53 이상에서 `PUT os-services/{id}`에 envelope 없는 본문을 보내며 Go `Update`가 같은 `status`, `disabled_reason`, `forced_down`을 만듭니다. 그러나 `UpdateOpts`의 세 필드는 모두 빈 값을 생략하고 확장 필드로도 덮을 수 없어서, `forced=False`나 `forced_down=False`로 강제 down을 푸는 요청을 만들 수 없습니다. `update_service`는 Python이 2.53 미만이면 요청 없이 거부합니다.

2.53 미만에서 Python은 `PUT os-services/enable`, `disable`, `disable-log-reason`, `force-down`에 `host`와 `binary`를 담아 보냅니다. Go도 `Update(ctx, "disable", UpdateOpts{}, WithUpdateField("host", ...), WithUpdateField("binary", ...))`로 같은 요청을 만들 수 있지만, Nova가 돌려주는 service에 `id`가 없어서 Gophercloud decoder가 오류를 냅니다. 서비스 인자 없이 host와 binary만 넘긴 2.53 이상 호출은 Python이 `services(host, binary)`로 하나를 고른 뒤 수정하므로, Go에서는 `List` 결과의 ID로 `Update`를 이어 부릅니다. 이런 이유로 이 네 메서드는 unresolved입니다.

`find_service`는 ID GET 없이 `os-services` 전체 목록을 query 없이 읽고, ID나 이름(없으면 binary)이 같은 항목 가운데 `host` 인자와 맞는 하나를 고릅니다. Go에는 이 helper가 없습니다. `delete_service`는 같은 DELETE를 보내지만 Go `Delete`는 404를 무시하는 옵션이 없고 204만 성공으로 봅니다.

## Migration

`migrations()`는 `GET os-migrations`이고, Python의 `host`, `status`, `migration_type`, `source_compute`, `user_id`, `project_id`, `changes_since`, `changes_before`, `server_id`, `limit`, `marker`는 Go `ListOpts`의 pointer 필드가 같은 query key(`changes-since`, `changes-before`, `instance_uuid` 포함)로 보냅니다. Python은 시각 문자열을 그대로 보내고 Go는 `time.Time`을 RFC3339로 바꿉니다. Python은 `migrations_links`의 다음 페이지를 따라가지만 Go는 첫 페이지만 읽으므로 `Marker`로 직접 이어 요청해야 합니다. Python은 query가 아닌 Body attribute를 응답에서 거르고, Go `WithListQuery`는 그런 key도 서버로 보냅니다.

`server_migrations`, `get_server_migration`, `abort_server_migration`, `force_complete_server_migration`이 쓰는 `servers/{id}/migrations` 경로와 `force_complete` action은 Go 호출이 없습니다.

## 서버 관리자 action

`reset_server_state`는 `{"os-resetState": {"state": ...}}`를 보내며 Go `ResetState`에 문자열을 `servers.ServerState`로 바꿔 넘깁니다.

`evacuate_server`는 넘긴 인자만 `evacuate` 객체에 담고 인자가 없으면 `{"evacuate": {}}`를 보냅니다. Go `EvacuateOpts.OnSharedStorage`는 omitempty가 없어서 항상 `onSharedStorage`를 보내고, Nova는 2.14부터 이 key를 거부합니다. 그래서 2.14 이상에서 Go로는 같은 요청을 만들 수 없습니다.

`migrate_server`는 host가 없으면 `{"migrate": null}`이고 Go `Migrate`도 같습니다. 하지만 2.56의 `{"migrate": {"host": ...}}`는 `Migrate`가 옵션을 받지 않아 보낼 수 없습니다.

`live_migrate_server`는 2.30 이상에서 `{"os-migrateLive": {"host": null, "block_migration": "auto"}}`를 기본으로 보냅니다. Go `LiveMigrateOpts.BlockMigration`은 `*bool`이고 `block_migration`은 typed key라서 `"auto"`를 보낼 수 없습니다. host와 `force=True`, true나 false인 `block_migration`을 함께 주는 호출은 `LiveMigrateOpts{Host, BlockMigration}`과 `WithLiveMigrateField("force", true)`로 같은 본문을 만듭니다.

`trigger_server_crash_dump`의 `trigger_crash_dump` action(2.17)은 Go 호출이 없습니다.

## 진단과 사용량

`get_server_diagnostics`는 서버 ID만 꺼내 `GET servers/{id}/diagnostics`를 보내고, Go `Diagnostics.Get`이 같은 요청의 응답을 `map[string]any`로 돌려줍니다. Python은 2.48의 정규화된 key를 attribute로 노출하고, Go map에서는 숫자가 `float64`입니다.

`usages(start, end, **query)`는 `GET os-simple-tenant-usage`에 `start.isoformat()`, `end.isoformat()`와 넘긴 query를 붙이고 `tenant_usages_links`를 따라갑니다. Go `AllTenants`도 다음 링크를 따라가며, `Detailed: true`는 `detailed=1`이 되고 다른 key는 `WithAllTenantsQuery`로 넣습니다. Python에서 `detailed=True`를 넘기면 `True`로 나가서 Nova가 상세 응답으로 보지 않으므로 Python에서도 `detailed=1`을 써야 같은 결과입니다.

`get_usage(project, start, end)`는 `GET os-simple-tenant-usage/{project}`를 한 번 보내고 첫 응답의 Usage를 돌려줍니다. Go `SingleTenant`는 `tenant_usage_links`를 따라가는 stream이므로 첫 값을 받은 뒤 순회를 멈추면 같은 요청 하나로 끝납니다. 시각은 Go가 offset을 버린 벽시계 시각으로 보내므로 UTC 값을 넘겨야 하고, Python은 timezone이 있는 datetime이면 `+00:00` 같은 offset을 붙여 보냅니다.

## unresolved 목록

- `find_aggregate`, `find_extension`, `find_hypervisor`, `find_service`: ID GET과 목록 fallback을 묶은 helper가 없습니다. aggregate는 숫자가 아닌 ID를 HTTP 전에 거부하고, hypervisor collection은 이름 조회를 지원하지 않으며, extension과 service에는 collection lookup이 없습니다.
- `hypervisors`: 기본값의 요약 목록 `GET os-hypervisors`와 2.53 미만의 `os-hypervisors/{pattern}/search` 경로를 보낼 수 없습니다.
- `update_aggregate`: `availability_zone`의 null을 보낼 수 없어 가용 영역을 지울 수 없습니다.
- `update_flavor`: description의 null이나 빈 문자열을 보낼 수 없어 설명을 지울 수 없습니다.
- `delete_flavor_extra_specs_property`, `delete_service`: Python의 `ignore_missing=True` 기본값에 해당하는 옵션이나 `Remove`가 없습니다.
- `enable_service`, `disable_service`: 2.53 미만 action 응답을 Go `Update`가 decode하지 못합니다.
- `update_service`, `update_service_forced_down`: `forced_down` false를 보낼 수 없고, 2.53 미만의 `force-down` action 응답도 decode하지 못합니다.
- `evacuate_server`: `onSharedStorage`를 빼고 보낼 수 없어서 2.14 이상에서 실패합니다.
- `migrate_server`: 2.56의 대상 `host`를 보낼 수 없습니다.
- `live_migrate_server`: 2.25 이상의 기본값인 `block_migration: "auto"`를 보낼 수 없습니다.
- `server_migrations`, `get_server_migration`, `abort_server_migration`, `force_complete_server_migration`: `servers/{id}/migrations` 경로와 `force_complete` action을 호출하는 Go API가 없습니다.
- `aggregate_precache_images`: `POST os-aggregates/{id}/images`를 보내는 Go API가 없습니다.
- `trigger_server_crash_dump`: `trigger_crash_dump` action을 보내는 Go API가 없습니다.
- `create_server_external_events`: `POST os-server-external-events`를 보내는 Go API가 없습니다.
- `get_quota_class_set`, `update_quota_class_set`: Nova `os-quota-class-sets` 경로를 호출하는 Go API가 없습니다.
- `create_share_attachment`, `get_share_attachment`, `share_attachments`, `delete_share_attachment`: Nova `servers/{id}/shares` 경로를 호출하는 Go API가 없습니다.
