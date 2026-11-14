# Clusterpolicies

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
binding = conn.clustering.get_cluster_policy("POLICY_NAME_OR_ID", "CLUSTER_NAME_OR_ID")
for binding in conn.clustering.cluster_policies("CLUSTER_NAME_OR_ID", is_enabled=False):
    print(binding.id, binding.policy_id)
```

See `Example_clusterpoliciesFragment1` in [example_clusterpolicies_fragment1_test.go](example_clusterpolicies_fragment1_test.go), compiled by `go test`.

```go
bindings, err := scope.All(ctx,
    request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error { v := false; c.Options.Enabled = &v; return nil }),
    request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error { c.Options.PolicyName = "POLICY_NAME"; return nil }),
    request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error { c.Options.PolicyType = "senlin.policy.scaling-1.0"; return nil }))
if err != nil { return err }
fmt.Println(len(bindings))
```

```go
err = scope.WaitForDelete(ctx, resource.ID("POLICY_NAME_OR_ID"),
    resource.WithTimeout(30*time.Second),
    resource.WithPollInterval(time.Second))
if err != nil { return err }
```

See `Example_clusterpoliciesFragment4` in [example_clusterpolicies_fragment4_test.go](example_clusterpolicies_fragment4_test.go), compiled by `go test`.

See `example_clusterpolicies1_test.go` in [example_clusterpolicies1_test.go](example_clusterpolicies1_test.go), compiled by `go test`.

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#cluster-policies)
- [16.0.0 cluster policy controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/cluster_policies.py)
- [16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
