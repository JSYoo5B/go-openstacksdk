# Nodes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
node = conn.clustering.create_node(
    name="worker_1", profile_id="PROFILE_ID", cluster_id="CLUSTER_ID",
    role="worker", metadata={"team": "infra"},
)
node = conn.clustering.get_node(node.id, details=True)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
node, err := service.Nodes.Create(ctx, nodes.CreateOpts{
    Name: "worker_1", ProfileID: "PROFILE_ID",
}, request.Custom(func(c *request.Config[nodes.CreateOpts]) error { c.Options.ClusterID = request.Present("CLUSTER_ID"); return nil }), request.Custom(func(c *request.Config[nodes.CreateOpts]) error { c.Options.Role = request.Present("worker"); return nil }),
   nodes.WithCreateMetadata(map[string]any{"team": "infra"}))
if err != nil { return err }
node, err = service.Nodes.Get(ctx, node.ID, request.Custom(func(c *request.Config[nodes.GetOpts]) error { v := true; c.Options.Details = &v; return nil }))
if err != nil { return err }
fmt.Println(node.ID, node.PhysicalID, node.Details)
```

```python
node = conn.clustering.update_node(
    "worker_1", name="worker_2", role=None, metadata={}, tainted=False,
)
```

```go
api := nodes.New(client)
node, err := api.Update(ctx, resource.Name("worker_1"), nodes.UpdateOpts{Role: request.Null[string]()},
    request.Custom(func(c *request.Config[nodes.UpdateOpts]) error { c.Options.Name = request.Present("worker_2"); return nil }),
    request.Custom(func(c *request.Config[nodes.UpdateOpts]) error { c.Options.Tainted = request.Present(false); return nil }), nodes.WithUpdateMetadata(map[string]any{}))
if err != nil { return err }
fmt.Println(node.ID, node.Operation.ActionID)
```

```python
nodes = conn.clustering.nodes(cluster_id="CLUSTER_ID", status="ACTIVE", limit=20)
node = conn.clustering.find_node("worker_1", ignore_missing=True)
```

```go
api := nodes.New(client)
for node, err := range api.List(ctx,
    nodes.WithListOptions(nodes.ListOpts{ClusterID: "CLUSTER_ID", Status: "ACTIVE", Limit: 20}),
    request.Custom(func(c *request.Config[nodes.ListOpts]) error { v := false; c.Options.ShowDetails = &v; return nil }), request.Custom(func(c *request.Config[nodes.ListOpts]) error { v := false; c.Options.GlobalProject = &v; return nil }),
    nodes.WithListFilter("metadata", map[string]any{"team": "infra"}),
) {
    if err != nil { return err }
    fmt.Println(node.ID, node.PhysicalID)
}
node, err := api.Find(ctx, resource.Name("worker_1"))
if err != nil { return err }
if node != nil { fmt.Println(node.ID) }
```

```python
action = conn.clustering.delete_node("NODE_ID", force_delete=True)
action = conn.clustering.get_action(action.id)
```

```go
api := nodes.New(client)
submission, err := api.Delete(ctx, resource.ID("NODE_ID"), request.Custom(func(c *request.Config[nodes.DeleteOpts]) error { c.Options.Force = true; return nil }))
if err != nil { return err }
if submission == nil { return nil }
action, err := actions.New(client).Get(ctx, submission.ActionID)
if err != nil { return err }
fmt.Println(action.ID, action.Status)
```

```python
checked = conn.clustering.check_node("NODE_ID")
recovered = conn.clustering.recover_node(
    "NODE_ID", operation="REBOOT", operation_params={}, check=False,
)
performed = conn.clustering.perform_operation_on_node("NODE_ID", "reboot", type="SOFT")
```

```go
api := nodes.New(client)
checked, err := api.Check(ctx, resource.ID("NODE_ID"))
if err != nil { return err }
recovered, err := api.Recover(ctx, resource.ID("NODE_ID"), nodes.RecoverOpts{},
    request.Custom(func(c *request.Config[nodes.RecoverOpts]) error { c.Options.Operation = request.Present("REBOOT"); return nil }), nodes.WithRecoverOperationParams(map[string]any{}),
    request.Custom(func(c *request.Config[nodes.RecoverOpts]) error { c.Options.Check = request.Present(false); return nil }))
if err != nil { return err }
performed, err := api.PerformOperation(ctx, resource.ID("NODE_ID"), "reboot",
    request.Field("type", "SOFT"))
if err != nil { return err }
fmt.Println(checked.ActionID, recovered.ActionID, performed.ActionID)
```

```python
node = conn.clustering.adopt_node(
    identity="PHYSICAL_RESOURCE_ID", type="os.nova.server-1.0",
    name="adopted_worker", snapshot=False, metadata={},
)
preview = conn.clustering.adopt_node(
    preview=True, identity="PHYSICAL_RESOURCE_ID", type="os.nova.server-1.0",
)
```

```go
api := nodes.New(client)
node, err := api.Adopt(ctx, nodes.AdoptOpts{
    Identity: "PHYSICAL_RESOURCE_ID", Type: "os.nova.server-1.0",
}, request.Custom(func(c *request.Config[nodes.AdoptOpts]) error { c.Options.Name = request.Present("adopted_worker"); return nil }), request.Custom(func(c *request.Config[nodes.AdoptOpts]) error { c.Options.Snapshot = request.Present(false); return nil }),
   nodes.WithAdoptMetadata(map[string]any{}))
if err != nil { return err }
preview, err := api.AdoptPreview(ctx, nodes.AdoptPreviewOpts{
    Identity: "PHYSICAL_RESOURCE_ID", Type: "os.nova.server-1.0",
})
if err != nil { return err }
fmt.Println(node.ID, node.PhysicalID, preview.Spec.Type, string(preview.Spec.Version))
```

```go
listingAPI := nodes.New(client)
values, err := listingAPI.All(ctx,
    nodes.WithListOptions(nodes.ListOpts{Limit: 20}),
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
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func FindNodeStrict(ctx context.Context, conn *sdk.Connection) (*nodes.Node, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Nodes.FindIdentity(ctx, "worker_1",
        request.IgnoreMissing(false))
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

func RenameTrackedNode(ctx context.Context, conn *sdk.Connection) (*nodes.Node, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Nodes.Load(ctx, resource.ID("NODE_ID"))
    if err != nil { return nil, err }
    if err := tracked.Edit(nodes.UpdateOpts{},
        request.Custom(func(c *request.Config[nodes.UpdateOpts]) error { c.Options.Name = request.Present("renamed"); return nil })); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

## References

- [Actions](../actions/README.md)
- [Clustering](https://docs.openstack.org/api-ref/clustering/#nodes)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Finding](../finding/README.md)
- [Async](../tracking/async/README.md)
- [Scoping](../scoping/README.md)
