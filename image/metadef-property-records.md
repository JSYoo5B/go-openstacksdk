# Metadef Property Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
value = conn.image.get_metadef_property(
    "hw_cpu_policy", "OS::Compute::Libvirt",
    title="fallback title", min_length=0,
)
print(value.to_dict())
```

See `Example_metadefPropertyRecords` in [example_metadef_property_records_test.go](example_metadef_property_records_test.go), compiled by `go test`.

## References

- [Metadef Property](metadef-property.md)
- [public getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1900-L1931)
- [Property descriptor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L21-L79)
- [Proxy getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L822-L865)
- [response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1398)
- [Record Get Test](v2/metadefproperties/record_get_test.go)
- [Connection Image Metadef Property Record Test](../connectiontest/connection_image_metadef_property_record_test.go)
- [leaf core fixture](v2/metadefproperties/core_test.go)
- [Contracts Test](v2/metadefproperties/contracts_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
