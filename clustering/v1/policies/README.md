# Senlin Policies

고정 openstacksdk의 `conn.clustering` policy 연산을 concrete 옵션과 `With...` 함수로 사용합니다. spec의 plugin properties는 JSON 객체로 지정하고 SDK가 요청과 응답 처리를 소유합니다.

| openstacksdk | Go | HTTP 계약 |
|---|---|---|
| `create_policy(name=..., spec=...)` | `Policies.Create(ctx, CreateOpts, options...)` | POST `/policies`, 201 |
| `get_policy(identity)` | `Policies.Get(ctx, identity)` | GET `/policies/{identity}`, 200 |
| `policies(**query)` | `Policies.List/All(ctx, options...)` | GET `/policies`, 200 |
| `update_policy(policy, name=...)` | `Policies.Update(...)` 또는 `Load/Track → Edit → Commit` | 객체 PATCH `/policies/{identity}`, 200; tracked는 dirty 필드만 보내고 clean이면 HTTP 없음 |
| `delete_policy(policy, ignore_missing=True)` | `Policies.Delete(ctx, ref, options...)` | DELETE, 204; 기본 미존재 무시 |
| `find_policy(identity, ignore_missing=True)` | `Policies.Find(ctx, ref, options...)` | 명시한 ID 또는 정확한 이름 검색 |
| `validate_policy(spec=...)` | `Policies.Validate(ctx, ValidateOpts, options...)` | POST `/policies/validate`, 200; 숫자 1.2 이상 |

## 생성·검증·갱신

다음 Go 예제는 `ctx context.Context`, `conn *gophercloudsdk.Connection`을 사용하는 함수 안에서 실행합니다. `policies`는 `gophercloudsdk/clustering/v1/policies`, `resource`는 `gophercloudsdk/resource`입니다.

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
policy, err := service.Policies.Create(ctx, policies.CreateOpts{Name: "scale_workers"},
    policies.WithCreateSpec(map[string]any{
        "type": "senlin.policy.scaling", "version": "1.0",
        "properties": map[string]any{
            "event": "CLUSTER_SCALE_OUT",
            "adjustment": map[string]any{"type": "CHANGE_IN_CAPACITY", "number": 1},
        },
    }))
if err != nil { return err }
fmt.Println(policy.ID, policy.Spec, policy.Header)
```

name은 ASCII 문자로 시작하고 ASCII 문자·숫자·`_`·`.`·`-`로 구성하는 255자 미만 값입니다. spec은 필수 JSON 객체이며 plugin schema는 서버가 검사합니다. 예제의 event/adjustment 형식은 [Scaling policy 1.0](https://docs.openstack.org/senlin/pike/user/policy_types/scaling.html)을 참고합니다. `WithCreateSpec`, `WithValidateSpec`, `With...Options`는 입력을 생성 시 깊게 snapshot하고 재사용할 때 새 복사본을 만듭니다. `json.Number` 또는 `json.RawMessage`로 큰 정수와 소수를 정확히 전송할 수 있습니다. spec의 nil/null/배열/scalar는 요청 전에 거부하며 빈 객체의 실제 schema 허용은 서버가 판정합니다.

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
checked, err := service.Policies.Validate(ctx, policies.ValidateOpts{},
    policies.WithValidateSpec(map[string]any{
        "type": "senlin.policy.scaling", "version": "1.0",
        "properties": map[string]any{
            "event": "CLUSTER_SCALE_OUT",
            "adjustment": map[string]any{"type": "CHANGE_IN_CAPACITY", "number": 1},
        },
    }))
if err != nil { return err }
fmt.Println(checked.Spec, checked.Body["id"])
```

Connection을 `sdk.WithMicroversion(sdk.Clustering, "1.2")` 또는 적합한 숫자 버전을 협상하는 range로 구성한 뒤 Validate를 사용합니다. 미선택·1.0·1.1·symbolic `latest`는 요청 전에 거부하며 공유 client를 자동 업그레이드하지 않습니다. 검증 응답의 null/생략 ID·타임스탬프는 원문에 보존하고 가짜 route ID를 만들지 않습니다.

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
renamed, err := service.Policies.Update(ctx, resource.Name("scale_workers"), policies.UpdateOpts{},
    policies.WithUpdateName("scale_out_workers"),
    policies.WithUpdateHeader("X-Vendor-Mode", "explicit"))
if err != nil { return err }
fmt.Println(renamed.ID, renamed.Name)
```

갱신 core 필드는 name입니다. spec/type/data/identity/소유자/타임스탬프는 core 갱신 필드가 아닙니다. body와 header는 이름 조회 전에 고정하고 정확한 이름을 한 번 해석한 ID로 객체 PATCH합니다. 명시 ID는 조회를 생략합니다. 추가 서버 field/header는 작업별 `With...Field/Header`를 사용하며 core와 SDK 소유 auth/version/transport 헤더 덮어쓰기는 거부합니다.

Go Update는 stateless 요청이며 빈 갱신을 요청 전에 거부합니다. 변경 추적은 SDK 소유 `TrackedPolicy`로 사용하며 앱 개발자가 builder나 lifecycle interface를 구현할 필요가 없습니다.

```python
policy = conn.clustering.get_policy("POLICY_ID")
policy.name = "scale_out_workers"
policy = conn.clustering.update_policy(policy)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
tracked, err := service.Policies.Load(ctx, resource.ID("POLICY_ID"))
if err != nil { return err }
if err := tracked.Edit(policies.UpdateOpts{}, policies.WithUpdateName("scale_out_workers")); err != nil {
    return err
}
policy, err := tracked.Commit(ctx)
if err != nil { return err }
fmt.Println(policy.Name, tracked.Dirty(), tracked.Response().Body)
```

기존 모델은 `service.Policies.Track(policy)`로 감쌀 수 있습니다. 같은 현재 값의 대입과 clean Commit은 HTTP를 생략하지만 `A → B → A`는 dirty로 남습니다. `RemoveName()`은 캐시에서 name을 제거하고 다음 Commit에 `name:null`을 남깁니다. null 허용은 서버 schema가 판정하며 readonly spec/type/data/소유자 등은 core 갱신 필드가 아닙니다.

성공 응답은 캐시에 필드별로 병합하고 nested 객체는 전체 교체합니다. 생략한 전송 필드도 clean으로 바뀌며 HTTP 처리 중 새 Edit은 보존합니다. `Value()`는 병합 캐시, `Response()`는 마지막 성공 응답의 필드인 독립 snapshot입니다. Track은 입력 Body를 기준으로 하고 Body 없는 수동 모델은 HTTP 증거를 만들지 않는 로컬 seed입니다. 요청 ID는 고정하며 실패는 dirty를 유지하고 accepted 응답 오류에서도 자동 재전송하지 않습니다. `Refresh`로 서버 상태를 확인할 수 있습니다. [전체 Python/Go 변경 추적 계약](../tracking/README.md)을 참고합니다.

## 목록·검색·삭제

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
for policy, err := range service.Policies.List(ctx,
    policies.WithListOptions(policies.ListOpts{Limit: 20, Sort: "created_at:desc"}),
    policies.WithListGlobalProject(false),
    policies.WithListFilter("data", map[string]any{"vendor": map[string]any{"enabled": false}})) {
    if err != nil { return err }
    fmt.Println(policy.ID, policy.Name)
}
found, err := service.Policies.Find(ctx, resource.Name("scale_out_workers"))
if err != nil { return err }
if found != nil {
    if err := service.Policies.Delete(ctx, resource.ID(found.ID)); err != nil { return err }
}
```

typed server query는 limit/marker/name/type/sort/global_project입니다. 0/빈 값과 global_project nil은 생략하며 명시 false는 전송합니다. sort는 `key[:asc|desc]` 쉼표 문법을 검증하고 실제 키 지원은 서버가 판정합니다. 기본 limit은 생략합니다. 양의 limit이 있으면 짧은 페이지를 포함한 모든 nonempty 페이지 뒤에 마지막 wire 응답 ID를 marker로 사용하고 빈 페이지에서 멈춥니다.

`WithListFilter`는 data/spec/id/name/type/project/domain/user/created_at/updated_at을 로컬에서 검사하고 project_id/domain_id/user_id 별칭을 지원합니다. 객체는 재귀 부분집합, 배열은 순서와 전체 값으로 비교하며 누락 필드는 null로 비교합니다. 빈 실제 객체는 Python dict filter처럼 객체 필터와 일치하지 않습니다. Go는 bool과 숫자를 구분하고 decimal 숫자는 float64 없이 정확하게 비교합니다. 필터는 서버 query로 보내지 않고 필터로 제외된 마지막 row도 wire marker에 사용합니다.

`WithListQuery`는 명시적인 추가 서버 query이며 core/local 필드 덮어쓰기를 거부합니다. Python unknown-query 생략과 다릅니다. body/header next 링크는 같은 collection origin/path만 허용하고 최초 필터·정렬·limit을 유지합니다. 반복 URL/marker나 범위를 바꾸는 continuation은 오류입니다. lazy iterator는 재순회할 수 있으며 break는 후속 요청을 막습니다. 사용자가 바꾼 모델 ID로 다음 marker를 만들지 않습니다.

Get은 이름·UUID·짧은 ID를 직접 컨트롤러로 전달합니다. `resource.ID(value)`는 UUID 모양을 추측하지 않고 직접 route를 지정합니다. `resource.Name(value)`는 전체 목록에서 정확히 해석하며 중복은 오류입니다. `API.Find`는 기본 미존재 무시이고 `resource.WithMissingError()`로 바꿀 수 있습니다. `Resources.Find`의 공통 기본값은 미존재 오류입니다. Python combined-string Find의 ID-first GET과 400/403/404 후 목록 fallback은 아직 별도 비교 범위입니다.

Delete는 기본 404만 무시하고 `resource.WithMissingError()`로 404도 반환합니다. 403·409 등의 오류를 숨기지 않으며 목록에서 찾은 ID가 없으면 collection 경로로 DELETE하지 않습니다. Go Delete는 error를, Python proxy는 None을 반환합니다. 상속한 Resource 삭제 lifecycle은 별도 계약입니다.

## 응답과 남은 범위

`Policy.Spec`/`Data`는 JSON 숫자를 보존하며 embedded `resource.Metadata`의 `Body`/`Header`/`StatusCode`는 unknown/null/생략 및 HTTP 증거를 보존합니다. wire project/domain/user는 각각 Go ProjectID/DomainID/UserID입니다. 단건 응답은 `policy` 객체 envelope를 요구하며 Python의 flat/empty fallback을 적용하지 않습니다. 승인된 mutation의 decode/read 오류는 `resource.ResponseError`에 원문·header·status를 남기고 mutation을 재전송하지 않습니다.

Policy update의 dirty/no-op/null 삭제/응답 병합과 reset은 [tracked lifecycle](../tracking/README.md)로 제공하며 사용법과 테스트 근거를 기준으로 Go mapping 판정합니다. readonly 보호, snapshot 반환, strict envelope와 선택 client의 버전·인증 소유권은 문서화한 Go 정책입니다. 상속한 목록의 per-call base_path/microversion/header, proxy JMESPath와 combined-string Find fallback은 남은 비교·구현 범위입니다. 리소스 갱신과 cluster의 policy 연결은 별도 API 계약이며 직접 연산 7개와 Python proxy 전체 계약 완료를 구분합니다.

근거: [고정 Policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/policy.py), [고정 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py), [공식 API](https://docs.openstack.org/api-ref/clustering/). [HTTP 계약](../../../api/clustering_policies_test.go), [응답 경계 계약](../../../api/clustering_policies_response_test.go), [tracked HTTP 회귀](../../../api/clustering_lifecycle_test.go), [지원 판정](../../../docs/sdk-support-ledger.md).

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit=n` | `WithListOptions(ListOpts{Limit: n})` | 양수는 wire page limit; 로컬 cap과 독립 |

```go
listingAPI := policies.New(client)
values, err := listingAPI.All(ctx,
    policies.WithListOptions(policies.ListOpts{Limit: 20}),
    policies.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, policies.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

명시한 wire limit이 없으면 양의 `MaxItems`를 limit hint로 보냅니다. 명시 limit은 그대로
유지하고 로컬 cap은 응답이 그 limit보다 커도 적용합니다. 반환 수는 로컬 필터나 서버의
page 정책에 따라 cap보다 적을 수 있습니다.

`max_items`와 `paginated`는 서버 query로 보내지 않으며 `WithListQuery`에서 같은 이름을
사용하면 사전 오류입니다. `WithListOptions`는 bool pointer도 snapshot으로 소유하고 재사용 시
독립적으로 적용합니다. cap에 도달하면 뒤의 행이나 next link를 처리하지 않지만 소비한 행의
잘못된 JSON·검증 오류는 전체 페이지 증거와 함께 반환합니다. 빈 페이지에서는 next link가
있어도 끝냅니다. `break`, context와 매 페이지의 source/version 검사도 유지합니다.
Pinned Python은 정확한 page 경계에서 cap 검사를 다음 raw 행까지 미뤄 continuation GET을
한 번 더 할 수 있지만 Go는 cap 직후 끝냅니다.

List 전체 계약은 partial입니다. 알려진 `WithListFilter`의 raw JSON 비교와 별도로 Python
Resource field/default/alias 정규화 및 query 소비, per-call base_path/microversion/header와
deprecated JMESPath는 계속 비교합니다. `WithListQuery`는 vendor query를 실제로 전달하는
Go 확장이며 Python unknown query 생략과 구별합니다.

공통 소비 정책과 남은 차이는 [Senlin 목록 제어](../listing/README.md), 실제 HTTP 근거는 [목록 제어 테스트](../../../api/clustering_typed_list_controls_test.go)를 참고합니다.
