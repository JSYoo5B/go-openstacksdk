# Image Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
image = conn.image.get_image("image-id")
print(image.to_dict(), image.image_import_methods)

for image in conn.image.images(status="active", limit=10, max_items=20):
    print(image.to_dict())

image = conn.image.find_image("ubuntu", ignore_missing=False)
print(image.to_dict())
```

See `Example_imageRecords` in [example_image_records_test.go](example_image_records_test.go), compiled by `go test`.

## References

- [Download API](download.md)
- [Images](images.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1013-L1054)
- [Install](../docs/install.md)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L45-L63)
- [QueryParameters validation·transpose](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L301-L382)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L79-L260)
- [Fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L128)
- [properties packing](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1221-L1230)
- [fetch translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1356-L1394)
- [constructor collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L808-L839)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L429-L437)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L494-L531)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2524-L2582)
- [Image Record Get Test](image_record_get_test.go)
- [Image Record Lists Test](image_record_lists_test.go)
- [Image Record Find Test](image_record_find_test.go)
- [Connection Image Records Test](../connectiontest/connection_image_records_test.go)
- [Python String Test](../internal/cloudfilter/python_string_test.go)
