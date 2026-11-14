# Quotas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

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
