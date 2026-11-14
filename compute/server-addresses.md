# Server Addresses

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack
from openstack.cloud import meta

conn = openstack.connect(cloud="dev")
server = conn.compute.find_server("web-01", ignore_missing=False)
print(conn.get_server_public_ip(server))
print(conn.get_server_private_ip(server))

expanded = meta.add_server_interfaces(conn, server)
print(expanded["public_v4"], expanded["public_v6"])
print(expanded["private_v4"], expanded["interface_ip"])
```

See `Example_serverAddresses` in [example_server_addresses_test.go](example_server_addresses_test.go), compiled by `go test`.

## References

- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L859-L873)
- [add_server_interfaces](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L528-L563)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L215-L269)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L147-L190)
- [configured default interface](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L348-L405)
- [Ipaddress](https://docs.python.org/3.13/library/ipaddress.html#ipaddress.IPv4Address.is_private)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L272-L327)
- [Network Roles](../network/network-roles.md)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L474-L525)
- [CreateWithFloatingIP](create-with-floating-ip.md)
- [server_address_selection_test.go](server_address_selection_test.go)
- [Server Address Expansion Test](server_address_expansion_test.go)
- [Connection Server Addresses Test](../connectiontest/connection_server_addresses_test.go)
- [Cloud Config Test](../internal/cloudconfig/cloudconfig_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Server Automatic Ip](server-automatic-ip.md)
