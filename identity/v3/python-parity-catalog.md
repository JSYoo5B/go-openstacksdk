# openstacksdk identity v3 catalog Proxy 대응

이 문서는 고정한 openstacksdk 커밋 `ef55d7d`의 [identity v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py) 가운데 endpoint, region, service, limit, registered limit, project endpoint, endpoint group, domain config 메서드와 공통 wait 메서드를 이 SDK의 `identity/v3` 호출과 비교합니다. Keystone 요청 자체의 status와 decode 규칙은 [native 관리자 호출 문서](native-admin-b.md)에 있고, 여기서는 Python 코드를 옮길 때 달라지는 기본값과 반환값을 정리합니다. 판정이 `go_mapping`이면 Python이 보내는 요청을 공개 Go API로 재현할 수 있다는 뜻이고, `unresolved`이면 Go에 없는 동작이 남아 있다는 뜻입니다. 기존 native 판정은 `endpoints/regions/services/limits/registeredlimits/projectendpoints`의 `python_parity_test.go`에 기록하고, 새 limits·registered_limits record 조회는 각 `python_records_test.go`와 [Limit 가이드](limits/python_records.md)·[RegisteredLimit 가이드](registeredlimits/python_records.md)에 근거를 둡니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`create_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L404) | `endpoints.API.Create` | go_mapping |
| [`delete_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L415) | `endpoints.API.Remove` | go_mapping |
| [`find_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L448) | `endpoints.API.Find`를 ID, 이름 순서로 호출 | go_mapping |
| [`get_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L468) | `endpoints.API.Get` | go_mapping |
| [`endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L483) | `endpoints.API.List` (세 필터만) | unresolved |
| [`update_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L497) | `endpoints.API.Update` | go_mapping |
| [`endpoint_projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1130) | 없음 | unresolved |
| [`project_endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L537) | `projectendpoints.API.List` (query 없는 호출만) | unresolved |
| [`associate_endpoint_with_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L557) | `projectendpoints.API.Create` | go_mapping |
| [`disassociate_endpoint_from_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L574) | `projectendpoints.API.Delete` | go_mapping |
| [`create_region`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1594) | `regions.API.Create` | go_mapping |
| [`delete_region`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1605) | `regions.API.Remove` | go_mapping |
| [`find_region`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1636) | `regions.API.Find` (ID 조회만) | unresolved |
| [`get_region`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1660) | `regions.API.Get` | go_mapping |
| [`regions`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1672) | `regions.API.List`, `regions.API.All` | unresolved |
| [`update_region`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1683) | `regions.API.Update` | go_mapping |
| [`create_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1232) | `services.API.Create` | go_mapping |
| [`delete_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1243) | `services.API.Remove` | go_mapping |
| [`find_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1274) | `services.API.Find`를 ID, 이름 순서로 호출 | go_mapping |
| [`get_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1293) | `services.API.Get` | go_mapping |
| [`services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1305) | `services.API.List`, `services.API.All` | unresolved |
| [`update_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1319) | `services.API.Update` | go_mapping |
| [`create_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2523) | 한 행짜리 `limits.API.BatchCreate` | go_mapping |
| [`delete_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2548) | `limits.API.Remove` | go_mapping |
| [`get_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2511) | `limits.API.Get` | go_mapping |
| [`limits`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2501) | `limits.API.ListRecords/AllRecords/ProxyListRecords` | go_mapping |
| [`update_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2534) | `limits.API.Update` | go_mapping |
| [`create_registered_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2442) | 한 행짜리 `registeredlimits.API.BatchCreate` | go_mapping |
| [`delete_registered_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2474) | `registeredlimits.API.Remove` | go_mapping |
| [`get_registered_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2425) | `registeredlimits.API.Get` | go_mapping |
| [`registered_limits`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2411) | `registeredlimits.API.ListRecords/AllRecords/ProxyListRecords` | go_mapping |
| [`update_registered_limit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2455) | `registeredlimits.API.Update` | go_mapping |
| [`create_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L593) | 없음 | unresolved |
| [`delete_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L606) | 없음 | unresolved |
| [`find_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L645) | 없음 | unresolved |
| [`get_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L668) | 없음 | unresolved |
| [`endpoint_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L684) | 없음 | unresolved |
| [`update_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L697) | 없음 | unresolved |
| [`project_endpoint_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L714) | 없음 | unresolved |
| [`endpoint_group_endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L511) | 없음 | unresolved |
| [`endpoint_group_projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1166) | 없음 | unresolved |
| [`associate_project_with_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1190) | 없음 | unresolved |
| [`disassociate_project_from_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1210) | 없음 | unresolved |
| [`create_domain_config`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L317) | 없음 | unresolved |
| [`delete_domain_config`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L339) | 없음 | unresolved |
| [`get_domain_config`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L362) | 없음 | unresolved |
| [`update_domain_config`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L382) | 없음 | unresolved |
| [`wait_for_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3343) | 각 패키지의 `WaitFor` | unresolved |
| [`wait_for_delete`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3382) | 각 패키지의 `WaitForDeletion` | unresolved |

이 표의 49개 선언 중 26개가 `go_mapping`, 23개가 `unresolved`입니다. 이번 정합성 갱신은 Project·Group·Limit 조회에 한정하며, 다른 연산의 기존 기록은 유지합니다. 전체 SDK 진행량은 [중앙 지원 기록](../../docs/sdk-support-ledger.md)에서 집계합니다.

## 공통 차이

Python Proxy는 `_create`, `_get`, `_update`, `_delete`를 거쳐 [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)의 create, fetch, commit, delete를 부르고, 응답은 `raise_from_response`로 검사해서 400 미만 status를 모두 성공으로 받습니다. 기존 native 호출은 Gophercloud가 정한 OkCodes를 유지하며 그 목록은 [native 문서](native-admin-b.md)에 있습니다. 새 Limit·RegisteredLimit record 목록은 Source HTTP100..399와 실제 read/Close/decode 오류 증거를 처리합니다. Source Limit·RegisteredLimit는 `_max_microversion=None`이라 자체 최대 버전 discovery를 요구하지 않습니다. 새 record 목록은 선택된 default/per-call microversion과 request header를 concrete 옵션으로 소유하고, 동적 session/version 협상은 별도 SDK 범위입니다.

Python은 ID 문자열과 Resource 객체를 모두 받지만 Go는 ID 문자열만 받습니다. 기존 native 반환값은 Python Resource 대신 Gophercloud 모델 pointer이고, Python의 `is_enabled`는 Go에서 `Enabled`, endpoint의 `interface`는 `Availability`입니다.

수정은 모두 `PATCH`입니다. Python `commit`은 ID를 변경 목록에서 빼므로 본문에 `id`가 들어가지 않고, 바뀐 속성이 없으면 요청 자체를 보내지 않습니다. Go `Update`는 빈 옵션이어도 빈 envelope로 PATCH를 보냅니다. Python은 `description=None`처럼 값을 null로 보내 지울 수 있지만, Go typed 필드는 빈 값을 생략하고 같은 이름의 `WithUpdateField`는 HTTP 전에 거부합니다. 그래서 pointer가 아닌 문자열 필드를 null이나 빈 문자열로 바꾸는 요청은 Go에서 만들 수 없습니다.

삭제는 Python `ignore_missing=True` 기본값에 맞춰 `Remove(ctx, resource.ID(id))`를 씁니다. 404는 nil이 되고 `resource.WithMissingError()`를 주면 Python `ignore_missing=False`처럼 `resource.ErrNotFound`를 돌려줍니다. 409 같은 다른 오류는 그대로 돌려줍니다. generated `Delete`를 직접 부르면 404도 오류입니다.

```go
// Python: conn.identity.delete_region("RegionOne")
err := regions.New(client).Remove(ctx, resource.ID("RegionOne"))
// Python: conn.identity.delete_region("RegionOne", ignore_missing=False)
err = regions.New(client).Remove(ctx, resource.ID("RegionOne"), resource.WithMissingError())
```

## endpoint

`create_endpoint`와 `update_endpoint`는 Python Endpoint 속성(`interface`, `enabled`, `region_id`, `service_id`, `url`, `name`) 가운데 넘긴 값만 `endpoint` envelope에 담습니다. Go `CreateOpts`의 `Region`은 옛 `region` key로 나가므로, Python과 같은 `region_id`를 보내려면 `WithCreateField("region_id", ...)`나 `WithUpdateField("region_id", ...)`를 씁니다. `CreateOpts`는 `interface`, `url`, `service_id`가 필수라서 하나라도 비면 Go는 HTTP 전에 거부하지만, Python은 그대로 보내고 Keystone의 거부를 받습니다. Python Endpoint에 없는 `description`은 Python이 보내지 못하고 Go는 보낼 수 있습니다.

```go
disabled := false
ep, err := endpoints.New(client).Create(ctx,
	endpoints.CreateOpts{Availability: gophercloud.AvailabilityPublic, URL: "https://nova", ServiceID: "svc-1", Enabled: &disabled},
	endpoints.WithCreateField("region_id", "RegionOne"))
```

`find_endpoint`는 `GET endpoints/{name_or_id}`가 400, 403, 404로 실패하면 Endpoint에 name query가 없으므로 필터 없는 `GET endpoints`를 보내고 ID나 이름이 같은 행을 고릅니다. Go에서는 `Find(ctx, resource.ID(x), resource.WithIgnoreMissing())`가 nil일 때 `Find(ctx, resource.Name(x), resource.WithIgnoreMissing())`를 이어 부르면 같은 두 요청이 나갑니다. 다만 Go는 ID GET의 404만 다음 단계로 넘기고 400과 403은 오류로 돌려주며, 목록에서는 이름만 비교합니다. 이름이 겹치면 Python은 `DuplicateResource`, Go는 `resource.ErrAmbiguous`입니다.

`endpoints(**query)`는 `interface`, `region_id`, `service_id`와 공통 `limit`, `marker`를 query로 보내고, 그 밖의 Endpoint 속성(`url`, `is_enabled` 등)은 받은 행을 로컬에서 거릅니다. Go `List`는 `ListOpts`의 세 필터만 보내고, `All(ctx, resource.WithQuery(...))`도 세 필터 key만 받아 `limit`과 `marker`는 HTTP 전에 거부합니다. 로컬 속성 필터도 없어서 이 메서드는 `unresolved`입니다.

## region

`create_region`은 넘긴 `id`, `description`, `parent_region_id`를 `region` envelope로 보내고 Go `CreateOpts`의 같은 이름 필드가 같은 본문을 만듭니다. `update_region`의 `description`은 Go에서 pointer라 빈 문자열도 보낼 수 있지만, `parent_region_id`를 null로 지우는 요청은 만들 수 없습니다.

`find_region`은 Python에서 deprecated 경고를 내고 `GET regions/{name_or_id}` 다음에 필터 없는 `GET regions`를 보내 ID나 이름을 비교합니다. Go `regions` collection에는 이름 selector가 없어서 `Find(resource.Name(x))`는 HTTP 전에 `resource.ErrUnsupported`를 돌려주고, ID GET 하나만 재현할 수 있습니다.

## service

`create_service`와 `update_service`는 `name`, `type`, `description`, `enabled` 가운데 넘긴 값만 보냅니다. Go `CreateOpts.Type`은 omitempty가 없어서 Python이 `type` 없이 만들 때와 달리 `"type": ""`를 항상 보냅니다. 수정의 `name`, `description`, `enabled`는 pointer라 빈 값도 보낼 수 있습니다.

`find_service`는 Service가 `name` query를 지원하므로 ID GET이 실패하면 `GET services?name={name_or_id}`를 보냅니다. Go의 `Find(resource.Name(x))`도 같은 `name` query를 보낸 뒤 이름을 다시 비교하므로 두 요청이 같습니다. 400과 403 처리, 중복 오류의 차이는 endpoint와 같습니다.

## limit과 registered limit

Python `create_limit`과 `create_registered_limit`은 단일 본문을 한 원소 배열로 바꿔 `{"limits": [...]}`, `{"registered_limits": [...]}`를 보내고, 응답 배열의 첫 원소만 Resource로 읽습니다. Go는 한 행짜리 `BatchCreate`가 같은 요청을 보내고 slice를 돌려주므로 `[0]`을 씁니다. Go는 `service_id`와 `resource_name`이 비면 HTTP 전에 거부하고, `resource_limit`과 `default_limit`은 0이어도 항상 보냅니다.

```go
rows, err := limits.New(client).BatchCreate(ctx, limits.BatchCreateOpts{
	{ServiceID: "svc-1", ProjectID: "p", ResourceName: "cores", ResourceLimit: 10},
})
limit := rows[0]
```

수정은 `limit`, `registered_limit` envelope의 PATCH이고, `ResourceLimit`, `DefaultLimit`, `Description`은 pointer라 0과 빈 문자열을 보낼 수 있습니다. registered limit의 `region_id`, `service_id`, `resource_name`을 null로 지우는 요청은 Go에서 만들 수 없습니다.

`limits(**query)`와 새 `limits.WithRecordListQuery`는 `service_id/region_id/resource_name/project_id/limit/marker`를 서버로 보내고 `domain_id/description/links/resource_limit/id/name`은 local Body filter로 처리합니다. RegisteredLimit의 server query에는 project_id가 없고 local 속성은 `description/links/default_limit/id/name`입니다. unknown kwargs는 JSON marshaler 호출 전에 버리며, `resource_limit/default_limit`은 Source type=int 선언이 없어 숫자·문자열·false·null 그대로 보존합니다. CPython loaded equality와 ordered nested filter, immutable 옵션, Source 기본값과 독립적인 Resource/Wire/Envelope/header/status를 SDK가 소유합니다. 기존 native `List/All`과 raw `WithQuery`는 종전 typed/wire 계약을 유지합니다.

두 목록은 `go_mapping`이며 [2025.2 Keystone 정책](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)의 `identity:list_limits/list_registered_limits`는 빈 check string과 system/domain/project scope를 사용합니다. 조회를 admin 전용으로 분류하지 않으며 인증·실제 서버 scope·운영 policy는 서버가 적용합니다. write/getter 정책을 이 목록 판정으로 확대하지 않습니다.

직접 record 조회의 location은 기본 null이고 `WithRecordListLocation(resource.CloudLocation)`으로 기록된 값을 명시적으로 공급합니다. Limit의 선언 project_id는 그 location에 반영하지만 RegisteredLimit에는 project_id descriptor가 없습니다. SDK-R1/C1/S1의 Connection 전체 Resource·cache·session 계약은 별도로 남습니다.

## 목록 공통

Python 목록은 Keystone의 `links` 객체를 rel/href 항목으로 해석해 `{"next": url}` 모양을 따라가지 않습니다. 새 limits/registered_limits record API도 dictionary next를 무시하고 root links가 plural links를 가리는 Source 우선순위를 지킵니다. explicit 양수 limit이면 짧은 nonempty page도 마지막 raw ID를 marker로 재요청하며 empty page에서 끝납니다. local filter로 제외한 행도 marker·raw cap에 포함합니다. typed `ListRecords`와 opaque generator/JMESPath 결과를 위한 `ProxyListRecords`를 구분하고, 순회 중 break·cap 후 추가 HTTP를 하지 않습니다. Go의 고정 origin/path/query·cycle 검증과 RFC HTTP Link 호환 처리는 별도 정책입니다. 기존 native `List/All`의 links.next pager는 유지하며, `regions/services/endpoints`의 local Body filter·limit marker 문제는 기존 unresolved 범위에 남습니다.

`project_endpoints(project)`는 query 없이 부르면 `GET OS-EP-FILTER/projects/{project}/endpoints` 하나를 보내고 Go `projectendpoints.API.List`가 같은 요청을 보냅니다. Go 목록은 옵션을 받지 않아 Endpoint query 필터와 `limit`, `marker`를 붙일 수 없고, 행 모델에는 `enabled`, `region_id`, `links`가 없습니다.

## project endpoint 연결

`associate_endpoint_with_project`와 `disassociate_endpoint_from_project`는 본문 없는 `PUT`, `DELETE OS-EP-FILTER/projects/{project}/endpoints/{endpoint}`를 보내고, 404를 포함한 오류 status에서 예외를 냅니다. Go `projectendpoints.API.Create`, `Delete`가 같은 요청이며 404도 오류로 돌려줍니다. Python은 400 미만 status를 모두 받지만 Go는 204만 받습니다.

## wait

`wait_for_status`는 Resource의 `status` 속성(또는 `attribute`)을 `interval`초마다 다시 읽고, `wait=None`이면 끝없이 기다리며, 실패 상태 기본값은 `ERROR`입니다. 이 문서의 리소스에는 status 속성이 없어서 Python은 `AttributeError`를 내고, Go `WaitFor`도 HTTP 전에 `resource.ErrUnsupported`를 돌려줍니다. 다른 필드를 비교하려면 `resource.WithStatusAttribute`를 쓰고, Python 기본값을 맞추려면 `resource.WithUnlimitedWait()`와 `resource.WithPollInterval(2*time.Second)`를 줍니다. Go 기본 제한 시간은 5분입니다.

`wait_for_delete`의 기본 제한 시간은 120초입니다. Go `WaitForDeletion`은 같은 ID를 다시 GET해서 404가 나오면 성공하므로 `resource.WithTimeout(120*time.Second)`를 주면 같은 동작입니다.

두 메서드 모두 identity v3의 모든 Resource를 받지만 Go 도우미는 패키지마다 따로 있고, Python처럼 이미 목표 상태인 Resource를 HTTP 없이 돌려주거나 삭제 후 원래 Resource를 돌려주는 동작이 없습니다. 그래서 둘 다 `unresolved`입니다.

## unresolved 목록

- `endpoints`, `regions`, `services`: Python의 로컬 Body 속성 필터와 `limit` 기반 marker 반복 요청이 Go에 없습니다. `endpoints`는 `limit`, `marker` query도 보낼 수 없습니다.
- `project_endpoints`: Go 목록이 옵션을 받지 않아 query 필터와 페이지 인자를 보낼 수 없습니다.
- `find_region`: region 이름 selector가 없어 ID GET 다음의 목록 fallback을 재현할 수 없습니다.
- `create_endpoint_group`, `delete_endpoint_group`, `find_endpoint_group`, `get_endpoint_group`, `endpoint_groups`, `update_endpoint_group`, `project_endpoint_groups`, `endpoint_group_endpoints`, `endpoint_group_projects`, `associate_project_with_endpoint_group`, `disassociate_project_from_endpoint_group`: Gophercloud v2.15.0에 `OS-EP-FILTER/endpoint_groups` 경로와 `OS-EP-FILTER/projects/{project}/endpoint_groups` 경로의 binding이 없어서 SDK 소유 호출이 필요합니다.
- `endpoint_projects`: `OS-EP-FILTER/endpoints/{endpoint}/projects` binding이 없습니다.
- `create_domain_config`, `delete_domain_config`, `get_domain_config`, `update_domain_config`: `domains/{domain}/config` binding이 없습니다.
- `wait_for_status`, `wait_for_delete`: 위 wait 절의 Resource 단락 동작과 반환값 차이가 남아 있습니다.
