# Metadef Property Record Write

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"
id = "hw_cpu_policy"

created = conn.image.create_metadef_property(
    namespace, name=id, type="string", title="CPU Policy",
    description="CPU allocation policy", is_readonly=False, min_length=0,
)
updated = conn.image.update_metadef_property(
    id, namespace, name=id, type="string", title="CPU Policy",
    description="Updated CPU allocation policy", is_readonly=False, min_length=0,
)
local = conn.image.update_metadef_property(id, namespace)
print(created.to_dict(), updated.to_dict(), local.to_dict())
```

See `Example_metadefPropertyRecordWrite` in [example_metadef_property_record_write_test.go](example_metadef_property_record_write_test.go), compiled by `go test`.

## References

- [Metadefproperties](v2/metadefproperties/README.md)
- [Metadef Property Records](metadef-property-records.md)
- [Metadef Property Record List](metadef-property-record-list.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1790-L1841)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L811)
- [constructor·attribute collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538-L609)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1231-L1336)
- [commit no-op](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L2017)
- [descriptor get/set](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L208-L291)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
