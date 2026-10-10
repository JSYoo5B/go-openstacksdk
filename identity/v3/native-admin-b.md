# Keystone v3 native catalog·policy·limit·mapping 관리자 호출

`service.Regions`, `service.Services`, `service.Endpoints`, `service.Policies`, `service.Limits`, `service.RegisteredLimits`, `service.Projectendpoints`, `service.Federation`(각각 `identity/v3`의 같은 이름 소문자 패키지)의 generated 메서드는 Gophercloud `v2.15.0`의 같은 이름 패키지 요청([regions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/regions/requests.go), [services](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/services/requests.go), [endpoints](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/endpoints/requests.go), [policies](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/policies/requests.go), [limits](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/limits/requests.go), [registeredlimits](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/registeredlimits/requests.go), [projectendpoints](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/projectendpoints/requests.go), [federation](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/federation/requests.go))을 바꾸지 않고 호출합니다. SDK는 단건 호출 오류에 `resource.OperationError{Resource: "<패키지 이름>"}` 문맥만 더하고, 허용하지 않은 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. 목록 stream 오류에는 이 문맥을 붙이지 않습니다.

이 문서의 생성·수정·삭제는 모두 기본 Keystone 정책상 관리자 호출입니다. 조회는 리소스마다 다릅니다. region과 registered limit 조회, limit enforcement model 조회는 인증된 사용자 누구나 할 수 있고, limit 조회는 자기 project나 domain의 limit에 한해 일반 사용자도 할 수 있습니다. service, endpoint, policy, project endpoint, mapping 조회는 관리자나 system reader 호출입니다.

## 공통 규칙

수정은 모두 `PATCH`입니다. 목록은 Keystone `links.next` 문자열을 따라가며 null이나 빈 문자열이면 멈춥니다. 다음 페이지 URL은 절대 URL 그대로 요청하므로 다른 경로를 가리켜도 따라갑니다. 목록 첫 요청은 native pager의 200, 204, 300을 받는데, 본문 없는 204는 JSON decode가 `io.EOF`로 실패해 그 오류 하나를 돌려줍니다. 빈 배열 페이지는 오류 없이 끝납니다.

단건 응답은 `{"region": ...}`처럼 envelope key를 pointer로 decode합니다. 그래서 본문이 `{}`이거나 값이 null이면 오류 없이 nil을 돌려주고, envelope 값이 객체가 아니면 decode 오류입니다. ID는 escape 없이 경로 segment로 이어 붙입니다.

`With{Operation}Field` 확장 필드는 단일 envelope 안에 들어갑니다. 입력 구조체의 json tag와 같은 key이거나 이미 본문에 있는 key와 겹치면 HTTP 전에 거부되고, nil 옵션도 HTTP 전에 오류입니다. `WithListQuery`는 native query 뒤에 덧붙입니다.

## region

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST regions`, `{"region": {...}}` | 201 |
| `Get(ctx, id)` | `GET regions/{id}` | 200 |
| `List(ctx, options...)` | `GET regions?parent_region_id=` | native pager |
| `Update(ctx, id, opts, options...)` | `PATCH regions/{id}`, `{"region": {...}}` | 200 |
| `Delete(ctx, id)` | `DELETE regions/{id}` | 202, 204 |

`CreateOpts`의 빈 값은 모두 생략하므로 빈 옵션은 `{"region": {}}`를 보내고 ID는 Keystone이 정합니다. `Extra` map은 envelope 안에 펼쳐지며 같은 key의 확장 필드와 겹치면 HTTP 전에 거부됩니다. `UpdateOpts`에는 `Extra`가 없고 `Description` pointer로 빈 문자열을 보낼 수 있습니다. 응답에 `extra` 객체가 있으면 그 값이 `Extra`가 되고, 없으면 알려지지 않은 key가 `Extra`에 모입니다.

## service

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST services`, `{"service": {...}}` | 201 |
| `Get(ctx, id)` | `GET services/{id}` | 200 |
| `List(ctx, options...)` | `GET services?type=&name=` | native pager |
| `Update(ctx, id, opts, options...)` | `PATCH services/{id}`, `{"service": {...}}` | 200 |
| `Delete(ctx, id)` | `DELETE services/{id}` | 202, 204 |

`CreateOpts.Type`은 필수 검사도 omitempty도 없어서 비어 있으면 `"type": ""`를 보내고 서버가 거부합니다. `Enabled`는 pointer라 false도 보낼 수 있고, `UpdateOpts`의 `Name`·`Description`도 pointer입니다. 생성과 수정 모두 `Extra` map을 envelope 안에 펼칩니다. 응답에 `extra` 객체가 없으면 `Extra`에는 알려지지 않은 key와 함께 `name`·`description` 사본도 들어갑니다.

## endpoint

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST endpoints`, `{"endpoint": {...}}` | 201, 202 |
| `Get(ctx, id)` | `GET endpoints/{id}` | 200 |
| `List(ctx, options...)` | `GET endpoints` | native pager |
| `Update(ctx, id, opts, options...)` | `PATCH endpoints/{id}`, `{"endpoint": {...}}` | 200, 202, 204 |
| `Delete(ctx, id)` | `DELETE endpoints/{id}` | 202, 204 |

endpoint 호출은 OkCodes를 지정하지 않아 생성은 POST 기본값, 수정은 PATCH 기본값을 따릅니다. 수정이 본문 없는 204를 받으면 오류 없이 nil endpoint를 돌려줍니다. `CreateOpts`의 `Availability`(`interface`), `URL`, `ServiceID`는 필수이고 `Enabled`는 pointer입니다. 응답의 `region_id`는 `RegionID`로 decode합니다. 이 패키지에는 `Extra`가 없습니다.

`List`에는 해결하지 못한 차이가 있습니다. native `List`는 builder의 `ToEndpointListParams`를 부르지 않고 받은 값에 직접 `BuildQueryString`을 적용합니다. generated facade는 `ListOpts`를 감싼 builder를 넘기므로 `WithListOptions`의 `interface`·`service_id`·`region_id` 필터와 `WithListQuery`가 모두 조용히 사라지고 항상 전체 목록을 요청합니다. Gophercloud `endpoints.List`에 `ListOpts`를 직접 넘기면 필터가 정상으로 전송되므로, 필터가 필요하면 `RawClient()`로 native 함수를 호출해야 합니다.

## policy

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST policies`, `{"policy": {...}}` | 201 |
| `Get(ctx, id)` | `GET policies/{id}` | 200 |
| `List(ctx, options...)` | `GET policies?type=` | native pager |
| `Update(ctx, id, opts, options...)` | `PATCH policies/{id}`, `{"policy": {...}}` | 200 |
| `Delete(ctx, id)` | `DELETE policies/{id}` | 202, 204 |

`CreateOpts`의 `Type`과 `Blob`은 필수입니다. `Blob`은 nil일 때만 누락으로 보므로 빈 slice는 `"blob": ""`로 보냅니다. `Type`이 255자를 넘으면 생성과 수정 모두 HTTP 전에 `StringFieldLengthExceedsLimit` 오류입니다. 생성 본문에는 `blob`이 항상 있어 같은 key의 확장 필드가 거부되지만, 수정은 빈 `Blob`을 생략하므로 `WithUpdateField("blob", ...)`가 그대로 전송됩니다. 두 옵션 모두 `Extra` map을 envelope 안에 펼치고, 응답의 `Extra`는 region과 같은 규칙으로 채웁니다.

`ListOpts.Filters`의 key는 `type__contains`처럼 `TYPE__COMPARATOR` 모양이어야 하며, 아니면 HTTP 없이 문맥 없는 `InvalidListFilter` 오류 하나를 돌려줍니다. 올바른 필터는 그 key로 전송되지만, native query 조립이 `q:"-"` tag의 map 필드를 건너뛰지 않아 `-={'type__contains':'json'}` 같은 Python dict 모양 parameter도 함께 보냅니다.

## limit과 registered limit

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Limits.BatchCreate(ctx, opts, options...)` | `POST limits`, `{"limits": [...]}` | 201 |
| `Limits.Get(ctx, id)` | `GET limits/{id}` | 200 |
| `Limits.GetEnforcementModel(ctx)` | `GET limits/model` | 200 |
| `Limits.List(ctx, options...)` | `GET limits?region_id=&project_id=&domain_id=&service_id=&resource_name=` | native pager |
| `Limits.Update(ctx, id, opts, options...)` | `PATCH limits/{id}`, `{"limit": {...}}` | 200 |
| `Limits.Delete(ctx, id)` | `DELETE limits/{id}` | 202, 204 |
| `RegisteredLimits.BatchCreate(ctx, opts, options...)` | `POST registered_limits`, `{"registered_limits": [...]}` | 201 |
| `RegisteredLimits.Get(ctx, id)` | `GET registered_limits/{id}` | 200 |
| `RegisteredLimits.List(ctx, options...)` | `GET registered_limits?region_id=&service_id=&resource_name=` | native pager |
| `RegisteredLimits.Update(ctx, id, opts, options...)` | `PATCH registered_limits/{id}`, `{"registered_limit": {...}}` | 200 |
| `RegisteredLimits.Delete(ctx, id)` | `DELETE registered_limits/{id}` | 202, 204 |

`BatchCreateOpts`는 `CreateOpts`의 slice이고 각 행은 envelope 없이 배열에 들어갑니다. 행마다 `ServiceID`와 `ResourceName`이 필수라 하나라도 비면 HTTP 전에 `gophercloud.ErrMissingInput`입니다. limit의 `resource_limit`은 omitempty가 없어 0도 보내고, registered limit의 `default_limit`은 필수지만 숫자라 0도 통과해 항상 보냅니다. 빈 batch나 nil batch는 거부하지 않고 빈 배열을 보냅니다.

batch 확장 필드는 행 안이 아니라 최상위의 배열 옆에 들어갑니다. batch 옵션이 slice라서 json tag로 보호되는 key가 없고, 이미 본문에 있는 `limits`나 `registered_limits`와 같은 key만 HTTP 전에 거부됩니다. 행별 확장 값은 넣을 수 없습니다. 응답은 같은 배열 key로 decode하며, 그 key가 없으면 오류 없이 nil slice입니다.

수정의 `Description`, `ResourceLimit`, `DefaultLimit`은 pointer라 빈 문자열과 0을 보낼 수 있습니다. registered limit 수정은 `RegionID`, `ServiceID`, `ResourceName`도 바꿀 수 있습니다. `GetEnforcementModel`은 `{"model": {"name": ..., "description": ...}}`를 decode하며 값이 null이면 nil입니다.

## project endpoint

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, projectID, endpointID)` | `PUT OS-EP-FILTER/projects/{project}/endpoints/{endpoint}` | 204 |
| `List(ctx, projectID)` | `GET OS-EP-FILTER/projects/{project}/endpoints` | native pager |
| `Delete(ctx, projectID, endpointID)` | `DELETE OS-EP-FILTER/projects/{project}/endpoints/{endpoint}` | 204 |

endpoint filter 연결과 해제는 본문 없이 보내고 둘 다 204만 받습니다. 그래서 다른 삭제와 달리 202도 오류입니다. 목록 행은 `id`, `interface`, `region`, `service_id`, `url`만 decode하며 옵션은 없습니다.

## federation mapping

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `CreateMapping(ctx, id, opts, options...)` | `PUT OS-FEDERATION/mappings/{id}`, `{"mapping": {"rules": [...]}}` | 201 |
| `GetMapping(ctx, id)` | `GET OS-FEDERATION/mappings/{id}` | 200 |
| `ListMappings(ctx)` | `GET OS-FEDERATION/mappings` | native pager |
| `UpdateMapping(ctx, id, opts, options...)` | `PATCH OS-FEDERATION/mappings/{id}`, `{"mapping": {"rules": [...]}}` | 200 |
| `DeleteMapping(ctx, id)` | `DELETE OS-FEDERATION/mappings/{id}` | 202, 204 |

mapping 생성은 호출자가 고른 ID로 보내는 `PUT`입니다. `Rules`는 omitempty가 없어 비어 있으면 `"rules": null`을 보냅니다. 규칙의 `local`·`remote` 배열도 omitempty가 없어 비면 null이고, remote의 `type`은 빈 문자열이어도 항상 보냅니다. 나머지 규칙 필드(`regex`, `any_one_of`, `not_any_of`, `blacklist`, `whitelist`, local의 `user`·`group`·`group_ids`·`groups`·`projects`·`domain`)는 비면 생략합니다. `schema_version` 같은 다른 mapping 필드는 `WithCreateMappingField`·`WithUpdateMappingField`로 envelope 안에 넣을 수 있고, `rules` key는 HTTP 전에 거부됩니다. 응답 규칙은 같은 구조체로 decode하며 사용자 `type`은 `UserTypeEphemeral`·`UserTypeLocal` 문자열 pointer입니다. 목록에는 옵션이 없습니다.
