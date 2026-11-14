# Server Automatic Ip

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None or server.status != "ACTIVE":
    raise RuntimeError("an existing ACTIVE server is required")
server = conn.add_ips_to_server(
    server,
    auto_ip=True,
    reuse=True,
    wait=True,
    timeout=180,
)
print(server.interface_ip)
```

See `Example_serverAutomaticIp` in [example_server_automatic_ip_test.go](example_server_automatic_ip_test.go), compiled by `go test`.

## References

- [legacy Nova IP backend](server-nova-floating-ip.md)
- [Floating Ip Plan](../network/floating-ip-plan.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
- [AddIPsToServer·AddIPList](server-ip-helpers.md)
- [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1525)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1527-L1606)
- [attach wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1164)
- [legacy Nova mutation](server-nova-floating-ip.md)
- [add_auto_ip](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1394-L1424)
- [legacy Nova assignment](server-nova-floating-ip.md)
- [Server Addresses](server-addresses.md)
- [Network Roles](../network/network-roles.md)
- [CreateWithAutomaticFloatingIP](create-with-automatic-floating-ip.md)
- [Server Ready](server-ready.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
