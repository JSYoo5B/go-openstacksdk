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
SDK alias와 initial pagination key는 확장 query로 덮어쓸 수 없습니다. List의 body Fields, secondary
Arguments와 per-request Headers는 지원하지 않습니다. Source client의 인증·endpoint prefix·선택
microversion은 일반 Senlin 요청 정책을 공유합니다.

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

List는 partial입니다. Python의 `paginated=False`, `max_items`의 local filter 이전 raw 행 수 계산,
per-call base path/microversion/header, deprecated JMESPath filter를 노출하지 않습니다. `id`,
`policy_id`, `cluster_name`, `data` 같은 Body-only local filter와 recursive subset 비교도 제공하지
않습니다. Python 공통 query의 초기 limit/marker와 Resource.id fallback은 stock controller의
whitelist에 없어서 Go가 지원하지 않으며, unknown query를 버리는 Python과 명시적 vendor query를
전달하는 Go의 차이도 유지합니다. Create/Update/Delete 및 임의 metadata subresource route는
이 읽기 API의 기능으로 만들지 않습니다.

계약 근거: [공식 API reference](https://docs.openstack.org/api-ref/clustering/#cluster-policies),
[16.0.0 cluster policy controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/cluster_policies.py),
[16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py).
Pinned Python 비교는 `openstack/clustering/v1/cluster_policy.py:16–43`와
`openstack/clustering/v1/_proxy.py:1059–1099`에 대응합니다.
