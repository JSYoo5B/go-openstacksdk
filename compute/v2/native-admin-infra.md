# Nova native 인프라 관리자 호출

`hypervisors`, `aggregates`, `services`, `migrations`, `diagnostics`, `availabilityzones`, `usage` 패키지의 generated 메서드는 Gophercloud `v2.15.0`의 compute/v2 요청을 바꾸지 않고 호출합니다. 여기서 설명하는 호출은 모두 기본 Nova 정책에서 관리자에게만 허용됩니다. SDK는 단건 호출의 오류에 `resource.OperationError{Resource: "<패키지 이름>"}` 문맥만 더하고, 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. 목록 stream의 HTTP·decode 오류에는 operation 문맥이 붙지 않으며, nil 옵션 같은 입력 오류만 문맥과 함께 HTTP 전에 반환됩니다. ID는 escape 없이 경로 segment로 이어 붙입니다.

Nova microversion은 호출자가 client에 지정한 값을 그대로 보내며 SDK가 고르지 않습니다. 응답 모양이 버전마다 다른 필드는 아래처럼 native decoder가 여러 형태를 받아 줍니다. 일반 사용자용 availability zone 목록과 단일 프로젝트 사용량은 [Availability Zone 일반 목록·이름 조회](../availability-zones.md)와 [사용량](usage/README.md)에 따로 설명합니다.

## Hypervisor

[요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/hypervisors/requests.go)·[결과](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/hypervisors/results.go)

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `List(ctx, options...)` | `GET os-hypervisors/detail` | native pager 200, 204, 300 |
| `Get(ctx, id)` | `GET os-hypervisors/{id}` | 200 |
| `GetExt(ctx, id, options...)` | `GET os-hypervisors/{id}?with_servers=...` | 200 |
| `GetStatistics(ctx)` | `GET os-hypervisors/statistics` | 200 |
| `GetUptime(ctx, id)` | `GET os-hypervisors/{id}/uptime` | 200 |

`ListOpts`의 `Limit`, `Marker`, `HypervisorHostnamePattern`, `WithServers`는 모두 pointer라 nil일 때만 생략하고, false나 0을 가리키면 그대로 보냅니다. `WithListQuery`와 `WithGetExtQuery`는 native query 뒤에 덧붙고 같은 key가 있으면 그 값을 바꿉니다. 목록은 항상 상세 경로를 쓰며 `hypervisors_links`의 next href를 따라갑니다. `Get`은 query 없는 `GetExt`와 같은 요청입니다.

`id`는 2.53 이전의 정수와 이후의 UUID 문자열을 모두 받아 문자열로 바꾸고, 내장 `service.id`도 같습니다. `cpu_info`는 2.28 이전의 JSON 문자열, 2.28부터 2.87까지의 객체를 모두 `CPUInfo`로 풀며, 2.88부터 빠진 `cpu_info`·`free_disk_gb`·`local_gb`는 0으로 남습니다. `hypervisor_version`처럼 지수 표기로 올 수 있는 숫자는 정수로 바꿉니다. 반면 `id`나 `hypervisor_version`이 없거나 null이면 decode 오류입니다. 그래서 `{"hypervisor": {}}`나 `{"hypervisor": null}`은 오류지만 envelope 자체가 없는 `{}`는 오류 없이 zero 값입니다. `service` 객체가 있는데 `id`가 없을 때도 오류입니다. 목록의 행 하나가 이렇게 실패하면 그 페이지의 어떤 행도 내보내지 않습니다. `servers`는 2.53 이상에서 `with_servers=true`일 때만 오므로 없으면 nil pointer입니다. `GetUptime`의 응답은 `hypervisor` key 아래 있고 `id`가 필수입니다.

## Host aggregate

[요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/aggregates/requests.go)·[결과](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/aggregates/results.go)

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `List(ctx)` | `GET os-aggregates` | native pager 200, 204, 300 |
| `Create(ctx, opts, options...)` | `POST os-aggregates`, `{"aggregate": {...}}` | 200 |
| `Get(ctx, id)` | `GET os-aggregates/{id}` | 200 |
| `Update(ctx, id, opts, options...)` | `PUT os-aggregates/{id}`, `{"aggregate": {...}}` | 200 |
| `Delete(ctx, id)` | `DELETE os-aggregates/{id}` | 200 |
| `AddHost(ctx, id, opts, options...)` | `POST os-aggregates/{id}/action`, `{"add_host": {"host": ...}}` | 200 |
| `RemoveHost(ctx, id, opts, options...)` | `POST os-aggregates/{id}/action`, `{"remove_host": {"host": ...}}` | 200 |
| `SetMetadata(ctx, id, opts, options...)` | `POST os-aggregates/{id}/action`, `{"set_metadata": {"metadata": {...}}}` | 200 |

aggregate ID는 `int`이고 경로에는 10진수 문자열로 들어갑니다. 생성과 action뿐 아니라 `Delete`도 200만 성공으로 보며 202와 204는 오류입니다. `CreateOpts.Name`은 필수라 비어 있으면 HTTP 전에 거부되고, `AvailabilityZone`과 `UpdateOpts`의 두 필드는 비면 생략합니다. 그래서 빈 `UpdateOpts`는 `{"aggregate": {}}`를 보냅니다. `WithCreateField`와 `WithUpdateField`의 확장 필드는 `aggregate` 안에 들어가며, 생략된 `availability_zone`을 포함해 기존 key와 겹치면 HTTP 전에 거부됩니다. 세 action은 native builder interface가 없어 확장 필드를 받지 않습니다. `Host`가 비거나 `Metadata`가 nil이면 HTTP 전에 오류입니다. metadata의 nil 값은 `null`로 보내 Nova에서 그 key를 지웁니다.

응답의 `aggregate`는 pointer로 decode하므로 key가 없거나 null이면 nil과 nil 오류를 돌려줍니다. 생성·수정·삭제 시각은 시간대 없는 `2006-01-02T15:04:05.999999` 형식만 받아서 끝에 `Z`가 붙으면 decode 오류이고, null이나 빈 문자열은 zero time입니다. `metadata` 값은 문자열이어야 하며 2.41부터 오는 `uuid`는 없으면 빈 문자열입니다. 목록은 단일 페이지라 응답에 link가 있어도 다시 요청하지 않습니다.

## Compute service

[요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/services/requests.go)·[결과](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/services/results.go)

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `List(ctx, options...)` | `GET os-services?binary=...&host=...` | native pager 200, 204, 300 |
| `Update(ctx, id, opts, options...)` | `PUT os-services/{id}`, envelope 없는 본문 | 200 |
| `Delete(ctx, id)` | `DELETE os-services/{id}` | 204 |

`Update` 본문은 envelope 없이 `status`, `disabled_reason`, `forced_down`을 최상위에 두고, 확장 필드도 최상위에 붙습니다. 세 필드는 모두 omitempty라 `ForcedDown: false`를 보낼 수 없으므로 이 native 호출로는 강제 down을 해제할 수 없습니다. 생략된 core key와 같은 이름의 확장 필드는 HTTP 전에 거부되므로 `WithUpdateField("forced_down", false)`도 쓸 수 없습니다. `Delete`는 204만 성공으로 보며 200과 202는 오류입니다. ID 경로는 2.53 이후 UUID, 그 이전은 정수 ID 문자열을 그대로 씁니다.

service의 `id`는 정수와 문자열 모두 문자열로 바꿉니다. `Update` 응답에서 `service` key가 없으면 zero 값이지만, 객체가 있거나 null인데 `id`가 없으면 decode 오류입니다. `updated_at`은 시간대 없는 형식만 받습니다. 목록은 단일 페이지이며 `id`가 없는 행이 하나라도 있으면 페이지 전체가 실패합니다.

## Migration 목록

[요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/migrations/request.go)·[결과](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/migrations/results.go)

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `List(ctx, options...)` | `GET os-migrations` | native pager 200, 204, 300 |

`ListOpts`의 필드는 모두 pointer라 nil일 때만 생략하며 `Limit`이 0을 가리켜도 `limit=0`을 보냅니다. `ChangesSince`와 `ChangesBefore`는 호출자의 시간대를 유지한 RFC3339 문자열(`changes-since`, `changes-before`)이 됩니다. 2.59부터 Nova가 `migrations_links`를 주지만 native page는 단일 페이지라 다음 페이지를 요청하지 않습니다. 따라서 `Limit`을 쓰면 첫 페이지만 받으므로 `Marker`로 직접 이어 요청해야 합니다. `id`는 `int64`이고 `created_at`·`updated_at`은 시간대 없는 형식만 받습니다.

## 서버 진단

[요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/diagnostics/requests.go)

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Get(ctx, serverID)` | `GET servers/{id}/diagnostics` | 200 |

native 호출은 OkCodes를 정하지 않아 GET 기본값 200만 받습니다. 응답은 2.48 이전의 driver별 key와 2.48 이후의 정규화된 key를 구분하지 않고 `map[string]any`로 돌려주며 숫자는 `float64`입니다. `null` 본문은 nil map이고 객체가 아닌 JSON은 decode 오류입니다.

## Availability zone 상세 목록

[요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/availabilityzones/requests.go)·[결과](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/availabilityzones/results.go)

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `ListDetail(ctx)` | `GET os-availability-zone/detail` | native pager 200, 204, 300 |

이 page는 `SinglePageBase`의 배열 전용 `IsEmpty`를 그대로 물려받아 객체 응답을 일반 page 순회로 읽으면 오류가 납니다. 그래서 facade는 native `AllPages`처럼 첫 페이지 하나만 읽는 `resource.SinglePageStream`을 쓰고, 정상 객체 응답의 `availabilityZoneInfo`를 모두 decode한 뒤에 행을 내보냅니다. 응답의 link는 따라가지 않습니다. `hosts`는 host 이름, service 이름, `ServiceState` 순서의 중첩 map이고 null이면 nil입니다. `updated_at`은 시간대 없는 형식만 받으며, 행 하나라도 decode에 실패하면 어떤 행도 내보내지 않습니다. `availabilityZoneInfo`가 없거나 null이면 행 없이 끝납니다.

## 전체 프로젝트 사용량

[요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/usage/requests.go)·[결과](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/usage/results.go)

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `AllTenants(ctx, opts, options...)` | `GET os-simple-tenant-usage` | native pager 200, 204, 300 |

`Detailed: true`는 `detailed=1`을 보내고 false는 생략합니다. `Start`와 `End`는 호출자의 시간대 offset을 버린 벽시계 시각(`2006-01-02T15:04:05`)으로 보내므로 UTC 값을 넘겨야 의도한 구간이 됩니다. `Limit`과 `Marker`는 0이나 빈 값이면 생략합니다. 목록은 `tenant_usages_links`의 next href를 따라가고, 각 행은 프로젝트별 `TenantUsage`입니다. `start`·`stop`과 server usage의 `started_at`·`ended_at`은 시간대 없는 형식만 받고 null은 zero time입니다.

## 공통 목록 동작

목록 호출은 lazy stream이라 순회를 시작해야 요청을 보냅니다. native pager는 200, 204, 300을 성공으로 보며 404 같은 status는 operation 문맥 없이 `gophercloud.ErrUnexpectedResponseCode{Expected: [200 204 300]}`로 나옵니다. 본문 없는 204는 JSON 본문을 읽다가 `io.EOF` 오류가 됩니다. 빈 배열이나 key가 없는 응답은 오류 없이 행 없이 끝납니다.
