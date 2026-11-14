# Profiles

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
spec := map[string]any{
    "type": "os.nova.server",
    "version": "1.0",
    "properties": map[string]any{"name": "worker", "flavor": "FLAVOR_ID", "image": "IMAGE_ID"},
}
profile, err := service.Profiles.Create(ctx, profiles.CreateOpts{Name: "worker_template"},
    profiles.WithCreateSpec(spec),
    profiles.WithCreateMetadata(map[string]any{"role": "worker"}))
if err != nil { return err }
fmt.Println(profile.ID, profile.UserMetadata, profile.Header)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
checked, err := service.Profiles.Validate(ctx, profiles.ValidateOpts{},
    profiles.WithValidateSpec(map[string]any{
        "type": "os.nova.server", "version": "1.0",
        "properties": map[string]any{"flavor": "FLAVOR_ID", "image": "IMAGE_ID"},
    }))
if err != nil { return err }
fmt.Println(checked.Spec, checked.Body["id"])
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
updated, err := service.Profiles.Update(ctx, resource.Name("worker_template"), profiles.UpdateOpts{},
    profiles.WithUpdateMetadata(map[string]any{}),
    request.Header("X-Vendor-Mode", "explicit"))
if err != nil { return err }
fmt.Println(updated.ID, updated.UserMetadata)
```

```python
profile = conn.clustering.get_profile("PROFILE_ID")
profile.name = "renamed_template"
profile.metadata = {"role": "worker"}
profile = conn.clustering.update_profile(profile)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
tracked, err := service.Profiles.Load(ctx, resource.ID("PROFILE_ID"))
if err != nil { return err }
if err := tracked.Edit(profiles.UpdateOpts{},
    request.Custom(func(c *request.Config[profiles.UpdateOpts]) error { v := "renamed_template"; c.Options.Name = &v; return nil }),
    profiles.WithUpdateMetadata(map[string]any{"role": "worker"})); err != nil {
    return err
}
profile, err := tracked.Commit(ctx)
if err != nil { return err }
fmt.Println(profile.Name, tracked.Dirty(), tracked.Response().Body)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
for profile, err := range service.Profiles.List(ctx,
    profiles.WithListOptions(profiles.ListOpts{Limit: 20, Sort: "created_at:desc"}),
    request.Custom(func(c *request.Config[profiles.ListOpts]) error { v := false; c.Options.GlobalProject = &v; return nil }),
    profiles.WithListFilter("metadata", map[string]any{"role": "worker"})) {
    if err != nil { return err }
    fmt.Println(profile.ID, profile.Name)
}
found, err := service.Profiles.Find(ctx, resource.Name("worker_template"))
if err != nil { return err }
if found != nil { fmt.Println(found.ID) }
```

```go
listingAPI := profiles.New(client)
values, err := listingAPI.All(ctx,
    profiles.WithListOptions(profiles.ListOpts{Limit: 20}),
    request.MaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, request.Paginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func FindProfileStrict(ctx context.Context, conn *sdk.Connection) (*profiles.Profile, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Profiles.FindIdentity(ctx, "worker_template",
        request.IgnoreMissing(false))
}
```

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
