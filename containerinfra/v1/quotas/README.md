# Quotas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
func createClusterQuota(ctx context.Context, client *gophercloud.ServiceClient) error {
    api := quotas.New(client)
    project, err := api.InProject(ctx, resource.ID("project-id"))
    if err != nil { return err }
    scope, err := project.ForResource(quotas.Cluster)
    if err != nil { return err }

    quota, err := scope.Create(ctx, request.Custom(func(c *request.Config[quotas.QuotaCreateOpts]) error { v := 0; c.Options.HardLimit = &v; return nil }),
        request.Field("vendor", map[string]string{"tier": "small"}))
    if err != nil { return err }
    fmt.Println(quota.RequestProjectID, quota.RequestResource,
        quota.ID, quota.HardLimit, quota.Header.Get("X-Openstack-Request-Id"))
    current, err := scope.Get(ctx)
    if err != nil { return err }
    fmt.Println(current.HardLimit)

    updated, err := scope.Update(ctx, request.Custom(func(c *request.Config[quotas.QuotaCreateOpts]) error { v := 2; c.Options.HardLimit = &v; return nil }))
    if err != nil { return err }
    fmt.Println(updated.StatusCode)

    _, err = scope.Delete(ctx)
    return err
}
```

```go
import (
    "context"
    "fmt"
    "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func listQuotas(ctx context.Context, api *quotas.API) error {
    for quota, err := range api.List(ctx,
        quotas.WithListOptions(quotas.ListOpts{Limit: 100}),
        request.Custom(func(c *request.Config[quotas.ListOpts]) error { v := true; c.Options.AllTenants = &v; return nil }),
    ) {
        if err != nil { return err }
        fmt.Println(quota.ID, quota.ProjectID, quota.Resource, quota.HardLimit)
    }
    return nil
}
```

## References

- [pinned native Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/containerinfra/v1/quotas/requests.go)
- [native quota model](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/containerinfra/v1/quotas/results.go)
- [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/container_infrastructure_management/v1/_proxy.py)
- [Container Infrastructure Management](https://docs.openstack.org/api-ref/container-infrastructure-management/#magnum-quota-api)
- [server quota controller](https://github.com/openstack/magnum/blob/master/magnum/api/controllers/v1/quota.py)
- [collection next](https://github.com/openstack/magnum/blob/master/magnum/api/controllers/v1/collection.py)
- [server limit validation](https://github.com/openstack/magnum/blob/master/magnum/api/utils.py)
- [Magnum Quota Create Test](../../../api/magnum_quota_create_test.go)
- [Magnum Quota Operations Test](../../../api/magnum_quota_operations_test.go)
- [Magnum Quota List Test](../../../api/magnum_quota_list_test.go)
