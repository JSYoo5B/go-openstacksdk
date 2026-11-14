# Trunk Address Actions Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
trunk = conn.network.add_trunk_subports(
    trunk_id,
    [{"port_id": child_port_id, "segmentation_type": "vlan", "segmentation_id": 100}],
)
trunk = conn.network.delete_trunk_subports(trunk, [{"port_id": child_port_id}])
group = conn.network.add_addresses_to_address_group(group_id, ["192.0.2.0/24"])
group = conn.network.remove_addresses_from_address_group(group, ["192.0.2.0/24"])
```

See `Example_trunkAddressActions` in [example_trunk_address_actions_test.go](example_trunk_address_actions_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Neutron 27.0.0 trunk policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/trunk.py)
- [address group policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/address_group.py)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py)
- [Policy](https://docs.openstack.org/neutron/2026.2/configuration/policy.html)
