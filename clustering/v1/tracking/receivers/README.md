# Receivers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
receiver = conn.clustering.get_receiver("RECEIVER_ID")
receiver.name = "scale_in"
receiver.action = "CLUSTER_SCALE_IN"
receiver.params = {"team": "platform"}
receiver = conn.clustering.update_receiver(receiver)
```

See `example_receivers1_test.go` in [example_receivers1_test.go](../../example_receivers1_test.go), compiled by `go test`.

See `example_receivers2_test.go` in [example_receivers2_test.go](../../example_receivers2_test.go), compiled by `go test`.

## References

- [update_receiver](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1114)
- [Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [TrackedReceiver](../../receivers/lifecycle.go)
- [UpdateScope](../../receivers/scope.go)
- [Tracked](../../../../internal/senlin/tracked.go)
- [Collection Path](../../../../internal/senlin/collection_path.go)
- [Clustering Receivers Lifecycle Test](../../../../api/clustering_receivers_lifecycle_test.go)
- [Tracked Test](../../../../internal/senlin/tracked_test.go)
- [Receivers](../../receivers/README.md)
