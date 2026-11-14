# Metadef Object Namespace Record Lists

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"

for value in conn.image.metadef_namespaces(
    visibility="private", owner="project-id", is_protected=False,
    limit=10, max_items=20, paginated=True
):
    print(value.to_dict())

for value in conn.image.metadef_objects(namespace):
    print(value.to_dict())
```

See `Example_metadefObjectNamespaceRecordLists` in [example_metadef_object_namespace_record_lists_test.go](example_metadef_object_namespace_record_lists_test.go), compiled by `go test`.

## References

- [Metadefobjects](v2/metadefobjects/README.md)
- [namespace](v2/metadefnamespaces/README.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1449-L1613)
- [Metadef Object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_object.py)
- [namespace class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py)
- [inherited list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2436)
- [jmespath.py1.0.1 interpreter](https://github.com/jmespath/jmespath.py/blob/1.0.1/jmespath/visitor.py)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
