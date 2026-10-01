# Senlin 목록 제어

`WithListMaxItems`와 `WithListPaginated`는 서비스별 typed `List` / `All`의 로컬 순회를
제어합니다. 공통 `Resources.List` / `All`에는 `resource.WithMaxItems`와
`resource.WithPaginated`를 사용합니다. 애플리케이션이 pager나 builder를 구현할 필요가 없습니다.

| Python Resource.list | typed Go 옵션 | 공통 Collection 옵션 |
|---|---|---|
| `max_items=50` | `WithListMaxItems(50)` | `WithMaxItems(50)` |
| `max_items=None` 또는 `0` | `WithListMaxItems(0)` | `WithMaxItems(0)` |
| `paginated=False` | `WithListPaginated(false)` | `WithPaginated(false)` |
| `paginated=True` | `WithListPaginated(true)` | `WithPaginated(true)` |
| wire `limit=20` | `ListOpts{Limit: 20}` | `WithPageSize(20)` |

MaxItems는 서버 응답에서 성공적으로 decode·검증한 행의 수입니다. 서버 query를 적용한
응답을 세고, 로컬 name/status/Body 필터를 적용하기 전에 제한합니다. 50행 중 로컬 필터를
통과한 행이 3개여도 50행을 소비하면 끝나며, 50개의 matching 결과를 채우려고 더 조회하지
않습니다. 0은 무제한이고 음수는 순회가 시작될 때 HTTP 전에 `ErrInvalidOption`입니다.
같은 옵션의 마지막 값이 우선하므로 뒤의 0으로 앞의 cap을 해제할 수도 있습니다.

Paginated의 기본값은 true입니다. false는 첫 페이지의 소비 가능한 행만 읽고, 이후 next
link의 해석·검증과 HTTP 요청을 하지 않습니다. MaxItems를 함께 지정하면 첫 페이지
안에서도 cap에서 멈춥니다. `List` 생성은 lazy이며 옵션 검증도 순회 시점에 수행하고,
iterator 재순회는 독립적인 카운터와 요청을 시작합니다. `break`도 추가 행·페이지를 소비하지
않습니다. 소비 중 취소된 context와 실제로 소비한 행의 오류는 그대로 반환합니다.

## 사용 예제

```python
for profile in conn.clustering.profiles(
    limit=20, max_items=50, paginated=True, metadata={"team": "infra"},
):
    print(profile.id)
```

아래 Go 코드는 `conn`이 준비된 상태에서 호출할 수 있는 함수입니다.

```go
package example

import (
    "context"
    "fmt"

    sdk "gophercloudsdk"
    "gophercloudsdk/clustering/v1/clusterpolicies"
    "gophercloudsdk/clustering/v1/profiles"
    "gophercloudsdk/resource"
)

func ListProfiles(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    for profile, err := range service.Profiles.List(ctx,
        profiles.WithListOptions(profiles.ListOpts{Limit: 20}),
        profiles.WithListMaxItems(50), profiles.WithListPaginated(true),
        profiles.WithListFilter("metadata", map[string]any{"team": "infra"})) {
        if err != nil { return err }
        fmt.Println(profile.ID)
    }
    return nil
}

func FirstProfilesPage(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    for profile, err := range service.Profiles.Resources.List(ctx,
        resource.WithMaxItems(5), resource.WithPaginated(false)) {
        if err != nil { return err }
        fmt.Println(profile.ID)
    }
    return nil
}

func FirstClusterPoliciesPage(ctx context.Context, conn *sdk.Connection,
    cluster resource.Ref) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    scope, err := service.ClusterPolicies.InCluster(ctx, cluster)
    if err != nil { return err }
    for binding, err := range scope.List(ctx,
        clusterpolicies.WithListMaxItems(5), clusterpolicies.WithListPaginated(false)) {
        if err != nil { return err }
        fmt.Println(binding.ID, binding.PolicyID)
    }
    return nil
}
```

## wire limit과 서비스 범위

Typed 제어는 Actions, Events, Profiles, Policies, Clusters, Nodes, Receivers, ProfileTypes,
PolicyTypes, Services와 고정 ClusterPolicies의 11개 목록에 있습니다. 앞의 10개는 positive
MaxItems를 지정하고 wire limit이 없으면 같은 값을 `limit` hint로 보냅니다. 이미 지정한
limit은 유지하며, 페이지 크기와 전체 raw cap은 서로 다른 값입니다. catalog endpoint가
limit을 실제로 지원하거나 요청 크기를 지킨다고 보장하는 것은 아닙니다.

ClusterPolicies는 stock 16.0.0 controller의 query whitelist에 limit이 없어 hint를 보내지
않습니다. cap과 첫 페이지 제어만 로컬에서 적용하고, 명시적인 initial limit/marker는 계속
unsupported입니다. next link를 서버가 실제로 제공한 경우에만 continuation을 사용합니다.
이 11개 Senlin 목록은 빈 페이지에서 next를 따라가지 않는 정책을 opt-in합니다.
BuildInfo singleton과 query를 버리는 collect_cluster_attrs proxy는 이 typed 제어 범위가
아닙니다. [Attribute collection](../clusterattributes/README.md)은 별도 고정 경로 계약입니다.

## Python과 공통 adapter의 경계

Pinned Python `Resource.list`도 raw 행을 로컬 필터 전에 세며, max_items가 있으면 지정되지
않은 limit을 채우고 빈 페이지 또는 paginated=False에서 종료합니다. 다만 cap을 다음 행의
처음에 검사하므로 정확히 페이지 끝에서 cap에 도달하면 continuation GET을 한 번 더 할 수
있습니다. Go의 Senlin REST iterator는 cap 직후 멈춰 후속 요청과 사용하지 않을 행·link
해석을 생략합니다. 이 차이는 요청을 줄이는 의도적인 Go 동작입니다.

공통 Collection도 같은 로컬 cap 옵션을 제공하지만 native Gophercloud pager는 한 페이지
전체를 먼저 Extract합니다. cap 뒤에 있는 malformed 행 때문에 해당 페이지 extraction이
실패할 수 있으며, 모든 binding이 Senlin의 행별 decode 중단까지 제공한다고 설명하지 않습니다.
page 경계를 노출하지 않는 custom Iterate는 cap을 적용할 수 있지만 first-page 제어를 지원하지
않으면 `WithPaginated(false)`에 `ErrUnsupported`를 반환합니다. 다른 서비스의 빈 페이지
처리나 limit hint는 해당 binding 정책을 따릅니다.

Python의 per-call base_path와 Resource cache·URI overlay·query 소비
전체까지 이 목록 제어의 구현으로 완료했다고 판정하지 않습니다. 비교 근거는
[pinned Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2358), Go의
[공통 리소스 정책](../../../resource/README.md)과
[REST 제어 계약 테스트](../../../internal/rest/list_control_test.go)입니다.

## 명시적인 raw Body 필터

Typed List/All 11개는 각 패키지의 `WithListFilter(key, value)`로 known Body 필드를 로컬에서
비교합니다. 기존 Profiles/Policies/Clusters/Nodes/Receivers의 필터에 더해
Actions/Events/Services/ProfileTypes/PolicyTypes/ClusterPolicies에도 같은 선택 경로를 제공합니다.
필터의 JSON 입력은 옵션 생성 시 snapshot으로 소유하고 재순회 시 독립적으로 적용합니다.
`Resources.List`의 공통 Name/Status 필터와는 다른 typed API이며 공통 Collection에 임의 Body
선택 옵션을 추가하지 않습니다. 애플리케이션이 필터 builder나 resource interface를 구현하지 않습니다.

Python은 `_query_mapping`에서 소비한 키를 server query로 보내고 남은 known Body 속성을
로컬 필터로 분류합니다. Go에서는 기존 concrete `ListOpts` 또는 서비스별 typed 옵션으로
서버 query를 선택하고, `WithListFilter`로 로컬 비교를 명시합니다. query-capable 키에도 로컬
옵션을 쓸 수 있다는 점은 의도적인 Go 확장입니다. 두 입력을 함께 주면 server query 응답에
로컬 필터를 추가 적용합니다. known Body 필드나 그 alias를 `WithListQuery`로 보내는 것은
사전 오류이며 unknown vendor query는 기존 explicit opt-in 정책을 유지합니다. 이 기능이
stock controller에 새로운 query나 schema를 추가하지는 않습니다.

| 추가 리소스 | canonical Body 필드 | 허용 Python alias |
|---|---|---|
| Actions | id, name, target, action, cause, owner, user, project, domain, interval, start_time, end_time, timeout, status, status_reason, inputs, outputs, depends_on, depended_by, created_at, updated_at, cluster_id | target_id→target, owner_id→owner, user_id→user, project_id→project, domain_id→domain, start_at→start_time, end_at→end_time |
| Events | id, name, timestamp, oid, oname, otype, cluster_id, level, user, project, action, status, status_reason, meta_data | generated_at→timestamp, obj_id→oid, obj_name→oname, obj_type→otype, user_id→user, project_id→project |
| Services | id, name, status, state, binary, disabled_reason, host, updated_at | 없음 |
| ProfileTypes / PolicyTypes | id, name, schema, support_status | 없음 |
| ClusterPolicies | id, name, policy_id, policy_name, cluster_name, policy_type, enabled, data | is_enabled→enabled |

Alias와 canonical 키는 같은 raw 필드에 대응하며 마지막 값이 적용됩니다. unknown Body 키와
Go에만 추가한 Actions.data, Services.topic, type catalog의 version은 거부합니다. ClusterPolicies의
cluster_id는 URI 속성으로 고정하므로 Body 필터가 아니며 URIClusterID도 Go의 scope 값입니다.
각 패키지는 자신의 필터 namespace만 허용하며 body Fields나 다른 filter Arguments를
받지 않습니다. 호출별 헤더·microversion 옵션은 필터와 별도로 소비합니다.

비교는 typed 모델에 변환된 값이 아닌 실제 raw JSON을 사용합니다. Event Level 숫자 20과 문자열
`"20"`은 다르고, bool과 숫자도 다른 타입입니다. JSON 숫자는 float64로 변환하지 않고 정확한
decimal 값으로 비교하므로 `1`, `1.0`, `1e0`은 같습니다. 호출자가 먼저 float64로 반올림한 값은
복원하지 않으므로 큰 수는 `json.Number`, `json.RawMessage` 또는 정확한 정수 입력으로 지정합니다.
scalar null 필터는 생략과 JSON null 모두에 매칭합니다. `false`, `0`, 빈 문자열과 빈 배열은
null과 다르며 배열은 순서와 전체 값이 같아야 합니다.

객체 필터는 재귀적으로 subset을 비교합니다. 실제 객체가 비어 있으면 빈 `{}` 필터에도 매칭하지
않고, 실제 객체가 비어 있지 않으면 `{}` 필터가 매칭합니다. 객체 필터와 null·scalar·배열은
매칭하지 않습니다. 이는 pinned Python의 dict truthiness를 유지하되 non-object에서 Python이
발생시킬 수 있는 `.get` 오류를 불일치로 처리하는 Go 정책입니다. Python의 `True == 1`과
`False == 0`도 적용하지 않습니다. 객체 내부의 없는 scalar 키와 null은 같은 비교를 사용합니다.

검증·cap·marker 계산은 Body 필터보다 먼저 적용합니다. 모든 행이 걸러져도 소비 행 수와 마지막
raw ID는 유지하며 cap 이후에 있는 결과를 채우려고 더 조회하지 않습니다. binding의 필수
ID/parent 검증은 불일치할 행에도 수행합니다. 필터는 Python Resource.id의 alternate-ID fallback을
만들지 않습니다. 예를 들어 type catalog의 raw `id`가 없으면 name을 대신 비교하지 않으며,
binding의 필수 raw `id`가 없으면 기존 응답 검증 오류입니다.

이 구현은 로컬 비교를 제공하지만 Python의 임의 mutable Resource/cache, 자동 query 소비,
unknown query 생략, per-call base_path와 deprecated JMESPath까지 완료한
것은 아닙니다. wire query 지원·버전·plugin schema는 기존 controller 계약을 그대로 따릅니다.
실제 공개 사용 예제는 [Actions](../actions/README.md), [Events](../events/README.md),
[Services](../services/README.md), [ProfileTypes](../profiletypes/README.md),
[PolicyTypes](../policytypes/README.md), [ClusterPolicies](../clusterpolicies/README.md)에 있습니다.
비교 근거는 [pinned Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2248-L2338),
[공통 필터 구현](../../../internal/senlin/filter.go)과 [6개 HTTP 계약](../../../api/clustering_body_filters_test.go)입니다.

## 목록 호출별 헤더와 버전

Typed List/All 11개는 `WithListHeader(key, value)`와 `WithListMicroversion(value)`를
제공합니다. 옵션을 생략하면 기존 source client의 헤더와 버전을 사용합니다. 명시한 옵션은
순회마다 독립된 요청 설정으로 적용하며, 공유 client·다른 API 호출·같은 iterator의 다른
순회를 변경하지 않습니다. 같은 header 이름은 대소문자와 관계없이 마지막 옵션이 우선하고
caller 값은 같은 이름의 source header보다 우선합니다. 기본 Accept는 JSON이며 Accept의
명시 변경도 가능합니다. 기존 로컬 Body 필터·raw cap·첫 페이지·query 옵션과 함께 사용합니다.

```python
for service in conn.clustering.services(
    microversion="1.7", headers={"X-Trace-ID": "engine-list"}, paginated=False,
):
    print(service.id)
```

```go
package example

import (
    "context"
    "fmt"

    sdk "gophercloudsdk"
    "gophercloudsdk/clustering/v1/services"
)

func ListServicesWithCallOptions(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    for engine, err := range service.Services.List(ctx,
        services.WithListMicroversion("1.7"),
        services.WithListHeader("X-Trace-ID", "engine-list"),
        services.WithListPaginated(false)) {
        if err != nil { return err }
        fmt.Println(engine.ID, engine.State)
    }
    return nil
}
```

선택은 숫자 `1.N`이며 `latest`, 공백, 잘못된 형식은 순회 시작 시 거부합니다.
`WithListMicroversion("")`는 source의 선택을 상속하지 않고 서버 기본 1.0을 사용하며 버전
헤더를 생략하는 Go 옵션입니다. Services의 1.7, Receiver user query의 1.4 같은 조건은
호출에 선택한 버전을 기준으로 매 페이지 검사합니다. 자동 range discovery는 Connection이
담당하며 이 옵션은 discovery나 자동 업그레이드를 하지 않습니다.

명시 호출 설정은 모든 페이지·retry·reauth에 유지하지만 인증 token은 공유 provider의 최신
값을 사용합니다. Endpoint prefix·collection 경로·고정 부모·continuation 검증은 기존 정책을
유지합니다. 원본 service type/provider/버전 header의 유효성도 매 페이지 검사합니다. 공유
클라이언트 설정을 동시 호출 중 변경하는 용도로 이 옵션을 사용하지 않습니다.

인증·서비스 token·Authorization·Host·Cookie·Content-Type·Content-Length와
OpenStack-API-Version은 SDK가 소유하며 `WithListHeader` 또는 custom option으로 덮어쓸 수
없습니다. Python의 caller header map을 그대로 병합하는 정책과 다른 Go의 명시적 소유권
규칙입니다. 버전은 `WithListMicroversion`으로 선택합니다. source `MoreHeaders`에 명시한
버전 header가 호출 선택과 충돌하면 원래 값을 자동 제거하지 않고 사전 오류로 처리합니다.

`headers`, `microversion`, `base_path`, `jmespath_filters`, `allow_unknown_params`,
`max_items`, `paginated`는 wire vendor query가 아니므로 `WithListQuery`나 custom query
주입으로 보내면 사전 오류입니다. per-call base_path와 deprecated JMESPath는 아직 노출하지
않습니다. 공통 `Resources.List/All`에 typed 호출 옵션을 전달하지 않으며, scope를 만든 뒤의
List 옵션은 부모 해석 요청이나 다른 Get/Wait를 변경하지 않습니다.
