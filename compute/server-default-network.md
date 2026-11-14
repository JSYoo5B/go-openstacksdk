# Server Default Network

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev", compute_api_version="2.37")
server = conn.create_server(
    name="web-01",
    image="ubuntu",
    flavor="small",
    auto_ip=False,
    wait=False,
)
print(server.id)
```

See `Example_serverDefaultNetwork` in [example_server_default_network_test.go](example_server_default_network_test.go), compiled by `go test`.

## References

- [Server Network Interfaces](server-network-interfaces.md)
- [Network Roles](../network/network-roles.md)
- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1084-L1110)
- [Cloud Region](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/config/cloud_region.py#L1475-L1481)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L252-L264)
- [shared network discovery](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L110-L365)
- [Network Mutations](../network/network-mutations.md)
- [Connection Network Role Consumers Test](../connectiontest/connection_network_role_consumers_test.go)
- [Default Network Test](default_network_test.go)
- [Connection Default Network Test](../connectiontest/connection_default_network_test.go)
- [Cloud Config Test](../internal/cloudconfig/cloudconfig_test.go)
- [Connection Cloud Network Test](../connectiontest/connection_cloud_network_test.go)
