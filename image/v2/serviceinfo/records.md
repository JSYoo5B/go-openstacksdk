# Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
info = conn.image.get_import_info()
print(info.to_dict())

for store in conn.image.stores(details=False):
    print(store.to_dict())

for store in conn.image.stores(
    details=False, is_default=True, properties={"region": "east"},
    limit=10, max_items=20, paginated=True
):
    print(store.to_dict())
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

## References

- [Python Store Details](python-store-details.md)
- [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2272-L2295)
- [Install](../../../docs/install.md)
- [Service Info](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L22-L44)
- [Fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L122)
- [Resource collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L808-L839)
- [Source fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1356-L1396)
- [Source constructor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2305-L2323)
- [Discovery](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L41-L51)
- [Discovery](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L53-L93)
- [Discovery](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L165-L193)
- [Discovery](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/discovery.py#L21-L33)
- [Import Record Test](import_record_test.go)
- [Store Records Test](store_records_test.go)
- [Connection Image Serviceinfo Records Test](../../../connectiontest/connection_image_serviceinfo_records_test.go)
