# Metadef Property

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
value = conn.image.get_metadef_property("hw_cpu_policy", "OS::Compute::Libvirt")
print(value.to_dict())
```

See `Example_metadefPropertyFragment1` in [example_metadef_property_fragment1_test.go](example_metadef_property_fragment1_test.go), compiled by `go test`.

```python
# TypeError before HTTP: Proxy._get already binds resource_type.
value = conn.image.get_metadef_property(
    "hw_cpu_policy", "OS::Compute::Libvirt", resource_type="OS::Nova::Server"
)
```

See `Example_metadefPropertyResourceType` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

See `Example_metadefProperty1` in [example_metadef_property1_test.go](example_metadef_property1_test.go), compiled by `go test`.

## References

- [Metadefproperties](v2/metadefproperties/README.md)
- [Metadef Property Records](metadef-property-records.md)
- [core](v2/metadefproperties/core_test.go)
- [Contracts Test](v2/metadefproperties/contracts_test.go)
- [Options Test](v2/metadefproperties/options_test.go)
- [Connection](../connectiontest/connection_image_metadef_properties_test.go)
- [Get Mapping Test](v2/metadefproperties/get_mapping_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [public getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1900-L1931)
- [MetadefProperty](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L21-L79)
- [Proxy._get_resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L605)
- [Proxy._get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L822-L865)
- [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1849)
- [response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1398)
