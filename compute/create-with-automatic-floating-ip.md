# Create With Automatic Floating Ip

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.create_server(
    name="web-01",
    image="ubuntu",
    flavor="small",
    network="private",
    auto_ip=True,
    reuse_ips=True,
    wait=True,
    timeout=180,
)
print(server.id, server.interface_ip)
```

See `Example_createWithAutomaticFloatingIp` in [example_create_with_automatic_floating_ip_test.go](example_create_with_automatic_floating_ip_test.go), compiled by `go test`.

## References

- [Server Ip Dispatch](server-ip-dispatch.md)
- [Server Nova Floating Ip](server-nova-floating-ip.md)
- [AddIPsToServer·AddIPList](server-ip-helpers.md)
- [create_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L915-L1245)
- [wait_for_server/get_active_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1481)
- [Reference](README.md)
- [NIC/defaultnetwork](server-network-interfaces.md)
- [Server Default Network](server-default-network.md)
- [Network Roles](../network/network-roles.md)
- [IP dispatch](server-ip-dispatch.md)
- [Server Ready](server-ready.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
