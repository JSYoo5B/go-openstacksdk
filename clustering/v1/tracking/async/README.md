# Async

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

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

See `example_async1_test.go` in [example_async1_test.go](../../example_async1_test.go), compiled by `go test`.

See `example_async2_test.go` in [example_async2_test.go](../../example_async2_test.go), compiled by `go test`.

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
