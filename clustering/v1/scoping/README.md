# Cluster·Node update 경로

`Clusters.AtBasePath(path)`와 `Nodes.AtBasePath(path)`는 SDK 소유 `UpdateScope`를 만듭니다. scope는 `Update`, `Load`, `Track`을 제공하며 앱에서 URL builder나 interface를 구현하지 않습니다. 경로는 서비스의 resource base 아래 상대 collection 경로입니다. `AtBasePath("clusters")`와 `AtBasePath("nodes")`는 기본 경로와 같습니다. 기본 `API.Update/Load/Track`도 그대로 사용합니다.

| 작업 | openstacksdk | Go |
|---|---|---|
| 기본 경로 | `update_cluster(cluster, **attrs)` | `Clusters.Update(ctx, ref, opts, options...)` |
| 경로 지정 | `update_cluster(cluster, base_path=path, **attrs)` | `scope, err := Clusters.AtBasePath(path)` → `scope.Update(...)` |
| cached 객체 | Resource에 attrs 반영 후 commit | `scope.Track(model)` → `Edit` → `Commit` |
| 이름 조회 | direct proxy의 Resource 입력 해석 | `scope.Update/Load(resource.Name(...))`는 같은 collection에서 정확한 이름 조회 |
| 후속 조회 | override는 해당 commit에만 적용 | scoped handle의 `Refresh`는 처음 선택한 collection과 ID 유지 |

```python
cluster = conn.clustering.get_cluster("CLUSTER_ID")
cluster = conn.clustering.update_cluster(
    cluster, base_path=collection_path, name="workers_renamed")

node = conn.clustering.get_node("NODE_ID")
node = conn.clustering.update_node(
    node, base_path=node_collection_path, role="worker")
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateScopedCluster(ctx context.Context, conn *sdk.Connection, collectionPath string) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    scope, err := service.Clusters.AtBasePath(collectionPath)
    if err != nil { return nil, err }
    return scope.Update(ctx, resource.Name("workers"), clusters.UpdateOpts{},
        clusters.WithUpdateName("workers_renamed"),
        clusters.WithUpdateHeader("X-Audit-Tag", "rename"))
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

func UpdateScopedNode(ctx context.Context, conn *sdk.Connection, collectionPath, nodeID string) (*nodes.Node, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    scope, err := service.Nodes.AtBasePath(collectionPath)
    if err != nil { return nil, err }
    tracked, err := scope.Load(ctx, resource.ID(nodeID))
    if err != nil { return nil, err }
    if err := tracked.Edit(nodes.UpdateOpts{}, nodes.WithUpdateRole("worker")); err != nil {
        return nil, err
    }
    accepted, err := tracked.Commit(ctx)
    if err != nil { return nil, err }
    // Later tracked.Refresh(ctx) uses the same collection and nodeID.
    return accepted, nil
}
```

## 경로와 소유권

scope 생성은 HTTP를 하지 않으며 unescaped UTF-8 segment를 한 번 escape합니다. 빈 경로·선행/후행 slash·빈 segment·`.`/`..`·URL scheme·backslash·query·fragment·percent escape/format template·공백·control은 생성 시 `resource.ErrInvalidOption`입니다. Python의 `%` URI 대입이나 전체 URL 대신 명시적인 literal collection 경로를 전달합니다. 이미 escape한 경로를 다시 입력하지 않습니다.

scope는 원래 service client와 Provider를 공유합니다. Endpoint, ResourceBase, MoreHeaders를 변경하지 않으며 최신 token과 Connection의 선택 microversion을 사용합니다. 이름 참조는 scoped 목록에서 해석하고, ID 참조는 추가 조회 없이 해당 ID로 Update합니다. 서로 다른 scope를 재사용해도 기본 API나 다른 scope의 경로는 바뀌지 않습니다.

`Load/Track`의 handle은 collection과 ID를 캡처합니다. 응답의 ID나 입력/getter 모델을 수정해도 Commit/Refresh의 route는 바뀌지 않습니다. `Edit`은 로컬 작업이며 clean Commit은 source/context/header를 검증한 뒤 HTTP 없이 반환합니다. dirty·null·버전 gate·실패·동시 편집·응답 snapshot 정책은 [Cluster·Node 변경 추적](../tracking/async/README.md)과 같습니다.

accepted action의 Location은 원래 서비스의 `actions` collection에서 검사합니다. scoped collection 뒤에 `actions`를 붙이거나 Location을 따라가지 않습니다. action의 접수와 적용 완료는 별개이며 응답을 다시 제출하지 않습니다. scope는 Create/Delete/List나 다른 명령의 경로를 바꾸지 않습니다.

## 고정 소스의 Go 매핑

Python [Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716)는 direct Cluster/Node Update attrs에서 `base_path`를 소비하고 [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)에 전달합니다. [요청 준비](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1322)는 URI 대입 후 리소스 ID를 붙이며 Resource의 기본 path를 바꾸지 않습니다. Go는 안전한 literal 경로를 concrete scope로 소유하고 Load/Refresh까지 같은 경로로 연결합니다. Python raw attrs의 alias/readonly·cached ID 수정은 Go의 mutable concrete 필드, core/readonly 보호, 고정 route로 매핑합니다. `prepend_key/has_body/retry_on_conflict` 등 inherited commit 제어는 이 두 direct proxy가 전달하는 인자가 아닙니다.

구현: [Cluster scope](../clusters/scope.go), [Node scope](../nodes/scope.go), [공통 경로 검증](../../../internal/senlin/collection_path.go). 검증: [scope HTTP 계약](../../../api/clustering_update_scope_test.go), [경로 검증 사례](../../../internal/senlin/collection_path_test.go).
