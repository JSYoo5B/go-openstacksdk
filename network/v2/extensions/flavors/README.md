# Flavors

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
flavor = conn.network.create_flavor(
    name="gold", service_type="L3_ROUTER_NAT", is_enabled=True)
conn.network.associate_flavor_with_service_profile(flavor, profile_id)
for f in conn.network.flavors(service_type="L3_ROUTER_NAT"):
    print(f.name, f.service_profile_ids)
conn.network.delete_flavor(flavor)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1630-L1794)
- [`flavor.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/flavor.py)
