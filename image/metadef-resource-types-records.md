# Metadef Resource Types Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"

for value in conn.image.metadef_resource_types(
    limit=10, max_items=20, paginated=False
):
    print(value.to_dict())

for value in conn.image.metadef_resource_type_associations(
    namespace, limit=10, max_items=20, name="OS::Nova::Server", paginated=False
):
    print(value.to_dict())
```

See `Example_metadefResourceTypesRecords` in [example_metadef_resource_types_records_test.go](example_metadef_resource_types_records_test.go), compiled by `go test`.

## References

- [Metadefresourcetypes](v2/metadefresourcetypes/README.md)
- [Resource collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L805-L844)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1696-L1786)
- [Resource class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_resource_type.py)
- [inherited list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2459)
- [List Records Test](v2/metadefresourcetypes/list_records_test.go)
- [Connection Image Metadef Records Test](../connectiontest/connection_image_metadef_records_test.go)
- [leaf core fixture](v2/metadefresourcetypes/core_test.go)
- [Contracts Test](v2/metadefresourcetypes/contracts_test.go)
- [List Control Test](../internal/rest/list_control_test.go)
- [List Short Page Test](../internal/rest/list_short_page_test.go)
- [Filter Test](../internal/jsonfilter/filter_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
