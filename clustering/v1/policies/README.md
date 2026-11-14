# Policies

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_policiesFragment1` in [example_policies_fragment1_test.go](example_policies_fragment1_test.go), compiled by `go test`.

See `Example_policiesFragment2` in [example_policies_fragment2_test.go](example_policies_fragment2_test.go), compiled by `go test`.

See `Example_policiesFragment3` in [example_policies_fragment3_test.go](example_policies_fragment3_test.go), compiled by `go test`.

```python
policy = conn.clustering.get_policy("POLICY_ID")
policy.name = "scale_out_workers"
policy = conn.clustering.update_policy(policy)
```

See `Example_policiesFragment4` in [example_policies_fragment4_test.go](example_policies_fragment4_test.go), compiled by `go test`.

See `Example_policiesFragment5` in [example_policies_fragment5_test.go](example_policies_fragment5_test.go), compiled by `go test`.

See `Example_policiesFragment6` in [example_policies_fragment6_test.go](example_policies_fragment6_test.go), compiled by `go test`.

See `example_policies1_test.go` in [example_policies1_test.go](example_policies1_test.go), compiled by `go test`.

## References

- [Scaling policy 1.0](https://docs.openstack.org/senlin/pike/user/policy_types/scaling.html)
- [Tracking](../tracking/README.md)
- [Finding](../finding/README.md)
- [tracked lifecycle](../tracking/README.md)
- [FindIdentity](../finding/README.md)
- [Policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/policy.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py)
- [Clustering](https://docs.openstack.org/api-ref/clustering/)
- [Clustering Policies Test](../../../api/clustering_policies_test.go)
- [Clustering Policies Response Test](../../../api/clustering_policies_response_test.go)
- [Clustering Lifecycle Test](../../../api/clustering_lifecycle_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
