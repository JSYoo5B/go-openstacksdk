# Metadef Property Record List

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
for value in conn.image.metadef_properties(
    "OS::Compute::Libvirt", min_length=0
):
    print(value.to_dict())
```

See `Example_metadefPropertyRecordList` in [example_metadef_property_record_list_test.go](example_metadef_property_record_list_test.go), compiled by `go test`.

## References

- [Metadefproperties](v2/metadefproperties/README.md)
- [Metadef Property Records](metadef-property-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1876-L1898)
- [Property list override](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L86-L185)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538-L609)
- [List Records Test](v2/metadefproperties/list_records_test.go)
- [Connection Image Metadef Property List Records Test](../connectiontest/connection_image_metadef_property_list_records_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
