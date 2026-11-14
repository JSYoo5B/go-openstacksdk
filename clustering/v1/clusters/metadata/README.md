# Metadata

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
cluster = conn.clustering.fetch_cluster_metadata("CLUSTER_NAME_OR_ID")
print(cluster.metadata)
```

See `Example_clustersFetchMetadata` in [example_metadata_fragment1_test.go](../example_metadata_fragment1_test.go), compiled by `go test`.

```python
conn.clustering.set_cluster_metadata("CLUSTER_ID", team="infra", nullable=None)
```

See `Example_clustersSetMetadata` in [example_metadata_fragment2_test.go](../example_metadata_fragment2_test.go), compiled by `go test`.

```python
conn.clustering.delete_cluster_metadata("CLUSTER_ID", keys=["team", "nullable"])
conn.clustering.delete_cluster_metadata("CLUSTER_ID")
```

See `Example_clustersDeleteMetadata` in [example_metadata_fragment3_test.go](../example_metadata_fragment3_test.go), compiled by `go test`.

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#update-a-cluster)
- [16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py)
- [cluster controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/clusters.py)
- [conductor](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/conductor/service.py)
