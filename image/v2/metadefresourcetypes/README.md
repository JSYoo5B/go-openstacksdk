# Metadefresourcetypes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_metadefresourcetypes_test.go` in [example_metadefresourcetypes_test.go](example_metadefresourcetypes_test.go), compiled by `go test`.

```python
def resource_type_association(conn, namespace, resource_type, prefix, target):
    global_types = list(conn.image.metadef_resource_types(paginated=False))
    created = conn.image.create_metadef_resource_type_association(
        namespace,
        name=resource_type,
        prefix=prefix,
        properties_target=target,
    )
    associations = list(
        conn.image.metadef_resource_type_associations(namespace, paginated=False)
    )
    conn.image.delete_metadef_resource_type_association(
        resource_type, namespace, ignore_missing=False
    )
    return global_types, created, associations
```

## References

- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#metadata-definition-resource-types)
- [Metadef Resource Types](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_resource_types.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1696-L1786)
- [Metadef Resource Type](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_resource_type.py)
- [schema](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_resource_types.py#L263-L317)
- [index·show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_resource_types.py#L49-L119)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Contracts Test](contracts_test.go)
- [Connection Image Metadef Resource Types Test](../../../connectiontest/connection_image_metadef_resource_types_test.go)
- [Glance Metadef Resource Types Test](../../../internal/cmd/sdkgen/glance_metadef_resource_types_test.go)
- [Metadef Resource Types Records](../../metadef-resource-types-records.md)
- [Metadef Resource Type Association Mutations](../../metadef-resource-type-association-mutations.md)
