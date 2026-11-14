# Tracking

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
profile = conn.clustering.get_profile("PROFILE_ID")
profile.name = "renamed-profile"
profile.metadata = {"team": "platform"}
profile = conn.clustering.update_profile(profile)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
tracked, err := service.Profiles.Load(ctx, resource.ID("PROFILE_ID"))
if err != nil { return err }
if err := tracked.Edit(profiles.UpdateOpts{},
    request.Custom(func(c *request.Config[profiles.UpdateOpts]) error { v := "renamed-profile"; c.Options.Name = &v; return nil }),
    profiles.WithUpdateMetadata(map[string]any{"team": "platform"})); err != nil {
    return err
}
profile, err := tracked.Commit(ctx, request.Header("X-Audit-Tag", "rename"))
if err != nil { return err }
fmt.Println(profile.Name, tracked.Dirty(), tracked.Response().Body)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
tracked, err := service.Policies.Load(ctx, resource.Name("existing-policy"))
if err != nil { return err }
current := tracked.Value().Name
if err := tracked.Edit(policies.UpdateOpts{}, request.Custom(func(c *request.Config[policies.UpdateOpts]) error { v := current; c.Options.Name = &v; return nil })); err != nil {
    return err
}
_, err = tracked.Commit(ctx)
return err
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
tracked, err := service.Profiles.Load(ctx, resource.ID("PROFILE_ID"))
if err != nil { return err }
if err := tracked.RemoveMetadata(); err != nil { return err }
_, err = tracked.Commit(ctx)
return err
```

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
