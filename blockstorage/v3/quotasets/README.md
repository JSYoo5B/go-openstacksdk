# Quotasets

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := quotasets.New(volumeClient)
scope, err := api.InProject(ctx, resource.ID("project-id"))
if err != nil { return err }

limits, err := scope.Get(ctx)
if err != nil { return err }
fmt.Println(scope.ProjectID(), limits.Volumes, limits.Header.Get("X-Openstack-Request-Id"))

defaults, err := scope.Defaults(ctx)
if err != nil { return err }
fmt.Println(defaults.Volumes)

usage, err := scope.Usage(ctx)
if err != nil { return err }
fmt.Println(usage.Volumes.Limit, usage.Volumes.InUse, usage.Volumes.Reserved)

zero, unlimited := 0, -1
updated, err := scope.Update(ctx, quotasets.UpdateOpts{
    Volumes: &unlimited,
    Snapshots: &zero,
}, quotasets.WithVolumeTypeQuota(quotasets.VolumeTypeVolumes, "SSD", 12))
if err != nil { return err }
fmt.Println(updated.Volumes, string(updated.Body["volumes_SSD"]))

reset, err := scope.Reset(ctx)
if err != nil { return err }
fmt.Println(reset.Header.Get("X-Openstack-Request-Id"))
```

```go
limit := 5
nested := map[string]any{"tier": "original"}
snapshot := quotasets.WithQuotaOptions(quotasets.UpdateOpts{
    Volumes: &limit,
    Extra: map[string]any{"vendor_quota": nested},
})
limit = 99
nested["tier"] = "changed"
_, err := scope.Update(ctx, quotasets.UpdateOpts{}, snapshot) // volumes=5, original
if err != nil { return err }
```

## References

- [Cinder quota API](https://docs.openstack.org/api-ref/block-storage/v3/#quota-sets-extension-os-quota-sets)
- [block storage proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [quota model](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/quota_set.py)
- [common QuotaSet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)
- [cloud quota helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py)
- [Gophercloud v2.15.0 requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/quotasets/requests.go)
- [Cinder Project Quotas Contracts Test](../../../api/cinder_project_quotas_contracts_test.go)
- [Cinder Project Quotas Responses Test](../../../api/cinder_project_quotas_responses_test.go)
- [Python Quota Class](python-quota-class.md)
- [Python Revert](python-revert.md)
