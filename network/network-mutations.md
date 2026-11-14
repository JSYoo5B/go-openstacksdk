# Network Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
created = conn.create_network("private", port_security_enabled=False)
print(created.id, created.name)
updated = conn.update_network(
    created.id,
    name="private-renamed",
    admin_state_up=False,
    dns_domain="",
)
print(updated.id, updated.name)
deleted = conn.delete_network(created.id)
print("deleted", deleted)
```

See `Example_networkMutations1` in [example_network_mutations1_test.go](example_network_mutations1_test.go), compiled by `go test`.

See `Example_clearNetworkDNSDomain` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

See `Example_createProviderNetwork` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

`CreateNetworkDetail` and `UpdateNetworkDetail` send the same requests and return `NetworkDetail`, which adds the extension properties that the native model omits. `Metadata.Body` keeps every response member.

See `Example_createExternalNetwork` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [Network Roles](network-roles.md)
- [Revision Updates](v2/revision-updates.md)
- [cloud CRUD](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L491-L692)
- [network Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3159-L3307)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L95-L108)
- [Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py#L16-L142)
- [Server Automatic Ip](../compute/server-automatic-ip.md)
- [Server Ip Dispatch](../compute/server-ip-dispatch.md)
- [Nova backend](../compute/server-nova-floating-ip.md)
- [Server Addresses](../compute/server-addresses.md)
- [Create With Automatic Floating Ip](../compute/create-with-automatic-floating-ip.md)
- [ready/wait](../compute/server-ready.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Network Mutations Test](network_mutations_test.go)
- [Network Mutation Boundaries Test](network_mutation_boundaries_test.go)
- [Connection Network Mutations Test](../connectiontest/connection_network_mutations_test.go)
