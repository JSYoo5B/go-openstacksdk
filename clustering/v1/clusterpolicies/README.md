# Clusterpolicies

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
binding = conn.clustering.get_cluster_policy("POLICY_NAME_OR_ID", "CLUSTER_NAME_OR_ID")
for binding in conn.clustering.cluster_policies("CLUSTER_NAME_OR_ID", is_enabled=False):
    print(binding.id, binding.policy_id)
```

See `Example_clusterpoliciesFragment1` in [example_clusterpolicies_fragment1_test.go](example_clusterpolicies_fragment1_test.go), compiled by `go test`.

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

See `Example_clusterpoliciesFragment4` in [example_clusterpolicies_fragment4_test.go](example_clusterpolicies_fragment4_test.go), compiled by `go test`.

See `example_clusterpolicies1_test.go` in [example_clusterpolicies1_test.go](example_clusterpolicies1_test.go), compiled by `go test`.

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#cluster-policies)
- [16.0.0 cluster policy controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/cluster_policies.py)
- [16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
