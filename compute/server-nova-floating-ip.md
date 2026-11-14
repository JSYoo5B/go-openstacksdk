# Server Nova Floating Ip

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_serverNovaFloatingIp` in [example_server_nova_floating_ip_test.go](example_server_nova_floating_ip_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="dev")  # configured floating_ip_source: nova
server = conn.get_server("web-01")
if server is None:
    raise RuntimeError("server not found")

# standalone defaults: timeout=60, wait=False, reuse=True
server = conn.add_ips_to_server(server, ip_pool="public", reuse=True)
server = conn.add_ip_list(
    server, ["198.51.100.10"],
    fixed_address="10.0.0.10", wait=True, timeout=60,
)
```

## References

- [Server Ip Helpers](server-ip-helpers.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
- [Server Ready](server-ready.md)
- [Create With Automatic Floating Ip](create-with-automatic-floating-ip.md)
- [API Microversion History](https://docs.openstack.org/nova/latest/reference/api-microversion-history.html#microversion-2-36)
- [action reference](https://docs.openstack.org/api-ref/compute/#add-associate-floating-ip-addfloatingip-action-deprecated)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1284-L1525)
- [Nova available/create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L673-L712)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1212)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L951-L975)
- [AvailableFloatingIP](floating-ip-available.md)
- [Floating Ip Queries](floating-ip-queries.md)
- [Floating Ip Delete](floating-ip-delete.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [CreateFloatingIP](floating-ip-create.md)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
