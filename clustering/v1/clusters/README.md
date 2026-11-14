# Clusters

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
cluster = conn.clustering.create_cluster(
    name="workers", profile_id="PROFILE_ID",
    min_size=0, max_size=-1, desired_capacity=1, timeout=None,
    metadata={"team": "infra"},
)
cluster = conn.clustering.get_cluster(cluster.id)
```

See `Example_clustersFragment1` in [example_clusters_fragment1_test.go](example_clusters_fragment1_test.go), compiled by `go test`.

```python
cluster = conn.clustering.update_cluster(
    "workers", name="workers_v2", timeout=None, metadata={}, profile_only=False,
)
```

See `Example_clustersFragment2` in [example_clusters_fragment2_test.go](example_clusters_fragment2_test.go), compiled by `go test`.

```python
clusters = conn.clustering.clusters(status="ACTIVE", limit=20)
cluster = conn.clustering.find_cluster("workers", ignore_missing=True)
```

See `Example_clustersFragment3` in [example_clusters_fragment3_test.go](example_clusters_fragment3_test.go), compiled by `go test`.

```python
action = conn.clustering.delete_cluster("CLUSTER_ID", force_delete=True)
action = conn.clustering.get_action(action.id)
```

See `Example_forceDelete` in [example_delete_test.go](example_delete_test.go), compiled by `go test`.

```python
action = conn.clustering.scale_out_cluster("CLUSTER_ID", count=1)
action = conn.clustering.resize_cluster(
    "CLUSTER_ID", adjustment_type="EXACT_CAPACITY", number=3, min_size=0, strict=False,
)
```

See `Example_clustersFragment5` in [example_clusters_fragment5_test.go](example_clusters_fragment5_test.go), compiled by `go test`.

```python
action = conn.clustering.add_nodes_to_cluster("CLUSTER_ID", ["NODE_ID"])
action = conn.clustering.remove_nodes_from_cluster(
    "CLUSTER_ID", ["NODE_ID"], destroy_after_deletion=False,
)
action = conn.clustering.replace_nodes_in_cluster("CLUSTER_ID", {"OLD_NODE": "NEW_NODE"})
```

See `Example_clustersFragment6` in [example_clusters_fragment6_test.go](example_clusters_fragment6_test.go), compiled by `go test`.

```python
checked = conn.clustering.check_cluster("CLUSTER_ID")
recovered = conn.clustering.recover_cluster("CLUSTER_ID", check=False, check_capacity=False)
operated = conn.clustering.perform_operation_on_cluster(
    "CLUSTER_ID", "reboot", filters={"role": "worker"}, params={"type": "SOFT"},
)
```

See `Example_clustersFragment7` in [example_clusters_fragment7_test.go](example_clusters_fragment7_test.go), compiled by `go test`.

```python
attached = conn.clustering.attach_policy_to_cluster("CLUSTER_ID", "POLICY_ID")
updated = conn.clustering.update_cluster_policy("CLUSTER_ID", "POLICY_ID", enabled=False)
detached = conn.clustering.detach_policy_from_cluster("CLUSTER_ID", "POLICY_ID")
```

See `Example_clustersFragment8` in [example_clusters_fragment8_test.go](example_clusters_fragment8_test.go), compiled by `go test`.

See `Example_clustersFragment9` in [example_clusters_fragment9_test.go](example_clusters_fragment9_test.go), compiled by `go test`.

See `example_clusters1_test.go` in [example_clusters1_test.go](example_clusters1_test.go), compiled by `go test`.

See `example_clusters2_test.go` in [example_clusters2_test.go](example_clusters2_test.go), compiled by `go test`.

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
