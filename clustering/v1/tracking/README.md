# Tracking

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
profile = conn.clustering.get_profile("PROFILE_ID")
profile.name = "renamed-profile"
profile.metadata = {"team": "platform"}
profile = conn.clustering.update_profile(profile)
```

See `Example_trackingFragment1` in [example_tracking_fragment1_test.go](../example_tracking_fragment1_test.go), compiled by `go test`.

See `Example_trackingFragment1` in [example_tracking_fragment1_test.go](../example_tracking_fragment1_test.go), compiled by `go test`.

See `Example_trackingFragment3` in [example_tracking_fragment3_test.go](../example_tracking_fragment3_test.go), compiled by `go test`.

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L214)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1914)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1380)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1231)
- [generic update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716)
- [Profile update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L226)
- [Policy update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1030)
- [Tracked Test](../../../internal/senlin/tracked_test.go)
- [Clustering Lifecycle Test](../../../api/clustering_lifecycle_test.go)
- [Async](async/README.md)
- [Receivers](receivers/README.md)
