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

Python의 per-call base_path/microversion/headers와 Resource cache·URI overlay·query 소비
전체까지 이 목록 제어의 구현으로 완료했다고 판정하지 않습니다. 비교 근거는
[pinned Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2358), Go의
[공통 리소스 정책](../../../resource/README.md)과
[REST 제어 계약 테스트](../../../internal/rest/list_control_test.go)입니다.
