# Senlin cluster policy bindings

`clusterpolicies.New(client).InCluster(ctx, ref)`는 부모 cluster를 해석한 고정 scope를 만들고,
그 안에서 policy binding을 조회합니다. Binding 조회에 사용하는 identity는 **policy의
이름·short ID·UUID**입니다. 응답의 binding UUID인 `ID`와 정책 UUID인 `PolicyID`를 구분합니다.
정책 연결·해제·설정 변경은 `clusters.API.AttachPolicy`, `DetachPolicy`, `UpdatePolicy`에서 합니다.

| openstacksdk | Go | HTTP 계약 |
|---|---|---|
| `get_cluster_policy(policy, cluster)` | `scope.Get(ctx, policyIdentity)` | GET `/clusters/{cluster}/policies/{policy}`, 200 + `cluster_policy` |
| `cluster_policies(cluster, **query)` | `scope.List` / `All` | GET `/clusters/{cluster}/policies`, 200 + `cluster_policies` |
| 부모를 URI에 직접 지정 | `InCluster(ctx, ref)` | 부모 GET 또는 정확한 Name 목록 검색 후 canonical ID 고정 |
| generic `wait_for_status` | `scope.WaitForStatus` | 기본 status 필드가 없어 HTTP 이전 unsupported |
| generic `wait_for_delete` | `scope.WaitForDelete` | 동일 scope GET에서 404를 기다림 |

## 조회

```python
binding = conn.clustering.get_cluster_policy("POLICY_NAME_OR_ID", "CLUSTER_NAME_OR_ID")
for binding in conn.clustering.cluster_policies("CLUSTER_NAME_OR_ID", is_enabled=False):
    print(binding.id, binding.policy_id)
```

```go
api := clusterpolicies.New(client)
scope, err := api.InCluster(ctx, resource.ID("CLUSTER_NAME_OR_ID"))
if err != nil { return err }
binding, err := scope.Get(ctx, "POLICY_NAME_OR_ID")
if err != nil { return err }
fmt.Println(binding.ID, binding.PolicyID, binding.ClusterID, binding.URIClusterID)

for binding, err := range scope.List(ctx, clusterpolicies.WithListEnabled(false)) {
    if err != nil { return err }
    fmt.Println(binding.PolicyName, binding.IsEnabled)
}
```

`resource.ID("cluster-name")`는 이름도 controller identity로 직접 GET합니다.
`resource.Name("cluster-name")`는 정확히 일치하는 이름을 목록에서 해석하고 중복을 거부합니다.
Scope 생성은 Python의 URI 직접 대입보다 부모 요청을 추가하며, Name 조회는 목록이 여러 페이지면
계속 검색합니다. 부모가 없거나 응답 Body ID가 누락·null·잘못된 값이면 scope를 만들지 않습니다.
Scope가 생긴 뒤 부모를 다시 해석하지 않으며, 반환 모델을 수정해도 URI parent는 바뀌지 않습니다.

각 응답은 별도의 `ID`, `PolicyID`, `ClusterID` 문자열을 요구합니다. `ClusterID`는 scope의 canonical
부모와 같아야 하고 `URIClusterID`는 조회에 사용한 고정 부모입니다. `ID == PolicyID`를 요구하거나
binding ID를 policy route에 자동으로 사용하지 않습니다. `Resources`의 ID callback과 Name lookup은
`PolicyID`를 controller identity로 사용합니다. `Get(ctx, "policy-name")`은 이름을 직접 전달하고
별도의 policy GET/Find를 수행하지 않습니다.

모델은 ClusterName/PolicyName/PolicyType/IsEnabled/Data와 raw Body/Header/StatusCode를 제공합니다.
Data의 숫자는 float64로 변환하지 않으며, Body는 unknown 필드와 null·생략 차이를 보존합니다.
잘못된 200 envelope나 응답 ID, 다른 부모의 행은 전체 응답 body/header/status를 가진
`resource.ResponseError`로 반환합니다. List에서는 잘못된 행이 포함된 원래 페이지 전체가 증거입니다.
HTTP 403/404 등 원래 오류와 context 원인을 유지하고, Location을 따라가거나 GET을 재전송하지 않습니다.

## 목록 옵션과 paging 범위

```go
bindings, err := scope.All(ctx,
    clusterpolicies.WithListEnabled(false),
    clusterpolicies.WithListPolicyName("POLICY_NAME"),
    clusterpolicies.WithListPolicyType("senlin.policy.scaling-1.0"))
if err != nil { return err }
fmt.Println(len(bindings))
```

Published 옵션은 Enabled/PolicyName/PolicyType/Sort입니다. `Enabled == nil`은 생략하고 false는
`enabled=false`로 보냅니다. `WithListOptions`는 pointer 입력까지 생성 시 snapshot을 만들고
재사용마다 독립적으로 적용합니다. `WithListEnabled`는 매 적용 시 새 pointer를 만듭니다.
Sort는 `key[:asc|desc]`의 comma grammar를 검사하며 키의 허용 여부는 서버가 판정합니다.
Python의 `is_enabled` alias는 Go의 Enabled에 대응하고 wire key는 `enabled`입니다.

16.0.0 서버 controller의 목록 whitelist에는 enabled/policy_name/policy_type/sort 네 query만 있습니다.
초기 limit/marker는 `Resources.List`에서도 `ErrUnsupported`이며, binding ID나 policy ID를 marker로
추측해 다음 요청을 생성하지 않습니다. Stock endpoint는 paging을 약속하지 않습니다.
서버가 실제로 제공한 body/HTTP next link만 따라가며, 그 링크는 같은 origin·부모 collection에
머물고 초기 filter를 유지해야 합니다. custom deployment가 제공한 링크의 marker는 허용합니다.
`break`는 다음 페이지를 조회하지 않고, List는 lazy이며 재순회할 때 독립적인 요청을 시작합니다.
모든 페이지에서 source와 선택 microversion을 다시 검사합니다. 이 API 자체에 추가 minimum gate를
추측해서 적용하지 않습니다.

`WithListQuery`는 custom deployment query의 명시적 opt-in입니다. Stock Senlin은 unknown query를
400으로 거부하므로 vendor query 지원을 보장하지 않습니다. concrete 옵션, fixed parent/응답 ID,
SDK alias와 initial pagination key는 확장 query로 덮어쓸 수 없습니다. List의 body Fields는 지원하지 않습니다. `WithListHeader`와 `WithListMicroversion`으로
목록 호출의 헤더·숫자 버전을 지정하고 source client의 인증·endpoint prefix는 공유합니다.
Secondary Arguments는 SDK 소유 로컬 Body 필터와 목록 microversion만 지원합니다.

## 대기와 Python Resource 차이

```go
err = scope.WaitForDelete(ctx, resource.ID("POLICY_NAME_OR_ID"),
    resource.WithTimeout(30*time.Second),
    resource.WithPollInterval(time.Second))
if err != nil { return err }
```

`WaitForDelete`는 연결 해제를 시작하지 않고 기본 120초 동안 GET에서 absence를 기다립니다.
직접 identity는 응답 binding ID나 PolicyID로 바꾸지 않고 같은 policy route를 계속 조회합니다.
Name Ref를 사용하면 scope 목록에서 한 번 해석한 PolicyID를 사용합니다. 권한 오류나 잘못된 성공
응답은 absence로 숨기지 않습니다. `WaitForStatus`는 기본 status가 없어 unsupported이고, 필요한
경우 caller가 `WithStatusAttribute`로 exported 문자열 속성을 명시할 수 있습니다.

Get은 stateless 응답 snapshot으로 HTTP 계약을 매핑합니다. Python Resource instance를 재사용하여
cache/URI 속성을 갱신하는 방식 대신 고정 scope와 독립된 모델을 반환합니다. Scope의 부모 확인
요청과 필수 canonical response ID/parent 검증은 명시적인 Go 정책입니다.

List는 partial입니다. `paginated=False`와 `max_items`의 raw 행 소비 제어는 제공하지만
per-call base path와 deprecated JMESPath filter는 노출하지 않습니다.
Python의 자동 query/Body 분류도 적용하지 않습니다. Python 공통 query의 초기 limit/marker는
stock controller의 whitelist에 없어 Go가 지원하지 않습니다. Resource.id fallback 대신 raw Body
필터를 사용하며, unknown query를 버리는 Python과 명시적 vendor query를
전달하는 Go의 차이도 유지합니다. Create/Update/Delete 및 임의 metadata subresource route는
이 읽기 API의 기능으로 만들지 않습니다.

계약 근거: [공식 API reference](https://docs.openstack.org/api-ref/clustering/#cluster-policies),
[16.0.0 cluster policy controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/cluster_policies.py),
[16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py).
Pinned Python 비교는 `openstack/clustering/v1/cluster_policy.py:16–43`와
`openstack/clustering/v1/_proxy.py:1059–1099`에 대응합니다.

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit` / `marker` | 초기 query에서 `ErrUnsupported` | stock controller의 whitelist에 없어 wire paging을 추측하지 않음 |

```go
listingAPI := clusterpolicies.New(client)
listingScope, err := listingAPI.InCluster(ctx, resource.ID("CLUSTER_ID"))
if err != nil { return err }
values, err := listingScope.All(ctx, clusterpolicies.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingScope.List(ctx, clusterpolicies.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.PolicyID)
}
```

`MaxItems`는 로컬 cap만 적용하고 `limit` hint를 보내지 않습니다. Python Resource.list가
max_items를 wire limit으로 사용하는 동작과 다른 Go 정책입니다. 명시적 server next link의
marker는 고정 parent/필터를 유지하는 경우에만 허용합니다.

`max_items`와 `paginated`는 서버 query로 보내지 않으며 `WithListQuery`에서 같은 이름을
사용하면 사전 오류입니다. `WithListOptions`는 bool pointer도 snapshot으로 소유하고 재사용 시
독립적으로 적용합니다. cap에 도달하면 뒤의 행이나 next link를 처리하지 않지만 소비한 행의
잘못된 JSON·검증 오류는 전체 페이지 증거와 함께 반환합니다. 빈 페이지에서는 next link가
있어도 끝냅니다. `break`, context와 매 페이지의 source/version 검사도 유지합니다.
Pinned Python은 정확한 page 경계에서 cap 검사를 다음 raw 행까지 미뤄 continuation GET을
한 번 더 할 수 있지만 Go는 cap 직후 끝냅니다.

공통 소비 정책과 남은 차이는 [Senlin 목록 제어](../listing/README.md), 실제 HTTP 근거는 [목록 제어 테스트](../../../api/clustering_catalog_list_controls_test.go)를 참고합니다.

## scope의 Body 필터

Python `cluster_policies(cluster, data={"team": "infra"})`의 로컬 비교는 Go에서 아래처럼
`clusterpolicies.WithListFilter`로 명시합니다. 지원 canonical 필드는
`id/name/policy_id/policy_name/cluster_name/policy_type/enabled/data`이며 `is_enabled` alias는
raw `enabled`를 비교합니다. query-capable `policy_name/policy_type/enabled`도 이 옵션을 쓰면
로컬에서만 비교합니다. 기존 `WithListEnabled/PolicyName/PolicyType`은 server query 입력입니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func FilterClusterPolicies(ctx context.Context, conn *sdk.Connection, cluster resource.Ref) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	scope, err := service.ClusterPolicies.InCluster(ctx, cluster)
	if err != nil {
		return err
	}
	for binding, err := range scope.List(ctx,
		clusterpolicies.WithListFilter("is_enabled", false),
		clusterpolicies.WithListFilter("data", map[string]any{"team": "infra"}),
		clusterpolicies.WithListMaxItems(50), clusterpolicies.WithListPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(binding.ID, binding.PolicyID, binding.URIClusterID)
	}
	return nil
}
```

`cluster_id`는 pinned URI 속성이므로 Body 필터로 받지 않습니다. Go 추가 `URIClusterID`나
unknown 필드도 거부합니다. 필터는 별도 policy GET 없이 이미 받은 raw Body를 비교하며
binding ID를 PolicyID로 대체하지 않습니다. 필수 `id/policy_id/cluster_id`와 고정 부모 검증은
필터보다 먼저 수행하므로, 필터에 불일치할 행이라도 필수 ID가 없거나 부모가 다르면
`resource.ResponseError`입니다. cap도 일치 결과 수가 아니라 검증한 원래 응답 행을 셉니다.

입력은 옵션 생성 시 JSON snapshot으로 소유합니다. 객체는 recursive subset, 배열은 순서와
전체 값, 숫자는 정확한 decimal 값으로 비교하고 bool과 구분합니다. scalar null은 생략과 같고
빈 실제 객체는 객체 필터와 매칭하지 않습니다. 기존 initial limit/marker unsupported·no hint·
source recheck와 continuation guard는 그대로입니다. [공통 정책](../listing/README.md)과
[Body 필터 HTTP 계약](../../../api/clustering_body_filters_test.go)을 참고합니다.

## 목록 호출별 헤더와 버전

`WithListHeader(key, value)`와 `WithListMicroversion("1.7")`은 이 패키지의 typed `List` /
`All`에만 적용합니다. 기본값은 source client 설정이며 명시 옵션은 공유 클라이언트를
수정하지 않습니다. 실제 wire 헤더·버전 선택, 재순회·페이지·인증 정책과 Python 비교 예제는
[Senlin 목록 호출 옵션](../listing/README.md#목록-호출별-헤더와-버전)을 참고합니다.
`headers`, `microversion`, `base_path`를 `WithListQuery`로 전달하면 HTTP 전에 오류입니다.
