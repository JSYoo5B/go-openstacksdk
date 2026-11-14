# Clusterattributes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
for attr in conn.clustering.collect_cluster_attrs("CLUSTER_ID", "$.status"):
    print(attr.node_id, attr.attr_value)
```

See `Example_clusterAttributesInCluster` in [example_clusterattributes_fragment1_test.go](example_clusterattributes_fragment1_test.go), compiled by `go test`.

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#collect-attributes-across-a-cluster)
- [16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py)
- [cluster controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/clusters.py)
- [WSGI Resource](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/common/wsgi.py)
- [conductor](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/conductor/service.py)
