# Scoping

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
cluster = conn.clustering.get_cluster("CLUSTER_ID")
cluster = conn.clustering.update_cluster(
    cluster, base_path=collection_path, name="workers_renamed")

node = conn.clustering.get_node("NODE_ID")
node = conn.clustering.update_node(
    node, base_path=node_collection_path, role="worker")
```

See `example_scoping1_test.go` in [example_scoping1_test.go](../example_scoping1_test.go), compiled by `go test`.

See `example_scoping2_test.go` in [example_scoping2_test.go](../example_scoping2_test.go), compiled by `go test`.

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
