# Finding

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
profile = conn.clustering.find_profile("worker_template")
policy = conn.clustering.find_policy("placement")
cluster = conn.clustering.find_cluster("workers")
node = conn.clustering.find_node("worker_1")
receiver = conn.clustering.find_receiver("scale_out")
```

See `example_finding1_test.go` in [example_finding1_test.go](../example_finding1_test.go), compiled by `go test`.

See `example_finding2_test.go` in [example_finding2_test.go](../example_finding2_test.go), compiled by `go test`.

## References

- [Listing](../listing/README.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2439)
- [Find Identity](../../../internal/senlin/find_identity.go)
- [Find Identity Test](../../../internal/senlin/find_identity_test.go)
- [Clustering Find Identity Test](../../../api/clustering_find_identity_test.go)
