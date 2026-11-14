# Create With Floating Ip

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
    ip_pool="public",
    auto_ip=False,
    reuse_ips=True,
    wait=True,
    timeout=180,
)
print(server.id)
```

See `Example_createWithFloatingIp` in [example_create_with_floating_ip_test.go](example_create_with_floating_ip_test.go), compiled by `go test`.

## References

- [Server Addresses](server-addresses.md)
- [Server Default Network](server-default-network.md)
- [Server Network Interfaces](server-network-interfaces.md)
- [Network Roles](../network/network-roles.md)
- [Connection Server Network Roles Test](../connectiontest/connection_server_network_roles_test.go)
- [Floating Ip Ensure](../network/floating-ip-ensure.md)
- [cloud create_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L915-L1245)
- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1481)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1606)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1608-L1709)
- [Network Mutations](../network/network-mutations.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Server Automatic Ip](server-automatic-ip.md)
- [CreateWithAutomaticFloatingIP](create-with-automatic-floating-ip.md)
