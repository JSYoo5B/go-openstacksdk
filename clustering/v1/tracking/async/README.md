# Async

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

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

```go
package example

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateTrackedCluster(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Clusters.Load(ctx, resource.Name("workers"))
    if err != nil { return nil, err }
    if err := tracked.Edit(clusters.UpdateOpts{},
        request.Custom(func(c *request.Config[clusters.UpdateOpts]) error { c.Options.Name = request.Present("workers_renamed"); return nil }),
        clusters.WithUpdateTimeout(0),
        clusters.WithUpdateMetadata(map[string]any{"team": "platform"}),
        clusters.WithUpdateProfileOnlyNull()); err != nil {
        return nil, err
    }
    accepted, err := tracked.Commit(ctx, request.Header("X-Audit-Tag", "rename"))
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

    "github.com/JSYoo5B/go-openstacksdk/request"
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
        request.Custom(func(c *request.Config[nodes.UpdateOpts]) error { c.Options.Role = request.Present(""); return nil }),
        nodes.WithUpdateMetadata(map[string]any{}),
        request.Custom(func(c *request.Config[nodes.UpdateOpts]) error { c.Options.Tainted = request.Present(false); return nil })); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

## References

- [..](../README.md)
- [Microversions](../../../../docs/microversions.md)
- [update_cluster](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L368)
- [update_node](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L826)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [AtBasePath scope](../../scoping/README.md)
- [TrackedCluster](../../clusters/lifecycle.go)
- [TrackedNode](../../nodes/lifecycle.go)
- [Tracked](../../../../internal/senlin/tracked.go)
- [accepted snapshot](../../actions/submission.go)
- [Clustering Async Lifecycle Test](../../../../api/clustering_async_lifecycle_test.go)
- [Submission Test](../../actions/submission_test.go)
- [Tracked Test](../../../../internal/senlin/tracked_test.go)
- [Clustering Cluster Profile Only Null Test](../../../../api/clustering_cluster_profile_only_null_test.go)
