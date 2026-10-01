# Nova limits singleton

`compute/v2/limits`는 native `API.Get` 위에 raw 응답을 보존하는 `API.Fetch`와 고정 프로젝트 `ProjectLimitsScope`를 제공합니다. Limits는 조회용 singleton이며 quota 변경, project 목록, generic Find·Wait 작업을 제공하지 않습니다. 기존 generated `Get`, `Limits`, `Absolute`, `GetOpts`와 옵션 계약은 유지합니다.

## 현재 응답과 고정 프로젝트

```go
api := limits.New(computeClient)
current, err := api.Fetch(ctx, limits.WithGetReserved(true))
if err != nil { return err }
fmt.Println(current.Absolute.MaxTotalCores, current.StatusCode)

scope, err := api.InProject(ctx, resource.ID("project-id"))
if err != nil { return err }
value, err := scope.Get(ctx, limits.WithGetReserved(false))
if err != nil { return err }
fmt.Println(value.ProjectID, value.Absolute.TotalCoresUsed)
```

`Fetch`는 `GET /limits`를 호출합니다. Project query를 생략하면 Nova가 인증된 프로젝트를 선택하고 결과의 `ProjectID`는 빈 문자열입니다. 응답의 `id`나 token/endpoint를 통해 프로젝트를 추측하지 않습니다. 명시적인 ID가 필요하면 `WithGetOptions(GetOpts{TenantID: "project-id"})` 또는 `InProject`를 사용합니다.

`InProject(ctx, resource.ID(...))`는 HTTP 조회 없이 project ID를 고정합니다. `resource.Name`은 `WithIdentityClient(identityClient)`로 전달한 별도의 Keystone v3 client가 필요합니다. 전체 collection에서 정확한 이름을 한 번 해석하며, 중복 이름·미존재·권한 오류를 숨기지 않습니다. UUID 모양 이름도 이름으로 처리하고 Compute client를 Keystone 요청에 재사용하지 않습니다.

`CurrentProject(ctx)`는 ProviderClient에 기록된 Keystone v3 project 또는 v2 token tenant를 읽어 scope를 고정합니다. 인증 갱신·HTTP 조회·token 또는 endpoint 추측을 하지 않습니다. 수동 token, system/domain/unscoped 인증, 누락된 인증 결과에는 `resource.ErrUnsupported`를 반환합니다. 나중에 인증 결과가 바뀌어도 scope의 project ID는 유지됩니다.

고정 scope의 `Get`은 `/limits?tenant_id={project}`를 사용합니다. `WithGetOptions`의 TenantID 또는 query의 `tenant_id`·`project_id`로 기존 scope를 대체할 수 없습니다. 빈 값·같은 ID를 포함한 identity override도 HTTP 전에 거부합니다. Scope Get은 응답 identity를 사용하거나 실패 시 current-project 요청으로 대체하지 않습니다.

## Query와 HTTP

[Nova limits API](https://docs.openstack.org/api-ref/compute/#limits-limits)의 성공 코드는 200입니다. `tenant_id`는 다른 프로젝트 조회를 위한 admin query이며, 권한은 서버 정책이 검증합니다. `CurrentProject` scope도 고정한 ID를 query로 명시합니다. 일반 current-project 조회를 원하면 `Fetch`에서 project query를 생략합니다.

| 옵션/호출 | query |
|---|---|
| Fetch(ctx) | 없음 |
| scope.Get(ctx) | `tenant_id={fixed-project}` |
| WithGetReserved(false) | `reserved=0` |
| WithGetReserved(true) | `reserved=1` |
| WithGetQuery("reserved", "-2") | native 값 `reserved=-2` 보존 |

Nova의 `reserved`는 boolean 문자열이 아닌 integer입니다. 0은 예약된 자원을 제외하고, nonzero는 포함하며, non-integer는 서버가 0처럼 처리합니다. `WithGetReserved`는 false/true를 0/1로 직렬화합니다. Raw `WithGetQuery`는 값을 변경하지 않습니다. 이 helper는 `Fetch`, scope Get, 기존 native Get에서 모두 사용합니다.

`Fetch`는 하나의 명시적인 `tenant_id`를 native opts 또는 query로 받지만 두 방식의 중복 설정과 여러 tenant 값을 거부합니다. Python의 `project_id` alias 대신 `InProject` 또는 native TenantID를 사용합니다. 그 밖의 query 확장은 서버가 검증하며 원래 값을 보존합니다. Header·body field·argument/custom builder 확장은 고수준 조회에서 받지 않습니다.

공유 fixed-request helper는 요청의 method·origin·경로·query를 redirect·retry·reauth 동안 유지합니다. 같은 대상 redirect는 원래 HTTPClient의 policy를 거칩니다. 원래 transport·HTTPClient 설정과 provider의 token·reauth·backoff·retry 동작을 보존하고 source client나 token lock을 바꾸지 않습니다. 예약 query나 tenant query를 재시도 때 중복으로 추가하지 않습니다.

## Typed 값과 raw 결과

`LimitsResource`는 embedded native `Limits` 외에 다음 정보를 제공합니다.

| 필드 | 의미 |
|---|---|
| ProjectID | 명시적으로 고정한 request project; 추측하지 않음 |
| Body | `limits` envelope 안의 모든 raw 필드 |
| AbsoluteBody | `absolute` 객체 안의 raw 필드와 확장 |
| Rate | native 모델에서 빠진 legacy rate group/rule |
| Header, StatusCode | 독립적으로 복사한 HTTP 헤더와 실제 200 status |

Typed absolute int는 null을 0으로 읽습니다. `Body`와 `AbsoluteBody`의 키 존재 여부와 raw 값으로 누락·null·0을 구분할 수 있습니다. 정수는 직접 JSON에서 decode하므로 플랫폼 int 범위의 큰 known 값과 raw 확장 값을 float64 반올림 없이 보존합니다. Microversion 때문에 사라진 필드를 SDK가 만들어내지 않습니다. Nil/empty rate slice와 raw 값은 누락·null·빈 배열을 구분합니다.

`RateLimits`는 `URI`, `Regex`, `Limits []RateLimit`, raw `Body`를 제공합니다. 각 `RateLimit`은 `Remaining`, `Value`, `Unit`, `Verb`, `NextAvailable`, raw `Body`를 보존합니다. `NextAvailable`은 `json.RawMessage`입니다. Pinned Python이 timestamp 표현의 타입을 제한하지 않으므로 숫자·문자열 표현과 큰 정수를 그대로 유지합니다.

```go
for _, group := range value.Rate {
    for _, rule := range group.Limits {
        fmt.Println(group.URI, rule.Verb, rule.Remaining,
            string(rule.NextAvailable))
    }
}
```

별개의 응답은 raw map/slice와 header를 공유하지 않습니다. Malformed JSON, 누락된/null/배열/scalar `limits` 객체, 잘못된 known absolute/rate 타입은 decode 오류입니다. `LimitsResponseError`는 성공 응답의 원본 본문·헤더·status를 보관하고 원래 JSON/read/context 오류를 unwrap합니다. Native HTTP 실패의 status·본문·헤더와 caller retry/transport 오류도 보존합니다. 404는 `resource.ErrNotFound`로 판별할 수 있습니다. Caller 취소·deadline은 header 대기, shared reauth 대기, body 읽기에도 적용합니다.

## Rate의 의미와 Python 비교

Pinned [Python limits resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/limits.py)는 absolute와 rate를 함께 모델링하지만 [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/limits/results.go)의 `Limits`에는 absolute만 있습니다. SDK-owned Fetch/scope는 rate도 유지하며, generated Get의 반환 타입은 바꾸지 않습니다.

Python [get_limits proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)는 조회 singleton입니다. Go도 fetch/read 작업만 제공합니다. Python Resource의 변경 추적·session/cache·skip_cache 동작과 alias 변환은 구현한 것으로 보지 않으며 partial로 남습니다. Pinned AbsoluteLimits의 최대 microversion 2.57을 전체 limits API의 최소 요구로 해석하거나 client microversion을 변경하지 않습니다.

Rate 값은 과거 Nova v2 응답을 읽기 위한 모델입니다. [Nova의 legacy v2 코드](https://docs.openstack.org/nova/15.0.4/stable_api.html#background)는 Newton(14.0.0)에서 제거되었으며, [현재 limits API](https://docs.openstack.org/api-ref/compute/#show-rate-and-absolute-limits)는 backward compatibility를 위한 빈 rate 배열을 반환합니다. SDK는 rate 정책을 강제로 적용하거나 legacy API를 복원하지 않습니다. 실제 서버가 legacy 데이터를 보낼 때만 그 값을 유지합니다.

검증은 [query·raw·error 계약](../../../api/nova_project_limits_contracts_test.go), [project binding·recorded auth](../../../api/nova_project_limits_projects_test.go), [redirect·retry·reauth·context](../../../api/nova_project_limits_transport_test.go) 테스트로 수행합니다.
