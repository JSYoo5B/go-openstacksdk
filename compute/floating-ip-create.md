# Floating Ip Create

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `ExampleCreateFloatingIPRequest` in [example_servers_test.go](example_servers_test.go), compiled by `go test`.

See `Example_floatingIpCreate1` in [example_floating_ip_create1_test.go](example_floating_ip_create1_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="dev")
ip = conn.create_floating_ip(network="public")
attached = conn.create_floating_ip(
    network="public", server={"id": "SERVER_ID"}, wait=True, timeout=60
)
# explicit port wins over server/fixed/NAT in the pinned implementation
on_port = conn.create_floating_ip(network="public", port="PORT_ID", wait=True)
```

## References

- [AvailableFloatingIP](floating-ip-available.md)
- [IP helper](server-ip-helpers.md)
- [Network Ensure](../network/floating-ip-ensure.md)
- [Floating Ip Queries](floating-ip-queries.md)
- [DeleteFloatingIP](floating-ip-delete.md)
- [public/backend selection](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L775-L842)
- [Neutron create/wait/cleanup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L850-L949)
- [Nova POST/compat GET](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L951-L975)
- [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L101)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
