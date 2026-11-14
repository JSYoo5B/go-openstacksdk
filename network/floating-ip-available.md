# Floating Ip Available

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_floatingIpAvailable` in [example_floating_ip_available_test.go](example_floating_ip_available_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="dev")  # floating_ip_source: neutron
ip = conn.available_floating_ip(network="public")
print(ip["id"], ip["floating_ip_address"])

server = conn.compute.find_server("web-01", ignore_missing=False)
ip = conn.available_floating_ip(network="public", server=server)
```

## References

- [Connection·Compute Available](../compute/floating-ip-available.md)
- [Ensure](floating-ip-ensure.md)
- [Attach](floating-ip-attach.md)
- [Network Roles](network-roles.md)
- [Ensure/PrepareEnsure](floating-ip-plan.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L773)
- [Neutron create·association](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L850-L949)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1608-L1710)
- [normalizer](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1732-L1830)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
