# Senlin clusters

`conn.ClusteringV1(ctx).Clusters`가 cluster의 생성·조회·목록·수정·삭제·이름 검색을 제공합니다.
Connection은 인증된 client와 선택한 numeric microversion을 공유합니다. 아래 profile_only
예제에는 `sdk.WithMicroversion(sdk.Clustering, "1.6")` 또는 그 이상의 숫자 버전을 설정합니다.

| openstacksdk | Go | HTTP 계약 |
|---|---|---|
| `create_cluster(**attrs)` | `Clusters.Create(ctx, opts, options...)` | POST `/clusters`, 201 + cluster |
| `get_cluster(identity)` | `Clusters.Get(ctx, identity)` | GET `/clusters/{identity}`, 200 |
| `clusters(**query)` | `Clusters.List` / `All` | GET `/clusters`, 200 |
| `update_cluster(identity, **attrs)` | `Clusters.Update(ctx, ref, opts, options...)` | 객체 PATCH, 202 + cluster와 Location |
| `delete_cluster(identity, ignore_missing=True, force_delete=False)` | `Clusters.Delete(ctx, ref, options...)` | DELETE, 202 + action Submission |
| `find_cluster(identity, ignore_missing=True)` | `Clusters.Find(ctx, ref, options...)` | 명시 ID 조회 또는 정확한 Name 검색 |

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
필드는 `Header`, `StatusCode`, `Body`에 있습니다. 생성 응답에 Location이 있으면
`Operation`에도 action 참조와 원문 응답을 보존합니다.

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
중복을 오류로 반환합니다. Python의 combined-string ID-first GET fallback은 별도 과제입니다.

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

이 단위 이후의 resize·scale·node membership·policy binding·metadata 작업, 일반 inherited
max_items/paginated/JMESPath·dirty commit/merge·ID-first Find는 별도로 추적합니다.
근거는 pinned openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`cluster.py`, `_async_resource.py`, `_proxy.py`, `resource.py`와
[공식 Cluster API](https://docs.openstack.org/api-ref/clustering/#clusters)입니다.
