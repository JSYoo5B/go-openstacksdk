# Quotas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := quotas.New(dnsClient)
scope, err := api.InProject(ctx, resource.ID("project-id"),
    quotas.WithAllProjects(true))
if err != nil { return err }

limits, err := scope.Get(ctx)
if err != nil { return err }
fmt.Println(limits.ProjectID, limits.Zones, limits.StatusCode)

zero := 0
updated, err := scope.Update(ctx, quotas.UpdateOpts{Zones: &zero})
if err != nil { return err }
fmt.Println(updated.Header.Get("X-Openstack-Request-Id"))

reset, err := scope.Reset(ctx)
if err != nil { return err }
fmt.Println(reset.ProjectID, reset.StatusCode)
```

```go
zones := 7
snapshot := quotas.WithQuotaOptions(quotas.UpdateOpts{Zones: &zones})
zones = 99
_, err := scope.Update(ctx, quotas.UpdateOpts{}, snapshot) // zones=7
```

## References

- [Designate quota API](https://docs.openstack.org/api-ref/dns/dns-api-v2-index.html#quotas)
- [DNS Quota resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/quota.py)
- [DNS proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/_proxy.py)
- [Gophercloud native API](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/dns/v2/quotas/requests.go)
- [Quotas](https://docs.openstack.org/designate/latest/admin/quotas.html#default-quotas)
- [Designate Project Quotas Contracts Test](../../../api/designate_project_quotas_contracts_test.go)
- [Designate Project Quotas Projects Test](../../../api/designate_project_quotas_projects_test.go)
- [Designate Project Quotas Transport Test](../../../api/designate_project_quotas_transport_test.go)
