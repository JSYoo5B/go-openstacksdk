# Floating Ip Available

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_floatingIpAvailable` in [example_floating_ip_available_test.go](example_floating_ip_available_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="dev")
ip = conn.available_floating_ip(network="public")
print(ip["id"], ip["floating_ip_address"])

server = conn.compute.find_server("web-01", ignore_missing=False)
ip = conn.available_floating_ip(network="public", server=server)
```

## References

- [IP helper](server-ip-helpers.md)
- [Ensure·readiness](server-ip-dispatch.md)
- [Floating Ip Available](../network/floating-ip-available.md)
- [Floating Ip Create](floating-ip-create.md)
- [Server Nova Floating Ip](server-nova-floating-ip.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L773)
- [Nova allocation+compat GET](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L951-L975)
- [normalization](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1732-L1830)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
- [Delete](floating-ip-delete.md)
- [Floating Ip Queries](floating-ip-queries.md)
- [Connection Floating Ip Available Nova Test](../connectiontest/connection_floating_ip_available_nova_test.go)
- [Connection Floating Ip Available Neutron Test](../connectiontest/connection_floating_ip_available_neutron_test.go)
- [Floating Ip Allocate Test](../network/floating_ip_allocate_test.go)
