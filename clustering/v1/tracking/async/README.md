# Cluster·Node 변경 추적

`Clusters.Load/Track`과 `Nodes.Load/Track`은 SDK 소유 `TrackedCluster`·`TrackedNode`를 반환합니다. 기존 concrete `UpdateOpts`·`WithUpdate...`를 `Edit`에 사용하고 `Commit`이 변경한 필드만 PATCH합니다. 앱에서 builder나 lifecycle interface를 구현할 필요가 없습니다. Profile·Policy와 같은 [공통 dirty 정책](../README.md)을 사용하며 HTTP 202의 접수와 작업 완료를 구분합니다.

| 작업 | openstacksdk | Go |
|---|---|---|
| 조회한 객체 편집 | `cluster = conn.clustering.get_cluster(id)` | `tracked, err := service.Clusters.Load(ctx, resource.ID(id))` |
| 기존 객체 편집 | `conn.clustering.update_node(node)` | `service.Nodes.Track(node)` → `Edit` → `Commit` |
| 속성 대입 | `cluster.name = "renamed"` | `tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateName("renamed"))` |
| 필드 삭제 | `del node.metadata` | `tracked.RemoveMetadata()`; Commit에 `metadata:null` |
| 변경 없음 | cached Resource commit의 no-op | `Commit`은 cache snapshot을 반환하고 PATCH 없음 |
| 갱신 결과 | 같은 Resource에 서버 필드 병합 | `Value()`는 병합 cache, `Response()`는 실제 응답 필드 |
| 새 조회 | `resource.fetch(session)` | `Refresh(ctx)`; 고정 ID로 GET200, 기존 dirty reset |
| 비동기 접수 | Resource 반환; Update는 Location을 Action으로 변환하지 않음 | `model.Operation`에 action ID·Location·원본 HTTP 증거 |

```python
cluster = conn.clustering.get_cluster("CLUSTER_ID")
cluster.name = "workers_renamed"
cluster.metadata = {"team": "platform"}
cluster = conn.clustering.update_cluster(cluster)

node = conn.clustering.get_node("NODE_ID")
node.role = ""
node.metadata = {}
node = conn.clustering.update_node(node)
```

다음 Go 예제의 Connection은 필요한 numeric microversion을 선택해야 합니다. Cluster의 명시 `profile_only`는 1.6 이상, Node의 `tainted`는 1.13 이상입니다. `sdk.WithMicroversion(sdk.Clustering, "1.13")` 또는 [버전 range 선택](../../../../docs/microversions.md)을 Connection 구성에 사용합니다. handle은 공유 client를 자동으로 업그레이드하지 않습니다.

```go
package example

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateTrackedCluster(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Clusters.Load(ctx, resource.Name("workers"))
    if err != nil { return nil, err }
    if err := tracked.Edit(clusters.UpdateOpts{},
        clusters.WithUpdateName("workers_renamed"),
        clusters.WithUpdateTimeout(0),
        clusters.WithUpdateMetadata(map[string]any{"team": "platform"}),
        clusters.WithUpdateProfileOnlyNull()); err != nil {
        return nil, err
    }
    accepted, err := tracked.Commit(ctx, clusters.WithUpdateHeader("X-Audit-Tag", "rename"))
    if err != nil { return nil, err }
    if accepted.Operation != nil {
        fmt.Println(accepted.Operation.ActionID, accepted.Operation.Location)
    }
    fmt.Println(tracked.Dirty(), tracked.Response().Body, accepted.UserMetadata)
    return accepted, nil
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateTrackedNode(ctx context.Context, conn *sdk.Connection) (*nodes.Node, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Nodes.Load(ctx, resource.ID("NODE_ID"))
    if err != nil { return nil, err }
    if err := tracked.Edit(nodes.UpdateOpts{},
        nodes.WithUpdateRole(""),
        nodes.WithUpdateMetadata(map[string]any{}),
        nodes.WithUpdateTainted(false)); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

## 로컬 편집과 전송 필드

`Edit`은 HTTP·버전 협상을 하지 않습니다. name·profile ID·JSON 객체 등 기존 stateless Update의 로컬 규칙과 extension 보호를 검증하고, 잘못된 옵션이면 이전 편집을 일부 적용하지 않습니다. 같은 현재 값의 대입은 clean을 유지하고 `A → B → A`는 sticky dirty로 남습니다. cache에 있는 `profile_only`·`tainted`만으로 Commit 버전 요구를 만들지 않으며 실제 pending key의 존재를 검사합니다. false·null·삭제도 전송하려는 필드이므로 각각의 버전 요구가 적용됩니다.

| handle | core 편집·삭제 helper |
|---|---|
| `TrackedCluster` | `name`, `profile_id`, `timeout`, `config`, `metadata`, `profile_only`; `RemoveName/ProfileID/Timeout/Config/Metadata/ProfileOnly()` |
| `TrackedNode` | `name`, `profile_id`, `role`, `metadata`, `tainted`; `RemoveName/ProfileID/Role/Metadata/Tainted()` |

`Remove...`는 cached field를 제거하고 Commit에 null을 남깁니다. 없던 필드의 삭제는 no-op입니다. `WithUpdateMetadata(nil)`처럼 명시 null을 대입하면 필드가 cache에 남습니다. 빈 metadata/config 객체, 명시 zero/false/empty와 생략은 다릅니다. `WithUpdateProfileOnlyNull()`은 없던 Cluster 필드에도 명시 null을 대입합니다. 기존 `ProfileOnly *bool` 입력과 `WithUpdateProfileOnly(bool)`는 유지하며 뒤 옵션이 앞 null/bool을 덮어씁니다. `RemoveProfileOnly()`는 기존 cached field를 제거하며 absent 필드에서는 no-op입니다. Python bool descriptor의 누락 기본값은 None이며 None/False 대입을 생략하지 않습니다. Go도 JSON presence를 구분하고 Python의 bool(...) 암묵변환을 제공하지 않습니다. `WithUpdateOptions`는 JSON으로 입력을 snapshot하며 명시 profile_only null도 보존합니다. 서버의 nullable 필드·plugin schema와 권한은 deployment가 결정합니다.

readonly·core 필드와 그 대소문자 별칭을 `WithUpdateField`로 우회하지 않습니다. unknown vendor 필드는 명시 편집할 수 있으며 서버가 검사합니다. name·profile·크기·status 같은 known 필드에는 서비스의 concrete 작업을 사용합니다. Cluster resize와 Node의 cluster membership은 tracked Update 필드가 아닙니다.

Commit 옵션은 기존 `WithUpdateHeader`만 받으며 body 변경은 Edit에 전달합니다. protected auth/version/transport header와 arbitrary query/argument를 거부합니다. clean Commit에도 context·서비스·버전 형식·header 검증을 적용하고, 유효한 header를 지정해도 dirty body가 없으면 HTTP를 보내지 않습니다. dirty 버전 gate 실패는 cache와 pending을 보존합니다.

## 접수·응답·소유권

Commit은 고정 리소스 route에 PATCH하고 `cluster`/`node` 객체 envelope의 202와 유효한 action Location을 요구합니다. action은 별도의 식별자이며 다른 origin·collection으로 이동하거나 자동 GET·poll을 하지 않습니다. `Operation`의 Body/Header/StatusCode는 accepted 요청의 전체 원문입니다. **dirty=false는 그 revision의 접수를 뜻하며 작업 완료를 뜻하지 않습니다.** 상태 완료가 필요하면 기존 Actions 또는 리소스 Wait API를 별도로 사용합니다.

응답은 whole field 단위로 cache에 병합합니다. nested 객체는 전체 교체하며 전송한 필드를 응답이 생략해도 그 revision은 clean이 되고 편집한 값은 cache에 남습니다. `Response().Body`는 서버가 실제로 반환한 필드만 보존합니다. `Value().Body`는 cache와 편집·응답을 합친 view여서 모든 필드가 서버의 마지막 응답에 있었다는 증거가 아닙니다.

모델·Body·raw JSON·Header·Operation은 입력과 getter마다 복사합니다. `Operation`은 JSON 밖의 SDK 필드이므로 별도로 snapshot하며 action body/header를 수정해도 handle과 다른 snapshot을 바꾸지 않습니다. `actions.Submission.Snapshot()`도 같은 복사 기능을 제공합니다. Body 없는 수동 모델은 로컬 seed로 감싸고 없던 HTTP 상태나 header를 만들지 않습니다. clean Commit의 Operation은 마지막 accepted 요청의 기록일 수 있으므로 새 PATCH가 있었다는 근거로 사용하지 않습니다.

`Load(resource.ID(...))`는 조회 응답의 ID가 다르거나 null·생략이어도 요청 ID를 이후 PATCH·Refresh에 고정합니다. `Load(resource.Name(...))`는 정확한 raw 이름을 모든 선택한 페이지에서 검색해 한 번 해석하고 detail GET을 추가하지 않습니다. 미존재·중복은 오류입니다. `Track`과 Name Load의 route는 exact lowercase raw `id`로 검사하며 unknown `ID`·`Name` extension이 route나 canonical 이름을 만들지 않습니다. ID/name의 null·생략은 Body에서 구분합니다. getter나 원본 모델의 ID를 바꿔도 handle route는 바뀌지 않습니다.

## 실패·Refresh·동시 편집

HTTP 실패·잘못된 accepted envelope·decode·Location 오류에서는 cache·실제 응답·Operation과 dirty 상태를 바꾸지 않습니다. accepted 응답 오류는 mutation이 처리된 뒤 발생할 수 있으며 SDK가 자동으로 재전송하지 않습니다. 유효한 context로 `Refresh`해 상태를 확인한 뒤 후속 편집·Commit을 결정합니다.

Refresh는 고정 route의 GET200 응답을 병합하고 GET 전에 있던 dirty revision을 초기화합니다. 실패하면 이전 상태를 보존합니다. 성공하면 Response가 GET 응답으로 바뀌고 마지막 `Operation`은 nil이 됩니다. GET200도 리소스의 상태 완료를 보장하지 않습니다. Commit·Refresh는 handle별로 직렬화하고, HTTP 중 새 Edit은 revision을 비교해 보존합니다. 응답이 새 편집을 덮어쓰거나 새 revision을 clean으로 만들지 않습니다.

## Python/Go 계약과 검증

고정 Python의 [update_cluster](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L368)와 [update_node](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L826)는 cached Resource 입력과 attrs를 generic update로 전달합니다. [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)의 dirty/no-op/응답 병합을 concrete handle로 매핑합니다. AsyncResource는 이 Update/commit을 override하지 않습니다.

Go는 명시 context/error, 고정 route, immutable snapshot, readonly 보호, 엄격한 객체 envelope와 accepted action Location, 명시적인 숫자 버전 선택을 사용합니다. Python의 permissive response fallback·객체 pointer mutation·임의 Resource subclass는 concrete 모델·handle로 매핑합니다. 직접 Update proxy의 `base_path`는 [AtBasePath scope](../../scoping/README.md)로 매핑하며 handle은 선택한 collection과 ID를 고정합니다. `prepend_key/has_body/retry_on_conflict`는 이 두 proxy가 commit에 전달하는 제어가 아니고 inherited Resource.commit의 별도 표면입니다. attrs의 microversion도 named commit override로 그대로 전달되지 않으며 Go는 Connection의 선택·range 협상을 사용합니다. 기존 stateless `API.Update`도 유지하며 빈 요청을 거부합니다.

구현: [TrackedCluster](../../clusters/lifecycle.go), [TrackedNode](../../nodes/lifecycle.go), [공통 dirty 상태](../../../../internal/senlin/tracked.go), [accepted snapshot](../../actions/submission.go). 검증: [Cluster·Node HTTP 계약](../../../../api/clustering_async_lifecycle_test.go), [Submission snapshot 테스트](../../actions/submission_test.go), [기존 공통 dirty 테스트](../../../../internal/senlin/tracked_test.go), [profile_only null HTTP 계약](../../../../api/clustering_cluster_profile_only_null_test.go).
