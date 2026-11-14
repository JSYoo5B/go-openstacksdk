# Listing

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
for profile in conn.clustering.profiles(
    limit=20, max_items=50, paginated=True, metadata={"team": "infra"},
):
    print(profile.id)
```

```go
package example

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListProfiles(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    for profile, err := range service.Profiles.List(ctx,
        profiles.WithListOptions(profiles.ListOpts{Limit: 20}),
        request.MaxItems(50), request.Paginated(true),
        profiles.WithListFilter("metadata", map[string]any{"team": "infra"})) {
        if err != nil { return err }
        fmt.Println(profile.ID)
    }
    return nil
}

func FirstProfilesPage(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    for profile, err := range service.Profiles.Resources.List(ctx,
        resource.WithMaxItems(5), resource.WithPaginated(false)) {
        if err != nil { return err }
        fmt.Println(profile.ID)
    }
    return nil
}

func FirstClusterPoliciesPage(ctx context.Context, conn *sdk.Connection,
    cluster resource.Ref) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    scope, err := service.ClusterPolicies.InCluster(ctx, cluster)
    if err != nil { return err }
    for binding, err := range scope.List(ctx,
        request.MaxItems(5), request.Paginated(false)) {
        if err != nil { return err }
        fmt.Println(binding.ID, binding.PolicyID)
    }
    return nil
}
```

```python
for service in conn.clustering.services(
    microversion="1.7", headers={"X-Trace-ID": "engine-list"}, paginated=False,
):
    print(service.id)
```

```go
package example

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func ListServicesWithCallOptions(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.ClusteringV1(ctx)
    if err != nil { return err }
    for engine, err := range service.Services.List(ctx,
        request.Microversion("1.7"),
        request.Header("X-Trace-ID", "engine-list"),
        request.Paginated(false)) {
        if err != nil { return err }
        fmt.Println(engine.ID, engine.State)
    }
    return nil
}
```

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
