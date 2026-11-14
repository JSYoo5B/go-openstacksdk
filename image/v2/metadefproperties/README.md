# Metadefproperties

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_metadefproperties1_test.go` in [example_metadefproperties1_test.go](example_metadefproperties1_test.go), compiled by `go test`.

```python
def manage_properties(conn, namespace, name, replacement_name, clear_namespace=False):
    created = conn.image.create_metadef_property(
        namespace, name=name, type="string", title="Hypervisor Type",
        description="First line\nSecond line", enum=["kvm", "qemu"], default="kvm", readonly=False)
    fetched = conn.image.get_metadef_property(name, namespace)
    rows = list(conn.image.metadef_properties(namespace))
    updated = conn.image.update_metadef_property(
        name, namespace, name=replacement_name, type="string", title="Hypervisor Type",
        description="First line\nSecond line", enum=["kvm", "qemu"], default="kvm", readonly=False)
    conn.image.delete_metadef_property(replacement_name, namespace, ignore_missing=False)
    if clear_namespace:
        conn.image.delete_all_metadef_properties(namespace)
    return created, fetched, rows, updated
```

See `example_metadefproperties2_test.go` in [example_metadefproperties2_test.go](example_metadefproperties2_test.go), compiled by `go test`.

```python
def delete_owned_properties(conn, namespace, id, clear_namespace=False):
    conn.image.delete_metadef_property(id, namespace, ignore_missing=False)
    if clear_namespace:
        conn.image.delete_all_metadef_properties(namespace)
```

## References

- [Metadef Property Records](../../metadef-property-records.md)
- [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1790-L1947)
- [Metadef Properties](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L254-L389)
- [Schema.raw](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/schema.py#L86-L103)
- [property schema](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L692-L806)
- [WSME PropertyType](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/model/metadef_property_type.py#L23-L62)
- [controller.update](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L185-L221)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)
- [server.show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L98-L146)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#show-property-definition)
- [MetadefProperty.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L86-L185)
- [Metadef Properties](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L64-L96)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-properties)
- [core tests](core_test.go)
- [option tests](options_test.go)
- [Contracts Test](contracts_test.go)
- [Connection contract](../../../connectiontest/connection_image_metadef_properties_test.go)
- [generator contract](../../../internal/cmd/sdkgen/glance_metadef_properties_test.go)
- [Metadef Property Record List](../../metadef-property-record-list.md)
- [Metadef Property Record Write](../../metadef-property-record-write.md)
