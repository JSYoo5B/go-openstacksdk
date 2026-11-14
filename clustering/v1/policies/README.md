# Policies

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
policy, err := service.Policies.Create(ctx, policies.CreateOpts{Name: "scale_workers"},
    policies.WithCreateSpec(map[string]any{
        "type": "senlin.policy.scaling", "version": "1.0",
        "properties": map[string]any{
            "event": "CLUSTER_SCALE_OUT",
            "adjustment": map[string]any{"type": "CHANGE_IN_CAPACITY", "number": 1},
        },
    }))
if err != nil { return err }
fmt.Println(policy.ID, policy.Spec, policy.Header)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
checked, err := service.Policies.Validate(ctx, policies.ValidateOpts{},
    policies.WithValidateSpec(map[string]any{
        "type": "senlin.policy.scaling", "version": "1.0",
        "properties": map[string]any{
            "event": "CLUSTER_SCALE_OUT",
            "adjustment": map[string]any{"type": "CHANGE_IN_CAPACITY", "number": 1},
        },
    }))
if err != nil { return err }
fmt.Println(checked.Spec, checked.Body["id"])
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
renamed, err := service.Policies.Update(ctx, resource.Name("scale_workers"), policies.UpdateOpts{},
    request.Custom(func(c *request.Config[policies.UpdateOpts]) error { v := "scale_out_workers"; c.Options.Name = &v; return nil }),
    request.Header("X-Vendor-Mode", "explicit"))
if err != nil { return err }
fmt.Println(renamed.ID, renamed.Name)
```

```python
policy = conn.clustering.get_policy("POLICY_ID")
policy.name = "scale_out_workers"
policy = conn.clustering.update_policy(policy)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
tracked, err := service.Policies.Load(ctx, resource.ID("POLICY_ID"))
if err != nil { return err }
if err := tracked.Edit(policies.UpdateOpts{}, request.Custom(func(c *request.Config[policies.UpdateOpts]) error { v := "scale_out_workers"; c.Options.Name = &v; return nil })); err != nil {
    return err
}
policy, err := tracked.Commit(ctx)
if err != nil { return err }
fmt.Println(policy.Name, tracked.Dirty(), tracked.Response().Body)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
for policy, err := range service.Policies.List(ctx,
    policies.WithListOptions(policies.ListOpts{Limit: 20, Sort: "created_at:desc"}),
    request.Custom(func(c *request.Config[policies.ListOpts]) error { v := false; c.Options.GlobalProject = &v; return nil }),
    policies.WithListFilter("data", map[string]any{"vendor": map[string]any{"enabled": false}})) {
    if err != nil { return err }
    fmt.Println(policy.ID, policy.Name)
}
found, err := service.Policies.Find(ctx, resource.Name("scale_out_workers"))
if err != nil { return err }
if found != nil {
    if err := service.Policies.Delete(ctx, resource.ID(found.ID)); err != nil { return err }
}
```

```go
listingAPI := policies.New(client)
values, err := listingAPI.All(ctx,
    policies.WithListOptions(policies.ListOpts{Limit: 20}),
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
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func FindPolicyStrict(ctx context.Context, conn *sdk.Connection) (*policies.Policy, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Policies.FindIdentity(ctx, "placement",
        request.IgnoreMissing(false))
}
```

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
