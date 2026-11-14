# Waiting

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
cluster = conn.clustering.get_cluster("CLUSTER_ID")
cluster = conn.clustering.wait_for_status(
    cluster, "ACTIVE", failures=["ERROR"], interval=2, wait=180,
)
```

See `Example_clustersWaitForStatus` in [example_waiting_fragment1_test.go](../example_waiting_fragment1_test.go), compiled by `go test`.

See `Example_profilesWaitForStatus` in [example_waiting_fragment2_test.go](../example_waiting_fragment2_test.go), compiled by `go test`.

```python
conn.clustering.delete_node(node)
node = conn.clustering.wait_for_delete(node)
```

See `Example_nodesDelete` in [example_waiting_fragment3_test.go](../example_waiting_fragment3_test.go), compiled by `go test`.

See `Example_nodesWaitForDelete` in [example_waiting_fragment4_test.go](../example_waiting_fragment4_test.go), compiled by `go test`.

## References

- [ClusterPolicy scope](../clusterpolicies/README.md)
- [ClusterAttributes](../clusterattributes/README.md)
- [Senlin proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1350)
- [Resource wait helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591)
- [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53)
- [Clustering Wait Contracts Test](../../../api/clustering_wait_contracts_test.go)
- [Clustering Cluster Policies Test](../../../api/clustering_cluster_policies_test.go)
