# Images

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_images_test.go` in [example_images_test.go](example_images_test.go), compiled by `go test`.

```python
def images_example(conn, image_id, image_name, owner_id):
    image = conn.image.get_image(image_id)
    rows = list(conn.image.images(
        name=image_name, owner=owner_id, tag=["ready", "approved"],
        sort_key=["name", "created_at"], sort_dir=["asc", "desc"], limit=2,
    ))
    return image, rows
```

## References

- [generated/native Image API](v2/images/README.md)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L243-L308)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L353-L550)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L517-L585)
- [Config](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/common/config.py#L282-L329)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1034-L1055)
- [Locations](locations.md)
- [serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1756-L1827)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1855-L1871)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L564-L574)
- [Images Core Test](images_core_test.go)
- [Images Options Test](images_options_test.go)
- [Images Contracts Test](images_contracts_test.go)
- [Connection Image Read Test](../connectiontest/connection_image_read_test.go)
- [Glance Image Read Test](../internal/cmd/sdkgen/glance_image_read_test.go)
