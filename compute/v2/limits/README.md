# Limits

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := limits.New(computeClient)
current, err := api.Fetch(ctx, limits.WithGetReserved(true))
if err != nil { return err }
fmt.Println(current.Absolute.MaxTotalCores, current.StatusCode)

scope, err := api.InProject(ctx, resource.ID("project-id"))
if err != nil { return err }
value, err := scope.Get(ctx, limits.WithGetReserved(false))
if err != nil { return err }
fmt.Println(value.ProjectID, value.Absolute.TotalCoresUsed)
```

```go
for _, group := range value.Rate {
    for _, rule := range group.Limits {
        fmt.Println(group.URI, rule.Verb, rule.Remaining,
            string(rule.NextAvailable))
    }
}
```

## References

- [Nova limits API](https://docs.openstack.org/api-ref/compute/#limits-limits)
- [Python limits resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/limits.py)
- [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/limits/results.go)
- [get_limits proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Stable API](https://docs.openstack.org/nova/15.0.4/stable_api.html#background)
- [Compute](https://docs.openstack.org/api-ref/compute/#show-rate-and-absolute-limits)
- [Nova Project Limits Contracts Test](../../../api/nova_project_limits_contracts_test.go)
- [project binding·recorded auth](../../../api/nova_project_limits_projects_test.go)
- [redirect·retry·reauth·context](../../../api/nova_project_limits_transport_test.go)
