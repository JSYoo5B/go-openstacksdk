# Nova cloud helper 대응: openstacksdk `ComputeCloudMixin`과 Go

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [cloud/_compute.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py)에 있는 `ComputeCloudMixin` helper 43개를 Go 호출과 비교합니다. cloud helper는 [compute v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py) 호출에 이름 조회, 결과 가공, 예외 변환을 덧붙인 것이라서 Proxy 대응은 [Nova v2 Python proxy 대응](v2/python-parity.md)과 [관리자 API 대응](v2/python-parity-admin.md)을 먼저 참고하면 좋습니다. 비교 기준이 되는 계약 테스트는 [compute/v2/python_parity_test.go](v2/python_parity_test.go)에 있습니다.

`go_mapping`은 기본 인자를 포함한 모든 인자에 대해 Python helper가 보내는 요청 순서와 돌려주는 정보를 공개 Go API가 그대로 만들 수 있다는 뜻입니다. 이때 Go 호출을 순서대로 이어 부르거나 찾은 결과가 nil인지 확인하는 정도는 caller가 맡습니다. 반면 목록 행을 직접 거르거나, 특정 HTTP status를 caller가 성공이나 빈 값으로 바꾸거나, Go에 없는 helper 로직을 직접 구현해야 한다면 `unresolved`로 판정했습니다. 표의 Go 호출은 `conn.ComputeV2(ctx)`가 돌려준 service의 필드를 기준으로 적었고, Glance 호출은 `conn.ImageV2(ctx)`의 `Images`입니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `add_flavor_access` | `Flavors.AddAccess(ctx, flavorID, flavors.AddAccessOpts{Tenant})` | go_mapping |
| `remove_flavor_access` | `Flavors.RemoveAccess(ctx, flavorID, flavors.RemoveAccessOpts{Tenant})` | go_mapping |
| `list_flavor_access` | `Flavors.ListAccesses(ctx, flavorID)` | go_mapping |
| `set_flavor_specs` | `Flavors.CreateExtraSpecs(ctx, flavorID, flavors.ExtraSpecsOpts{...})` | go_mapping |
| `unset_flavor_specs` | key마다 `resource.IgnoreMissing(Flavors.DeleteExtraSpec(ctx, flavorID, key))` | go_mapping |
| `create_flavor` | `Flavors.Create(ctx, flavors.CreateOpts{...})` | go_mapping |
| `delete_flavor` | `Flavors.FindIdentity(ctx, nameOrID)` 다음 `Flavors.Remove(ctx, resource.ID(id))` | go_mapping |
| `get_flavor_name` | `Flavors.FindIdentity(ctx, flavorID)`의 `Name` | go_mapping |
| `get_flavor_by_ram` | `conn.Compute(ctx)`의 `AllFlavors` 뒤 선택 helper 없음 | unresolved |
| `create_aggregate` | `Aggregates.Create(ctx, aggregates.CreateOpts{Name, AvailabilityZone})` | go_mapping |
| `list_aggregates` | `Aggregates.All(ctx, resource.WithName(...))`, `Aggregates.List(ctx)` | go_mapping |
| `search_aggregates` | `Aggregates.All(ctx)` 뒤 cloud 필터 helper 없음 | unresolved |
| `get_aggregate` | 이름 fallback 조회 helper 없음 | unresolved |
| `update_aggregate` | 이름 fallback 조회 helper 없음, `Aggregates.Update` | unresolved |
| `delete_aggregate` | 이름 fallback 조회 helper 없음, `Aggregates.Delete` | unresolved |
| `set_aggregate_metadata` | 이름 fallback 조회 helper 없음, `Aggregates.SetMetadata` | unresolved |
| `add_host_to_aggregate` | 이름 fallback 조회 helper 없음, `Aggregates.AddHost` | unresolved |
| `remove_host_from_aggregate` | 이름 fallback 조회 helper 없음, `Aggregates.RemoveHost` | unresolved |
| `list_hypervisors` | `Hypervisors.List(ctx, ...)`, 2.53 미만 검색 경로 없음 | unresolved |
| `create_server_group` | `ServerGroups.Create(ctx, servergroups.CreateOpts{Name, Policy})` | go_mapping |
| `list_server_groups` | `ServerGroups.List(ctx)` | go_mapping |
| `search_server_groups` | `Connection.SearchServerGroups(ctx, nameOrID, options...)` ([사용법](cloud-server-group-reads-python.md)) | unresolved |
| `get_server_group` | `Connection.GetServerGroup(ctx, nameOrID, options...)` ([사용법](cloud-server-group-reads-python.md)) | unresolved |
| `delete_server_group` | `ServerGroups.FindRecord(ctx, nameOrID)` 다음 `ServerGroups.Delete(ctx, id)` | go_mapping |
| `get_server_id` | `Servers.FindRecord(ctx, nameOrID)`의 `id` | go_mapping |
| `set_server_metadata` | `Servers.FindRecord` 다음 `Servers.UpdateMetadata(ctx, id, servers.MetadataOpts{...})` | go_mapping |
| `delete_server_metadata` | `Servers.FindRecord` 다음 key마다 `Servers.DeleteMetadatum(ctx, id, key)` | go_mapping |
| `list_server_security_groups` | `Servers.Get(ctx, id)` 다음 `SecurityGroups.ListByServer(ctx, id)` | go_mapping |
| `create_image_snapshot` | `Servers.FindRecord`, `Servers.CreateImage`, `Images.FindIdentity`, `Images.WaitFor` | go_mapping |
| `get_server` | `Servers.FindRecord` 뒤 주소 확장 통합 없음 | unresolved |
| `get_server_by_id` | `Servers.Get` 뒤 주소 확장 통합 없음 | unresolved |
| `list_servers` | `Servers.List` 뒤 주소 확장 통합 없음 | unresolved |
| `search_servers` | cloud 필터 helper와 주소 확장 없음 | unresolved |
| `update_server` | `Servers.FindRecord`, `Servers.Update` 뒤 주소 확장 통합 없음 | unresolved |
| `rebuild_server` | `Servers.Rebuild` 뒤 주소 확장과 대기 helper 없음 | unresolved |
| `delete_server` | `Servers.FindRecord`, `Servers.Remove`, 대기와 floating IP 정리 없음 | unresolved |
| `get_server_console` | `Connection.GetServerConsole(ctx, nameOrID, options...)` ([사용법](cloud-server-console-python.md)) | unresolved |
| `add_server_security_groups` | cloud security group 조회 helper 없음, `SecurityGroups.AddServer` | unresolved |
| `remove_server_security_groups` | cloud security group 조회 helper 없음, `SecurityGroups.RemoveServer` | unresolved |
| `get_compute_limits` | `Limits.Fetch(ctx)`, project 이름 fallback 조회 없음 | unresolved |
| `get_compute_usage` | `Usage.SingleTenant`, project 이름 fallback 조회 없음 | unresolved |
| `get_openstack_vars` | 없음 | unresolved |
| `get_server_meta` | 없음 | unresolved |

새 Source 전용 서버 그룹·콘솔 helper의 옵션·microversion·Resource 결과는 위 가이드의 계약을 적용합니다. 아래 typed 기존 API의 공통 차이를 새 helper 전체에 확대 적용하지 않습니다.

## 공통 차이

Python cloud helper는 Resource 객체, dict 또는 이름이나 ID 문자열을 받지만 Go 호출은 ID 문자열이나 `resource.Ref`를 받습니다. 결과도 Python Resource 대신 Gophercloud typed model이나 SDK record이므로 모델에 없는 응답 field는 typed 결과에 남지 않습니다. `Servers.FindRecord`와 `ServerGroups.FindRecord`는 선언된 Python 속성을 `Resource.Body`에, 실제 응답을 `Wire`에 JSON으로 담습니다.

Python은 Resource의 `_max_microversion`(Flavor 2.61, Aggregate 2.81, ServerGroup 2.64, Server 2.100, Limits 2.57, Usage 2.75, Hypervisor 2.88)과 서버 최대값 가운데 작은 값을 요청마다 골라 보냅니다. Go는 Connection에 설정하거나 협상한 client microversion 하나를 보냅니다. 같은 header가 필요하면 `RawClient()` 복사본의 `Microversion`을 바꿔 새 API를 만듭니다.

Python helper가 True/False나 None을 돌려주는 자리에서 Go는 찾은 값이나 nil, 그리고 error를 돌려줍니다. 그래서 caller는 `FindIdentity`나 `FindRecord`가 nil을 돌려주면 Python의 False 또는 None으로 해석하고, 그 뒤 요청을 보내지 않습니다. Python이 `SDKException`으로 바꾸어 올리는 오류는 Go에서 원래 HTTP 오류나 조회 오류로 남습니다.

## Flavor helper

`add_flavor_access`, `remove_flavor_access`는 `POST flavors/{id}/action`에 `addTenantAccess`나 `removeTenantAccess`를 보내고, `list_flavor_access`는 `GET flavors/{id}/os-flavor-access`를 읽습니다. Python은 action에서 None을 돌려주고 Go는 응답의 access 목록을 돌려줍니다. Python `_action`은 빈 Accept header를 보내고 400 미만을 모두 성공으로 보지만 Go는 Gophercloud 기본값을 씁니다. `set_flavor_specs`는 `CreateExtraSpecs`와 같은 `POST os-extra_specs`를 보내며 Go `ExtraSpecsOpts`는 문자열 값만 받습니다.

`unset_flavor_specs`는 key마다 Proxy의 `delete_flavor_extra_specs_property`를 부르고, 이 Proxy는 `ignore_missing=True`가 기본이라 404를 무시합니다. Go에서는 key마다 `resource.IgnoreMissing(Flavors.DeleteExtraSpec(...))`을 부르면 404만 nil이 되고 409 같은 다른 오류는 그대로 돌아오므로 Python처럼 그 자리에서 멈춥니다.

`create_flavor`는 기본값 `flavorid="auto"`, `ephemeral=0`, `swap=0`, `rxtx_factor=1.0`, `is_public=True`를 모두 본문에 넣습니다. Go `CreateOpts`에는 `Swap`, `Ephemeral`, `IsPublic`을 pointer로, `RxTxFactor`를 1.0으로 직접 채워야 같은 본문이 됩니다. Python은 `flavorid="auto"`와 description 생략을 `"id": null`, `"description": null`로 보내지만 Go는 두 key를 생략하며, Nova는 두 경우를 같게 처리합니다.

`get_flavor_name`과 `delete_flavor`는 `find_flavor`로 이름이나 ID를 찾습니다. `Flavors.FindIdentity`가 `GET flavors/{name_or_id}`를 먼저 보내고 없으면 `GET flavors/detail?is_public=None` 목록에서 찾으며, 찾지 못하면 nil을 돌려줍니다. `delete_flavor`는 그 뒤 `Flavors.Remove(ctx, resource.ID(id))`로 404를 무시하는 DELETE를 보냅니다. Python은 실패를 `Unable to delete flavor` 메시지의 `SDKException`으로 바꾸지만 Go는 원래 오류를 돌려줍니다.

## Aggregate helper

`create_aggregate`는 Python이 zone을 주지 않으면 `"availability_zone": null`을 보내고 Go는 key를 생략합니다. `list_aggregates`는 `GET os-aggregates`이며, Python은 filters의 Aggregate 속성 key를 응답 행에 대한 로컬 필터로 씁니다. Go에서는 같은 조건을 `resource.WithName`이나 `resource.WithFilter` 옵션으로 `Aggregates.All`에 넘깁니다.

이름이나 ID를 받는 aggregate helper는 모두 `find_aggregate(name_or_id, ignore_missing=True)`를 먼저 부릅니다. 이 조회는 `GET os-aggregates/{name_or_id}`를 보내고 400·403·404이면 목록에서 찾습니다. Go `Aggregates.Find(resource.ID(x))`는 정수가 아닌 이름을 HTTP 전에 거부하고 `Find(resource.Name(x))`는 첫 GET을 건너뛰므로 같은 요청 순서를 만들 수 없습니다.

## Hypervisor helper

`list_hypervisors()`는 항상 `details=True`라서 기본 호출은 Go `Hypervisors.List`와 같은 `GET os-hypervisors/detail`입니다. 다만 filters에 `hypervisor_hostname_pattern`을 주고 서버가 2.53 미만이면 Python은 `GET os-hypervisors/{pattern}/search`로 바꿉니다. Gophercloud v2.15.0에는 이 경로의 binding이 없습니다.

## Server group helper

`create_server_group(name, policies=[...])`는 Proxy를 거치며 서버가 2.64 이상이면 `policies`의 첫 값을 `policy`로 바꾸고, 그보다 낮으면 `policy`를 `policies`로 바꿉니다. Go는 이 변환을 하지 않으므로 caller가 client microversion에 맞춰 `Policy` 또는 `Policies`를 채웁니다. `list_server_groups`는 같은 `GET os-server-groups`입니다.

`delete_server_group`은 `find_server_group`으로 찾은 뒤 `ignore_missing=False`로 DELETE를 보냅니다. Go도 `ServerGroups.FindRecord` 다음 `ServerGroups.Delete`를 부르며, DELETE의 404는 Python의 `NotFoundException`처럼 오류로 돌아옵니다. 찾지 못한 group은 두 쪽 모두 DELETE를 보내지 않습니다. `Connection.GetServerGroup`은 filters nil/null의 기본 Source Find와 명시 filters의 cloud 검색 경로를 구분합니다. `SearchServerGroups`는 glob·dict·JMESPath와 finite 숫자 비교를 처리하지만 Resource 클래스 함수·plain dictionary와의 동등성·전체 Python 산술/직렬화/비유한 숫자가 남아 두 연산은 unresolved입니다.

## Server helper

이름이나 ID로 서버를 받는 helper는 `get_server(name_or_id, bare=True)`로 서버를 찾고, 이것은 `find_server(details=True, all_projects=False)`와 같습니다. Go `Servers.FindRecord(ctx, nameOrID)`의 기본값이 같아서 `GET servers/{name_or_id}` 다음 `GET servers/detail?name=...`을 보내고, 없으면 nil을 돌려줍니다. 서버 ID는 `Resource.Body["id"]`에서 읽습니다. Python은 서버가 없을 때 `set_server_metadata`와 `delete_server_metadata`에서 `SDKException`을 올리고, Go caller는 nil을 받은 시점에서 멈춥니다.

`set_server_metadata`는 `POST servers/{id}/metadata`로 병합하고, Go `UpdateMetadata`는 병합된 metadata 전체를 돌려줍니다. `delete_server_metadata`는 key마다 `DELETE servers/{id}/metadata/{key}`를 보냅니다. `list_server_security_groups`는 이름 조회 없이 `GET servers/{id}`와 `GET servers/{id}/os-security-groups`를 보냅니다. Python은 `secgroup_source` 설정이 비어 있거나 nova·neutron이 아니면 요청 없이 빈 목록을 돌려주지만 Go에는 이 설정이 없습니다.

`create_image_snapshot`은 서버를 찾은 뒤 `createImage` action을 보내고 Glance에서 image를 읽습니다. `wait=True`이면 `Images.WaitFor(ctx, resource.ID(id), "active", resource.WithTimeout(3600*time.Second), resource.WithFailureStates("error"))`가 Python의 3600초 기본 대기에 해당합니다. Python은 metadata 인자가 없어도 `"metadata": {}`를 보내지만 Go는 빈 map을 생략합니다. Python 대기는 image를 찾지 못해도 계속 polling하고 timeout 예외를 올리며, Go는 404를 바로 오류로 돌려주고 timeout은 context deadline 오류로 나타납니다.

`get_server`, `get_server_by_id`, `list_servers`, `search_servers`, `update_server`, `rebuild_server`는 기본값 `bare=False`에서 `_expand_server`를 거쳐 `meta.add_server_interfaces`로 Neutron이나 Nova floating IP와 port를 조회하고 `public_v4`, `public_v6`, `private_v4`, `interface_ip`를 채웁니다. `detailed=True`이면 `meta.get_hostvars_from_server`까지 실행합니다. Go의 Get·Find·List·Update·Rebuild 결과에는 이 확장이 적용되지 않고, `compute.Service.ExpandServerInterfaces`는 [별도 view](server-addresses.md)를 만드는 독립 호출이라서 같은 결과를 돌려주지 않습니다.

## Limits와 usage helper

`get_compute_limits()`처럼 project를 주지 않으면 Go `Limits.Fetch(ctx)`가 같은 `GET limits`를 보내고 `Absolute`에 같은 값을 담습니다. project 이름이나 ID를 주면 Python은 Identity v3 `find_project(ignore_missing=False)`로 ID를 찾은 뒤 `tenant_id` query를 붙입니다. `get_compute_usage`도 항상 같은 조회를 하므로 두 helper 모두 Go에서 같은 조회 요청을 만들 수 없습니다.

## 남은 메서드와 부족한 기능

- `get_flavor_by_ram`: flavor 목록을 `ram`으로 정렬하고 `include` 부분 문자열로 고르는 helper가 없습니다.
- `search_aggregates`, `search_servers`: cloud `_utils._filter_list`의 name_or_id glob, dict 필터, JMESPath 필터를 적용하는 공개 helper가 없습니다. `search_servers`는 서버 주소 확장도 없습니다.
- `get_aggregate`, `update_aggregate`, `delete_aggregate`, `set_aggregate_metadata`, `add_host_to_aggregate`, `remove_host_from_aggregate`: `find_aggregate`의 GET 후 목록 fallback이 없습니다. `update_aggregate`는 Proxy 수준에서도 `availability_zone=None`을 보낼 수 없고, `delete_aggregate`는 숫자 ID의 404를 False로 바꾸는 동작을 caller가 직접 처리해야 합니다.
- `list_hypervisors`: 2.53 미만의 `os-hypervisors/{pattern}/search` 경로가 없습니다.
- `search_server_groups`, `get_server_group`: [공개 helper](cloud-server-group-reads-python.md)는 구현됐지만 Resource 클래스의 JMESPath 함수/동등성 및 전체 Python 산술·직렬화·비유한 숫자 처리 차이가 남습니다.
- `get_server`, `get_server_by_id`, `list_servers`, `update_server`, `rebuild_server`: 기본 주소 확장이 Go 조회·변경 결과에 통합되어 있지 않습니다. `get_server_by_id`는 404를 None으로 바꾸는 동작도 없고, `rebuild_server(wait=True)`는 미해결인 `wait_for_server`와 admin password 보존이 필요합니다.
- `delete_server`: `wait=True`가 쓰는 Proxy `wait_for_delete`가 미해결이고, `delete_ips=True`의 floating IP 정리와 재시도를 서버 삭제와 묶는 helper가 없습니다.
- `get_server_console`: [공개 helper](cloud-server-console-python.md)가 bare 조회·supplied dictionary·길이 옵션·깨끗한 action400 변환을 처리합니다. Requests JSON charset 처리와 Python 비유한 숫자는 미구현입니다.
- `add_server_security_groups`, `remove_server_security_groups`: `secgroup_source`에 따라 Neutron이나 Nova에서 security group을 찾는 cloud `get_security_group`과 기능 확인이 없습니다. 제거 helper는 group별 404를 False로 바꾸는 동작도 필요합니다.
- `get_compute_limits`, `get_compute_usage`: Identity v3 `find_project`의 이름·ID fallback 조회가 없고, `get_compute_usage`는 ISO 8601 문자열 날짜를 해석하는 helper도 없습니다.
- `get_openstack_vars`, `get_server_meta`: Ansible hostvars와 inventory group을 만드는 `meta.get_hostvars_from_server`, `meta.get_groups_from_server`에 해당하는 Go helper가 없습니다.
