# Quotasets

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := quotasets.New(computeClient)
scope, err := api.InProject(ctx, resource.ID("project-id"))
if err != nil { return err }
limits, err := scope.Get(ctx)
if err != nil { return err }
fmt.Println(scope.ProjectID(), limits.Cores, limits.Header.Get("X-Openstack-Request-Id"))

usage, err := scope.Detail(ctx)
if err != nil { return err }
fmt.Println(usage.Cores.Limit, usage.Cores.InUse, usage.Cores.Reserved)

zero, unlimited := 0, -1
updated, err := scope.Update(ctx, quotasets.UpdateOpts{
    Cores: &unlimited,
    Instances: &zero,
}, quotasets.WithUpdateForce(false))
if err != nil { return err }
fmt.Println(updated.Cores, updated.Instances)

reset, err := scope.Reset(ctx)
if err != nil { return err }
fmt.Println(reset.Header.Get("X-Openstack-Request-Id"))
```

```go
userScope, err := scope.InUser(ctx, resource.ID("user-id"))
if err != nil { return err }
userLimits, err := userScope.Get(ctx)
if err != nil { return err }
fmt.Println(userLimits.ProjectID, userLimits.UserID, userLimits.Cores)

zero := 0
_, err = userScope.Update(ctx, quotasets.UpdateOpts{Cores: &zero},
    quotasets.WithUpdateForce(false))
if err != nil { return err }
_, err = userScope.Reset(ctx)
if err != nil { return err }
```

```go
cores := 7
snapshot := quotasets.WithQuotaOptions(quotasets.UpdateOpts{Cores: &cores})
cores = 99
_, err := scope.Update(ctx, quotasets.UpdateOpts{}, snapshot) // cores=7
if err != nil { return err }
```

## References

- [Compute](https://docs.openstack.org/api-ref/compute/#quota-sets-os-quota-sets)
- [Compute](https://docs.openstack.org/api-ref/compute/#update-quotas)
- [Nova version history](https://docs.openstack.org/nova/latest/reference/api-microversion-history.html)
- [compute proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [cloud quota helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py)
- [common QuotaSet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)
- [native quota API](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/quotasets/requests.go)
- [Project Quotas Contracts Test](../../../api/project_quotas_contracts_test.go)
- [Nova Project Quotas Precision Test](../../../api/nova_project_quotas_precision_test.go)
- [Nova Quota Defaults Users Test](../../../api/nova_quota_defaults_users_test.go)
- [Python Quota Class](python-quota-class.md)
