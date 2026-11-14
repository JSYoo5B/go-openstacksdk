# Clusters

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

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
    Name: "workers", ProfileID: "PROFILE_ID", Timeout: request.Null[int](),
}, request.Custom(func(c *request.Config[clusters.CreateOpts]) error { v := 0; c.Options.MinSize = &v; return nil }), request.Custom(func(c *request.Config[clusters.CreateOpts]) error { v := -1; c.Options.MaxSize = &v; return nil }),
   request.Custom(func(c *request.Config[clusters.CreateOpts]) error { v := 1; c.Options.DesiredCapacity = &v; return nil }),
   clusters.WithCreateMetadata(map[string]any{"team": "infra"}))
if err != nil { return err }
cluster, err = service.Clusters.Get(ctx, cluster.ID)
if err != nil { return err }
fmt.Println(cluster.ID, cluster.Status, cluster.NodeIDs)
```

```python
cluster = conn.clustering.update_cluster(
    "workers", name="workers_v2", timeout=None, metadata={}, profile_only=False,
)
```

```go
api := clusters.New(client)
cluster, err := api.Update(ctx, resource.Name("workers"), clusters.UpdateOpts{Timeout: request.Null[int]()},
    request.Custom(func(c *request.Config[clusters.UpdateOpts]) error { c.Options.Name = request.Present("workers_v2"); return nil }),
    clusters.WithUpdateMetadata(map[string]any{}), clusters.WithUpdateProfileOnly(false))
if err != nil { return err }
fmt.Println(cluster.ID, cluster.Operation.ActionID)
```

```python
clusters = conn.clustering.clusters(status="ACTIVE", limit=20)
cluster = conn.clustering.find_cluster("workers", ignore_missing=True)
```

```go
api := clusters.New(client)
for cluster, err := range api.List(ctx,
    clusters.WithListOptions(clusters.ListOpts{Status: "ACTIVE", Limit: 20}),
    request.Custom(func(c *request.Config[clusters.ListOpts]) error { v := false; c.Options.GlobalProject = &v; return nil }),
    clusters.WithListFilter("metadata", map[string]any{"team": "infra"}),
) {
    if err != nil { return err }
    fmt.Println(cluster.ID, cluster.Name)
}
cluster, err := api.Find(ctx, resource.Name("workers"))
if err != nil { return err }
if cluster != nil { fmt.Println(cluster.ID) }
```

```python
action = conn.clustering.delete_cluster("CLUSTER_ID", force_delete=True)
action = conn.clustering.get_action(action.id)
```

```go
api := clusters.New(client)
submission, err := api.Delete(ctx, resource.ID("CLUSTER_ID"), request.Custom(func(c *request.Config[clusters.DeleteOpts]) error { c.Options.Force = true; return nil }))
if err != nil { return err }
if submission == nil { return nil }
action, err := actions.New(client).Get(ctx, submission.ActionID)
if err != nil { return err }
fmt.Println(action.ID, action.Status)
```

```python
action = conn.clustering.scale_out_cluster("CLUSTER_ID", count=1)
action = conn.clustering.resize_cluster(
    "CLUSTER_ID", adjustment_type="EXACT_CAPACITY", number=3, min_size=0, strict=False,
)
```

```go
api := clusters.New(client)
scaled, err := api.ScaleOut(ctx, resource.ID("CLUSTER_ID"), clusters.ScaleOutOpts{},
    request.Custom(func(c *request.Config[clusters.ScaleOutOpts]) error { c.Options.Count = request.Present(1); return nil }))
if err != nil { return err }
resized, err := api.Resize(ctx, resource.ID("CLUSTER_ID"), clusters.ResizeOpts{},
    request.Custom(func(c *request.Config[clusters.ResizeOpts]) error { c.Options.AdjustmentType = request.Present(clusters.ExactCapacity); return nil }),
    request.Custom(func(c *request.Config[clusters.ResizeOpts]) error { c.Options.Number = request.Present(json.Number("3")); return nil }), request.Custom(func(c *request.Config[clusters.ResizeOpts]) error { c.Options.MinSize = request.Present(0); return nil }),
    request.Custom(func(c *request.Config[clusters.ResizeOpts]) error { c.Options.Strict = request.Present(false); return nil }))
if err != nil { return err }
fmt.Println(scaled.ActionID, resized.ActionID)
```

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
}, request.Custom(func(c *request.Config[clusters.RemoveNodesOpts]) error { c.Options.DestroyAfterDeletion = request.Present(false); return nil }))
if err != nil { return err }
replaced, err := api.ReplaceNodes(ctx, resource.ID("CLUSTER_ID"), clusters.ReplaceNodesOpts{
    Nodes: map[string]string{"OLD_NODE": "NEW_NODE"},
})
if err != nil { return err }
fmt.Println(added.ActionID, removed.ActionID, replaced.ActionID)
```

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
    request.Custom(func(c *request.Config[clusters.RecoverOpts]) error { c.Options.Check = request.Present(false); return nil }), request.Custom(func(c *request.Config[clusters.RecoverOpts]) error { c.Options.CheckCapacity = request.Present(false); return nil }))
if err != nil { return err }
operated, err := api.PerformOperation(ctx, resource.ID("CLUSTER_ID"), "reboot",
    clusters.PerformOperationOpts{},
    clusters.WithPerformOperationFilters(map[string]string{"role": "worker"}),
    clusters.WithPerformOperationParams(map[string]string{"type": "SOFT"}))
if err != nil { return err }
fmt.Println(checked.ActionID, recovered.ActionID, operated.ActionID)
```

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
    clusters.UpdatePolicyOpts{PolicyID: "POLICY_ID"}, request.Custom(func(c *request.Config[clusters.UpdatePolicyOpts]) error { c.Options.Enabled = request.Present(false); return nil }))
if err != nil { return err }
detached, err := api.DetachPolicy(ctx, resource.ID("CLUSTER_ID"),
    clusters.DetachPolicyOpts{PolicyID: "POLICY_ID"})
if err != nil { return err }
fmt.Println(attached.ActionID, updated.ActionID, detached.ActionID)
```

```go
listingAPI := clusters.New(client)
values, err := listingAPI.All(ctx,
    clusters.WithListOptions(clusters.ListOpts{Limit: 20}),
    request.MaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, request.Paginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func FindClusterStrict(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Clusters.FindIdentity(ctx, "workers",
        request.IgnoreMissing(false))
}
```

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func RenameTrackedCluster(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Clusters.Load(ctx, resource.ID("CLUSTER_ID"))
    if err != nil { return nil, err }
    if err := tracked.Edit(clusters.UpdateOpts{},
        request.Custom(func(c *request.Config[clusters.UpdateOpts]) error { c.Options.Name = request.Present("renamed"); return nil })); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

## References

- [Finding](../finding/README.md)
- [Actions](../actions/README.md)
- [Clustering](https://docs.openstack.org/api-ref/clustering/#clusters)
- [Clustering](https://docs.openstack.org/api-ref/clustering/#update-a-policy-on-a-cluster)
- [Cluster](https://docs.openstack.org/senlin/ocata/developer/cluster.html#cluster-policy-bindings)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Async](../tracking/async/README.md)
- [Scoping](../scoping/README.md)
