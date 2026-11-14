# Metadef Resource Type Association Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"
resource_type = "OS::Nova::Server"

created = conn.image.create_metadef_resource_type_association(
    namespace, name=resource_type, prefix="hw:", properties_target="metadata",
)
print(created.to_dict())
conn.image.delete_metadef_resource_type_association(
    resource_type, namespace, ignore_missing=False,
)
```

See `Example_metadefResourceTypeAssociationMutations` in [example_metadef_resource_type_association_mutations_test.go](example_metadef_resource_type_association_mutations_test.go), compiled by `go test`.

## References

- [Metadef Resource Types Records](metadef-resource-types-records.md)
- [Metadefresourcetypes](v2/metadefresourcetypes/README.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1711-L1761)
- [association class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_resource_type.py#L34-L61)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L609)
- [dirty request](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1231-L1336)
- [Response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1398)
- [Delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094-L2154)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
