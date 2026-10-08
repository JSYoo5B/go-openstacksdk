# Senlin 이름/ID 자동 조회

`Profiles`·`Policies`·`Clusters`·`Nodes`·`Receivers`의 `FindIdentity`는 이름, UUID, 짧은 ID를 같은 문자열로 받습니다. SDK가 직접 GET부터 목록 검색과 중복 검사까지 처리합니다. 호출자는 builder나 resolver interface를 구현할 필요가 없습니다. 명시적인 `Find(ctx, resource.ID(...))`와 `Find(ctx, resource.Name(...))`도 사용할 수 있습니다.

| openstacksdk | Go |
|---|---|
| `conn.clustering.find_profile("worker_template")` | `service.Profiles.FindIdentity(ctx, "worker_template")` |
| `conn.clustering.find_policy("placement")` | `service.Policies.FindIdentity(ctx, "placement")` |
| `conn.clustering.find_cluster("workers")` | `service.Clusters.FindIdentity(ctx, "workers")` |
| `conn.clustering.find_node("worker_1")` | `service.Nodes.FindIdentity(ctx, "worker_1")` |
| `conn.clustering.find_receiver("scale_out")` | `service.Receivers.FindIdentity(ctx, "scale_out")` |
| `ignore_missing=False` | 해당 패키지의 `WithFindIgnoreMissing(false)` |

```python
profile = conn.clustering.find_profile("worker_template")
policy = conn.clustering.find_policy("placement")
cluster = conn.clustering.find_cluster("workers")
node = conn.clustering.find_node("worker_1")
receiver = conn.clustering.find_receiver("scale_out")
```

아래 Go 함수는 이미 인증 설정한 Connection을 사용합니다. Python `None`은 Go `nil, nil`, 논리적 미존재 오류는 `errors.Is(err, resource.ErrNotFound)`, 중복은 `resource.ErrAmbiguous`로 매핑됩니다.

```go
package example

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
)

func FindSenlinResources(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.Clustering(ctx)
    if err != nil { return err }
    profile, err := service.Profiles.FindIdentity(ctx, "worker_template")
    if err != nil { return err }
    policy, err := service.Policies.FindIdentity(ctx, "placement")
    if err != nil { return err }
    cluster, err := service.Clusters.FindIdentity(ctx, "workers")
    if err != nil { return err }
    node, err := service.Nodes.FindIdentity(ctx, "worker_1")
    if err != nil { return err }
    receiver, err := service.Receivers.FindIdentity(ctx, "scale_out")
    if err != nil { return err }
    if profile != nil { fmt.Println("profile", profile.ID) }
    if policy != nil { fmt.Println("policy", policy.ID) }
    if cluster != nil { fmt.Println("cluster", cluster.ID) }
    if node != nil { fmt.Println("node", node.ID) }
    if receiver != nil { fmt.Println("receiver", receiver.ID) }
    return nil
}
```

## 조회 순서와 기본값

1. 문자열 형태로 UUID 여부를 추측하지 않고 `/resources/{identity}`로 GET합니다. 성공하면 컨트롤러가 반환한 canonical ID로 모델을 돌려주며 목록 요청을 보내지 않습니다.
2. 기본 `resource.FindFallbackCompatible`에서는 GET의 400·403·404만 목록 검색으로 전환합니다. 목록에는 literal `name=identity`를 보내며 서버가 반환한 각 행의 정확한 `id` 또는 `name`을 로컬에서 비교합니다. 대소문자 변환·부분 일치·정규식은 적용하지 않습니다.
3. 한 후보를 찾은 뒤에도 모든 advertised next link를 검사합니다. 두 번째 일치 행은 같은 ID여도 중복 오류입니다. 후속 페이지 오류·취소·잘못된 응답이 발생하면 앞의 후보를 성공으로 반환하지 않습니다. 두 번째 후보가 나오면 즉시 중복 오류를 반환합니다.
4. 목록 검색을 마쳐도 일치 행이 없으면 기본값은 `nil, nil`입니다. `WithFindIgnoreMissing(false)`이면 `ErrNotFound`입니다. 억제했던 GET의 400·403은 최종 논리적 미존재 오류의 원인으로 붙이지 않습니다.

Find는 limit·marker·cap·single-page 값을 만들지 않습니다. 서버가 표시한 next link를 따라가며, next link 없는 응답은 추가 marker 요청 없이 끝납니다. 이 다섯 Python proxy의 직접 선언은 문자열과 `ignore_missing`만 받습니다. inherited `Resource.find`의 임의 query·URI·base_path·all_projects 인자를 이 proxy 계약에 포함시키지 않습니다.

| fallback 정책 | GET 400·403 | GET 404 |
|---|---|---|
| `FindFallbackCompatible` (기본) | 목록 검색 | 목록 검색 |
| `FindFallbackNotFoundOnly` | 원래 HTTP 오류 | 목록 검색 |
| `FindFallbackNever` | 원래 HTTP 오류 | 목록 없이 미존재 처리 |

401·409·5xx·전송 실패·성공 응답 decode 실패는 fallback 대상이 아닙니다. 목록에서 발생한 HTTP 오류는 403·404도 그대로 반환하며 ignore-missing으로 숨기지 않습니다. `FindFallbackNever`의 strict GET404는 원래 GET 오류를 원인으로 보존합니다. SDK는 Find 전용 재시도나 캐시를 만들지 않으며 ProviderClient의 기존 인증 갱신 정책을 사용합니다.

## 호출별 옵션과 오류 처리

각 패키지는 같은 `FindOpts`/`FindOption`과 `WithFindOptions`, `WithFindIgnoreMissing`, `WithFindFallback`, `WithFindHeader`, `WithFindMicroversion`을 제공합니다. `FindOpts.IgnoreMissing`의 nil은 기본 true이며 명시 false는 strict입니다. `WithFindOptions`는 pointer를 포함한 입력을 생성 시 복사하고 재사용마다 새 복사본을 만듭니다. 뒤 옵션이 앞 옵션을 덮어씁니다.

```go
package example

import (
    "context"
    "errors"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindClusterStrict(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.Clustering(ctx)
    if err != nil { return err }
    value, err := service.Clusters.FindIdentity(ctx, "workers",
        clusters.WithFindIgnoreMissing(false),
        clusters.WithFindFallback(resource.FindFallbackNotFoundOnly),
        clusters.WithFindHeader("X-Vendor-Trace", "lookup"),
        clusters.WithFindMicroversion("1.7"))
    if errors.Is(err, resource.ErrNotFound) { return fmt.Errorf("workers cluster: %w", err) }
    if err != nil { return err }
    fmt.Println(value.ID, value.Name, value.Header, value.StatusCode)
    return nil
}
```

호출별 header·microversion은 GET과 fallback의 모든 페이지에 같은 선택을 적용합니다. 숫자 버전과 clustering service type·원본 version header를 검사하며 공유 ServiceClient를 변경하지 않습니다. `WithFindMicroversion("")`는 effective 1.0을 선택하고 버전 header를 생략합니다. source의 충돌한 version header는 재설정으로 숨기지 않습니다. 호출별 header나 microversion을 하나라도 선택하면 source에서 상속한 설정도 그 호출에 고정합니다. 둘 다 생략하면 기존 client의 live header/version 선택을 유지합니다. 원본 source는 요청 사이에도 재검사하며 ProviderClient의 현재 token·reauth는 공유합니다. [목록 호출 설정](../listing/README.md#목록-호출별-헤더와-버전)의 동일한 header 우선순위·소유권 규칙을 사용합니다.

auth·version·transport header는 보호되며 전용 microversion 옵션을 사용합니다. Find의 body field·arbitrary query·foreign argument는 지원하지 않아 요청 전에 거부합니다. 원본 Get/Find/List/Wait와 다른 호출은 이 선택으로 변경하지 않습니다.

## Go 매핑과 응답 식별자

FindIdentity의 문자열은 단일 unescaped route segment여야 합니다. 빈 값·공백·`.`·`..`·`/`·`\`·`?`·`#`·`%`를 거부합니다. 이 route 정책에 맞지 않는 이름을 목록만으로 검색하려면 기존 `Find(ctx, resource.Name(name))`를 사용합니다. ID를 미리 알고 있을 때의 `Find(ctx, resource.ID(id))`도 GET 실패 후 목록으로 전환하지 않습니다. 두 explicit Ref convenience는 기본 미존재 무시이며 공통 `Resources.Find`는 strict 기본값을 유지합니다.

응답의 정확한 lowercase `id`는 nonempty 안전한 string이어야 하며 모든 소비된 fallback 행에서 검사합니다. `name`은 string 또는 null/생략이며 null/생략은 typed 모델에서 빈 이름이 됩니다. `ID`·`Name` 같은 case-variant extension이 canonical typed 식별자를 덮어쓰지 않습니다. 원문 Body에서는 case-variant·unknown 필드와 정확한 숫자를 보존합니다. 비호환 extension 타입으로 typed decoder가 실패하면 오류를 반환하며 성공으로 바꾸지 않습니다. strict envelope·응답 식별자 검증 실패도 `ResponseError`의 body/header/status 근거를 보존합니다.

고정 Python `Resource.find`는 per-call microversion을 GET에만 전달합니다. Go의 전용 옵션은 두 단계에 일관되게 적용합니다. Python의 mutable Resource/cache mutation·permissive envelope·향후 기본값 변경 경고는 Go의 concrete 모델/context/error 정책으로 매핑합니다. 이 문서는 다섯 직접 선언 Senlin find proxy를 설명하며 다른 서비스의 자동 Find 전체 지원을 뜻하지 않습니다.

소스 근거: [고정 Senlin proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py), [고정 Resource.find와 match](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2439). 구현은 [공통 helper](../../../internal/senlin/find_identity.go), 검증은 [공통 helper 테스트](../../../internal/senlin/find_identity_test.go)와 [5개 facade HTTP 테스트](../../../api/clustering_find_identity_test.go)를 사용합니다.
