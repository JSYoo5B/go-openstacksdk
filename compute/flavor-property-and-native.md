# Flavor Property And Native

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
flavor = conn.get_flavor_by_id("7", get_extra=False)
value = conn.compute.get_flavor_extra_specs_property(flavor, "hw:cpu_policy")
print(flavor.id, value)
```

See `Example_flavorPropertyAndNative` in [example_flavor_property_and_native_test.go](example_flavor_property_and_native_test.go), compiled by `go test`.

## References

- [Policy](https://docs.openstack.org/nova/latest/configuration/policy.html)
- [Flavor Extra Specs](flavor-extra-specs.md)
- [User Read APIs](user-read-apis.md)
- [Finding Identities](../docs/finding-identities.md)
