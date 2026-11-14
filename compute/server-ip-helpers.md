# Server Ip Helpers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Choosing a floating IP helper

`AddIPsToServer` with a pool or in automatic mode is idempotent: a server that already has a matching floating IP keeps it, and no further IP is allocated or attached. Use `AddIPFromPool` or `AddAutoIP` when you need Python's `_add_ip_from_pool` or `_add_auto_ip` behavior instead, including reuse of the first unattached pool IP, the Nova fallback after a Neutron NotFound, and, for `AddAutoIP`, deletion of an IP it created when the wait times out. An address list follows Python's `add_ip_list` steps in every helper.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None:
    raise RuntimeError("server not found")

# standalone defaults: timeout=60, wait=False, auto_ip=True, reuse=True
added = conn.add_ips_to_server(server, ip_pool="public", auto_ip=False)
added = conn.add_ip_list(server, ["198.51.100.10", "198.51.100.11"], wait=True)
```

See `Example_serverIpHelpers` in [example_server_ip_helpers_test.go](example_server_ip_helpers_test.go), compiled by `go test`.

## References

- [Server Ip Dispatch](server-ip-dispatch.md)
- [Attach](../network/floating-ip-attach.md)
- [Ensure](../network/floating-ip-ensure.md)
- [Server Ready](server-ready.md)
- [Create With Automatic Floating Ip](create-with-automatic-floating-ip.md)
- [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1525)
- [add_ip_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1343-L1392)
- [attach helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1163)
- [legacy Nova backend](server-nova-floating-ip.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
