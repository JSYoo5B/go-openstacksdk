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

See `Example_nodesFragment1` in [example_nodes_fragment1_test.go](example_nodes_fragment1_test.go), compiled by `go test`.

```python
node = conn.clustering.update_node(
    "worker_1", name="worker_2", role=None, metadata={}, tainted=False,
)
```

See `Example_nodesFragment2` in [example_nodes_fragment2_test.go](example_nodes_fragment2_test.go), compiled by `go test`.

```python
nodes = conn.clustering.nodes(cluster_id="CLUSTER_ID", status="ACTIVE", limit=20)
node = conn.clustering.find_node("worker_1", ignore_missing=True)
```

See `Example_nodesFragment3` in [example_nodes_fragment3_test.go](example_nodes_fragment3_test.go), compiled by `go test`.

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

See `Example_nodesFragment5` in [example_nodes_fragment5_test.go](example_nodes_fragment5_test.go), compiled by `go test`.

```python
node = conn.clustering.adopt_node(
    identity="PHYSICAL_RESOURCE_ID", type="os.nova.server-1.0",
    name="adopted_worker", snapshot=False, metadata={},
)
preview = conn.clustering.adopt_node(
    preview=True, identity="PHYSICAL_RESOURCE_ID", type="os.nova.server-1.0",
)
```

See `Example_nodesFragment6` in [example_nodes_fragment6_test.go](example_nodes_fragment6_test.go), compiled by `go test`.

See `Example_nodesFragment7` in [example_nodes_fragment7_test.go](example_nodes_fragment7_test.go), compiled by `go test`.

See `example_nodes1_test.go` in [example_nodes1_test.go](example_nodes1_test.go), compiled by `go test`.

See `example_nodes2_test.go` in [example_nodes2_test.go](example_nodes2_test.go), compiled by `go test`.

## References

- [Actions](../actions/README.md)
- [Clustering](https://docs.openstack.org/api-ref/clustering/#nodes)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Finding](../finding/README.md)
- [Async](../tracking/async/README.md)
- [Scoping](../scoping/README.md)
