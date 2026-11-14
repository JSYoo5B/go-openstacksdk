# Listing

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
for profile in conn.clustering.profiles(
    limit=20, max_items=50, paginated=True, metadata={"team": "infra"},
):
    print(profile.id)
```

See `example_listing1_test.go` in [example_listing1_test.go](../example_listing1_test.go), compiled by `go test`.

```python
for service in conn.clustering.services(
    microversion="1.7", headers={"X-Trace-ID": "engine-list"}, paginated=False,
):
    print(service.id)
```

See `example_listing2_test.go` in [example_listing2_test.go](../example_listing2_test.go), compiled by `go test`.

## References

- [Attribute collection](../clusterattributes/README.md)
- [pinned Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2358)
- [Resource](../../../resource/README.md)
- [List Control Test](../../../internal/rest/list_control_test.go)
- [Actions](../actions/README.md)
- [Events](../events/README.md)
- [Services](../services/README.md)
- [ProfileTypes](../profiletypes/README.md)
- [PolicyTypes](../policytypes/README.md)
- [ClusterPolicies](../clusterpolicies/README.md)
- [pinned Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2248-L2338)
- [Filter](../../../internal/senlin/filter.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)

## Explicit raw Body filters

The eleven typed Senlin List/All APIs accept package-specific WithListFilter for pinned known Body fields. Comparisons use owned JSON snapshots, declared aliases, exact numbers, and recursive object subsets after raw caps and ID validation. Concrete list options select server queries; raw Body keys cannot be forwarded through request.Query. Additional Go model fields and URI attributes do not become semantic filters automatically.

## Per-call list headers and versions

Typed List/All APIs accept request.Header and request.Microversion without mutating the shared client. Options persist across pages while authentication uses the current shared token. Service gates are checked against the selected call version. Authentication, version, and transport headers remain protected, and local control keys cannot become extension queries. Common Resources, parent resolution, and other Get/Wait calls do not accept these typed list options.
