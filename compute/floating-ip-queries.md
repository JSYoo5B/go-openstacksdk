# Floating Ip Queries

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_floatingIpQueries` in [example_floating_ip_queries_test.go](example_floating_ip_queries_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="dev")
all_ips = conn.list_floating_ips()
selected = conn.search_floating_ips(id="IP_ID")
ip = conn.get_floating_ip("IP_ID")
direct = conn.get_floating_ip_by_id("IP_ID")
pools = conn.search_floating_ip_pools(name="public*")
```

## References

- [Available](floating-ip-available.md)
- [IP helper](server-ip-helpers.md)
- [Nova model](server-nova-floating-ip.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L435-L606)
- [ Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L201)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2217-L2358)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1766-L1830)
- [Floating Ip Delete](floating-ip-delete.md)
- [CreateFloatingIP](floating-ip-create.md)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
