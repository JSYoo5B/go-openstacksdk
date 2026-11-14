# Quotas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [Designate quota API](https://docs.openstack.org/api-ref/dns/dns-api-v2-index.html#quotas)
- [DNS Quota resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/quota.py)
- [DNS proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/_proxy.py)
- [Gophercloud native API](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/dns/v2/quotas/requests.go)
- [Quotas](https://docs.openstack.org/designate/latest/admin/quotas.html#default-quotas)
- [Designate Project Quotas Contracts Test](../../../api/designate_project_quotas_contracts_test.go)
- [Designate Project Quotas Projects Test](../../../api/designate_project_quotas_projects_test.go)
- [Designate Project Quotas Transport Test](../../../api/designate_project_quotas_transport_test.go)
