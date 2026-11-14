# Image Record Stage

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
record = conn.image.stage_image(record, filename="image.qcow2")

# with open("image.qcow2", "rb") as data:
#     record = conn.image.stage_image(record, data=data, size=0)
```

See `Example_imageRecordStage` in [example_image_record_stage_test.go](example_image_record_stage_test.go), compiled by `go test`.

## References

- [`API.ImageData.StageImage/StageKnownImage`](v2/imagedata/README.md)
- [Install](../docs/install.md)
- [Image Records](image-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L538-L581)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L325-L366)
- [Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462-L491)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1403)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)
- [Image Data](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_data.py#L330-L360)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)
