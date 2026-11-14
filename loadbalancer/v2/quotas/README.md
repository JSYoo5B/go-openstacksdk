# Quotas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

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
