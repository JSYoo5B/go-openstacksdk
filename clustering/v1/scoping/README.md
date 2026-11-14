# Scoping

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

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
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateScopedCluster(ctx context.Context, conn *sdk.Connection, collectionPath string) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    scope, err := service.Clusters.AtBasePath(collectionPath)
    if err != nil { return nil, err }
    return scope.Update(ctx, resource.Name("workers"), clusters.UpdateOpts{},
        request.Custom(func(c *request.Config[clusters.UpdateOpts]) error { c.Options.Name = request.Present("workers_renamed"); return nil }),
        request.Header("X-Audit-Tag", "rename"))
}
```

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/request"
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
    if err := tracked.Edit(nodes.UpdateOpts{}, request.Custom(func(c *request.Config[nodes.UpdateOpts]) error { c.Options.Role = request.Present("worker"); return nil })); err != nil {
        return nil, err
    }
    accepted, err := tracked.Commit(ctx)
    if err != nil { return nil, err }
    // Later tracked.Refresh(ctx) uses the same collection and nodeID.
    return accepted, nil
}
```

## References

- [Async](../tracking/async/README.md)
- [Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1322)
- [Cluster scope](../clusters/scope.go)
- [Node scope](../nodes/scope.go)
- [Collection Path](../../../internal/senlin/collection_path.go)
- [Clustering Update Scope Test](../../../api/clustering_update_scope_test.go)
- [Collection Path Test](../../../internal/senlin/collection_path_test.go)
