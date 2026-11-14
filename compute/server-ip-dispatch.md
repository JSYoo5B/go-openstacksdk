# Server Ip Dispatch

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None:
    raise RuntimeError("server not found")

ready = conn.wait_for_server(
    server,
    ip_pool="public",
    ips=["198.51.100.10", "198.51.100.11"],
    auto_ip=False,
    reuse=True,
    timeout=120,
)

ready = conn.wait_for_server(
    server,
    ips=["198.51.100.10", "198.51.100.11"],
    auto_ip=False,
    timeout=120,
)
```

See `Example_serverIpDispatch` in [example_server_ip_dispatch_test.go](example_server_ip_dispatch_test.go), compiled by `go test`.

## References

- [AddIPsToServer·AddIPList](server-ip-helpers.md)
- [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1527)
- [add_ip_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1343-L1392)
- [_add_ip_from_pool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1284-L1342)
- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1481)
- [Ensure](../network/floating-ip-ensure.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L671)
- [Attach](../network/floating-ip-attach.md)
- [legacy Nova backend](server-nova-floating-ip.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
