# Quotas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
func manageQuotas(ctx context.Context, client *gophercloud.ServiceClient) error {
    api := quotas.New(client)
    scope, err := api.InProject(ctx, resource.ID("project-id"))
    if err != nil { return err }

    limits, err := scope.Get(ctx)
    if err != nil { return err }
    fmt.Println(limits.ProjectID, limits.Loadbalancer, limits.Listener)

    defaults, err := api.Defaults(ctx)
    if err != nil { return err }
    fmt.Println(defaults.Loadbalancer, defaults.Listener)

    zero, unlimited := 0, -1
    updated, err := scope.Update(ctx, quotas.UpdateOpts{
        Listener: &zero, Loadbalancer: &unlimited,
    }, quotas.WithDefaultLimit(quotas.LimitMembers))
    if err != nil { return err }
    fmt.Println(updated.Header.Get("X-Openstack-Request-Id"))

    for quota, err := range api.ListProjects(ctx, quotas.WithProjectListOptions(
        quotas.ProjectListOpts{Limit: 100},
    )) {
        if err != nil { return err }
        fmt.Println(quota.ProjectID, quota.Pool)
    }

    _, err = scope.Reset(ctx)
    return err
}
```

## References

- [Pagination](https://docs.openstack.org/octavia/latest/_modules/octavia/api/common/pagination.html)
- [V2](https://docs.openstack.org/api-ref/load-balancer/v2/#quotas)
- [native requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/loadbalancer/v2/quotas/requests.go)
- [native URL](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/loadbalancer/v2/quotas/urls.go)
- [Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/load_balancer/v2/_proxy.py)
- [Python quota resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/load_balancer/v2/quota.py)
- [Octavia Project Quotas Contracts Test](../../../api/octavia_project_quotas_contracts_test.go)
- [Octavia Project Quotas Projects Test](../../../api/octavia_project_quotas_projects_test.go)
- [Octavia Project Quotas List Test](../../../api/octavia_project_quotas_list_test.go)
