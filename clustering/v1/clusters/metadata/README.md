# Senlin cluster metadata

`Clusters.FetchMetadata` / `GetMetadata` / `SetMetadata` / `DeleteMetadata`는 실제 Senlin
cluster GET/PATCH 위에 metadata 읽기와 merge·삭제를 제공합니다. 16.0.0 router에는
`/clusters/{id}/metadata`와 `/metadata/{key}` route가 없습니다. Pinned Python
`MetadataMixin`이 사용하는 subresource transport와 Go의 transport는 다릅니다.

| openstacksdk | Go | 실제 Go 요청 |
|---|---|---|
| `fetch_cluster_metadata(cluster)` | `FetchMetadata(ctx, ref)` | GET `/clusters/{identity}`, 200 + `cluster` |
| `get_cluster_metadata(cluster)` | `GetMetadata(ctx, ref)` | FetchMetadata alias |
| `set_cluster_metadata(cluster, **metadata)` | `SetMetadata(ctx, ref, values, options...)` | GET 후 기존 키를 shallow merge하여 객체 PATCH, 202 + cluster와 Location |
| `delete_cluster_metadata(cluster, keys=None)` | `DeleteMetadata(ctx, ref, keys, options...)` | GET 후 선택 키 또는 전체 제거, 한 번의 객체 PATCH |

Python mixin은 metadata GET·POST·PUT·key별 DELETE를 사용하고 로컬 Resource metadata를
바꿉니다. Go는 지원되지 않는 route를 호출하지 않으며, 실제 GET 응답과 요청한 replacement,
수락된 PATCH 응답을 각각 보존합니다. 이 functional layer는 Python transport나 cache
lifecycle의 동일 구현으로 판정하지 않습니다.

## 읽기

```python
cluster = conn.clustering.fetch_cluster_metadata("CLUSTER_NAME_OR_ID")
print(cluster.metadata)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
view, err := service.Clusters.FetchMetadata(ctx, resource.ID("CLUSTER_NAME_OR_ID"))
if err != nil { return err }
fmt.Println(view.Present, view.Null, string(view.Values["team"]))
```

직접 service client를 가진 경우 `clusters.New(client)`로 같은 API를 사용할 수 있습니다.
`MetadataView.Values`는 `map[string]json.RawMessage`입니다. 큰 정수·decimal·배열·object·
boolean·null을 float64로 변환하지 않습니다. `Present == false`는 metadata 필드 생략,
`Present && Null`은 명시 null이며, 빈 object는 nonnil empty map입니다. `Cluster`는
raw Body/Header/StatusCode를 가진 관측 응답입니다. Values, Cluster.UserMetadata,
Cluster.Body의 byte slice는 서로 독립적입니다.

알려진 metadata는 정확한 lower-case `Body["metadata"]`에서 읽습니다. 대소문자가 다른
extension 필드가 값을 바꾸지 않습니다. 잘못된 JSON이나 object/null 이외의 metadata를
받으면 GET 전체 원문 body/header/status가 있는 `resource.ResponseError`를 반환합니다.
관측 metadata가 생략 또는 null이면 merge·삭제 계산에서 빈 object로 취급하되 원문과
presence flags는 보존합니다.

## 기존 키 보존과 queued replacement

```python
conn.clustering.set_cluster_metadata("CLUSTER_ID", team="infra", nullable=None)
```

```go
result, err := service.Clusters.SetMetadata(ctx, resource.ID("CLUSTER_ID"),
    map[string]any{
        "team": "infra",
        "nullable": nil,
        "big": json.Number("9007199254740993"),
    }, clusters.WithMetadataHeader("X-Request-Id", "metadata-change"))
if err != nil { return err }
fmt.Println(result.Changed, string(result.Previous["team"]), string(result.Requested["team"]))
if result.Operation != nil { fmt.Println(result.Operation.ActionID) }
```

Set은 입력 map을 HTTP 이전에 JSON snapshot으로 저장합니다. top-level 키를 merge하므로
다른 키는 관측 값 그대로 보존하지만, 입력에 있는 nested object는 그 키 전체를 교체합니다.
nil 값은 JSON null이고 키 삭제를 뜻하지 않습니다. nil map과 empty map은 추가할 키가 없는
Set이며, GET을 수행한 뒤 PATCH를 생략합니다. malformed RawMessage나 JSON 직렬화 불가능
값은 HTTP 전에 오류입니다.

`MetadataResult.Previous`와 `Requested`는 독립적인 raw map입니다. Previous는 GET 관측 값,
Requested는 전송 대상으로 계산한 전체 object입니다. `PreviousPresent` / `PreviousNull`로
관측 필드 생략·null을 구분합니다. `ResponseCluster`는 실제 마지막 성공 응답이며, 변경을
제출했다면 PATCH 응답이고 no-op이면 GET 응답입니다. `Operation`은 accepted action 참조와
독립적인 원문/헤더/status를 보존합니다.

`Changed == true`는 replacement PATCH가 202로 수락되었다는 뜻입니다. 요청 값이 적용되거나
action이 완료되었다는 뜻은 아닙니다. PATCH의 ResponseCluster.metadata가 Requested와
다르거나 생략되어도 요청 값을 응답에 합성하지 않습니다. action 조회·Location follow·대기는
자동 수행하지 않으며, 필요한 확인은 명시적인 Actions 조회 또는 후속 FetchMetadata로 합니다.

관측 값과 요청 값이 같으면 PATCH를 생략합니다. 비교는 object 순서에 의존하지 않고 정확한
decimal 값으로 하며 큰 exponent를 float64로 바꾸지 않습니다. 1과 1.0은 같고 false와 0은
다릅니다. nested object에서 생략한 키와 null 키도 구분합니다. 같은 값에 대한 서버의
불필요한 update 거부를 피하며, no-op의 Changed는 false이고 Operation은 nil입니다.

## 삭제

```python
conn.clustering.delete_cluster_metadata("CLUSTER_ID", keys=["team", "nullable"])
conn.clustering.delete_cluster_metadata("CLUSTER_ID")
```

```go
result, err := service.Clusters.DeleteMetadata(ctx, resource.ID("CLUSTER_ID"),
    []string{"team", "nullable"})
if err != nil { return err }
result, err = service.Clusters.DeleteMetadata(ctx, resource.ID("CLUSTER_ID"), nil)
if err != nil { return err }
fmt.Println(result.Changed)
```

nil keys는 전체 metadata를 `{}`로 교체합니다. nonnil empty slice는 요청할 삭제가 없어
GET과 PATCH를 모두 생략하고 observation 없는 결과를 반환합니다. 이 경우 ResponseCluster,
Previous, Requested와 Operation은 nil이고 Changed는 false입니다. 선택 키는 독립적인
slice snapshot을 만들고 중복 키까지 한 번의 replacement에 반영합니다. 없는 키만 선택하거나
전체 삭제 대상이 이미 empty/생략/null이면 GET 이후 PATCH를 생략합니다. key별 DELETE나
전체 metadata subresource PUT을 호출하지 않습니다.

## identity, 옵션과 동시 변경

명시적인 `resource.ID("cluster-name")`은 supplied identity를 controller에 직접 전달합니다.
응답에 다른 ID가 있더라도 GET과 PATCH의 target을 바꾸지 않습니다. `resource.Name("name")`은
정확한 Name 목록 검색을 한 번 수행하고 canonical raw `id`로 GET/PATCH합니다. raw `name`과
`id`를 검증하므로 대소문자가 다른 extension이 이름 매칭이나 route를 바꾸지 않습니다. 이름
중복·부모 미존재·잘못된 canonical response ID는 mutation 전에 실패합니다.

MetadataOption은 header-only입니다. 인증·version·Host·content header는 SDK 소유이고,
body Fields·query·secondary Arguments는 거부합니다. 옵션과 입력을 HTTP 전에 준비하고
header를 독립적으로 복사합니다. custom header는 metadata GET/PATCH에 사용하며 Name 목록
lookup에는 source client의 공통 header 정책을 사용합니다. 각 요청과 no-op 반환 전에
source/context를 검사하고 별도의 minimum microversion gate는 추측하지 않습니다.

GET→PATCH는 전체 metadata를 읽고 교체하는 과정입니다. 서버 CAS나 revision 검사가 없으므로
GET 이후 다른 요청이 추가한 키를 이 replacement가 잃을 수 있고, 먼저 queued된 metadata
action의 적용 시점과도 경쟁할 수 있습니다. no-op 역시 관측 시점의 판단입니다. 자동 merge
재시도나 rollback을 하지 않으며 원자적인 동시 변경 보존을 약속하지 않습니다.

HTTP 403/404/409 등 서버 오류와 context 원인을 유지합니다. 수락 후 envelope·metadata·필수
Location 해석이 실패하거나 response 읽기 중 취소되면 202 전체 응답 증거를 보존합니다.
이 오류가 action 미수락을 뜻하지 않으며 PATCH를 재전송하지 않습니다.

계약 근거: [공식 cluster update](https://docs.openstack.org/api-ref/clustering/#update-a-cluster),
[16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py),
[cluster controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/clusters.py),
[conductor](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/conductor/service.py).
Pinned Python 비교는 `openstack/clustering/v1/_proxy.py:384–451`과
`openstack/common/metadata.py:33–92,137–160`에 대응합니다.
