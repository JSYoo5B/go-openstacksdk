# Image Record Tags

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")

result = conn.image.add_tag("image-id", "reviewed")
assert result is None

image = conn.image.get_image("image-id")
conn.image.remove_tag(image, "reviewed")
print(image.tags)
```

See `Example_imageRecordTags` in [example_image_record_tags_test.go](example_image_record_tags_test.go), compiled by `go test`.

## References

- [ImageRecord](image-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1196-L1220)
- [TagMixin add/remove](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py#L105-L145)
- [Install](../docs/install.md)
- [urljoin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L43-L50)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L602)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L785-L839)
- [Image Record Tag Test](image_record_tag_test.go)
- [Connection Image Record Tags Test](../connectiontest/connection_image_record_tags_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
