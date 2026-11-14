# Quotas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
func manageQuotas(ctx context.Context, networkClient *gophercloud.ServiceClient) error {
    api := quotas.New(networkClient)
    scope, err := api.InProject(ctx, resource.ID("project-id"))
    if err != nil { return err }

    limits, err := scope.Get(ctx)
    if err != nil { return err }
    fmt.Println(scope.ProjectID(), limits.Network, limits.Port)

    defaults, err := scope.Defaults(ctx)
    if err != nil { return err }
    fmt.Println(defaults.Network, defaults.Port)

    usage, err := scope.Detail(ctx)
    if err != nil { return err }
    fmt.Println(usage.Network.Used, usage.Network.Reserved, usage.Network.Limit)

    zero, unlimited := 0, -1
    updated, err := scope.Update(ctx, quotas.UpdateOpts{
        Port: &zero, Network: &unlimited,
    }, quotas.WithUpdateCheckLimit(false))
    if err != nil { return err }
    fmt.Println(updated.Port, updated.Header.Get("X-Openstack-Request-Id"))

    reset, err := scope.Delete(ctx)
    if err != nil { return err }
    fmt.Println(reset.StatusCode, reset.Header.Get("X-Openstack-Request-Id"))
    return nil
}
```

```go
func listQuotaOverrides(ctx context.Context, networkClient *gophercloud.ServiceClient) error {
    api := quotas.New(networkClient)
    for quota, err := range api.ListProjects(ctx, quotas.WithListProjectID("project-id"), request.MaxItems(1)) {
        if err != nil { return err }
        fmt.Println(quota.ProjectID, quota.Network, quota.Port)
        fmt.Println(quota.Header.Get("X-Openstack-Request-Id"), quota.StatusCode)
    }
    return nil
}
```

## References

- [Neutron API reference](https://docs.openstack.org/api-ref/network/v2/#quotas-extension-quotas)
- [Gophercloud requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/quotas/requests.go)
- [Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Python quota resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/quota.py)
- [Neutron quota details API](https://docs.openstack.org/api-ref/network/v2/#quotas-details-extension-quota-details)
- [Neutron Project Quotas Contracts Test](../../../../api/neutron_project_quotas_contracts_test.go)
- [Neutron Project Quotas Projects Test](../../../../api/neutron_project_quotas_projects_test.go)
- [Neutron Quota Defaults Test](../../../../api/neutron_quota_defaults_test.go)
- [QuotaSetsController](https://github.com/openstack/neutron/blob/cadc448a995a30d9b18514089acd7b1770142032/neutron/extensions/quotasv2.py)
- [DbQuotaDriver](https://github.com/openstack/neutron/blob/cadc448a995a30d9b18514089acd7b1770142032/neutron/db/quota/driver.py)
- [Neutron quota API](https://docs.openstack.org/api-ref/network/v2/#quotas-extension-quotas)
- [Neutron Quotas List Test](../../../../api/neutron_quotas_list_test.go)
