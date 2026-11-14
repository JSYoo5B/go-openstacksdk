# Flavor Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(compute_api_version="2.55")
for flavor in conn.compute.flavors(min_ram=1024, disk=20):
    print(flavor.to_dict())

found = conn.compute.find_flavor("m1.small", ignore_missing=True,
                                 get_extra_specs=True)
if found is not None:
    print(found.to_dict())
```

See `Example_flavorRecords` in [example_flavor_records_test.go](example_flavor_records_test.go), compiled by `go test`.

## References

- [FetchExtraSpecs](flavor-extra-specs.md)
- [Console Selection](console-selection.md)
- [Flavor Property And Native](flavor-property-and-native.md)
- [Proxy flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L269-L296)
- [Proxy find_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L165-L199)
- [Flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py#L43-L138)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
