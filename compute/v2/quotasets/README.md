# Quotasets

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `ExampleAPI_InProject` in [example_test.go](example_test.go), compiled by `go test`.

See `ExampleWithQuotaOptions` in [example_test.go](example_test.go), compiled by `go test`.

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
