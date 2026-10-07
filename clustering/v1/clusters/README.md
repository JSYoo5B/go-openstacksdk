# Senlin clusters

`conn.ClusteringV1(ctx).Clusters`가 cluster의 생성·조회·목록·수정·삭제·이름 검색을 제공합니다.
Connection은 인증된 client와 선택한 numeric microversion을 공유합니다. 아래 profile_only
예제에는 `sdk.WithMicroversion(sdk.Clustering, "1.6")` 또는 그 이상의 숫자 버전을 설정합니다.

| openstacksdk | Go | HTTP 계약 |
|---|---|---|
| `create_cluster(**attrs)` | `Clusters.Create(ctx, opts, options...)` | POST `/clusters`, 201 또는 202 + cluster; 202는 action Location 필수 |
| `get_cluster(identity)` | `Clusters.Get(ctx, identity)` | GET `/clusters/{identity}`, 200 |
| `clusters(**query)` | `Clusters.List` / `All` | GET `/clusters`, 200 |
| `update_cluster(identity, **attrs)` | `Clusters.Update(...)` / `Load/Track → Edit → Commit` | 객체 PATCH, 202 + cluster와 Location |
| `delete_cluster(identity, ignore_missing=True, force_delete=False)` | `Clusters.Delete(ctx, ref, options...)` | DELETE, 202 + action Submission |
| `find_cluster(identity, ignore_missing=True)` | `Clusters.FindIdentity(ctx, identity, options...)` | GET 후 400·403·404 목록 fallback, 정확 ID 또는 이름 검색 |
| `scale_in_cluster` / `scale_out_cluster` | `ScaleIn` / `ScaleOut` | POST `/clusters/{id}/actions`, 202 |
| `resize_cluster` | `Resize` | 같은 경로, `resize` 매개변수 |
| `add_nodes_to_cluster` / `remove_nodes_from_cluster` | `AddNodes` / `RemoveNodes` | 같은 경로, `add_nodes` / `del_nodes` |
| `replace_nodes_in_cluster` | `ReplaceNodes` | 같은 경로, `replace_nodes`, 1.3 이상 |

## 생성과 조회

```python
cluster = conn.clustering.create_cluster(
    name="workers", profile_id="PROFILE_ID",
    min_size=0, max_size=-1, desired_capacity=1, timeout=None,
    metadata={"team": "infra"},
)
cluster = conn.clustering.get_cluster(cluster.id)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
cluster, err := service.Clusters.Create(ctx, clusters.CreateOpts{
    Name: "workers", ProfileID: "PROFILE_ID",
}, clusters.WithCreateMinSize(0), clusters.WithCreateMaxSize(-1),
   clusters.WithCreateDesiredCapacity(1), clusters.WithCreateTimeoutNull(),
   clusters.WithCreateMetadata(map[string]any{"team": "infra"}))
if err != nil { return err }
cluster, err = service.Clusters.Get(ctx, cluster.ID)
if err != nil { return err }
fmt.Println(cluster.ID, cluster.Status, cluster.NodeIDs)
```

생략한 크기·timeout·config는 서버 기본값을 사용합니다. 명시한 0과 max_size=-1을
생략하지 않습니다. ProfileID는 profile 이름·UUID·short-ID를 서버에 전달하며 별도 profile
GET을 하지 않습니다. name은 ASCII 문자로 시작하고 ASCII 문자·숫자·`_`·`.`·`-`를 사용하며
255자 미만입니다. plugin 구성의 실제 의미는 Senlin이 검증합니다.

`Cluster.ID`와 action ID는 별개입니다. 응답의 숫자는 `*json.Number`, config/metadata/data와
dependents는 raw JSON 값으로 보존합니다. 사용자 metadata는 `UserMetadata`, HTTP와 원래
필드는 `Header`, `StatusCode`, `Body`에 있습니다. Create는 공개 API reference의 201과
Senlin 16.0.0 router의 202를 모두 허용합니다. 202에서는 유효한 action Location이 필수이며
`Operation`에 action 참조와 원문 응답·header·실제 status를 보존합니다. 201은 Location을
생략할 수 있고 그때 Operation은 nil입니다. 201에 Location이 있으면 같은 action URI 검증을
적용합니다. Create는 action을 자동 조회하거나 완료를 기다리지 않으며, 잘못된 accepted
응답은 증거를 보존한 `resource.ResponseError`로 반환하고 재전송하지 않습니다.
[16.0.0 source audit](../../../docs/senlin-server-contracts.md)은 공개 문서와 release 코드의
차이를 기록하며 실제 cloud 검증과 구분합니다.

## 수정과 null

```python
cluster = conn.clustering.update_cluster(
    "workers", name="workers_v2", timeout=None, metadata={}, profile_only=False,
)
```

```go
api := clusters.New(client)
cluster, err := api.Update(ctx, resource.Name("workers"), clusters.UpdateOpts{},
    clusters.WithUpdateName("workers_v2"), clusters.WithUpdateTimeoutNull(),
    clusters.WithUpdateMetadata(map[string]any{}), clusters.WithUpdateProfileOnly(false))
if err != nil { return err }
fmt.Println(cluster.ID, cluster.Operation.ActionID)
```

Name/ProfileID/Timeout의 zero Optional은 생략, `request.Null[T]()`은 JSON null,
`request.Present(value)`는 명시 값입니다. Config/Metadata의 nil RawMessage는 생략하고
raw null과 `{}`는 별도 값입니다. `With...Config/Metadata(nil)`도 명시 null을 만듭니다.
null의 실제 갱신 의미는 필드별 서버 계약을 따릅니다. ProfileOnly는 false를 포함해
명시하면 1.6 이상이 필요합니다. 크기 변경은 별도 resize 작업의 계약으로 다룹니다.
위 예제처럼 `WithCreateMinSize`, `WithUpdateTimeout`, `WithUpdateTimeoutNull` 등을 사용하면
호출자가 포인터나 Optional을 직접 만들 필요가 없습니다.

옵션이 body와 header를 준비한 뒤 이름을 한 번 해석합니다. 이름 lookup이 준비한 입력을
바꿀 수 없고, mutation 직전에 선택한 service/version을 다시 검사합니다. typed 핵심 필드,
응답 ID·owner·status·크기와 인증/version header를 확장 필드로 덮어쓸 수 없습니다.
`With...Options`와 JSON 옵션은 생성 시 snapshot을 저장하고 재사용마다 독립 값을 만듭니다.

## 목록과 이름 검색

```python
clusters = conn.clustering.clusters(status="ACTIVE", limit=20)
cluster = conn.clustering.find_cluster("workers", ignore_missing=True)
```

```go
api := clusters.New(client)
for cluster, err := range api.List(ctx,
    clusters.WithListOptions(clusters.ListOpts{Status: "ACTIVE", Limit: 20}),
    clusters.WithListGlobalProject(false),
    clusters.WithListFilter("metadata", map[string]any{"team": "infra"}),
) {
    if err != nil { return err }
    fmt.Println(cluster.ID, cluster.Name)
}
cluster, err := api.Find(ctx, resource.Name("workers"))
if err != nil { return err }
if cluster != nil { fmt.Println(cluster.ID) }
```

Limit/Marker/Name/Status/Sort/GlobalProject는 서버 query입니다. GlobalProject nil은 기본값을
생략하고 명시 false는 전송합니다. sort grammar는 `created_at:desc,name` 형태이며
서버가 지원하는 key를 판정합니다. `WithListFilter`는 알려진 Body 속성의 로컬 필터로,
project_id/domain_id/user_id 별칭·재귀 object subset·배열·정확한 decimal 값을 지원합니다.
JSON bool과 숫자는 구분합니다. `WithListQuery`는 추가 서버 query이며 핵심·로컬 필터
필드를 덮어쓰지 않습니다.

목록은 lazy입니다. next/link와 HTTP Link, 명시 limit의 짧은 nonempty 페이지 뒤 wire ID
marker를 지원하며 빈 페이지에서 멈춥니다. consumer가 모델을 바꾸어도 원래 마지막 ID를
사용하고, collection origin/path와 필터를 유지하며 순환·취소·break를 처리합니다.
API.Find는 미존재를 기본적으로 `nil, nil`로 반환하고 `resource.WithMissingError()`로
변경합니다. Resources.Find는 공통 strict 기본값을 사용합니다. Name은 정확한 검색이며
중복을 오류로 반환합니다. `FindIdentity`의 GET-first·400/403/404 목록 fallback은 [자동 조회](../finding/README.md)에서 설명합니다.

## 비동기 삭제

```python
action = conn.clustering.delete_cluster("CLUSTER_ID", force_delete=True)
action = conn.clustering.get_action(action.id)
```

```go
api := clusters.New(client)
submission, err := api.Delete(ctx, resource.ID("CLUSTER_ID"), clusters.WithDeleteForce(true))
if err != nil { return err }
if submission == nil { return nil }
action, err := actions.New(client).Get(ctx, submission.ActionID)
if err != nil { return err }
fmt.Println(action.ID, action.Status)
```

일반 삭제는 body 없이 요청하고 미존재를 기본적으로 허용합니다. Force true는
`{"force":true}`를 보내며 pinned Python과 같이 기본 strict404를 사용합니다.
`WithDeleteIgnoreMissing`은 두 경로의 기본값을 명시적으로 바꾸는 Go 옵션입니다.
403·409·통신 오류·잘못된 조회 ID는 미존재로 숨기지 않습니다. Force의 숫자 버전 gate를
확인한 소스에서 찾지 못했으므로 임의로 추가하지 않습니다.

Delete는 action 조회나 완료 대기를 자동 수행하지 않습니다. 반환한
[Submission과 Location 정책](../actions/README.md)은 원문/헤더/status를 보존합니다.
Resources.Delete는 이 결과를 버리는 error-only 형태이므로 `ErrUnsupported`를 반환합니다.
Resources는 ID 조회·목록·이름 검색·상태 polling을 제공하지만 Python proxy wait 전체의
cached Resource·기본값을 완료한 의미는 아닙니다.

## 크기와 scaling 명령

```python
action = conn.clustering.scale_out_cluster("CLUSTER_ID", count=1)
action = conn.clustering.resize_cluster(
    "CLUSTER_ID", adjustment_type="EXACT_CAPACITY", number=3, min_size=0, strict=False,
)
```

```go
api := clusters.New(client)
scaled, err := api.ScaleOut(ctx, resource.ID("CLUSTER_ID"), clusters.ScaleOutOpts{},
    clusters.WithScaleOutCount(1))
if err != nil { return err }
resized, err := api.Resize(ctx, resource.ID("CLUSTER_ID"), clusters.ResizeOpts{},
    clusters.WithResizeAdjustmentType(clusters.ExactCapacity),
    clusters.WithResizeNumber(json.Number("3")), clusters.WithResizeMinSize(0),
    clusters.WithResizeStrict(false))
if err != nil { return err }
fmt.Println(scaled.ActionID, resized.ActionID)
```

ScaleIn/ScaleOut에서 count를 지정하지 않으면 pinned Python과 같이 `count:null`을 보냅니다.
명시한 0도 보존하고, scaling policy·크기 제약의 실제 판정은 Senlin이 합니다. Resize는
adjustment_type/number/min_size/max_size/min_step/strict를 제공하며 Optional의 생략·null·0·false를
구분합니다. Number는 정확한 JSON decimal 값입니다. 기본 scale/resize 명령에 임의의 최소
버전 gate를 추가하지 않습니다. 선택한 policy나 조합의 유효성은 서버가 검증합니다.

## node membership 명령

```python
action = conn.clustering.add_nodes_to_cluster("CLUSTER_ID", ["NODE_ID"])
action = conn.clustering.remove_nodes_from_cluster(
    "CLUSTER_ID", ["NODE_ID"], destroy_after_deletion=False,
)
action = conn.clustering.replace_nodes_in_cluster("CLUSTER_ID", {"OLD_NODE": "NEW_NODE"})
```

```go
api := clusters.New(client)
added, err := api.AddNodes(ctx, resource.ID("CLUSTER_ID"), clusters.AddNodesOpts{
    Nodes: []string{"NODE_ID"},
})
if err != nil { return err }
removed, err := api.RemoveNodes(ctx, resource.ID("CLUSTER_ID"), clusters.RemoveNodesOpts{
    Nodes: []string{"NODE_ID"},
}, clusters.WithRemoveNodesDestroyAfterDeletion(false))
if err != nil { return err }
replaced, err := api.ReplaceNodes(ctx, resource.ID("CLUSTER_ID"), clusters.ReplaceNodesOpts{
    Nodes: map[string]string{"OLD_NODE": "NEW_NODE"},
})
if err != nil { return err }
fmt.Println(added.ActionID, removed.ActionID, replaced.ActionID)
```

이 예제에는 numeric microversion 1.4 이상을 설정합니다. ReplaceNodes 전체는 1.3 이상,
RemoveNodes의 DestroyAfterDeletion은 false/null을 포함해 명시하면 1.4 이상이 필요합니다.
기본 AddNodes/RemoveNodes는 별도 gate 없이 동작합니다. Node identity는 서버에 전달하며
각 node를 추가 GET하지 않습니다. 비어 있는 node 목록·replacement map과 잘못된 identity는
lookup 전에 거부하고, slice/map은 이름 lookup 전에 복제합니다.

여섯 명령은 `{"action":"ID"}`와 Location이 같은 action을 가리키는 202 응답을
Submission으로 반환합니다. Python은 response JSON을 직접 반환하고 Location을 검사하지
않으므로 Go의 stricter 검증을 차이로 문서화합니다. 추가 명령 매개변수는 `With...Field`로
전달하고 typed 매개변수는 덮어쓸 수 없습니다. 매개변수는 Resource 속성과 다른 범위라
plugin의 id/status 같은 이름을 임의로 금지하지 않습니다. 잘못된 accepted 응답은 원문을
보존하고 재전송·자동 action 조회를 하지 않습니다. 이름 해석·source/version 재검사와
header 입력 보관은 CRUD와 같은 정책입니다.

## 상태 검사·복구·profile operation

```python
checked = conn.clustering.check_cluster("CLUSTER_ID")
recovered = conn.clustering.recover_cluster("CLUSTER_ID", check=False, check_capacity=False)
operated = conn.clustering.perform_operation_on_cluster(
    "CLUSTER_ID", "reboot", filters={"role": "worker"}, params={"type": "SOFT"},
)
```

```go
api := clusters.New(client)
checked, err := api.Check(ctx, resource.ID("CLUSTER_ID"))
if err != nil { return err }
recovered, err := api.Recover(ctx, resource.ID("CLUSTER_ID"), clusters.RecoverOpts{},
    clusters.WithRecoverCheck(false), clusters.WithRecoverCheckCapacity(false))
if err != nil { return err }
operated, err := api.PerformOperation(ctx, resource.ID("CLUSTER_ID"), "reboot",
    clusters.PerformOperationOpts{},
    clusters.WithPerformOperationFilters(map[string]string{"role": "worker"}),
    clusters.WithPerformOperationParams(map[string]string{"type": "SOFT"}))
if err != nil { return err }
fmt.Println(checked.ActionID, recovered.ActionID, operated.ActionID)
```

이 예제에는 numeric microversion 1.7 이상을 선택합니다. Check와 기본 Recover는
`{"check":{}}`, `{"recover":{}}`를 보내며 별도 gate가 없습니다. Recover의 Operation은
생략·빈 문자열·null, OperationParams는 생략·object·null을 보존합니다. Check를 false/null을
포함해 명시하면 1.6 이상, CheckCapacity를 명시하면 1.7 이상이 필요합니다. pinned Python은
서버 문서의 기본 false를 요청에 채우지 않으며 Go도 생략을 유지합니다.

PerformOperation은 1.4 이상에서 `/clusters/{identity}/ops`를 사용합니다. Filters는 서버가
operation 대상 node를 선택하는 객체이고 Params는 profile에 전달할 객체입니다. 로컬 목록
필터로 적용하지 않습니다. 필드의 생략과 null을 구별하며 null의 허용·필터 조건·operation
이름과 매개변수 schema는 서버가 검증합니다. 추가 plugin 필드는 내부 매개변수 객체에
들어가며 typed 필드와 인증/version/transport header를 덮어쓸 수 없습니다.

세 명령은 202의 action 문자열과 필수 Location을 같은 Submission 정책으로 검증합니다.
명시한 ID는 GET 없이 전달하고 Name은 정확한 목록 조회로 해석합니다. pinned Python의
세 proxy도 문자열 입력을 HTTP 조회 없이 resource ID로 사용하므로 이름 조회는 Go에서
명시적으로 선택하는 편의 기능입니다. 이름 lookup 전에 body/header/최소 버전을 고정하고
POST 전에 source/version을 재검사합니다. 응답 해석 실패에도 접수 요청을 재전송하거나
action을 자동 조회하지 않습니다.

이 단위 이후의 policy binding·metadata·attribute 수집 작업, 일반 inherited
per-call base_path와 JMESPath·dirty commit/merge·ID-first Find는 별도로 추적합니다.
근거는 pinned openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`cluster.py`, `_async_resource.py`, `_proxy.py`, `resource.py`와
[공식 Cluster API](https://docs.openstack.org/api-ref/clustering/#clusters)입니다.

## 정책 연결·해제·갱신

```python
attached = conn.clustering.attach_policy_to_cluster("CLUSTER_ID", "POLICY_ID")
updated = conn.clustering.update_cluster_policy("CLUSTER_ID", "POLICY_ID", enabled=False)
detached = conn.clustering.detach_policy_from_cluster("CLUSTER_ID", "POLICY_ID")
```

```go
api := clusters.New(client)
attached, err := api.AttachPolicy(ctx, resource.ID("CLUSTER_ID"),
    clusters.AttachPolicyOpts{PolicyID: "POLICY_ID"})
if err != nil { return err }
updated, err := api.UpdatePolicy(ctx, resource.ID("CLUSTER_ID"),
    clusters.UpdatePolicyOpts{PolicyID: "POLICY_ID"}, clusters.WithUpdatePolicyEnabled(false))
if err != nil { return err }
detached, err := api.DetachPolicy(ctx, resource.ID("CLUSTER_ID"),
    clusters.DetachPolicyOpts{PolicyID: "POLICY_ID"})
if err != nil { return err }
fmt.Println(attached.ActionID, updated.ActionID, detached.ActionID)
```

세 연산은 POST `/clusters/{cluster}/actions`로 각각 `policy_attach`, `policy_detach`,
`policy_update` 본문을 보내며 HTTP 202를 받습니다. 별도 microversion gate가 없고 선택한
버전이 비어 있으면 서버 기본 버전을 사용합니다. SDK는 버전을 자동으로 올리지 않습니다.
PolicyID는 필수 body 문자열이며 정책 이름·UUID·short ID를 그대로 전달합니다. 정책 조회나
URL path identifier 검증을 추가하지 않습니다. 이 UpdatePolicy는 연결 관계를 갱신하며
독립 정책 리소스의 이름이나 spec을 갱신하지 않습니다.

Attach/Update의 Enabled는 생략·false·true·null을 구분합니다. SDK는 생략한 enabled의
기본값을 요청에 채우지 않으며 `WithAttachPolicyEnabledNull`과
`WithUpdatePolicyEnabledNull`로 null을 지정합니다. 서버가 null의 의미와 정책별 매개변수를
검증합니다. Enabled를 생략한 Update도 policy_id를 가진 명령을 전송합니다.

`With...Options`와 `With...Field`는 생성 시 입력을 snapshot으로 보관하고 재사용마다
독립적으로 적용합니다. 추가 필드는 내부 명령 매개변수에 들어가므로 id/status/action/location
같은 plugin 키도 전달할 수 있습니다. concrete PolicyID/Enabled와 SDK 소유 인증·version·
transport header를 덮어쓸 수 없으며 Query/Arguments도 허용하지 않습니다. pinned Python의
attach/update kwargs가 policy_id를 덮어쓸 수 있는 동작은 Go에서 사전 오류로 처리합니다.
Detach는 기본 policy_id만 보내며 `WithDetachPolicyField`는 Python의 고정 본문보다 확장된
Go 입력입니다. 확장 필드의 지원 여부는 서버가 결정합니다.

Python의 세 proxy는 문자열 cluster를 `_find(..., ignore_missing=False)`로 먼저 조회합니다.
Go의 명시적 ID는 GET 없이 전달하고 Name은 정확한 이름의 목록 조회와 중복 검사로 한 번
해석합니다. 조회 전에 body/header를 고정하고 POST 직전에 source/version을 재검사합니다.
반환 Submission은 action 문자열과 같은 action을 가리키는 필수 Location을 검증하고
Body/Header/StatusCode를 보존합니다. 잘못된 accepted 응답은 원문 증거를 가진
`resource.ResponseError`이며 요청을 재전송하거나 action을 자동 조회하지 않습니다.

[API-ref의 정책 갱신 예제](https://docs.openstack.org/api-ref/clustering/#update-a-policy-on-a-cluster)는
`update_policy`로 표기하지만 pinned `cluster.py:168-174`와
[공식 contributor 설명](https://docs.openstack.org/senlin/ocata/developer/cluster.html#cluster-policy-bindings)은
`policy_update`를 사용합니다. Go는 pinned 요청 이름을 따릅니다. 정책 binding Get/List와
metadata는 별도 API 단위로 추적합니다.

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit=n` | `WithListOptions(ListOpts{Limit: n})` | 양수는 wire page limit; 로컬 cap과 독립 |

```go
listingAPI := clusters.New(client)
values, err := listingAPI.All(ctx,
    clusters.WithListOptions(clusters.ListOpts{Limit: 20}),
    clusters.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, clusters.WithListPaginated(false)) {
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
Resource field/default/alias 정규화 및 query 소비, per-call base_path와
deprecated JMESPath는 계속 비교합니다. `WithListQuery`는 vendor query를 실제로 전달하는
Go 확장이며 Python unknown query 생략과 구별합니다.

공통 소비 정책과 남은 차이는 [Senlin 목록 제어](../listing/README.md), 실제 HTTP 근거는 [목록 제어 테스트](../../../api/clustering_typed_list_controls_test.go)를 참고합니다.

## 목록 호출별 헤더와 버전

`WithListHeader(key, value)`와 `WithListMicroversion("1.7")`은 이 패키지의 typed `List` /
`All`에만 적용합니다. 기본값은 source client 설정이며 명시 옵션은 공유 클라이언트를
수정하지 않습니다. 실제 wire 헤더·버전 선택, 재순회·페이지·인증 정책과 Python 비교 예제는
[Senlin 목록 호출 옵션](../listing/README.md#목록-호출별-헤더와-버전)을 참고합니다.
`headers`, `microversion`, `base_path`를 `WithListQuery`로 전달하면 HTTP 전에 오류입니다.

## 문자열 이름/ID 자동 조회

Python `find_cluster(identity, ignore_missing=False)`는 `FindIdentity`와 `WithFindIgnoreMissing(false)`로 호출합니다. SDK가 GET-first, literal 이름 query, 모든 advertised 페이지의 정확한 ID/이름 일치와 중복·후속 오류를 처리합니다. 기본 미존재는 `nil, nil`이며 아래 예제는 strict입니다.

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/clustering/v1/clusters"
)

func FindClusterStrict(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Clusters.FindIdentity(ctx, "workers",
        clusters.WithFindIgnoreMissing(false))
}
```

`WithFindFallback`로 404-only·GET-only 정책을 선택하고 `WithFindHeader`·`WithFindMicroversion`으로 GET과 fallback의 동일한 호출 설정을 지정합니다. 원본 client·다른 호출은 변경하지 않습니다. [공통 FindIdentity 계약과 Python/Go 차이](../finding/README.md)에 입력 segment 정책·응답 canonical ID·오류 근거·옵션 snapshot을 설명합니다. 기존 `Find(ctx, resource.ID/Name(...))`는 명시한 경로와 기존 옵션을 유지합니다.

## cached 객체의 변경 전송

Python `update_cluster(resource)`의 dirty/no-op/응답 병합은 `TrackedCluster`로 사용합니다. concrete Update 옵션을 Edit에 전달하고 Commit이 변경된 필드만 PATCH202합니다.

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/clustering/v1/clusters"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func RenameTrackedCluster(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Clusters.Load(ctx, resource.ID("CLUSTER_ID"))
    if err != nil { return nil, err }
    if err := tracked.Edit(clusters.UpdateOpts{},
        clusters.WithUpdateName("renamed")); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

`Value()`는 병합 cache, `Response()`는 실제 응답 필드이며 Body/Header/Operation을 각각 복사합니다. clean Commit은 HTTP를 보내지 않고, accepted revision만 clean으로 바뀝니다. 반환 Operation은 작업 접수이며 완료가 아니고 action을 자동 조회하거나 poll하지 않습니다. Refresh는 고정 ID로 GET200하고 Operation을 nil로 바꿉니다. 실패·요청 중 새 Edit의 보존, null 삭제, pending 필드의 버전 gate와 Python/Go 차이는 [비동기 변경 추적](../tracking/async/README.md)을 참고합니다.

`WithUpdateProfileOnlyNull()`은 stateless Update와 tracked Edit 모두에서 명시 `profile_only:null`을 보냅니다. nil pointer의 기본 생략과 다르며 false·null·삭제 모두 실제 전송 시 1.6 이상이 필요합니다. bool helper와 null helper는 뒤 옵션이 우선하고 `WithUpdateOptions` snapshot도 null presence를 보존합니다.

`AtBasePath(path)`는 Update·Load·Track 전용 concrete scope를 반환합니다. [경로 지정과 Python/Go 사용 비교](../scoping/README.md)에서 literal 경로와 고정 Commit/Refresh 정책을 확인합니다.
