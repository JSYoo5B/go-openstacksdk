# Floating Ip Ensure

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.compute.find_server("web-01", ignore_missing=False)
server = conn.add_ips_to_server(
    server,
    ip_pool="public",
    nat_destination="private",
    reuse=True,
    wait=True,
    timeout=60,
)
print(server.interface_ip)
```

See `Example_floatingIpEnsure` in [example_floating_ip_ensure_test.go](example_floating_ip_ensure_test.go), compiled by `go test`.

## References

- [PrepareEnsure·EnsurePrepared](floating-ip-plan.md)
- [AddIPsToServer·AddIPList](../compute/server-ip-helpers.md)
- [FloatingIPs.Available](floating-ip-available.md)
- [Connection AvailableFloatingIP](../compute/floating-ip-available.md)
- [Network Roles](network-roles.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L671)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1394-L1487)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1608-L1709)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1163)
- [Compute IP dispatch](../compute/server-ip-dispatch.md)
- [Network Mutations](network-mutations.md)
- [Floating Ip Roles Test](floating_ip_roles_test.go)
- [Floating Ip Ensure Test](floating_ip_ensure_test.go)
- [Floating Ip Selection Test](floating_ip_selection_test.go)
- [Connection Floating Ip Ensure Test](../connectiontest/connection_floating_ip_ensure_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Compute legacy Nova backend](../compute/server-nova-floating-ip.md)
