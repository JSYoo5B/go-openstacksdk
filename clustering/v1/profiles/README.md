# Profiles

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_profilesFragment1` in [example_profiles_fragment1_test.go](example_profiles_fragment1_test.go), compiled by `go test`.

See `Example_profilesFragment2` in [example_profiles_fragment2_test.go](example_profiles_fragment2_test.go), compiled by `go test`.

See `Example_profilesFragment3` in [example_profiles_fragment3_test.go](example_profiles_fragment3_test.go), compiled by `go test`.

```python
profile = conn.clustering.get_profile("PROFILE_ID")
profile.name = "renamed_template"
profile.metadata = {"role": "worker"}
profile = conn.clustering.update_profile(profile)
```

See `Example_profilesFragment4` in [example_profiles_fragment4_test.go](example_profiles_fragment4_test.go), compiled by `go test`.

See `Example_profilesFragment5` in [example_profiles_fragment5_test.go](example_profiles_fragment5_test.go), compiled by `go test`.

See `Example_profilesFragment6` in [example_profiles_fragment6_test.go](example_profiles_fragment6_test.go), compiled by `go test`.

See `example_profiles1_test.go` in [example_profiles1_test.go](example_profiles1_test.go), compiled by `go test`.

## References

- [Nova profile spec](https://docs.openstack.org/senlin/2023.2/user/profile_types/nova.html)
- [Tracking](../tracking/README.md)
- [Finding](../finding/README.md)
- [tracked lifecycle](../tracking/README.md)
- [FindIdentity](../finding/README.md)
- [Profile](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/profile.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py)
- [Clustering](https://docs.openstack.org/api-ref/clustering/)
- [Clustering Profiles Test](../../../api/clustering_profiles_test.go)
- [Clustering Profiles Update Snapshot Test](../../../api/clustering_profiles_update_snapshot_test.go)
- [Clustering Lifecycle Test](../../../api/clustering_lifecycle_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
