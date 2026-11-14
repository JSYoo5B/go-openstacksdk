# Schema Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace_schema = conn.image.get_metadef_namespace_schema()
task_schema = conn.image.get_task_schema()
print(namespace_schema.to_dict())
print(task_schema.to_dict())
```

See `Example_schemaRecords` in [example_schema_records_test.go](example_schema_records_test.go), compiled by `go test`.

## References

- [Schemas](schemas.md)
- [Schema Records Test](schema_records_test.go)
- [Connection Image Schema Records Test](../connectiontest/connection_image_schema_records_test.go)
- [Schemas Core Test](schemas_core_test.go)
- [Schemas Contracts Test](schemas_contracts_test.go)
- [Schemas Options Test](schemas_options_test.go)
- [Connection](../connectiontest/connection_image_schemas_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1951-L2160)
- [Schema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/schema.py#L17-L26)
- [MetadefSchema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_schema.py#L17-L30)
