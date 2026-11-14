# Server Network Interfaces

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(compute_api_version="2.42")
port = conn.network.find_port("app-port", ignore_missing=False)
server = conn.create_server(
    name="web-01", image="ubuntu", flavor="small",
    nics=[
        {"net-name": "private", "fixed_ip": "10.0.0.10", "tag": "app"},
        {"port-id": port.id, "tag": "existing-port"},
    ],
    auto_ip=False, wait=False,
)
```

See `Example_serverNetworkInterfaces` in [example_server_network_interfaces_test.go](example_server_network_interfaces_test.go), compiled by `go test`.

## References

- [Reference](README.md)
- [Server Default Network](server-default-network.md)
- [API Microversion History](https://docs.openstack.org/nova/2026.1/reference/api-microversion-history.html)
- [WithMicroversionRange/WithLatestMicroversion](../docs/microversions.md)
- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1074-L1154)
- [Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L242-L312)
- [Network Roles](../network/network-roles.md)
- [Create With Floating Ip](create-with-floating-ip.md)
- [Network Mutations](../network/network-mutations.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
