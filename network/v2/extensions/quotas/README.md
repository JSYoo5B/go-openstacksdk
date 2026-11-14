# Quotas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

See `Example_listProjects` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

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
