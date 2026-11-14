# Quotasets

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `ExampleAPI_InProject` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

See `ExampleWithQuotaOptions` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

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
