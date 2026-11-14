# Update

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_update_test.go` in [example_update_test.go](example_update_test.go), compiled by `go test`.

```python
from openstack.image.v2.image import Image

def update_image_example(conn, image_id):
    cached = Image.existing(id=image_id, name="old-name", team="platform")
    updated = conn.image.update_image(cached, name="release-image", is_protected=False)
    did_update = conn.image.update_image_properties(
        image=updated, meta={"team": "platform"}, min_ram=0,
    )
    return updated, did_update
```

## References

- [ImageInfo](images.md)
- [generated/native Update](v2/images/README.md)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L559-L682)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L617-L655)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L657-L713)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1406-L1442)
- [parser](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1450-L1551)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L2011)
- [`update_image_properties`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1139-L1179)
- [native Request](https://github.com/gophercloud/gophercloud/blob/v2.15.0/service_client.go#L144-L163)
- [Update Core Test](update_core_test.go)
- [Update Options Test](update_options_test.go)
- [Update Contracts Test](update_contracts_test.go)
- [Connection Image Update Test](../connectiontest/connection_image_update_test.go)
- [Glance Image Update Test](../internal/cmd/sdkgen/glance_image_update_test.go)
