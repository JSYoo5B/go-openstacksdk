# Floating Ip Plan

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.compute.find_server("web-01", ignore_missing=False)
server = conn.add_ips_to_server(
    server,
    ip_pool="public",
    reuse=True,
    wait=True,
    timeout=60,
)
print(server.interface_ip)
```

See `Example_floatingIpPlan` in [example_floating_ip_plan_test.go](example_floating_ip_plan_test.go), compiled by `go test`.

## References

- [AddIPsToServer·AddIPList](../compute/server-ip-helpers.md)
- [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1525)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1527-L1606)
- [public add_auto_ip](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1394-L1424)
- [Network Roles](network-roles.md)
- [Server Addresses](../compute/server-addresses.md)
- [Ensure](floating-ip-ensure.md)
- [CreateWithFloatingIP](../compute/create-with-floating-ip.md)
- [Server Automatic Ip](../compute/server-automatic-ip.md)
- [Create With Automatic Floating Ip](../compute/create-with-automatic-floating-ip.md)
- [GetActive/Wait](../compute/server-ready.md)
- [Server Ip Dispatch](../compute/server-ip-dispatch.md)
- [Floating Ip Plan Test](floating_ip_plan_test.go)
- [Floating Ip Plan Boundaries Test](floating_ip_plan_boundaries_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Compute legacy Nova backend](../compute/server-nova-floating-ip.md)
